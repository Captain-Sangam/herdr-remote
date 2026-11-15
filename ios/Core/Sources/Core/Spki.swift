import CryptoKit
import Foundation

/// The P-256 SubjectPublicKeyInfo and its pin (docs/architecture.md section 4.3).
public enum Spki {
  /// The fixed DER header of a P-256 SPKI: SEQUENCE { SEQUENCE { id-ecPublicKey, prime256v1 },
  /// BIT STRING }. The 65-byte uncompressed point follows it.
  static let p256Header: [UInt8] = [
    0x30, 0x59, 0x30, 0x13, 0x06, 0x07, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x02, 0x01,
    0x06, 0x08, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07, 0x03, 0x42, 0x00,
  ]

  /// The SPKI DER of a P-256 key, from its 65-byte X9.63 point (what
  /// `SecKeyCopyExternalRepresentation` gives). Nil for any other key.
  public static func der(p256Point point: Data) -> Data? {
    guard point.count == 65, point.first == 0x04 else { return nil }
    return Data(p256Header) + point
  }

  /// The base64url SHA-256 of an SPKI DER, without padding: the pairing pin.
  public static func hashB64(_ spkiDer: Data) -> String {
    Base64URL.encode(Data(SHA256.hash(data: spkiDer)))
  }

  /// True only when the point is a P-256 key whose SPKI hash equals the pin. The compare takes
  /// the same time for every pin of the same length.
  public static func matches(p256Point point: Data, pin: String) -> Bool {
    guard let der = der(p256Point: point) else { return false }
    let got = Array(hashB64(der).utf8)
    let want = Array(pin.utf8)
    guard got.count == want.count else { return false }
    return zip(got, want).reduce(0) { $0 | ($1.0 ^ $1.1) } == 0
  }
}

/// Base64url without padding (RFC 4648 section 5).
public enum Base64URL {
  public static func encode(_ data: Data) -> String {
    data.base64EncodedString()
      .replacingOccurrences(of: "+", with: "-")
      .replacingOccurrences(of: "/", with: "_")
      .replacingOccurrences(of: "=", with: "")
  }

  public static func decode(_ text: String) -> Data? {
    var s = text.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
    s += String(repeating: "=", count: (4 - s.count % 4) % 4)
    return Data(base64Encoded: s)
  }
}
