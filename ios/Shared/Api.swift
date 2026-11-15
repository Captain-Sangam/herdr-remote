import Core
import Foundation
import Security

/// The daemon's API (docs/api.md), with the timeouts of "Timeouts on the phone".
final class HerdrApi: Sendable {
  static let connectTimeout: TimeInterval = 10

  let store: Store
  private let tls: PinnedTLS
  private let calls: URLSession

  init(store: Store, pin: String, identity: SecIdentity) {
    self.store = store
    tls = PinnedTLS(pin: pin, identity: identity)
    calls = tls.session(idle: 15)
  }

  /// The app's API, or nil before pairing.
  static func paired(_ store: Store) -> HerdrApi? {
    guard let pin = store.state.pin, store.state.paired, let id = DeviceKey.identity() else {
      return nil
    }
    return HerdrApi(store: store, pin: pin, identity: id)
  }

  /// The only URL builder: always https.
  static func url(_ host: String, _ port: Int, _ path: String) -> URL {
    let h = host.contains(":") ? "[\(host)]" : host
    return URL(string: "https://\(h):\(port)\(path)")!
  }

  static func segment(_ s: String) -> String {
    s.addingPercentEncoding(withAllowedCharacters: .alphanumerics) ?? ""
  }

  /// The last endpoint that worked first (even if /v1/hello no longer lists it), then the
  /// others in order.
  func hosts() -> [String] {
    let s = store.state
    guard let last = s.lastGood else { return s.endpoints }
    return [last] + s.endpoints.filter { $0 != last }
  }

  func markGood(_ host: String) {
    store.update { $0.lastGood = host }
  }

  /// The Mac's clock, for UUIDv7 request ids.
  func now() -> Int64 {
    Int64(Date().timeIntervalSince1970 * 1000) + store.state.clockOffsetMs
  }

  /// A 2xx body, or an ApiError with the daemon's error body. A 3xx (a refused redirect) is an
  /// error too.
  static func body(_ data: Data, _ resp: URLResponse) throws -> Data {
    guard let http = resp as? HTTPURLResponse else { throw URLError(.badServerResponse) }
    guard (200..<300).contains(http.statusCode) else {
      throw ApiError(status: http.statusCode, body: data)
    }
    return data
  }

  /// Tries each endpoint until one answers. The daemon's own reply ends the search.
  private func get<T: Decodable>(_ path: String) async throws -> T {
    var last: Error?
    for host in hosts() {
      do {
        let (data, resp) = try await calls.data(from: Self.url(host, store.state.port, path))
        let body = try Self.body(data, resp)
        markGood(host)
        return try JSONDecoder().decode(T.self, from: body)
      } catch let e as ApiError {
        markGood(host)
        throw e
      } catch let e as URLError {
        last = e
      }
    }
    throw UnreachableError(last?.localizedDescription ?? "No endpoint is reachable")
  }

  func hello() async throws -> Hello {
    let h: Hello = try await get("/v1/hello")
    store.update {
      $0.clockOffsetMs = h.serverTimeMs - Int64(Date().timeIntervalSince1970 * 1000)
      if !h.endpoints.isEmpty { $0.endpoints = h.endpoints }
    }
    return h
  }

  func spaces() async throws -> [Space] { try await get("/v1/spaces") }
  func agents() async throws -> [Agent] { try await get("/v1/agents") }
  func terminal(_ t: String) async throws -> Agent {
    try await get("/v1/terminals/\(Self.segment(t))")
  }
  func prompt(_ t: String) async throws -> PromptView {
    try await get("/v1/terminals/\(Self.segment(t))/prompt")
  }
  func skills(_ t: String) async throws -> Skills {
    try await get("/v1/terminals/\(Self.segment(t))/skills")
  }
  func transcript(_ t: String, cursor: String?) async throws -> TranscriptPage {
    let q = cursor.flatMap { $0.isEmpty ? nil : "?cursor=\(Self.segment($0))" } ?? ""
    return try await get("/v1/terminals/\(Self.segment(t))/transcript\(q)")
  }

  func decide(_ t: String, _ req: DecisionRequest) async throws -> String {
    try await action("/v1/terminals/\(Self.segment(t))/decision", JSONEncoder().encode(req))
  }

  func message(_ t: String, _ req: MessageRequest) async throws -> String {
    try await action("/v1/terminals/\(Self.segment(t))/message", JSONEncoder().encode(req))
  }

  /// Posts one action (decision 6). The next endpoint is tried only while the request never
  /// started. After that there is no retry of any kind: the error goes up as it is, and the app
  /// shows "Outcome unknown". Each action has its own session, so no stale connection from an
  /// earlier call can make URLSession send it again.
  private func action(_ path: String, _ body: Data) async throws -> String {
    let session = tls.session(idle: 150)
    defer { session.finishTasksAndInvalidate() }
    var last: Error?
    for host in hosts() {
      var req = URLRequest(url: Self.url(host, store.state.port, path))
      req.httpMethod = "POST"
      req.setValue("application/json", forHTTPHeaderField: "Content-Type")
      req.httpBody = body
      let probe = TaskProbe(connect: Self.connectTimeout)
      do {
        let (data, resp) = try await session.data(for: req, delegate: probe)
        let out = try JSONDecoder().decode(ResultBody.self, from: Self.body(data, resp)).result
        markGood(host)
        return out
      } catch let e as ApiError {
        markGood(host)
        throw e
      } catch {
        guard ActionRule.mayTryNext((error as? URLError)?.code, requestStarted: probe.started)
        else { throw error }
        last = error
      }
    }
    throw UnreachableError(last?.localizedDescription ?? "No endpoint is reachable")
  }
}
