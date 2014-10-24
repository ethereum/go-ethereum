package p2p

import (
	"bytes"
	"fmt"
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
}

func (self *TestListener) Accept() (conn net.Conn, err error) {
	self.i++
	if self.i > self.max {
		err = fmt.Errorf("no more")
	} else {
		addr := &TestAddr{fmt.Sprintf("inboundpeer-%d", self.i)}
		tconn := NewTestNetworkConnection(addr)
		key := tconn.RemoteAddr().String()
		self.connections[key] = tconn
		conn = net.Conn(tconn)
		fmt.Printf("accepted connection from: %v \n", addr)
	}
	return
}

func (self *TestListener) Close() error {
	return nil
}

func (self *TestListener) Addr() net.Addr {
	return self.addr
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
	network, server := SetupTestServer(nil)
	server.Start(true, false)
	time.Sleep(10 * time.Millisecond)
	server.Stop()
	peer1, ok := network.connections["inboundpeer-1"]
	if !ok {
		t.Error("not found inbound peer 1")
	} else {
		fmt.Printf("out: %v\n", peer1.Out)
		if len(peer1.Out) != 2 {
			t.Errorf("not enough messages sent to peer 1: %v ", len(peer1.Out))
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
		fmt.Printf("out: %v\n", peer1.Out)
		if len(peer1.Out) != 2 {
			t.Errorf("not enough messages sent to peer 1: %v ", len(peer1.Out))
		}
	}
}

func TestServerBroadcast(t *testing.T) {
	handlers := make(Handlers)
	testProtocol := &TestProtocol{Msgs: []*Msg{}}
	handlers["aaa"] = func(p *Peer) Protocol { return testProtocol }
	network, server := SetupTestServer(handlers)
	server.Start(true, true)
	server.peerConnect <- &TestAddr{"outboundpeer-1"}
	time.Sleep(10 * time.Millisecond)
	msg, _ := NewMsg(0)
	server.Broadcast("", msg)
	packet := Packet(0, 0)
	time.Sleep(10 * time.Millisecond)
	server.Stop()
	peer1, ok := network.connections["outboundpeer-1"]
	if !ok {
		t.Error("not found outbound peer 1")
	} else {
		fmt.Printf("out: %v\n", peer1.Out)
		if len(peer1.Out) != 3 {
			t.Errorf("not enough messages sent to peer 1: %v ", len(peer1.Out))
		} else {
			if bytes.Compare(peer1.Out[1], packet) != 0 {
				t.Errorf("incorrect broadcast packet %v != %v", peer1.Out[1], packet)
			}
		}
	}
	peer2, ok := network.connections["inboundpeer-1"]
	if !ok {
		t.Error("not found inbound peer 2")
	} else {
		fmt.Printf("out: %v\n", peer2.Out)
		if len(peer1.Out) != 3 {
			t.Errorf("not enough messages sent to peer 2: %v ", len(peer2.Out))
		} else {
			if bytes.Compare(peer2.Out[1], packet) != 0 {
				t.Errorf("incorrect broadcast packet %v != %v", peer2.Out[1], packet)
			}
		}
	}
}

func TestServerPeersMessage(t *testing.T) {
	handlers := make(Handlers)
	_, server := SetupTestServer(handlers)
	server.Start(true, true)
	defer server.Stop()
	server.peerConnect <- &TestAddr{"outboundpeer-1"}
	time.Sleep(10 * time.Millisecond)
	peersMsg, err := server.PeersMessage()
	fmt.Println(peersMsg)
	if err != nil {
		t.Errorf("expect no error, got %v", err)
	}
	if c := server.PeerCount(); c != 2 {
		t.Errorf("expect 2 peers, got %v", c)
	}
}
