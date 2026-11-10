package main

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

func (d *testDaemon) waitLive(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if d.live.count() == n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("live count %d, want %d", d.live.count(), n)
}

func readsEOF(rd io.Reader, within time.Duration) bool {
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := rd.Read(buf); err != nil {
				close(done)
				return
			}
		}
	}()
	select {
	case <-done:
		return true
	case <-time.After(within):
		return false
	}
}

func TestLiveStreamEndsOnDisconnectSizeAndRevoke(t *testing.T) {
	d := startTestDaemon(t, seedAgents)
	cert, dev := d.pairPhone(t, "p")
	d.waitOK(t)
	open := func() (*http.Response, *bufio.Reader) {
		t.Helper()
		resp, err := d.client(&cert, nil).Get(d.pairedURL("/v1/terminals/t1/live"))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("status %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		rd := bufio.NewReader(resp.Body)
		first, err := rd.ReadString('\n')
		if err != nil || !strings.Contains(first, "PAINT t1") || !strings.Contains(first, "\x1b[31m") {
			t.Fatalf("first paint %q %v", first, err)
		}
		return resp, rd
	}

	resp, _ := open()
	d.waitLive(t, 1)
	pids := d.live.pids()
	resp.Body.Close()
	d.waitLive(t, 0)
	for _, pid := range pids {
		if err := syscall.Kill(pid, 0); err == nil {
			t.Fatalf("attach process %d still alive", pid)
		}
	}

	resp, rd := open()
	d.waitLive(t, 1)
	d.f.emitEvent("size", "t1", 300, `{"cols":100,"rows":40}`)
	if !readsEOF(rd, 2*time.Second) {
		t.Fatal("stream survived a size event")
	}
	resp.Body.Close()
	d.waitLive(t, 0)
	if term := d.state.terminal("t1"); term.Cols != 100 || term.Rows != 40 {
		t.Fatalf("size not applied: %d x %d", term.Cols, term.Rows)
	}

	resp, rd = open()
	d.waitLive(t, 1)
	if err := d.revoke(dev.ID); err != nil {
		t.Fatal(err)
	}
	if !readsEOF(rd, 2*time.Second) {
		t.Fatal("stream survived a revoke")
	}
	resp.Body.Close()
	d.waitLive(t, 0)
}

func TestLiveLimitAndTargets(t *testing.T) {
	d := startTestDaemon(t, seedAgents)
	cert, _ := d.pairPhone(t, "p")
	d.waitOK(t)
	var open []*http.Response
	for i := 0; i < 2; i++ {
		resp, err := d.client(&cert, nil).Get(d.pairedURL("/v1/terminals/t1/live"))
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("stream %d: %v %v", i, err, resp)
		}
		open = append(open, resp)
	}
	d.waitLive(t, 2)
	st, b := d.get(t, &cert, "/v1/terminals/t3/live")
	if st != 409 || !strings.Contains(string(b), "live_limit") {
		t.Fatalf("third stream: %d %s", st, b)
	}
	open[0].Body.Close()
	d.waitLive(t, 1)
	resp, err := d.client(&cert, nil).Get(d.pairedURL("/v1/terminals/t3/live"))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("after close: %v %v", err, resp)
	}
	resp.Body.Close()
	open[1].Body.Close()
	d.waitLive(t, 0)

	if st, b := d.get(t, &cert, "/v1/terminals/t2/live"); st != 409 || !strings.Contains(string(b), "not_agent") {
		t.Fatalf("shell: %d %s", st, b)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/box%2Ft4/live"); st != 400 {
		t.Fatalf("remote: %d", st)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/t99/live"); st != 404 {
		t.Fatalf("missing: %d", st)
	}
	if calls := d.f.calls(); len(calls) > 0 {
		for _, c := range calls {
			if c[0] == "attach" && (len(c) != 5 || c[1] != "--observe" || c[2] != "--history" || c[3] != "500") {
				t.Fatalf("attach argv %v", c)
			}
		}
	}
}
