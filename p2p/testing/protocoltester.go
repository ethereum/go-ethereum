package testing

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
)

type ProtocolTester struct {
	*ProtocolSession
	network *simulations.Network
}

func NewProtocolTester(t *testing.T, id *adapters.NodeId, n int, run func(*p2p.Peer, p2p.MsgReadWriter) error) *ProtocolTester {
	services := adapters.Services{
		"test": func(id *adapters.NodeId, _ []byte) node.Service {
			return &testNode{run}
		},
		"mock": func(id *adapters.NodeId, _ []byte) node.Service {
			return newMockNode()
		},
	}
	adapter := adapters.NewSimAdapter(services)
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{})
	if _, err := net.NewNodeWithConfig(&adapters.NodeConfig{Id: id, Services: []string{"test"}}); err != nil {
		panic(err.Error())
	}
	if err := net.Start(id); err != nil {
		panic(err.Error())
	}

	node := net.GetNode(id).Node.(*adapters.SimNode)
	peers := make([]*adapters.NodeConfig, n)
	peerIDs := make([]*adapters.NodeId, n)
	for i := 0; i < n; i++ {
		peers[i] = adapters.RandomNodeConfig()
		peers[i].Services = []string{"mock"}
		peerIDs[i] = peers[i].Id
	}
	events := make(chan *p2p.PeerEvent, 1000)
	node.SubscribeEvents(events)
	ps := &ProtocolSession{
		Server:  node.Server(),
		Ids:     peerIDs,
		adapter: adapter,
		events:  events,
	}
	self := &ProtocolTester{
		ProtocolSession: ps,
		network:         net,
	}

	self.Connect(id, peers...)

	return self
}

func (self *ProtocolTester) Stop() error {
	return self.Server.Stop()
}

func (self *ProtocolTester) Connect(selfId *adapters.NodeId, peers ...*adapters.NodeConfig) {
	for _, peer := range peers {
		log.Trace(fmt.Sprintf("start node %v", peer.Id))
		if _, err := self.network.NewNodeWithConfig(peer); err != nil {
			panic(fmt.Sprintf("error starting peer %v: %v", peer.Id, err))
		}
		if err := self.network.Start(peer.Id); err != nil {
			panic(fmt.Sprintf("error starting peer %v: %v", peer.Id, err))
		}
		log.Trace(fmt.Sprintf("connect to %v", peer.Id))
		if err := self.network.Connect(selfId, peer.Id); err != nil {
			panic(fmt.Sprintf("error connecting to peer %v: %v", peer.Id, err))
		}
	}

}

// testNode wraps a protocol run function and implements the node.Service
// interface
type testNode struct {
	run func(*p2p.Peer, p2p.MsgReadWriter) error
}

func (t *testNode) Protocols() []p2p.Protocol {
	return []p2p.Protocol{{
		Length: 100,
		Run:    t.run,
	}}
}

func (t *testNode) APIs() []rpc.API {
	return nil
}

func (t *testNode) Start(server *p2p.Server) error {
	return nil
}

func (t *testNode) Stop() error {
	return nil
}

// mockNode is a testNode which doesn't actually run a protocol, instead
// exposing channels so that tests can manually trigger and expect certain
// messages
type mockNode struct {
	testNode

	trigger  chan *Trigger
	expect   chan *Expect
	err      chan error
	stop     chan struct{}
	stopOnce sync.Once
}

func newMockNode() *mockNode {
	mock := &mockNode{
		trigger: make(chan *Trigger),
		expect:  make(chan *Expect),
		err:     make(chan error),
		stop:    make(chan struct{}),
	}
	mock.testNode.run = mock.Run
	return mock
}

// Run is a protocol run function which just loops waiting for tests to
// instruct it to either trigger or expect a message from the peer
func (m *mockNode) Run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	for {
		select {
		case trig := <-m.trigger:
			m.err <- p2p.Send(rw, trig.Code, trig.Msg)
		case exp := <-m.expect:
			m.err <- p2p.ExpectMsg(rw, exp.Code, exp.Msg)
		case <-m.stop:
			return nil
		}
	}
}

func (m *mockNode) Trigger(trig *Trigger) error {
	m.trigger <- trig
	return <-m.err
}

func (m *mockNode) Expect(exp *Expect) error {
	m.expect <- exp
	return <-m.err
}

func (m *mockNode) Stop() error {
	m.stopOnce.Do(func() { close(m.stop) })
	return nil
}
