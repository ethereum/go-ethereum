package p2p

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func startTestServer(t *testing.T, pf peerFunc) *Server {
	server := &Server{
		Identity:    &peerId{},
		MaxPeers:    10,
		ListenAddr:  "127.0.0.1:0",
		newPeerFunc: pf,
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
	srv := startTestServer(t, func(srv *Server, conn net.Conn, dialAddr *peerAddr) *Peer {
		if conn == nil {
			t.Error("peer func called with nil conn")
		}
		if dialAddr != nil {
			t.Error("peer func called with non-nil dialAddr")
		}
		peer := newPeer(conn, nil, dialAddr)
		connected <- peer
		return peer
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
		if peer.conn.LocalAddr().String() != conn.RemoteAddr().String() {
			t.Errorf("peer started with wrong conn: got %v, want %v",
				peer.conn.LocalAddr(), conn.RemoteAddr())
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not accept within one second")
	}
}

func TestServerDial(t *testing.T) {
	defer testlog(t).detach()

	// run a fake TCP server to handle the connection.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not setup listener: %v")
	}
	defer listener.Close()
	accepted := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Error("acccept error:", err)
		}
		conn.Close()
		accepted <- conn
	}()

	// start the test server
	connected := make(chan *Peer)
	srv := startTestServer(t, func(srv *Server, conn net.Conn, dialAddr *peerAddr) *Peer {
		if conn == nil {
			t.Error("peer func called with nil conn")
		}
		peer := newPeer(conn, nil, dialAddr)
		connected <- peer
		return peer
	})
	defer close(connected)
	defer srv.Stop()

	// tell the server to connect.
	connAddr := newPeerAddr(listener.Addr(), nil)
	srv.peerConnect <- connAddr

	select {
	case conn := <-accepted:
		select {
		case peer := <-connected:
			if peer.conn.RemoteAddr().String() != conn.LocalAddr().String() {
				t.Errorf("peer started with wrong conn: got %v, want %v",
					peer.conn.RemoteAddr(), conn.LocalAddr())
			}
			if peer.dialAddr != connAddr {
				t.Errorf("peer started with wrong dialAddr: got %v, want %v",
					peer.dialAddr, connAddr)
			}
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
	srv := startTestServer(t, func(srv *Server, c net.Conn, dialAddr *peerAddr) *Peer {
		peer := newPeer(c, []Protocol{discard}, dialAddr)
		peer.startSubprotocols([]Cap{discard.cap()})
		connected.Done()
		return peer
	})
	defer srv.Stop()

	// dial a bunch of conns
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
