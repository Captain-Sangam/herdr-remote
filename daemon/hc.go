package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

// Key is one of the keys the daemon may press: 1 to 9, y, esc or tab (docs/adr/0021).
type Key string

const (
	KeyOne Key = "1"
	KeyY   Key = "y"
	KeyEsc Key = "esc"
)

func allowedKey(k Key) bool {
	return k == KeyY || k == KeyEsc || k == "tab" || len(k) == 1 && k[0] >= '1' && k[0] <= '9'
}

var (
	errBadTarget   = errors.New("bad target")
	errNotFound    = errors.New("no such terminal")
	errNotAgent    = errors.New("not a live reporting terminal")
	errBlocked     = errors.New("terminal is blocked")
	errUnavailable = errors.New("hc unavailable")
)

var targetRe = regexp.MustCompile(`^t[0-9a-z]+$`)

// validTarget accepts a local terminal id only: no names, no machine/ prefix.
func validTarget(t string) bool { return targetRe.MatchString(t) }

// allowedMethods is every JSON-RPC method the daemon may send (ADR 0003).
var allowedMethods = map[string]bool{
	"hello":            true,
	"terminal.list":    true,
	"terminal.get":     true,
	"terminal.read":    true,
	"events.subscribe": true,
	"terminal.send":    true,
	"terminal.keys":    true,
}

// Terminal is the subset of hc's Terminal facts the daemon uses.
type Terminal struct {
	Address    string            `json:"address"`
	Name       string            `json:"name"`
	Cols       int               `json:"cols"`
	Rows       int               `json:"rows"`
	CreatedMS  int64             `json:"created_ms"`
	Reported   bool              `json:"reported"`
	Turn       int64             `json:"turn"`
	Title      string            `json:"title"`
	State      string            `json:"state"`
	Labels     map[string]string `json:"labels"`
	Foreground *Foreground       `json:"foreground"`
	Status     *Status           `json:"status"`
}

type Foreground struct {
	PID  int      `json:"pid"`
	Name string   `json:"name"`
	Argv []string `json:"argv"`
	Cwd  string   `json:"cwd"`
}

type Status struct {
	State string `json:"state"`
	App   string `json:"app"`
	Kind  string `json:"kind"`
	Msg   string `json:"msg"`
}

type ReadResult struct {
	Text     string `json:"text"`
	Complete bool   `json:"complete"`
	How      string `json:"how"`
	Why      string `json:"why"`
}

type SendResult struct {
	Turn    int64  `json:"turn"`
	Status  string `json:"status"`
	Started *bool  `json:"started"`
}

type KeysResult struct {
	Turn   int64  `json:"turn"`
	Status string `json:"status"`
}

type Snapshot struct {
	Seq       int64      `json:"seq"`
	Terminals []Terminal `json:"terminals"`
}

// Note is one line from a subscribed connection.
type Note struct {
	Method   string // event, heartbeat, missed or snapshot
	Type     string // for event: created, status, turn, title, labels, size, foreground, finished, removed, ...
	Terminal string
	Seq      int64
	Data     json.RawMessage
	Snapshot *Snapshot
}

// Event payloads seen in testdata/events-probe.jsonl.
type statusData struct {
	Reported bool    `json:"reported"`
	Status   *Status `json:"status"`
}
type turnData struct {
	State string `json:"state"`
	Turn  int64  `json:"turn"`
}
type titleData struct {
	Title string `json:"title"`
}
type labelsData struct {
	Labels map[string]string `json:"labels"`
}
type sizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

func parseNote(line []byte) (Note, error) {
	var raw struct {
		Method    string          `json:"method"`
		Params    json.RawMessage `json:"params"`
		Terminals json.RawMessage `json:"terminals"`
		Seq       int64           `json:"seq"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return Note{}, err
	}
	snapshot := func(b []byte) (Note, error) {
		var s Snapshot
		if err := json.Unmarshal(b, &s); err != nil {
			return Note{}, err
		}
		return Note{Method: "snapshot", Seq: s.Seq, Snapshot: &s}, nil
	}
	switch raw.Method {
	case "":
		if raw.Terminals != nil {
			return snapshot(line)
		}
		return Note{}, errors.New("not a notification")
	case "event":
		var p struct {
			Type     string          `json:"type"`
			Terminal string          `json:"terminal"`
			Seq      int64           `json:"seq"`
			Data     json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw.Params, &p); err != nil {
			return Note{}, err
		}
		return Note{Method: "event", Type: p.Type, Terminal: p.Terminal, Seq: p.Seq, Data: p.Data}, nil
	case "heartbeat", "missed":
		var p struct {
			Seq int64 `json:"seq"`
		}
		_ = json.Unmarshal(raw.Params, &p)
		return Note{Method: raw.Method, Seq: p.Seq}, nil
	default:
		// The fresh snapshot after `missed` was not observed; accept any
		// notification whose params carry a terminals array as one.
		var p struct {
			Terminals json.RawMessage `json:"terminals"`
		}
		if json.Unmarshal(raw.Params, &p) == nil && p.Terminals != nil {
			return snapshot(raw.Params)
		}
		return Note{Method: raw.Method}, nil
	}
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Kind string `json:"kind"`
		Hint string `json:"hint"`
	} `json:"data"`
}

func (e *rpcError) Error() string { return "hc: " + e.Data.Kind + ": " + e.Message }

type rpcReply struct {
	result json.RawMessage
	err    error
}

// hcConn is one `hc connect` subprocess speaking JSON-RPC lines.
type hcConn struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	cancel  context.CancelFunc
	wmu     sync.Mutex
	pmu     sync.Mutex
	nextID  int64
	pending map[int64]chan rpcReply
	notes   chan Note // set before events.subscribe; nil otherwise
	done    chan struct{}
	once    sync.Once
	err     error
}

func dialHC(ctx context.Context, r *runner, notes chan Note) (*hcConn, error) {
	pctx, cancel := context.WithCancel(context.Background())
	cmd, err := r.hcConnect(pctx)
	if err != nil {
		cancel()
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	c := &hcConn{cmd: cmd, stdin: stdin, cancel: cancel, pending: map[int64]chan rpcReply{}, notes: notes, done: make(chan struct{})}
	go c.readLoop(stdout)
	go func() {
		err := cmd.Wait()
		if err == nil {
			err = errors.New("hc connect exited")
		}
		c.fail(err)
	}()
	hctx, hcancel := context.WithTimeout(ctx, 5*time.Second)
	defer hcancel()
	if _, err := c.call(hctx, "hello", struct{}{}); err != nil {
		c.close()
		return nil, fmt.Errorf("hello: %w", err)
	}
	return c, nil
}

func (c *hcConn) readLoop(rd io.Reader) {
	br := bufio.NewReaderSize(rd, 1<<20)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			c.dispatch(line)
		}
		if err != nil {
			c.fail(err)
			return
		}
	}
}

func (c *hcConn) dispatch(line []byte) {
	var head struct {
		ID     *int64          `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
		Method string          `json:"method"`
	}
	if json.Unmarshal(line, &head) != nil {
		return
	}
	if head.ID != nil {
		c.pmu.Lock()
		ch := c.pending[*head.ID]
		delete(c.pending, *head.ID)
		c.pmu.Unlock()
		if ch != nil {
			r := rpcReply{result: head.Result}
			if head.Error != nil {
				r.err = head.Error
			}
			ch <- r
		}
		return
	}
	if c.notes == nil {
		return
	}
	n, err := parseNote(line)
	if err != nil {
		return
	}
	// Block rather than drop: a slow reader makes hc send `missed` and a fresh snapshot.
	select {
	case c.notes <- n:
	case <-c.done:
	}
}

func (c *hcConn) fail(err error) {
	c.once.Do(func() {
		c.err = err
		close(c.done)
		c.cancel()
		c.pmu.Lock()
		for id, ch := range c.pending {
			ch <- rpcReply{err: fmt.Errorf("%w: %v", errUnavailable, err)}
			delete(c.pending, id)
		}
		c.pmu.Unlock()
	})
}

func (c *hcConn) close() { c.fail(errors.New("closed")) }

func (c *hcConn) alive() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

func (c *hcConn) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if !allowedMethods[method] {
		return nil, fmt.Errorf("hc: method %q is not allowed", method)
	}
	id := atomic.AddInt64(&c.nextID, 1)
	ch := make(chan rpcReply, 1)
	c.pmu.Lock()
	c.pending[id] = ch
	c.pmu.Unlock()
	b, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}{"2.0", id, method, params})
	if err != nil {
		return nil, err
	}
	c.wmu.Lock()
	_, err = c.stdin.Write(append(b, '\n'))
	c.wmu.Unlock()
	if err != nil {
		c.fail(err)
		return nil, fmt.Errorf("%w: %v", errUnavailable, err)
	}
	select {
	case r := <-ch:
		return r.result, r.err
	case <-ctx.Done():
		c.pmu.Lock()
		delete(c.pending, id)
		c.pmu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		return nil, fmt.Errorf("%w: %v", errUnavailable, c.err)
	}
}

// HC holds the reads and actions connections and reconnects them with backoff.
// The events connection is separate: see Subscribe.
type HC struct {
	r        *runner
	mu       sync.Mutex
	conns    map[string]*hcConn
	nextDial map[string]time.Time
	delay    map[string]time.Duration
}

func newHC(r *runner) *HC {
	return &HC{r: r, conns: map[string]*hcConn{}, nextDial: map[string]time.Time{}, delay: map[string]time.Duration{}}
}

const maxBackoff = 30 * time.Second

func (h *HC) conn(ctx context.Context, which string) (*hcConn, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c := h.conns[which]; c != nil && c.alive() {
		return c, nil
	}
	if time.Now().Before(h.nextDial[which]) {
		return nil, errUnavailable
	}
	c, err := dialHC(ctx, h.r, nil)
	if err != nil {
		d := h.delay[which]
		if d == 0 {
			d = time.Second
		}
		h.nextDial[which] = time.Now().Add(d)
		h.delay[which] = min(2*d, maxBackoff)
		return nil, fmt.Errorf("%w: %v", errUnavailable, err)
	}
	h.delay[which] = 0
	h.conns[which] = c
	return c, nil
}

// call sends one allowed method on the named connection and decodes the result.
func (h *HC) call(ctx context.Context, which, method string, params, out any) error {
	c, err := h.conn(ctx, which)
	if err != nil {
		return err
	}
	raw, err := c.call(ctx, method, params)
	if err != nil {
		var re *rpcError
		if errors.As(err, &re) {
			switch re.Data.Kind {
			case "not_found":
				return errNotFound
			case "blocked":
				return errBlocked
			case "finished":
				return errNotAgent
			}
			return err
		}
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// Close ends every connection; used by tests and on shutdown.
func (h *HC) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, c := range h.conns {
		c.close()
		delete(h.conns, k)
	}
}

type terminalParam struct {
	Terminal string `json:"terminal"`
}

func (h *HC) List(ctx context.Context) ([]Terminal, error) {
	var res struct {
		Terminals []Terminal `json:"terminals"`
	}
	err := h.call(ctx, "reads", "terminal.list", struct{}{}, &res)
	return res.Terminals, err
}

func (h *HC) Get(ctx context.Context, t string) (*Terminal, error) {
	if !validTarget(t) {
		return nil, errBadTarget
	}
	var term Terminal
	if err := h.call(ctx, "reads", "terminal.get", terminalParam{t}, &term); err != nil {
		return nil, err
	}
	return &term, nil
}

func (h *HC) Read(ctx context.Context, t string, lines int) (*ReadResult, error) {
	if !validTarget(t) {
		return nil, errBadTarget
	}
	if lines < 0 {
		lines = 0
	}
	var res ReadResult
	err := h.call(ctx, "reads", "terminal.read", struct {
		Terminal string `json:"terminal"`
		Lines    int    `json:"lines"`
	}{t, lines}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// agentTarget refuses anything but a live, reporting local terminal.
func (h *HC) agentTarget(ctx context.Context, t string) (*Terminal, error) {
	term, err := h.Get(ctx, t)
	if err != nil {
		return nil, err
	}
	if term.State != "live" || !term.Reported {
		return nil, errNotAgent
	}
	return term, nil
}

// Send pastes text as a plain prompt: force is always false and there is no caller.
func (h *HC) Send(ctx context.Context, t, text string) (*SendResult, error) {
	if _, err := h.agentTarget(ctx, t); err != nil {
		return nil, err
	}
	var res SendResult
	err := h.call(ctx, "actions", "terminal.send", struct {
		Terminal string `json:"terminal"`
		Text     string `json:"text"`
		Force    bool   `json:"force"`
	}{t, text, false}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Keys presses exactly one allowed key.
func (h *HC) Keys(ctx context.Context, t string, k Key) (*KeysResult, error) {
	if !allowedKey(k) {
		return nil, fmt.Errorf("key %q is not allowed", string(k))
	}
	if _, err := h.agentTarget(ctx, t); err != nil {
		return nil, err
	}
	var res KeysResult
	err := h.call(ctx, "actions", "terminal.keys", struct {
		Terminal string   `json:"terminal"`
		Keys     []string `json:"keys"`
	}{t, []string{string(k)}}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Subscribe opens a fresh connection that from then on carries only notifications.
// The caller reads conn.notes until conn.done closes, then calls conn.close().
func (h *HC) Subscribe(ctx context.Context) (*Snapshot, *hcConn, error) {
	c, err := dialHC(ctx, h.r, make(chan Note, 256))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", errUnavailable, err)
	}
	raw, err := c.call(ctx, "events.subscribe", struct{}{})
	if err != nil {
		c.close()
		return nil, nil, fmt.Errorf("%w: %v", errUnavailable, err)
	}
	var s Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		c.close()
		return nil, nil, err
	}
	return &s, c, nil
}

func unmarshalData(data json.RawMessage, v any) error { return json.Unmarshal(data, v) }
