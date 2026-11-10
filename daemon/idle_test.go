package main

import (
	"testing"
	"time"
)

func TestParseHIDIdle(t *testing.T) {
	out := []byte("+-o IOHIDSystem  <class IOHIDSystem, id 0x100000456>\n    | |     \"HIDIdleTime\" = 90006570458\n    | |     \"HIDParameters\" = {}\n")
	d, err := parseHIDIdle(out)
	if err != nil || d != 90006570458*time.Nanosecond {
		t.Fatalf("got %v %v", d, err)
	}
	two := []byte("\"HIDIdleTime\" = 5000000000\n\"HIDIdleTime\" = 1000000000\n")
	if d, _ := parseHIDIdle(two); d != time.Second {
		t.Fatalf("min not taken: %v", d)
	}
	if _, err := parseHIDIdle([]byte("nothing here")); err == nil {
		t.Fatal("missing value parsed")
	}
	if _, err := parseHIDIdle([]byte("\"HIDIdleTime\" = 99999999999999999999999")); err == nil {
		t.Fatal("overflow parsed")
	}
}

func TestMacIdleFailsClosed(t *testing.T) {
	f := newFake(t)
	ctx := testCtx(t)
	d, err := f.r.MacIdle(ctx)
	if err != nil || d < 90*time.Second || d > 91*time.Second {
		t.Fatalf("got %v %v", d, err)
	}
	f.remove("ioreg.txt")
	if _, err := f.r.MacIdle(ctx); err == nil {
		t.Fatal("ioreg failure did not error")
	}
}
