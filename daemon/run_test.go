package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunRefusesProgramOutsideList(t *testing.T) {
	r := &runner{progs: Programs{HC: "/bin/echo"}}
	if _, err := r.output(context.Background(), time.Second, program(42), "x"); err == nil {
		t.Fatal("unknown program ran")
	}
	if _, err := r.output(context.Background(), time.Second, progTailscale, "status"); err == nil {
		t.Fatal("absent program ran")
	}
}

func TestRunRefusesRelativePath(t *testing.T) {
	r := &runner{progs: Programs{HC: "echo"}}
	if _, err := r.hcConnect(context.Background()); err == nil {
		t.Fatal("relative path accepted")
	}
}

func TestRunBareEnvironment(t *testing.T) {
	t.Setenv("HC_TERMINAL_ID", "t1")
	t.Setenv("HC_TERMINAL_SOCKET", "/x")
	r := &runner{progs: Programs{HC: "/bin/echo"}}
	cmd, err := r.hcConnect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, "HC_") {
			t.Fatalf("inherited %s", kv)
		}
	}
	home, _ := os.UserHomeDir()
	if !contains(cmd.Env, "HOME="+home) || !contains(cmd.Env, "PATH=/usr/bin:/bin") {
		t.Fatalf("env %v", cmd.Env)
	}
}

func TestRunFixedShapes(t *testing.T) {
	f := newFake(t)
	ctx := testCtx(t)
	f.write("ps.txt", "1 0 launchd\n")
	f.write("lsof-42.txt", "p42\n")
	if _, err := f.r.ioregIdle(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.r.psTree(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.r.lsofFiles(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if _, err := f.r.lsofFiles(ctx, 0); err == nil {
		t.Fatal("pid 0 accepted")
	}
	if _, err := f.r.herdrAppLs(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.r.hcAttach(ctx, "box/t7"); err == nil {
		t.Fatal("machine target accepted by attach")
	}
	want := [][]string{{"-c", "IOHIDSystem", "-d", "4"}, {"-axo", "pid=,ppid=,comm="}, {"-Fn", "-p", "42"}, {"ls"}}
	got := f.calls()
	if len(got) != len(want) {
		t.Fatalf("calls %v", got)
	}
	for i := range want {
		if strings.Join(got[i], " ") != strings.Join(want[i], " ") {
			t.Fatalf("call %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
