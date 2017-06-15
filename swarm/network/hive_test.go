package network

import (
	"testing"
	"time"

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func newHiveTester(t *testing.T, params *HiveParams) (*bzzTester, *Hive) {
	// setup
	addr := RandomAddr() // tested peers peer address
	to := NewKademlia(addr.OAddr, NewKadParams())
	pp := NewHive(params, to, nil) // hive

	return newBzzBaseTester(t, 1, addr, DiscoverySpec, pp.Run), pp
}

func TestRegisterAndConnect(t *testing.T) {
	//t.Skip("deadlocked")
	params := NewHiveParams()
	s, pp := newHiveTester(t, params)
	defer s.Stop()

	id := s.IDs[0]
	raddr := NewAddrFromNodeID(id)

	ch := make(chan OverlayAddr)
	go func() {
		ch <- raddr
		close(ch)
	}()
	pp.Register(ch)

	// start the hive and wait for the connection
	tick := make(chan time.Time)
	pp.tick = tick
	pp.Start(s.Server)
	defer pp.Stop()
	tick <- time.Now()
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
				Code: 2,
				Msg:  &subPeersMsg{0},
				Peer: id,
			},
			p2ptest.Expect{
				Code: 1,
				Msg:  &getPeersMsg{uint8(o), 5},
				Peer: id,
			},
		},
	})
}
