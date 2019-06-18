// Copyright 2018 The go-ethereum Authors
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

package storage

import (
	"sync"
	"time"

	"github.com/ethersphere/swarm/network/timeouts"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// Request encapsulates all the necessary arguments when making a request to NetStore.
// These could have also been added as part of the interface of NetStore.Get, but a request struct seemed
// like a better option
type Request struct {
	Addr        Address  // chunk address
	Origin      enode.ID // who is sending us that request? we compare Origin to the suggested peer from RequestFromPeers
	PeersToSkip sync.Map // peers not to request chunk from
}

// NewRequest returns a new instance of Request based on chunk address skip check and
// a map of peers to skip.
func NewRequest(addr Address) *Request {
	return &Request{
		Addr: addr,
	}
}

// SkipPeer returns if the peer with nodeID should not be requested to deliver a chunk.
// Peers to skip are kept per Request and for a time period of FailedPeerSkipDelay.
func (r *Request) SkipPeer(nodeID string) bool {
	val, ok := r.PeersToSkip.Load(nodeID)
	if !ok {
		return false
	}
	t, ok := val.(time.Time)
	if ok && time.Now().After(t.Add(timeouts.FailedPeerSkipDelay)) {
		r.PeersToSkip.Delete(nodeID)
		return false
	}
	return true
}
