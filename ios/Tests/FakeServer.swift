import Foundation
import Network
import Security

/// A TLS server on 127.0.0.1 for the transport tests. Each connection reads one request, counts
/// it, and gets `reply` (raw HTTP), or is closed with no reply when `reply` is nil.
final class FakeServer: @unchecked Sendable {
  let listener: NWListener
  private let queue = DispatchQueue(label: "fake-server")
  private let lock = NSLock()
  private var _requests = 0
  var reply: String?

  var requests: Int { lock.withLock { _requests } }
  var port: Int { Int(listener.port?.rawValue ?? 0) }

  init(identity: SecIdentity, tls: tls_protocol_version_t = .TLSv13, reply: String?) throws {
    let opts = NWProtocolTLS.Options()
    let sec = opts.securityProtocolOptions
    sec_protocol_options_set_local_identity(sec, sec_identity_create(identity)!)
    sec_protocol_options_set_min_tls_protocol_version(sec, tls)
    sec_protocol_options_set_max_tls_protocol_version(sec, tls)
    let params = NWParameters(tls: opts)
    params.requiredLocalEndpoint = NWEndpoint.hostPort(host: "127.0.0.1", port: .any)
    listener = try NWListener(using: params)
    self.reply = reply
    let ready = DispatchSemaphore(value: 0)
    listener.stateUpdateHandler = { if case .ready = $0 { ready.signal() } }
    listener.newConnectionHandler = { [weak self] c in self?.serve(c) }
    listener.start(queue: queue)
    _ = ready.wait(timeout: .now() + 5)
  }

  private func serve(_ c: NWConnection) {
    c.start(queue: queue)
    c.receive(minimumIncompleteLength: 1, maximumLength: 65536) { [weak self] data, _, _, _ in
      guard let self, data != nil else { return c.cancel() }
      self.lock.withLock { self._requests += 1 }
      guard let reply = self.reply else { return c.cancel() }
      c.send(
        content: Data(reply.utf8),
        completion: .contentProcessed { _ in c.cancel() })
    }
  }

  func stop() { listener.cancel() }

  static func ok(_ body: String) -> String {
    "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: \(body.utf8.count)\r\n"
      + "Connection: close\r\n\r\n\(body)"
  }
}
