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

package simulation

import (
	"strings"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

// ConnectToPivotNode connects the node with provided NodeID
// to the pivot node, already set by Simulation.SetPivotNode method.
// It is useful when constructing a star network topology
// when simulation adds and removes nodes dynamically.
func (s *Simulation) ConnectToPivotNode(id discover.NodeID) (err error) {
	pid := s.PivotNodeID()
	if pid == nil {
		return ErrNoPivotNode
	}
	return s.connect(*pid, id)
}

// ConnectToLastNode connects the node with provided NodeID
// to the last node that is up, and avoiding connection to self.
// It is useful when constructing a chain network topology
// when simulation adds and removes nodes dynamically.
func (s *Simulation) ConnectToLastNode(id discover.NodeID) (err error) {
	ids := s.UpNodeIDs()
	l := len(ids)
	if l < 2 {
		return nil
	}
	lid := ids[l-1]
	if lid == id {
		lid = ids[l-2]
	}
	return s.connect(lid, id)
}

// ConnectToRandomNode connects the node with provieded NodeID
// to a random node that is up.
func (s *Simulation) ConnectToRandomNode(id discover.NodeID) (err error) {
	n := s.RandomUpNode(id)
	if n == nil {
		return ErrNodeNotFound
	}
	return s.connect(n.ID, id)
}

// ConnectNodesFull connects all nodes one to another.
// It provides a complete connectivity in the network
// which should be rarely needed.
func (s *Simulation) ConnectNodesFull(ids []discover.NodeID) (err error) {
	if ids == nil {
		ids = s.UpNodeIDs()
	}
	l := len(ids)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			err = s.connect(ids[i], ids[j])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ConnectNodesChain connects all nodes in a chain topology.
// If ids argument is nil, all nodes that are up will be connected.
func (s *Simulation) ConnectNodesChain(ids []discover.NodeID) (err error) {
	if ids == nil {
		ids = s.UpNodeIDs()
	}
	l := len(ids)
	for i := 0; i < l-1; i++ {
		err = s.connect(ids[i], ids[i+1])
		if err != nil {
			return err
		}
	}
	return nil
}

// ConnectNodesRing connects all nodes in a ring topology.
// If ids argument is nil, all nodes that are up will be connected.
func (s *Simulation) ConnectNodesRing(ids []discover.NodeID) (err error) {
	if ids == nil {
		ids = s.UpNodeIDs()
	}
	l := len(ids)
	if l < 2 {
		return nil
	}
	for i := 0; i < l-1; i++ {
		err = s.connect(ids[i], ids[i+1])
		if err != nil {
			return err
		}
	}
	return s.connect(ids[l-1], ids[0])
}

// ConnectNodesStar connects all nodes in a star topology
// with the center at provided NodeID.
// If ids argument is nil, all nodes that are up will be connected.
func (s *Simulation) ConnectNodesStar(id discover.NodeID, ids []discover.NodeID) (err error) {
	if ids == nil {
		ids = s.UpNodeIDs()
	}
	l := len(ids)
	for i := 0; i < l; i++ {
		if id == ids[i] {
			continue
		}
		err = s.connect(id, ids[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// ConnectNodesStarPivot connects all nodes in a star topology
// with the center at already set pivot node.
// If ids argument is nil, all nodes that are up will be connected.
func (s *Simulation) ConnectNodesStarPivot(ids []discover.NodeID) (err error) {
	id := s.PivotNodeID()
	if id == nil {
		return ErrNoPivotNode
	}
	return s.ConnectNodesStar(*id, ids)
}

// connect connects two nodes but ignores already connected error.
func (s *Simulation) connect(oneID, otherID discover.NodeID) error {
	return ignoreAlreadyConnectedErr(s.Net.Connect(oneID, otherID))
}

func ignoreAlreadyConnectedErr(err error) error {
	if err == nil || strings.Contains(err.Error(), "already connected") {
		return nil
	}
	return err
}
