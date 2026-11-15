import Core
import Foundation
import XCTest

@testable import HerdrRemote

/// Step 3 of the plan: the device identity, the store, endpoint failover and the action rule
/// (decision 6), against a fake TLS server. The device identity also serves as the server's.
final class ApiTests: XCTestCase {
  let store = Store()

  override func tearDown() {
    store.forget()
    TestIdentity.remove()
  }

  private func api(endpoints: [String], port: Int) throws -> (HerdrApi, SecIdentity) {
    let id = try DeviceKey.ensure()
    let pin = Spki.hashB64(try XCTUnwrap(DeviceKey.spki(id)))
    store.update {
      $0.pin = pin
      $0.deviceId = "test"
      $0.endpoints = endpoints
      $0.lastGood = nil
      $0.port = port
    }
    return (HerdrApi(store: store, pin: pin, identity: id), id)
  }

  private func decision() -> DecisionRequest {
    DecisionRequest(
      requestId: Uuid7.make(nowMs: 1), decision: "approve", turn: 1, promptHash: "h", key: nil)
  }

  func testTheDeviceKeyFormsOneIdentityAndForgetRemovesIt() throws {
    let a = try XCTUnwrap(DeviceKey.spki(try DeviceKey.ensure()))
    let b = try XCTUnwrap(DeviceKey.spki(try DeviceKey.ensure()))
    XCTAssertEqual(a, b)
    XCTAssertEqual(a.count, 91)
    store.update { $0.macName = "mac" }
    XCTAssertEqual(Store.read()?.macName, "mac")
    store.forget()
    XCTAssertNil(DeviceKey.identity())
    XCTAssertNil(Store.read())
    XCTAssertEqual(store.state, StoreState())
  }

  func testARefusedFirstEndpointMovesToTheSecond() async throws {
    let (_, id) = try api(endpoints: [], port: 0)
    let server = try FakeServer(identity: id, reply: FakeServer.ok(#"{"result":"applied"}"#))
    defer { server.stop() }
    let (api, _) = try self.api(endpoints: ["::1", "127.0.0.1"], port: server.port)
    let result = try await api.decide("t1", decision())
    XCTAssertEqual(result, "applied")
    XCTAssertEqual(server.requests, 1)
    XCTAssertEqual(store.state.lastGood, "127.0.0.1")
  }

  func testACloseAfterTheRequestIsOutcomeUnknownAndOneRequest() async throws {
    let (_, id) = try api(endpoints: [], port: 0)
    let server = try FakeServer(identity: id, reply: nil)
    defer { server.stop() }
    let (api, _) = try self.api(endpoints: ["127.0.0.1", "localhost"], port: server.port)
    do {
      _ = try await api.decide("t1", decision())
      XCTFail("the action succeeded")
    } catch {
      XCTAssertEqual(Texts.actionError(error), Texts.result("outcome_unknown"), "\(error)")
    }
    XCTAssertEqual(server.requests, 1, "the server saw the action once across all endpoints")
  }

  /// SentMarkTest.kt: a refused connection never started, a request with no reply may have
  /// arrived.
  func testTheProbeKnowsWhetherTheRequestStarted() async throws {
    let id = try DeviceKey.ensure()
    let pin = Spki.hashB64(try XCTUnwrap(DeviceKey.spki(id)))
    let server = try FakeServer(identity: id, reply: nil)
    defer { server.stop() }
    let session = PinnedTLS(pin: pin, identity: id).session(idle: 5)
    defer { session.invalidateAndCancel() }
    func post(_ host: String) async -> Bool {
      var req = URLRequest(url: HerdrApi.url(host, server.port, "/v1/x"))
      req.httpMethod = "POST"
      req.httpBody = Data("{}".utf8)
      let probe = TaskProbe(connect: nil)
      _ = try? await session.data(for: req, delegate: probe)
      return probe.started
    }
    let refused = await post("::1")
    let started = await post("127.0.0.1")
    XCTAssertFalse(refused)
    XCTAssertTrue(started)
  }

  func testARedirectIsRefused() async throws {
    let (_, id) = try api(endpoints: [], port: 0)
    let server = try FakeServer(identity: id, reply: nil)
    defer { server.stop() }
    server.reply =
      "HTTP/1.1 302 Found\r\nLocation: https://127.0.0.1:\(server.port)/v1/other\r\n"
      + "Content-Length: 0\r\nConnection: close\r\n\r\n"
    let (api, _) = try self.api(endpoints: ["127.0.0.1"], port: server.port)
    do {
      _ = try await api.hello()
      XCTFail("followed the redirect")
    } catch let e as ApiError {
      XCTAssertEqual(e.status, 302)
    }
    XCTAssertEqual(server.requests, 1)
  }

  /// URLSession has no connect timeout: the probe ends the silent endpoint after 10 s.
  func testAnEndpointThatNeverConnectsMovesOnAfterTheConnectTimeout() async throws {
    let (_, id) = try api(endpoints: [], port: 0)
    let server = try FakeServer(identity: id, reply: FakeServer.ok(#"{"result":"applied"}"#))
    defer { server.stop() }
    let silent = try SilentServer(port: server.port)
    defer { silent.stop() }
    let (api, _) = try self.api(endpoints: ["::1", "127.0.0.1"], port: server.port)
    let start = Date()
    let result = try await api.decide("t1", decision())
    XCTAssertEqual(result, "applied")
    XCTAssertEqual(server.requests, 1)
    XCTAssertEqual(Date().timeIntervalSince(start), HerdrApi.connectTimeout, accuracy: 3)
  }

  /// The manual harness: hello and spaces over its paired phone key.
  func testHelloAndSpacesAgainstTheHarness() async throws {
    let pin = try Harness.pin()
    store.update {
      $0.pin = pin
      $0.deviceId = "harness"
      $0.endpoints = ["127.0.0.1"]
      $0.port = (try? Harness.port()) ?? 0
    }
    let api = HerdrApi(store: store, pin: pin, identity: try Harness.identity())
    let hello = try await api.hello()
    XCTAssertFalse(hello.version.isEmpty)
    XCTAssertGreaterThan(hello.serverTimeMs, 0)
    XCTAssertEqual(store.state.lastGood, "127.0.0.1")
    _ = try await api.spaces()
  }
}
