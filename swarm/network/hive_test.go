package network

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	// "github.com/ethereum/go-ethereum/p2p/adapters"
	// "github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func init() {
	glog.SetV(6)
	glog.SetToStderr(true)
}

const orders = 8

type testOverlay struct {
	mu     sync.Mutex
	addr   []byte
	pos    [][]*testNodeAddr
	posMap map[string]*testNodeAddr
}

type testNodeAddr struct {
	NodeAddr
	Node Node
}

func (self *testOverlay) Register(na NodeAddr) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.register(na)
}

func (self *testOverlay) register(na NodeAddr) error {
	tna := &testNodeAddr{NodeAddr: na}
	addr := na.RemoteOverlayAddr()
	self.posMap[string(addr)] = tna
	o := order(addr)
	glog.V(6).Infof("PO: %v, orders: %v", o, orders)
	self.pos[o] = append(self.pos[o], tna)
	return nil
}

func order(addr []byte) int {
	return int(addr[0]) / 32
}

func (self *testOverlay) On(n Node) (Node, error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	addr := n.RemoteOverlayAddr()
	na := self.posMap[string(addr)]
	if na == nil {
		self.register(n)
		na = self.posMap[string(addr)]
	} else if na.Node != nil {
		return nil, nil
	}
	glog.V(6).Infof("Online: %v", fmt.Sprintf("%x", addr[:4]))
	na.Node = n
	o := order(addr)
	ons := self.on(self.pos[o])
	if len(ons) > 2 {
		return ons[0], nil
	}
	return nil, nil
}

func (self *testOverlay) Off(n Node) {
	self.mu.Lock()
	defer self.mu.Unlock()
	addr := n.RemoteOverlayAddr()
	na := self.posMap[string(addr)]
	if na == nil {
		return
	}
	na.Node = nil
}

// caller must hold the lock
func (self *testOverlay) on(po []*testNodeAddr) (nodes []Node) {
	for _, na := range po {
		if na.Node != nil {
			nodes = append(nodes, na.Node)
		}
	}
	return nodes
}

// caller must hold the lock
func (self *testOverlay) off(po []*testNodeAddr) (nas []NodeAddr) {
	for _, na := range po {
		if na.Node == nil {
			nas = append(nas, NodeAddr(na))
		}
	}
	return nas
}

func (self *testOverlay) EachNode(base []byte, o int, f func(Node) bool) {
	if base == nil {
		base = self.addr
	}
	for i := o; i < len(self.pos); i++ {
		for _, na := range self.pos[i] {
			if na.Node != nil {
				if !f(na.Node) {
					glog.V(6).Infof("executed last time")
					return
				}
				glog.V(6).Infof("executed...")
			}
		}
	}
}

func (self *testOverlay) EachNodeAddr(base []byte, o int, f func(NodeAddr) bool) {
	if base == nil {
		base = self.addr
	}
	for i := o; i < len(self.pos); i++ {
		for _, na := range self.pos[i] {
			if !f(na) {
				return
			}
		}
	}
}

func (self *testOverlay) SuggestNodeAddr() NodeAddr {
	self.mu.Lock()
	defer self.mu.Unlock()
	for _, po := range self.pos {
		ons := self.on(po)
		if len(ons) < 2 {
			offs := self.off(po)
			if len(offs) > 0 {
				return offs[0]
			}
		}
	}
	return nil
}

func (self *testOverlay) SuggestOrder() int {
	self.mu.Lock()
	defer self.mu.Unlock()
	for o, po := range self.pos {
		off := self.off(po)
		if len(off) < 5 {
			glog.V(6).Infof("suggest PO%02d / %v", o, len(self.pos)-1)
			return o
		}
	}
	return 256

}

func (self *testOverlay) Info() string {
	self.mu.Lock()
	defer self.mu.Unlock()
	var t []string
	var ons, offs int
	var ns []Node
	var nas []NodeAddr
	for o, po := range self.pos {
		var row []string
		ns = self.on(po)
		ons = len(ns)
		for _, n := range ns {
			addr := n.RemoteOverlayAddr()
			row = append(row, fmt.Sprintf("%x", addr[:4]))
		}
		row = append(row, "|")
		nas = self.off(po)
		offs = len(nas)
		for _, na := range nas {
			addr := na.RemoteOverlayAddr()
			row = append(row, fmt.Sprintf("%x", addr[:4]))
		}
		t = append(t, fmt.Sprintf("%v: (%v/%v) %v", o, ons, offs, strings.Join(row, " ")))
	}
	return strings.Join(t, "\n")
}

func NewTestOverlay(addr []byte) *testOverlay {
	return &testOverlay{
		addr:   addr,
		posMap: make(map[string]*testNodeAddr),
		pos:    make([][]*testNodeAddr, orders),
	}
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

func newBzzHiveTester(t *testing.T, n int, addr *peerAddr, pp PeerPool, ct *protocols.CodeMap, services func(Node) error) *bzzTester {
	s := p2ptest.NewProtocolTester(t, NodeID(addr), n, newTestBzzProtocol(addr, pp, ct, services))
	return &bzzTester{
		addr:            addr,
		flushCode:       3,
		ExchangeSession: s,
	}
}

func TestOverlayRegistration(t *testing.T) {
	// setup
	addr := randomAddr()                           // tested peers peer address
	to := NewTestOverlay(addr.RemoteOverlayAddr()) // overlay topology driver
	pp := NewHive(NewHiveParams(), to)             // hive
	ct := bzzCodeMap(hiveMsgs...)                  // bzz protocol code map
	s := newBzzHiveTester(t, 1, addr, pp, ct, nil)

	// connect to the other peer
	id := s.IDs[0]
	raddr := nodeID2addr(id)
	s.runHandshakes()

	// hive should have called the overlay
	if to.posMap[string(raddr.OverlayAddr)] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}
}

func TestRegisterAndConnect(t *testing.T) {
	addr := randomAddr()
	to := NewTestOverlay(addr.RemoteOverlayAddr())
	pp := NewHive(NewHiveParams(), to)
	ct := bzzCodeMap(hiveMsgs...)
	s := newBzzHiveTester(t, 0, addr, pp, ct, nil)

	// register the node with the peerPool
	id := p2ptest.RandomNodeID()
	s.StartNode(id)
	raddr := nodeID2addr(id)
	// raddr.OverlayAddr[0] = 66
	pp.Register(raddr)
	glog.V(5).Infof("%v", pp.Info())
	// start the hive and wait for the connection
	tc := &testConnect{
		connectf: func(c string) error {
			s.Connect(hexToNodeID(c))
			return nil
		},
		ticker: make(chan time.Time),
	}
	pp.Start(tc.connect, tc.ping)
	tc.ticker <- time.Now()
	s.runHandshakes()
	if to.posMap[string(raddr.OverlayAddr)] == nil {
		t.Fatalf("Overlay#On not called on new peer")
	}
	glog.V(6).Infof("check peer requests for %v", id)
	// tc.ticker <- time.Now()

	// shakeHands(s, addr, id)
	// s.Flush(int(ct.Length())-1, 0)
	// time.Sleep(3)
	ord := order(raddr.RemoteOverlayAddr())
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
		// 		Msg:  &peersMsg{[]*peerAddr{randomAddr()}},
		// 		Peer: 0,
		// 	},
		// },
	})
}
