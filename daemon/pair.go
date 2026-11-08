package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"
)

type pairState string

const (
	pairNone    pairState = "none"
	pairOpen    pairState = "open"
	pairPending pairState = "pending"
	pairAllowed pairState = "allowed"
	pairDenied  pairState = "denied"
	pairExpired pairState = "expired"
)

type pairStatus struct {
	State      pairState `json:"state"`
	DeviceName string    `json:"device_name,omitempty"`
	Code       string    `json:"code,omitempty"`
}

// pairWindow is one open pairing: a token, a 120 s timer and port 7767.
type pairWindow struct {
	token    []byte
	expires  time.Time
	state    pairState
	bound    *[32]byte // the first client SPKI that used the token
	name     string
	code     string
	deviceID string
	decided  chan struct{}
	ln       net.Listener
	srv      *http.Server
	timer    *time.Timer
}

type pairing struct {
	d      *Daemon
	mu     sync.Mutex
	win    *pairWindow
	last   pairStatus // what status reports once the window has closed
	window time.Duration
}

func newPairing(d *Daemon) *pairing {
	return &pairing{d: d, last: pairStatus{State: pairNone}, window: 120 * time.Second}
}

// open starts a window, replacing any open one, and returns the QR matrix and expiry.
func (p *pairing) open() (map[string]any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.win != nil {
		p.closeWindowLocked(pairNone)
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", p.d.enrollAddr)
	if err != nil {
		return nil, fmt.Errorf("enrollment port: %w", err)
	}
	w := &pairWindow{token: token, expires: time.Now().Add(p.window), state: pairOpen, decided: make(chan struct{}), ln: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/enroll", p.handleEnroll)
	w.srv = &http.Server{
		Handler:           mux,
		TLSConfig:         mutualTLS(p.d.cert, func(*x509.Certificate) error { return nil }),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go w.srv.ServeTLS(ln, "", "")
	w.timer = time.AfterFunc(p.window, func() { p.expire(w) })
	p.win = w
	matrix, err := qrMatrix(p.qrTextLocked(w))
	if err != nil {
		p.closeWindowLocked(pairNone)
		return nil, err
	}
	return map[string]any{"matrix": matrix, "expires_at": w.expires.Unix()}, nil
}

func (p *pairing) qrTextLocked(w *pairWindow) string {
	port := w.ln.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("herdr-remote://pair?v=1&pin=%s&ep=%s&p=%d&ep_port=%d&t=%s&exp=%d",
		p.d.pin(), strings.Join(p.d.endpoints(), ","), p.d.pairedPort, port, b64url(w.token), w.expires.Unix())
}

// qrText is the current window's QR content (tests).
func (p *pairing) qrText() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.win == nil {
		return ""
	}
	return p.qrTextLocked(p.win)
}

func qrMatrix(text string) ([]string, error) {
	q, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	q.DisableBorder = true
	bm := q.Bitmap()
	rows := make([]string, len(bm))
	for y, row := range bm {
		b := make([]byte, len(row))
		for x, dark := range row {
			if dark {
				b[x] = '1'
			} else {
				b[x] = '0'
			}
		}
		rows[y] = string(b)
	}
	return rows, nil
}

func (p *pairing) status() pairStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.win == nil {
		return p.last
	}
	s := pairStatus{State: p.win.state}
	if p.win.state == pairPending {
		s.DeviceName, s.Code = p.win.name, p.win.code
	}
	return s
}

var errNoPending = errors.New("no phone is waiting")

func (p *pairing) allow() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	w := p.win
	if w == nil || w.state != pairPending {
		return errNoPending
	}
	dev, err := p.d.devices.add(w.name, *w.bound)
	if err != nil {
		return err
	}
	w.deviceID = dev.ID
	p.d.audit.write(auditLine{Action: "allow", DeviceID: dev.ID})
	p.closeWindowLocked(pairAllowed)
	p.last.DeviceName = dev.Name
	return nil
}

func (p *pairing) deny() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	w := p.win
	if w == nil || w.state != pairPending {
		return errNoPending
	}
	p.d.audit.write(auditLine{Action: "deny"})
	p.closeWindowLocked(pairDenied)
	return nil
}

func (p *pairing) cancel() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.win != nil {
		p.closeWindowLocked(pairNone)
	}
	p.last = pairStatus{State: pairNone}
}

func (p *pairing) expire(w *pairWindow) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.win == w {
		p.closeWindowLocked(pairExpired)
	}
}

// closeWindowLocked ends the window: long-polls wake with the final state and
// port 7767 closes. Nothing is stored here.
func (p *pairing) closeWindowLocked(final pairState) {
	w := p.win
	p.win = nil
	w.state = final
	w.timer.Stop()
	close(w.decided)
	p.last = pairStatus{State: final}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		w.srv.Shutdown(ctx)
		w.ln.Close()
	}()
}

func validName(n string) bool {
	if !utf8.ValidString(n) {
		return false
	}
	c := utf8.RuneCountInString(n)
	if c < 1 || c > 64 {
		return false
	}
	for _, r := range n {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// handleEnroll serves POST /v1/enroll on port 7767 and long-polls for the decision.
func (p *pairing) handleEnroll(wr http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(wr, r.Body, 4096)).Decode(&body); err != nil {
		writeJSON(wr, 400, errBody("bad body", ""))
		return
	}
	if !validName(body.Name) {
		writeJSON(wr, 400, errBody("bad name", ""))
		return
	}
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		writeJSON(wr, 403, errBody("no client certificate", ""))
		return
	}
	leaf := r.TLS.PeerCertificates[0]
	h := spkiHash(leaf)
	tok, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(body.Token, "="))
	if err != nil {
		tok = nil
	}

	p.mu.Lock()
	w := p.win
	if w == nil {
		p.mu.Unlock()
		writeJSON(wr, 410, errBody("no pairing window", ""))
		return
	}
	if len(tok) != len(w.token) || subtle.ConstantTimeCompare(tok, w.token) != 1 {
		p.mu.Unlock()
		p.d.audit.write(auditLine{Action: "refusal", Result: "enroll_wrong_token"})
		writeJSON(wr, 403, errBody("wrong token", ""))
		return
	}
	if w.bound != nil && *w.bound != h {
		p.mu.Unlock()
		p.d.audit.write(auditLine{Action: "refusal", Result: "enroll_token_bound"})
		writeJSON(wr, 403, errBody("token bound to another key", ""))
		return
	}
	if w.bound == nil {
		w.bound = &h
		w.name = body.Name
		w.code = pairCode(p.d.cert.Leaf.RawSubjectPublicKeyInfo, leaf.RawSubjectPublicKeyInfo)
		w.state = pairPending
		p.d.audit.write(auditLine{Action: "enroll", State: "pending"})
	}
	decided := w.decided
	p.mu.Unlock()

	select {
	case <-decided:
	case <-r.Context().Done():
		return
	}
	p.mu.Lock()
	final, id := w.state, w.deviceID
	p.mu.Unlock()
	switch final {
	case pairAllowed:
		writeJSON(wr, 200, map[string]string{"device_id": id})
	case pairDenied:
		writeJSON(wr, 403, errBody("denied", ""))
	default:
		writeJSON(wr, 410, errBody("expired", ""))
	}
}

func errBody(e, reason string) map[string]any {
	m := map[string]any{"error": e}
	if reason != "" {
		m["reason"] = reason
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
