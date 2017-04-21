package testing

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type TestPeer interface {
	ID() discover.NodeID
	Drop(error)
}

// TestPeerPool is an example peerPool to demonstrate registration of peer connections
type TestPeerPool struct {
	lock  sync.Mutex
	peers map[discover.NodeID]TestPeer
}

func NewTestPeerPool() *TestPeerPool {
	return &TestPeerPool{peers: make(map[discover.NodeID]TestPeer)}
}

func (self *TestPeerPool) Add(p TestPeer) {
	self.lock.Lock()
	defer self.lock.Unlock()
	log.Trace(fmt.Sprintf("pp add peer  %v", p.ID()))
	self.peers[p.ID()] = p

}

func (self *TestPeerPool) Remove(p TestPeer) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.peers, p.ID())
}

func (self *TestPeerPool) Has(n *adapters.NodeId) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	_, ok := self.peers[n.NodeID]
	return ok
}

func (self *TestPeerPool) Get(n *adapters.NodeId) TestPeer {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.peers[n.NodeID]
}
