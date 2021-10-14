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

// tracker keeps track of the newly inserted/deleted trie nodes.
type tracker struct {
	lock     sync.RWMutex
	inserted map[string]struct{} // Set of inserted nodes, indexed by **storage** key
	deleted  map[string]struct{} // Set of deleted nodes, indexed by **storage** key
}

// newTracker initializes diff tracker.
func newTracker() *tracker {
	return &tracker{
		inserted: make(map[string]struct{}),
		deleted:  make(map[string]struct{}),
	}
}

// onInsert tracks the newly inserted trie node. If it's already
// in the deletion set(resurrected node), then just wipe it from
// the deletion set as the "untouched".
func (t *tracker) onInsert(key []byte) {
	// Don't panic on uninitialized tracker, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, present := t.deleted[string(key)]; present {
		delete(t.deleted, string(key))
		return
	}
	t.inserted[string(key)] = struct{}{}
}

// onDelete tracks the newly deleted trie node. If it's already
// in the addition set, then just wipe it from the addition set
// as the "untouched".
func (t *tracker) onDelete(key []byte) {
	// Don't panic on uninitialized tracker, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, present := t.inserted[string(key)]; present {
		delete(t.inserted, string(key))
		return
	}
	t.deleted[string(key)] = struct{}{}
}

// keyList returns the tracked inserted/deleted trie nodes in list.
func (t *tracker) keyList() ([][]byte, [][]byte) {
	// Don't panic on uninitialized tracker, it's possible in testing.
	if t == nil {
		return nil, nil
	}
	t.lock.RLock()
	defer t.lock.RUnlock()

	var (
		inserted [][]byte
		deleted  [][]byte
	)
	for key := range t.inserted {
		inserted = append(inserted, []byte(key))
	}
	for key := range t.deleted {
		deleted = append(deleted, []byte(key))
	}
	return inserted, deleted
}

// keyList returns the tracked inserted/deleted trie nodes in list.
func (t *tracker) reset() {
	// Don't panic on uninitialized tracker, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	t.inserted = make(map[string]struct{})
	t.deleted = make(map[string]struct{})
	t.lock.Unlock()
}

// nodeSet is the accumulated dirty nodes set acts as the temporary
// database for storing immature nodes.
type nodeSet struct {
	lock  sync.RWMutex
	nodes map[string]*cachedNode // Set of dirty nodes, indexed by **internal** key
}

// newNodeSet initializes the dirty set.
func newNodeSet() *nodeSet {
	return &nodeSet{
		nodes: make(map[string]*cachedNode),
	}
}

// get retrieves the trie node in the set with **internal** format key.
// Note the returned value shouldn't be changed by callers.
func (set *nodeSet) get(key []byte) (node, bool) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return nil, false
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	if blob, ok := set.nodes[string(key)]; ok {
		_, hash := DecodeInternalKey(key)
		return blob.obj(hash), true
	}
	return nil, false
}

// getBlob retrieves the encoded trie node in the set with **internal** format key.
// Note the returned value shouldn't be changed by callers.
func (set *nodeSet) getBlob(key []byte) ([]byte, bool) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return nil, false
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	if blob, ok := set.nodes[string(key)]; ok {
		return blob.rlp(), true
	}
	return nil, false
}

// put stores the given state entry in the set. If the val is nil which means
// the state is deleted. The given key should be encoded in the internal format.
// Note the val shouldn't be changed by caller later.
func (set *nodeSet) put(key []byte, n node, size int) {
	// Don't panic on uninitialized set, it's possible in testing.
	if set == nil {
		return
	}
	set.lock.Lock()
	defer set.lock.Unlock()

	set.nodes[string(key)] = &cachedNode{
		node: n,
		size: uint16(size),
	}
}

// merge merges the dirty nodes from the other set.
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

// CommitResult wraps the trie commit result in the single struct.
type CommitResult struct {
	Root common.Hash // The re-calculated trie root hash after commit

	// UpdatedNodes is the collection of newly updated and created nodes since
	// last commit. Nodes are indexed by **internal** key.
	UpdatedNodes *nodeSet

	// DeletedNodes is the key list of newly deleted nodes since last commit
	// The embedded node will also be included here which doesn't have a
	// corresponding database entry, but it shouldn't affect the correctness.
	// The node key is encoded in **storage** format.
	DeletedNodes [][]byte

	// insertedNodes is the key list of newly inserted nodes since last commit.
	// It's mainly for testing right now.
	// The node key is encoded in **storage** format.
	insertedNodes [][]byte
}

// CommitTo commits the tracked state diff into the given container.
func (result *CommitResult) CommitTo(nodes map[string]*cachedNode) map[string]*cachedNode {
	if nodes == nil {
		nodes = make(map[string]*cachedNode)
	}
	result.UpdatedNodes.forEach(func(key string, n *cachedNode) {
		nodes[key] = n
	})
	return nodes
}

// Modified returns the number of modified items.
func (result *CommitResult) Modified() int {
	return result.UpdatedNodes.len()
}

// Merge merges the dirty nodes from the other set.
func (result *CommitResult) Merge(other *CommitResult) {
	result.UpdatedNodes.merge(other.UpdatedNodes)
}

// Nodes returns all contained nodes in RLP-encoded format.
func (result *CommitResult) Nodes() map[string][]byte {
	ret := make(map[string][]byte)
	result.UpdatedNodes.forEachBlob(func(k string, v []byte) {
		ret[k] = v
	})
	return ret
}
