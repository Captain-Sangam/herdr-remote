import Foundation
import LocalAuthentication

/// The app gate (ADR 0014): locked at start and after 5 minutes in the background. Nothing shows
/// and no action runs while it is locked.
public struct Gate: Sendable {
  public static let lockAfterMs: Int64 = 5 * 60 * 1000

  public private(set) var locked = true
  private var backgroundedAt: Int64?

  public init() {}

  public mutating func unlock() {
    locked = false
    backgroundedAt = nil
  }

  public mutating func onBackground(nowMs: Int64) {
    backgroundedAt = nowMs
  }

  public mutating func onForeground(nowMs: Int64) {
    if Self.shouldLock(locked: locked, backgroundedAt: backgroundedAt, nowMs: nowMs) {
      locked = true
    }
    backgroundedAt = nil
  }

  public static func shouldLock(locked: Bool, backgroundedAt: Int64?, nowMs: Int64) -> Bool {
    locked || backgroundedAt.map { nowMs - $0 >= lockAfterMs } ?? false
  }

  /// What the gate does with the result of `LAContext.canEvaluatePolicy`.
  public enum Check: Equatable, Sendable {
    case prompt, open, error
  }

  /// Only a device with no passcode at all opens without a prompt. Any other error keeps the
  /// gate locked. `nil` means that the policy can be evaluated.
  public static func check(_ error: LAError.Code?) -> Check {
    switch error {
    case nil: .prompt
    case .passcodeNotSet: .open
    default: .error
    }
  }
}

/// Decision 6 of the plan: an action tries the next endpoint only if the request never started.
/// That is a connect-phase error, and no `requestStartDate` in the task's metrics. An unknown
/// error counts as started: the app then shows "Outcome unknown" and never posts again.
public enum ActionRule {
  static let connectPhase: Set<URLError.Code> = [
    .cannotConnectToHost, .cannotFindHost, .dnsLookupFailed, .notConnectedToInternet,
    .timedOut, .secureConnectionFailed, .serverCertificateUntrusted,
    .serverCertificateHasBadDate, .serverCertificateHasUnknownRoot,
    .serverCertificateNotYetValid, .clientCertificateRejected, .clientCertificateRequired,
    .cancelled, .internationalRoamingOff, .dataNotAllowed, .callIsActive,
  ]

  /// `code` is the URLError code, or nil for any other error.
  public static func mayTryNext(_ code: URLError.Code?, requestStarted: Bool) -> Bool {
    guard !requestStarted, let code else { return false }
    return connectPhase.contains(code)
  }
}
