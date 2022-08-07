// Copyright 2022 The go-ethereum Authors
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

package trie

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// memoryNode is all the information we know about a single cached trie node
// in the memory.
type memoryNode struct {
	hash common.Hash // Node hash, computed by hashing rlp value
	size uint16      // Byte size of the useful cached data
	node node        // Cached collapsed trie node, or raw rlp data
}

// NodeSet contains all dirty nodes collected during the commit operation.
// Each node is keyed by path. It's not thread-safe to use.
type NodeSet struct {
	owner  common.Hash            // the identifier of the trie
	paths  []string               // the path of dirty nodes, sort by insertion order
	nodes  map[string]*memoryNode // the map of dirty nodes, keyed by node path
	leaves []*leaf                // the list of dirty leaves
}

// NewNodeSet initializes an empty node set to be used for tracking dirty nodes
// from a specific account or storage trie. The owner is zero for the account
// trie and the owning account address hash for storage tries.
func NewNodeSet(owner common.Hash) *NodeSet {
	return &NodeSet{
		owner: owner,
		nodes: make(map[string]*memoryNode),
	}
}

// add caches node with provided path and node object.
func (set *NodeSet) add(path string, node *memoryNode) {
	set.paths = append(set.paths, path)
	set.nodes[path] = node
}

// addLeaf caches the provided leaf node.
func (set *NodeSet) addLeaf(node *leaf) {
	set.leaves = append(set.leaves, node)
}

// Len returns the number of dirty nodes contained in the set.
func (set *NodeSet) Len() int {
	return len(set.nodes)
}

// MergedNodeSet represents a merged dirty node set for a group of tries.
type MergedNodeSet struct {
	sets map[common.Hash]*NodeSet
}

// NewMergedNodeSet initializes an empty merged set.
func NewMergedNodeSet() *MergedNodeSet {
	return &MergedNodeSet{sets: make(map[common.Hash]*NodeSet)}
}

// NewWithNodeSet constructs a merged nodeset with the provided single set.
func NewWithNodeSet(set *NodeSet) *MergedNodeSet {
	merged := NewMergedNodeSet()
	merged.Merge(set)
	return merged
}

// Merge merges the provided dirty nodes of a trie into the set. The assumption
// is held that no duplicated set belonging to the same trie will be merged twice.
func (set *MergedNodeSet) Merge(other *NodeSet) error {
	_, present := set.sets[other.owner]
	if present {
		return fmt.Errorf("duplicate trie for owner %#x", other.owner)
	}
	set.sets[other.owner] = other
	return nil
}
