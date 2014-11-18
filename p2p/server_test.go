package p2p

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

type TestNetwork struct {
	connections map[string]*TestNetworkConnection
	dialer      Dialer
	maxinbound  int
}

func NewTestNetwork(maxinbound int) *TestNetwork {
	connections := make(map[string]*TestNetworkConnection)
	return &TestNetwork{
		connections: connections,
		dialer:      &TestDialer{connections},
		maxinbound:  maxinbound,
	}
}

func (self *TestNetwork) Dialer(addr net.Addr) (Dialer, error) {
	return self.dialer, nil
}

func (self *TestNetwork) Listener(addr net.Addr) (net.Listener, error) {
	return &TestListener{
		connections: self.connections,
		addr:        addr,
		max:         self.maxinbound,
		close:       make(chan struct{}),
	}, nil
}

func (self *TestNetwork) Start() error {
	return nil
}

func (self *TestNetwork) NewAddr(string, int) (addr net.Addr, err error) {
	return
}

func (self *TestNetwork) ParseAddr(string) (addr net.Addr, err error) {
	return
}

type TestAddr struct {
	name string
}

func (self *TestAddr) String() string {
	return self.name
}

func (*TestAddr) Network() string {
	return "test"
}

type TestDialer struct {
	connections map[string]*TestNetworkConnection
}

func (self *TestDialer) Dial(network string, addr string) (conn net.Conn, err error) {
	address := &TestAddr{addr}
	tconn := NewTestNetworkConnection(address)
	self.connections[addr] = tconn
	conn = net.Conn(tconn)
	return
}

type TestListener struct {
	connections map[string]*TestNetworkConnection
	addr        net.Addr
	max         int
	i           int
	close       chan struct{}
}

func (self *TestListener) Accept() (net.Conn, error) {
	self.i++
	if self.i > self.max {
		<-self.close
		return nil, io.EOF
	}
	addr := &TestAddr{fmt.Sprintf("inboundpeer-%d", self.i)}
	tconn := NewTestNetworkConnection(addr)
	key := tconn.RemoteAddr().String()
	self.connections[key] = tconn
	fmt.Printf("accepted connection from: %v \n", addr)
	return tconn, nil
}

func (self *TestListener) Close() error {
	close(self.close)
	return nil
}

func (self *TestListener) Addr() net.Addr {
	return self.addr
}

type TestNetworkConnection struct {
	in      chan []byte
	close   chan struct{}
	current []byte
	Out     [][]byte
	addr    net.Addr
}

func NewTestNetworkConnection(addr net.Addr) *TestNetworkConnection {
	return &TestNetworkConnection{
		in:      make(chan []byte),
		close:   make(chan struct{}),
		current: []byte{},
		Out:     [][]byte{},
		addr:    addr,
	}
}

func (self *TestNetworkConnection) In(latency time.Duration, packets ...[]byte) {
	time.Sleep(latency)
	for _, s := range packets {
		self.in <- s
	}
}

func (self *TestNetworkConnection) Read(buff []byte) (n int, err error) {
	if len(self.current) == 0 {
		var ok bool
		select {
		case self.current, ok = <-self.in:
			if !ok {
				return 0, io.EOF
			}
		case <-self.close:
			return 0, io.EOF
		}
	}
	length := len(self.current)
	if length > len(buff) {
		copy(buff[:], self.current[:len(buff)])
		self.current = self.current[len(buff):]
		return len(buff), nil
	} else {
		copy(buff[:length], self.current[:])
		self.current = []byte{}
		return length, io.EOF
	}
}

func (self *TestNetworkConnection) Write(buff []byte) (n int, err error) {
	self.Out = append(self.Out, buff)
	fmt.Printf("net write(%d): %x\n", len(self.Out), buff)
	return len(buff), nil
}

func (self *TestNetworkConnection) Close() error {
	close(self.close)
	return nil
}

func (self *TestNetworkConnection) LocalAddr() (addr net.Addr) {
	return
}

func (self *TestNetworkConnection) RemoteAddr() (addr net.Addr) {
	return self.addr
}

func (self *TestNetworkConnection) SetDeadline(t time.Time) (err error) {
	return
}

func (self *TestNetworkConnection) SetReadDeadline(t time.Time) (err error) {
	return
}

func (self *TestNetworkConnection) SetWriteDeadline(t time.Time) (err error) {
	return
}

func SetupTestServer(handlers Handlers) (network *TestNetwork, server *Server) {
	network = NewTestNetwork(1)
	addr := &TestAddr{"test:30303"}
	identity := NewSimpleClientIdentity("clientIdentifier", "version", "customIdentifier", "pubkey")
	maxPeers := 2
	if handlers == nil {
		handlers = make(Handlers)
	}
	blackist := NewBlacklist()
	server = New(network, addr, identity, handlers, maxPeers, blackist)
	fmt.Println(server.identity.Pubkey())
	return
}

func TestServerListener(t *testing.T) {
	t.SkipNow()

	network, server := SetupTestServer(nil)
	server.Start(true, false)
	time.Sleep(10 * time.Millisecond)
	server.Stop()
	peer1, ok := network.connections["inboundpeer-1"]
	if !ok {
		t.Error("not found inbound peer 1")
	} else {
		if len(peer1.Out) != 2 {
			t.Errorf("wrong number of writes to peer 1: got %d, want %d", len(peer1.Out), 2)
		}
	}
}

func TestServerDialer(t *testing.T) {
	network, server := SetupTestServer(nil)
	server.Start(false, true)
	server.peerConnect <- &TestAddr{"outboundpeer-1"}
	time.Sleep(10 * time.Millisecond)
	server.Stop()
	peer1, ok := network.connections["outboundpeer-1"]
	if !ok {
		t.Error("not found outbound peer 1")
	} else {
		if len(peer1.Out) != 2 {
			t.Errorf("wrong number of writes to peer 1: got %d, want %d", len(peer1.Out), 2)
		}
	}
}

// func TestServerBroadcast(t *testing.T) {
// 	handlers := make(Handlers)
// 	testProtocol := &TestProtocol{Msgs: []*Msg{}}
// 	handlers["aaa"] = func(p *Peer) Protocol { return testProtocol }
// 	network, server := SetupTestServer(handlers)
// 	server.Start(true, true)
// 	server.peerConnect <- &TestAddr{"outboundpeer-1"}
// 	time.Sleep(10 * time.Millisecond)
// 	msg := NewMsg(0)
// 	server.Broadcast("", msg)
// 	packet := Packet(0, 0)
// 	time.Sleep(10 * time.Millisecond)
// 	server.Stop()
// 	peer1, ok := network.connections["outboundpeer-1"]
// 	if !ok {
// 		t.Error("not found outbound peer 1")
// 	} else {
// 		fmt.Printf("out: %v\n", peer1.Out)
// 		if len(peer1.Out) != 3 {
// 			t.Errorf("not enough messages sent to peer 1: %v ", len(peer1.Out))
// 		} else {
// 			if bytes.Compare(peer1.Out[1], packet) != 0 {
// 				t.Errorf("incorrect broadcast packet %v != %v", peer1.Out[1], packet)
// 			}
// 		}
// 	}
// 	peer2, ok := network.connections["inboundpeer-1"]
// 	if !ok {
// 		t.Error("not found inbound peer 2")
// 	} else {
// 		fmt.Printf("out: %v\n", peer2.Out)
// 		if len(peer1.Out) != 3 {
// 			t.Errorf("not enough messages sent to peer 2: %v ", len(peer2.Out))
// 		} else {
// 			if bytes.Compare(peer2.Out[1], packet) != 0 {
// 				t.Errorf("incorrect broadcast packet %v != %v", peer2.Out[1], packet)
// 			}
// 		}
// 	}
// }

func TestServerPeersMessage(t *testing.T) {
	t.SkipNow()
	_, server := SetupTestServer(nil)
	server.Start(true, true)
	defer server.Stop()
	server.peerConnect <- &TestAddr{"outboundpeer-1"}
	time.Sleep(2000 * time.Millisecond)

	pl := server.encodedPeerList()
	if pl == nil {
		t.Errorf("expect non-nil peer list")
	}
	if c := server.PeerCount(); c != 2 {
		t.Errorf("expect 2 peers, got %v", c)
	}
}
