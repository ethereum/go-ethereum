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
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// aggregatorMemoryLimit is the maximum size of the bottom-most diff layer
	// that aggregates the writes from above until it's flushed into the disk
	// layer.
	//
	// Note, bumping this up might drastically increase the size of the bloom
	// filters that's stored in every diff layer. Don't do that without fully
	// understanding all the implications.
	aggregatorMemoryLimit = uint64(16 * 1024 * 1024)
)

// diffLayer represents a collection of modifications made to the in-memory tries
// after running a block on top.
//
// The goal of a diff layer is to act as a journal, tracking recent modifications
// made to the state, that have not yet graduated into a semi-immutable state.
type diffLayer struct {
	db     *Database  // Main database handler for accessing immature dirty nodes
	origin *diskLayer // Base disk layer to directly use on bloom misses
	parent snapshot   // Parent snapshot modified by this one, never nil
	memory uint64     // Approximate guess as to how much memory we use

	root  common.Hash            // Root hash to which this snapshot diff belongs to
	stale uint32                 // Signals that the layer became stale (state progressed)
	nodes map[string]*cachedNode // Keyed trie nodes for retrieval. (nil means deleted)
	lock  sync.RWMutex
}

// newDiffLayer creates a new diff on top of an existing snapshot, whether that's a low
// level persistent database or a hierarchical diff already.
func newDiffLayer(parent snapshot, root common.Hash, nodes map[string]*cachedNode, db *Database) *diffLayer {
	dl := &diffLayer{
		db:     db,
		parent: parent,
		root:   root,
		nodes:  nodes,
	}
	switch parent := parent.(type) {
	case *diskLayer:
		dl.origin = parent
	case *diffLayer:
		dl.origin = parent.origin
	default:
		panic("unknown parent type")
	}
	for key, node := range nodes {
		dl.memory += uint64(len(key) + int(node.size) + cachedNodeSize)
	}
	return dl
}

// Root returns the root hash for which this snapshot was made.
func (dl *diffLayer) Root() common.Hash {
	return dl.root
}

// Parent returns the subsequent layer of a diff layer.
func (dl *diffLayer) Parent() snapshot {
	return dl.parent
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diffLayer) Stale() bool {
	return atomic.LoadUint32(&dl.stale) != 0
}

// Node retrieves the trie node associated with a particular key.
// The given key must be the internal format node key.
func (dl *diffLayer) Node(key []byte) (node, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If the trie node is known locally, return it
	if n, ok := dl.nodes[string(key)]; ok {
		// The trie node is marked as deleted, don't bother parent anymore.
		if n == nil {
			return nil, nil
		}
		_, hash := DecodeInternalKey(key)
		return n.obj(hash), nil
	}
	return dl.parent.Node(key)
}

// NodeBlob retrieves the trie node blob associated with a particular key.
// The given key must be the internal format node key.
func (dl *diffLayer) NodeBlob(key []byte) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If the trie node is known locally, return it
	if n, ok := dl.nodes[string(key)]; ok {
		// The trie node is marked as deleted, don't bother parent anymore.
		if n == nil {
			return nil, nil
		}
		return n.rlp(), nil
	}
	return dl.parent.NodeBlob(key)
}

// Update creates a new layer on top of the existing snapshot diff tree with
// the specified data items.
func (dl *diffLayer) Update(blockRoot common.Hash, nodes map[string]*cachedNode, db *Database) *diffLayer {
	return newDiffLayer(dl, blockRoot, nodes, db)
}

// flatten pushes all data from this point downwards, flattening everything into
// a single diff at the bottom. Since usually the lowermost diff is the largest,
// the flattening builds up from there in reverse.
func (dl *diffLayer) flatten() snapshot {
	// If the parent is not diff, we're the first in line, return unmodified
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		return dl
	}
	// Parent is a diff, flatten it first (note, apart from weird corned cases,
	// flatten will realistically only ever merge 1 layer, so there's no need to
	// be smarter about grouping flattens together).
	parent = parent.flatten().(*diffLayer)

	parent.lock.Lock()
	defer parent.lock.Unlock()

	// Before actually writing all our data to the parent, first ensure that the
	// parent hasn't been 'corrupted' by someone else already flattening into it
	if atomic.SwapUint32(&parent.stale, 1) != 0 {
		panic("parent diff layer is stale") // we've flattened into the same parent from two children, boo
	}
	// Merge nodes of two layers together, overwrite the nodes with same path.
	storages := make(map[string]string)
	for key := range parent.nodes {
		storage, _ := DecodeInternalKey([]byte(key))
		storages[string(storage)] = key
	}
	for key, data := range dl.nodes {
		storage, _ := DecodeInternalKey([]byte(key))
		if internal, ok := storages[string(storage)]; ok {
			delete(parent.nodes, internal)
		}
		parent.nodes[key] = data
	}
	// Return the combo parent
	return &diffLayer{
		origin: parent.origin,
		parent: parent.parent,
		memory: parent.memory + dl.memory,
		root:   dl.root,
		nodes:  parent.nodes,
	}
}
