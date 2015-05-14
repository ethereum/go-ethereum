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
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func startTestServer(t *testing.T, pf newPeerHook) *Server {
	server := &Server{
		Name:        "test",
		MaxPeers:    10,
		ListenAddr:  "127.0.0.1:0",
		PrivateKey:  newkey(),
		newPeerHook: pf,
		setupFunc: func(fd net.Conn, prv *ecdsa.PrivateKey, our *protoHandshake, dial *discover.Node, keepconn func(discover.NodeID) bool) (*conn, error) {
			id := randomID()
			if !keepconn(id) {
				return nil, DiscAlreadyConnected
			}
			rw := newRlpxFrameRW(fd, secrets{
				MAC:        zero16,
				AES:        zero16,
				IngressMAC: sha3.NewKeccak256(),
				EgressMAC:  sha3.NewKeccak256(),
			})
			return &conn{
				MsgReadWriter:  rw,
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
	srv.staticDial <- &discover.Node{IP: tcpAddr.IP, TCP: uint16(tcpAddr.Port)}

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
	srv.Broadcast("discard", 0, []string{"foo"})
	golden := unhex("66e94d166f0a2c3b884cfa59ca34")

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

// This test checks that connections are disconnected
// just after the encryption handshake when the server is
// at capacity.
//
// It also serves as a light-weight integration test.
func TestServerDisconnectAtCap(t *testing.T) {
	started := make(chan *Peer)
	srv := &Server{
		ListenAddr: "127.0.0.1:0",
		PrivateKey: newkey(),
		MaxPeers:   10,
		NoDial:     true,
		// This hook signals that the peer was actually started. We
		// need to wait for the peer to be started before dialing the
		// next connection to get a deterministic peer count.
		newPeerHook: func(p *Peer) { started <- p },
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	nconns := srv.MaxPeers + 1
	dialer := &net.Dialer{Deadline: time.Now().Add(3 * time.Second)}
	for i := 0; i < nconns; i++ {
		conn, err := dialer.Dial("tcp", srv.ListenAddr)
		if err != nil {
			t.Fatalf("conn %d: dial error: %v", i, err)
		}
		// Close the connection when the test ends, before
		// shutting down the server.
		defer conn.Close()
		// Run the handshakes just like a real peer would.
		key := newkey()
		hs := &protoHandshake{Version: baseProtocolVersion, ID: discover.PubkeyID(&key.PublicKey)}
		_, err = setupConn(conn, key, hs, srv.Self(), keepalways)
		if i == nconns-1 {
			// When handling the last connection, the server should
			// disconnect immediately instead of running the protocol
			// handshake.
			if err != DiscTooManyPeers {
				t.Errorf("conn %d: got error %q, expected %q", i, err, DiscTooManyPeers)
			}
		} else {
			// For all earlier connections, the handshake should go through.
			if err != nil {
				t.Fatalf("conn %d: unexpected error: %v", i, err)
			}
			// Wait for runPeer to be started.
			<-started
		}
	}
}

// Tests that static peers are (re)connected, and done so even above max peers.
func TestServerStaticPeers(t *testing.T) {
	// Create a test server with limited connection slots
	started := make(chan *Peer)
	server := &Server{
		ListenAddr:  "127.0.0.1:0",
		PrivateKey:  newkey(),
		MaxPeers:    3,
		newPeerHook: func(p *Peer) { started <- p },
		staticCycle: time.Second,
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	// Fill up all the slots on the server
	dialer := &net.Dialer{Deadline: time.Now().Add(3 * time.Second)}
	for i := 0; i < server.MaxPeers; i++ {
		// Establish a new connection
		conn, err := dialer.Dial("tcp", server.ListenAddr)
		if err != nil {
			t.Fatalf("conn %d: dial error: %v", i, err)
		}
		defer conn.Close()

		// Run the handshakes just like a real peer would, and wait for completion
		key := newkey()
		shake := &protoHandshake{Version: baseProtocolVersion, ID: discover.PubkeyID(&key.PublicKey)}
		if _, err = setupConn(conn, key, shake, server.Self(), keepalways); err != nil {
			t.Fatalf("conn %d: unexpected error: %v", i, err)
		}
		<-started
	}
	// Open a TCP listener to accept static connections
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to setup listener: %v", err)
	}
	defer listener.Close()

	connected := make(chan net.Conn)
	go func() {
		for i := 0; i < 3; i++ {
			conn, err := listener.Accept()
			if err == nil {
				connected <- conn
			}
		}
	}()
	// Inject a static node and wait for a remote dial, then redial, then nothing
	addr := listener.Addr().(*net.TCPAddr)
	static := &discover.Node{
		ID:  discover.PubkeyID(&newkey().PublicKey),
		IP:  addr.IP,
		TCP: uint16(addr.Port),
	}
	server.AddPeer(static)

	select {
	case conn := <-connected:
		// Close the first connection, expect redial
		conn.Close()

	case <-time.After(2 * server.staticCycle):
		t.Fatalf("remote dial timeout")
	}

	select {
	case conn := <-connected:
		// Keep the second connection, don't expect redial
		defer conn.Close()

	case <-time.After(2 * server.staticCycle):
		t.Fatalf("remote re-dial timeout")
	}

	select {
	case <-time.After(2 * server.staticCycle):
		// Timeout as no dial occurred

	case <-connected:
		t.Fatalf("connected node dialed")
	}
}

// Tests that trusted peers and can connect above max peer caps.
func TestServerTrustedPeers(t *testing.T) {

	// Create a trusted peer to accept connections from
	key := newkey()
	trusted := &discover.Node{
		ID: discover.PubkeyID(&key.PublicKey),
	}
	// Create a test server with limited connection slots
	started := make(chan *Peer)
	server := &Server{
		ListenAddr:   "127.0.0.1:0",
		PrivateKey:   newkey(),
		MaxPeers:     3,
		NoDial:       true,
		TrustedNodes: []*discover.Node{trusted},
		newPeerHook:  func(p *Peer) { started <- p },
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop()

	// Fill up all the slots on the server
	dialer := &net.Dialer{Deadline: time.Now().Add(3 * time.Second)}
	for i := 0; i < server.MaxPeers; i++ {
		// Establish a new connection
		conn, err := dialer.Dial("tcp", server.ListenAddr)
		if err != nil {
			t.Fatalf("conn %d: dial error: %v", i, err)
		}
		defer conn.Close()

		// Run the handshakes just like a real peer would, and wait for completion
		key := newkey()
		shake := &protoHandshake{Version: baseProtocolVersion, ID: discover.PubkeyID(&key.PublicKey)}
		if _, err = setupConn(conn, key, shake, server.Self(), keepalways); err != nil {
			t.Fatalf("conn %d: unexpected error: %v", i, err)
		}
		<-started
	}
	// Dial from the trusted peer, ensure connection is accepted
	conn, err := dialer.Dial("tcp", server.ListenAddr)
	if err != nil {
		t.Fatalf("trusted node: dial error: %v", err)
	}
	defer conn.Close()

	shake := &protoHandshake{Version: baseProtocolVersion, ID: trusted.ID}
	if _, err = setupConn(conn, key, shake, server.Self(), keepalways); err != nil {
		t.Fatalf("trusted node: unexpected error: %v", err)
	}
	select {
	case <-started:
		// Ok, trusted peer accepted

	case <-time.After(100 * time.Millisecond):
		t.Fatalf("trusted node timeout")
	}
}

// Tests that a failed dial will temporarily throttle a peer.
func TestServerMaxPendingDials(t *testing.T) {
	// Start a simple test server
	server := &Server{
		ListenAddr:      "127.0.0.1:0",
		PrivateKey:      newkey(),
		MaxPeers:        10,
		MaxPendingPeers: 1,
	}
	if err := server.Start(); err != nil {
		t.Fatal("failed to start test server: %v", err)
	}
	defer server.Stop()

	// Simulate two separate remote peers
	peers := make(chan *discover.Node, 2)
	conns := make(chan net.Conn, 2)
	for i := 0; i < 2; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listener %d: failed to setup: %v", i, err)
		}
		defer listener.Close()

		addr := listener.Addr().(*net.TCPAddr)
		peers <- &discover.Node{
			ID:  discover.PubkeyID(&newkey().PublicKey),
			IP:  addr.IP,
			TCP: uint16(addr.Port),
		}
		go func() {
			conn, err := listener.Accept()
			if err == nil {
				conns <- conn
			}
		}()
	}
	// Request a dial for both peers
	go func() {
		for i := 0; i < 2; i++ {
			server.staticDial <- <-peers // hack piggybacking the static implementation
		}
	}()

	// Make sure only one outbound connection goes through
	var conn net.Conn

	select {
	case conn = <-conns:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("first dial timeout")
	}
	select {
	case conn = <-conns:
		t.Fatalf("second dial completed prematurely")
	case <-time.After(100 * time.Millisecond):
	}
	// Finish the first dial, check the second
	conn.Close()
	select {
	case conn = <-conns:
		conn.Close()

	case <-time.After(100 * time.Millisecond):
		t.Fatalf("second dial timeout")
	}
}

func TestServerMaxPendingAccepts(t *testing.T) {
	// Start a test server and a peer sink for synchronization
	started := make(chan *Peer)
	server := &Server{
		ListenAddr:      "127.0.0.1:0",
		PrivateKey:      newkey(),
		MaxPeers:        10,
		MaxPendingPeers: 1,
		NoDial:          true,
		newPeerHook:     func(p *Peer) { started <- p },
	}
	if err := server.Start(); err != nil {
		t.Fatal("failed to start test server: %v", err)
	}
	defer server.Stop()

	// Try and connect to the server on multiple threads concurrently
	conns := make([]net.Conn, 2)
	for i := 0; i < 2; i++ {
		dialer := &net.Dialer{Deadline: time.Now().Add(3 * time.Second)}

		conn, err := dialer.Dial("tcp", server.ListenAddr)
		if err != nil {
			t.Fatalf("failed to dial server: %v", err)
		}
		conns[i] = conn
	}
	// Check that a handshake on the second doesn't pass
	go func() {
		key := newkey()
		shake := &protoHandshake{Version: baseProtocolVersion, ID: discover.PubkeyID(&key.PublicKey)}
		if _, err := setupConn(conns[1], key, shake, server.Self(), keepalways); err != nil {
			t.Fatalf("failed to run handshake: %v", err)
		}
	}()
	select {
	case <-started:
		t.Fatalf("handshake on second connection accepted")

	case <-time.After(time.Second):
	}
	// Shake on first, check that both go through
	go func() {
		key := newkey()
		shake := &protoHandshake{Version: baseProtocolVersion, ID: discover.PubkeyID(&key.PublicKey)}
		if _, err := setupConn(conns[0], key, shake, server.Self(), keepalways); err != nil {
			t.Fatalf("failed to run handshake: %v", err)
		}
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("peer %d: handshake timeout", i)
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

func keepalways(id discover.NodeID) bool {
	return true
}
