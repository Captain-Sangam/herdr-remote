import Core
import CryptoKit
import Foundation
import Security
import XCTest

@testable import HerdrRemote

/// The manual harness of docs/testing.md (`TestManualServe`). Its folder comes from
/// `TEST_RUNNER_HR_MANUAL_DIR`: the simulator reads the Mac's files. The harness's own phone key
/// stands in for the device key until the app can pair.
enum Harness {
  static var dir: URL? {
    ProcessInfo.processInfo.environment["HR_MANUAL_DIR"].map { URL(fileURLWithPath: $0) }
  }

  static func requireDir() throws -> URL {
    guard let dir else { throw XCTSkip("set TEST_RUNNER_HR_MANUAL_DIR to run the harness tests") }
    return dir
  }

  static func port() throws -> Int {
    let text = try String(contentsOf: requireDir().appendingPathComponent("port"), encoding: .utf8)
    return Int(text.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0
  }

  /// The pin of the harness daemon, from its public server.crt.
  static func pin() throws -> String {
    let cert = try certificate(requireDir().appendingPathComponent("state/server.crt"))
    return try XCTUnwrap(pin(of: cert))
  }

  static func pin(of cert: SecCertificate) -> String? {
    guard let key = SecCertificateCopyKey(cert),
      let point = SecKeyCopyExternalRepresentation(key, nil) as Data?,
      let der = Spki.der(p256Point: point)
    else { return nil }
    return Spki.hashB64(der)
  }

  static func certificate(_ url: URL) throws -> SecCertificate {
    try XCTUnwrap(SecCertificateCreateWithData(nil, pem(url) as CFData))
  }

  /// The harness phone key and certificate as a keychain identity, under a test label.
  static func identity() throws -> SecIdentity {
    let dir = try requireDir()
    let key = try P256.Signing.PrivateKey(
      pemRepresentation: String(
        contentsOf: dir.appendingPathComponent("client.key"), encoding: .utf8))
    let cert = try certificate(dir.appendingPathComponent("client.crt"))
    return try TestIdentity.add(x963: key.x963Representation, certificate: cert)
  }

  static func pem(_ url: URL) throws -> Data {
    let text = try String(contentsOf: url, encoding: .utf8)
    let body = text.split(separator: "\n").filter { !$0.hasPrefix("-----") }.joined()
    return try XCTUnwrap(Data(base64Encoded: body))
  }
}

/// A software key and a certificate in the keychain, read back as one identity.
enum TestIdentity {
  static let label = "herdr-remote-test"

  static func add(x963: Data, certificate: SecCertificate) throws -> SecIdentity {
    remove()
    var error: Unmanaged<CFError>?
    let key = try XCTUnwrap(
      SecKeyCreateWithData(
        x963 as CFData,
        [
          kSecAttrKeyType: kSecAttrKeyTypeECSECPrimeRandom,
          kSecAttrKeyClass: kSecAttrKeyClassPrivate,
        ]
          as CFDictionary, &error), "\(String(describing: error))")
    try check(SecItemAdd([kSecValueRef: key, kSecAttrLabel: label] as CFDictionary, nil))
    try check(SecItemAdd([kSecValueRef: certificate, kSecAttrLabel: label] as CFDictionary, nil))
    var out: CFTypeRef?
    try check(
      SecItemCopyMatching(
        [kSecClass: kSecClassIdentity, kSecAttrLabel: label, kSecReturnRef: true] as CFDictionary,
        &out))
    guard let out, CFGetTypeID(out) == SecIdentityGetTypeID() else { throw XCTSkip("no identity") }
    return out as! SecIdentity
  }

  static func remove() {
    for cls in [kSecClassKey, kSecClassCertificate] {
      SecItemDelete([kSecClass: cls, kSecAttrLabel: label] as CFDictionary)
    }
  }

  static func check(_ status: OSStatus, line: UInt = #line) throws {
    guard status == errSecSuccess else {
      throw NSError(domain: NSOSStatusErrorDomain, code: Int(status), userInfo: ["line": line])
    }
  }
}
