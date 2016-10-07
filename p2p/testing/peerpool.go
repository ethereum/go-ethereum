package testing

import (
	"sync"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

type TestPeer interface {
	ID() discover.NodeID
	Drop()
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
	self.peers[p.ID()] = p
}

func (self *TestPeerPool) Remove(p TestPeer) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.peers, p.ID())
}

func (self *TestPeerPool) Has(n *discover.NodeID) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	_, ok := self.peers[*n]
	return ok
}

func (self *TestPeerPool) Get(n *discover.NodeID) TestPeer {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.peers[*n]
}
