import Foundation
import Testing

@testable import Core

// PairingTest.kt.

let good =
  "herdr-remote://pair?v=1&pin=QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4&ep=192.168.1.3,100.64.0.9,mac.tail.ts.net&p=7766&ep_port=7767&t=abcDEF123_-&exp=2000000000"

@Test func parsesAllFields() throws {
  let l = try Pairing.parse(good, nowSec: 1_900_000_000)
  #expect(l.pin == "QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4")
  #expect(l.endpoints == ["192.168.1.3", "100.64.0.9", "mac.tail.ts.net"])
  #expect(l.port == 7766)
  #expect(l.enrollPort == 7767)
  #expect(l.token == "abcDEF123_-")
  #expect(l.expiresAt == 2_000_000_000)
}

@Test func refusesBadVersionExpiredAndMissingFields() {
  func failure(_ text: String, nowSec: Int64 = 1_900_000_000) -> String {
    do {
      _ = try Pairing.parse(text, nowSec: nowSec)
      return "parsed"
    } catch {
      return (error as? PairingError)?.message ?? "\(error)"
    }
  }
  #expect(failure(good.replacingOccurrences(of: "v=1", with: "v=2")).contains("newer app"))
  #expect(failure(good, nowSec: 2_000_000_001).contains("expired"))
  #expect(failure(good.replacingOccurrences(of: "&t=abcDEF123_-", with: "")).contains("incomplete"))
  #expect(failure(good.replacingOccurrences(of: "&ep_port=7767", with: "")).contains("incomplete"))
  #expect(
    failure(good.replacingOccurrences(of: "ep=192.168.1.3,100.64.0.9,mac.tail.ts.net", with: "ep="))
      .contains("incomplete"))
  #expect(
    failure(
      good.replacingOccurrences(of: "pin=QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4", with: "pin=")
    )
    .contains("incomplete"))
  #expect(failure("https://example.com/pair?v=1").contains("not a Herdr Remote"))
  #expect(failure("").contains("not a Herdr Remote"))
}

@Test func codeMatchesSharedVectors() throws {
  let vectors = try pairingVectors()
  #expect(vectors.count >= 5)
  for (i, v) in vectors.enumerated() {
    let code = Pairing.code(
      serverSpki: try #require(Base64URL.decode(v["server_spki_b64url"] ?? "")),
      clientSpki: try #require(Base64URL.decode(v["client_spki_b64url"] ?? "")))
    #expect(code == v["code"], "vector \(i)")
  }
}

@Test func uuid7HasVersionAndTime() {
  let id = Array(Uuid7.make(nowMs: 1_794_122_774_714))
  #expect(id.count == 36)
  #expect(id[14] == "7")
  #expect("89ab".contains(id[19]))
  #expect(Int64(String(id[0..<8] + id[9..<13]), radix: 16) == 1_794_122_774_714)
  #expect(Uuid7.make(nowMs: 1) != Uuid7.make(nowMs: 1))
}
