// Copyright 2017 The go-ethereum Authors
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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// MemPool is an intermediate write layer between the trie data structures and
// the disk database. The aim is to accumulate trie writes in-memory and only
// periodically flush a couple tries to disk, garbage collecting the remainder.
type MemPool struct {
	cache map[common.Hash][]byte // Cached data blocks of the trie nodes

	parents  map[common.Hash]int                      // Number of live nodes referencing a given one
	children map[common.Hash]map[common.Hash]struct{} // Set of children referenced by given nodes

	gctime  time.Duration      // Time spent on garbage collection since last commit
	gcnodes uint64             // Nodes garbage collected since last commit
	gcsize  common.StorageSize // Data storage garbage collected since last commit

	size common.StorageSize // Storage size of the memory pool
	lock sync.RWMutex
}

// NewMemPool creates a new memory pool to store ephemeral trie nodes before they
// are written out to disk or garbage collected.
func NewMemPool() *MemPool {
	pool := &MemPool{
		cache:    make(map[common.Hash][]byte),
		parents:  make(map[common.Hash]int),
		children: make(map[common.Hash]map[common.Hash]struct{}),
	}
	pool.children[common.Hash{}] = make(map[common.Hash]struct{})
	return pool
}

// insert writes a new trie node to the memory pool if it's yet unknown. The pool
// will make a copy of the slice.
//
// Note, this method assumes that the pool's lock is held!
func (pool *MemPool) insert(hash common.Hash, blob []byte) {
	if _, ok := pool.cache[hash]; ok {
		return
	}
	pool.cache[hash] = common.CopyBytes(blob)
	pool.children[hash] = make(map[common.Hash]struct{})

	pool.size += common.StorageSize(common.HashLength + len(blob))
}

// Fetch retrieves a cached trie node from memory, or returns nil if the pool
// does not have this particular piece of data.
func (pool *MemPool) Fetch(hash common.Hash) []byte {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return pool.cache[hash]
}

// Reference adds a new reference from parent to node.
func (pool *MemPool) Reference(node common.Hash, parent common.Hash) {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	pool.reference(node, parent)
}

// reference is the private locked version of Reference.
func (pool *MemPool) reference(node common.Hash, parent common.Hash) {
	// If the node does not exist, it's a node pulled from disk, skip
	if _, ok := pool.cache[node]; !ok {
		return
	}
	pool.parents[node]++
	pool.children[parent][node] = struct{}{}
}

// Dereference removes an existing reference from parent to node.
func (pool *MemPool) Dereference(node common.Hash, parent common.Hash) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	nodes, storage, start := len(pool.cache), pool.size, time.Now()
	pool.dereference(node, parent)

	pool.gcnodes += uint64(nodes - len(pool.cache))
	pool.gcsize += storage - pool.size
	pool.gctime += time.Since(start)
}

// dereference is the private locked version of Dereference.
func (pool *MemPool) dereference(node common.Hash, parent common.Hash) {
	// If the node does not exist, it's a previously comitted node.
	blob, ok := pool.cache[node]
	if !ok {
		return
	}
	delete(pool.children[parent], node)
	pool.parents[node]--

	// If there are no more references to the child, delete it and cascade
	if pool.parents[node] == 0 {
		for child := range pool.children[node] {
			pool.dereference(child, node)
		}
		delete(pool.cache, node)
		delete(pool.parents, node)
		delete(pool.children, node)

		pool.size -= common.StorageSize(common.HashLength + len(blob))
	}
}

// Commit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions.
func (pool *MemPool) Commit(node common.Hash, db DatabaseWriter) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	nodes, storage, start := len(pool.cache), pool.size, time.Now()
	pool.commit(node, db)

	log.Debug("Committed trie from memory pool", "nodes", nodes-len(pool.cache), "size", storage-pool.size, "time", time.Since(start),
		"gcnodes", pool.gcnodes, "gcsize", pool.gcsize, "gctime", pool.gctime, "livenodes", len(pool.cache), "livesize", pool.size)

	// Reset the garbage collection statistics
	pool.gcnodes, pool.gcsize, pool.gctime = 0, 0, 0

	// Sanity check that we don't have dangling nodes in the pool (missing refs)
	for hash, refs := range pool.parents {
		if refs == 0 {
			log.Warn("dangling node in mempool", "hash", hash)
			break
		}
	}
}

// commit is the private locked version of Commit.
func (pool *MemPool) commit(node common.Hash, db DatabaseWriter) {
	// If the node does not exist, it's a previously comitted node.
	blob, ok := pool.cache[node]
	if !ok {
		return
	}
	for child := range pool.children[node] {
		pool.commit(child, db)
	}
	db.Put(node[:], blob)

	delete(pool.cache, node)
	delete(pool.parents, node)
	delete(pool.children, node)

	pool.size -= common.StorageSize(common.HashLength + len(blob))
}
