package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	maxClaudeUsageBytes = 64 << 10
	maxCodexUsageBytes  = 1 << 20
)

type usageResponse struct {
	Claude *usageSource `json:"claude"`
	Codex  *usageSource `json:"codex"`
}

type usageSource struct {
	AsOf    int64         `json:"as_of"`
	Plan    string        `json:"plan,omitempty"`
	Windows []usageWindow `json:"windows"`
}

type usageWindow struct {
	Name        string  `json:"name"`
	UsedPercent float64 `json:"used_percent"`
	ResetsAt    int64   `json:"resets_at"`
}

type rawClaudeWindow struct {
	UsedPercent *float64 `json:"used_percentage"`
	ResetsAt    int64    `json:"resets_at"`
}

type rawCodexWindow struct {
	UsedPercent   *float64 `json:"used_percent"`
	WindowMinutes int      `json:"window_minutes"`
	ResetsAt      int64    `json:"resets_at"`
}

func (d *Daemon) handleUsage(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, usageResponse{
		Claude: readClaudeUsage(d.cfg.claudeUsagePath()),
		Codex:  readCodexUsage(filepath.Join(d.transcripts.codexDir, "sessions")),
	})
}

func (c config) claudeUsagePath() string {
	path := c.ClaudeUsageFile
	if path == "" {
		path = "~/.cache/herdr-slack/claude-usage.json"
	}
	rel, expand := strings.CutPrefix(path, "~/")
	if !expand {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, rel)
}

// openUsageFile rejects non-regular files before reading. A replaced file may
// be a pipe, so the open itself must not wait for a writer.
func openUsageFile(path string) (*os.File, os.FileInfo) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil
	}
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		f.Close()
		return nil, nil
	}
	return f, stat
}

func unixMillis(seconds int64) int64 {
	if seconds <= 0 || seconds > math.MaxInt64/1000 {
		return 0
	}
	return seconds * 1000
}

func (s *usageSource) addWindow(name string, used *float64, resetSeconds int64) {
	reset := unixMillis(resetSeconds)
	if used != nil && *used >= 0 && reset != 0 {
		s.Windows = append(s.Windows, usageWindow{Name: name, UsedPercent: *used, ResetsAt: reset})
	}
}

func readClaudeUsage(path string) *usageSource {
	f, stat := openUsageFile(path)
	if f == nil {
		return nil
	}
	defer f.Close()
	if stat.Size() > maxClaudeUsageBytes {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(f, maxClaudeUsageBytes))
	if err != nil {
		return nil
	}
	if stat, err := f.Stat(); err != nil || stat.Size() > maxClaudeUsageBytes {
		return nil
	}
	var reading struct {
		At         int64 `json:"at"`
		RateLimits struct {
			FiveHour rawClaudeWindow `json:"five_hour"`
			SevenDay rawClaudeWindow `json:"seven_day"`
		} `json:"rate_limits"`
	}
	if json.Unmarshal(data, &reading) != nil || unixMillis(reading.At) == 0 {
		return nil
	}
	source := usageSource{AsOf: unixMillis(reading.At)}
	source.addWindow("5h", reading.RateLimits.FiveHour.UsedPercent, reading.RateLimits.FiveHour.ResetsAt)
	source.addWindow("7d", reading.RateLimits.SevenDay.UsedPercent, reading.RateLimits.SevenDay.ResetsAt)
	if len(source.Windows) == 0 {
		return nil
	}
	return &source
}

func newestUsageRollout(root string) (string, os.FileInfo) {
	var latest string
	var newest os.FileInfo
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := strings.Count(rel, string(filepath.Separator))
		name := entry.Name()
		if entry.IsDir() {
			width := 2
			if depth == 0 {
				width = 4
			}
			if depth >= 3 || len(name) != width || strings.Trim(name, "0123456789") != "" {
				return filepath.SkipDir
			}
			return nil
		}
		if depth != 3 || !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		real, ok := underRoot(path, root)
		if !ok {
			return nil
		}
		info, err := os.Stat(real)
		if err == nil && info.Mode().IsRegular() && (newest == nil || info.ModTime().After(newest.ModTime())) {
			latest, newest = real, info
		}
		return nil
	})
	return latest, newest
}

func readCodexUsage(root string) *usageSource {
	path, chosen := newestUsageRollout(root)
	if chosen == nil {
		return nil
	}
	f, stat := openUsageFile(path)
	if f == nil {
		return nil
	}
	defer f.Close()
	// A rename or symlink change after selection must not switch the file we read.
	if !os.SameFile(stat, chosen) {
		return nil
	}
	start := max(0, stat.Size()-maxCodexUsageBytes)
	tail := make([]byte, stat.Size()-start)
	if _, err := f.ReadAt(tail, start); err != nil {
		return nil
	}
	if start > 0 {
		// The first record may be incomplete because the tail starts mid-line.
		_, rest, found := bytes.Cut(tail, []byte("\n"))
		if !found {
			return nil
		}
		tail = rest
	}
	lines := bytes.Split(tail, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if source, found := codexUsageEvent(lines[i]); found {
			return source
		}
	}
	return nil
}

// found distinguishes a rate limit record without any usable windows from an
// unrelated event. The latter must not hide an earlier token_count record.
func codexUsageEvent(line []byte) (source *usageSource, found bool) {
	var event struct {
		Type      string    `json:"type"`
		Timestamp time.Time `json:"timestamp"`
		Payload   struct {
			Type       string `json:"type"`
			RateLimits *struct {
				Primary   *rawCodexWindow `json:"primary"`
				Secondary *rawCodexWindow `json:"secondary"`
				PlanType  string          `json:"plan_type"`
			} `json:"rate_limits"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &event) != nil || event.Type != "event_msg" ||
		event.Payload.Type != "token_count" || event.Payload.RateLimits == nil {
		return nil, false
	}
	if event.Timestamp.UnixMilli() <= 0 {
		return nil, true
	}
	limits := event.Payload.RateLimits
	reading := usageSource{AsOf: event.Timestamp.UnixMilli(), Plan: limits.PlanType}
	for _, window := range []*rawCodexWindow{limits.Primary, limits.Secondary} {
		if window != nil && window.WindowMinutes > 0 {
			reading.addWindow(usageWindowName(window.WindowMinutes), window.UsedPercent, window.ResetsAt)
		}
	}
	if len(reading.Windows) == 0 {
		return nil, true
	}
	return &reading, true
}

func usageWindowName(minutes int) string {
	if minutes%1440 == 0 {
		return strconv.Itoa(minutes/1440) + "d"
	}
	if minutes%60 == 0 {
		return strconv.Itoa(minutes/60) + "h"
	}
	return strconv.Itoa(minutes) + "m"
}
