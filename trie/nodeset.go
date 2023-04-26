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
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// NodeSet contains all dirty nodes collected during the commit operation.
// Each node is keyed by path. It's not thread-safe to use.
type NodeSet struct {
	owner   common.Hash // the identifier of the trie
	leaves  []*leaf     // the list of dirty leaves
	updates int         // the count of updated and inserted nodes
	deletes int         // the count of deleted nodes

	// The set of all dirty nodes. Dirty nodes include newly inserted nodes,
	// deleted nodes and updated nodes. The original value of the newly
	// inserted node must be nil, and the original value of the other two
	// types must be non-nil.
	nodes map[string]*trienode.WithPrev
}

// NewNodeSet initializes an empty node set to be used for tracking dirty nodes
// from a specific account or storage trie. The owner is zero for the account
// trie and the owning account address hash for storage tries.
func NewNodeSet(owner common.Hash) *NodeSet {
	return &NodeSet{
		owner: owner,
		nodes: make(map[string]*trienode.WithPrev),
	}
}

// forEachWithOrder iterates the dirty nodes with the order from bottom to top,
// right to left, nodes with the longest path will be iterated first.
func (set *NodeSet) forEachWithOrder(callback func(path string, n *trienode.Node)) {
	var paths sort.StringSlice
	for path := range set.nodes {
		paths = append(paths, path)
	}
	// Bottom-up, longest path first
	sort.Sort(sort.Reverse(paths))
	for _, path := range paths {
		callback(path, set.nodes[path].Unwrap())
	}
}

// addNode adds the provided dirty node into set.
func (set *NodeSet) addNode(path []byte, n *trienode.WithPrev) {
	if n.IsDeleted() {
		set.deletes += 1
	} else {
		set.updates += 1
	}
	set.nodes[string(path)] = n
}

// addLeaf adds the provided leaf node into set.
func (set *NodeSet) addLeaf(node *leaf) {
	set.leaves = append(set.leaves, node)
}

// Size returns the number of dirty nodes in set.
func (set *NodeSet) Size() (int, int) {
	return set.updates, set.deletes
}

// Hashes returns the hashes of all updated nodes. TODO(rjl493456442) how can
// we get rid of it?
func (set *NodeSet) Hashes() []common.Hash {
	var ret []common.Hash
	for _, node := range set.nodes {
		ret = append(ret, node.Hash)
	}
	return ret
}

// Summary returns a string-representation of the NodeSet.
func (set *NodeSet) Summary() string {
	var out = new(strings.Builder)
	fmt.Fprintf(out, "nodeset owner: %v\n", set.owner)
	if set.nodes != nil {
		for path, n := range set.nodes {
			// Deletion
			if n.IsDeleted() {
				fmt.Fprintf(out, "  [-]: %x prev: %x\n", path, n.Prev)
				continue
			}
			// Insertion
			if len(n.Prev) == 0 {
				fmt.Fprintf(out, "  [+]: %x -> %v\n", path, n.Hash)
				continue
			}
			// Update
			fmt.Fprintf(out, "  [*]: %x -> %v prev: %x\n", path, n.Hash, n.Prev)
		}
	}
	for _, n := range set.leaves {
		fmt.Fprintf(out, "[leaf]: %v\n", n)
	}
	return out.String()
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
