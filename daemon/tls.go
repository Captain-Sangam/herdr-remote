package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// loadOrCreateIdentity returns the server certificate, writing server.key
// (0600) and server.crt (0644) on first start.
func loadOrCreateIdentity(dir string) (tls.Certificate, error) {
	keyPath := filepath.Join(dir, "server.key")
	crtPath := filepath.Join(dir, "server.crt")
	if cert, err := tls.LoadX509KeyPair(crtPath, keyPath); err == nil {
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		return cert, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "herdr-remoted"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := writeFileAtomic(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return tls.Certificate{}, err
	}
	if err := writeFileAtomic(crtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return tls.Certificate{}, err
	}
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	return cert, err
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func spkiHash(c *x509.Certificate) [32]byte { return sha256.Sum256(c.RawSubjectPublicKeyInfo) }

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func isP256(c *x509.Certificate) bool {
	k, ok := c.PublicKey.(*ecdsa.PublicKey)
	return ok && k.Curve == elliptic.P256()
}

// pairCode is the 6-digit code both screens show during pairing.
func pairCode(serverSPKI, clientSPKI []byte) string {
	h := sha256.Sum256(append(append([]byte{}, serverSPKI...), clientSPKI...))
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(h[:4])%1000000)
}

// mutualTLS is the shared listener configuration (decision 4). verify runs on
// every handshake, including resumptions, with a parsed P-256 leaf.
func mutualTLS(cert tls.Certificate, verify func(leaf *x509.Certificate) error) *tls.Config {
	return &tls.Config{
		Certificates:           []tls.Certificate{cert},
		MinVersion:             tls.VersionTLS13,
		ClientAuth:             tls.RequireAnyClientCert,
		SessionTicketsDisabled: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return errors.New("no client certificate")
			}
			leaf := cs.PeerCertificates[0]
			if !isP256(leaf) {
				return errors.New("client key is not P-256")
			}
			return verify(leaf)
		},
	}
}
