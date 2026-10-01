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
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

// dialQUICNode opens a WebTransport session without a nonce, mirroring a node
// dial: identity is established by mutual authentication over the TLS exporter.
func dialQUICNode(ctx context.Context, ln *quicListener) (*quicConn, error) {
	rot, err := newQUICCertRotator()
	if err != nil {
		return nil, err
	}
	wd := &webtransport.Dialer{
		TLSClientConfig: newQUICTLSConfig(rot),
		QUICConfig:      quicConfig,
	}
	_, sess, err := wd.Dial(ctx, "https://"+ln.Addr().String()+quicWTPath, nil)
	if err != nil {
		return nil, err
	}
	str, err := sess.AcceptStream(ctx)
	if err != nil {
		sess.CloseWithError(0, "")
		return nil, err
	}
	return newQUICConn(sess, str, nil), nil
}

// TestQUICNodeHandshake checks that two nodes authenticate each other over the
// TLS exporter: each recovers the other's real node identity.
func TestQUICNodeHandshake(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverKey := newkey()
	dialerKey := newkey()

	type dialResult struct {
		remote *ecdsa.PublicKey
		err    error
	}
	dialCh := make(chan dialResult, 1)
	go func() {
		conn, err := dialQUICNode(ctx, ln)
		if err != nil {
			dialCh <- dialResult{err: err}
			return
		}
		tr := newQUICTransport(conn, conn.session, conn.nonce, &serverKey.PublicKey)
		remote, err := tr.doEncHandshake(dialerKey)
		dialCh <- dialResult{remote: remote, err: err}
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	qc := conn.(*quicConn)
	lt := newQUICTransport(conn, qc.session, qc.nonce, nil)
	remote, err := lt.doEncHandshake(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	// The server recovers the dialer's real identity, not a random one.
	if !remote.Equal(&dialerKey.PublicKey) {
		t.Fatal("server did not recover the dialer's real identity")
	}

	res := <-dialCh
	if res.err != nil {
		t.Fatal(res.err)
	}
	// The dialer verified the server against the key it dialed.
	if !res.remote.Equal(&serverKey.PublicKey) {
		t.Fatal("dialer did not verify the server identity")
	}
}

// TestQUICNodeHandshakeMismatch checks that a node dial fails when the server's
// proven identity does not match the dialed key.
func TestQUICNodeHandshakeMismatch(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialErr := make(chan error, 1)
	go func() {
		conn, err := dialQUICNode(ctx, ln)
		if err != nil {
			dialErr <- err
			return
		}
		wrong := newkey()
		tr := newQUICTransport(conn, conn.session, conn.nonce, &wrong.PublicKey)
		_, err = tr.doEncHandshake(newkey())
		dialErr <- err
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	qc := conn.(*quicConn)
	lt := newQUICTransport(conn, qc.session, qc.nonce, nil)
	// The server proves itself (unblocking the dialer) then waits for the dialer's
	// proof, which never comes after the mismatch; run it off the test goroutine.
	go lt.doEncHandshake(newkey())
	if err := <-dialErr; err == nil {
		t.Fatal("dial handshake succeeded with wrong dialDest, want identity mismatch")
	}
}
