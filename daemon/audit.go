package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

const rotateSize = 10 << 20

// auditLine is one record. Only the fields that apply are set. It never
// carries a token, a code, a key, transcript text, terminal bytes or message text.
type auditLine struct {
	TS         string `json:"ts"`
	DeviceID   string `json:"device_id,omitempty"`
	Action     string `json:"action"`
	State      string `json:"state,omitempty"`
	Terminal   string `json:"terminal,omitempty"`
	Turn       *int64 `json:"turn,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	Decision   string `json:"decision,omitempty"`
	PromptHash string `json:"prompt_hash,omitempty"`
	Length     *int   `json:"length,omitempty"`
	IdleMS     *int64 `json:"idle_ms,omitempty"`
	Result     string `json:"result,omitempty"`
}

type auditLog struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

func openAudit(path string) (*auditLog, error) {
	a := &auditLog{path: path}
	return a, a.open()
}

func (a *auditLog) open() error {
	f, err := os.OpenFile(a.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	a.f, a.size = f, st.Size()
	return nil
}

// write appends one line and fsyncs it. An error means nothing durable was written.
func (a *auditLog) write(l auditLine) error {
	l.TS = time.Now().UTC().Format(time.RFC3339Nano)
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.size+int64(len(b)) > rotateSize {
		a.f.Close()
		if err := os.Rename(a.path, a.path+".1"); err != nil {
			return err
		}
		if err := a.open(); err != nil {
			return err
		}
	}
	n, err := a.f.Write(b)
	if err != nil {
		return err
	}
	a.size += int64(n)
	return a.f.Sync()
}

// recent returns the lines of audit.log.1 and audit.log whose ts is after since.
func (a *auditLog) recent(since time.Time) []auditLine {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []auditLine
	for _, p := range []string{a.path + ".1", a.path} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range splitLines(b) {
			var l auditLine
			if json.Unmarshal(line, &l) != nil {
				continue
			}
			ts, err := time.Parse(time.RFC3339Nano, l.TS)
			if err != nil || ts.Before(since) {
				continue
			}
			out = append(out, l)
		}
	}
	return out
}

// rotatingWriter is daemon.log: plain lines, rotated at 10 MiB.
type rotatingWriter struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

func openRotating(path string) (*rotatingWriter, error) {
	w := &rotatingWriter{path: path}
	return w, w.open()
}

func (w *rotatingWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	w.f, w.size = f, st.Size()
	return nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size+int64(len(p)) > rotateSize {
		w.f.Close()
		os.Rename(w.path, w.path+".1")
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	n, err := w.f.Write(p)
	w.size += int64(n)
	return n, err
}

func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			if i > start {
				out = append(out, b[start:i])
			}
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}
