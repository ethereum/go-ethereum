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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/quic-go/quic-go"
)

const ALPN = "devp2p" //todo

// quicCertHash is the "qh" ENR entry, holding the sha256 hash of the node's
// TLS certificate.
type quicCertHash [sha256.Size]byte

func (quicCertHash) ENRKey() string { return "qh" }

// quicClientTLSConfig sets the tls.Config to dial a node whose ENR advertises
// the hash of provided certificate.
func quicClientTLSConfig(base *tls.Config, certHash quicCertHash) *tls.Config {
	conf := base.Clone()
	conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if sha256.Sum256(rawCerts[0]) != certHash {
			return errors.New("quic: certificate does not match qh in node record")
		}
		return nil
	}
	return conf
}

// quicListener is a QUIC listener which implements net.Listener
type quicListener struct {
	conn    *net.UDPConn
	tr      *quic.Transport
	ln      *quic.Listener
	tlsConf *tls.Config

	conns  chan *quicConn
	ctx    context.Context
	cancel context.CancelFunc
}

func newQUICListener(addr string, tlsConf *tls.Config) (*quicListener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	udp, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	tr := &quic.Transport{Conn: udp}
	ln, err := tr.Listen(tlsConf, nil)
	if err != nil {
		tr.Close()
		udp.Close()
		return nil, err
	}
	l := &quicListener{conn: udp, tr: tr, ln: ln, tlsConf: tlsConf, conns: make(chan *quicConn)}
	l.ctx, l.cancel = context.WithCancel(context.Background())
	go l.acceptLoop()
	return l, nil
}

// acceptLoop waits for the message stream of each accepted connection in its
// own goroutine, so a dialer that never opens the stream cannot block other
// accepts. The number of pending connections is bounded by the semaphore.
func (l *quicListener) acceptLoop() {
	sem := make(chan struct{}, defaultMaxPendingPeers)
	for {
		select {
		case sem <- struct{}{}:
		case <-l.ctx.Done():
			return
		}
		qc, err := l.ln.Accept(l.ctx)
		if err != nil {
			return
		}
		go func() {
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(l.ctx, handshakeTimeout)
			defer cancel()
			str, err := qc.AcceptStream(ctx)
			if err != nil {
				qc.CloseWithError(0, "")
				return
			}
			select {
			case l.conns <- newQUICConn(qc, str):
			case <-l.ctx.Done():
				qc.CloseWithError(0, "")
			}
		}()
	}
}

func (l *quicListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.ctx.Done():
		return nil, net.ErrClosed
	}
}

func (l *quicListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *quicListener) Close() error {
	l.cancel()
	l.ln.Close()
	l.tr.Close()
	return l.conn.Close()
}

// quicDialer dials over QUIC when the destination advertises a QUIC endpoint
// and certificate hash, falling back to TCP otherwise.
type quicDialer struct {
	tcp NodeDialer
	ln  *quicListener
}

func (d *quicDialer) Dial(ctx context.Context, dest *enode.Node) (net.Conn, error) {
	ep, ok := dest.QUICEndpoint()
	var qh quicCertHash
	if !ok || dest.Load(&qh) != nil {
		return d.tcp.Dial(ctx, dest)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()
	qc, err := d.ln.tr.Dial(ctx, net.UDPAddrFromAddrPort(ep), quicClientTLSConfig(d.ln.tlsConf, qh), nil)
	if err != nil {
		return nil, err
	}
	str, err := qc.OpenStreamSync(ctx)
	if err != nil {
		qc.CloseWithError(0, "")
		return nil, err
	}
	return newQUICConn(qc, str), nil
}

// unwrapQUICConn returns the quicConn carried by fd, reaching through the
// metering wrapper applied by listenLoop and dialTask.
func unwrapQUICConn(fd net.Conn) *quicConn {
	if mc, ok := fd.(*meteredConn); ok {
		fd = mc.Conn
	}
	qc, _ := fd.(*quicConn)
	return qc
}

// quicConn is a net.Conn view of a QUIC connection and its single
// bidirectional stream.
type quicConn struct {
	*quic.Stream
	qc *quic.Conn
}

var _ net.Conn = (*quicConn)(nil)

func newQUICConn(qc *quic.Conn, str *quic.Stream) *quicConn {
	return &quicConn{Stream: str, qc: qc}
}

func (c *quicConn) Close() error {
	return c.qc.CloseWithError(0, "")
}

func (c *quicConn) LocalAddr() net.Addr {
	return c.qc.LocalAddr()
}

func (c *quicConn) RemoteAddr() net.Addr {
	return c.qc.RemoteAddr()
}

// todo: create Certificate rotater, replace the current one
func generateQUICTLSConfig() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{{Certificate: [][]byte{cert}, PrivateKey: key}},
		InsecureSkipVerify: true,
		ClientAuth:         tls.RequireAnyClientCert,
		NextProtos:         []string{ALPN},
	}, nil
}
