import CryptoKit
import Foundation

/// The content of the pairing QR code (docs/api.md, Pairing link).
public struct PairingLink: Equatable, Sendable {
  public let pin: String
  public let endpoints: [String]
  public let port: Int
  public let enrollPort: Int
  public let token: String
  public let expiresAt: Int64
}

public struct PairingError: Error, Equatable {
  public let message: String
}

public enum Pairing {
  static let prefix = "herdr-remote://pair?"

  public static func parse(
    _ text: String, nowSec: Int64 = Int64(Date().timeIntervalSince1970)
  ) throws -> PairingLink {
    let t = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard t.hasPrefix(prefix) else {
      throw PairingError(message: "This is not a Herdr Remote pairing code")
    }
    var q: [String: String] = [:]
    for part in t.dropFirst(prefix.count).split(separator: "&", omittingEmptySubsequences: false) {
      let kv = part.split(separator: "=", maxSplits: 1, omittingEmptySubsequences: false)
      q[String(kv[0])] = kv.count > 1 ? String(kv[1]) : ""
    }
    guard q["v"] == "1" else { throw PairingError(message: "This pairing code needs a newer app") }
    let pin = q["pin"] ?? ""
    let eps = (q["ep"] ?? "").split(separator: ",").map {
      $0.trimmingCharacters(in: .whitespaces)
    }.filter { !$0.isEmpty }
    let port = Int(q["p"] ?? "") ?? 0
    let enrollPort = Int(q["ep_port"] ?? "") ?? 0
    let token = q["t"] ?? ""
    let exp = Int64(q["exp"] ?? "") ?? 0
    guard !pin.isEmpty, !eps.isEmpty, (1...65535).contains(port), (1...65535).contains(enrollPort),
      !token.isEmpty, exp != 0
    else { throw PairingError(message: "The pairing code is incomplete") }
    guard exp > nowSec else {
      throw PairingError(
        message: "This pairing code has expired. Click Pair a phone again on the Mac.")
    }
    return PairingLink(
      pin: pin, endpoints: eps, port: port, enrollPort: enrollPort, token: token, expiresAt: exp)
  }

  /// The 6-digit code both screens show: SHA-256(server SPKI ‖ client SPKI), first 4 bytes
  /// big-endian, mod 1e6.
  public static func code(serverSpki: Data, clientSpki: Data) -> String {
    let h = Array(SHA256.hash(data: serverSpki + clientSpki))
    let n = h[0..<4].reduce(UInt32(0)) { $0 << 8 | UInt32($1) }
    let s = String(n % 1_000_000)
    return String(repeating: "0", count: 6 - s.count) + s
  }
}
