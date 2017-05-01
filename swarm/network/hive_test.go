package network

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

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

func newHiveTester(t *testing.T, params *HiveParams) (*bzzTester, *Hive) {
	// setup
	addr := RandomAddr()                     // tested peers peer address
	to := NewTestOverlay(addr.OverlayAddr()) // overlay topology drive
	pp := NewHive(params, to)                // hive

	ct := BzzCodeMap(DiscoveryMsgs...) // bzz protocol code map
	services := func(p Peer) error {
		pp.Add(p)
		p.DisconnectHook(func(err error) {
			pp.Remove(p)
		})
		return nil
	}

	return newBzzBaseTester(t, 1, addr, ct, services), pp
}

func TestOverlayRegistration(t *testing.T) {
	params := NewHiveParams()
	params.Discovery = false
	s, pp := newHiveTester(t, params)
	defer s.Stop()

	id := s.Ids[0]
	raddr := NewPeerAddrFromNodeId(id)

	s.runHandshakes()

	// hive should have called the overlay
	if pp.Overlay.(*testOverlay).posMap[string(raddr.OverlayAddr())] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}

}

func TestRegisterAndConnect(t *testing.T) {
	params := NewHiveParams()
	s, pp := newHiveTester(t, params)
	defer s.Stop()

	id := s.Ids[0]
	raddr := NewPeerAddrFromNodeId(id)

	pp.Register(raddr)

	// start the hive and wait for the connection
	tc := &testConnect{
		connectf: func(c string) error {
			s.Connect(adapters.NewNodeIdFromHex(c))
			return nil
		},
		ticker: make(chan time.Time),
	}
	pp.Start(s, tc.ping)
	defer pp.Stop()
	tc.ticker <- time.Now()

	s.runHandshakes()

	if pp.Overlay.(*testOverlay).posMap[string(raddr.OverlayAddr())] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}

	// retrieve and broadcast
	ord := order(raddr.OverlayAddr())
	o := 0
	if ord == 0 {
		o = 1
	}
	s.TestExchanges(p2ptest.Exchange{
		Label: "getPeersMsg message",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg:  &getPeersMsg{uint8(o), 5},
				Peer: id,
			},
		},
	})
}
