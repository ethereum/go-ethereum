package network

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/adapters"
	// "github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func init() {
	glog.SetV(6)
	glog.SetToStderr(true)
}

type testConnect struct {
	mu       sync.Mutex
	conns    []string
	connectf func(c string) error
	ticker   chan time.Time
}

func (self *testConnect) ping() <-chan time.Time {
	return self.ticker
}

func (self *testConnect) connect(na string) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.conns = append(self.conns, na)
	self.connectf(na)
	return nil
}

func newBzzHiveTester(t *testing.T, n int, addr *peerAddr, pp PeerPool, ct *protocols.CodeMap, services func(Peer) error) *bzzTester {
	s := p2ptest.NewProtocolTester(t, NodeId(addr), n, newTestBzzProtocol(addr, pp, ct, services))
	return &bzzTester{
		addr:            addr,
		flushCode:       3,
		ExchangeSession: s,
	}
}

func TestOverlayRegistration(t *testing.T) {
	// setup
	addr := RandomAddr()                     // tested peers peer address
	to := NewTestOverlay(addr.OverlayAddr()) // overlay topology driver
	pp := NewHive(NewHiveParams(), to)       // hive
	ct := BzzCodeMap(HiveMsgs...)            // bzz protocol code map
	s := newBzzHiveTester(t, 1, addr, pp, ct, nil)

	// connect to the other peer
	id := s.Ids[0]
	raddr := NewPeerAddrFromNodeId(id)
	s.runHandshakes()

	// hive should have called the overlay
	if to.posMap[string(raddr.OverlayAddr())] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}
}

func TestRegisterAndConnect(t *testing.T) {
	addr := RandomAddr()
	to := NewTestOverlay(addr.OverlayAddr())
	pp := NewHive(NewHiveParams(), to)
	ct := BzzCodeMap(HiveMsgs...)
	s := newBzzHiveTester(t, 0, addr, pp, ct, nil)

	// register the node with the peerPool
	id := p2ptest.RandomNodeId()
	s.Start(id)
	raddr := NewPeerAddrFromNodeId(id)
	pp.Register(raddr)
	glog.V(5).Infof("%v", pp)
	// start the hive and wait for the connection
	tc := &testConnect{
		connectf: func(c string) error {
			s.Connect(adapters.NewNodeIdFromHex(c))
			return nil
		},
		ticker: make(chan time.Time),
	}
	pp.Start(tc.connect, tc.ping)
	tc.ticker <- time.Now()
	s.runHandshakes()
	if to.posMap[string(raddr.OverlayAddr())] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}
	glog.V(6).Infof("check peer requests for %v", id)
	// tc.ticker <- time.Now()

	// shakeHands(s, addr, id)
	// s.Flush(int(ct.Length())-1, 0)
	// time.Sleep(3)
	ord := order(raddr.OverlayAddr())
	o := 0
	if ord == 0 {
		o = 1
	}
	s.TestExchanges(p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg:  &getPeersMsg{uint(o), 5},
				Peer: id,
			},
		},
		// Triggers: []p2ptest.Trigger{
		// 	p2ptest.Trigger{
		// 		Code: 1,
		// 		Msg:  &getPeersMsg{0, 1},
		// 		Peer: 0,
		// 	},
		// },
		// Expects: []p2ptest.Expect{
		// 	p2ptest.Expect{
		// 		Code: 1,
		// 		Msg:  &peersMsg{[]*peerAddr{RandomAddr()}},
		// 		Peer: 0,
		// 	},
		// },
	})
}
