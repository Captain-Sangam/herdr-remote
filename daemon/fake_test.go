package main

// The test binary doubles as every program the daemon runs when HR_FAKE_DIR is
// set. Files in that directory:
//   state.json        terminals, screens and scripted reactions (reloaded per request)
//   requests.jsonl    every JSON-RPC request `hc connect` received
//   calls.jsonl       the argv of every other program run
//   notes.jsonl       lines the events connection streams after events.subscribe
//   ioreg.txt, herdr-app-ls.json, ps.txt, lsof-<pid>.txt, tailscale.json

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if dir := os.Getenv("HR_FAKE_DIR"); dir != "" {
		os.Exit(fakeMain(dir, os.Args[1:]))
	}
	os.Exit(m.Run())
}

type fakeState struct {
	Seq         int64                      `json:"seq"`
	Terminals   map[string]json.RawMessage `json:"terminals"`
	Reads       map[string]string          `json:"reads"`
	OnKeys      map[string]json.RawMessage `json:"on_keys"` // merged into the terminal after terminal.keys
	OnSend      map[string]json.RawMessage `json:"on_send"` // merged into the terminal after terminal.send
	KeysDelayMS int                        `json:"keys_delay_ms"`
}

func fakeMain(dir string, args []string) int {
	if len(args) == 0 {
		return 2
	}
	b, _ := json.Marshal(args)
	appendLine(dir, "calls.jsonl", b)
	switch args[0] {
	case "connect":
		return fakeConnect(dir)
	case "attach":
		return fakeAttach(dir, args[len(args)-1])
	case "ls":
		return fakeCat(dir, "herdr-app-ls.json")
	case "-c":
		return fakeCat(dir, "ioreg.txt")
	case "-Fn":
		if len(args) < 3 {
			return 2
		}
		return fakeCat(dir, "lsof-"+args[2]+".txt")
	case "-axo":
		return fakeCat(dir, "ps.txt")
	case "status":
		return fakeCat(dir, "tailscale.json")
	}
	return 2
}

func appendLine(dir, name string, line []byte) {
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

func fakeCat(dir, name string) int {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintln(os.Stderr, "fake: no", name)
		return 1
	}
	os.Stdout.Write(b)
	return 0
}

func loadFakeState(dir string) fakeState {
	var st fakeState
	b, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	json.Unmarshal(b, &st)
	if st.Terminals == nil {
		st.Terminals = map[string]json.RawMessage{}
	}
	return st
}

func saveFakeState(dir string, st fakeState) {
	b, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(dir, "state.json"), b, 0o600)
}

func jsonPatch(base, p json.RawMessage) json.RawMessage {
	var m, q map[string]any
	json.Unmarshal(base, &m)
	json.Unmarshal(p, &q)
	if m == nil {
		m = map[string]any{}
	}
	for k, v := range q {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return b
}

func rpcErr(kind, msg string) *rpcError {
	e := &rpcError{Code: -32000, Message: msg}
	e.Data.Kind = kind
	return e
}

func fakeConnect(dir string) int {
	in := bufio.NewReaderSize(os.Stdin, 1<<20)
	var wmu sync.Mutex
	write := func(v any) {
		b, _ := json.Marshal(v)
		wmu.Lock()
		os.Stdout.Write(append(b, '\n'))
		wmu.Unlock()
	}
	subscribed := false
	for {
		line, err := in.ReadBytes('\n')
		if len(line) == 0 && err != nil {
			return 0
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				Terminal string   `json:"terminal"`
				Keys     []string `json:"keys"`
				Text     string   `json:"text"`
			} `json:"params"`
		}
		if json.Unmarshal(line, &req) != nil {
			continue
		}
		appendLine(dir, "requests.jsonl", line)
		st := loadFakeState(dir)
		var result any
		var e *rpcError
		term, has := st.Terminals[req.Params.Terminal]
		switch req.Method {
		case "hello":
			result = map[string]any{"generation": 1, "version": "fake", "framed": false}
		case "terminal.list":
			var ts []json.RawMessage
			for _, t := range st.Terminals {
				ts = append(ts, t)
			}
			result = map[string]any{"terminals": ts, "hidden_finished": 0, "machines": []any{}}
		case "terminal.get":
			if !has {
				e = rpcErr("not_found", "no terminal")
			} else {
				result = term
			}
		case "terminal.read":
			if !has {
				e = rpcErr("not_found", "no terminal")
			} else {
				result = map[string]any{"text": st.Reads[req.Params.Terminal], "complete": true, "how": "screen"}
			}
		case "terminal.send":
			if !has {
				e = rpcErr("not_found", "no terminal")
				break
			}
			var t Terminal
			json.Unmarshal(term, &t)
			if t.Status != nil && t.Status.State == "blocked" {
				e = rpcErr("blocked", "blocked")
				break
			}
			if p, ok := st.OnSend[req.Params.Terminal]; ok {
				st.Terminals[req.Params.Terminal] = jsonPatch(term, p)
				saveFakeState(dir, st)
			}
			started := true
			result = map[string]any{"turn": t.Turn, "status": statusWord(t), "started": started}
		case "terminal.keys":
			if !has {
				e = rpcErr("not_found", "no terminal")
				break
			}
			if len(req.Params.Keys) != 1 {
				e = rpcErr("invalid_params", "keys")
				break
			}
			var t Terminal
			json.Unmarshal(term, &t)
			if st.KeysDelayMS > 0 {
				time.Sleep(time.Duration(st.KeysDelayMS) * time.Millisecond)
			}
			if p, ok := st.OnKeys[req.Params.Terminal]; ok {
				st.Terminals[req.Params.Terminal] = jsonPatch(term, p)
				saveFakeState(dir, st)
			}
			result = map[string]any{"turn": t.Turn, "status": statusWord(t)}
		case "events.subscribe":
			var ts []json.RawMessage
			for _, t := range st.Terminals {
				ts = append(ts, t)
			}
			result = map[string]any{"seq": st.Seq, "terminals": ts, "channel": nil}
			subscribed = true
		default:
			e = &rpcError{Code: -32601, Message: "method not found"}
		}
		if e != nil {
			write(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": e})
		} else {
			write(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}
		if subscribed {
			streamNotes(dir, &wmu)
			return 0
		}
	}
}

func statusWord(t Terminal) any {
	if t.Status == nil {
		return nil
	}
	return t.Status.State
}

// streamNotes writes each new line of notes.jsonl to stdout until stdin closes.
func streamNotes(dir string, wmu *sync.Mutex) {
	gone := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		for {
			if _, err := os.Stdin.Read(buf); err != nil {
				close(gone)
				return
			}
		}
	}()
	off := 0
	for {
		select {
		case <-gone:
			return
		case <-time.After(20 * time.Millisecond):
		}
		b, _ := os.ReadFile(filepath.Join(dir, "notes.jsonl"))
		if len(b) <= off {
			continue
		}
		chunk := b[off:]
		// only whole lines
		last := -1
		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] == '\n' {
				last = i
				break
			}
		}
		if last < 0 {
			continue
		}
		wmu.Lock()
		if _, err := os.Stdout.Write(chunk[:last+1]); err != nil {
			wmu.Unlock()
			return
		}
		wmu.Unlock()
		off += last + 1
	}
}

// fakeAttach paints once, then ticks until its stdout closes or it is killed.
func fakeAttach(dir, t string) int {
	fmt.Fprintf(os.Stdout, "\x1b[2J\x1b[HPAINT %s\x1b[31m red\x1b[0m\n", t)
	for i := 0; ; i++ {
		time.Sleep(50 * time.Millisecond)
		if _, err := fmt.Fprintf(os.Stdout, "tick %d\n", i); err != nil {
			return 0
		}
	}
}

// fake is the test-side handle on the fake programs.
type fake struct {
	t   *testing.T
	dir string
	r   *runner
}

func newFake(t *testing.T) *fake {
	t.Helper()
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r := &runner{
		progs:    Programs{HC: exe, HerdrApp: exe, Lsof: exe, Ps: exe, Ioreg: exe, Tailscale: exe},
		extraEnv: []string{"HR_FAKE_DIR=" + dir},
	}
	f := &fake{t: t, dir: dir, r: r}
	f.setState(fakeState{Seq: 100, Terminals: map[string]json.RawMessage{}, Reads: map[string]string{}, OnKeys: map[string]json.RawMessage{}, OnSend: map[string]json.RawMessage{}})
	f.write("ioreg.txt", `    | |     "HIDIdleTime" = 90006570458`+"\n")
	f.write("herdr-app-ls.json", `{"spaces":[],"recent":[]}`)
	return f
}

func (f *fake) state() fakeState { return loadFakeState(f.dir) }

func (f *fake) setState(st fakeState) { saveFakeState(f.dir, st) }

func (f *fake) write(name, content string) {
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(content), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fake) remove(name string) { os.Remove(filepath.Join(f.dir, name)) }

func (f *fake) setTerminal(term Terminal) {
	st := f.state()
	b, _ := json.Marshal(term)
	st.Terminals[term.Address] = b
	f.setState(st)
}

func (f *fake) removeTerminal(addr string) {
	st := f.state()
	delete(st.Terminals, addr)
	f.setState(st)
}

func (f *fake) setRead(addr, text string) {
	st := f.state()
	if st.Reads == nil {
		st.Reads = map[string]string{}
	}
	st.Reads[addr] = text
	f.setState(st)
}

func (f *fake) onKeys(addr string, patch string) {
	st := f.state()
	if st.OnKeys == nil {
		st.OnKeys = map[string]json.RawMessage{}
	}
	st.OnKeys[addr] = json.RawMessage(patch)
	f.setState(st)
}

func (f *fake) setKeysDelay(d time.Duration) {
	st := f.state()
	st.KeysDelayMS = int(d / time.Millisecond)
	f.setState(st)
}

type fakeReq struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

func (f *fake) requests() []fakeReq {
	b, _ := os.ReadFile(filepath.Join(f.dir, "requests.jsonl"))
	var out []fakeReq
	for _, line := range splitLines(b) {
		var r fakeReq
		if json.Unmarshal(line, &r) == nil {
			out = append(out, r)
		}
	}
	return out
}

func (f *fake) requestsFor(method string) []fakeReq {
	var out []fakeReq
	for _, r := range f.requests() {
		if r.Method == method {
			out = append(out, r)
		}
	}
	return out
}

func (f *fake) calls() [][]string {
	b, _ := os.ReadFile(filepath.Join(f.dir, "calls.jsonl"))
	var out [][]string
	for _, line := range splitLines(b) {
		var a []string
		if json.Unmarshal(line, &a) == nil {
			out = append(out, a)
		}
	}
	return out
}

// emit appends one notification line for the events connection to stream.
func (f *fake) emit(line string) { appendLine(f.dir, "notes.jsonl", []byte(line)) }

func (f *fake) emitEvent(typ, terminal string, seq int64, data string) {
	f.emit(fmt.Sprintf(`{"jsonrpc":"2.0","method":"event","params":{"data":%s,"seq":%d,"terminal":%q,"type":%q}}`, data, seq, terminal, typ))
}

func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func liveAgent(addr, app, state string, turn int64) Terminal {
	return Terminal{Address: addr, State: "live", Reported: true, Turn: turn, Cols: 200, Rows: 120, Labels: map[string]string{}, Status: &Status{State: state, App: app}}
}
