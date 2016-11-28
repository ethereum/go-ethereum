package network

import (
	"fmt"
	"strings"
	"sync"

	// "github.com/ethereum/go-ethereum/p2p/adapters"
	// "github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/logger/glog"
)

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
	addr := na.OverlayAddr()
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
	addr := n.OverlayAddr()
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
	addr := n.OverlayAddr()
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
					return
				}
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
				glog.V(6).Infof("node %v is off", offs[0])
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
			addr := n.OverlayAddr()
			row = append(row, fmt.Sprintf("%x", addr[:4]))
		}
		row = append(row, "|")
		nas = self.off(po)
		offs = len(nas)
		for _, na := range nas {
			addr := na.OverlayAddr()
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
