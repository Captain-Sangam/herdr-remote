import Foundation

/// The device certificate (decision 3 of the plan): a self-signed X.509 v3 certificate in DER,
/// CN=herdr-remote, ECDSA with SHA-256. The daemon reads only its P-256 key, so it has no
/// extensions. The signature comes from a closure, so that the Secure Enclave key signs it.
public enum Certificate {
  /// `point` is the 65-byte X9.63 public key. `sign` gets the to-be-signed DER and returns the
  /// DER ECDSA signature of its SHA-256 (`ecdsaSignatureMessageX962SHA256`).
  public static func selfSigned(
    point: Data, serial: Data, notBefore: Date, notAfter: Date,
    sign: (Data) throws -> Data
  ) throws -> Data {
    guard let spki = Spki.der(p256Point: point) else {
      throw PairingError(message: "not a P-256 key")
    }
    let name = seq(set(seq(oid([0x55, 0x04, 0x03]) + tlv(0x0C, Data("herdr-remote".utf8)))))
    let tbs = seq(
      tlv(0xA0, int(Data([2])))  // version v3
        + int(serial) + ecdsaSHA256 + name
        + seq(time(notBefore) + time(notAfter))
        + name + spki)
    return seq(tbs + ecdsaSHA256 + tlv(0x03, Data([0]) + (try sign(tbs))))
  }

  /// ecdsa-with-SHA256 (1.2.840.10045.4.3.2), with no parameters.
  static let ecdsaSHA256 = seq(oid([0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x04, 0x03, 0x02]))

  static func tlv(_ tag: UInt8, _ body: Data) -> Data {
    var out = Data([tag])
    let n = body.count
    if n < 0x80 {
      out.append(UInt8(n))
    } else {
      let len = withUnsafeBytes(of: UInt32(n).bigEndian) { Data($0) }.drop { $0 == 0 }
      out.append(0x80 | UInt8(len.count))
      out.append(contentsOf: len)
    }
    return out + body
  }

  static func seq(_ body: Data) -> Data { tlv(0x30, body) }
  static func set(_ body: Data) -> Data { tlv(0x31, body) }
  static func oid(_ bytes: [UInt8]) -> Data { tlv(0x06, Data(bytes)) }

  /// A positive INTEGER: no leading zero bytes, and a 0 in front of a high bit.
  static func int(_ value: Data) -> Data {
    var v = Data(value.drop { $0 == 0 })
    if v.isEmpty || v[v.startIndex] & 0x80 != 0 { v.insert(0, at: 0) }
    return tlv(0x02, v)
  }

  /// UTCTime before 2050, GeneralizedTime from 2050 (RFC 5280 section 4.1.2.5).
  static func time(_ date: Date) -> Data {
    let f = DateFormatter()
    f.locale = Locale(identifier: "en_US_POSIX")
    f.timeZone = TimeZone(identifier: "UTC")
    f.dateFormat = "yyyy"
    let utc = Int(f.string(from: date)) ?? 0 < 2050
    f.dateFormat = utc ? "yyMMddHHmmss'Z'" : "yyyyMMddHHmmss'Z'"
    return tlv(utc ? 0x17 : 0x18, Data(f.string(from: date).utf8))
  }
}
