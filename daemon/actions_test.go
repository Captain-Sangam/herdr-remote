package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type actFixture struct {
	d    *testDaemon
	cert tls.Certificate
	dev  Device
}

func screenFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func setupActions(t *testing.T) *actFixture {
	t.Helper()
	d := startTestDaemon(t)
	d.actions.settle = 300 * time.Millisecond
	d.actions.perSec = 1000 // TestRateLimitThirdActionInASecond sets the real limit
	claude := liveAgent("t1", "claude-code", "blocked", 1)
	claude.Status.Kind, claude.Status.Msg = "permission", "approve Bash: touch x"
	d.f.setTerminal(claude)
	d.f.setRead("t1", screenFixture(t, "screen-claude-cmd.txt"))
	codex := liveAgent("t2", "codex", "blocked", 1)
	codex.Status.Kind = "permission"
	d.f.setTerminal(codex)
	d.f.setRead("t2", screenFixture(t, "screen-codex-cmd.txt"))
	idle := liveAgent("t3", "claude-code", "idle", 4)
	d.f.setTerminal(idle)
	d.f.setRead("t3", "❯ \n")
	shell := liveAgent("t8", "", "idle", 0)
	shell.Reported, shell.Status = false, nil
	d.f.setTerminal(shell)
	f := &actFixture{d: d}
	f.cert, f.dev = d.pairPhone(t, "p")
	d.waitOK(t)
	return f
}

func uuid7(at time.Time) string {
	var r [10]byte
	rand.Read(r[:])
	ms := uint64(at.UnixMilli())
	return fmt.Sprintf("%08x-%04x-7%03x-%04x-%012x", ms>>16, ms&0xffff, uint16(r[0])<<4|uint16(r[1])>>4&0xfff, 0x8000|uint16(r[2])&0x3fff, r[3:9])
}

func (f *actFixture) post(t *testing.T, path string, body any, timeout time.Duration) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	c := f.d.client(&f.cert, nil)
	if timeout > 0 {
		c.Timeout = timeout
	}
	resp, err := c.Post(f.d.pairedURL(path), "application/json", bytes.NewReader(b))
	if err != nil {
		return 0, map[string]any{"error": err.Error()}
	}
	defer resp.Body.Close()
	var m map[string]any
	rb, _ := io.ReadAll(resp.Body)
	json.Unmarshal(rb, &m)
	return resp.StatusCode, m
}

func (f *actFixture) prompt(t *testing.T, addr string) promptView {
	t.Helper()
	st, b := f.d.get(t, &f.cert, "/v1/terminals/"+addr+"/prompt")
	if st != 200 {
		t.Fatalf("prompt %s: %d %s", addr, st, b)
	}
	var v promptView
	json.Unmarshal(b, &v)
	return v
}

func (f *actFixture) decide(t *testing.T, addr, id, decision string, turn int64, hash string) (int, map[string]any) {
	t.Helper()
	return f.post(t, "/v1/terminals/"+addr+"/decision", map[string]any{"request_id": id, "decision": decision, "turn": turn, "prompt_hash": hash}, 0)
}

func (f *actFixture) message(t *testing.T, addr, id, text string) (int, map[string]any) {
	t.Helper()
	return f.post(t, "/v1/terminals/"+addr+"/message", map[string]any{"request_id": id, "text": text}, 0)
}

func (f *actFixture) keys() []fakeReq { return f.d.f.requestsFor("terminal.keys") }

func (f *actFixture) audit(t *testing.T) string {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(f.d.dir, "audit.log"))
	return string(b)
}

func TestPromptBlocksAndHashes(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	if !v.Actionable || v.Reason != "" || v.App != "claude-code" || v.Kind != "permission" || v.Msg != "approve Bash: touch x" || v.Turn != 1 || len(v.PromptHash) != 64 {
		t.Fatalf("%+v", v)
	}
	if !strings.HasPrefix(strings.TrimSpace(v.Block), "Do you want to proceed?") || !strings.HasSuffix(v.Block, "Esc to cancel · Tab to amend") || !strings.Contains(v.Block, "4. No") {
		t.Fatalf("block %q", v.Block)
	}
	if strings.Contains(v.Block, "Bash command") {
		t.Fatalf("block starts too early: %q", v.Block)
	}
	c := f.prompt(t, "t2")
	if !c.Actionable || !strings.HasPrefix(strings.TrimSpace(c.Block), "Would you like to run the following command?") || !strings.HasSuffix(c.Block, "Press enter to confirm or esc to cancel") || !strings.Contains(c.Block, "Yes, proceed (y)") {
		t.Fatalf("%+v", c)
	}
	if c.PromptHash == v.PromptHash || c.Msg != "" {
		t.Fatal("hash or msg wrong")
	}
	// The hash follows the turn.
	cl := liveAgent("t1", "claude-code", "blocked", 2)
	cl.Status.Kind, cl.Status.Msg = "permission", "approve Bash: touch x"
	f.d.f.setTerminal(cl)
	if v2 := f.prompt(t, "t1"); v2.PromptHash == v.PromptHash || v2.Turn != 2 {
		t.Fatal("hash ignores the turn")
	}
	// Not blocked, a question, an unknown app, missing markers: view only.
	idle := f.prompt(t, "t3")
	if idle.Actionable || idle.Reason != "not_blocked" || idle.Block != "" {
		t.Fatalf("%+v", idle)
	}
	q := liveAgent("t1", "claude-code", "blocked", 1)
	q.Status.Kind, q.Status.Msg = "question", "Which option?"
	f.d.f.setTerminal(q)
	if p := f.prompt(t, "t1"); p.Actionable || p.Reason != "view_only" {
		t.Fatalf("%+v", p)
	}
	pi := liveAgent("t1", "pi", "blocked", 1)
	pi.Status.Kind = "permission"
	f.d.f.setTerminal(pi)
	if p := f.prompt(t, "t1"); p.Actionable || p.Reason != "view_only" {
		t.Fatalf("%+v", p)
	}
	f.d.f.setTerminal(claudeBlocked(1))
	f.d.f.setRead("t1", "Do you want to proceed?\n  something else\n")
	if p := f.prompt(t, "t1"); p.Actionable || p.Reason != "no_markers" || p.Block != "" {
		t.Fatalf("%+v", p)
	}
	auth := liveAgent("t1", "claude-code", "blocked", 1)
	auth.Status.Kind = "auth"
	f.d.f.setTerminal(auth)
	if p := f.prompt(t, "t1"); p.Actionable || p.Reason != "view_only" {
		t.Fatalf("%+v", p)
	}
}

func claudeBlocked(turn int64) Terminal {
	c := liveAgent("t1", "claude-code", "blocked", turn)
	c.Status.Kind, c.Status.Msg = "permission", "approve Bash: touch x"
	return c
}

func TestDecisionPressesTheRightKey(t *testing.T) {
	f := setupActions(t)
	f.d.f.onKeys("t1", `{"status":{"app":"claude-code","state":"done"},"turn":2}`)
	v := f.prompt(t, "t1")
	st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash)
	if st != 200 || res["result"] != "applied" {
		t.Fatalf("%d %v", st, res)
	}
	c := f.prompt(t, "t2")
	st, res = f.decide(t, "t2", uuid7(time.Now()), "deny", c.Turn, c.PromptHash)
	if st != 200 || res["result"] != "sent_unconfirmed" {
		t.Fatalf("%d %v", st, res)
	}
	// Codex approve, on a fresh turn.
	cx := liveAgent("t2", "codex", "blocked", 7)
	cx.Status.Kind = "permission"
	f.d.f.setTerminal(cx)
	c = f.prompt(t, "t2")
	if st, res = f.decide(t, "t2", uuid7(time.Now()), "approve", 7, c.PromptHash); st != 200 {
		t.Fatalf("%d %v", st, res)
	}
	keys := f.keys()
	if len(keys) != 3 {
		t.Fatalf("%d keys", len(keys))
	}
	for i, want := range []string{"1", "esc", "y"} {
		if k := keys[i].Params["keys"].([]any); k[0] != want {
			t.Fatalf("key %d = %v, want %s", i, k, want)
		}
	}
	a := f.audit(t)
	if strings.Index(a, `"state":"dispatching"`) > strings.Index(a, `"state":"result"`) || !strings.Contains(a, `"result":"applied"`) || !strings.Contains(a, `"decision":"approve"`) {
		t.Fatalf("audit: %s", a)
	}
	if strings.Contains(a, "Do you want") || strings.Contains(a, "touch x") {
		t.Fatalf("audit holds screen text: %s", a)
	}
}

func TestDecisionGuardsPressNoKey(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	check := func(name string, wantStatus int, wantReason string, call func() (int, map[string]any)) {
		t.Helper()
		st, res := call()
		if st != wantStatus || (wantReason != "" && res["reason"] != wantReason) {
			t.Fatalf("%s: %d %v", name, st, res)
		}
	}
	check("stale turn", 409, "stale", func() (int, map[string]any) {
		return f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn+1, v.PromptHash)
	})
	check("changed hash", 409, "changed", func() (int, map[string]any) {
		return f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, strings.Repeat("0", 64))
	})
	check("remote target", 400, "", func() (int, map[string]any) {
		return f.decide(t, "box%2Ft4", uuid7(time.Now()), "approve", 1, v.PromptHash)
	})
	check("missing terminal", 404, "", func() (int, map[string]any) { return f.decide(t, "t99", uuid7(time.Now()), "approve", 1, v.PromptHash) })
	check("idle agent", 409, "not_actionable", func() (int, map[string]any) { return f.decide(t, "t3", uuid7(time.Now()), "approve", 4, v.PromptHash) })
	check("bad decision", 400, "", func() (int, map[string]any) {
		return f.decide(t, "t1", uuid7(time.Now()), "maybe", v.Turn, v.PromptHash)
	})
	check("bad id", 400, "", func() (int, map[string]any) { return f.decide(t, "t1", "not-a-uuid", "approve", v.Turn, v.PromptHash) })

	f.d.f.setRead("t1", strings.Replace(screenFixture(t, "screen-claude-cmd.txt"), "1. Yes", "1. Sure", 1))
	check("missing markers", 409, "not_actionable", func() (int, map[string]any) {
		return f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash)
	})
	f.d.f.setRead("t1", screenFixture(t, "screen-claude-cmd.txt"))

	q := claudeBlocked(1)
	q.Status.Kind = "question"
	f.d.f.setTerminal(q)
	check("question", 409, "not_actionable", func() (int, map[string]any) { return f.decide(t, "t1", uuid7(time.Now()), "approve", 1, v.PromptHash) })
	pi := liveAgent("t1", "pi", "blocked", 1)
	pi.Status.Kind = "permission"
	f.d.f.setTerminal(pi)
	check("pi", 409, "not_actionable", func() (int, map[string]any) { return f.decide(t, "t1", uuid7(time.Now()), "approve", 1, v.PromptHash) })
	f.d.f.setTerminal(claudeBlocked(1))
	// Answered on the Mac first: the prompt is gone.
	done := liveAgent("t1", "claude-code", "done", 2)
	f.d.f.setTerminal(done)
	check("answered on the Mac", 409, "not_actionable", func() (int, map[string]any) {
		return f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash)
	})

	if n := len(f.keys()); n != 0 {
		t.Fatalf("%d keys pressed", n)
	}
	if !strings.Contains(f.audit(t), `"action":"refusal"`) {
		t.Fatal("refusals not audited")
	}
}

func TestDecisionConcurrentPressesOneKey(t *testing.T) {
	f := setupActions(t)
	f.d.f.setKeysDelay(300 * time.Millisecond)
	v := f.prompt(t, "t1")
	var wg sync.WaitGroup
	results := make([]int, 2)
	reasons := make([]any, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash)
			results[i], reasons[i] = st, res["reason"]
		}(i)
	}
	wg.Wait()
	ok, refused := 0, 0
	for i := range results {
		switch {
		case results[i] == 200:
			ok++
		case results[i] == 409 && reasons[i] == "already_answered":
			refused++
		}
	}
	if ok != 1 || refused != 1 {
		t.Fatalf("results %v %v", results, reasons)
	}
	if n := len(f.keys()); n != 1 {
		t.Fatalf("%d keys", n)
	}
	// A second phone with a fresh id on the same prompt is refused too.
	if st, res := f.decide(t, "t1", uuid7(time.Now()), "deny", v.Turn, v.PromptHash); st != 409 || res["reason"] != "already_answered" {
		t.Fatalf("%d %v", st, res)
	}
	// Once the turn moves, the marker no longer applies.
	f.d.f.setTerminal(claudeBlocked(2))
	v2 := f.prompt(t, "t1")
	if st, _ := f.decide(t, "t1", uuid7(time.Now()), "deny", 2, v2.PromptHash); st != 200 {
		t.Fatalf("new turn refused: %d", st)
	}
	if n := len(f.keys()); n != 2 {
		t.Fatalf("%d keys", n)
	}
}

func TestDecisionSameRequestIDReturnsFirstResult(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	id := uuid7(time.Now())
	st, res := f.decide(t, "t1", id, "approve", v.Turn, v.PromptHash)
	if st != 200 {
		t.Fatalf("%d %v", st, res)
	}
	st2, res2 := f.decide(t, "t1", id, "deny", v.Turn+5, "different")
	if st2 != 200 || res2["result"] != res["result"] {
		t.Fatalf("%d %v vs %v", st2, res2, res)
	}
	// A known id returns its recorded result whatever the body, even another terminal.
	st3, res3 := f.decide(t, "t3", id, "deny", 0, "x")
	if st3 != 200 || res3["result"] != res["result"] {
		t.Fatalf("id reused on another terminal: %d %v", st3, res3)
	}
	if n := len(f.keys()); n != 1 {
		t.Fatalf("%d keys", n)
	}
}

func TestDecisionExpiredRequestID(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	for _, at := range []time.Time{time.Now().Add(-6 * time.Minute), time.Now().Add(6 * time.Minute)} {
		if st, res := f.decide(t, "t1", uuid7(at), "approve", v.Turn, v.PromptHash); st != 409 || res["reason"] != "expired_request" {
			t.Fatalf("%v: %d %v", at, st, res)
		}
	}
	if st, _ := f.decide(t, "t1", uuid7(time.Now().Add(-4*time.Minute)), "approve", v.Turn, v.PromptHash); st != 200 {
		t.Fatalf("4 minute old id refused: %d", st)
	}
	if n := len(f.keys()); n != 1 {
		t.Fatalf("%d keys", n)
	}
}

func TestRateLimitThirdActionInASecond(t *testing.T) {
	f := setupActions(t)
	f.d.actions.perSec = 2
	v := f.prompt(t, "t1")
	codes := []int{}
	for i := 0; i < 3; i++ {
		st, _ := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn+9, v.PromptHash)
		codes = append(codes, st)
	}
	if codes[0] != 409 || codes[1] != 409 || codes[2] != 429 {
		t.Fatalf("%v", codes)
	}
	st, _ := f.message(t, "t3", uuid7(time.Now()), "hello")
	if st != 429 {
		t.Fatalf("message after limit: %d", st)
	}
	time.Sleep(1100 * time.Millisecond)
	if st, _ := f.message(t, "t3", uuid7(time.Now()), "hello"); st != 200 {
		t.Fatalf("after a second: %d", st)
	}
}

func (f *actFixture) restart(t *testing.T) {
	t.Helper()
	f.d.stop()
	d, err := newDaemon(f.d.dir, f.d.f.r, "127.0.0.1:0", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d.adminPath = f.d.adminPath
	d.actions.settle = 300 * time.Millisecond
	if err := d.start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.stop)
	f.d.Daemon = d
	f.d.waitOK(t)
}

func TestRestartRebuildsFromAuditAcrossRotation(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	inflight, finished, msg := uuid7(time.Now()), uuid7(time.Now()), uuid7(time.Now())
	turn := v.Turn
	line := func(l auditLine) string {
		l.TS = time.Now().UTC().Format(time.RFC3339Nano)
		b, _ := json.Marshal(l)
		return string(b) + "\n"
	}
	f.d.stop()
	old := line(auditLine{Action: "decision", State: "dispatching", DeviceID: f.dev.ID, Terminal: "t1", Turn: &turn, RequestID: inflight, Decision: "approve", PromptHash: v.PromptHash}) +
		line(auditLine{Action: "decision", State: "dispatching", DeviceID: f.dev.ID, Terminal: "t2", Turn: &turn, RequestID: finished, Decision: "deny", PromptHash: "h2"}) +
		line(auditLine{Action: "message", State: "dispatching", DeviceID: f.dev.ID, Terminal: "t3", RequestID: msg})
	os.WriteFile(filepath.Join(f.d.dir, "audit.log.1"), []byte(old), 0o600)
	os.WriteFile(filepath.Join(f.d.dir, "audit.log"), []byte(line(auditLine{Action: "decision", State: "result", DeviceID: f.dev.ID, Terminal: "t2", RequestID: finished, Result: "applied"})), 0o600)
	d, err := newDaemon(f.d.dir, f.d.f.r, "127.0.0.1:0", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d.adminPath = f.d.adminPath
	if err := d.start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.stop)
	f.d.Daemon = d
	f.d.waitOK(t)

	if st, res := f.decide(t, "t1", inflight, "approve", v.Turn, v.PromptHash); st != 200 || res["result"] != "outcome_unknown" {
		t.Fatalf("in-flight id: %d %v", st, res)
	}
	if st, res := f.decide(t, "t2", finished, "deny", 1, "h2"); st != 200 || res["result"] != "applied" {
		t.Fatalf("finished id: %d %v", st, res)
	}
	if st, res := f.message(t, "t3", msg, "again"); st != 200 || res["result"] != "outcome_unknown" {
		t.Fatalf("message id: %d %v", st, res)
	}
	// The consumed marker came back: the same prompt cannot be answered again.
	if st, res := f.decide(t, "t1", uuid7(time.Now()), "deny", v.Turn, v.PromptHash); st != 409 || res["reason"] != "already_answered" {
		t.Fatalf("marker lost: %d %v", st, res)
	}
	if n := len(f.keys()); n != 0 {
		t.Fatalf("%d keys", n)
	}
	if n := len(f.d.f.requestsFor("terminal.send")); n != 0 {
		t.Fatalf("%d sends", n)
	}
}

func TestRestartBetweenDispatchAndResult(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	id := uuid7(time.Now())
	turn := v.Turn
	// Simulate a crash right after the dispatching line.
	if err := f.d.audit.write(auditLine{Action: "decision", State: "dispatching", DeviceID: f.dev.ID, Terminal: "t1", Turn: &turn, RequestID: id, Decision: "approve", PromptHash: v.PromptHash}); err != nil {
		t.Fatal(err)
	}
	f.restart(t)
	if st, res := f.decide(t, "t1", id, "approve", v.Turn, v.PromptHash); st != 200 || res["result"] != "outcome_unknown" {
		t.Fatalf("%d %v", st, res)
	}
	if n := len(f.keys()); n != 0 {
		t.Fatalf("%d keys", n)
	}
}

func TestAuditWriteFailureNoKey(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	f.d.audit.f.Close()
	if st, _ := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash); st != 503 {
		t.Fatalf("decision: %d", st)
	}
	if st, _ := f.message(t, "t3", uuid7(time.Now()), "hello"); st != 503 {
		t.Fatalf("message: %d", st)
	}
	if n := len(f.keys()); n != 0 {
		t.Fatalf("%d keys", n)
	}
	if n := len(f.d.f.requestsFor("terminal.send")); n != 0 {
		t.Fatalf("%d sends", n)
	}
}

func TestIoregFailureRefuses(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	f.d.f.remove("ioreg.txt")
	if st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash); st != 409 || res["reason"] != "mac_in_use" {
		t.Fatalf("%d %v", st, res)
	}
	f.d.f.write("ioreg.txt", `"HIDIdleTime" = garbage`)
	if st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash); st != 409 || res["reason"] != "mac_in_use" {
		t.Fatalf("%d %v", st, res)
	}
	if n := len(f.keys()); n != 0 {
		t.Fatalf("%d keys", n)
	}
}

func TestRetryAfterDroppedResponsePressesOneKey(t *testing.T) {
	f := setupActions(t)
	f.d.f.setKeysDelay(700 * time.Millisecond)
	v := f.prompt(t, "t1")
	id := uuid7(time.Now())
	st, res := f.post(t, "/v1/terminals/t1/decision", map[string]any{"request_id": id, "decision": "approve", "turn": v.Turn, "prompt_hash": v.PromptHash}, 250*time.Millisecond)
	if st != 0 {
		t.Fatalf("first call did not time out: %d %v", st, res)
	}
	st, res = f.decide(t, "t1", id, "approve", v.Turn, v.PromptHash)
	if st != 200 || res["result"] != "sent_unconfirmed" {
		t.Fatalf("retry: %d %v", st, res)
	}
	if n := len(f.keys()); n != 1 {
		t.Fatalf("%d keys", n)
	}
}

func TestMacInUseAndUncertain(t *testing.T) {
	f := setupActions(t)
	v := f.prompt(t, "t1")
	f.d.f.write("ioreg.txt", `"HIDIdleTime" = 5000000000`)
	st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash)
	if st != 409 || res["reason"] != "mac_in_use" || res["idle_ms"] != 5000.0 {
		t.Fatalf("%d %v", st, res)
	}
	if !strings.Contains(f.audit(t), `"result":"mac_in_use"`) || !strings.Contains(f.audit(t), `"idle_ms":5000`) {
		t.Fatalf("audit: %s", f.audit(t))
	}
	if n := len(f.keys()); n != 0 {
		t.Fatalf("%d keys", n)
	}
	// Guard off: a busy Mac does not refuse.
	zero := 0.0
	f.d.cfg.MacIdleSeconds = &zero
	if st, _ = f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash); st != 200 {
		t.Fatalf("guard off: %d", st)
	}
	f.d.cfg.MacIdleSeconds = nil
	// Input during the call: idle, then a fresh tiny idle time after the key.
	f.d.f.write("ioreg.txt", `"HIDIdleTime" = 90000000000`)
	f.d.f.setTerminal(claudeBlocked(2))
	f.d.f.setKeysDelay(400 * time.Millisecond)
	v2 := f.prompt(t, "t1")
	before := len(f.d.f.calls())
	go func() {
		for i := 0; i < 100; i++ {
			time.Sleep(20 * time.Millisecond)
			n := 0
			cs := f.d.f.calls()
			if len(cs) <= before {
				continue
			}
			for _, c := range cs[before:] {
				if c[0] == "-c" {
					n++
				}
			}
			if n >= 1 {
				f.d.f.write("ioreg.txt", `"HIDIdleTime" = 10000000`)
				return
			}
		}
	}()
	st, res = f.decide(t, "t1", uuid7(time.Now()), "approve", 2, v2.PromptHash)
	if st != 200 || res["result"] != "uncertain" {
		t.Fatalf("%d %v", st, res)
	}
	if !strings.Contains(f.audit(t), `"result":"uncertain"`) {
		t.Fatal("uncertain not audited")
	}
}

func TestMessageCleansAndSendsOnce(t *testing.T) {
	f := setupActions(t)
	id := uuid7(time.Now())
	raw := "hi\x1b[31m there\u0085\n\ttab\x00end\u009fé"
	st, res := f.message(t, "t3", id, raw)
	if st != 200 || res["result"] != "sent" {
		t.Fatalf("%d %v", st, res)
	}
	if st, res = f.message(t, "t3", id, "something else"); st != 200 || res["result"] != "sent" {
		t.Fatalf("retry: %d %v", st, res)
	}
	sends := f.d.f.requestsFor("terminal.send")
	if len(sends) != 1 || sends[0].Params["text"] != "hi[31m there\n\ttabendé" || sends[0].Params["force"] != false {
		t.Fatalf("sends %v", sends)
	}
	if _, has := sends[0].Params["caller"]; has {
		t.Fatal("caller sent")
	}
	a := f.audit(t)
	if strings.Contains(a, "there") || strings.Contains(a, "tabend") || !strings.Contains(a, `"length":21`) || !strings.Contains(a, `"result":"sent"`) {
		t.Fatalf("audit: %s", a)
	}
	if st, _ = f.message(t, "t3", uuid7(time.Now()), "\x01\x02"); st != 400 {
		t.Fatalf("control-only text: %d", st)
	}
	if st, _ = f.message(t, "t3", uuid7(time.Now()), strings.Repeat("é", 4001)); st != 400 {
		t.Fatalf("too long: %d", st)
	}
	time.Sleep(1100 * time.Millisecond)
	if st, _ = f.message(t, "t3", uuid7(time.Now()), strings.Repeat("é", 4000)); st != 200 {
		t.Fatalf("4000 chars: %d", st)
	}
}

func TestMessageRefusedTargets(t *testing.T) {
	f := setupActions(t)
	if st, res := f.message(t, "t1", uuid7(time.Now()), "hello"); st != 409 || res["reason"] != "blocked" {
		t.Fatalf("blocked: %d %v", st, res)
	}
	if st, _ := f.message(t, "t99", uuid7(time.Now()), "hello"); st != 404 {
		t.Fatalf("missing: %d", st)
	}
	if st, _ := f.message(t, "t8", uuid7(time.Now()), "hello"); st != 409 {
		t.Fatalf("unreported: %d", st)
	}
	if st, res := f.message(t, "t3", uuid7(time.Now().Add(-10*time.Minute)), "hello"); st != 409 || res["reason"] != "expired_request" {
		t.Fatalf("expired: %d %v", st, res)
	}
	if n := len(f.d.f.requestsFor("terminal.send")); n != 0 {
		t.Fatalf("%d sends", n)
	}
	if !strings.Contains(f.audit(t), `"result":"blocked"`) {
		t.Fatal("blocked refusal not audited")
	}
}

// hc holds a key while someone types at the Mac and presses it when the Mac is
// quiet. The daemon waits long enough for that, so the idle check after the key runs.
func TestQueuedKeyGetsTheIdleCheck(t *testing.T) {
	f := setupActions(t)
	if f.d.actions.keyWait != 120*time.Second {
		t.Fatalf("key wait %v", f.d.actions.keyWait)
	}
	ioregs := func() int {
		n := 0
		for _, c := range f.d.f.calls() {
			if c[0] == "-c" {
				n++
			}
		}
		return n
	}
	f.d.f.setKeysDelay(500 * time.Millisecond)
	f.d.f.onKeys("t1", `{"status":{"app":"claude-code","state":"done"},"turn":2}`)
	// A wait shorter than hc's delivery: outcome_unknown, and no idle check after the key.
	f.d.actions.keyWait = 100 * time.Millisecond
	v := f.prompt(t, "t1")
	before := ioregs()
	if st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash); st != 200 || res["result"] != "outcome_unknown" {
		t.Fatalf("short wait: %d %v", st, res)
	}
	if n := ioregs() - before; n != 1 {
		t.Fatalf("%d idle reads with a short wait", n)
	}
	// The full wait covers the queued key: the idle check runs and the result is applied.
	time.Sleep(600 * time.Millisecond)
	f.d.f.setTerminal(claudeBlocked(3))
	f.d.f.onKeys("t1", `{"status":{"app":"claude-code","state":"done"},"turn":4}`)
	f.d.actions.keyWait = keyWaitTime
	v = f.prompt(t, "t1")
	before = ioregs()
	if st, res := f.decide(t, "t1", uuid7(time.Now()), "approve", v.Turn, v.PromptHash); st != 200 || res["result"] != "applied" {
		t.Fatalf("full wait: %d %v", st, res)
	}
	if n := ioregs() - before; n != 2 {
		t.Fatalf("%d idle reads with the full wait", n)
	}
	if n := len(f.keys()); n != 2 {
		t.Fatalf("%d keys", n)
	}
}
