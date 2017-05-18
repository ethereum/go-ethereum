package network

//
// import (
// 	"fmt"
// 	"strings"
// 	"sync"
//
// 	"github.com/ethereum/go-ethereum/log"
// )
//
// const orders = 8
//
// type testOverlay struct {
// 	mu     sync.Mutex
// 	addr   []byte
// 	pos    [][]OverlayAddr
// 	posMap map[string]OverlayAddr
// }
//
// type testPeerAddr struct {
// 	Addr
// 	Peer
// }
//
// func (self *testPeerAddr) Address() []byte {
// 	return nil
// }
//
// func (self *testPeerAddr) Update(a OverlayAddr) OverlayAddr {
// 	return self
// }
//
// func (self *testPeerAddr) On(p OverlayConn) OverlayConn {
// 	return self
// }
//
// func (self *testPeerAddr) Off() OverlayAddr {
// 	return self
// }
//
// func (self *testOverlay) Register(peers chan OverlayAddr) error {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()
// 	var nas []OverlayAddr
// 	for a := range peers {
// 		nas = append(nas, a)
// 	}
// 	return self.register(nas...)
// }
//
// func (self *testOverlay) BaseAddr() []byte {
// 	return nil
// }
//
// func (self *testOverlay) register(nas ...OverlayAddr) error {
// 	for _, na := range nas {
// 		addr := na.Address()
// 		if self.posMap[string(addr)] != nil {
// 			continue
// 		}
// 		self.posMap[string(addr)] = na
// 		o := order(addr)
// 		log.Trace(fmt.Sprintf("PO: %v, orders: %v", o, orders))
// 		self.pos[o] = append(self.pos[o], na)
// 	}
// 	return nil
// }
//
// func order(addr []byte) int {
// 	return int(addr[0]) / 32
// }
//
// func (self *testOverlay) On(n OverlayConn) {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()
// 	addr := n.Address()
// 	na := self.posMap[string(addr)]
// 	if na == nil {
// 		self.register(n)
// 		na = self.posMap[string(addr)]
// 	} else if na.Peer != nil {
// 		return
// 	}
// 	log.Trace(fmt.Sprintf("Online: %x", addr[:4]))
// 	na.Peer = n
// 	return
// }
//
// func (self *testOverlay) Off(n OverlayConn) {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()
// 	addr := n.Over()
// 	na := self.posMap[string(addr)]
// 	if na == nil {
// 		return
// 	}
// 	delete(self.posMap, string(addr))
// 	na.Peer = nil
// }
//
// // caller must hold the lock
// func (self *testOverlay) on(po []*testPeerAddr) (nodes []OverlayConn) {
// 	for _, na := range po {
// 		if na.Peer != nil {
// 			nodes = append(nodes, na)
// 		}
// 	}
// 	return nodes
// }
//
// // caller must hold the lock
// func (self *testOverlay) off(po []*testPeerAddr) (nas []OverlayAddr) {
// 	for _, na := range po {
// 		if na.Peer == (*bzzPeer)(nil) {
// 			nas = append(nas, Addr(na))
// 		}
// 	}
// 	return nas
// }
//
// func (self *testOverlay) EachConn(base []byte, o int, f func(OverlayConn, int, bool) bool) {
// 	for i := o; i < len(self.pos); i++ {
// 		for _, na := range self.pos[i] {
// 			if na.Peer != nil {
// 				if !f(na, o, false) {
// 					return
// 				}
// 			}
// 		}
// 	}
// }
//
// func (self *testOverlay) EachAddr(base []byte, o int, f func(OverlayAddr, int) bool) {
// 	for i := o; i < len(self.pos); i++ {
// 		for _, na := range self.pos[i] {
// 			if !f(na, i) {
// 				return
// 			}
// 		}
// 	}
// }
//
// func (self *testOverlay) SuggestPeer() (OverlayAddr, int, bool) {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()
// 	for i, po := range self.pos {
// 		ons := self.on(po)
// 		if len(ons) < 2 {
// 			offs := self.off(po)
// 			if len(offs) > 0 {
// 				log.Trace(fmt.Sprintf("node %v is off", offs[0]))
// 				return offs[0], i, true
// 			}
// 		}
// 	}
// 	return nil, 0, true
// }
//
// func (self *testOverlay) String() string {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()
// 	var t []string
// 	var ons, offs int
// 	var ns []Peer
// 	var nas []Addr
// 	for o, po := range self.pos {
// 		var row []string
// 		ns = self.on(po)
// 		nas = self.off(po)
// 		ons = len(ns)
// 		for _, n := range ns {
// 			addr := n.Over()
// 			row = append(row, fmt.Sprintf("%x", addr[:4]))
// 		}
// 		row = append(row, "|")
// 		offs = len(nas)
// 		for _, na := range nas {
// 			addr := na.Over()
// 			row = append(row, fmt.Sprintf("%x", addr[:4]))
// 		}
// 		t = append(t, fmt.Sprintf("%v: (%v/%v) %v", o, ons, offs, strings.Join(row, " ")))
// 	}
// 	return strings.Join(t, "\n")
// }
//
// func NewTestOverlay(addr []byte) *testOverlay {
// 	return &testOverlay{
// 		addr:   addr,
// 		posMap: make(map[string]*testPeerAddr),
// 		pos:    make([][]*testPeerAddr, orders),
// 	}
// }
