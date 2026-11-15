import Core
import Foundation
import XCTest

@testable import HerdrRemote

/// Step 1 of the plan: URLSession presents a keychain identity, pins the SPKI, needs TLS 1.3 and
/// refuses a redirect, against the manual harness. Skipped without `TEST_RUNNER_HR_MANUAL_DIR`.
/// `TEST_RUNNER_HR_SPIKE_NAME` adds a DNS name that reaches the harness: a MagicDNS name, or a
/// stand-in name that resolves to 127.0.0.1 and has the same ATS exception in Info.plist.
final class SpikeTests: XCTestCase {
  override func tearDown() {
    TestIdentity.remove()
  }

  private func get(_ url: String, pin: String) async throws -> (Data, HTTPURLResponse) {
    let tls = PinnedTLS(pin: pin, identity: try Harness.identity())
    let session = tls.session(idle: 10)
    defer { session.invalidateAndCancel() }
    let (data, resp) = try await session.data(from: try XCTUnwrap(URL(string: url)))
    return (data, try XCTUnwrap(resp as? HTTPURLResponse))
  }

  private func hello(host: String) async throws {
    let (data, resp) = try await get(
      "https://\(host):\(try Harness.port())/v1/hello", pin: try Harness.pin())
    XCTAssertEqual(resp.statusCode, 200, host)
    let json = try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    XCTAssertNotNil(json["server_time_ms"], host)
  }

  func testHelloOverAnIPAddress() async throws {
    try await hello(host: "127.0.0.1")
  }

  func testHelloOverAName() async throws {
    try await hello(host: "localhost")
    if let name = ProcessInfo.processInfo.environment["HR_SPIKE_NAME"] {
      try await hello(host: name)
    }
  }

  /// Also over the name: the ATS exception must not skip the pin.
  func testAWrongPinNeverConnects() async throws {
    let name = ProcessInfo.processInfo.environment["HR_SPIKE_NAME"]
    for host in ["127.0.0.1"] + (name.map { [$0] } ?? []) {
      do {
        _ = try await get(
          "https://\(host):\(try Harness.port())/v1/hello",
          pin: "QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4")
        XCTFail("a wrong pin connected to \(host)")
      } catch let e as URLError {
        XCTAssertEqual(e.code, .cancelled, host)
      }
    }
  }

  /// Go's ServeMux answers a path with "//" with a 301 to the clean path.
  func testARedirectIsRefused() async throws {
    let (_, resp) = try await get(
      "https://127.0.0.1:\(try Harness.port())/v1//hello", pin: try Harness.pin())
    XCTAssertEqual(resp.statusCode, 301)
    XCTAssertEqual(resp.value(forHTTPHeaderField: "Location"), "/v1/hello")
  }

  /// The harness phone key serves as the fake server's key: the pin is its SPKI hash.
  func testTLS12IsRefused() async throws {
    let identity = try Harness.identity()
    let pin = try XCTUnwrap(
      Harness.pin(
        of: Harness.certificate(try Harness.requireDir().appendingPathComponent("client.crt"))))
    let ok = FakeServer.ok("{}")
    let tls13 = try FakeServer(identity: identity, tls: .TLSv13, reply: ok)
    defer { tls13.stop() }
    let (_, resp) = try await get("https://127.0.0.1:\(tls13.port)/x", pin: pin)
    XCTAssertEqual(resp.statusCode, 200, "the control over TLS 1.3")

    let tls12 = try FakeServer(identity: identity, tls: .TLSv12, reply: ok)
    defer { tls12.stop() }
    do {
      _ = try await get("https://127.0.0.1:\(tls12.port)/x", pin: pin)
      XCTFail("TLS 1.2 connected")
    } catch is URLError {}
    XCTAssertEqual(tls12.requests, 0)
  }
}
