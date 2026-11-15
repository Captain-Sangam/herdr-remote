import Foundation
import Testing

@testable import Core

// TextsTest.kt.

func items(_ t: String...) -> [TranscriptItem] {
  t.map { TranscriptItem(role: "assistant", kind: "text", text: $0, ts: 1) }
}

@Test func newestPageMergesOntoLoadedItems() {
  // The window moved on by one item: the earlier pages stay, the new item comes after.
  #expect(
    mergeTail(items("a", "b", "c", "d"), items("c", "d", "e")) == items("a", "b", "c", "d", "e"))
  // The same page: nothing changes.
  #expect(mergeTail(items("a", "b"), items("a", "b")) == items("a", "b"))
  // Repeated items: the longest overlap wins, so nothing shows twice.
  #expect(
    mergeTail(items("x", "y", "x", "y"), items("x", "y", "x", "y", "z"))
      == items("x", "y", "x", "y", "z"))
  // Nothing loaded yet: the page.
  #expect(mergeTail([], items("a")) == items("a"))
  // No overlap (another file, or terminal text): nil, so the caller replaces.
  #expect(mergeTail(items("a", "b"), items("c")) == nil)
}

@Test func stepsFoldBetweenThePromptAndTheAnswer() {
  let ask = TranscriptItem(role: "user", kind: "text", text: "ask")
  let note = TranscriptItem(role: "assistant", kind: "text", text: "let me look")
  let call = TranscriptItem(role: "assistant", kind: "tool_call", text: "Bash ls")
  let out = TranscriptItem(role: "user", kind: "tool_result", text: "a b")
  let answer = TranscriptItem(role: "assistant", kind: "text", text: "done")
  let run = TranscriptItem(role: "assistant", kind: "tool_call", text: "Bash make")
  #expect(
    fold([ask, note, call, out, answer, ask, note, run]) == [
      Entry([ask], steps: false),
      Entry([note, call, out], steps: true),
      Entry([answer], steps: false),
      // A turn that runs: all of it folds.
      Entry([ask], steps: false),
      Entry([note, run], steps: true),
    ])
  // No tool steps (terminal text, or a plain answer): nothing folds.
  #expect(fold([answer]) == [Entry([answer], steps: false)])
}

@Test func everyResultAndReasonHasWords() {
  for r in ["applied", "sent_unconfirmed", "uncertain", "outcome_unknown", "sent"] {
    #expect(Texts.result(r).count > 5 && Texts.result(r) != r, "\(r)")
  }
  #expect(Texts.result("outcome_unknown").contains("Check the transcript before you send again"))
  #expect(Texts.result("uncertain").contains("Check the transcript"))
  for reason in [
    "stale", "changed", "already_answered", "mac_in_use", "not_actionable", "blocked", "not_agent",
    "expired_request", "cursor",
  ] {
    let t = Texts.error(ApiError(status: 409, error: "x", reason: reason, idleMs: 4200))
    #expect(t.count > 10 && !t.contains("_"), "\(reason)")
  }
  #expect(
    Texts.error(ApiError(status: 409, error: "x", reason: "mac_in_use", idleMs: 4200))
      == "Someone is using the Mac right now (idle 4 s). Try again when it is free.")
  #expect(Texts.error(ApiError(status: 429, error: "too many actions")).contains("Too fast"))
  #expect(Texts.error(ApiError(status: 503, error: "hc unavailable")).contains("Herdr"))
  #expect(Texts.error(UnreachableError("x")).contains("not reachable"))
  #expect(Texts.error(URLError(.badServerResponse)).contains("Outcome unknown"))
}

@Test func anActionKeepsTheDaemonsReasonAndNothingElse() {
  #expect(
    Texts.actionError(ApiError(status: 409, error: "x", reason: "stale"))
      == "The prompt has moved on. Reload it.")
  #expect(
    Texts.actionError(ApiError(status: 503, error: "hc unavailable"))
      == "The Mac cannot reach Herdr right now.")
  let errors: [Error] = [
    UnreachableError("x"), URLError(.timedOut), URLError(.networkConnectionLost),
    CocoaError(.fileNoSuchFile),
  ]
  for e in errors {
    #expect(
      Texts.actionError(e) == "Outcome unknown. Check the transcript before you send again.",
      "\(e)")
  }
}

@Test func theRefreshAfterAnActionKeepsTheActionsNote() {
  let note = Texts.result("outcome_unknown")
  #expect(Texts.refreshError(current: note, UnreachableError("x"), manual: false) == note)
  #expect(Texts.refreshError(current: nil, UnreachableError("x"), manual: false) == nil)
  #expect(
    Texts.refreshError(current: note, UnreachableError("x"), manual: true)
      == "The Mac is not reachable.")
}

@Test func aNoteLooksLikeItsOutcome() {
  #expect(Texts.kind(Texts.result("applied")) == .ok)
  #expect(Texts.kind(Texts.result("sent")) == .ok)
  for r in ["sent_unconfirmed", "uncertain", "outcome_unknown"] {
    #expect(Texts.kind(Texts.result(r)) == .warn, "\(r)")
  }
  #expect(
    Texts.kind(Texts.error(ApiError(status: 409, error: "x", reason: "mac_in_use", idleMs: 4200)))
      == .error)
  #expect(Texts.kind("Unlock the app first.") == .error)
}

@Test func endpointsAreWifiOrTailscale() {
  #expect(endpointKind("192.168.1.24") == "Wi-Fi")
  #expect(endpointKind("10.0.0.5") == "Wi-Fi")
  #expect(endpointKind("100.101.2.3") == "Tailscale")
  #expect(endpointKind("100.128.2.3") == "Wi-Fi")
  #expect(endpointKind("zeus-mbp.tail3c2e.ts.net") == "Tailscale")
}

@Test func anErrorBodyGivesItsReason() {
  let e = ApiError(
    status: 409, body: Data(#"{"error":"conflict","reason":"mac_in_use","idle_ms":4200}"#.utf8))
  #expect(e == ApiError(status: 409, error: "conflict", reason: "mac_in_use", idleMs: 4200))
  #expect(ApiError(status: 502, body: Data("bad gateway".utf8)).error == "bad gateway")
  #expect(ApiError(status: 500, body: Data()).error == "HTTP 500")
  #expect(ApiError(status: 404, body: Data("{}".utf8)).error == "HTTP 404")
}
