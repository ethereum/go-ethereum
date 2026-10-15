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
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/dunglas/httpsfv"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"

	p2pquic "github.com/ethereum/go-ethereum/p2p/quic"
)

// quicWTPath is the HTTP path browsers and nodes use to open a WebTransport
// session to a devp2p node.
const quicWTPath = "/devp2p"

// quicNonceProto is the WT-Available-Protocols entry the dialer sends to carry
// its hex-encoded nonce for the id-proof, formatted as "ENR-key-proof-v1=<hex>".
const quicNonceProto = "ENR-key-proof-v1="

// quicNonceLen is the expected length of the id-proof nonce.
const quicNonceLen = 32

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

	sem    chan struct{} // bounds concurrently served connections
	conns  chan *p2pquic.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func newQUICListener(addr string, tlsConf *tls.Config, maxConns int) (*quicListener, error) {
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
	l := &quicListener{conn: udp, tr: tr, ln: ln, tlsConf: tlsConf, conns: make(chan *p2pquic.Conn), sem: make(chan struct{}, maxConns)}
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
		// Acquire a slot before accepting so the number of connections we serve
		// is bounded. When full, we stop accepting and QUIC refuses new
		// connections, throttling handshakes before they reach the peer checks.
		select {
		case l.sem <- struct{}{}:
		case <-l.ctx.Done():
			return
		}
		qc, err := l.ln.Accept(l.ctx)
		if err != nil {
			<-l.sem
			return
		}
		go func() {
			defer func() { <-l.sem }()
			l.wt.ServeQUICConn(qc)
		}()
	}
}

// handleSession upgrades an HTTP/3 request to a WebTransport session and opens
// the message stream.
func (l *quicListener) handleSession(w http.ResponseWriter, r *http.Request) {
	nonce, caps, err := readConnect(r)
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
	case l.conns <- p2pquic.NewConn(sess, str, nil, nonce, caps):
	case <-ctx.Done():
		sess.CloseWithError(0, "")
	}
}

// readConnect parses the WT-Available-Protocols header of the CONNECT request.
// A browser lists the id-proof nonce ("ENR-key-proof-v1=<hex>") and its advertised
// capabilities. If there is no nonce entry, it will be marked as a node peer.
func readConnect(r *http.Request) (nonce []byte, caps []string, err error) {
	hdr := r.Header.Values("WT-Available-Protocols")
	if len(hdr) == 0 {
		return nil, nil, nil
	}
	list, err := httpsfv.UnmarshalList(hdr)
	if err != nil {
		return nil, nil, errors.New("quic: invalid WT-Available-Protocols")
	}
	for _, member := range list {
		item, ok := member.(httpsfv.Item)
		if !ok {
			continue
		}
		s, ok := item.Value.(string)
		if !ok {
			continue
		}
		if hexNonce, ok := strings.CutPrefix(s, quicNonceProto); ok {
			nonce, err = hex.DecodeString(hexNonce)
			if err != nil || len(nonce) != quicNonceLen {
				return nil, nil, errors.New("quic: invalid nonce")
			}
			continue
		}
		caps = append(caps, s)
	}
	return nonce, caps, nil
}

// parseCap parses a "name/version" capability entry, e.g. "eth/70".
func parseCap(s string) (Cap, bool) {
	i := strings.LastIndexByte(s, '/')
	if i <= 0 {
		return Cap{}, false
	}
	version, err := strconv.ParseUint(s[i+1:], 10, 0)
	if err != nil {
		return Cap{}, false
	}
	return Cap{Name: s[:i], Version: uint(version)}, true
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
