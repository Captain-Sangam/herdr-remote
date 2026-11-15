import Foundation
import Testing

@testable import Core

// ReducerTest.kt, and the SSE parser.

let snapshot = """
  [{"address":"t1","space":"Niti","tab":"main","where":"Niti / main","app":"claude-code","state":"blocked","kind":"permission","msg":"approve Bash: touch x","turn":2,"title":"t"},
  {"address":"t3","space":"","tab":"","where":"Email / codex","app":"codex","state":"idle","turn":5,"title":""}]
  """

func parse(_ event: String, _ data: String) throws -> SseEvent {
  try #require(try Reducer.parse(event: event, data: data))
}

@Test func snapshotChangeRemovedAndMissed() throws {
  var s = AgentsState()
  s = Reducer.reduce(s, try parse("snapshot", snapshot), now: 1000)
  #expect(Set(s.agents.keys) == ["t1", "t3"])
  #expect(s.connected)
  #expect(s.agents["t1"]?.kind == "permission")
  #expect(s.updatedAt == 1000)

  s = Reducer.reduce(
    s,
    try parse(
      "agent",
      #"{"address":"t3","app":"codex","state":"blocked","kind":"permission","turn":6,"title":""}"#),
    now: 2000)
  #expect(s.agents["t3"]?.state == "blocked")
  #expect(s.agents["t3"]?.turn == 6)
  #expect(s.updatedAt == 2000)
  // Both blocked: the empty space name sorts first.
  #expect(s.sorted.map(\.address) == ["t3", "t1"])

  s = Reducer.reduce(s, try parse("agent_removed", #"{"address":"t1"}"#), now: 3000)
  #expect(Set(s.agents.keys) == ["t3"])

  // missed: a fresh snapshot replaces everything that was there.
  s = Reducer.reduce(
    s,
    try parse("snapshot", #"[{"address":"t9","app":"pi","state":"working","turn":1,"title":""}]"#),
    now: 4000)
  #expect(Set(s.agents.keys) == ["t9"])
  #expect(try Reducer.parse(event: "bogus", data: "{}") == nil)
}

@Test func blockedFirstInSortedList() throws {
  var s = Reducer.reduce(AgentsState(), try parse("snapshot", snapshot), now: 1)
  var t3 = try #require(s.agents["t3"])
  t3.state = "blocked"
  t3.kind = "permission"
  s.agents["t3"] = t3
  s.agents["t0"] = Agent(address: "t0", state: "idle")
  #expect(s.sorted.map(\.address) == ["t3", "t1", "t0"])
  #expect(s.sorted.prefix { $0.blocked }.count == 2)
}

@Test func oneNotificationPerEpisode() throws {
  let n = Notifier()
  func alerts(_ a: Agent...) -> [String] { n.alerts(a).map { "\($0.0.address) \($0.1)" } }
  guard case .snapshot(let first) = try parse("snapshot", snapshot) else { return }
  // The first snapshot: the blocked agent alerts, the idle one does not.
  #expect(n.alerts(first).map { "\($0.0.address) \($0.1)" } == ["t1 blocked"])
  #expect(n.alerts(first).isEmpty)
  let again = try #require(first.first { $0.address == "t1" })
  func with(turn: Int64, state: String = "blocked", msg: String? = nil) -> Agent {
    var a = again
    a.turn = turn
    a.state = state
    if let msg { a.msg = msg }
    return a
  }
  // hc sends the turn after the blocked status: the same episode, a quiet refresh.
  #expect(alerts(with(turn: 3)) == ["t1 refresh"])
  #expect(alerts(with(turn: 3)) == [])
  // Another prompt while still blocked: another episode.
  #expect(alerts(with(turn: 3, msg: "approve Write: y")) == ["t1 blocked"])
  // Approved: working clears the prompt. Then done, or idle when someone looked, alerts.
  #expect(alerts(with(turn: 4, state: "working")) == ["t1 clear"])
  #expect(alerts(with(turn: 5, state: "idle")) == ["t1 done"])
  #expect(alerts(with(turn: 5, state: "done")) == [])
  // The next block is a new episode, even with the same prompt.
  #expect(alerts(with(turn: 5)) == ["t1 blocked"])
  // Denied: idle, no finish alert.
  #expect(alerts(with(turn: 5, state: "idle")) == ["t1 clear"])
  // Blocked straight to done: finished.
  _ = alerts(with(turn: 6))
  #expect(alerts(with(turn: 7, state: "done")) == ["t1 done"])
  // An agent gone and back: like a first sight.
  n.forget("t1")
  #expect(alerts(with(turn: 2, state: "done")) == [])
}

@Test func sseParserSplitsEventsAndSkipsKeepalives() {
  var p = SseParser()
  var out: [String] = []
  let stream =
    ": ka\n\nevent: snapshot\ndata: [1,\ndata: 2]\n\nevent: agent\r\ndata: {}\r\n\r\n: ka\n\ndata: x\n\n"
  for b in stream.utf8 {
    if let e = p.feed(b) { out.append("\(e.event)=\(e.data)") }
  }
  #expect(out == ["snapshot=[1,\n2]", "agent={}"])
}

@Test func agentDefaultsAndNames() throws {
  let a = try JSONDecoder().decode(Agent.self, from: Data(#"{"address":"t7"}"#.utf8))
  #expect(a.state == "" && a.turn == 0 && a.kind == nil)
  #expect(a.label == "t7")
  #expect(a.alertName == "t7")
  var b = a
  b.tab = "main"
  b.where = "Niti / main"
  #expect(b.label == "main")
  #expect(b.alertName == "Niti / main")
}
