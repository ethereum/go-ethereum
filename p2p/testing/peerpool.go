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

	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
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

func (pp *TestPeerPool) Add(p TestPeer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	log.Trace(fmt.Sprintf("pp add peer  %v", p.ID()))
	pp.peers[p.ID()] = p

}

func (pp *TestPeerPool) Remove(p TestPeer) {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	delete(pp.peers, p.ID())
}

func (pp *TestPeerPool) Has(id discover.NodeID) bool {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	_, ok := pp.peers[id]
	return ok
}

func (pp *TestPeerPool) Get(id discover.NodeID) TestPeer {
	pp.lock.Lock()
	defer pp.lock.Unlock()
	return pp.peers[id]
}
