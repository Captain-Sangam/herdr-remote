import Foundation
import Testing

@testable import Core

/// The shared vectors of the Android and daemon tests (testdata/pairing-vectors.json).
func pairingVectors() throws -> [[String: String]] {
  let url = URL(fileURLWithPath: #filePath)
    .deletingLastPathComponent().appendingPathComponent("../../../../testdata/pairing-vectors.json")
  let root = try JSONSerialization.jsonObject(with: Data(contentsOf: url)) as? [String: Any]
  return root?["vectors"] as? [[String: String]] ?? []
}

@Test func spkiDerFromThePointMatchesTheSharedVectors() throws {
  let vectors = try pairingVectors()
  #expect(vectors.count >= 5)
  for v in vectors {
    for field in ["server_spki_b64url", "client_spki_b64url"] {
      let der = try #require(Base64URL.decode(v[field] ?? ""))
      #expect(der.count == 91)
      #expect(Spki.der(p256Point: der.suffix(65)) == der)
    }
  }
}

@Test func spkiRefusesAnythingButAP256Point() {
  #expect(Spki.der(p256Point: Data(repeating: 4, count: 64)) == nil)
  #expect(Spki.der(p256Point: Data([0x02]) + Data(repeating: 1, count: 64)) == nil)
  #expect(Spki.der(p256Point: Data()) == nil)
}

@Test func thePinMatchesOnlyItsOwnKey() throws {
  let der = try #require(Base64URL.decode(try pairingVectors()[0]["server_spki_b64url"] ?? ""))
  let point = der.suffix(65)
  let pin = Spki.hashB64(der)
  #expect(pin.count == 43)
  #expect(Spki.matches(p256Point: point, pin: pin))
  for bad in [
    "", String(pin.dropLast()), pin.lowercased(), "QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4",
  ] {
    #expect(!Spki.matches(p256Point: point, pin: bad))
  }
}
