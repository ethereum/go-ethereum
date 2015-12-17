// Copyright 2015 The go-ethereum Authors
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

package p2pint

import "github.com/ethereum/go-ethereum/p2p/discover"

type Flag int

const (
	DynDialedConn Flag = 1 << iota
	StaticDialedConn
	InboundConn
	TrustedConn
)

// PeerSet tracks connected peers and various indices
// associcated with them.
type PeerSet struct {
	limit     int
	counts    map[string]int
	all       map[discover.NodeID]Peer
	preferred map[discover.NodeID][]string
	static    map[discover.NodeID]struct{}
}

type Peer interface {
	ID() discover.NodeID
	ActiveProtocols() []string
	Flag() Flag
}

func NewPeerSet(limit int) *PeerSet {
	return &PeerSet{
		limit:     limit,
		counts:    make(map[string]int),
		all:       make(map[discover.NodeID]psPeer),
		preferred: make(map[discover.NodeID][]string),
		static:    make(map[discover.NodeID]struct{}),
	}
}

func (ps *PeerSet) Add(p psPeer) {
	for _, protoName := range p.activeProtocols() {
		ps.counts[protoName]++
	}
	ps.all[p.ID()] = p
}

func (ps *PeerSet) Remove(p psPeer) {
	for _, protoName := range p.activeProtocols() {
		ps.setIdle(p, protoName)
		ps.counts[protoName]--
	}
	delete(ps.all, p.ID())
}

func (ps *PeerSet) NumPeers() int {
	return len(ps.all)
}

func (ps *PeerSet) IsAtCapacity() bool {
	for _, numProtoPeers := range ps.counts {
		if numProtoPeers < ps.limit {
			return false
		}
	}
	return true
}

func (ps *PeerSet) IsConnected(id discover.NodeID) bool {
	_, ok := ps.all[id]
	return ok
}

// disables auto-disconnect for a particular peer
func (ps *PeerSet) SetPreferred(p psPeer, protoName string) {
	pl := ps.preferred[p.ID()]
	for _, name := range pl {
		if name == protoName {
			return
		}
	}
	ps.preferred[p.ID()] = append(pl, protoName)
}

// opposite of setPreferred, removes from ps.preferred
func (ps *PeerSet) SetIdle(p psPeer, protoName string) {
	pl := ps.preferred[p.ID()]
	pos := -1
	for i, name := range pl {
		if name == protoName {
			pos = i
			break
		}
	}
	if pos >= 0 {
		if len(pl) == 1 {
			delete(ps.preferred, p.ID())
		} else {
			ps.preferred[p.ID()] = append(pl[:pos], pl[pos:]...)
		}
	}
}
