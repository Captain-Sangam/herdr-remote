import Foundation

/// One Server-Sent Event from /v1/events.
public enum SseEvent: Equatable, Sendable {
  case snapshot([Agent])
  case changed(Agent)
  case removed(String)
}

/// Splits the bytes of an SSE stream into events: `event:` and `data:` lines, a blank line ends
/// an event, and a line that starts with ":" is a keepalive comment. It reads bytes, because
/// `AsyncBytes.lines` drops the blank lines.
public struct SseParser: Sendable {
  private var line: [UInt8] = []
  private var event = ""
  private var data = ""

  public init() {}

  /// Feeds one byte. Returns the event name and data when a blank line ends an event.
  public mutating func feed(_ byte: UInt8) -> (event: String, data: String)? {
    guard byte == 0x0A else {
      line.append(byte)
      return nil
    }
    if line.last == 0x0D { line.removeLast() }
    let text = String(decoding: line, as: UTF8.self)
    line.removeAll(keepingCapacity: true)
    if text.isEmpty {
      defer {
        event = ""
        data = ""
      }
      return event.isEmpty ? nil : (event, data)
    }
    if text.hasPrefix("event: ") {
      event = String(text.dropFirst(7))
    } else if text.hasPrefix("data: ") {
      if !data.isEmpty { data += "\n" }
      data += text.dropFirst(6)
    }
    return nil
  }
}

public struct AgentsState: Equatable, Sendable {
  public var agents: [String: Agent] = [:]
  /// Phone clock, ms, of the last event. 0 before the first snapshot.
  public var updatedAt: Int64 = 0
  public var connected = false

  public init() {}

  /// Blocked first, then by space, tab and address.
  public var sorted: [Agent] {
    agents.values.sorted {
      ($0.blocked ? 0 : 1, $0.space, $0.tab, $0.address)
        < ($1.blocked ? 0 : 1, $1.space, $1.tab, $1.address)
    }
  }
}

public enum Reducer {
  private struct Removed: Decodable { var address: String }

  /// Nil for an unknown event, so that a newer daemon can add events.
  public static func parse(event: String, data: String) throws -> SseEvent? {
    let d = Data(data.utf8)
    let json = JSONDecoder()
    switch event {
    case "snapshot": return .snapshot(try json.decode([Agent].self, from: d))
    case "agent": return .changed(try json.decode(Agent.self, from: d))
    case "agent_removed": return .removed(try json.decode(Removed.self, from: d).address)
    default: return nil
    }
  }

  /// A snapshot (first, or after `missed`) replaces everything. A change or a removal edits one
  /// agent.
  public static func reduce(_ state: AgentsState, _ ev: SseEvent, now: Int64) -> AgentsState {
    var s = state
    s.updatedAt = now
    switch ev {
    case .snapshot(let agents):
      s.agents = Dictionary(agents.map { ($0.address, $0) }, uniquingKeysWith: { $1 })
      s.connected = true
    case .changed(let a): s.agents[a.address] = a
    case .removed(let address): s.agents[address] = nil
    }
    return s
  }
}

/// What the app does with an agent's alert.
public enum AlertKind: Equatable, Sendable {
  /// A new blocked episode: alert.
  case blocked
  /// The same episode with a new turn: update quietly.
  case refresh
  /// It stopped working (done, idle or error): alert.
  case done
  /// It left blocked some other way: the old alert goes.
  case clear
}

/// The alerts (ADR 0013, ADR 0019), as `watch/Reducer.kt` has them. A blocked episode starts
/// when an agent becomes blocked, or shows another prompt while blocked. Not the turn: hc sends
/// the turn after the blocked status, so one episode has two turns. A reconnect's snapshot with
/// the same prompt alerts for nothing. Done needs a state the app saw before, so the first
/// snapshot never alerts for it. Herdr says `idle`, not `done`, when someone looked at the agent.
public final class Notifier {
  private var last: [String: Agent] = [:]

  public init() {}

  public func alerts(_ agents: [Agent]) -> [(Agent, AlertKind)] {
    agents.compactMap { a in
      let prev = last.updateValue(a, forKey: a.address)
      let alert: AlertKind?
      if a.blocked {
        if prev?.blocked != true || (prev?.msg ?? "") != (a.msg ?? "") {
          alert = .blocked
        } else if prev?.turn != a.turn {
          alert = .refresh
        } else {
          alert = nil
        }
      } else if a.state != "working"
        && (prev?.state == "working" || (prev?.blocked == true && a.state == "done"))
      {
        alert = .done
      } else if prev?.blocked == true {
        alert = .clear
      } else {
        alert = nil
      }
      return alert.map { (a, $0) }
    }
  }

  public func forget(_ address: String) {
    last[address] = nil
  }
}
