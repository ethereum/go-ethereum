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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/quic"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/quic-go/webtransport-go"
)

func newTestQUICListener(t *testing.T) *quicListener {
	t.Helper()

	rot, err := quic.NewCertRotator(testlog.Logger(t, log.LvlDebug))
	if err != nil {
		t.Fatal(err)
	}
	ln, err := newQUICListener("127.0.0.1:0", quic.NewTLSConfig(rot), 64)
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

// dialQUICBrowser opens a WebTransport session like a browser, announcing the
// id-proof nonce and capabilities in the CONNECT request.
func dialQUICBrowser(ctx context.Context, ln *quicListener, nonce []byte, caps ...string) (*quic.Conn, error) {
	rot, err := quic.NewCertRotator(log.Root())
	if err != nil {
		return nil, err
	}
	wd := &webtransport.Dialer{
		TLSClientConfig: quic.NewTLSConfig(rot),
		QUICConfig:      quicConfig,
	}
	protos := []string{strconv.Quote(quicNonceProto + hex.EncodeToString(nonce))}
	for _, c := range caps {
		protos = append(protos, strconv.Quote(c))
	}
	hdr := http.Header{}
	hdr.Set("WT-Available-Protocols", strings.Join(protos, ", "))
	_, sess, err := wd.Dial(ctx, "https://"+ln.Addr().String()+quicWTPath, hdr)
	if err != nil {
		return nil, err
	}
	str, err := sess.AcceptStream(ctx)
	if err != nil {
		sess.CloseWithError(0, "")
		return nil, err
	}
	return quic.NewConn(sess, str, nil, nil), nil
}

func TestQUICListenerAccept(t *testing.T) {
	ln := newTestQUICListener(t)
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The server opens the stream and writes first; the dialer reads.
	readCh := make(chan string, 1)
	go func() {
		conn, err := dialQUICBrowser(ctx, ln, make([]byte, quicNonceLen))
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
	var qh quic.CertHashEntry
	if err := srv.LocalNode().Node().Load(&qh); err != nil {
		t.Fatal(err)
	}
	if len(qh) != 2*sha256.Size {
		t.Fatalf("qh length %d, want %d", len(qh), 2*sha256.Size)
	}
}

// This test checks the browser side of the QUIC transport against a running
// Server. The server must offer its browser protocols in the encryption
// handshake, i.e. before the peer identity is known, because the chosen
// capabilities are reported to the browser in it.
func TestQUICTransport(t *testing.T) {
	// browserproto forwards the payload of every message it receives.
	msgCh := make(chan []string, 1)
	browserProto := Protocol{
		Name:         "browserproto",
		Version:      1,
		Length:       2,
		AllowBrowser: true,
		Run: func(p *Peer, rw MsgReadWriter) error {
			for {
				msg, err := rw.ReadMsg()
				if err != nil {
					return err
				}
				var payload []string
				if err := msg.Decode(&payload); err != nil {
					return err
				}
				msgCh <- payload
			}
		},
	}
	srv := &Server{
		Config: Config{
			PrivateKey:      newkey(),
			MaxPeers:        10,
			MaxBrowserPeers: 4,
			NoDial:          true,
			NoDiscovery:     true,
			ListenQUICAddr:  "127.0.0.1:0",
			Protocols: []Protocol{
				drainProtocol("nodeproto", false),
				browserProto,
			},
			Logger: testlog.Logger(t, log.LevelError),
		},
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// The dialer acts as a browser, announcing both protocols. Only the browser
	// protocol may be chosen by the server.
	nonce := make([]byte, quicNonceLen)
	rand.Read(nonce)
	conn, err := dialQUICBrowser(ctx, srv.quicListener, nonce, "browserproto/1", "nodeproto/1")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// The server sends the id-proof and the chosen capabilities in the
	// encryption handshake. The server key is recovered from the signature over
	// the nonce.
	var hs struct {
		Proof []byte
		Caps  []string
	}
	if err := rlp.NewStream(conn, 0).Decode(&hs); err != nil {
		t.Fatal(err)
	}
	remote, err := crypto.SigToPub(crypto.Keccak256([]byte("ENR-key-proof-v1"), nonce), hs.Proof)
	if err != nil {
		t.Fatal(err)
	}
	if !remote.Equal(&srv.PrivateKey.PublicKey) {
		t.Fatal("id-proof does not recover the server key")
	}
	if len(hs.Caps) != 1 || hs.Caps[0] != "browserproto/1" {
		t.Fatalf("chosen caps = %v, want [browserproto/1]", hs.Caps)
	}

	// Send a message to the browser protocol. QUIC always uses snappy, and the
	// single chosen protocol starts right after the base protocol messages.
	conn.SetSnappy(true)
	if err := Send(newQUICTransport(conn, nil), baseProtocolLength, []string{"hello", "quic"}); err != nil {
		t.Fatal(err)
	}
	select {
	case payload := <-msgCh:
		if len(payload) != 2 || payload[0] != "hello" || payload[1] != "quic" {
			t.Fatalf("payload mismatch: %v", payload)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for browserproto to receive the message")
	}
	if c := srv.PeerCount(); c != 1 {
		t.Fatalf("peer count = %d, want 1", c)
	}
}

func TestENRBrowserID(t *testing.T) {
	var r enr.Record
	r.Set(enr.QUIC(3333))
	id := enode.ID{8, 8, 4, 4, 2, 2, 5, 5}
	n, err := signBrowser(&r, id)
	if err != nil {
		t.Fatal("error signing record:", err)
	}
	if n.ID() != id {
		t.Fatal("ID not stored on record")
	}
	if !bytes.Equal(n.Record().Signature(), id[:]) {
		t.Fatal("signature is not equal to ID")
	}
	var qp enr.QUIC
	if err := n.Load(&qp); err != nil || qp != 3333 {
		t.Fatal("QUIC port not stored on signed record:", err)
	}
	t.Log(n.String())
}

func TestPenalizeBrowser(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"invalid message", DiscProtocolError, true},
		{"idle timeout", DiscReadTimeout, true},
		{"invalid subprotocol input", DiscSubprotocolError, true},
		{"clean single-shot", errProtocolReturned, false},
		{"quitting", DiscQuitting, false},
		{"remote requested", DiscRequested, false},
		{"network error reason", DiscNetworkError, false},
		{"dropped connection", io.EOF, false},
		{"plain error", errors.New("boom"), false},
		{"no error", nil, false},
	}
	for _, tt := range tests {
		if got := penalizeBrowser(tt.err); got != tt.want {
			t.Errorf("penalizeBrowser(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// drainProtocol returns a subprotocol which reads and discards all messages.
func drainProtocol(name string, browser bool) Protocol {
	return Protocol{
		Name:         name,
		Version:      1,
		Length:       2,
		AllowBrowser: browser,
		Run: func(p *Peer, rw MsgReadWriter) error {
			for {
				if _, err := rw.ReadMsg(); err != nil {
					return err
				}
			}
		},
	}
}
