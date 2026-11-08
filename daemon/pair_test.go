package main

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testDaemon struct {
	*Daemon
	f *fake
}

func startTestDaemon(t *testing.T) *testDaemon {
	t.Helper()
	f := newFake(t)
	dir := filepath.Join(t.TempDir(), "state")
	d, err := newDaemon(dir, f.r, "127.0.0.1:0", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// Unix socket paths are limited to 104 bytes on macOS; t.TempDir is longer.
	sock, err := os.MkdirTemp("", "hr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(sock) })
	d.adminPath = filepath.Join(sock, "admin.sock")
	if err := d.start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.stop)
	return &testDaemon{d, f}
}

func selfSigned(t *testing.T, key crypto.Signer) tls.Certificate {
	t.Helper()
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "phone"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(der)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
}

func clientCert(t *testing.T) (tls.Certificate, [32]byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	c := selfSigned(t, key)
	return c, spkiHash(c.Leaf)
}

func rsaClientCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return selfSigned(t, key)
}

func clientTLS(cert *tls.Certificate, pin [32]byte, cache tls.ClientSessionCache) *tls.Config {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
		ClientSessionCache: cache,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 || spkiHash(cs.PeerCertificates[0]) != pin {
				return errors.New("pin mismatch")
			}
			return nil
		},
	}
	if cert != nil {
		cfg.Certificates = []tls.Certificate{*cert}
	}
	return cfg
}

func (d *testDaemon) serverPin() [32]byte { return spkiHash(d.cert.Leaf) }

func (d *testDaemon) client(cert *tls.Certificate, cache tls.ClientSessionCache) *http.Client {
	return &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS(cert, d.serverPin(), cache)}, Timeout: 130 * time.Second}
}

func (d *testDaemon) pairedURL(path string) string {
	return fmt.Sprintf("https://127.0.0.1:%d%s", d.pairedPort, path)
}

type qrFields struct {
	pin, token string
	eps        []string
	port       int
	enrollPort int
	exp        int64
}

var qrRe = regexp.MustCompile(`^herdr-remote://pair\?v=1&pin=([A-Za-z0-9_-]+)&ep=([^&]*)&p=(\d+)&ep_port=(\d+)&t=([A-Za-z0-9_-]+)&exp=(\d+)$`)

func (d *testDaemon) openPair(t *testing.T) qrFields {
	t.Helper()
	res, err := d.pair.open()
	if err != nil {
		t.Fatal(err)
	}
	rows := res["matrix"].([]string)
	if len(rows) < 21 || len(rows[0]) != len(rows) {
		t.Fatalf("matrix %d x %d", len(rows), len(rows[0]))
	}
	m := qrRe.FindStringSubmatch(d.pair.qrText())
	if m == nil {
		t.Fatalf("qr text %q", d.pair.qrText())
	}
	port, _ := strconv.Atoi(m[3])
	ep, _ := strconv.Atoi(m[4])
	exp, _ := strconv.ParseInt(m[6], 10, 64)
	if exp != res["expires_at"].(int64) {
		t.Fatal("expiry mismatch")
	}
	return qrFields{pin: m[1], eps: strings.Split(m[2], ","), port: port, enrollPort: ep, token: m[5], exp: exp}
}

type enrollResult struct {
	status int
	body   map[string]any
	err    error
}

func (d *testDaemon) enroll(cert *tls.Certificate, enrollPort int, token, name string) enrollResult {
	body, _ := json.Marshal(map[string]string{"token": token, "name": name})
	resp, err := d.client(cert, nil).Post(fmt.Sprintf("https://127.0.0.1:%d/v1/enroll", enrollPort), "application/json", bytes.NewReader(body))
	if err != nil {
		return enrollResult{err: err}
	}
	defer resp.Body.Close()
	var m map[string]any
	json.NewDecoder(resp.Body).Decode(&m)
	return enrollResult{status: resp.StatusCode, body: m}
}

func (d *testDaemon) enrollAsync(cert *tls.Certificate, enrollPort int, token, name string) chan enrollResult {
	ch := make(chan enrollResult, 1)
	go func() { ch <- d.enroll(cert, enrollPort, token, name) }()
	return ch
}

func (d *testDaemon) waitState(t *testing.T, want pairState) pairStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		st := d.pair.status()
		if st.State == want {
			return st
		}
		if time.Now().After(deadline) {
			t.Fatalf("pair state %s, want %s", st.State, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func portOpen(port int) bool {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func waitClosed(t *testing.T, port int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if !portOpen(port) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("port %d still open", port)
}

// pairPhone runs a full pairing with Allow and returns the phone's certificate.
func (d *testDaemon) pairPhone(t *testing.T, name string) (tls.Certificate, Device) {
	t.Helper()
	cert, _ := clientCert(t)
	q := d.openPair(t)
	res := d.enrollAsync(&cert, q.enrollPort, q.token, name)
	d.waitState(t, pairPending)
	if err := d.pair.allow(); err != nil {
		t.Fatal(err)
	}
	r := <-res
	if r.err != nil || r.status != 200 {
		t.Fatalf("enroll: %+v", r)
	}
	dev := d.devices.byID(r.body["device_id"].(string))
	if dev == nil {
		t.Fatal("device not stored")
	}
	return cert, *dev
}

func TestPairAllowFlow(t *testing.T) {
	d := startTestDaemon(t)
	cert, h := clientCert(t)
	q := d.openPair(t)
	if q.pin != d.pin() || q.pin != b64url(func() []byte { s := sha256.Sum256(d.cert.Leaf.RawSubjectPublicKeyInfo); return s[:] }()) {
		t.Fatalf("pin %s", q.pin)
	}
	if q.port != d.pairedPort || q.exp < time.Now().Unix()+100 || len(q.eps) == 0 {
		t.Fatalf("fields %+v", q)
	}
	if st := d.pair.status(); st.State != pairOpen || st.Code != "" {
		t.Fatalf("status %+v", st)
	}
	res := d.enrollAsync(&cert, q.enrollPort, q.token, "Pixel 9")
	st := d.waitState(t, pairPending)
	want := pairCode(d.cert.Leaf.RawSubjectPublicKeyInfo, cert.Leaf.RawSubjectPublicKeyInfo)
	if st.Code != want || len(st.Code) != 6 || st.DeviceName != "Pixel 9" {
		t.Fatalf("status %+v want code %s", st, want)
	}
	select {
	case r := <-res:
		t.Fatalf("enroll returned before Allow: %+v", r)
	case <-time.After(300 * time.Millisecond):
	}
	if err := d.pair.allow(); err != nil {
		t.Fatal(err)
	}
	r := <-res
	if r.err != nil || r.status != 200 || r.body["device_id"] == "" {
		t.Fatalf("enroll result %+v", r)
	}
	dev := d.devices.bySPKI(h)
	if dev == nil || dev.Name != "Pixel 9" || dev.ID != r.body["device_id"] {
		t.Fatalf("device %+v", dev)
	}
	if st := d.pair.status(); st.State != pairAllowed {
		t.Fatalf("status after allow %+v", st)
	}
	waitClosed(t, q.enrollPort)
	// devices.json is 0600 and holds the device.
	fi, _ := os.Stat(filepath.Join(d.dir, "devices.json"))
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("devices.json mode %v", fi.Mode())
	}
	// The paired phone now passes the handshake on the paired port.
	resp, err := d.client(&cert, nil).Get(d.pairedURL("/nothing"))
	if err != nil {
		t.Fatalf("paired handshake: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == 401 {
		t.Fatal("paired device got 401")
	}
}

func TestPairAllowAfterSixtySeconds(t *testing.T) {
	if testing.Short() {
		t.Skip("60 s long-poll")
	}
	d := startTestDaemon(t)
	cert, _ := clientCert(t)
	q := d.openPair(t)
	res := d.enrollAsync(&cert, q.enrollPort, q.token, "slow")
	d.waitState(t, pairPending)
	time.Sleep(60 * time.Second)
	if err := d.pair.allow(); err != nil {
		t.Fatal(err)
	}
	r := <-res
	if r.err != nil || r.status != 200 {
		t.Fatalf("%+v", r)
	}
}

func TestPairDenyCancelExpiryStoreNothing(t *testing.T) {
	d := startTestDaemon(t)
	cert, _ := clientCert(t)

	q := d.openPair(t)
	res := d.enrollAsync(&cert, q.enrollPort, q.token, "deny me")
	d.waitState(t, pairPending)
	if err := d.pair.deny(); err != nil {
		t.Fatal(err)
	}
	if r := <-res; r.status != 403 {
		t.Fatalf("deny: %+v", r)
	}
	if st := d.pair.status(); st.State != pairDenied {
		t.Fatalf("%+v", st)
	}
	waitClosed(t, q.enrollPort)

	q = d.openPair(t)
	res = d.enrollAsync(&cert, q.enrollPort, q.token, "cancel me")
	d.waitState(t, pairPending)
	d.pair.cancel()
	if r := <-res; r.status != 410 {
		t.Fatalf("cancel: %+v", r)
	}
	if st := d.pair.status(); st.State != pairNone {
		t.Fatalf("%+v", st)
	}
	waitClosed(t, q.enrollPort)

	d.pair.window = 700 * time.Millisecond
	q = d.openPair(t)
	res = d.enrollAsync(&cert, q.enrollPort, q.token, "expire me")
	d.waitState(t, pairPending)
	if r := <-res; r.status != 410 {
		t.Fatalf("expiry: %+v", r)
	}
	d.waitState(t, pairExpired)
	waitClosed(t, q.enrollPort)

	// An open window with no scan also expires and closes the port.
	q = d.openPair(t)
	d.waitState(t, pairExpired)
	waitClosed(t, q.enrollPort)

	if n := len(d.devices.all()); n != 0 {
		t.Fatalf("%d devices stored", n)
	}
	if _, err := os.Stat(filepath.Join(d.dir, "devices.json")); err == nil {
		t.Fatal("devices.json written without Allow")
	}
}

func TestPairSecondKeyRefusedSameKeyRetries(t *testing.T) {
	d := startTestDaemon(t)
	a, _ := clientCert(t)
	b, _ := clientCert(t)
	q := d.openPair(t)
	first := d.enrollAsync(&a, q.enrollPort, q.token, "phone A")
	d.waitState(t, pairPending)
	if r := d.enroll(&b, q.enrollPort, q.token, "phone B"); r.status != 403 {
		t.Fatalf("second key: %+v", r)
	}
	if st := d.pair.status(); st.DeviceName != "phone A" {
		t.Fatalf("%+v", st)
	}
	retry := d.enrollAsync(&a, q.enrollPort, q.token, "phone A again")
	time.Sleep(200 * time.Millisecond)
	if err := d.pair.allow(); err != nil {
		t.Fatal(err)
	}
	r1, r2 := <-first, <-retry
	if r1.status != 200 || r2.status != 200 || r1.body["device_id"] != r2.body["device_id"] {
		t.Fatalf("%+v %+v", r1, r2)
	}
	if n := len(d.devices.all()); n != 1 {
		t.Fatalf("%d devices", n)
	}
}

func TestPairBadTokenAndName(t *testing.T) {
	d := startTestDaemon(t)
	cert, _ := clientCert(t)
	q := d.openPair(t)
	wrong := b64url(bytes.Repeat([]byte{7}, 32))
	if r := d.enroll(&cert, q.enrollPort, wrong, "x"); r.status != 403 {
		t.Fatalf("wrong token: %+v", r)
	}
	if r := d.enroll(&cert, q.enrollPort, "", "x"); r.status != 403 {
		t.Fatalf("empty token: %+v", r)
	}
	for _, name := range []string{"", strings.Repeat("a", 65), "bad\x01name", "tab\tname", "nl\nname"} {
		if r := d.enroll(&cert, q.enrollPort, q.token, name); r.status != 400 {
			t.Fatalf("name %q: %+v", name, r)
		}
	}
	if st := d.pair.status(); st.State != pairOpen {
		t.Fatalf("window moved to %s", st.State)
	}
	if n := len(d.devices.all()); n != 0 {
		t.Fatal("device stored")
	}
	// A 64-rune name with spaces and unicode is fine.
	ok := d.enrollAsync(&cert, q.enrollPort, q.token, strings.Repeat("é ", 32))
	d.waitState(t, pairPending)
	d.pair.cancel()
	<-ok
}

func TestEnrollPortClosedOutsideWindow(t *testing.T) {
	d := startTestDaemon(t)
	q := d.openPair(t)
	if !portOpen(q.enrollPort) {
		t.Fatal("enroll port closed during window")
	}
	d.pair.cancel()
	waitClosed(t, q.enrollPort)
	if portOpen(q.enrollPort) {
		t.Fatal("enroll port open after cancel")
	}
}

func TestHandshakeRequiresPairedP256Key(t *testing.T) {
	d := startTestDaemon(t)
	paired, _ := d.pairPhone(t, "ok")
	unpaired, _ := clientCert(t)
	rsaCert := rsaClientCert(t)
	cases := []struct {
		name string
		cert *tls.Certificate
		ok   bool
	}{{"paired", &paired, true}, {"unpaired", &unpaired, false}, {"no certificate", nil, false}, {"rsa", &rsaCert, false}}
	for _, c := range cases {
		resp, err := d.client(c.cert, nil).Get(d.pairedURL("/nothing"))
		if c.ok && err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !c.ok && err == nil {
			resp.Body.Close()
			t.Fatalf("%s: handshake succeeded", c.name)
		}
		if err == nil {
			resp.Body.Close()
		}
	}
	// A wrong pin on the phone side never connects either.
	bad := clientTLS(&paired, [32]byte{1}, nil)
	if _, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: bad}}).Get(d.pairedURL("/nothing")); err == nil {
		t.Fatal("wrong pin connected")
	}
}

func TestRevokeClosesConnectionsAndBlocksResumption(t *testing.T) {
	d := startTestDaemon(t)
	cert, dev := d.pairPhone(t, "lost phone")
	cache := tls.NewLRUClientSessionCache(8)
	client := d.client(&cert, cache)
	resp, err := client.Get(d.pairedURL("/nothing"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	var h [32]byte
	raw, _ := base64.RawURLEncoding.DecodeString(dev.SPKI)
	copy(h[:], raw)
	d.conns.mu.Lock()
	open := len(d.conns.byKey[h])
	d.conns.mu.Unlock()
	if open == 0 {
		t.Fatal("connection not registered")
	}
	if err := d.revoke(dev.ID); err != nil {
		t.Fatal(err)
	}
	d.conns.mu.Lock()
	open = len(d.conns.byKey[h])
	d.conns.mu.Unlock()
	if open != 0 {
		t.Fatalf("%d connections still registered", open)
	}
	if _, err := client.Get(d.pairedURL("/nothing")); err == nil {
		t.Fatal("revoked phone reconnected")
	}
	if _, err := d.client(&cert, cache).Get(d.pairedURL("/nothing")); err == nil {
		t.Fatal("revoked phone connected with a fresh client and its session cache")
	}
	if err := d.revoke(dev.ID); err == nil {
		t.Fatal("second revoke succeeded")
	}
}

func TestAuditHasNoSecrets(t *testing.T) {
	d := startTestDaemon(t)
	cert, _ := clientCert(t)
	q := d.openPair(t)
	res := d.enrollAsync(&cert, q.enrollPort, q.token, "audited")
	st := d.waitState(t, pairPending)
	d.pair.allow()
	<-res
	b, err := os.ReadFile(filepath.Join(d.dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, q.token) || strings.Contains(s, st.Code) || strings.Contains(s, q.pin) {
		t.Fatalf("audit leaks a secret: %s", s)
	}
	for _, want := range []string{`"action":"enroll"`, `"action":"allow"`, `"device_id":"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("audit lacks %s: %s", want, s)
		}
	}
	fi, _ := os.Stat(filepath.Join(d.dir, "audit.log"))
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("audit.log mode %v", fi.Mode())
	}
	for _, f := range []string{filepath.Join(d.dir, "server.key"), d.adminPath} {
		fi, err := os.Stat(f)
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("%s: %v %v", f, fi, err)
		}
	}
}

func TestAdminSocketCommands(t *testing.T) {
	d := startTestDaemon(t)
	call := func(cmd, id string) json.RawMessage {
		t.Helper()
		res, err := adminCall(d.adminPath, adminReq{Cmd: cmd, ID: id})
		if err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
		return res
	}
	if s := string(call("pair_status", "")); s != `{"state":"none"}` {
		t.Fatal(s)
	}
	if s := string(call("devices_ls", "")); s != `[]` {
		t.Fatal(s)
	}
	var res map[string]any
	json.Unmarshal(call("pair", ""), &res)
	if _, ok := res["matrix"].([]any); !ok || res["expires_at"] == nil {
		t.Fatalf("pair result %v", res)
	}
	if _, err := adminCall(d.adminPath, adminReq{Cmd: "pair_allow"}); err == nil {
		t.Fatal("allow without a pending phone succeeded")
	}
	call("pair_cancel", "")
	_, dev := d.pairPhone(t, "admin phone")
	var list []map[string]string
	json.Unmarshal(call("devices_ls", ""), &list)
	if len(list) != 1 || list[0]["id"] != dev.ID || list[0]["name"] != "admin phone" || list[0]["added"] == "" {
		t.Fatalf("%v", list)
	}
	if _, has := list[0]["spki_sha256"]; has {
		t.Fatal("ls exposes the key hash")
	}
	call("devices_rm", dev.ID)
	if s := string(call("devices_ls", "")); s != `[]` {
		t.Fatal(s)
	}
	if _, err := adminCall(d.adminPath, adminReq{Cmd: "devices_rm", ID: dev.ID}); err == nil {
		t.Fatal("rm of a missing device succeeded")
	}
	if _, err := adminCall(d.adminPath, adminReq{Cmd: "bogus"}); err == nil {
		t.Fatal("unknown command succeeded")
	}
}

func TestPairCodeVectors(t *testing.T) {
	b, err := os.ReadFile("../testdata/pairing-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Vectors []struct {
			Server string `json:"server_spki_b64url"`
			Client string `json:"client_spki_b64url"`
			Code   string `json:"code"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Vectors) < 5 {
		t.Fatal("too few vectors")
	}
	for i, v := range f.Vectors {
		s, _ := base64.RawURLEncoding.DecodeString(v.Server)
		c, _ := base64.RawURLEncoding.DecodeString(v.Client)
		if got := pairCode(s, c); got != v.Code {
			t.Fatalf("vector %d: got %s want %s", i, got, v.Code)
		}
	}
}

func TestIdentityPersists(t *testing.T) {
	dir := t.TempDir()
	a, err := loadOrCreateIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := loadOrCreateIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spkiHash(a.Leaf) != spkiHash(b.Leaf) {
		t.Fatal("identity changed on reload")
	}
	if a.Leaf.Subject.CommonName != "herdr-remoted" || !isP256(a.Leaf) || a.Leaf.NotAfter.Before(time.Now().AddDate(9, 11, 0)) {
		t.Fatalf("certificate %+v", a.Leaf.Subject)
	}
	fi, _ := os.Stat(filepath.Join(dir, "server.key"))
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("server.key mode %v", fi.Mode())
	}
}
