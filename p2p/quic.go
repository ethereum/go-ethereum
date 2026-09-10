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
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/dunglas/httpsfv"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// quicWTPath is the HTTP path browsers and nodes use to open a WebTransport
// session to a devp2p node.
const quicWTPath = "/devp2p"

// quicConfig is the QUIC config shared by the listener and dialer.
// Both are required by WebTransport.
var quicConfig = &quic.Config{
	EnableDatagrams:                  true,
	EnableStreamResetPartialDelivery: true,
}

// quicListener is a QUIC listener which implements net.Listener
type quicListener struct {
	conn    *net.UDPConn
	tr      *quic.Transport
	ln      *quic.Listener
	wt      *webtransport.Server
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
	ln, err := tr.Listen(tlsConf, quicConfig)
	if err != nil {
		tr.Close()
		udp.Close()
		return nil, err
	}
	l := &quicListener{conn: udp, tr: tr, ln: ln, tlsConf: tlsConf, conns: make(chan *quicConn)}
	l.ctx, l.cancel = context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc(quicWTPath, l.handleSession)
	h3 := &http3.Server{TLSConfig: tlsConf, QUICConfig: quicConfig, Handler: mux}

	webtransport.ConfigureHTTP3Server(h3)
	l.wt = &webtransport.Server{
		H3:          h3,
		CheckOrigin: func(*http.Request) bool { return true },
	}

	go l.acceptLoop()
	return l, nil
}

func (l *quicListener) acceptLoop() {
	for {
		qc, err := l.ln.Accept(l.ctx)
		if err != nil {
			return
		}
		go l.wt.ServeQUICConn(qc)
	}
}

// handleSession upgrades an HTTP/3 request to a WebTransport session and opens
// the message stream.
func (l *quicListener) handleSession(w http.ResponseWriter, r *http.Request) {
	nonce, err := readNonce(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sess, err := l.wt.Upgrade(w, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(l.ctx, handshakeTimeout)
	defer cancel()
	str, err := sess.OpenStreamSync(ctx)
	if err != nil {
		sess.CloseWithError(0, "")
		return
	}
	select {
	case l.conns <- newQUICConn(sess, str, nonce):
	case <-ctx.Done():
		sess.CloseWithError(0, "")
	}
}

// readNonce extracts the dialer's hex-encoded nonce from the WT-Available-Protocols
// header of the CONNECT request.
func readNonce(r *http.Request) ([]byte, error) {
	list, err := httpsfv.UnmarshalList(r.Header.Values("WT-Available-Protocols"))
	if err != nil || len(list) == 0 {
		return nil, errors.New("quic: missing nonce")
	}
	item, ok := list[0].(httpsfv.Item)
	if !ok {
		return nil, errors.New("quic: invalid nonce")
	}
	s, ok := item.Value.(string)
	if !ok {
		return nil, errors.New("quic: invalid nonce")
	}
	nonce, err := hex.DecodeString(s)
	if err != nil || len(nonce) != quicNonceLen {
		return nil, errors.New("quic: invalid nonce")
	}
	return nonce, nil
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
	l.wt.Close()
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
	if !ok || dest.Load(&qh) != nil || !qh.valid() {
		return d.tcp.Dial(ctx, dest)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	nonce := make([]byte, quicNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	wd := &webtransport.Dialer{
		TLSClientConfig:      quicClientTLSConfig(d.ln.tlsConf, qh),
		QUICConfig:           quicConfig,
		ApplicationProtocols: []string{hex.EncodeToString(nonce)},
		DialAddr: func(ctx context.Context, _ string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
			return d.ln.tr.Dial(ctx, net.UDPAddrFromAddrPort(ep), tlsCfg, cfg)
		},
	}
	url := fmt.Sprintf("https://%s%s", ep.String(), quicWTPath)
	_, sess, err := wd.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	str, err := sess.AcceptStream(ctx)
	if err != nil {
		sess.CloseWithError(0, "")
		return nil, err
	}
	return newQUICConn(sess, str, nonce), nil
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
	*webtransport.Stream
	session *webtransport.Session
	nonce   []byte
}

var _ net.Conn = (*quicConn)(nil)

func newQUICConn(session *webtransport.Session, str *webtransport.Stream, nonce []byte) *quicConn {
	return &quicConn{Stream: str, session: session, nonce: nonce}
}

func (c *quicConn) Close() error {
	return c.session.CloseWithError(0, "")
}

func (c *quicConn) LocalAddr() net.Addr {
	return c.session.LocalAddr()
}

func (c *quicConn) RemoteAddr() net.Addr {
	return c.session.RemoteAddr()
}
