// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package testing

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
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

func (self *TestPeerPool) Has(id discover.NodeID) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	_, ok := self.peers[id]
	return ok
}

func (self *TestPeerPool) Get(id discover.NodeID) TestPeer {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.peers[id]
}
