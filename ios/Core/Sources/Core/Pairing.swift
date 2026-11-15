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

  /// The daemon's rule for a device name (`validName` in daemon/pair.go): 1 to 64 Unicode
  /// scalars, each a letter, mark, number, punctuation, symbol or the ASCII space.
  public static func validName(_ name: String) -> Bool {
    let scalars = name.unicodeScalars
    guard (1...64).contains(scalars.count) else { return false }
    return scalars.allSatisfy { u in
      if u == " " { return true }
      switch u.properties.generalCategory {
      case .uppercaseLetter, .lowercaseLetter, .titlecaseLetter, .modifierLetter, .otherLetter,
        .nonspacingMark, .spacingMark, .enclosingMark, .decimalNumber, .letterNumber, .otherNumber,
        .connectorPunctuation, .dashPunctuation, .openPunctuation, .closePunctuation,
        .initialPunctuation, .finalPunctuation, .otherPunctuation, .mathSymbol, .currencySymbol,
        .modifierSymbol, .otherSymbol:
        return true
      default: return false
      }
    }
  }

  /// The default device name: the model and the hardware identifier, "iPad (iPad13,18)".
  public static func defaultName(model: String, machine: String) -> String {
    let name = machine.isEmpty ? model : "\(model) (\(machine))"
    return String(name.unicodeScalars.prefix(64).map(Character.init))
  }

  /// What the pair screen says when enroll fails: the words of `ui/PairScreen.kt`.
  public static func failure(_ e: Error) -> String {
    if let e = e as? ApiError {
      switch e.status {
      case 403:
        return e.error == "denied"
          ? "The Mac denied this phone." : "The Mac refused this code: \(e.error)."
      case 410:
        return "The pairing window closed before Allow. Click Pair a phone again on the Mac."
      default: return "The Mac answered \(e.status): \(e.error)"
      }
    }
    if let e = e as? PairingError { return e.message }
    let message = (e as? UnreachableError)?.message ?? e.localizedDescription
    return "Could not reach the Mac: \(message)"
  }

  /// The failure when the Mac's key does not match the pin: no code shows, no token goes out.
  public static let pinMismatch = "The Mac's key does not match the pairing"
}
