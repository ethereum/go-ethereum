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

import "github.com/ethereum/go-ethereum/p2p/enode"

// BucketKey is the type that should be used for keys in simulation buckets.
type BucketKey string

// NodeItem returns an item set in ServiceFunc function for a particualar node.
func (s *Simulation) NodeItem(id enode.ID, key interface{}) (value interface{}, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[id]; !ok {
		return nil, false
	}
	return s.buckets[id].Load(key)
}

// SetNodeItem sets a new item associated with the node with provided NodeID.
// Buckets should be used to avoid managing separate simulation global state.
func (s *Simulation) SetNodeItem(id enode.ID, key interface{}, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buckets[id].Store(key, value)
}

// NodesItems returns a map of items from all nodes that are all set under the
// same BucketKey.
func (s *Simulation) NodesItems(key interface{}) (values map[enode.ID]interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.NodeIDs()
	values = make(map[enode.ID]interface{}, len(ids))
	for _, id := range ids {
		if _, ok := s.buckets[id]; !ok {
			continue
		}
		if v, ok := s.buckets[id].Load(key); ok {
			values[id] = v
		}
	}
	return values
}

// UpNodesItems returns a map of items with the same BucketKey from all nodes that are up.
func (s *Simulation) UpNodesItems(key interface{}) (values map[enode.ID]interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.UpNodeIDs()
	values = make(map[enode.ID]interface{})
	for _, id := range ids {
		if _, ok := s.buckets[id]; !ok {
			continue
		}
		if v, ok := s.buckets[id].Load(key); ok {
			values[id] = v
		}
	}
	return values
}
