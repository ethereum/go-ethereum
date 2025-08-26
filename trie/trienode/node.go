// Copyright 2023 The go-ethereum Authors
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

package trienode

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Node is a wrapper which contains the encoded blob of the trie node and its
// node hash. It is general enough that can be used to represent trie node
// corresponding to different trie implementations.
type Node struct {
	Hash common.Hash // Node hash, empty for deleted node
	Blob []byte      // Encoded node blob, nil for the deleted node
}

// Size returns the total memory size used by this node.
func (n *Node) Size() int {
	return len(n.Blob) + common.HashLength
}

// IsDeleted returns the indicator if the node is marked as deleted.
func (n *Node) IsDeleted() bool {
	return len(n.Blob) == 0
}

// New constructs a node with provided node information.
func New(hash common.Hash, blob []byte) *Node {
	return &Node{Hash: hash, Blob: blob}
}

// NewDeleted constructs a node which is deleted.
func NewDeleted() *Node { return New(common.Hash{}, nil) }

// NodeWithPrev is a wrapper over Node by tracking the original value of node.
type NodeWithPrev struct {
	*Node
	Prev []byte // Nil means the node was not existent
}

// NewNodeWithPrev constructs a node with the additional original value.
func NewNodeWithPrev(hash common.Hash, blob []byte, prev []byte) *NodeWithPrev {
	return &NodeWithPrev{
		Node: &Node{
			Hash: hash,
			Blob: blob,
		},
		Prev: prev,
	}
}

// NewDeletedWithPrev constructs a node which is deleted with the additional
// original value.
func NewDeletedWithPrev(prev []byte) *NodeWithPrev {
	return &NodeWithPrev{
		Node: &Node{
			Hash: common.Hash{},
			Blob: nil,
		},
		Prev: prev,
	}
}

// leaf represents a trie leaf node
type leaf struct {
	Blob   []byte      // raw blob of leaf
	Parent common.Hash // the hash of parent node
}

// NodeSet contains a set of nodes collected during the commit operation.
// Each node is keyed by path. It's not thread-safe to use.
type NodeSet struct {
	Owner   common.Hash
	Leaves  []*leaf
	Nodes   map[string]*Node
	Origins map[string][]byte

	updates int // the count of updated and inserted nodes
	deletes int // the count of deleted nodes
}

// NewNodeSet initializes a node set. The owner is zero for the account trie and
// the owning account address hash for storage tries.
func NewNodeSet(owner common.Hash) *NodeSet {
	return &NodeSet{
		Owner:   owner,
		Nodes:   make(map[string]*Node),
		Origins: make(map[string][]byte),
	}
}

// ForEachWithOrder iterates the nodes with the order from bottom to top,
// right to left, nodes with the longest path will be iterated first.
func (set *NodeSet) ForEachWithOrder(callback func(path string, n *Node)) {
	paths := make([]string, 0, len(set.Nodes))
	for path := range set.Nodes {
		paths = append(paths, path)
	}
	// Bottom-up, the longest path first
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		callback(path, set.Nodes[path])
	}
}

// AddNode adds the provided node into set.
func (set *NodeSet) AddNode(path []byte, n *NodeWithPrev) {
	if n.IsDeleted() {
		set.deletes += 1
	} else {
		set.updates += 1
	}
	key := string(path)
	set.Nodes[key] = n.Node
	set.Origins[key] = n.Prev
}

// MergeDisjoint merges this 'set' with 'other'. It assumes that the sets are disjoint,
// and thus does not deduplicate data (count deletes, dedup leaves etc).
func (set *NodeSet) MergeDisjoint(other *NodeSet) error {
	if set.Owner != other.Owner {
		return fmt.Errorf("nodesets belong to different owner are not mergeable %x-%x", set.Owner, other.Owner)
	}
	maps.Copy(set.Nodes, other.Nodes)
	maps.Copy(set.Origins, other.Origins)

	set.deletes += other.deletes
	set.updates += other.updates

	// Since we assume the sets are disjoint, we can safely append leaves
	// like this without deduplication.
	set.Leaves = append(set.Leaves, other.Leaves...)
	return nil
}

// Merge adds a set of nodes to the current set. It assumes the sets may overlap,
// so deduplication is performed.
func (set *NodeSet) Merge(other *NodeSet) error {
	if set.Owner != other.Owner {
		return fmt.Errorf("nodesets belong to different owner are not mergeable %x-%x", set.Owner, other.Owner)
	}
	for path, node := range other.Nodes {
		prev, ok := set.Nodes[path]
		if ok {
			// overwrite happens, revoke the counter
			if prev.IsDeleted() {
				set.deletes -= 1
			} else {
				set.updates -= 1
			}
		}
		if node.IsDeleted() {
			set.deletes += 1
		} else {
			set.updates += 1
		}
		set.Nodes[path] = node // overwrite the node with new value

		// Add the original value only if it was previously non-existent.
		// If multiple mutations are made to the same node, the first one
		// is considered the true original value.
		if _, exist := set.Origins[path]; !exist {
			set.Origins[path] = other.Origins[path]
		}
	}
	// TODO leaves are not aggregated, as they are not used in storage tries.
	// TODO(rjl493456442) deprecate the leaves along with the legacy hash mode.
	return nil
}

// AddLeaf adds the provided leaf node into set. TODO(rjl493456442) how can
// we get rid of it?
func (set *NodeSet) AddLeaf(parent common.Hash, blob []byte) {
	set.Leaves = append(set.Leaves, &leaf{Blob: blob, Parent: parent})
}

// Size returns the number of dirty nodes in set.
func (set *NodeSet) Size() (int, int) {
	return set.updates, set.deletes
}

// HashSet returns a set of trie nodes keyed by node hash.
func (set *NodeSet) HashSet() map[common.Hash][]byte {
	ret := make(map[common.Hash][]byte, len(set.Nodes))
	for _, n := range set.Nodes {
		ret[n.Hash] = n.Blob
	}
	return ret
}

// Summary returns a string-representation of the NodeSet.
func (set *NodeSet) Summary() string {
	var out = new(strings.Builder)
	fmt.Fprintf(out, "nodeset owner: %v\n", set.Owner)
	for path, n := range set.Nodes {
		// Deletion
		if n.IsDeleted() {
			fmt.Fprintf(out, " [-]: %x prev: %x\n", path, set.Origins[path])
			continue
		}
		// Insertion
		if len(set.Origins[path]) == 0 {
			fmt.Fprintf(out, "  [+]: %x -> %v\n", path, n.Hash)
			continue
		}
		// Update
		fmt.Fprintf(out, " [*]: %x -> %v prev: %x\n", path, n.Hash, set.Origins[path])
	}
	for _, n := range set.Leaves {
		fmt.Fprintf(out, "[leaf]: %v\n", n)
	}
	return out.String()
}

// MergedNodeSet represents a merged node set for a group of tries.
type MergedNodeSet struct {
	Sets map[common.Hash]*NodeSet
}

// NewMergedNodeSet initializes an empty merged set.
func NewMergedNodeSet() *MergedNodeSet {
	return &MergedNodeSet{Sets: make(map[common.Hash]*NodeSet)}
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
	subset, present := set.Sets[other.Owner]
	if present {
		return subset.Merge(other)
	}
	set.Sets[other.Owner] = other
	return nil
}

// Flatten returns a two-dimensional map for internal nodes.
func (set *MergedNodeSet) Flatten() map[common.Hash]map[string]*Node {
	nodes := make(map[common.Hash]map[string]*Node, len(set.Sets))
	for owner, set := range set.Sets {
		nodes[owner] = set.Nodes
	}
	return nodes
}
