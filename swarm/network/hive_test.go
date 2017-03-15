package network

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func init() {
	glog.SetV(logger.Detail)
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

func TestOverlayRegistration(t *testing.T) {
	// setup
	addr := RandomAddr()                     // tested peers peer address
	to := NewTestOverlay(addr.OverlayAddr()) // overlay topology driver
	pp := NewHive(NewHiveParams(), to)       // hive
	ct := BzzCodeMap(HiveMsgs...)            // bzz protocol code map
	services := func(p Peer) error {
		pp.Add(p)
		p.DisconnectHook(func(err error) {
			pp.Remove(p)
		})
		return nil
	}

	s := newBzzBaseTester(t, 1, addr, ct, services)
	id := s.Ids[0]
	raddr := NewPeerAddrFromNodeId(id)

	s.runHandshakes()

	// hive should have called the overlay
	if to.posMap[string(raddr.OverlayAddr())] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}

}

func TestRegisterAndConnect(t *testing.T) {
	// setup
	addr := RandomAddr()                     // tested peers peer address
	to := NewTestOverlay(addr.OverlayAddr()) // overlay topology driver
	pp := NewHive(NewHiveParams(), to)       // hive
	ct := BzzCodeMap(HiveMsgs...)            // bzz protocol code map
	services := func(p Peer) error {
		pp.Add(p)
		p.DisconnectHook(func(error) {
			pp.Remove(p)
		})
		return nil
	}

	s := newBzzBaseTester(t, 1, addr, ct, services)

	id := s.Ids[0]
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

	// retrieve and broadcast
	glog.V(6).Infof("check peer requests for %v", id)
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
	// s.TestExchanges(p2ptest.Exchange{
	// 	Label: "SubPeersMsg message outgoing",
	// 	Expects: []p2ptest.Expect{
	// 		p2ptest.Expect{
	// 			Code: 3,
	// 			Msg:  &SubPeersMsg{ProxLimit: 0, MinProxBinSize: 8},
	// 			Peer: id,
	// 		},
	// 	},
	// })
}
