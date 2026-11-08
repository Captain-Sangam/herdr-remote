package main

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCallRefusesMethodOutsideList(t *testing.T) {
	f := newFake(t)
	h := newHC(f.r)
	defer h.Close()
	ctx := testCtx(t)
	for _, m := range []string{"terminal.run", "terminal.kill", "terminal.rm", "machine.exec", "file.read", "file.write", "server.stop", "terminal.attach", "terminal.interrupt", "doc.set"} {
		err := h.call(ctx, "reads", m, struct{}{}, nil)
		if err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Fatalf("%s: %v", m, err)
		}
	}
	for _, r := range f.requests() {
		if !allowedMethods[r.Method] {
			t.Fatalf("sent %s", r.Method)
		}
	}
}

func TestSendForceFalseNoCaller(t *testing.T) {
	f := newFake(t)
	f.setTerminal(liveAgent("t7", "claude-code", "idle", 3))
	h := newHC(f.r)
	defer h.Close()
	res, err := h.Send(testCtx(t), "t7", "hello there")
	if err != nil || res.Turn != 3 {
		t.Fatalf("%v %v", res, err)
	}
	sends := f.requestsFor("terminal.send")
	if len(sends) != 1 {
		t.Fatalf("sends %v", sends)
	}
	p := sends[0].Params
	if p["force"] != false || p["text"] != "hello there" || p["terminal"] != "t7" {
		t.Fatalf("params %v", p)
	}
	if _, has := p["caller"]; has {
		t.Fatal("caller was sent")
	}
	if _, has := p["if"]; has {
		t.Fatal("if was sent")
	}
}

func TestSendRefusesUnreportedAndFinished(t *testing.T) {
	f := newFake(t)
	shell := liveAgent("t8", "", "idle", 0)
	shell.Reported = false
	shell.Status = nil
	f.setTerminal(shell)
	done := liveAgent("t9", "claude-code", "done", 1)
	done.State = "finished"
	f.setTerminal(done)
	blocked := liveAgent("t10", "claude-code", "blocked", 1)
	f.setTerminal(blocked)
	h := newHC(f.r)
	defer h.Close()
	ctx := testCtx(t)
	if _, err := h.Send(ctx, "t8", "x"); !errors.Is(err, errNotAgent) {
		t.Fatalf("unreported: %v", err)
	}
	if _, err := h.Send(ctx, "t9", "x"); !errors.Is(err, errNotAgent) {
		t.Fatalf("finished: %v", err)
	}
	if _, err := h.Send(ctx, "t10", "x"); !errors.Is(err, errBlocked) {
		t.Fatalf("blocked: %v", err)
	}
	if _, err := h.Send(ctx, "t999", "x"); !errors.Is(err, errNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if n := len(f.requestsFor("terminal.send")); n != 1 {
		t.Fatalf("%d sends, want only the blocked attempt", n)
	}
}

func TestKeysOnlyThreeKeys(t *testing.T) {
	f := newFake(t)
	f.setTerminal(liveAgent("t7", "claude-code", "blocked", 1))
	h := newHC(f.r)
	defer h.Close()
	ctx := testCtx(t)
	for _, k := range []Key{"enter", "2", "4", "ctrl+c", "", "1 "} {
		if _, err := h.Keys(ctx, "t7", k); err == nil {
			t.Fatalf("key %q accepted", k)
		}
	}
	if len(f.requestsFor("terminal.keys")) != 0 {
		t.Fatal("a refused key reached hc")
	}
	for _, k := range []Key{KeyOne, KeyY, KeyEsc} {
		if _, err := h.Keys(ctx, "t7", k); err != nil {
			t.Fatal(err)
		}
	}
	reqs := f.requestsFor("terminal.keys")
	if len(reqs) != 3 {
		t.Fatalf("%d keys requests", len(reqs))
	}
	for i, want := range []string{"1", "y", "esc"} {
		keys, _ := reqs[i].Params["keys"].([]any)
		if len(keys) != 1 || keys[0] != want {
			t.Fatalf("request %d keys %v", i, reqs[i].Params["keys"])
		}
		if _, has := reqs[i].Params["caller"]; has {
			t.Fatal("caller sent")
		}
	}
}

func TestMachineAndNameTargetsRefused(t *testing.T) {
	f := newFake(t)
	h := newHC(f.r)
	defer h.Close()
	ctx := testCtx(t)
	for _, bad := range []string{"box/t7", "review", "T7", "t7/x", "", "t-1", "machine/t1"} {
		if _, err := h.Get(ctx, bad); !errors.Is(err, errBadTarget) {
			t.Fatalf("Get %q: %v", bad, err)
		}
		if _, err := h.Read(ctx, bad, 0); !errors.Is(err, errBadTarget) {
			t.Fatalf("Read %q: %v", bad, err)
		}
		if _, err := h.Send(ctx, bad, "x"); !errors.Is(err, errBadTarget) {
			t.Fatalf("Send %q: %v", bad, err)
		}
		if _, err := h.Keys(ctx, bad, KeyEsc); !errors.Is(err, errBadTarget) {
			t.Fatalf("Keys %q: %v", bad, err)
		}
	}
	if n := len(f.requests()); n != 0 {
		t.Fatalf("%d requests reached hc", n)
	}
}

func TestFixtureParses(t *testing.T) {
	b, err := os.ReadFile("testdata/events-probe.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	lines := splitLines(b)
	first, err := parseNote(lines[0])
	if err != nil || first.Method != "snapshot" || len(first.Snapshot.Terminals) != 2 {
		t.Fatalf("snapshot: %+v %v", first, err)
	}
	if first.Snapshot.Terminals[0].Address != "t2c" || first.Snapshot.Terminals[0].Reported {
		t.Fatalf("snapshot terminal: %+v", first.Snapshot.Terminals[0])
	}
	counts := map[string]int{}
	var sawPermission, sawSize, sawFinished, sawRemoved bool
	for _, l := range lines[1:] {
		n, err := parseNote(l)
		if err != nil {
			t.Fatalf("%s: %v", l, err)
		}
		key := n.Method
		if n.Method == "event" {
			key = n.Type
		}
		counts[key]++
		switch n.Type {
		case "status":
			var d statusData
			if err := unmarshalData(n.Data, &d); err != nil {
				t.Fatal(err)
			}
			if d.Status != nil && d.Status.Kind == "permission" && d.Status.State == "blocked" && n.Terminal == "t2c" {
				sawPermission = true
			}
		case "size":
			var d sizeData
			if err := unmarshalData(n.Data, &d); err != nil || d.Cols == 0 {
				t.Fatalf("size %s", n.Data)
			}
			sawSize = true
		case "created":
			var term Terminal
			if err := unmarshalData(n.Data, &term); err != nil || term.Address != n.Terminal {
				t.Fatalf("created %s", n.Data)
			}
		case "turn":
			var d turnData
			if err := unmarshalData(n.Data, &d); err != nil || d.Turn == 0 {
				t.Fatalf("turn %s", n.Data)
			}
		case "finished":
			sawFinished = true
		case "removed":
			sawRemoved = true
		}
	}
	for _, k := range []string{"created", "status", "turn", "title", "labels", "size", "foreground", "heartbeat"} {
		if counts[k] == 0 {
			t.Fatalf("no %s events in fixture: %v", k, counts)
		}
	}
	if !sawPermission || !sawSize || !sawFinished || !sawRemoved {
		t.Fatalf("missing shapes: permission=%v size=%v finished=%v removed=%v", sawPermission, sawSize, sawFinished, sawRemoved)
	}
}

func TestSubscribeStreamsNotes(t *testing.T) {
	f := newFake(t)
	f.setTerminal(liveAgent("t2c", "claude-code", "working", 0))
	h := newHC(f.r)
	defer h.Close()
	ctx := testCtx(t)
	snap, conn, err := h.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.close()
	if snap.Seq != 100 || len(snap.Terminals) != 1 {
		t.Fatalf("snapshot %+v", snap)
	}
	b, _ := os.ReadFile("testdata/events-probe.jsonl")
	lines := splitLines(b)
	for _, l := range lines[1:] {
		f.emit(string(l))
	}
	got := 0
	var status int
	timeout := time.After(10 * time.Second)
	for got < len(lines)-1 {
		select {
		case n := <-conn.notes:
			got++
			if n.Type == "status" {
				status++
			}
		case <-conn.done:
			t.Fatal("connection died")
		case <-timeout:
			t.Fatalf("got %d of %d notes", got, len(lines)-1)
		}
	}
	if status == 0 {
		t.Fatal("no status notes")
	}
}

func TestReconnectsAfterHCDies(t *testing.T) {
	f := newFake(t)
	f.setTerminal(liveAgent("t7", "claude-code", "idle", 1))
	h := newHC(f.r)
	defer h.Close()
	ctx := testCtx(t)
	if _, err := h.Get(ctx, "t7"); err != nil {
		t.Fatal(err)
	}
	h.conns["reads"].cmd.Process.Kill()
	<-h.conns["reads"].done
	if _, err := h.Get(ctx, "t7"); err != nil {
		t.Fatalf("after reconnect: %v", err)
	}
	// A dial failure backs off instead of hammering.
	h.Close()
	h.r.progs.HC = "/nonexistent/hc"
	if _, err := h.Get(ctx, "t7"); !errors.Is(err, errUnavailable) {
		t.Fatalf("want unavailable, got %v", err)
	}
	if h.delay["reads"] != 2*time.Second || time.Until(h.nextDial["reads"]) <= 0 {
		t.Fatalf("no backoff: %v %v", h.delay, h.nextDial)
	}
}
