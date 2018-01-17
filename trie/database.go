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

// secureKeyPrefix is the database key prefix used to store trie node preimages.
var secureKeyPrefix = []byte("secure-key-")

// secureKeyLength is the length of the above prefix + 32byte hash.
const secureKeyLength = 11 + 32

// DatabaseReader wraps the Get and Has method of a backing store for the trie.
type DatabaseReader interface {
	// Get retrieves the value associated with key form the database.
	Get(key []byte) (value []byte, err error)

	// Has retrieves whether a key is present in the database.
	Has(key []byte) (bool, error)
}

// DatabaseWriter wraps the Put method of a backing store for the trie.
type DatabaseWriter interface {
	// Put stores the mapping key->value in the database. Implementations must not
	// hold onto the value as the trie will reuse the slice across calls to Put.
	Put(key, value []byte) error
}

// Database is an intermediate write layer between the trie data structures and
// the disk database. The aim is to accumulate trie writes in-memory and only
// periodically flush a couple tries to disk, garbage collecting the remainder.
type Database struct {
	diskdb DatabaseReader // Persistent storage for matured trie nodes

	nodes    map[common.Hash][]byte                   // Cached data blocks of the trie nodes
	parents  map[common.Hash]int                      // Number of live nodes referencing a given one
	children map[common.Hash]map[common.Hash]struct{} // Set of children referenced by given nodes

	preimages map[common.Hash][]byte // Preimages of nodes from the secure trie
	seckeybuf [secureKeyLength]byte  // Ephemeral buffer for calculating preimage keys

	gctime  time.Duration      // Time spent on garbage collection since last commit
	gcnodes uint64             // Nodes garbage collected since last commit
	gcsize  common.StorageSize // Data storage garbage collected since last commit

	size common.StorageSize // Storage size of the memory cache
	lock sync.RWMutex
}

// NewDatabase creates a new trie database to store ephemeral trie content before
// its written out to disk or garbage collected.
func NewDatabase(diskdb DatabaseReader) *Database {
	db := &Database{
		diskdb:    diskdb,
		nodes:     make(map[common.Hash][]byte),
		parents:   make(map[common.Hash]int),
		children:  make(map[common.Hash]map[common.Hash]struct{}),
		preimages: make(map[common.Hash][]byte),
	}
	db.children[common.Hash{}] = make(map[common.Hash]struct{})
	return db
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() DatabaseReader {
	return db.diskdb
}

// Insert writes a new trie node to the memory database if it's yet unknown. The
// method will make a copy of the slice.
func (db *Database) Insert(hash common.Hash, blob []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.insert(hash, blob)
}

// insert is the private locked version of Insert.
func (db *Database) insert(hash common.Hash, blob []byte) {
	if _, ok := db.nodes[hash]; ok {
		return
	}
	db.nodes[hash] = common.CopyBytes(blob)
	db.children[hash] = make(map[common.Hash]struct{})

	db.size += common.StorageSize(common.HashLength + len(blob))
}

// insertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will make a copy of the slice.
//
// Note, this method assumes that the database's lock is held!
func (db *Database) insertPreimage(hash common.Hash, preimage []byte) {
	if _, ok := db.preimages[hash]; ok {
		return
	}
	db.preimages[hash] = common.CopyBytes(preimage)
	db.size += common.StorageSize(common.HashLength + len(preimage))
}

// Node retrieves a cached trie node from memory. If it cannot be found cached,
// the method queries the persistent database for the content.
func (db *Database) Node(hash common.Hash) ([]byte, error) {
	// Retrieve the node from cache if available
	db.lock.RLock()
	blob := db.nodes[hash]
	db.lock.RUnlock()

	if blob != nil {
		return blob, nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	return db.diskdb.Get(hash[:])
}

// preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *Database) preimage(hash common.Hash) ([]byte, error) {
	// Retrieve the node from cache if available
	db.lock.RLock()
	preimage := db.preimages[hash]
	db.lock.RUnlock()

	if preimage != nil {
		return preimage, nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	return db.diskdb.Get(db.secureKey(hash[:]))
}

// secureKey returns the database key for the preimage of key, as an ephemeral
// buffer. The caller must not hold onto the return value because it will become
// invalid on the next call.
func (db *Database) secureKey(key []byte) []byte {
	buf := append(db.seckeybuf[:0], secureKeyPrefix...)
	buf = append(buf, key...)
	return buf
}

// Nodes retrieves the hashes of all the nodes cached within the memory database.
// This method is extremely expensive and should only be used to validate internal
// states in test code.
func (db *Database) Nodes() []common.Hash {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var hashes = make([]common.Hash, 0, len(db.nodes))
	for hash := range db.nodes {
		hashes = append(hashes, hash)
	}
	return hashes
}

// Preimages retrieves the hashes of all the node pre-images cached within the
// memory database. This method is extremely expensive and should only be used
// to validate internal states in test code.
func (db *Database) Preimages() []common.Hash {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var hashes = make([]common.Hash, 0, len(db.nodes))
	for hash := range db.preimages {
		hashes = append(hashes, hash)
	}
	return hashes
}

// Reference adds a new reference from a parent node to a child node.
func (db *Database) Reference(child common.Hash, parent common.Hash) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	db.reference(child, parent)
}

// reference is the private locked version of Reference.
func (db *Database) reference(child common.Hash, parent common.Hash) {
	// If the node does not exist, it's a node pulled from disk, skip
	if _, ok := db.nodes[child]; !ok {
		return
	}
	db.parents[child]++
	db.children[parent][child] = struct{}{}
}

// Dereference removes an existing reference from a parent node to a child node.
func (db *Database) Dereference(child common.Hash, parent common.Hash) {
	db.lock.Lock()
	defer db.lock.Unlock()

	nodes, storage, start := len(db.nodes), db.size, time.Now()
	db.dereference(child, parent)

	db.gcnodes += uint64(nodes - len(db.nodes))
	db.gcsize += storage - db.size
	db.gctime += time.Since(start)
}

// dereference is the private locked version of Dereference.
func (db *Database) dereference(child common.Hash, parent common.Hash) {
	// If the node does not exist, it's a previously comitted node.
	blob, ok := db.nodes[child]
	if !ok {
		return
	}
	delete(db.children[parent], child)
	db.parents[child]--

	// If there are no more references to the child, delete it and cascade
	if db.parents[child] == 0 {
		for child := range db.children[child] {
			db.dereference(child, child)
		}
		delete(db.nodes, child)
		delete(db.parents, child)
		delete(db.children, child)

		db.size -= common.StorageSize(common.HashLength + len(blob))
	}
}

// Commit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions.
//
// As a side effect, all pre-images accumulated up to this point are also written.
func (db *Database) Commit(node common.Hash, writer DatabaseWriter) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Write out all the accumulated trie node preimages
	for hash, preimage := range db.preimages {
		if err := writer.Put(db.secureKey(hash[:]), preimage); err != nil {
			log.Error("Failed to commit preimage from mempool", "err", err)
			return err
		}
		db.size -= common.StorageSize(common.HashLength + len(preimage))
	}
	db.preimages = make(map[common.Hash][]byte)

	// Write out the trie itself and dereference any flushed content
	nodes, storage, start := len(db.nodes), db.size, time.Now()
	if err := db.commit(node, writer); err != nil {
		log.Error("Failed to commit trie from mempool", "err", err)
		return err
	}
	log.Debug("Persistend trie from memory database", "nodes", nodes-len(db.nodes), "size", storage-db.size, "time", time.Since(start),
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.nodes), "livesize", db.size)

	// Reset the garbage collection statistics
	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0
	return nil
}

// commit is the private locked version of Commit.
func (db *Database) commit(node common.Hash, writer DatabaseWriter) error {
	// If the node does not exist, it's a previously comitted node.
	blob, ok := db.nodes[node]
	if !ok {
		return nil
	}
	for child := range db.children[node] {
		if err := db.commit(child, writer); err != nil {
			return err
		}
	}
	if err := writer.Put(node[:], blob); err != nil {
		return err
	}
	delete(db.nodes, node)
	delete(db.parents, node)
	delete(db.children, node)

	db.size -= common.StorageSize(common.HashLength + len(blob))
	return nil
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *Database) Size() common.StorageSize {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.size
}
