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
	pos    [][]*testPeerAddr
	posMap map[string]*testPeerAddr
}

type testPeerAddr struct {
	PeerAddr
	Peer Peer
}

func (self *testOverlay) Register(nas ...PeerAddr) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.register(nas...)
}

func (self *testOverlay) register(nas ...PeerAddr) error {
	for _, na := range nas {
		tna := &testPeerAddr{PeerAddr: na}
		addr := na.OverlayAddr()
		if self.posMap[string(addr)] != nil {
			continue
		}
		self.posMap[string(addr)] = tna
		o := order(addr)
		glog.V(6).Infof("PO: %v, orders: %v", o, orders)
		self.pos[o] = append(self.pos[o], tna)
	}
	return nil
}

func order(addr []byte) int {
	return int(addr[0]) / 32
}

func (self *testOverlay) On(n Peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	addr := n.OverlayAddr()
	na := self.posMap[string(addr)]
	if na == nil {
		self.register(n)
		na = self.posMap[string(addr)]
	} else if na.Peer != nil {
		return
	}
	glog.V(6).Infof("Online: %v", fmt.Sprintf("%x", addr[:4]))
	na.Peer = n
	return
}

func (self *testOverlay) Off(n Peer) {
	self.mu.Lock()
	defer self.mu.Unlock()
	addr := n.OverlayAddr()
	na := self.posMap[string(addr)]
	if na == nil {
		return
	}
	delete(self.posMap, string(addr))
	na.Peer = nil
}

// caller must hold the lock
func (self *testOverlay) on(po []*testPeerAddr) (nodes []Peer) {
	for _, na := range po {
		if na.Peer != nil {
			nodes = append(nodes, na.Peer)
		}
	}
	return nodes
}

// caller must hold the lock
func (self *testOverlay) off(po []*testPeerAddr) (nas []PeerAddr) {
	for _, na := range po {
		if na.Peer == nil {
			nas = append(nas, PeerAddr(na))
		}
	}
	return nas
}

func (self *testOverlay) EachLivePeer(base []byte, o int, f func(Peer) bool) {
	if base == nil {
		base = self.addr
	}
	for i := o; i < len(self.pos); i++ {
		for _, na := range self.pos[i] {
			if na.Peer != nil {
				if !f(na.Peer) {
					return
				}
			}
		}
	}
}

func (self *testOverlay) EachPeer(base []byte, o int, f func(PeerAddr) bool) {
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

func (self *testOverlay) SuggestPeer() (PeerAddr, int, bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	for i, po := range self.pos {
		ons := self.on(po)
		if len(ons) < 2 {
			offs := self.off(po)
			if len(offs) > 0 {
				glog.V(6).Infof("node %v is off", offs[0])
				return offs[0], i, true
			}
		}
	}
	return nil, 0, true
}

func (self *testOverlay) String() string {
	self.mu.Lock()
	defer self.mu.Unlock()
	var t []string
	var ons, offs int
	var ns []Peer
	var nas []PeerAddr
	for o, po := range self.pos {
		var row []string
		ns = self.on(po)
		nas = self.off(po)
		ons = len(ns)
		for _, n := range ns {
			addr := n.OverlayAddr()
			row = append(row, fmt.Sprintf("%x", addr[:4]))
		}
		row = append(row, "|")
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
		posMap: make(map[string]*testPeerAddr),
		pos:    make([][]*testPeerAddr, orders),
	}
}
