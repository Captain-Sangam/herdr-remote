package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
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
// per-device rate limit (docs/architecture.md section 3.5, and ADR 0012).
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

// hasMarkers checks the option markers verified in docs/interfaces.md.
func hasMarkers(app, block string) bool {
	switch app {
	case "claude-code":
		return strings.Contains(block, "1. Yes") && strings.Contains(block, "Esc to cancel")
	case "codex":
		return strings.Contains(block, "Yes, proceed (y)") && strings.Contains(block, "(esc)")
	}
	return false
}

// questionBlock is a Claude Code question: from its header (the question tabs) to the end of
// the screen, without the rules (docs/interfaces.md).
func questionBlock(screen string) string {
	lines := strings.Split(strings.TrimRight(screen, " \t\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		tl := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(tl, "←") && !strings.HasPrefix(tl, "☐") && !strings.HasPrefix(tl, "☒") {
			continue
		}
		var out []string
		for _, l := range lines[i:] {
			if strings.Trim(l, " ─…") != "" || l == "" {
				out = append(out, strings.TrimRight(l, " \t"))
			}
		}
		return strings.Join(out, "\n")
	}
	return ""
}

var optionRe = regexp.MustCompile(`^\s*(?:❯\s*)?([1-9])\.\s+(.+?)\s*$`)

// parseQuestion splits a question block into the text above its options and its keys. The
// options are the last numbered list that counts 1, 2, 3, each with the lines under it as its
// detail. "tab" (Next) follows a multi-select. A list that skips a number gives no keys.
// "Type something" is left out: it opens a text field that only the Mac can fill.
func parseQuestion(block string) (string, []promptOption) {
	lines := strings.Split(block, "\n")
	var run []promptOption
	start := 0
	cur := -1 // the option that takes the next detail line
	for i, l := range lines {
		m := optionRe.FindStringSubmatch(l)
		if m == nil {
			// A description line is indented. A blank line or a key hint ends it.
			if strings.TrimSpace(l) == "" || !strings.HasPrefix(l, " ") {
				cur = -1
			} else if cur >= 0 {
				run[cur].Detail = strings.TrimSpace(run[cur].Detail + " " + strings.TrimSpace(l))
			}
			continue
		}
		if m[1] == "1" {
			run, start = nil, i
		}
		if m[1] != strconv.Itoa(len(run)+1) {
			return "", nil
		}
		run = append(run, promptOption{Key: m[1], Label: m[2]})
		cur = len(run) - 1
	}
	if run == nil {
		return "", nil
	}
	var out []promptOption
	multi := false
	for _, o := range run {
		name := o.Label
		if j := strings.Index(name, "] "); strings.HasPrefix(name, "[") && j >= 0 {
			multi, name = true, name[j+2:]
		}
		if o.Detail == name {
			o.Detail = "" // the screen repeats the label
		}
		if !strings.Contains(o.Label, "Type something") {
			out = append(out, o)
		}
	}
	if multi {
		out = append(out, promptOption{Key: "tab", Label: "Next"})
	}
	return strings.TrimRight(strings.Join(lines[:start], "\n"), " \n"), out
}

func promptHash(app string, turn int64, msg, block string) string {
	h := sha256.New()
	for _, part := range []string{app, strconv.FormatInt(turn, 10), msg, block} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

type promptOption struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"` // the description under the option
}

type promptView struct {
	App        string         `json:"app"`
	Kind       string         `json:"kind"`
	Msg        string         `json:"msg"`
	Block      string         `json:"block"`
	Question   string         `json:"question,omitempty"` // a question's block above its options
	Options    []promptOption `json:"options,omitempty"`  // a question's keys
	Turn       int64          `json:"turn"`
	PromptHash string         `json:"prompt_hash"`
	Actionable bool           `json:"actionable"`
	Reason     string         `json:"reason,omitempty"`
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
		} else if !(v.Kind == "permission" && (v.App == "claude-code" || v.App == "codex")) && !(v.Kind == "question" && v.App == "claude-code") {
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
		ok := false
		if v.Kind == "question" {
			v.Block = questionBlock(rr.Text)
			v.Question, v.Options = parseQuestion(v.Block)
			ok = len(v.Options) > 0 && (strings.Contains(v.Block, "Esc to cancel") || strings.Contains(v.Block, "Submit answers"))
		} else {
			v.Block = promptBlock(rr.Text)
			ok = v.Block != "" && hasMarkers(v.App, v.Block)
		}
		if !ok {
			v.Reason = "no_markers"
			v.Block, v.Question, v.Options = "", "", nil
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

// handleDecision runs the approval guard. The Rule N comments are the rules in
// docs/architecture.md section 3.5.
func (d *Daemon) handleDecision(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		RequestID  string `json:"request_id"`
		Decision   string `json:"decision"`
		Key        string `json:"key"` // the option, for an answer
		Turn       int64  `json:"turn"`
		PromptHash string `json:"prompt_hash"`
	}
	if !readBody(w, r, &body) {
		return
	}
	answer := body.Decision == "answer"
	if (body.Decision != "approve" && body.Decision != "deny" && !answer) || answer != (body.Key != "") || body.PromptHash == "" || body.Turn < 0 {
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
	// An answer is for a question, approve and deny for a permission.
	if !v.Actionable || answer != (v.Kind == "question") {
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
	key := KeyEsc
	switch {
	case answer:
		key = ""
		for _, o := range v.Options {
			if o.Key == body.Key {
				key = Key(o.Key)
			}
		}
		if key == "" {
			a.release(body.RequestID, e)
			apiError(w, 400, "no such option", "")
			return
		}
	case body.Decision == "approve" && v.App == "codex":
		key = KeyY
	case body.Decision == "approve":
		key = KeyOne
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
	decision := body.Decision
	if answer {
		decision += ":" + body.Key // the key only, never the option's text
	}
	if err := d.audit.write(auditLine{Action: "decision", State: "dispatching", DeviceID: dev.ID, Terminal: t, Turn: &turn, RequestID: body.RequestID, Decision: decision, PromptHash: v.PromptHash}); err != nil {
		a.release(body.RequestID, e)
		apiError(w, 503, "audit write failed", "")
		return
	}

	// Rule 7: the marker, then exactly one key.
	a.mu.Lock()
	a.consumed[t] = marker{v.Turn, v.PromptHash}
	a.mu.Unlock()
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
				// A question stays blocked between its steps: a new screen shows the key landed.
				cur, _, err := a.prompt(ctx, t)
				if err == nil && (cur.Reason == "not_blocked" || cur.Turn != v.Turn || cur.PromptHash != v.PromptHash) {
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
