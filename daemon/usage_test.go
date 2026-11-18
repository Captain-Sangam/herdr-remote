package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const claudeUsageFixture = `{"at":1794997440,"rate_limits":{"five_hour":{"used_percentage":72.5,"resets_at":1795005000},"seven_day":{"used_percentage":62,"resets_at":1795410000}}}`

func TestClaudeUsage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.json")
	if got := readClaudeUsage(path); got != nil {
		t.Fatalf("missing file gave %+v", got)
	}
	writeLines(t, path, []string{claudeUsageFixture}, false)
	want := &usageSource{
		AsOf: 1794997440000,
		Windows: []usageWindow{
			{Name: "5h", UsedPercent: 72.5, ResetsAt: 1795005000000},
			{Name: "7d", UsedPercent: 62, ResetsAt: 1795410000000},
		},
	}
	if got := readClaudeUsage(path); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	writeLines(t, path, []string{"{"}, false)
	if got := readClaudeUsage(path); got != nil {
		t.Fatalf("bad JSON gave %+v", got)
	}
	writeLines(t, path, []string{claudeUsageFixture + strings.Repeat(" ", maxClaudeUsageBytes)}, false)
	if got := readClaudeUsage(path); got != nil {
		t.Fatalf("oversized file gave %+v", got)
	}
}

func TestClaudeUsagePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, want := (config{}).claudeUsagePath(), filepath.Join(home, ".cache", "herdr-slack", "claude-usage.json"); got != want {
		t.Fatalf("default %q, want %q", got, want)
	}
	if got, want := (config{ClaudeUsageFile: "~/quota.json"}).claudeUsagePath(), filepath.Join(home, "quota.json"); got != want {
		t.Fatalf("custom %q, want %q", got, want)
	}
}

func usageRolloutDir(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "2026", "11", "18")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func usageEvent(timestamp, limits string) string {
	return fmt.Sprintf(`{"type":"event_msg","timestamp":%q,"payload":{"type":"token_count","rate_limits":%s}}`, timestamp, limits)
}

const shortUsageLimits = `{"primary":{"used_percent":94,"window_minutes":300,"resets_at":1795005000},"secondary":{"used_percent":2,"window_minutes":10080,"resets_at":1795410000},"plan_type":"plus"}`

func TestCodexUsageNewestRolloutAndLastLimit(t *testing.T) {
	root := t.TempDir()
	dir := usageRolloutDir(t, root)
	writtenFirst := usageEvent("2026-11-18T10:20:00Z", `{"primary":{"used_percent":72,"window_minutes":300,"resets_at":1795005000}}`)
	writtenLast := usageEvent("2026-11-18T10:22:00.123Z", shortUsageLimits)
	older := filepath.Join(dir, "rollout-z.jsonl")
	writeLines(t, older, []string{writtenFirst}, true)
	olderTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(older, olderTime, olderTime); err != nil {
		t.Fatal(err)
	}
	writeLines(t, filepath.Join(dir, "rollout-a.jsonl"), []string{
		strings.Repeat(" ", maxCodexUsageBytes),
		writtenFirst,
		writtenLast,
		usageEvent("2026-11-18T10:23:00Z", "null"),
		`{"type":"event_msg","payload":{"type":"token_count"`,
	}, false)
	want := &usageSource{
		AsOf: time.Date(2026, 11, 18, 10, 22, 0, 123_000_000, time.UTC).UnixMilli(),
		Plan: "plus",
		Windows: []usageWindow{
			{Name: "5h", UsedPercent: 94, ResetsAt: 1795005000000},
			{Name: "7d", UsedPercent: 2, ResetsAt: 1795410000000},
		},
	}
	if got := readCodexUsage(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCodexUsageTailDoesNotReadEarlierRecords(t *testing.T) {
	root := t.TempDir()
	dir := usageRolloutDir(t, root)
	writeLines(t, filepath.Join(dir, "rollout-old-event.jsonl"), []string{
		usageEvent("2026-11-18T10:22:00Z", shortUsageLimits),
		strings.Repeat(" ", maxCodexUsageBytes),
	}, true)
	if got := readCodexUsage(root); got != nil {
		t.Fatalf("event outside tail gave %+v", got)
	}
}

func TestCodexUsageRefusesRolloutOutsideSessions(t *testing.T) {
	root := t.TempDir()
	dir := usageRolloutDir(t, root)
	if got := readCodexUsage(root); got != nil {
		t.Fatalf("empty sessions directory gave %+v", got)
	}
	outside := filepath.Join(t.TempDir(), "rollout-outside.jsonl")
	writeLines(t, outside, []string{usageEvent("2026-11-18T10:22:00Z", shortUsageLimits)}, true)
	if err := os.Symlink(outside, filepath.Join(dir, "rollout-link.jsonl")); err != nil {
		t.Fatal(err)
	}
	if got := readCodexUsage(root); got != nil {
		t.Fatalf("rollout outside sessions gave %+v", got)
	}
}

func TestUsageWindowName(t *testing.T) {
	for _, tc := range []struct {
		minutes int
		want    string
	}{{300, "5h"}, {10080, "7d"}, {120, "2h"}, {2880, "2d"}, {75, "75m"}} {
		if got := usageWindowName(tc.minutes); got != tc.want {
			t.Errorf("window of %d min is %q, want %q", tc.minutes, got, tc.want)
		}
	}
}

func TestUsageEndpoint(t *testing.T) {
	d := startTestDaemon(t)
	d.cfg.ClaudeUsageFile = filepath.Join(t.TempDir(), "usage.json")
	d.transcripts.codexDir = t.TempDir()
	writeLines(t, d.cfg.ClaudeUsageFile, []string{claudeUsageFixture}, false)
	url := fmt.Sprintf("https://127.0.0.1:%d/v1/usage", d.pairedPort)
	if resp, err := d.client(nil, nil).Get(url); err == nil {
		resp.Body.Close()
		t.Fatalf("usage without a device certificate returned %d", resp.StatusCode)
	}
	cert, _ := d.pairPhone(t, "usage-test")
	logPath := filepath.Join(d.dir, "audit.log")
	before, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	code, body := d.get(t, &cert, "/v1/usage")
	var response usageResponse
	if code != 200 || json.Unmarshal(body, &response) != nil ||
		response.Claude == nil || len(response.Claude.Windows) != 2 ||
		response.Codex != nil || !strings.Contains(string(body), `"codex":null`) {
		t.Fatalf("usage response %d %s", code, body)
	}
	after, err := os.Stat(logPath)
	if err != nil || after.Size() != before.Size() {
		t.Fatal("usage request changed the audit log")
	}
}
