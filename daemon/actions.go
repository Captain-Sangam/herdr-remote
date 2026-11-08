package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	requestWindow    = 5 * time.Minute  // a request id is valid this long from its UUIDv7 time
	requestRetention = 10 * time.Minute // and remembered this long
	messageLimit     = 4000
	settleTime       = 3 * time.Second
	keyWaitTime      = 120 * time.Second // hc holds a key while someone types at the Mac
)

type marker struct {
	turn int64
	hash string
}

type reqResult struct {
	status int
	body   map[string]any
}

type reqEntry struct {
	done chan struct{}
	res  *reqResult
	at   time.Time
}

// actions holds the per-terminal locks, consumed markers, request ids and the
// per-device rate limit (section 3.5 and decision 12).
type actions struct {
	d        *Daemon
	mu       sync.Mutex
	reqs     map[string]*reqEntry
	consumed map[string]marker
	locks    map[string]*sync.Mutex
	rate     map[string][]time.Time
	settle   time.Duration
	keyWait  time.Duration // how long to wait for hc's reply to terminal.keys
	perSec   int           // actions allowed per device per second
}

func newActions(d *Daemon) *actions {
	a := &actions{d: d, reqs: map[string]*reqEntry{}, consumed: map[string]marker{}, locks: map[string]*sync.Mutex{}, rate: map[string][]time.Time{}, settle: settleTime, keyWait: keyWaitTime, perSec: 2}
	a.rebuild()
	return a
}

// rebuild restores request ids and consumed markers from the last 10 minutes
// of audit.log and audit.log.1, so a restart never repeats a key or a send.
func (a *actions) rebuild() {
	for _, l := range a.d.audit.recent(time.Now().Add(-requestRetention)) {
		if l.RequestID == "" || (l.Action != "decision" && l.Action != "message") {
			continue
		}
		ts, _ := time.Parse(time.RFC3339Nano, l.TS)
		switch l.State {
		case "dispatching":
			e := &reqEntry{done: make(chan struct{}), res: &reqResult{200, map[string]any{"result": "outcome_unknown"}}, at: ts}
			close(e.done)
			a.reqs[l.RequestID] = e
			if l.Action == "decision" && l.Turn != nil {
				a.consumed[l.Terminal] = marker{*l.Turn, l.PromptHash}
			}
		case "result":
			if e := a.reqs[l.RequestID]; e != nil {
				e.res = &reqResult{200, map[string]any{"result": l.Result}}
			}
		}
	}
}

func (a *actions) lock(t string) *sync.Mutex {
	a.mu.Lock()
	defer a.mu.Unlock()
	m := a.locks[t]
	if m == nil {
		m = &sync.Mutex{}
		a.locks[t] = m
	}
	return m
}

// allow is the rate limit: more than 2 actions in one second from one device is refused.
func (a *actions) allow(dev string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	var kept []time.Time
	for _, t := range a.rate[dev] {
		if now.Sub(t) < time.Second {
			kept = append(kept, t)
		}
	}
	if len(kept) >= a.perSec {
		a.rate[dev] = kept
		return false
	}
	a.rate[dev] = append(kept, now)
	return true
}

// uuid7Time returns the millisecond timestamp of a UUIDv7, or an error.
func uuid7Time(id string) (time.Time, error) {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' || id[14] != '7' {
		return time.Time{}, errors.New("not a UUIDv7")
	}
	hexs := strings.ReplaceAll(id, "-", "")
	if _, err := hex.DecodeString(hexs); err != nil {
		return time.Time{}, errors.New("not a UUIDv7")
	}
	ms, err := strconv.ParseInt(hexs[:12], 16, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms), nil
}

// claim registers a request id. It returns the existing entry when the id is
// known (the caller returns its result), else a fresh in-flight entry.
func (a *actions) claim(id string) (*reqEntry, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, e := range a.reqs {
		if now.Sub(e.at) > requestRetention {
			delete(a.reqs, k)
		}
	}
	if e := a.reqs[id]; e != nil {
		return e, true
	}
	e := &reqEntry{done: make(chan struct{}), at: now}
	a.reqs[id] = e
	return e, false
}

func (a *actions) finish(e *reqEntry, status int, body map[string]any) {
	a.mu.Lock()
	e.res = &reqResult{status, body}
	a.mu.Unlock()
	close(e.done)
}

// release forgets a request that was refused before dispatch, so a retry is re-evaluated.
func (a *actions) release(id string, e *reqEntry) {
	a.mu.Lock()
	delete(a.reqs, id)
	a.mu.Unlock()
	close(e.done)
}

func (a *actions) waitResult(ctx context.Context, e *reqEntry) *reqResult {
	select {
	case <-e.done:
		a.mu.Lock()
		defer a.mu.Unlock()
		return e.res
	case <-ctx.Done():
		return nil
	}
}

var dialogStarts = []string{"Do you want", "Would you like"}
var dialogEnds = []string{"Esc to cancel", "Press enter to confirm"}

// promptBlock is the screen region from the dialog's first line to its key hint line.
func promptBlock(screen string) string {
	lines := strings.Split(screen, "\n")
	start := -1
	for i, l := range lines {
		tl := strings.TrimSpace(l)
		for _, s := range dialogStarts {
			if strings.HasPrefix(tl, s) {
				start = i
			}
		}
	}
	if start < 0 {
		return ""
	}
	for j := start; j < len(lines); j++ {
		for _, e := range dialogEnds {
			if strings.Contains(lines[j], e) {
				var out []string
				for _, l := range lines[start : j+1] {
					out = append(out, strings.TrimRight(l, " \t"))
				}
				return strings.Join(out, "\n")
			}
		}
	}
	return ""
}

// hasMarkers checks the option markers verified in sections 2.6 and 2.7.
func hasMarkers(app, block string) bool {
	switch app {
	case "claude-code":
		return strings.Contains(block, "1. Yes") && strings.Contains(block, "Esc to cancel")
	case "codex":
		return strings.Contains(block, "Yes, proceed (y)") && strings.Contains(block, "(esc)")
	}
	return false
}

func promptHash(app string, turn int64, msg, block string) string {
	h := sha256.New()
	for _, part := range []string{app, strconv.FormatInt(turn, 10), msg, block} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

type promptView struct {
	App        string `json:"app"`
	Kind       string `json:"kind"`
	Msg        string `json:"msg"`
	Block      string `json:"block"`
	Turn       int64  `json:"turn"`
	PromptHash string `json:"prompt_hash"`
	Actionable bool   `json:"actionable"`
	Reason     string `json:"reason,omitempty"`
}

// prompt reads the fresh facts and screen of a terminal and describes its dialog.
func (a *actions) prompt(ctx context.Context, t string) (*promptView, *Terminal, error) {
	term, err := a.d.hc.Get(ctx, t)
	if err != nil {
		return nil, nil, err
	}
	if !isAgent(term) {
		return nil, term, errNotAgent
	}
	v := &promptView{Turn: term.Turn}
	if term.Status != nil {
		v.App, v.Kind, v.Msg = term.Status.App, term.Status.Kind, term.Status.Msg
		if term.Status.State != "blocked" {
			v.Reason = "not_blocked"
		} else if v.Kind != "permission" || (v.App != "claude-code" && v.App != "codex") {
			v.Reason = "view_only"
		}
	} else {
		v.Reason = "not_blocked"
	}
	if v.Reason == "" {
		rr, err := a.d.hc.Read(ctx, t, 0)
		if err != nil {
			return nil, term, err
		}
		v.Block = promptBlock(rr.Text)
		if v.Block == "" || !hasMarkers(v.App, v.Block) {
			v.Reason = "no_markers"
			v.Block = ""
		} else {
			v.Actionable = true
		}
	}
	v.PromptHash = promptHash(v.App, v.Turn, v.Msg, v.Block)
	return v, term, nil
}

func (d *Daemon) handlePrompt(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	v, _, err := d.actions.prompt(r.Context(), term.Address)
	if err != nil {
		writeHCError(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func writeHCError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBadTarget):
		apiError(w, 400, "bad target", "")
	case errors.Is(err, errNotFound):
		apiError(w, 404, "no such terminal", "")
	case errors.Is(err, errNotAgent):
		apiError(w, 409, "not an agent", "not_agent")
	case errors.Is(err, errBlocked):
		apiError(w, 409, "agent is blocked", "blocked")
	default:
		apiError(w, 503, "hc unavailable", "")
	}
}

func readBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(v); err != nil {
		apiError(w, 400, "bad body", "")
		return false
	}
	return true
}

// checkRequestID validates the id and its 5-minute window. It writes the error itself.
func (a *actions) checkRequestID(w http.ResponseWriter, id string) bool {
	ts, err := uuid7Time(id)
	if err != nil {
		apiError(w, 400, "request_id must be a UUIDv7", "")
		return false
	}
	if d := time.Since(ts); d > requestWindow || d < -requestWindow {
		apiError(w, 409, "request too old or too far ahead", "expired_request")
		return false
	}
	return true
}

func (d *Daemon) handleDecision(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		RequestID  string `json:"request_id"`
		Decision   string `json:"decision"`
		Turn       int64  `json:"turn"`
		PromptHash string `json:"prompt_hash"`
	}
	if !readBody(w, r, &body) {
		return
	}
	if (body.Decision != "approve" && body.Decision != "deny") || body.PromptHash == "" || body.Turn < 0 {
		apiError(w, 400, "bad body", "")
		return
	}
	a := d.actions
	if !a.checkRequestID(w, body.RequestID) {
		return
	}
	dev := deviceOf(r)
	e, known := a.claim(body.RequestID)
	if known {
		if res := a.waitResult(r.Context(), e); res != nil {
			writeJSON(w, res.status, res.body)
		}
		return
	}
	if !a.allow(dev.ID) {
		a.release(body.RequestID, e)
		apiError(w, 429, "too many actions", "")
		return
	}
	t := term.Address
	lk := a.lock(t)
	lk.Lock()
	defer lk.Unlock()
	ctx := r.Context()
	refuse := func(status int, msg, reason string, extra map[string]any) {
		a.release(body.RequestID, e)
		b := errBody(msg, reason)
		for k, v := range extra {
			b[k] = v
		}
		if status == 409 {
			d.audit.write(auditLine{Action: "refusal", DeviceID: dev.ID, Terminal: t, RequestID: body.RequestID, Result: reason})
		}
		writeJSON(w, status, b)
	}

	// Rule 4: the same state, kind, app, turn, hash and markers, read fresh.
	v, _, err := a.prompt(ctx, t)
	if err != nil {
		a.release(body.RequestID, e)
		writeHCError(w, err)
		return
	}
	if !v.Actionable {
		refuse(409, "prompt is not actionable", "not_actionable", nil)
		return
	}
	if v.Turn != body.Turn {
		refuse(409, "the prompt's turn changed", "stale", nil)
		return
	}
	if v.PromptHash != body.PromptHash {
		refuse(409, "the prompt changed", "changed", nil)
		return
	}
	a.mu.Lock()
	m, used := a.consumed[t]
	a.mu.Unlock()
	if used && m.turn == v.Turn && m.hash == v.PromptHash {
		refuse(409, "already answered", "already_answered", nil)
		return
	}

	// Rule 5: the Mac-in-use guard, last. It fails closed.
	idleAt := time.Now()
	idle, err := d.r.MacIdle(ctx)
	if err != nil {
		refuse(409, "cannot read Mac idle time", "mac_in_use", map[string]any{"idle_ms": 0})
		return
	}
	if limit := d.cfg.macIdle(); limit > 0 && idle < limit {
		ms := idle.Milliseconds()
		d.audit.write(auditLine{Action: "refusal", DeviceID: dev.ID, Terminal: t, RequestID: body.RequestID, Result: "mac_in_use", IdleMS: &ms})
		a.release(body.RequestID, e)
		b := errBody("the Mac is in use", "mac_in_use")
		b["idle_ms"] = ms
		writeJSON(w, 409, b)
		return
	}

	// Rule 6: a durable dispatching line before any key.
	turn := v.Turn
	if err := d.audit.write(auditLine{Action: "decision", State: "dispatching", DeviceID: dev.ID, Terminal: t, Turn: &turn, RequestID: body.RequestID, Decision: body.Decision, PromptHash: v.PromptHash}); err != nil {
		a.release(body.RequestID, e)
		apiError(w, 503, "audit write failed", "")
		return
	}

	// Rule 7: the marker, then exactly one key.
	a.mu.Lock()
	a.consumed[t] = marker{v.Turn, v.PromptHash}
	a.mu.Unlock()
	key := KeyEsc
	if body.Decision == "approve" {
		if v.App == "codex" {
			key = KeyY
		} else {
			key = KeyOne
		}
	}
	// From here on the phone may give up, but the key and the result still land.
	// hc holds the key while someone types at the Mac, so the wait covers that
	// queue and the idle check after the key runs when hc delivers it.
	ctx, cancel := context.WithTimeout(context.Background(), a.keyWait)
	defer cancel()
	result := "outcome_unknown"
	if _, err := d.hc.Keys(ctx, t, key); err == nil {
		// Rule 8: input during the call makes the outcome uncertain.
		idle2, err := d.r.MacIdle(ctx)
		if err != nil || idle2 < time.Since(idleAt) {
			result = "uncertain"
		} else {
			result = "sent_unconfirmed"
			deadline := time.Now().Add(a.settle)
			for time.Now().Before(deadline) {
				cur, err := d.hc.Get(ctx, t)
				if err == nil && (cur.Turn != v.Turn || cur.Status == nil || cur.Status.State != "blocked") {
					result = "applied"
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	d.audit.write(auditLine{Action: "decision", State: "result", DeviceID: dev.ID, Terminal: t, RequestID: body.RequestID, Result: result})
	a.finish(e, 200, map[string]any{"result": result})
	writeJSON(w, 200, map[string]any{"result": result})
}

// cleanMessage drops C0 and C1 control characters, keeping newline and tab.
func cleanMessage(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == utf8.RuneError {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (d *Daemon) handleMessage(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		RequestID string `json:"request_id"`
		Text      string `json:"text"`
	}
	if !readBody(w, r, &body) {
		return
	}
	text := cleanMessage(body.Text)
	n := utf8.RuneCountInString(text)
	if n == 0 {
		apiError(w, 400, "empty text", "")
		return
	}
	if n > messageLimit {
		apiError(w, 400, "text too long", "")
		return
	}
	a := d.actions
	if !a.checkRequestID(w, body.RequestID) {
		return
	}
	dev := deviceOf(r)
	e, known := a.claim(body.RequestID)
	if known {
		if res := a.waitResult(r.Context(), e); res != nil {
			writeJSON(w, res.status, res.body)
		}
		return
	}
	if !a.allow(dev.ID) {
		a.release(body.RequestID, e)
		apiError(w, 429, "too many actions", "")
		return
	}
	t := term.Address
	ctx := r.Context()
	cur, err := d.hc.Get(ctx, t)
	if err != nil {
		a.release(body.RequestID, e)
		writeHCError(w, err)
		return
	}
	if !isAgent(cur) {
		a.release(body.RequestID, e)
		apiError(w, 409, "not an agent", "not_agent")
		return
	}
	if cur.Status != nil && cur.Status.State == "blocked" {
		a.release(body.RequestID, e)
		d.audit.write(auditLine{Action: "refusal", DeviceID: dev.ID, Terminal: t, RequestID: body.RequestID, Result: "blocked"})
		apiError(w, 409, "agent is blocked", "blocked")
		return
	}
	if err := d.audit.write(auditLine{Action: "message", State: "dispatching", DeviceID: dev.ID, Terminal: t, RequestID: body.RequestID, Length: &n}); err != nil {
		a.release(body.RequestID, e)
		apiError(w, 503, "audit write failed", "")
		return
	}
	sctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	status, res := 200, "sent"
	if _, err := d.hc.Send(sctx, t, text); err != nil {
		if errors.Is(err, errBlocked) {
			status, res = 409, "blocked"
		} else {
			res = "outcome_unknown"
		}
	}
	d.audit.write(auditLine{Action: "message", State: "result", DeviceID: dev.ID, Terminal: t, RequestID: body.RequestID, Result: res})
	var out map[string]any
	if status == 409 {
		out = errBody("agent is blocked", "blocked")
	} else {
		out = map[string]any{"result": res}
	}
	a.finish(e, status, out)
	writeJSON(w, status, out)
}
