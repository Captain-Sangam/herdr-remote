package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultPageSize = 2 << 20
	resultCut       = 2000
	codexHint       = "Start Codex with `--no-daemon` for the full transcript"
	tooLargeText    = "[record too large]"
)

type transcriptItem struct {
	Role string `json:"role"`
	Kind string `json:"kind"` // text, tool_call or tool_result
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

type transcriptPage struct {
	Source     string           `json:"source"` // claude, codex or terminal
	Hint       string           `json:"hint,omitempty"`
	Model      string           `json:"model,omitempty"` // the last model named in this page
	Items      []transcriptItem `json:"items"`
	NextCursor string           `json:"next_cursor"`
}

type cursor struct {
	File   string `json:"f"`
	Inode  uint64 `json:"i"`
	Offset int64  `json:"o"`
}

func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursor, error) {
	var c cursor
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.File == "" || c.Offset < 0 {
		return c, errors.New("bad cursor")
	}
	return c, nil
}

// transcripts resolves and pages transcript files. Roots are fields so tests
// can point them at fixtures.
type transcripts struct {
	d         *Daemon
	claudeDir string // ~/.claude
	codexDir  string // ~/.codex
	pageSize  int64
}

func newTranscripts(d *Daemon) *transcripts {
	home, _ := os.UserHomeDir()
	return &transcripts{d: d, claudeDir: filepath.Join(home, ".claude"), codexDir: filepath.Join(home, ".codex"), pageSize: defaultPageSize}
}

// underRoot reports whether the real path of p is inside root.
func underRoot(p, root string) (string, bool) {
	rp, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", false
	}
	rr, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rr, rp)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rp, true
}

// claudeFile is source 1: the sessions file of the foreground pid, then the
// single projects file with that session id.
func (tr *transcripts) claudeFile(term *Terminal) string {
	if term.Foreground == nil || term.Foreground.PID <= 0 {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(tr.claudeDir, "sessions", strconv.Itoa(term.Foreground.PID)+".json"))
	if err != nil {
		return ""
	}
	var s struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(b, &s) != nil || s.SessionID == "" || strings.ContainsAny(s.SessionID, "/\\*?[") {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(tr.claudeDir, "projects", "*", s.SessionID+".jsonl"))
	if len(matches) != 1 {
		return ""
	}
	rp, ok := underRoot(matches[0], filepath.Join(tr.claudeDir, "projects"))
	if !ok {
		return ""
	}
	return rp
}

// codexFile is source 2: the rollout a `codex --no-daemon` child of the
// foreground process keeps open, whose session_meta has no parent thread.
func (tr *transcripts) codexFile(ctx context.Context, term *Terminal) string {
	if term.Foreground == nil || term.Foreground.PID <= 0 {
		return ""
	}
	out, err := tr.d.r.psTree(ctx)
	if err != nil {
		return ""
	}
	kids := map[int][]int{}
	comm := map[int]string{}
	for _, line := range splitLines(out) {
		f := strings.Fields(string(line))
		if len(f) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		kids[ppid] = append(kids[ppid], pid)
		comm[pid] = filepath.Base(strings.Join(f[2:], " "))
	}
	var codexPIDs []int
	queue := []int{term.Foreground.PID}
	seen := map[int]bool{}
	for len(queue) > 0 && len(seen) < 500 {
		p := queue[0]
		queue = queue[1:]
		if seen[p] {
			continue
		}
		seen[p] = true
		if comm[p] == "codex" {
			codexPIDs = append(codexPIDs, p)
		}
		queue = append(queue, kids[p]...)
	}
	root := filepath.Join(tr.codexDir, "sessions")
	for _, pid := range codexPIDs {
		out, err := tr.d.r.lsofFiles(ctx, pid)
		if err != nil {
			continue
		}
		for _, line := range splitLines(out) {
			if len(line) < 2 || line[0] != 'n' {
				continue
			}
			p := string(line[1:])
			if !strings.HasPrefix(filepath.Base(p), "rollout-") || !strings.HasSuffix(p, ".jsonl") {
				continue
			}
			rp, ok := underRoot(p, root)
			if !ok {
				continue
			}
			if isMainRollout(rp) {
				return rp
			}
		}
	}
	return ""
}

func isMainRollout(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	n, _ := f.Read(buf)
	line := buf[:n]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	var meta struct {
		Type    string `json:"type"`
		Payload struct {
			ParentThreadID *string `json:"parent_thread_id"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &meta) != nil || meta.Type != "session_meta" {
		return false
	}
	return meta.Payload.ParentThreadID == nil
}

var errCursor = errors.New("cursor is for another file")

// page reads one page of records back from end (or from the file end when end < 0).
func (tr *transcripts) page(path string, cur *cursor) ([][]byte, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	var inode uint64
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		inode = sys.Ino
	}
	end := st.Size()
	if cur != nil {
		if cur.File != path || cur.Inode != inode || cur.Offset > st.Size() {
			return nil, "", errCursor
		}
		end = cur.Offset
	} else {
		// A partial last line is ignored.
		end = tr.lineEnd(f, end)
	}
	if end == 0 {
		return nil, "", nil
	}
	start := max(0, end-tr.pageSize)
	buf := make([]byte, end-start)
	if _, err := f.ReadAt(buf, start); err != nil {
		return nil, "", err
	}
	var recs [][]byte
	nextEnd := int64(0)
	if start == 0 {
		recs = completeLines(buf)
	} else {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 || i == len(buf)-1 {
			// No complete record fits in a page: show one marker and skip the record.
			recStart := tr.prevNewline(f, start) + 1
			recs = [][]byte{[]byte(tooLargeText)}
			nextEnd = recStart
		} else {
			recs = completeLines(buf[i+1:])
			nextEnd = start + int64(i) + 1
		}
	}
	next := ""
	if nextEnd > 0 {
		next = encodeCursor(cursor{File: path, Inode: inode, Offset: nextEnd})
	}
	return recs, next, nil
}

// lineEnd returns end if the byte before it is a newline, else the position after the last newline.
func (tr *transcripts) lineEnd(f *os.File, end int64) int64 {
	if end == 0 {
		return 0
	}
	one := make([]byte, 1)
	if _, err := f.ReadAt(one, end-1); err == nil && one[0] == '\n' {
		return end
	}
	return tr.prevNewline(f, end) + 1
}

// prevNewline finds the last newline before pos, or -1.
func (tr *transcripts) prevNewline(f *os.File, pos int64) int64 {
	chunk := int64(64 << 10)
	for pos > 0 {
		start := max(0, pos-chunk)
		buf := make([]byte, pos-start)
		if _, err := f.ReadAt(buf, start); err != nil {
			return -1
		}
		if i := bytes.LastIndexByte(buf, '\n'); i >= 0 {
			return start + int64(i)
		}
		pos = start
	}
	return -1
}

func completeLines(buf []byte) [][]byte {
	var out [][]byte
	for len(buf) > 0 {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			break
		}
		if i > 0 {
			out = append(out, buf[:i])
		}
		buf = buf[i+1:]
	}
	return out
}

func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// keep whole runes
	i := n
	for i > 0 && i < len(s) && s[i]&0xc0 == 0x80 {
		i--
	}
	return s[:i] + "…"
}

func parseTS(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

func compactJSON(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}

// claudeItems turns one Claude Code record into items.
func claudeItems(rec []byte) []transcriptItem {
	if bytes.Equal(rec, []byte(tooLargeText)) {
		return []transcriptItem{{Role: "system", Kind: "text", Text: tooLargeText}}
	}
	var r struct {
		Type      string `json:"type"`
		IsMeta    bool   `json:"isMeta"`
		Timestamp string `json:"timestamp"`
		Message   struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(rec, &r) != nil || (r.Type != "user" && r.Type != "assistant") || r.IsMeta {
		return nil
	}
	ts := parseTS(r.Timestamp)
	var text string
	if json.Unmarshal(r.Message.Content, &text) == nil {
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return []transcriptItem{{Role: r.Type, Kind: "text", Text: text, TS: ts}}
	}
	var blocks []struct {
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Name    string          `json:"name"`
		Input   json.RawMessage `json:"input"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(r.Message.Content, &blocks) != nil {
		return nil
	}
	var out []transcriptItem
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				out = append(out, transcriptItem{Role: r.Type, Kind: "text", Text: b.Text, TS: ts})
			}
		case "tool_use":
			out = append(out, transcriptItem{Role: r.Type, Kind: "tool_call", Text: cut(b.Name+" "+compactJSON(b.Input), resultCut), TS: ts})
		case "tool_result":
			out = append(out, transcriptItem{Role: r.Type, Kind: "tool_result", Text: cut(contentText(b.Content), resultCut), TS: ts})
		}
	}
	return out
}

// recordModel returns the model a record names: message.model in Claude Code,
// payload.model in a Codex turn_context. Claude's own "<synthetic>" is not a model.
func recordModel(rec []byte) string {
	if !bytes.Contains(rec, []byte(`"model"`)) {
		return ""
	}
	var r struct {
		Message struct {
			Model string `json:"model"`
		} `json:"message"`
		Payload struct {
			Model string `json:"model"`
		} `json:"payload"`
	}
	if json.Unmarshal(rec, &r) != nil || r.Message.Model == "<synthetic>" {
		return ""
	}
	return r.Message.Model + r.Payload.Model
}

// contentText flattens a string or a list of text blocks.
func contentText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return compactJSON(raw)
	}
	var parts []string
	for _, b := range blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// codexItems turns one Codex rollout record into items.
func codexItems(rec []byte) []transcriptItem {
	if bytes.Equal(rec, []byte(tooLargeText)) {
		return []transcriptItem{{Role: "system", Kind: "text", Text: tooLargeText}}
	}
	var r struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Payload   struct {
			Type      string          `json:"type"`
			Role      string          `json:"role"`
			Content   json.RawMessage `json:"content"`
			Name      string          `json:"name"`
			Input     string          `json:"input"`
			Arguments string          `json:"arguments"`
			Output    json.RawMessage `json:"output"`
		} `json:"payload"`
	}
	if json.Unmarshal(rec, &r) != nil || r.Type != "response_item" {
		return nil
	}
	ts := parseTS(r.Timestamp)
	p := r.Payload
	switch p.Type {
	case "message":
		if p.Role != "user" && p.Role != "assistant" {
			return nil
		}
		text := contentText(p.Content)
		if strings.TrimSpace(text) == "" || (p.Role == "user" && (strings.HasPrefix(text, "# AGENTS.md instructions") || strings.HasPrefix(text, "<environment_context>"))) {
			return nil
		}
		return []transcriptItem{{Role: p.Role, Kind: "text", Text: text, TS: ts}}
	case "custom_tool_call":
		return []transcriptItem{{Role: "assistant", Kind: "tool_call", Text: cut(p.Name+" "+p.Input, resultCut), TS: ts}}
	case "function_call":
		return []transcriptItem{{Role: "assistant", Kind: "tool_call", Text: cut(p.Name+" "+p.Arguments, resultCut), TS: ts}}
	case "custom_tool_call_output", "function_call_output":
		return []transcriptItem{{Role: "user", Kind: "tool_result", Text: cut(contentText(p.Output), resultCut), TS: ts}}
	}
	return nil
}

// read resolves the source for a terminal and returns one page.
func (tr *transcripts) read(ctx context.Context, term *Terminal, cursorStr string) (*transcriptPage, error) {
	var cur *cursor
	if cursorStr != "" {
		c, err := decodeCursor(cursorStr)
		if err != nil {
			return nil, errCursor
		}
		cur = &c
	}
	source, path, parse := "", "", claudeItems
	if p := tr.claudeFile(term); p != "" {
		source, path = "claude", p
	} else if p := tr.codexFile(ctx, term); p != "" {
		source, path, parse = "codex", p, codexItems
	}
	if path != "" {
		if pg := tr.filePage(source, path, cur, parse); pg != nil {
			return pg, nil
		}
	}
	// No source, a file that cannot be read, or one with no record that parses:
	// terminal text. A cursor then belongs to another source: the phone starts again from the end.
	if cur != nil {
		return nil, errCursor
	}
	rr, err := tr.d.hc.Read(ctx, term.Address, 2000)
	if err != nil {
		return nil, err
	}
	pg := &transcriptPage{Source: "terminal", Items: []transcriptItem{{Role: "terminal", Kind: "text", Text: rr.Text, TS: time.Now().UnixMilli()}}}
	if term.Status != nil && term.Status.App == "codex" {
		pg.Hint = codexHint
	}
	return pg, nil
}

// filePage returns one page of the file, or nil when the file cannot be read
// or gives no record that parses, so the caller falls back to terminal text.
// The first request (no cursor) pages back until it has an item or reaches
// the start of the file. A later page with no item keeps its cursor.
func (tr *transcripts) filePage(source, path string, cur *cursor, parse func([]byte) []transcriptItem) *transcriptPage {
	first := cur == nil
	pg := &transcriptPage{Source: source, Items: []transcriptItem{}}
	for {
		recs, next, err := tr.page(path, cur)
		if err != nil {
			return nil
		}
		for _, rec := range recs {
			pg.Items = append(pg.Items, parse(rec)...)
			if m := recordModel(rec); m != "" {
				pg.Model = m
			}
		}
		pg.NextCursor = next
		if !first || len(pg.Items) > 0 {
			return pg
		}
		if next == "" {
			return nil
		}
		c, err := decodeCursor(next)
		if err != nil {
			return nil
		}
		cur = &c
	}
}

func (d *Daemon) handleTranscript(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	pg, err := d.transcripts.read(r.Context(), term, r.URL.Query().Get("cursor"))
	switch {
	case errors.Is(err, errCursor):
		apiError(w, 409, "cursor is for another file", "cursor")
	case errors.Is(err, errUnavailable):
		apiError(w, 503, "hc unavailable", "")
	case errors.Is(err, errNotFound):
		apiError(w, 404, "no such terminal", "")
	case err != nil:
		apiError(w, 503, "transcript unavailable", "")
	default:
		writeJSON(w, 200, pg)
	}
}
