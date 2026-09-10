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
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/quic-go/webtransport-go"
)

// dialQUIC opens a WebTransport session to ln and returns its message stream as
// a quicConn, mirroring what quicDialer does for real dials (nonce in the
// CONNECT request, server opens the stream).
func dialQUIC(ctx context.Context, ln *quicListener) (*quicConn, error) {
	rot, err := newQUICCertRotator()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, quicNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	wd := &webtransport.Dialer{
		TLSClientConfig:      newQUICTLSConfig(rot),
		QUICConfig:           quicConfig,
		ApplicationProtocols: []string{hex.EncodeToString(nonce)},
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
	return newQUICConn(sess, str, nonce), nil
}

func newTestQUICListener(t *testing.T) *quicListener {
	t.Helper()
	rot, err := newQUICCertRotator()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := newQUICListener("127.0.0.1:0", newQUICTLSConfig(rot))
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func TestQUICListenerAccept(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The server opens the stream and writes first; the dialer reads.
	readCh := make(chan string, 1)
	go func() {
		conn, err := dialQUIC(ctx, ln)
		if err != nil {
			t.Error(err)
			ln.Close()
			return
		}
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 5)
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Error(err)
			ln.Close()
			return
		}
		readCh <- string(buf)
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, ok := conn.RemoteAddr().(*net.UDPAddr); !ok {
		t.Fatalf("remote addr is %T, want *net.UDPAddr", conn.RemoteAddr())
	}
	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-readCh:
		if got != "hello" {
			t.Fatalf("read %q, want %q", got, "hello")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for dialer to read")
	}
}

func TestServerQUICListen(t *testing.T) {
	srv := &Server{
		Config: Config{
			PrivateKey:     newkey(),
			MaxPeers:       10,
			NoDial:         true,
			NoDiscovery:    true,
			ListenQUICAddr: "127.0.0.1:0",
			Logger:         testlog.Logger(t, log.LvlTrace),
		},
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	if srv.ListenQUICAddr == "127.0.0.1:0" {
		t.Fatal("ListenQUICAddr not updated with actual port")
	}
	var port enr.QUIC
	if err := srv.LocalNode().Node().Load(&port); err != nil {
		t.Fatal(err)
	}
	if port == 0 {
		t.Fatal("quic port in node record is zero")
	}
	var qh quicCertHash
	if err := srv.LocalNode().Node().Load(&qh); err != nil {
		t.Fatal(err)
	}
	if len(qh) != 2*sha256.Size {
		t.Fatalf("qh length %d, want %d", len(qh), 2*sha256.Size)
	}
}

func TestQUICTransport(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverKey := newkey()

	testHandshake := func(name string) *protoHandshake {
		pub := crypto.FromECDSAPub(&newkey().PublicKey)
		return &protoHandshake{Version: baseProtocolVersion, Name: name, ID: pub[1:]}
	}

	type dialResult struct {
		tr    transport
		their *protoHandshake
		err   error
	}
	dialCh := make(chan dialResult, 1)
	go func() {
		conn, err := dialQUIC(ctx, ln)
		if err != nil {
			dialCh <- dialResult{err: err}
			return
		}
		tr := newQUICTransport(conn, conn.session, conn.nonce, &serverKey.PublicKey)
		if _, err := tr.doEncHandshake(newkey()); err != nil {
			dialCh <- dialResult{err: err}
			return
		}
		their, err := tr.doProtoHandshake(testHandshake("dialer"))
		dialCh <- dialResult{tr: tr, their: their, err: err}
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
	// The server assigns the unverified dialer a random identity.
	if remote == nil || remote.Equal(&serverKey.PublicKey) {
		t.Fatal("server did not assign a random identity")
	}
	their, err := lt.doProtoHandshake(testHandshake("listener"))
	if err != nil {
		t.Fatal(err)
	}
	if their.Name != "dialer" {
		t.Fatalf("got handshake name %q, want %q", their.Name, "dialer")
	}
	// The assigned identity replaces the dialer's self-reported one.
	if !bytes.Equal(their.ID, crypto.FromECDSAPub(remote)[1:]) {
		t.Fatal("proto handshake ID not overridden with assigned identity")
	}
	res := <-dialCh
	if res.err != nil {
		t.Fatal(res.err)
	}
	if res.their.Name != "listener" {
		t.Fatalf("got handshake name %q, want %q", res.their.Name, "listener")
	}
	wireSnappy := func(tr transport) bool {
		return tr.(*wireTransport).conn.(*quicWire).snappyReadBuffer != nil
	}
	if !wireSnappy(lt) || !wireSnappy(res.tr) {
		t.Fatal("snappy not enabled after handshake")
	}

	if err := Send(res.tr, 0x42, []string{"hello", "quic"}); err != nil {
		t.Fatal(err)
	}
	msg, err := lt.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Code != 0x42 {
		t.Fatalf("got code %d, want 0x42", msg.Code)
	}
	var payload []string
	if err := msg.Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 2 || payload[0] != "hello" || payload[1] != "quic" {
		t.Fatalf("payload mismatch: %v", payload)
	}
}

func TestQUICTransportIdentityMismatch(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialErr := make(chan error, 1)
	go func() {
		conn, err := dialQUIC(ctx, ln)
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
	if _, err := lt.doEncHandshake(newkey()); err != nil {
		t.Fatal(err)
	}
	if err := <-dialErr; err == nil {
		t.Fatal("dial handshake succeeded with wrong dialDest, want identity mismatch")
	}
}

func TestServerQUICPeer(t *testing.T) {
	newServer := func(name string) *Server {
		srv := &Server{
			Config: Config{
				Name:           name,
				PrivateKey:     newkey(),
				MaxPeers:       10,
				NoDiscovery:    true,
				ListenQUICAddr: "127.0.0.1:0",
				Logger:         testlog.Logger(t, log.LvlTrace),
			},
		}
		if err := srv.Start(); err != nil {
			t.Fatal(err)
		}
		return srv
	}
	srvA := newServer("A")
	defer srvA.Stop()
	srvB := newServer("B")
	defer srvB.Stop()

	srvB.AddPeer(srvA.LocalNode().Node())

	deadline := time.Now().Add(10 * time.Second)
	for srvA.PeerCount() == 0 || srvB.PeerCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("peers not connected: A=%d B=%d", srvA.PeerCount(), srvB.PeerCount())
		}
		time.Sleep(50 * time.Millisecond)
	}
	peer := srvB.Peers()[0]
	if peer.ID() != srvA.LocalNode().ID() {
		t.Fatalf("connected to wrong peer %v", peer.ID())
	}
	if _, ok := peer.RemoteAddr().(*net.UDPAddr); !ok {
		t.Fatalf("peer connected over %T, want *net.UDPAddr", peer.RemoteAddr())
	}
}
