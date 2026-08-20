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
	"crypto/sha256"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/quic-go/quic-go"
)

func dialQUIC(ctx context.Context, addr string) (*quic.Conn, error) {
	tlsConf, err := generateQUICTLSConfig()
	if err != nil {
		return nil, err
	}
	return quic.DialAddr(ctx, addr, tlsConf, nil)
}

func newTestQUICListener(t *testing.T) *quicListener {
	t.Helper()
	tlsConf, err := generateQUICTLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := newQUICListener("127.0.0.1:0", tlsConf)
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

	go func() {
		qc, err := dialQUIC(ctx, ln.Addr().String())
		if err == nil {
			var str *quic.Stream
			if str, err = qc.OpenStreamSync(ctx); err == nil {
				_, err = str.Write([]byte("hello"))
			}
		}
		if err != nil {
			t.Error(err)
			ln.Close()
		}
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, ok := conn.RemoteAddr().(*net.UDPAddr); !ok {
		t.Fatalf("remote addr is %T, want *net.UDPAddr", conn.RemoteAddr())
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 5)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "hello" {
		t.Fatalf("read %q, want %q", buf, "hello")
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
}

func TestQUICTransport(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientKey, serverKey := newkey(), newkey()

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
		qc, err := dialQUIC(ctx, ln.Addr().String())
		if err != nil {
			dialCh <- dialResult{err: err}
			return
		}
		str, err := qc.OpenStreamSync(ctx)
		if err != nil {
			dialCh <- dialResult{err: err}
			return
		}
		tr := newQUICTransport(newQUICConn(qc, str), qc, &serverKey.PublicKey)
		if _, err := tr.doEncHandshake(clientKey); err != nil {
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
	lt := newQUICTransport(conn, conn.(*quicConn).qc, nil)
	remote, err := lt.doEncHandshake(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	if !remote.Equal(&clientKey.PublicKey) {
		t.Fatal("server sees wrong client identity")
	}
	their, err := lt.doProtoHandshake(testHandshake("listener"))
	if err != nil {
		t.Fatal(err)
	}
	if their.Name != "dialer" {
		t.Fatalf("got handshake name %q, want %q", their.Name, "dialer")
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
		qc, err := dialQUIC(ctx, ln.Addr().String())
		if err != nil {
			dialErr <- err
			return
		}
		str, err := qc.OpenStreamSync(ctx)
		if err != nil {
			dialErr <- err
			return
		}
		wrong := newkey()
		tr := newQUICTransport(newQUICConn(qc, str), qc, &wrong.PublicKey)
		_, err = tr.doEncHandshake(newkey())
		dialErr <- err
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	lt := newQUICTransport(conn, conn.(*quicConn).qc, nil)
	if _, err := lt.doEncHandshake(newkey()); err != nil {
		t.Fatal(err)
	}
	if err := <-dialErr; err == nil {
		t.Fatal("dial handshake succeeded with wrong dialDest, want identity mismatch")
	}
}

func TestQUICCertHash(t *testing.T) {
	serverConf, err := generateQUICTLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := newQUICListener("127.0.0.1:0", serverConf)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	base, err := generateQUICTLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	certHash := sha256.Sum256(serverConf.Certificates[0].Certificate[0])

	qc, err := quic.DialAddr(ctx, ln.Addr().String(), quicClientTLSConfig(base, certHash), nil)
	if err != nil {
		t.Fatalf("dial with correct qh failed: %v", err)
	}
	qc.CloseWithError(0, "")

	var wrong quicCertHash
	if _, err := quic.DialAddr(ctx, ln.Addr().String(), quicClientTLSConfig(base, wrong), nil); err == nil {
		t.Fatal("dial with wrong qh succeeded")
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
