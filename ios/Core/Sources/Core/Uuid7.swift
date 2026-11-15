import Foundation

/// A UUIDv7 for a request id. The time comes from the caller, so that the clock offset to the Mac
/// can correct it.
public enum Uuid7 {
  public static func make(nowMs: Int64) -> String {
    var r = [UInt8](repeating: 0, count: 10)
    for i in r.indices { r[i] = UInt8.random(in: 0...255) }
    let ms = UInt64(bitPattern: nowMs) & 0xFFFF_FFFF_FFFF
    var b = [UInt8](repeating: 0, count: 16)
    for i in 0..<6 { b[i] = UInt8(truncatingIfNeeded: ms >> (40 - 8 * UInt64(i))) }
    b[6] = 0x70 | (r[0] & 0x0F)
    b[7] = r[1]
    b[8] = 0x80 | (r[2] & 0x3F)
    b[9] = r[3]
    for i in 0..<6 { b[10 + i] = r[4 + i] }
    let hex = b.map { String(format: "%02x", $0) }.joined()
    let p = Array(hex)
    return [p[0..<8], p[8..<12], p[12..<16], p[16..<20], p[20..<32]].map { String($0) }.joined(
      separator: "-")
  }
}
