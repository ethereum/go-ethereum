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

package simulations

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	ErrNodeNotFound = errors.New("node not found")
	ErrNoPivotNode  = errors.New("no pivot node set")
)

// ConnectToPivotNode connects the node with provided NodeID
// to the pivot node, already set by Network.SetPivotNode method.
// It is useful when constructing a star network topology
// when Network adds and removes nodes dynamically.
func (net *Network) ConnectToPivotNode(id enode.ID) (err error) {
	pivot := net.GetPivotNode()
	if pivot == nil {
		return ErrNoPivotNode
	}
	return net.connect(pivot.ID(), id)
}

// ConnectToLastNode connects the node with provided NodeID
// to the last node that is up, and avoiding connection to self.
// It is useful when constructing a chain network topology
// when Network adds and removes nodes dynamically.
func (net *Network) ConnectToLastNode(id enode.ID) (err error) {
	ids := net.getUpNodeIDs()
	l := len(ids)
	if l < 2 {
		return nil
	}
	last := ids[l-1]
	if last == id {
		last = ids[l-2]
	}
	return net.connect(last, id)
}

// ConnectToRandomNode connects the node with provided NodeID
// to a random node that is up.
func (net *Network) ConnectToRandomNode(id enode.ID) (err error) {
	selected := net.GetRandomUpNode(id)
	if selected == nil {
		return ErrNodeNotFound
	}
	return net.connect(selected.ID(), id)
}

// ConnectNodesFull connects all nodes one to another.
// It provides a complete connectivity in the network
// which should be rarely needed.
func (net *Network) ConnectNodesFull(ids []enode.ID) (err error) {
	if ids == nil {
		ids = net.getUpNodeIDs()
	}
	for i, lid := range ids {
		for _, rid := range ids[i+1:] {
			if err = net.connect(lid, rid); err != nil {
				return err
			}
		}
	}
	return nil
}

// ConnectNodesChain connects all nodes in a chain topology.
// If ids argument is nil, all nodes that are up will be connected.
func (net *Network) ConnectNodesChain(ids []enode.ID) (err error) {
	if ids == nil {
		ids = net.getUpNodeIDs()
	}
	l := len(ids)
	for i := 0; i < l-1; i++ {
		if err := net.connect(ids[i], ids[i+1]); err != nil {
			return err
		}
	}
	return nil
}

// ConnectNodesRing connects all nodes in a ring topology.
// If ids argument is nil, all nodes that are up will be connected.
func (net *Network) ConnectNodesRing(ids []enode.ID) (err error) {
	if ids == nil {
		ids = net.getUpNodeIDs()
	}
	l := len(ids)
	if l < 2 {
		return nil
	}
	if err := net.ConnectNodesChain(ids); err != nil {
		return err
	}
	return net.connect(ids[l-1], ids[0])
}

// ConnectNodesStar connects all nodes in a star topology
// with the center at provided NodeID.
// If ids argument is nil, all nodes that are up will be connected.
func (net *Network) ConnectNodesStar(pivot enode.ID, ids []enode.ID) (err error) {
	if ids == nil {
		ids = net.getUpNodeIDs()
	}
	for _, id := range ids {
		if pivot == id {
			continue
		}
		if err := net.connect(pivot, id); err != nil {
			return err
		}
	}
	return nil
}

// ConnectNodesStarPivot connects all nodes in a star topology
// with the center at already set pivot node.
// If ids argument is nil, all nodes that are up will be connected.
func (net *Network) ConnectNodesStarPivot(ids []enode.ID) (err error) {
	pivot := net.GetPivotNode()
	if pivot == nil {
		return ErrNoPivotNode
	}
	return net.ConnectNodesStar(pivot.ID(), ids)
}

// connect connects two nodes but ignores already connected error.
func (net *Network) connect(oneID, otherID enode.ID) error {
	return ignoreAlreadyConnectedErr(net.Connect(oneID, otherID))
}

func ignoreAlreadyConnectedErr(err error) error {
	if err == nil || strings.Contains(err.Error(), "already connected") {
		return nil
	}
	return err
}

// SetPivotNode sets the NodeID of the network's pivot node.
// Pivot node is just a specific node that should be treated
// differently then other nodes in test. SetPivotNode and
// GetPivotNode are just a convenient functions to set and
// retrieve it.
func (net *Network) SetPivotNode(id enode.ID) {
	net.lock.Lock()
	defer net.lock.Unlock()
	net.pivotNodeID = id
}

// GetPivotNode returns NodeID of the pivot node set by
// Network.SetPivotNode method.
func (net *Network) GetPivotNode() (node *Node) {
	net.lock.RLock()
	defer net.lock.RUnlock()
	return net.getNode(net.pivotNodeID)
}
