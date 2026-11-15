import Core
import Foundation
import Security

/// TLS for every call to the Mac (ADR 0025, decision 4). The session presents the device
/// identity, accepts only a server key whose SPKI SHA-256 equals the pin, needs TLS 1.3 and
/// refuses every redirect. Certificate names and addresses play no part, so LAN and Tailscale
/// endpoints work the same way.
final class PinnedTLS: NSObject, URLSessionTaskDelegate, @unchecked Sendable {
  let pin: String
  let identity: SecIdentity

  init(pin: String, identity: SecIdentity) {
    self.pin = pin
    self.identity = identity
  }

  /// A new session. `idle` is the read timeout: the most time between two packets.
  func session(idle: TimeInterval) -> URLSession {
    let c = URLSessionConfiguration.ephemeral
    c.tlsMinimumSupportedProtocolVersion = .TLSv13
    c.timeoutIntervalForRequest = idle
    c.timeoutIntervalForResource = .infinity
    c.urlCache = nil
    c.requestCachePolicy = .reloadIgnoringLocalCacheData
    c.httpCookieStorage = nil
    c.httpShouldSetCookies = false
    c.urlCredentialStorage = nil
    return URLSession(configuration: c, delegate: self, delegateQueue: nil)
  }

  func urlSession(
    _ session: URLSession, task: URLSessionTask, didReceive challenge: URLAuthenticationChallenge
  ) async -> (URLSession.AuthChallengeDisposition, URLCredential?) {
    switch challenge.protectionSpace.authenticationMethod {
    case NSURLAuthenticationMethodServerTrust:
      guard let trust = challenge.protectionSpace.serverTrust, Self.matches(trust, pin: pin) else {
        return (.cancelAuthenticationChallenge, nil)
      }
      return (.useCredential, URLCredential(trust: trust))
    case NSURLAuthenticationMethodClientCertificate:
      return (
        .useCredential, URLCredential(identity: identity, certificates: nil, persistence: .none)
      )
    default:
      return (.cancelAuthenticationChallenge, nil)
    }
  }

  /// No redirect: the reply is the 3xx itself, which the caller treats as an error.
  func urlSession(
    _ session: URLSession, task: URLSessionTask,
    willPerformHTTPRedirection response: HTTPURLResponse,
    newRequest request: URLRequest
  ) async -> URLRequest? {
    nil
  }

  /// The server's P-256 point: the leaf certificate's key in X9.63 form.
  static func serverPoint(_ trust: SecTrust) -> Data? {
    guard let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate],
      let leaf = chain.first, let key = SecCertificateCopyKey(leaf)
    else { return nil }
    return SecKeyCopyExternalRepresentation(key, nil) as Data?
  }

  static func matches(_ trust: SecTrust, pin: String) -> Bool {
    guard let point = serverPoint(trust) else { return false }
    return Spki.matches(p256Point: point, pin: pin)
  }
}

/// A per-task delegate. It records whether the request started (decision 6), and it ends a
/// task that has not connected after `connect` seconds: URLSession has no connect timeout of
/// its own, only a read timeout.
final class TaskProbe: NSObject, URLSessionTaskDelegate, @unchecked Sendable {
  private let lock = NSLock()
  private var _started = false
  private let connect: TimeInterval?

  /// True once a transaction has a `requestStartDate`: the Mac may have the request.
  var started: Bool { lock.withLock { _started } }

  init(connect: TimeInterval?) {
    self.connect = connect
  }

  func urlSession(_ session: URLSession, didCreateTask task: URLSessionTask) {
    guard let connect else { return }
    DispatchQueue.global().asyncAfter(deadline: .now() + connect) {
      // Nothing sent and nothing received: the connection never came up.
      if task.state == .running, task.response == nil, task.countOfBytesSent == 0,
        task.countOfBytesReceived == 0
      {
        task.cancel()
      }
    }
  }

  func urlSession(
    _ session: URLSession, task: URLSessionTask, didFinishCollecting metrics: URLSessionTaskMetrics
  ) {
    let started = metrics.transactionMetrics.contains { $0.requestStartDate != nil }
    lock.withLock { _started = started }
  }
}
