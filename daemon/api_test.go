package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func (d *testDaemon) get(t *testing.T, cert *tls.Certificate, path string) (int, []byte) {
	t.Helper()
	resp, err := d.client(cert, nil).Get(d.pairedURL(path))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func (d *testDaemon) waitOK(t *testing.T) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		if d.state.isOK() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("state never became ok")
}

func (d *testDaemon) waitNotOK(t *testing.T) {
	t.Helper()
	for i := 0; i < 500; i++ {
		if !d.state.isOK() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("state stayed ok")
}

type sseMsg struct {
	event   string
	data    string
	comment bool
}

type sseClient struct {
	resp *http.Response
	msgs chan sseMsg
	done chan error
}

func (d *testDaemon) sse(t *testing.T, cert *tls.Certificate) *sseClient {
	t.Helper()
	resp, err := d.client(cert, nil).Get(d.pairedURL("/v1/events"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		t.Fatalf("events status %d", resp.StatusCode)
	}
	c := &sseClient{resp: resp, msgs: make(chan sseMsg, 64), done: make(chan error, 1)}
	go func() {
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		var cur sseMsg
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if cur.event != "" || cur.data != "" {
					c.msgs <- cur
				}
				cur = sseMsg{}
			case strings.HasPrefix(line, ": "):
				c.msgs <- sseMsg{comment: true, data: line}
			case strings.HasPrefix(line, "event: "):
				cur.event = line[7:]
			case strings.HasPrefix(line, "data: "):
				cur.data = line[6:]
			}
		}
		c.done <- sc.Err()
	}()
	t.Cleanup(func() { resp.Body.Close() })
	return c
}

func (c *sseClient) next(t *testing.T, timeout time.Duration) sseMsg {
	t.Helper()
	select {
	case m := <-c.msgs:
		return m
	case err := <-c.done:
		t.Fatalf("stream ended: %v", err)
	case <-time.After(timeout):
		t.Fatal("no SSE message")
	}
	return sseMsg{}
}

func (c *sseClient) nextEvent(t *testing.T, timeout time.Duration) sseMsg {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		m := c.next(t, time.Until(deadline))
		if !m.comment {
			return m
		}
	}
}

func (c *sseClient) ended(timeout time.Duration) bool {
	for {
		select {
		case <-c.msgs:
		case <-c.done:
			return true
		case <-time.After(timeout):
			return false
		}
	}
}

const herdrLs = `{"spaces":[
 {"id":"sp1","name":"Niti","machine":null,"active":true,"tabs":[{"id":"tb1","name":"main","active":true,"terminals":[{"terminal":"t1","pane":"p1","status":"blocked","app":"claude-code","name":null,"sub_agents":["t3"]}]}]},
 {"id":"sp2","name":"Remote","machine":"vps","active":false,"tabs":[{"id":"tb2","name":"x","active":true,"terminals":[{"terminal":"t9","pane":"p2","status":"idle","app":"codex","name":null,"sub_agents":[]}]}]}
],"recent":[{"name":"Tools","folder":"/x"}]}`

func seedAgents(f *fake) {
	t1 := liveAgent("t1", "claude-code", "blocked", 2)
	t1.Status.Kind, t1.Status.Msg = "permission", "approve Bash: touch x"
	t1.Title = "touching"
	t1.Labels = map[string]string{"herdr.space": "sp1", "herdr.tab": "tb1", "where": "Niti / main"}
	f.setTerminal(t1)
	t2 := liveAgent("t2", "", "idle", 0)
	t2.Reported, t2.Status = false, nil
	f.setTerminal(t2)
	t3 := liveAgent("t3", "codex", "idle", 5)
	t3.Labels = map[string]string{"where": "Email-Agent / codex"}
	f.setTerminal(t3)
	t7 := liveAgent("t7", "pi", "working", 1)
	t7.Labels = map[string]string{"where": "Zeta / 1"}
	f.setTerminal(t7)
	remote := liveAgent("box/t4", "claude-code", "idle", 1)
	f.setTerminal(remote)
	f.write("herdr-app-ls.json", herdrLs)
}

func TestHelloAgentsAndTerminal(t *testing.T) {
	d := startTestDaemon(t)
	seedAgents(d.f)
	cert, _ := d.pairPhone(t, "p")
	d.waitOK(t)

	st, b := d.get(t, &cert, "/v1/hello")
	var hello struct {
		Version   string   `json:"daemon_version"`
		Time      int64    `json:"server_time_ms"`
		Endpoints []string `json:"endpoints"`
	}
	json.Unmarshal(b, &hello)
	if st != 200 || hello.Version != version || hello.Time < time.Now().UnixMilli()-5000 {
		t.Fatalf("%d %s", st, b)
	}

	st, b = d.get(t, &cert, "/v1/agents")
	var agents []agentJSON
	json.Unmarshal(b, &agents)
	if st != 200 || len(agents) != 3 || agents[0].Address != "t1" || agents[1].Address != "t3" || agents[2].Address != "t7" {
		t.Fatalf("%d %s", st, b)
	}
	a := agents[0]
	if a.Kind != "permission" || a.Msg != "approve Bash: touch x" || a.State != "blocked" || a.App != "claude-code" || a.Turn != 2 || a.Title != "touching" || a.Space != "Niti" || a.Tab != "main" || a.Where != "Niti / main" {
		t.Fatalf("%+v", a)
	}
	if agents[1].Kind != "" || agents[1].Msg != "" || agents[1].Space != "Niti" || agents[1].Tab != "main" {
		t.Fatalf("sub-agent: %+v", agents[1])
	}
	if agents[2].Space != "" || agents[2].Tab != "" {
		t.Fatalf("hidden agent placed: %+v", agents[2])
	}
	if strings.Contains(string(b), `"kind":""`) {
		t.Fatalf("empty kind serialised: %s", b)
	}

	st, b = d.get(t, &cert, "/v1/terminals/t1")
	var m map[string]any
	json.Unmarshal(b, &m)
	if st != 200 || m["cols"] != 200.0 || m["rows"] != 120.0 || m["address"] != "t1" {
		t.Fatalf("%d %s", st, b)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/t2"); st != 409 {
		t.Fatalf("shell terminal: %d", st)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/t9"); st != 404 {
		t.Fatalf("missing terminal: %d", st)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/box%2Ft4"); st != 400 {
		t.Fatalf("remote terminal: %d", st)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/review"); st != 400 {
		t.Fatalf("named terminal: %d", st)
	}
}

func TestSpacesFollowHerdrApp(t *testing.T) {
	d := startTestDaemon(t)
	seedAgents(d.f)
	cert, _ := d.pairPhone(t, "p")
	d.waitOK(t)
	var spaces []spaceJSON
	st, b := d.get(t, &cert, "/v1/spaces")
	json.Unmarshal(b, &spaces)
	if st != 200 || len(spaces) != 2 || spaces[0].Name != "Niti" || len(spaces[0].Agents) != 2 || spaces[0].Agents[0].Address != "t1" || spaces[0].Agents[1].Address != "t3" {
		t.Fatalf("%d %s", st, b)
	}
	if spaces[1].ID != "other" || len(spaces[1].Agents) != 1 || spaces[1].Agents[0].Address != "t7" {
		t.Fatalf("remote space not dropped or hidden agent lost: %s", b)
	}
	// A rename makes no hc event, but the next request sees it.
	d.f.write("herdr-app-ls.json", strings.Replace(herdrLs, `"name":"Niti"`, `"name":"Niti renamed"`, 1))
	_, b = d.get(t, &cert, "/v1/spaces")
	json.Unmarshal(b, &spaces)
	if spaces[0].Name != "Niti renamed" {
		t.Fatalf("%s", b)
	}
	// Herdr.app closed: grouped under one space, placement from the where label.
	d.f.remove("herdr-app-ls.json")
	_, b = d.get(t, &cert, "/v1/spaces")
	json.Unmarshal(b, &spaces)
	if len(spaces) != 1 || spaces[0].Name != "Herdr window closed" || len(spaces[0].Agents) != 3 {
		t.Fatalf("%s", b)
	}
	if a := spaces[0].Agents[0]; a.Address != "t3" || a.Space != "Email-Agent" || a.Tab != "codex" {
		t.Fatalf("%+v", a)
	}
	if a := spaces[0].Agents[2]; a.Address != "t7" || a.Space != "Zeta" || a.Tab != "1" {
		t.Fatalf("%+v", a)
	}
}

func TestEventsSSE(t *testing.T) {
	d := startTestDaemon(t)
	seedAgents(d.f)
	cert, _ := d.pairPhone(t, "p")
	d.waitOK(t)
	c := d.sse(t, &cert)
	m := c.nextEvent(t, 5*time.Second)
	var agents []agentJSON
	json.Unmarshal([]byte(m.data), &agents)
	if m.event != "snapshot" || len(agents) != 3 {
		t.Fatalf("%+v", m)
	}

	d.f.emitEvent("status", "t3", 101, `{"records":{},"reported":true,"status":{"app":"codex","state":"blocked","kind":"permission"}}`)
	m = c.nextEvent(t, 5*time.Second)
	var a agentJSON
	json.Unmarshal([]byte(m.data), &a)
	if m.event != "agent" || a.Address != "t3" || a.State != "blocked" || a.Kind != "permission" || a.Turn != 5 {
		t.Fatalf("%+v %+v", m, a)
	}

	d.f.emitEvent("turn", "t3", 102, `{"state":"done","turn":6}`)
	m = c.nextEvent(t, 5*time.Second)
	json.Unmarshal([]byte(m.data), &a)
	if m.event != "agent" || a.Turn != 6 {
		t.Fatalf("%+v", m)
	}

	d.f.emitEvent("finished", "t3", 103, `{"exit_code":0,"finished_ms":1,"reason":"exited","signal":null}`)
	m = c.nextEvent(t, 5*time.Second)
	if m.event != "agent_removed" || !strings.Contains(m.data, `"address":"t3"`) {
		t.Fatalf("%+v", m)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/t3"); st != 409 {
		t.Fatalf("finished terminal still an agent: %d", st)
	}

	// A new unreported terminal makes no event; its first status report does.
	t6 := liveAgent("t6", "", "idle", 0)
	t6.Reported, t6.Status = false, nil
	d.f.setTerminal(t6)
	b, _ := json.Marshal(t6)
	d.f.emitEvent("created", "t6", 104, string(b))
	d.f.emitEvent("title", "t6", 105, `{"title":"shell"}`)
	d.f.emitEvent("status", "t6", 106, `{"records":{},"reported":true,"status":{"app":"pi","state":"working"}}`)
	m = c.nextEvent(t, 5*time.Second)
	json.Unmarshal([]byte(m.data), &a)
	if m.event != "agent" || a.Address != "t6" || a.App != "pi" || a.Title != "shell" {
		t.Fatalf("%+v", m)
	}

	// missed, then a fresh snapshot.
	d.f.emit(`{"jsonrpc":"2.0","method":"missed","params":{"seq":106}}`)
	snap := `{"jsonrpc":"2.0","method":"snapshot","params":{"seq":200,"terminals":[` + string(b) + `]}}`
	d.f.emit(snap)
	m = c.nextEvent(t, 5*time.Second)
	json.Unmarshal([]byte(m.data), &agents)
	if m.event != "snapshot" || len(agents) != 0 {
		t.Fatalf("%+v", m)
	}
}

func TestLayoutParsesRealShape(t *testing.T) {
	d := startTestDaemon(t)
	d.f.write("herdr-app-ls.json", herdrLs)
	lay := d.state.refreshLayout(testCtx(t))
	if !lay.open || len(lay.spaces) != 1 || lay.place["t3"].tabName != "main" || lay.place["t1"].spaceName != "Niti" {
		t.Fatalf("%+v", lay)
	}
	if _, placed := lay.place["t9"]; placed {
		t.Fatal("remote terminal placed")
	}
}

func TestEventsKeepaliveWithin11s(t *testing.T) {
	d := startTestDaemon(t)
	seedAgents(d.f)
	cert, _ := d.pairPhone(t, "p")
	d.waitOK(t)
	c := d.sse(t, &cert)
	c.nextEvent(t, 5*time.Second)
	start := time.Now()
	m := c.next(t, 11*time.Second)
	if !m.comment || m.data != ": ka" {
		t.Fatalf("%+v", m)
	}
	if time.Since(start) > 11*time.Second {
		t.Fatal("keepalive too late")
	}
}

func TestHCLostGives503ThenRecovers(t *testing.T) {
	d := startTestDaemon(t)
	seedAgents(d.f)
	cert, _ := d.pairPhone(t, "p")
	d.waitOK(t)
	c := d.sse(t, &cert)
	c.nextEvent(t, 5*time.Second)
	d.state.mu.RLock()
	conn := d.state.conn
	d.state.mu.RUnlock()
	hcPath := d.r.progs.HC
	d.r.progs.HC = "/nonexistent/hc"
	d.hc.Close()
	conn.cmd.Process.Kill()
	d.waitNotOK(t)
	if !c.ended(3 * time.Second) {
		t.Fatal("SSE survived hc loss")
	}
	if st, _ := d.get(t, &cert, "/v1/agents"); st != 503 {
		t.Fatalf("agents while hc lost: %d", st)
	}
	if st, _ := d.get(t, &cert, "/v1/spaces"); st != 503 {
		t.Fatalf("spaces while hc lost: %d", st)
	}
	if st, _ := d.get(t, &cert, "/v1/terminals/t1"); st != 503 {
		t.Fatalf("terminal while hc lost: %d", st)
	}
	resp, err := d.client(&cert, nil).Get(d.pairedURL("/v1/events"))
	if err != nil || resp.StatusCode != 503 {
		t.Fatalf("events while hc lost: %v %v", err, resp)
	}
	resp.Body.Close()
	d.r.progs.HC = hcPath
	d.waitOK(t)
	if st, _ := d.get(t, &cert, "/v1/agents"); st != 200 {
		t.Fatalf("agents after reconnect: %d", st)
	}
	c = d.sse(t, &cert)
	if m := c.nextEvent(t, 5*time.Second); m.event != "snapshot" {
		t.Fatalf("%+v", m)
	}
}

func TestRevokeEndsStreams(t *testing.T) {
	d := startTestDaemon(t)
	seedAgents(d.f)
	cert, dev := d.pairPhone(t, "p")
	d.waitOK(t)
	c := d.sse(t, &cert)
	c.nextEvent(t, 5*time.Second)
	if err := d.revoke(dev.ID); err != nil {
		t.Fatal(err)
	}
	if !c.ended(2 * time.Second) {
		t.Fatal("SSE survived revoke")
	}
	if _, err := d.client(&cert, nil).Get(d.pairedURL("/v1/agents")); err == nil {
		t.Fatal("revoked device reconnected")
	}
}
