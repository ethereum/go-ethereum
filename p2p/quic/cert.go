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

package quic

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
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

// CertHashEntry is the sha256 hashes of the node's current
// TLS certificate and the next one ("qh" ENR entry).
type CertHashEntry []byte

func (CertHashEntry) ENRKey() string { return "qh" }

// CertRotator maintains the node's current and next TLS certificate.
// Both hashes are announced in "qh", so a node record fetched up to one
// rotation ago still verifies the certificate presented after a rotation.
type CertRotator struct {
	mu            sync.Mutex
	current, next *tls.Certificate
	log           log.Logger
	wg            sync.WaitGroup
	quit          chan struct{}
}

func NewCertRotator(log log.Logger) (*CertRotator, error) {
	now := time.Now()
	current, err := generateCertificate(now)
	if err != nil {
		return nil, err
	}
	// next covers the period after current expires and is announced now.
	next, err := generateCertificate(now.Add(quicCertValidity))
	if err != nil {
		return nil, err
	}
	r := &CertRotator{current: current, next: next}
	return r, nil
}

func (r *CertRotator) Get() *tls.Certificate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current
}

// Start enables certificate rotation.
func (r *CertRotator) Start(ln *enode.LocalNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.quit != nil {
		panic("already started")
	}
	r.quit = make(chan struct{})
	r.wg.Add(1)
	go r.loop(ln)
}

// Stop makes the certificate rotation end.
func (r *CertRotator) Stop() {
	r.mu.Lock()
	if r.quit == nil {
		panic("not running")
	}
	close(r.quit)
	r.quit = nil
	defer r.mu.Unlock()
	r.wg.Wait()
}

func (r *CertRotator) loop(ln *enode.LocalNode) {
	timer := time.NewTimer(quicCertRotation)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			if err := r.rotate(); err != nil {
				r.log.Error("Failed to rotate QUIC certificate", "err", err)
			} else {
				ln.Set(r.QHEntry())
			}
			timer.Reset(quicCertRotation)
		case <-r.quit:
			return
		}
	}
}

// rotate promotes the next certificate and generates a new one whose validity
// begins when the promoted certificate expires.
func (r *CertRotator) rotate() error {
	r.mu.Lock()
	notBefore := r.next.Leaf.NotAfter
	r.mu.Unlock()
	next, err := generateCertificate(notBefore)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.current, r.next = r.next, next
	r.mu.Unlock()
	return nil
}

// QHEntry returns the current version of the ENR entry.
func (r *CertRotator) QHEntry() enr.Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := sha256.Sum256(r.current.Certificate[0])
	next := sha256.Sum256(r.next.Certificate[0])
	return CertHashEntry(append(current[:], next[:]...))
}

func generateCertificate(notBefore time.Time) (*tls.Certificate, error) {
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
		KeyUsage:     x509.KeyUsageDigitalSignature,
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

// NewTLSConfig serves the rotator's current certificate on both the
// listening and dialing side, so certificates rotate without restarting
// the listener.
func NewTLSConfig(rot *CertRotator) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return rot.Get(), nil
		},
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return rot.Get(), nil
		},
		InsecureSkipVerify: true,
		ClientAuth:         tls.NoClientCert,
		NextProtos:         []string{ALPN},
	}
}
