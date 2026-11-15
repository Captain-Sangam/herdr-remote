import Core
import Foundation
import Security

/// The keychain access group that the app and its extensions share (decisions 3 and 7). The
/// build puts `$(AppIdentifierPrefix)dev.herdrremote.shared` into each Info.plist.
enum Keychain {
  static let group = Bundle.main.object(forInfoDictionaryKey: "HRKeychainGroup") as? String ?? ""

  struct Failure: Error {
    let status: OSStatus
  }

  static func check(_ status: OSStatus) throws {
    if status != errSecSuccess { throw Failure(status: status) }
  }
}

/// The device key and its certificate, as one keychain identity (ADR 0025). The private key
/// never leaves the keychain: on a device it is in the Secure Enclave. It needs no user
/// presence and is usable after the first unlock, so alerts work while the device is locked.
enum DeviceKey {
  static let label = "herdr-remote-device"
  static let tag = Data("dev.herdrremote.device".utf8)

  /// The identity, made on first use.
  static func ensure() throws -> SecIdentity {
    if let id = identity() { return id }
    let key = try privateKey() ?? makeKey()
    guard let pub = SecKeyCopyPublicKey(key),
      let point = SecKeyCopyExternalRepresentation(pub, nil) as Data?
    else { throw Keychain.Failure(status: errSecInvalidKeyRef) }
    var serial = Data(count: 16)
    for i in serial.indices { serial[i] = UInt8.random(in: 0...255) }
    let now = Date()
    let der = try Certificate.selfSigned(
      point: point, serial: serial, notBefore: now.addingTimeInterval(-3600),
      notAfter: now.addingTimeInterval(10 * 365 * 24 * 3600)
    ) { tbs in
      var error: Unmanaged<CFError>?
      guard
        let sig = SecKeyCreateSignature(
          key, .ecdsaSignatureMessageX962SHA256, tbs as CFData, &error) as Data?
      else { throw error!.takeRetainedValue() as Error }
      return sig
    }
    guard let cert = SecCertificateCreateWithData(nil, der as CFData) else {
      throw Keychain.Failure(status: errSecDecode)
    }
    SecItemDelete(
      [kSecClass: kSecClassCertificate, kSecAttrLabel: label, kSecAttrAccessGroup: Keychain.group]
        as CFDictionary)
    try Keychain.check(
      SecItemAdd(
        [
          kSecValueRef: cert, kSecAttrLabel: label, kSecAttrAccessGroup: Keychain.group,
          kSecAttrAccessible: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ] as CFDictionary, nil))
    guard let id = identity() else { throw Keychain.Failure(status: errSecItemNotFound) }
    return id
  }

  /// The SPKI DER of the device key, for the pairing code.
  static func spki(_ identity: SecIdentity) -> Data? {
    var cert: SecCertificate?
    guard SecIdentityCopyCertificate(identity, &cert) == errSecSuccess, let cert,
      let key = SecCertificateCopyKey(cert),
      let point = SecKeyCopyExternalRepresentation(key, nil) as Data?
    else { return nil }
    return Spki.der(p256Point: point)
  }

  static func identity() -> SecIdentity? {
    var out: CFTypeRef?
    let status = SecItemCopyMatching(
      [
        kSecClass: kSecClassIdentity, kSecAttrLabel: label, kSecAttrAccessGroup: Keychain.group,
        kSecReturnRef: true,
      ] as CFDictionary, &out)
    guard status == errSecSuccess, let out, CFGetTypeID(out) == SecIdentityGetTypeID() else {
      return nil
    }
    return (out as! SecIdentity)
  }

  private static func privateKey() -> SecKey? {
    var out: CFTypeRef?
    let status = SecItemCopyMatching(
      [
        kSecClass: kSecClassKey, kSecAttrApplicationTag: tag, kSecAttrAccessGroup: Keychain.group,
        kSecReturnRef: true,
      ] as CFDictionary, &out)
    guard status == errSecSuccess, let out, CFGetTypeID(out) == SecKeyGetTypeID() else {
      return nil
    }
    return (out as! SecKey)
  }

  private static func makeKey() throws -> SecKey {
    var priv: [CFString: Any] = [
      kSecAttrIsPermanent: true, kSecAttrApplicationTag: tag, kSecAttrLabel: label,
      kSecAttrAccessGroup: Keychain.group,
    ]
    var attrs: [CFString: Any] = [
      kSecAttrKeyType: kSecAttrKeyTypeECSECPrimeRandom, kSecAttrKeySizeInBits: 256,
    ]
    #if targetEnvironment(simulator)
      // The simulator has no Secure Enclave: a software key, in simulator builds only.
      priv[kSecAttrAccessible] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
    #else
      var error: Unmanaged<CFError>?
      guard
        let access = SecAccessControlCreateWithFlags(
          nil, kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly, .privateKeyUsage, &error)
      else { throw error!.takeRetainedValue() as Error }
      priv[kSecAttrAccessControl] = access
      attrs[kSecAttrTokenID] = kSecAttrTokenIDSecureEnclave
    #endif
    attrs[kSecPrivateKeyAttrs] = priv
    var keyError: Unmanaged<CFError>?
    guard let key = SecKeyCreateRandomKey(attrs as CFDictionary, &keyError) else {
      throw keyError!.takeRetainedValue() as Error
    }
    return key
  }

  static func forget() {
    for cls in [kSecClassKey, kSecClassCertificate] {
      SecItemDelete(
        [kSecClass: cls, kSecAttrLabel: label, kSecAttrAccessGroup: Keychain.group] as CFDictionary)
    }
  }
}

/// What the app remembers about its Mac (decision 7): one keychain item that holds JSON, which
/// the extensions also read, also on a locked device after the first unlock. Nothing in it is
/// a secret: the key is in the keychain on its own.
struct StoreState: Codable, Equatable {
  var pin: String?
  var endpoints: [String] = []
  var lastGood: String?
  var port = 7766
  var deviceId: String?
  var macName = ""
  var watch = false
  /// server_time_ms minus the phone clock, from the last /v1/hello.
  var clockOffsetMs: Int64 = 0

  var paired: Bool { pin != nil && deviceId != nil }
}

final class Store: @unchecked Sendable {
  static let service = "dev.herdrremote"
  static let account = "state"

  private let lock = NSLock()
  private var cached: StoreState

  /// Reads the item. Before the first unlock it cannot be read: the state is then empty.
  init() {
    cached = Self.read() ?? StoreState()
  }

  var state: StoreState { lock.withLock { cached } }

  /// Changes the state and writes it. A failed write keeps the change in memory only.
  func update(_ change: (inout StoreState) -> Void) {
    let data: Data? = lock.withLock {
      var s = cached
      change(&s)
      guard s != cached else { return nil }
      cached = s
      return try? JSONEncoder().encode(s)
    }
    if let data { Self.write(data) }
  }

  /// Deletes the state, the key and the certificate.
  func forget() {
    lock.withLock { cached = StoreState() }
    SecItemDelete(Self.query as CFDictionary)
    DeviceKey.forget()
  }

  private static var query: [CFString: Any] {
    [
      kSecClass: kSecClassGenericPassword, kSecAttrService: service, kSecAttrAccount: account,
      kSecAttrAccessGroup: Keychain.group,
    ]
  }

  static func read() -> StoreState? {
    var q = query
    q[kSecReturnData] = true
    var out: CFTypeRef?
    guard SecItemCopyMatching(q as CFDictionary, &out) == errSecSuccess, let data = out as? Data
    else { return nil }
    return try? JSONDecoder().decode(StoreState.self, from: data)
  }

  private static func write(_ data: Data) {
    let status = SecItemUpdate(query as CFDictionary, [kSecValueData: data] as CFDictionary)
    guard status == errSecItemNotFound else { return }
    var q = query
    q[kSecValueData] = data
    q[kSecAttrAccessible] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
    SecItemAdd(q as CFDictionary, nil)
  }
}
