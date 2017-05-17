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

func (self *testConnect) Ch() <-chan time.Time {
	return self.ticker
}

func (self *testConnect) Stop() {
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
	addr := RandomAddr() // tested peers peer address
	to := NewKademlia(addr.OAddr, NewKadParams())
	pp := NewHive(params, to, nil) // hive

	return newBzzBaseTester(t, 1, addr, DiscoverySpec, pp.Run), pp
}

func TestRegisterAndConnect(t *testing.T) {
	params := NewHiveParams()
	s, pp := newHiveTester(t, params)
	defer s.Stop()

	id := s.Ids[0]
	raddr := NewAddrFromNodeId(id)

	ch := make(chan OverlayAddr)
	go func() {
		ch <- raddr
		close(ch)
	}()
	pp.Register(ch)

	// start the hive and wait for the connection
	tc := &testConnect{
		connectf: func(c string) error {
			s.Connect(adapters.NewNodeIdFromHex(c))
			return nil
		},
		ticker: make(chan time.Time),
	}
	pp.newTicker = func() hiveTicker { return tc }
	pp.Start(s.Server)
	defer pp.Stop()
	tc.ticker <- time.Now()
	// retrieve and broadcast
	ord := raddr.Over()[0] / 32
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
