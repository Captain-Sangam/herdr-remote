package main

import (
	"context"
	"net/http"
	"os/exec"
	"sync"
)

// liveStream is one `hc attach --observe` subprocess copied to one phone.
type liveStream struct {
	device   string
	terminal string
	cmd      *exec.Cmd
	cancel   context.CancelFunc
}

type liveManager struct {
	d       *Daemon
	mu      sync.Mutex
	streams map[*liveStream]struct{}
	perDev  int
}

func newLiveManager(d *Daemon) *liveManager {
	return &liveManager{d: d, streams: map[*liveStream]struct{}{}, perDev: 2}
}

func (m *liveManager) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.streams)
}

func (m *liveManager) pids() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []int
	for s := range m.streams {
		if s.cmd.Process != nil {
			out = append(out, s.cmd.Process.Pid)
		}
	}
	return out
}

func (m *liveManager) endWhere(match func(*liveStream) bool) {
	m.mu.Lock()
	var ending []*liveStream
	for s := range m.streams {
		if match(s) {
			ending = append(ending, s)
		}
	}
	m.mu.Unlock()
	for _, s := range ending {
		s.cancel()
	}
}

// endTerminal ends the streams of a resized terminal; the phone reopens for a fresh paint.
func (m *liveManager) endTerminal(t string) {
	m.endWhere(func(s *liveStream) bool { return s.terminal == t })
}

// endDevice ends every stream of a revoked device.
func (m *liveManager) endDevice(dev string) {
	m.endWhere(func(s *liveStream) bool { return s.device == dev })
}

func (m *liveManager) add(s *liveStream) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for x := range m.streams {
		if x.device == s.device {
			n++
		}
	}
	if n >= m.perDev {
		return false
	}
	m.streams[s] = struct{}{}
	return true
}

func (m *liveManager) remove(s *liveStream) {
	m.mu.Lock()
	delete(m.streams, s)
	m.mu.Unlock()
}

// handleLive copies the attach stream to the phone until the phone leaves,
// the terminal is resized, the device is revoked or the subprocess ends.
func (d *Daemon) handleLive(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		apiError(w, 500, "no streaming", "")
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	cmd, err := d.r.hcAttach(ctx, term.Address)
	if err != nil {
		apiError(w, 400, "bad target", "")
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		apiError(w, 503, "attach failed", "")
		return
	}
	s := &liveStream{device: deviceOf(r).ID, terminal: term.Address, cmd: cmd, cancel: cancel}
	if !d.live.add(s) {
		apiError(w, 409, "too many live views", "live_limit")
		return
	}
	defer d.live.remove(s)
	if err := cmd.Start(); err != nil {
		apiError(w, 503, "attach failed", "")
		return
	}
	defer cmd.Wait()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(200)
	fl.Flush()
	buf := make([]byte, 32<<10)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			fl.Flush()
		}
		if err != nil {
			return
		}
	}
}
