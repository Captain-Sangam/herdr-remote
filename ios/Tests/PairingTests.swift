import Core
import Foundation
import XCTest

@testable import HerdrRemote

/// Step 5 of the plan: enroll as one request, against the manual harness with
/// `HR_MANUAL_AUTOALLOW=1`. `TEST_RUNNER_HR_PAIR_LINK` is a fresh link from
/// `herdr-remoted pair` on the harness, and the test sets its endpoint to 127.0.0.1. Skipped
/// without it.
final class PairingTests: XCTestCase {
  let store = Store()

  override func tearDown() {
    store.forget()
  }

  final class Codes: @unchecked Sendable {
    private let lock = NSLock()
    private var seen: [String] = []
    func add(_ c: String) { lock.withLock { seen.append(c) } }
    var all: [String] { lock.withLock { seen } }
  }

  /// A wrong pin first: no code, and the token never goes out, so the same link then pairs.
  func testAWrongPinShowsNoCodeAndThenTheLinkPairs() async throws {
    guard let raw = ProcessInfo.processInfo.environment["HR_PAIR_LINK"] else {
      throw XCTSkip("set TEST_RUNNER_HR_PAIR_LINK to a fresh harness link")
    }
    // The harness listens on 127.0.0.1 only.
    let text = raw.replacingOccurrences(
      of: "ep=[^&]*", with: "ep=127.0.0.1", options: .regularExpression)
    let link = try Pairing.parse(text)
    store.forget()
    let identity = try DeviceKey.ensure()

    let bad = try Pairing.parse(
      text.replacingOccurrences(
        of: "pin=\(link.pin)", with: "pin=QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4"))
    let none = Codes()
    do {
      _ = try await Enroll.run(link: bad, name: "ios test", identity: identity) { none.add($0) }
      XCTFail("a wrong pin paired")
    } catch {
      XCTAssertEqual(
        Pairing.failure(error), "Could not reach the Mac: The Mac's key does not match the pairing")
    }
    XCTAssertEqual(none.all, [])

    let codes = Codes()
    let deviceId = try await Enroll.run(link: link, name: "ios test", identity: identity) {
      codes.add($0)
    }
    XCTAssertFalse(deviceId.isEmpty)
    // The code is the one of the harness's key and this device's key.
    let server = try XCTUnwrap(
      Spki.der(
        p256Point: try XCTUnwrap(
          SecKeyCopyExternalRepresentation(
            try XCTUnwrap(
              SecCertificateCopyKey(
                try Harness.certificate(
                  try Harness.requireDir().appendingPathComponent("state/server.crt")))), nil)
            as Data?)))
    let expected = Pairing.code(
      serverSpki: server, clientSpki: try XCTUnwrap(DeviceKey.spki(identity)))
    XCTAssertEqual(Set(codes.all), [expected])

    // The new device identity reaches the paired port.
    store.update {
      $0.pin = link.pin
      $0.endpoints = link.endpoints
      $0.port = link.port
      $0.deviceId = deviceId
    }
    let api = try XCTUnwrap(HerdrApi.paired(store))
    let hello = try await api.hello()
    XCTAssertFalse(hello.version.isEmpty)
  }
}
