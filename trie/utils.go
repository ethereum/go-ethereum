// Copyright 2021 The go-ethereum Authors
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
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// nodeSet is the accumulated dirty nodes set acts as the temporary
// database for storing immature nodes.
type nodeSet struct {
	lock  sync.RWMutex
	nodes map[string]*cachedNode // Set of dirty nodes, indexed by **storage** key
}

// newNodeSet initializes the dirty set.
func newNodeSet() *nodeSet {
	return &nodeSet{
		nodes: make(map[string]*cachedNode),
	}
}

// get retrieves the trie node in the set with **storage** format key.
// Note the returned value shouldn't be changed by callers.
func (set *nodeSet) get(storage []byte, hash common.Hash) (node, bool) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return nil, false
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	if node, ok := set.nodes[string(storage)]; ok && node.hash == hash {
		return node.obj(hash), true
	}
	return nil, false
}

// getBlob retrieves the encoded trie node in the set with **storage** format key.
// Note the returned value shouldn't be changed by callers.
func (set *nodeSet) getBlob(storage []byte, hash common.Hash) ([]byte, bool) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return nil, false
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	if node, ok := set.nodes[string(storage)]; ok && node.hash == hash {
		return node.rlp(), true
	}
	return nil, false
}

// put stores the given state entry in the set. The given key should be encoded in
// the storage format. Note the val shouldn't be changed by caller later.
func (set *nodeSet) put(storage []byte, n node, size int, hash common.Hash) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return
	}
	set.lock.Lock()
	defer set.lock.Unlock()

	set.nodes[string(storage)] = &cachedNode{
		hash: hash,
		node: n,
		size: uint16(size),
	}
}

// del deletes the node from the nodeset with the given key and node hash.
// Note it's mainly used in testing!
func (set *nodeSet) del(storage []byte, hash common.Hash) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return
	}
	set.lock.Lock()
	defer set.lock.Unlock()

	if node, ok := set.nodes[string(storage)]; ok && node.hash == hash {
		delete(set.nodes, string(storage))
	}
}

// merge merges the dirty nodes from the other set. If there are two
// nodes with same key, then update with the node in other set.
func (set *nodeSet) merge(other *nodeSet) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil || other == nil {
		return
	}
	set.lock.Lock()
	defer set.lock.Unlock()

	other.lock.RLock()
	defer other.lock.RUnlock()

	for key, n := range other.nodes {
		set.nodes[key] = n
	}
}

// forEach iterates the dirty nodes in the set and executes the given function.
func (set *nodeSet) forEach(fn func(string, *cachedNode)) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	for key, n := range set.nodes {
		fn(key, n)
	}
}

// forEachBlob iterates the dirty nodes in the set and pass them in RLP-encoded format.
func (set *nodeSet) forEachBlob(fn func(string, []byte)) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	for key, n := range set.nodes {
		fn(key, n.rlp())
	}
}

// len returns the items maintained in the set.
func (set *nodeSet) len() int {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return 0
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	return len(set.nodes)
}

// CommitResult wraps the trie commit result in a single struct.
type CommitResult struct {
	Root common.Hash // The re-calculated trie root hash after commit

	// WrittenNodes is the collection of newly updated and created nodes
	// since last commit. Nodes are indexed by **internal** key.
	WrittenNodes *nodeSet
}

// CommitTo commits the tracked state diff into the given container.
func (result *CommitResult) CommitTo(nodes map[string]*cachedNode) map[string]*cachedNode {
	if nodes == nil {
		nodes = make(map[string]*cachedNode)
	}
	result.WrittenNodes.forEach(func(key string, n *cachedNode) {
		nodes[key] = n
	})
	return nodes
}

// Modified returns the number of modified items.
func (result *CommitResult) Modified() int {
	return result.WrittenNodes.len()
}

// Merge merges the dirty nodes from the other set.
func (result *CommitResult) Merge(other *CommitResult) {
	result.WrittenNodes.merge(other.WrittenNodes)
}

// Nodes returns all contained nodes, key in storage format and value in RLP-encoded format.
func (result *CommitResult) Nodes() map[string][]byte {
	ret := make(map[string][]byte)
	result.WrittenNodes.forEachBlob(func(k string, v []byte) {
		ret[k] = v
	})
	return ret
}

// NewResultFromDeletionSet constructs a commit result with the given dirty node set.
func NewResultFromDeletionSet(set map[common.Hash][]byte) *CommitResult {
	updated := newNodeSet()
	for hash, storage := range set {
		updated.put(storage, nil, 0, hash)
	}
	return &CommitResult{
		Root:         common.Hash{},
		WrittenNodes: updated,
	}
}
