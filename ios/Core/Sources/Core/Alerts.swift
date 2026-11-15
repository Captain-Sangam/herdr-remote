import Foundation

/// The notification texts (decision 10 of the plan). A title holds the agent's name and what it
/// needs. No title or body holds the prompt or `msg`.
public enum AlertText {
  public static func blocked(_ name: String, approvable: Bool) -> String {
    "\(name) needs \(approvable ? "approval" : "attention")"
  }

  public static func done(_ a: Agent) -> String {
    "\(a.alertName) \(a.state == "error" ? "stopped with an error" : "finished")"
  }

  public static func cleared(_ name: String) -> String { "\(name) no longer waits" }

  /// The generic titles of a push, before the service extension adds the name.
  public static let genericBlocked = "An agent needs attention"
  public static let genericDone = "An agent finished"
  public static let genericCleared = "An agent no longer waits"
}

/// The fields of a push (decision 14 of the plan): the address and the kind, beside `aps`. No
/// name and no content.
public struct PushPayload: Equatable, Sendable {
  public enum Kind: String, Sendable {
    case blocked, done, clear
  }

  public let address: String
  public let kind: Kind

  /// Nil when a field is missing or unknown: the alert then keeps its generic title.
  public init?(_ userInfo: [AnyHashable: Any]) {
    guard let address = userInfo["address"] as? String, !address.isEmpty,
      let kind = (userInfo["kind"] as? String).flatMap(Kind.init(rawValue:))
    else { return nil }
    self.address = address
    self.kind = kind
  }

  /// The title with the agent's name, from a fresh `/v1/terminals/{t}`. Without a permission
  /// prompt, a blocked agent needs attention.
  public func title(_ a: Agent) -> String {
    switch kind {
    case .blocked: AlertText.blocked(a.alertName, approvable: a.kind == "permission")
    case .done: AlertText.done(a)
    case .clear: AlertText.cleared(a.alertName)
    }
  }
}
