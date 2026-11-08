package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type config struct {
	MacIdleSeconds *float64 `json:"mac_idle_seconds"`
}

func (c config) macIdle() time.Duration {
	if c.MacIdleSeconds == nil {
		return 10 * time.Second
	}
	return time.Duration(*c.MacIdleSeconds * float64(time.Second))
}

type Daemon struct {
	dir         string
	pairedAddr  string
	enrollAddr  string
	adminPath   string
	r           *runner
	hc          *HC
	cert        tls.Certificate
	devices     *deviceStore
	audit       *auditLog
	pair        *pairing
	cfg         config
	conns       connRegistry
	mux         *http.ServeMux
	pairedLn    net.Listener
	pairedSrv   *http.Server
	adminLn     net.Listener
	pairedPort  int
	state       *state
	live        *liveManager
	transcripts *transcripts
	actions     *actions
	ctx         context.Context
	cancel      context.CancelFunc
}

func defaultStateDir() string {
	if d := os.Getenv("HERDR_REMOTE_STATE_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "herdr-remote")
}

func newDaemon(dir string, r *runner, pairedAddr, enrollAddr string) (*Daemon, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	d := &Daemon{dir: dir, pairedAddr: pairedAddr, enrollAddr: enrollAddr, adminPath: filepath.Join(dir, "admin.sock"), r: r, mux: http.NewServeMux()}
	var err error
	if d.cert, err = loadOrCreateIdentity(dir); err != nil {
		return nil, err
	}
	if d.devices, err = openDevices(filepath.Join(dir, "devices.json")); err != nil {
		return nil, err
	}
	if d.audit, err = openAudit(filepath.Join(dir, "audit.log")); err != nil {
		return nil, err
	}
	if b, err := os.ReadFile(filepath.Join(dir, "config.json")); err == nil {
		if err := json.Unmarshal(b, &d.cfg); err != nil {
			return nil, fmt.Errorf("config.json: %w", err)
		}
	}
	d.hc = newHC(r)
	d.pair = newPairing(d)
	d.conns.init()
	d.state = newState(d)
	d.live = newLiveManager(d)
	d.transcripts = newTranscripts(d)
	d.actions = newActions(d)
	d.state.onSize = d.live.endTerminal
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.routes()
	return d, nil
}

// pin is the base64url SHA-256 of the server SPKI, which the phone pins.
func (d *Daemon) pin() string {
	h := spkiHash(d.cert.Leaf)
	return b64url(h[:])
}

// start opens the paired listener and the admin socket.
func (d *Daemon) start() error {
	ln, err := net.Listen("tcp", d.pairedAddr)
	if err != nil {
		return err
	}
	d.pairedLn = ln
	d.pairedPort = ln.Addr().(*net.TCPAddr).Port
	d.pairedSrv = &http.Server{
		Handler: d.deviceMiddleware(d.mux),
		TLSConfig: mutualTLS(d.cert, func(leaf *x509.Certificate) error {
			if d.devices.bySPKI(spkiHash(leaf)) == nil {
				return errors.New("unknown device")
			}
			return nil
		}),
		ConnState:         d.conns.onState,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go d.pairedSrv.ServeTLS(ln, "", "")
	if err := d.serveAdmin(); err != nil {
		ln.Close()
		return err
	}
	go d.state.run(d.ctx)
	return nil
}

func (d *Daemon) stop() {
	d.cancel()
	d.pair.cancel()
	if d.adminLn != nil {
		d.adminLn.Close()
	}
	if d.pairedSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		d.pairedSrv.Shutdown(ctx)
		d.pairedSrv.Close()
	}
	d.hc.Close()
}

type ctxKey int

const ctxDevice ctxKey = 1

// deviceMiddleware looks the device up again on every request (decision 4).
func (d *Daemon) deviceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			writeJSON(w, 401, errBody("no client certificate", ""))
			return
		}
		dev := d.devices.bySPKI(spkiHash(r.TLS.PeerCertificates[0]))
		if dev == nil {
			writeJSON(w, 401, errBody("unknown device", ""))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxDevice, dev)))
	})
}

func deviceOf(r *http.Request) *Device {
	d, _ := r.Context().Value(ctxDevice).(*Device)
	return d
}

// revoke marks the device as revoked, so the handshake and per-request checks
// treat it as unknown, ends its connections and live views, then writes
// devices.json without it. A failed write keeps the mark and returns the
// error, so a retry finds the device and writes again.
func (d *Daemon) revoke(id string) error {
	dev, ok := d.devices.mark(id)
	if !ok {
		return fmt.Errorf("no device %q", id)
	}
	raw, err := base64.RawURLEncoding.DecodeString(dev.SPKI)
	if err == nil && len(raw) == 32 {
		var h [32]byte
		copy(h[:], raw)
		d.conns.closeAll(h)
	}
	d.onRevoke(dev)
	if err := d.devices.drop(id); err != nil {
		return err
	}
	return d.audit.write(auditLine{Action: "revoke", DeviceID: dev.ID})
}

// onRevoke ends the live views of a revoked device; its connections are already closed.
func (d *Daemon) onRevoke(dev Device) { d.live.endDevice(dev.ID) }

// connRegistry tracks open paired connections by client key, so a revoke can close them.
type connRegistry struct {
	mu     sync.Mutex
	byKey  map[[32]byte]map[net.Conn]struct{}
	byConn map[net.Conn][32]byte
}

func (c *connRegistry) init() {
	c.byKey = map[[32]byte]map[net.Conn]struct{}{}
	c.byConn = map[net.Conn][32]byte{}
}

func (c *connRegistry) onState(conn net.Conn, st http.ConnState) {
	switch st {
	case http.StateActive:
		tc, ok := conn.(*tls.Conn)
		if !ok {
			return
		}
		cs := tc.ConnectionState()
		if len(cs.PeerCertificates) == 0 {
			return
		}
		h := spkiHash(cs.PeerCertificates[0])
		c.mu.Lock()
		if _, seen := c.byConn[conn]; !seen {
			c.byConn[conn] = h
			if c.byKey[h] == nil {
				c.byKey[h] = map[net.Conn]struct{}{}
			}
			c.byKey[h][conn] = struct{}{}
		}
		c.mu.Unlock()
	case http.StateClosed, http.StateHijacked:
		c.mu.Lock()
		if h, seen := c.byConn[conn]; seen {
			delete(c.byConn, conn)
			delete(c.byKey[h], conn)
		}
		c.mu.Unlock()
	}
}

func (c *connRegistry) closeAll(h [32]byte) int {
	c.mu.Lock()
	conns := c.byKey[h]
	delete(c.byKey, h)
	for conn := range conns {
		delete(c.byConn, conn)
	}
	c.mu.Unlock()
	for conn := range conns {
		conn.Close()
	}
	return len(conns)
}

// endpoints lists where the phone can reach this Mac: private IPv4 addresses
// of interfaces that are up, any Tailscale address, and the MagicDNS name.
func (d *Daemon) endpoints() []string {
	var out []string
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipn.IP.To4()
			if ip == nil {
				continue
			}
			if ip.IsPrivate() || isCGNAT(ip) {
				out = append(out, ip.String())
			}
		}
	}
	if name := d.magicDNS(); name != "" {
		out = append(out, name)
	}
	return out
}

func isCGNAT(ip net.IP) bool { return ip[0] == 100 && ip[1]&0xc0 == 64 }

func (d *Daemon) magicDNS() string {
	if d.r.progs.Tailscale == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b, err := d.r.tailscaleStatus(ctx)
	if err != nil {
		return ""
	}
	var st struct {
		Self struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}
	if json.Unmarshal(b, &st) != nil {
		return ""
	}
	return strings.TrimSuffix(st.Self.DNSName, ".")
}
