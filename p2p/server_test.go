package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func startTestServer(t *testing.T, pf newPeerHook) *Server {
	server := &Server{
		Name:        "test",
		MaxPeers:    10,
		ListenAddr:  "127.0.0.1:0",
		PrivateKey:  newkey(),
		newPeerHook: pf,
		setupFunc: func(fd net.Conn, prv *ecdsa.PrivateKey, our *protoHandshake, dial *discover.Node) (*conn, error) {
			id := randomID()
			return &conn{
				frameRW:        newFrameRW(fd, msgWriteTimeout),
				protoHandshake: &protoHandshake{ID: id, Version: baseProtocolVersion},
			}, nil
		},
	}
	if err := server.Start(); err != nil {
		t.Fatalf("Could not start server: %v", err)
	}
	return server
}

func TestServerListen(t *testing.T) {
	defer testlog(t).detach()

	// start the test server
	connected := make(chan *Peer)
	srv := startTestServer(t, func(p *Peer) {
		if p == nil {
			t.Error("peer func called with nil conn")
		}
		connected <- p
	})
	defer close(connected)
	defer srv.Stop()

	// dial the test server
	conn, err := net.DialTimeout("tcp", srv.ListenAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	defer conn.Close()

	select {
	case peer := <-connected:
		if peer.LocalAddr().String() != conn.RemoteAddr().String() {
			t.Errorf("peer started with wrong conn: got %v, want %v",
				peer.LocalAddr(), conn.RemoteAddr())
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not accept within one second")
	}
}

func TestServerDial(t *testing.T) {
	defer testlog(t).detach()

	// run a one-shot TCP server to handle the connection.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not setup listener: %v")
	}
	defer listener.Close()
	accepted := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Error("accept error:", err)
			return
		}
		conn.Close()
		accepted <- conn
	}()

	// start the server
	connected := make(chan *Peer)
	srv := startTestServer(t, func(p *Peer) { connected <- p })
	defer close(connected)
	defer srv.Stop()

	// tell the server to connect
	tcpAddr := listener.Addr().(*net.TCPAddr)
	srv.SuggestPeer(&discover.Node{IP: tcpAddr.IP, TCPPort: tcpAddr.Port})

	select {
	case conn := <-accepted:
		select {
		case peer := <-connected:
			if peer.RemoteAddr().String() != conn.LocalAddr().String() {
				t.Errorf("peer started with wrong conn: got %v, want %v",
					peer.RemoteAddr(), conn.LocalAddr())
			}
			// TODO: validate more fields
		case <-time.After(1 * time.Second):
			t.Error("server did not launch peer within one second")
		}

	case <-time.After(1 * time.Second):
		t.Error("server did not connect within one second")
	}
}

func TestServerBroadcast(t *testing.T) {
	defer testlog(t).detach()

	var connected sync.WaitGroup
	srv := startTestServer(t, func(p *Peer) {
		p.running = matchProtocols([]Protocol{discard}, []Cap{discard.cap()}, p.rw)
		connected.Done()
	})
	defer srv.Stop()

	// create a few peers
	var conns = make([]net.Conn, 8)
	connected.Add(len(conns))
	deadline := time.Now().Add(3 * time.Second)
	dialer := &net.Dialer{Deadline: deadline}
	for i := range conns {
		conn, err := dialer.Dial("tcp", srv.ListenAddr)
		if err != nil {
			t.Fatalf("conn %d: dial error: %v", i, err)
		}
		defer conn.Close()
		conn.SetDeadline(deadline)
		conns[i] = conn
	}
	connected.Wait()

	// broadcast one message
	srv.Broadcast("discard", 0, "foo")
	goldbuf := new(bytes.Buffer)
	writeMsg(goldbuf, NewMsg(16, "foo"))
	golden := goldbuf.Bytes()

	// check that the message has been written everywhere
	for i, conn := range conns {
		buf := make([]byte, len(golden))
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Errorf("conn %d: read error: %v", i, err)
		} else if !bytes.Equal(buf, golden) {
			t.Errorf("conn %d: msg mismatch\ngot:  %x\nwant: %x", i, buf, golden)
		}
	}
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}

func randomID() (id discover.NodeID) {
	for i := range id {
		id[i] = byte(rand.Intn(255))
	}
	return id
}
