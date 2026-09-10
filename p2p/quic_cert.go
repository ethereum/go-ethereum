// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"sync"
	"time"
)

// ALPN is the application protocol negotiated for QUIC connections. Nodes and
// browsers both connect over WebTransport, which runs on HTTP/3.
const ALPN = "h3"

const (
	// quicCertValidity is the certificate lifetime, at the two-week maximum
	// that browsers allow for serverCertificateHashes.
	quicCertValidity = 14 * 24 * time.Hour
	// quicCertRotation is how often the current certificate is replaced by the
	// pre-announced next one, which happens when the current certificate
	// expires. current and next cover consecutive, non-overlapping validity
	// periods, and next is announced a full period ahead so peers learn it
	// before it is used.
	quicCertRotation = quicCertValidity
)

// quicCertHash is the sha256 hashes of the node's current
// TLS certificate and the next one ("qh" ENR entry).
type quicCertHash []byte

func (quicCertHash) ENRKey() string { return "qh" }

func (qh quicCertHash) valid() bool {
	return len(qh) == sha256.Size || len(qh) == 2*sha256.Size
}

// matches reports whether h is one of the announced certificate hashes.
func (qh quicCertHash) matches(h [sha256.Size]byte) bool {
	for len(qh) >= sha256.Size {
		if bytes.Equal(qh[:sha256.Size], h[:]) {
			return true
		}
		qh = qh[sha256.Size:]
	}
	return false
}

// quicCertRotator maintains the node's current and next TLS certificate.
// Both hashes are announced in "qh", so a node record fetched up to one
// rotation ago still verifies the certificate presented after a rotation.
//
// TODO: peers whose node record is more than one rotation stale (e.g. from a
// DNS discovery list) will not match the current certificate. Widen the
// tolerance if that becomes a problem, for example by announcing more hashes.
type quicCertRotator struct {
	mu            sync.Mutex
	current, next *tls.Certificate
}

func newQUICCertRotator() (*quicCertRotator, error) {
	now := time.Now()
	current, err := generateQUICCert(now)
	if err != nil {
		return nil, err
	}
	// next covers the period after current expires and is announced now.
	next, err := generateQUICCert(now.Add(quicCertValidity))
	if err != nil {
		return nil, err
	}
	return &quicCertRotator{current: current, next: next}, nil
}

func (r *quicCertRotator) getCert() *tls.Certificate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current
}

// rotate promotes the next certificate and generates a new one whose validity
// begins when the promoted certificate expires.
func (r *quicCertRotator) rotate() error {
	r.mu.Lock()
	notBefore := r.next.Leaf.NotAfter
	r.mu.Unlock()
	next, err := generateQUICCert(notBefore)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.current, r.next = r.next, next
	r.mu.Unlock()
	return nil
}

func (r *quicCertRotator) qh() quicCertHash {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := sha256.Sum256(r.current.Certificate[0])
	next := sha256.Sum256(r.next.Certificate[0])
	return append(current[:], next[:]...)
}

func generateQUICCert(notBefore time.Time) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: serial,
		NotBefore:    notBefore,
		NotAfter:     notBefore.Add(quicCertValidity),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, nil
}

// newQUICTLSConfig serves the rotator's current certificate on both the
// listening and dialing side, so certificates rotate without restarting
// the listener.
func newQUICTLSConfig(rot *quicCertRotator) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return rot.getCert(), nil
		},
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return rot.getCert(), nil
		},
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequireAnyClientCert,
		NextProtos:         []string{ALPN},
	}
}

// quicClientTLSConfig sets the tls.Config to dial a node whose ENR advertises
// the hashes of its certificates. Certificate validity is enforced here because
// the base config disables the standard chain verification.
func quicClientTLSConfig(base *tls.Config, qh quicCertHash) *tls.Config {
	conf := base.Clone()
	conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if !qh.matches(sha256.Sum256(rawCerts[0])) {
			return errors.New("quic: certificate does not match qh in node record")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}
		now := time.Now()
		if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
			return errors.New("quic: certificate expired or not yet valid")
		}
		return nil
	}
	return conf
}
