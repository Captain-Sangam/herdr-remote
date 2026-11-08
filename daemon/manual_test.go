package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestManualServe is the harness for the manual checks: with HR_MANUAL_DIR set it
// runs a daemon against the real hc and herdr-app on ports 17766/17767, pairs a
// generated phone key, writes client.crt, client.key and port into that
// directory, and serves until a file named stop appears there. HR_MANUAL_PAGE
// sets the transcript page size in bytes; HR_MANUAL_AUTOALLOW=1 allows every
// pending phone, for the emulator tests.
//
//	HR_MANUAL_DIR=/tmp/hr go test -run TestManualServe -count=1 -timeout 3h .
//	curl --cert $D/client.crt --key $D/client.key -k https://127.0.0.1:17766/v1/agents
func TestManualServe(t *testing.T) {
	out := os.Getenv("HR_MANUAL_DIR")
	if out == "" {
		t.Skip("set HR_MANUAL_DIR to run the manual harness")
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(out, "stop"))
	r := &runner{progs: findPrograms()}
	d, err := newDaemon(filepath.Join(out, "state"), r, "127.0.0.1:17766", "127.0.0.1:17767")
	if err != nil {
		t.Fatal(err)
	}
	// A short path (unix sockets are limited to 104 bytes) that the CLI reaches with
	// HERDR_REMOTE_STATE_DIR=$TMPDIR/hr-manual.
	adminDir := filepath.Join(os.TempDir(), "hr-manual")
	os.MkdirAll(adminDir, 0o700)
	d.adminPath = filepath.Join(adminDir, "admin.sock")
	if n, err := strconv.ParseInt(os.Getenv("HR_MANUAL_PAGE"), 10, 64); err == nil && n > 0 {
		d.transcripts.pageSize = n // small pages make the three-page check possible on a short transcript
	}
	if err := d.start(); err != nil {
		t.Fatal(err)
	}
	defer d.stop()
	td := &testDaemon{Daemon: d}
	cert, dev := td.pairPhone(t, "manual phone")
	keyDER, err := x509.MarshalECPrivateKey(cert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(out, "client.key"), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)
	os.WriteFile(filepath.Join(out, "client.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o600)
	os.WriteFile(filepath.Join(out, "device_id"), []byte(dev.ID), 0o600)
	os.WriteFile(filepath.Join(out, "port"), []byte(fmt.Sprint(d.pairedPort)), 0o600)
	fmt.Printf("manual daemon on %d, admin %s, device %s\n", d.pairedPort, d.adminPath, dev.ID)
	if os.Getenv("HR_MANUAL_AUTOALLOW") == "1" {
		// The emulator cannot click Allow on the Mac: this scratch daemon allows any pending phone.
		go func() {
			for {
				time.Sleep(500 * time.Millisecond)
				if d.pair.status().State == pairPending {
					d.pair.allow()
				}
			}
		}()
	}
	for {
		if _, err := os.Stat(filepath.Join(out, "stop")); err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}
