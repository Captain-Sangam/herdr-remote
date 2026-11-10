package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type trFixture struct {
	d      *testDaemon
	cert   tls.Certificate
	claude string
	codex  string
}

func setupTranscripts(t *testing.T) *trFixture {
	t.Helper()
	d := startTestDaemon(t, func(f *fake) {
		claude := liveAgent("t1", "claude-code", "done", 3)
		claude.Foreground = &Foreground{PID: 4242, Name: "claude.exe"}
		f.setTerminal(claude)
		f.setRead("t1", "screen of t1")
		codex := liveAgent("t2", "codex", "idle", 2)
		codex.Foreground = &Foreground{PID: 4343, Name: "node"}
		f.setTerminal(codex)
		f.setRead("t2", "screen of t2")
	})
	root := t.TempDir()
	f := &trFixture{d: d, claude: filepath.Join(root, "claude"), codex: filepath.Join(root, "codex")}
	d.transcripts.claudeDir, d.transcripts.codexDir = f.claude, f.codex
	os.MkdirAll(filepath.Join(f.claude, "sessions"), 0o700)
	os.MkdirAll(filepath.Join(f.claude, "projects", "-x"), 0o700)
	os.MkdirAll(filepath.Join(f.codex, "sessions", "2026", "11", "08"), 0o700)
	f.cert, _ = d.pairPhone(t, "p")
	d.waitOK(t)
	return f
}

func (f *trFixture) claudeFile(sid string) string {
	return filepath.Join(f.claude, "projects", "-x", sid+".jsonl")
}

func (f *trFixture) bindClaude(t *testing.T, pid int, sid string) {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"pid": pid, "sessionId": sid, "cwd": "/x"})
	if err := os.WriteFile(filepath.Join(f.claude, "sessions", fmt.Sprint(pid)+".json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeLines(t *testing.T, path string, lines []string, trailingNewline bool) {
	t.Helper()
	s := strings.Join(lines, "\n")
	if trailingNewline {
		s += "\n"
	}
	if err := os.WriteFile(path, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
}

func claudeUser(text string) string {
	b, _ := json.Marshal(map[string]any{"type": "user", "timestamp": "2026-11-08T07:26:00.000Z", "message": map[string]any{"role": "user", "content": text}})
	return string(b)
}

func (f *trFixture) get(t *testing.T, path string) (int, transcriptPage, map[string]any) {
	t.Helper()
	st, b := f.d.get(t, &f.cert, path)
	var pg transcriptPage
	var raw map[string]any
	json.Unmarshal(b, &pg)
	json.Unmarshal(b, &raw)
	return st, pg, raw
}

func texts(items []transcriptItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Text)
	}
	return out
}

// pageAll follows next_cursor back to the start and returns every text, oldest first.
func (f *trFixture) pageAll(t *testing.T, path string) ([]string, int) {
	t.Helper()
	var all []string
	cursor := ""
	pages := 0
	for {
		url := path
		if cursor != "" {
			url += "?cursor=" + cursor
		}
		st, pg, _ := f.get(t, url)
		if st != 200 {
			t.Fatalf("page %d: %d", pages, st)
		}
		pages++
		all = append(texts(pg.Items), all...)
		if pg.NextCursor == "" {
			return all, pages
		}
		cursor = pg.NextCursor
		if pages > 200 {
			t.Fatal("too many pages")
		}
	}
}

func TestTranscriptClaudeParses(t *testing.T) {
	f := setupTranscripts(t)
	f.bindClaude(t, 4242, "sid-1")
	long := strings.Repeat("r", 3000)
	lines := []string{
		`{"type":"attachment","attachment":{"x":1}}`,
		`{"type":"user","isMeta":true,"message":{"content":"<local-command>"}}`,
		claudeUser("hello"),
		`{"type":"assistant","timestamp":"2026-11-08T07:26:14.714Z","message":{"model":"claude-opus-5-5","content":[{"type":"thinking","thinking":"secret"},{"type":"text","text":"hi"},{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}]}}`,
		`{"type":"user","timestamp":"2026-11-08T07:26:15.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"tu1","content":"` + long + `"}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu2","content":[{"type":"text","text":"part a"},{"type":"text","text":"part b"}]}]}}`,
		`{"type":"assistant","message":{"model":"<synthetic>","content":"No response requested."}}`,
		`{"type":"ai-title","title":"x"}`,
	}
	writeLines(t, f.claudeFile("sid-1"), lines, true)
	st, pg, raw := f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || pg.Source != "claude" || pg.NextCursor != "" || pg.Model != "claude-opus-5-5" {
		t.Fatalf("%d %+v", st, pg)
	}
	if _, has := raw["hint"]; has {
		t.Fatal("hint on a claude transcript")
	}
	want := []struct{ role, kind, text string }{
		{"user", "text", "hello"},
		{"assistant", "text", "hi"},
		{"assistant", "tool_call", `Bash {"command":"ls"}`},
		{"user", "tool_result", strings.Repeat("r", 2000) + "…"},
		{"user", "tool_result", "part a\npart b"},
		{"assistant", "text", "No response requested."},
	}
	if len(pg.Items) != len(want) {
		t.Fatalf("items %v", texts(pg.Items))
	}
	for i, w := range want {
		it := pg.Items[i]
		if it.Role != w.role || it.Kind != w.kind || it.Text != w.text {
			t.Fatalf("item %d: %+v", i, it)
		}
	}
	if pg.Items[1].TS != 1794122774714 || pg.Items[0].TS == 0 {
		t.Fatalf("ts %d %d", pg.Items[0].TS, pg.Items[1].TS)
	}
	if strings.Contains(fmt.Sprint(pg.Items), "secret") {
		t.Fatal("thinking leaked")
	}
}

func (f *trFixture) rollout(name, parent string, lines []string) string {
	path := filepath.Join(f.codex, "sessions", "2026", "11", "08", name)
	meta := map[string]any{"timestamp": "2026-11-08T07:26:13.421Z", "type": "session_meta", "payload": map[string]any{"id": "thr", "cwd": "/x", "source": "cli"}}
	if parent != "" {
		meta["payload"].(map[string]any)["parent_thread_id"] = parent
	}
	mb, _ := json.Marshal(meta)
	s := string(mb) + "\n" + strings.Join(lines, "\n") + "\n"
	os.WriteFile(path, []byte(s), 0o600)
	return path
}

func codexMsg(role, text string) string {
	kind := "input_text"
	if role == "assistant" {
		kind = "output_text"
	}
	b, _ := json.Marshal(map[string]any{"timestamp": "2026-11-08T07:26:20.000Z", "type": "response_item", "payload": map[string]any{"type": "message", "role": role, "content": []map[string]string{{"type": kind, "text": text}}}})
	return string(b)
}

func (f *trFixture) bindCodex(t *testing.T, codexPID int, paths ...string) {
	t.Helper()
	f.d.f.write("ps.txt", fmt.Sprintf("4343 1 node\n%d 4343 /opt/x/bin/codex\n4345 %d /opt/x/bin/codex-code-mode-host\n", codexPID, codexPID))
	var sb strings.Builder
	fmt.Fprintf(&sb, "p%d\nfcwd\nn/x\n", codexPID)
	for _, p := range paths {
		fmt.Fprintf(&sb, "f5\nn%s\n", p)
	}
	f.d.f.write(fmt.Sprintf("lsof-%d.txt", codexPID), sb.String())
}

func TestTranscriptCodexParsesAndIgnoresGuardian(t *testing.T) {
	f := setupTranscripts(t)
	main := f.rollout("rollout-2026-11-08T12-56-13-main.jsonl", "", []string{
		codexMsg("developer", "You are Codex"),
		codexMsg("user", "# AGENTS.md instructions\n\n<INSTRUCTIONS>"),
		codexMsg("user", "Run: touch ok2"),
		codexMsg("assistant", "I'll run touch."),
		`{"timestamp":"2026-11-08T07:26:21.000Z","type":"response_item","payload":{"type":"custom_tool_call","name":"exec","input":"touch ok2","call_id":"c1"}}`,
		`{"timestamp":"2026-11-08T07:26:22.000Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"c1","output":[{"type":"input_text","text":"Script completed"},{"type":"input_text","text":"{\"chunk\":1}"}]}}`,
		`{"timestamp":"2026-11-08T07:26:23.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{\"cmd\":\"ls\"}","call_id":"c2"}}`,
		`{"timestamp":"2026-11-08T07:26:24.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c2","output":"a b c"}}`,
		`{"timestamp":"2026-11-08T07:26:25.000Z","type":"event_msg","payload":{"type":"task_complete"}}`,
		`{"timestamp":"2026-11-08T07:26:26.000Z","type":"turn_context","payload":{"model":"gpt-6.1-sol"}}`,
	})
	guardian := f.rollout("rollout-2026-11-08T12-56-14-guardian.jsonl", "thr", []string{codexMsg("user", "guardian only")})
	f.bindCodex(t, 4344, guardian, main)
	st, pg, raw := f.get(t, "/v1/terminals/t2/transcript")
	if st != 200 || pg.Source != "codex" || pg.Model != "gpt-6.1-sol" {
		t.Fatalf("%d %+v", st, pg)
	}
	if _, has := raw["hint"]; has {
		t.Fatal("hint on a bound codex transcript")
	}
	want := []struct{ role, kind, text string }{
		{"user", "text", "Run: touch ok2"},
		{"assistant", "text", "I'll run touch."},
		{"assistant", "tool_call", "exec touch ok2"},
		{"user", "tool_result", "Script completed\n{\"chunk\":1}"},
		{"assistant", "tool_call", `shell {"cmd":"ls"}`},
		{"user", "tool_result", "a b c"},
	}
	if len(pg.Items) != len(want) {
		t.Fatalf("items %v", texts(pg.Items))
	}
	for i, w := range want {
		it := pg.Items[i]
		if it.Role != w.role || it.Kind != w.kind || it.Text != w.text || it.TS == 0 {
			t.Fatalf("item %d: %+v", i, it)
		}
	}
	if strings.Contains(fmt.Sprint(pg.Items), "guardian") || strings.Contains(fmt.Sprint(pg.Items), "You are Codex") {
		t.Fatalf("guardian or developer text shown: %v", texts(pg.Items))
	}
}

func TestTranscriptPartialLastLineIgnored(t *testing.T) {
	f := setupTranscripts(t)
	f.bindClaude(t, 4242, "sid-p")
	writeLines(t, f.claudeFile("sid-p"), []string{claudeUser("one"), claudeUser("two")[:20]}, false)
	_, pg, _ := f.get(t, "/v1/terminals/t1/transcript")
	if got := texts(pg.Items); len(got) != 1 || got[0] != "one" {
		t.Fatalf("%v", got)
	}
	writeLines(t, f.claudeFile("sid-p"), []string{claudeUser("one"), claudeUser("two")}, true)
	_, pg, _ = f.get(t, "/v1/terminals/t1/transcript")
	if got := texts(pg.Items); len(got) != 2 || got[1] != "two" {
		t.Fatalf("%v", got)
	}
}

func TestTranscriptPagesBackWithoutLossOrRepeat(t *testing.T) {
	f := setupTranscripts(t)
	f.bindClaude(t, 4242, "sid-pg")
	var lines, want []string
	for i := 0; i < 40; i++ {
		text := fmt.Sprintf("msg-%02d %s", i, strings.Repeat("x", 20+i%7))
		lines = append(lines, claudeUser(text))
		want = append(want, text)
	}
	writeLines(t, f.claudeFile("sid-pg"), lines, true)
	f.d.transcripts.pageSize = 450 // a few records per page; boundaries fall inside records
	got, pages := f.pageAll(t, "/v1/terminals/t1/transcript")
	if pages < 3 {
		t.Fatalf("only %d pages", pages)
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("pages %d\n got %v\nwant %v", pages, got, want)
	}
}

func TestTranscriptRecordLargerThanPage(t *testing.T) {
	f := setupTranscripts(t)
	f.bindClaude(t, 4242, "sid-big")
	lines := []string{claudeUser("first"), claudeUser(strings.Repeat("H", 3000)), claudeUser("second"), claudeUser("third")}
	writeLines(t, f.claudeFile("sid-big"), lines, true)
	f.d.transcripts.pageSize = 400
	got, pages := f.pageAll(t, "/v1/terminals/t1/transcript")
	want := []string{"first", tooLargeText, "second", "third"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("pages %d: %v", pages, got)
	}
	// With the default page size the big record is shown whole.
	f.d.transcripts.pageSize = defaultPageSize
	got, _ = f.pageAll(t, "/v1/terminals/t1/transcript")
	if len(got) != 4 || got[1] != strings.Repeat("H", 3000) {
		t.Fatalf("%d items", len(got))
	}
}

func TestTranscriptStaleCursor(t *testing.T) {
	f := setupTranscripts(t)
	f.bindClaude(t, 4242, "sid-a")
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, claudeUser(fmt.Sprintf("a-%d", i)))
	}
	writeLines(t, f.claudeFile("sid-a"), lines, true)
	f.d.transcripts.pageSize = 300
	st, pg, _ := f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || pg.NextCursor == "" {
		t.Fatalf("%d %+v", st, pg)
	}
	cur := pg.NextCursor
	// The same file, read again with the cursor: fine.
	if st, _, _ = f.get(t, "/v1/terminals/t1/transcript?cursor="+cur); st != 200 {
		t.Fatalf("valid cursor: %d", st)
	}
	// Another session file: 409 cursor.
	f.bindClaude(t, 4242, "sid-b")
	writeLines(t, f.claudeFile("sid-b"), lines, true)
	st, _, raw := f.get(t, "/v1/terminals/t1/transcript?cursor="+cur)
	if st != 409 || raw["reason"] != "cursor" {
		t.Fatalf("other file: %d %v", st, raw)
	}
	// The same path replaced by a new file (new inode): 409. The old file moves
	// aside rather than going: Linux reuses a freed inode at once.
	f.bindClaude(t, 4242, "sid-a")
	os.Rename(f.claudeFile("sid-a"), f.claudeFile("sid-a")+".old")
	writeLines(t, f.claudeFile("sid-a"), lines, true)
	if st, _, raw = f.get(t, "/v1/terminals/t1/transcript?cursor="+cur); st != 409 || raw["reason"] != "cursor" {
		t.Fatalf("replaced file: %d %v", st, raw)
	}
	// A truncated file: the cursor points past the end.
	_, pg, _ = f.get(t, "/v1/terminals/t1/transcript")
	cur = pg.NextCursor
	writeLines(t, f.claudeFile("sid-a"), lines[:1], true)
	if st, _, raw = f.get(t, "/v1/terminals/t1/transcript?cursor="+cur); st != 409 || raw["reason"] != "cursor" {
		t.Fatalf("truncated file: %d %v", st, raw)
	}
	// Garbage cursors.
	for _, bad := range []string{"zzz", "e30", "bm90anNvbg"} {
		if st, _, _ = f.get(t, "/v1/terminals/t1/transcript?cursor="+bad); st != 409 {
			t.Fatalf("bad cursor %q: %d", bad, st)
		}
	}
	// A cursor while the source is terminal text: 409.
	os.Remove(filepath.Join(f.claude, "sessions", "4242.json"))
	if st, _, raw = f.get(t, "/v1/terminals/t1/transcript?cursor="+cur); st != 409 || raw["reason"] != "cursor" {
		t.Fatalf("terminal source with cursor: %d %v", st, raw)
	}
}

func TestTranscriptSymlinkOutsideRootsRefused(t *testing.T) {
	f := setupTranscripts(t)
	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	writeLines(t, outside, []string{claudeUser("leaked")}, true)
	f.bindClaude(t, 4242, "sid-ln")
	if err := os.Symlink(outside, f.claudeFile("sid-ln")); err != nil {
		t.Fatal(err)
	}
	st, pg, raw := f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || pg.Source != "terminal" || len(pg.Items) != 1 || pg.Items[0].Text != "screen of t1" {
		t.Fatalf("%d %+v", st, pg)
	}
	if _, has := raw["hint"]; has {
		t.Fatal("codex hint on a claude agent")
	}
	// Codex: an open rollout whose real path is outside ~/.codex/sessions is ignored.
	link := filepath.Join(f.codex, "sessions", "2026", "11", "08", "rollout-link.jsonl")
	target := f.rollout("real.jsonl", "", []string{codexMsg("user", "x")})
	moved := filepath.Join(t.TempDir(), "rollout-moved.jsonl")
	os.Rename(target, moved)
	os.Symlink(moved, link)
	f.bindCodex(t, 4344, link)
	st, pg, raw = f.get(t, "/v1/terminals/t2/transcript")
	if st != 200 || pg.Source != "terminal" || raw["hint"] != codexHint {
		t.Fatalf("%d %+v %v", st, pg, raw)
	}
	if strings.Contains(fmt.Sprint(pg.Items), "leaked") {
		t.Fatal("outside file read")
	}
}

func TestTranscriptDaemonModeCodexGetsTerminalTextAndHint(t *testing.T) {
	f := setupTranscripts(t)
	// A plain codex: the terminal's node process has no codex child of its own.
	f.d.f.write("ps.txt", "4343 1 node\n4399 1 codex\n")
	f.d.f.write("lsof-4399.txt", "p4399\nf5\nn"+f.rollout("rollout-daemon.jsonl", "", []string{codexMsg("user", "someone else")})+"\n")
	st, pg, raw := f.get(t, "/v1/terminals/t2/transcript")
	if st != 200 || pg.Source != "terminal" || raw["hint"] != codexHint || pg.Items[0].Text != "screen of t2" || pg.NextCursor != "" {
		t.Fatalf("%d %+v %v", st, pg, raw)
	}
	if strings.Contains(fmt.Sprint(pg.Items), "someone else") {
		t.Fatal("another client's rollout shown")
	}
	reads := f.d.f.requestsFor("terminal.read")
	if len(reads) == 0 || reads[len(reads)-1].Params["lines"] != 2000.0 || reads[len(reads)-1].Params["terminal"] != "t2" {
		t.Fatalf("read params %v", reads)
	}
}

func TestTranscriptNoSessionsFileFallsBack(t *testing.T) {
	f := setupTranscripts(t)
	st, pg, raw := f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || pg.Source != "terminal" || pg.Items[0].Role != "terminal" || pg.Items[0].Text != "screen of t1" {
		t.Fatalf("%d %+v", st, pg)
	}
	if _, has := raw["hint"]; has {
		t.Fatal("hint for a claude agent")
	}
	// A sessions file that points at a session with no projects file: terminal text too.
	f.bindClaude(t, 4242, "sid-missing")
	if st, pg, _ = f.get(t, "/v1/terminals/t1/transcript"); st != 200 || pg.Source != "terminal" {
		t.Fatalf("%d %+v", st, pg)
	}
	// Two project files with the same session id are ambiguous: terminal text.
	os.MkdirAll(filepath.Join(f.claude, "projects", "-y"), 0o700)
	writeLines(t, f.claudeFile("sid-missing"), []string{claudeUser("x")}, true)
	writeLines(t, filepath.Join(f.claude, "projects", "-y", "sid-missing.jsonl"), []string{claudeUser("y")}, true)
	if st, pg, _ = f.get(t, "/v1/terminals/t1/transcript"); st != 200 || pg.Source != "terminal" {
		t.Fatalf("ambiguous: %d %+v", st, pg)
	}
	if st, _, _ = f.get(t, "/v1/terminals/t9/transcript"); st != 404 {
		t.Fatalf("missing terminal: %d", st)
	}
}

// A file that cannot be read, or one with no record that parses, falls back to
// terminal text. The first request pages back past records that give no item;
// a later page with no item keeps its cursor.
func TestTranscriptFallsBackAndPagesPastEmptyPages(t *testing.T) {
	f := setupTranscripts(t)
	f.bindClaude(t, 4242, "sid-e")
	path := f.claudeFile("sid-e")
	att := `{"type":"attachment","attachment":{"x":1}}`
	// Only metadata records: terminal text, no cursor.
	writeLines(t, path, []string{att, att}, true)
	st, pg, _ := f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || pg.Source != "terminal" || len(pg.Items) != 1 || pg.Items[0].Text != "screen of t1" || pg.NextCursor != "" {
		t.Fatalf("metadata only: %d %+v", st, pg)
	}
	// An unreadable file: terminal text.
	writeLines(t, path, []string{claudeUser("hidden")}, true)
	os.Chmod(path, 0)
	st, pg, _ = f.get(t, "/v1/terminals/t1/transcript")
	os.Chmod(path, 0o600)
	if st != 200 || pg.Source != "terminal" || len(pg.Items) != 1 || pg.Items[0].Text != "screen of t1" {
		t.Fatalf("unreadable: %d %+v", st, pg)
	}
	// The last page holds only metadata: the first request pages back to an item.
	f.d.transcripts.pageSize = 120 // one record of 96 bytes, or two of 42, per page
	writeLines(t, path, []string{claudeUser("one"), att, att}, true)
	st, pg, _ = f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || pg.Source != "claude" || len(pg.Items) != 1 || pg.Items[0].Text != "one" || pg.NextCursor != "" {
		t.Fatalf("empty last page: %d %+v", st, pg)
	}
	// A later page with no item keeps its cursor.
	writeLines(t, path, []string{claudeUser("one"), att, att, claudeUser("two")}, true)
	st, pg, _ = f.get(t, "/v1/terminals/t1/transcript")
	if st != 200 || len(pg.Items) != 1 || pg.Items[0].Text != "two" || pg.NextCursor == "" {
		t.Fatalf("last page: %d %+v", st, pg)
	}
	st, pg, _ = f.get(t, "/v1/terminals/t1/transcript?cursor="+pg.NextCursor)
	if st != 200 || len(pg.Items) != 0 || pg.NextCursor == "" {
		t.Fatalf("middle page: %d %+v", st, pg)
	}
	cur := pg.NextCursor
	st, pg, _ = f.get(t, "/v1/terminals/t1/transcript?cursor="+cur)
	if st != 200 || len(pg.Items) != 1 || pg.Items[0].Text != "one" || pg.NextCursor != "" {
		t.Fatalf("first page: %d %+v", st, pg)
	}
	// A cursor into a file that cannot be read now: 409, so the phone starts again from the end.
	os.Chmod(path, 0)
	st, _, raw := f.get(t, "/v1/terminals/t1/transcript?cursor="+cur)
	os.Chmod(path, 0o600)
	if st != 409 || raw["reason"] != "cursor" {
		t.Fatalf("cursor into an unreadable file: %d %v", st, raw)
	}
}
