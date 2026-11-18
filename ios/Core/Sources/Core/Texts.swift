import Foundation

/// How a note looks: Ok when applied or sent, Warn when not confirmed, Error for the rest.
public enum NoteKind: Equatable, Sendable {
  case ok, warn, error
}

/// What the app says for each result and each 409 reason. The words are those of
/// `ui/AgentViewModel.kt` in the Android app.
public enum Texts {
  public static func result(_ result: String) -> String {
    switch result {
    case "applied": "Done. The agent continues."
    case "sent_unconfirmed": "Sent. The agent has not confirmed it yet."
    case "uncertain":
      "Someone used the Mac at the same moment. Check the transcript before you act again."
    case "sent": "Sent to the agent."
    case "outcome_unknown": "Outcome unknown. Check the transcript before you send again."
    default: result
    }
  }

  public static func error(_ e: Error) -> String {
    if let e = e as? ApiError {
      switch e.reason {
      case "stale": return "The prompt has moved on. Reload it."
      case "changed": return "The prompt changed. Reload it."
      case "already_answered": return "This prompt was already answered."
      case "mac_in_use":
        return
          "Someone is using the Mac right now (idle \((e.idleMs ?? 0) / 1000) s). Try again when it is free."
      case "not_actionable":
        return "This prompt cannot be answered from the phone. Answer it on the Mac."
      case "blocked": return "The agent is waiting on a prompt. Answer that first."
      case "not_agent": return "This terminal is not an agent."
      case "expired_request": return "The phone's clock differs from the Mac's. Try again."
      case "cursor": return "The transcript changed. Start again from the end."
      default:
        switch e.status {
        case 429: return "Too fast. Wait a second and try again."
        case 503: return "The Mac cannot reach Herdr right now."
        default: return e.error
        }
      }
    }
    if e is UnreachableError { return "The Mac is not reachable." }
    return result("outcome_unknown")
  }

  public static func kind(_ note: String) -> NoteKind {
    switch note {
    case result("applied"), result("sent"): .ok
    case result("sent_unconfirmed"), result("uncertain"), result("outcome_unknown"): .warn
    default: .error
    }
  }

  /// After an action: the daemon's own reply keeps its reason. Any other failure may have
  /// reached the Mac.
  public static func actionError(_ e: Error) -> String {
    e is ApiError ? error(e) : result("outcome_unknown")
  }

  /// The note when a prompt refresh fails: a Reload that the user asked for reports it. The
  /// refresh after an action keeps the action's note.
  public static func refreshError(current: String?, _ e: Error, manual: Bool) -> String? {
    manual ? error(e) : current
  }
}

/// The newest page on top of the items the app holds. A transcript only grows, so the page
/// starts inside `old`: what lies before it (the earlier pages the user loaded) stays. Nil when
/// they do not overlap (another file, or terminal text).
public func mergeTail(_ old: [TranscriptItem], _ page: [TranscriptItem]) -> [TranscriptItem]? {
  if old.isEmpty { return page }
  for i in old.indices where old.count - i <= page.count {
    let n = old.count - i
    if old[i...].elementsEqual(page[..<n]) { return Array(old[..<i]) + page }
  }
  return nil
}

/// A transcript list entry: one message, or a run of tool steps in one fold.
public struct Entry: Equatable, Sendable {
  public let items: [TranscriptItem]
  public let steps: Bool

  public init(_ items: [TranscriptItem], steps: Bool) {
    self.items = items
    self.steps = steps
  }
}

/// Folds a transcript: every text shows, and each run of tool calls and results between two texts
/// goes into one fold.
public func fold(_ items: [TranscriptItem]) -> [Entry] {
  var out: [Entry] = []
  var run: [TranscriptItem] = []
  for it in items {
    if it.kind == "text" {
      if !run.isEmpty { out.append(Entry(run, steps: true)) }
      run.removeAll()
      out.append(Entry([it], steps: false))
    } else {
      run.append(it)
    }
  }
  if !run.isEmpty { out.append(Entry(run, steps: true)) }
  return out
}

/// How the phone reaches an endpoint: the daemon lists private addresses, then Tailscale ones (a
/// 100.64.0.0/10 address or the MagicDNS name).
public func endpointKind(_ endpoint: String) -> String {
  let o = endpoint.split(separator: ".", omittingEmptySubsequences: false)
  let cgnat = o.count == 4 && o[0] == "100" && (64...127).contains(Int(o[1]) ?? 0)
  return cgnat || endpoint.hasSuffix(".ts.net") ? "Tailscale" : "Wi-Fi"
}
