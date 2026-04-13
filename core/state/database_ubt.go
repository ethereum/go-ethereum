// Copyright 2026 The go-ethereum Authors
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

package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
)

// UBTDB is an implementation of Database interface for Universal Binary Tries.
type UBTDB struct {
	triedb *triedb.Database
	codedb *CodeDB
	snap   *snapshot.Tree
}

// WithSnapshot configures the snapshot tree. Note that this registration must
// be performed before the UBTDB is used.
func (db *UBTDB) WithSnapshot(snapshot *snapshot.Tree) Database {
	db.snap = snapshot
	return db
}

// StateReader returns a state reader associated with the specified state root.
func (db *UBTDB) StateReader(stateRoot common.Hash) (StateReader, error) {
	var readers []StateReader

	// Configure the state reader using the path database in path mode.
	// This reader offers improved performance but is optional and only
	// partially useful if the snapshot data in path database is not
	// fully generated.
	if db.TrieDB().Scheme() == rawdb.PathScheme {
		reader, err := db.triedb.StateReader(stateRoot)
		if err == nil {
			readers = append(readers, newFlatReader(reader))
		}
	}
	// Configure the trie reader, which is expected to be available as the
	// gatekeeper unless the state is corrupted.
	tr, err := newTrieReader(stateRoot, db.triedb)
	if err != nil {
		return nil, err
	}
	readers = append(readers, tr)

	return newMultiStateReader(readers...)
}

// Reader implements Database, returning a reader associated with the specified
// state root.
func (db *UBTDB) Reader(stateRoot common.Hash) (Reader, error) {
	sr, err := db.StateReader(stateRoot)
	if err != nil {
		return nil, err
	}
	return newReader(db.codedb.Reader(), sr), nil
}

// ReadersWithCacheStats creates a pair of state readers that share the same
// underlying state reader and internal state cache, while maintaining separate
// statistics respectively.
func (db *UBTDB) ReadersWithCacheStats(stateRoot common.Hash) (Reader, Reader, error) {
	r, err := db.StateReader(stateRoot)
	if err != nil {
		return nil, nil, err
	}
	sr := newStateReaderWithCache(r)
	ra := newReader(db.codedb.Reader(), newStateReaderWithStats(sr))
	rb := newReader(db.codedb.Reader(), newStateReaderWithStats(sr))
	return ra, rb, nil
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *UBTDB) OpenTrie(root common.Hash) (Trie, error) {
	return bintrie.NewBinaryTrie(root, db.triedb)
}

// OpenStorageTrie opens the storage trie of an account. In binary trie mode,
// all state objects share one unified trie, so the main trie is returned.
func (db *UBTDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, self Trie) (Trie, error) {
	return self, nil
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *UBTDB) TrieDB() *triedb.Database {
	return db.triedb
}

// Snapshot returns the underlying state snapshot.
func (db *UBTDB) Snapshot() *snapshot.Tree {
	return db.snap
}

// Commit flushes all pending writes and finalizes the state transition,
// committing the changes to the underlying storage. It returns an error
// if the commit fails.
func (db *UBTDB) Commit(update *stateUpdate) error {
	// Short circuit if nothing to commit
	if update.empty() {
		return nil
	}
	// Commit dirty contract code if any exists
	if len(update.codes) > 0 {
		batch := db.codedb.NewBatchWithSize(len(update.codes))
		for _, code := range update.codes {
			batch.Put(code.hash, code.blob)
		}
		if err := batch.Commit(); err != nil {
			return err
		}
	}
	// If snapshotting is enabled, update the snapshot tree with this new version
	if db.snap != nil && db.snap.Snapshot(update.originRoot) != nil {
		if err := db.snap.Update(update.root, update.originRoot, update.accounts, update.storages); err != nil {
			log.Warn("Failed to update snapshot tree", "from", update.originRoot, "to", update.root, "err", err)
		}
		// Keep 128 diff layers in the memory, persistent layer is 129th.
		// - head layer is paired with HEAD state
		// - head-1 layer is paired with HEAD-1 state
		// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
		if err := db.snap.Cap(update.root, TriesInMemory); err != nil {
			log.Warn("Failed to cap snapshot tree", "root", update.root, "layers", TriesInMemory, "err", err)
		}
	}
	return db.triedb.Update(update.root, update.originRoot, update.blockNumber, update.nodes, update.stateSet())
}

// Iteratee returns a state iteratee associated with the specified state root,
// through which the account iterator and storage iterator can be created.
func (db *UBTDB) Iteratee(root common.Hash) (Iteratee, error) {
	return newStateIteratee(false, root, db.triedb, db.snap)
}
