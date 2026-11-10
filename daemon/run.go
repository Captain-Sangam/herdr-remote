package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// program is the closed set of programs the daemon may start.
type program int

const (
	progHC program = iota + 1
	progHerdrApp
	progLsof
	progPs
	progIoreg
	progTailscale
)

// Programs holds the absolute path of each program, found once at start.
// An empty path means the program is absent; only tailscale may be.
type Programs struct {
	HC, HerdrApp, Lsof, Ps, Ioreg, Tailscale string
}

func (p Programs) path(prog program) (string, error) {
	var s string
	switch prog {
	case progHC:
		s = p.HC
	case progHerdrApp:
		s = p.HerdrApp
	case progLsof:
		s = p.Lsof
	case progPs:
		s = p.Ps
	case progIoreg:
		s = p.Ioreg
	case progTailscale:
		s = p.Tailscale
	default:
		return "", fmt.Errorf("run: program %d is not allowed", prog)
	}
	if s == "" {
		return "", fmt.Errorf("run: program %d was not found at start", prog)
	}
	if !filepath.IsAbs(s) {
		return "", fmt.Errorf("run: %q is not an absolute path", s)
	}
	return s, nil
}

// findPrograms looks each program up once, at fixed places first, then on PATH.
func findPrograms() Programs {
	home, _ := os.UserHomeDir()
	find := func(name string, fixed ...string) string {
		for _, f := range fixed {
			if st, err := os.Stat(f); err == nil && !st.IsDir() {
				return f
			}
		}
		if p, err := exec.LookPath(name); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
		}
		return ""
	}
	return Programs{
		HC:        find("hc", filepath.Join(home, ".local/bin/hc")),
		HerdrApp:  find("herdr-app", filepath.Join(home, ".local/bin/herdr-app")),
		Lsof:      find("lsof", "/usr/sbin/lsof"),
		Ps:        find("ps", "/bin/ps"),
		Ioreg:     find("ioreg", "/usr/sbin/ioreg"),
		Tailscale: find("tailscale", "/opt/homebrew/bin/tailscale", "/usr/local/bin/tailscale", "/Applications/Tailscale.app/Contents/MacOS/Tailscale"),
	}
}

// runner is the only place that builds an exec.Cmd. Every program runs with a
// bare environment, so the daemon behaves the same under launchd and by hand,
// and never inherits HC_TERMINAL_ID (which would tag its sends).
type runner struct {
	progs    Programs
	extraEnv []string // tests only
}

func (r *runner) command(ctx context.Context, prog program, args ...string) (*exec.Cmd, error) {
	path, err := r.progs.path(prog)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, args...)
	home, _ := os.UserHomeDir()
	cmd.Env = append([]string{"HOME=" + home, "PATH=/usr/bin:/bin", "LANG=en_US.UTF-8"}, r.extraEnv...)
	return cmd, nil
}

// output runs a short program to completion and returns its stdout.
func (r *runner) output(ctx context.Context, timeout time.Duration, prog program, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd, err := r.command(ctx, prog, args...)
	if err != nil {
		return nil, err
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		// lsof exits 1 when some files could not be described but still prints the rest.
		if prog == progLsof && errors.As(err, &ee) && out.Len() > 0 {
			return out.Bytes(), nil
		}
		msg := errb.String()
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("%s: %w: %s", filepath.Base(cmd.Path), err, msg)
	}
	return out.Bytes(), nil
}

// The fixed argument shapes. Nothing else reaches command().

func (r *runner) herdrAppLs(ctx context.Context) ([]byte, error) {
	return r.output(ctx, 5*time.Second, progHerdrApp, "ls")
}

func (r *runner) lsofFiles(ctx context.Context, pid int) ([]byte, error) {
	if pid <= 0 {
		return nil, errors.New("lsof: bad pid")
	}
	return r.output(ctx, 10*time.Second, progLsof, "-Fn", "-p", strconv.Itoa(pid))
}

func (r *runner) psTree(ctx context.Context) ([]byte, error) {
	return r.output(ctx, 5*time.Second, progPs, "-axo", "pid=,ppid=,comm=")
}

func (r *runner) ioregIdle(ctx context.Context) ([]byte, error) {
	return r.output(ctx, 5*time.Second, progIoreg, "-c", "IOHIDSystem", "-d", "4")
}

func (r *runner) tailscaleStatus(ctx context.Context) ([]byte, error) {
	return r.output(ctx, 5*time.Second, progTailscale, "status", "--json")
}

// hcConnect starts a long-lived `hc connect`; the caller wires its pipes.
func (r *runner) hcConnect(ctx context.Context) (*exec.Cmd, error) {
	return r.command(ctx, progHC, "connect")
}

// hcAttach starts a watch-only live stream of one terminal.
func (r *runner) hcAttach(ctx context.Context, t string) (*exec.Cmd, error) {
	if !validTarget(t) {
		return nil, errBadTarget
	}
	return r.command(ctx, progHC, "attach", "--observe", "--history", "500", t)
}
