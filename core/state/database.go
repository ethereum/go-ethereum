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

package state

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/overlay"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/codedb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/transitiontrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// Database wraps access to tries and contract code.
type Database interface {
	// Reader returns a state reader associated with the specified state root.
	Reader(root common.Hash) (Reader, error)

	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, trie Trie) (Trie, error)

	// TrieDB returns the underlying trie database for managing trie nodes.
	TrieDB() *triedb.Database

	// Snapshot returns the underlying state snapshot.
	Snapshot() *snapshot.Tree

	// Commit flushes all pending writes and finalizes the state transition,
	// committing the changes to the underlying storage. It returns an error
	// if the commit fails.
	Commit(update *stateUpdate) error
}

// Trie is a Ethereum Merkle Patricia trie.
type Trie interface {
	// GetKey returns the sha3 preimage of a hashed key that was previously used
	// to store a value.
	//
	// TODO(fjl): remove this when StateTrie is removed
	GetKey([]byte) []byte

	// GetAccount abstracts an account read from the trie. It retrieves the
	// account blob from the trie with provided account address and decodes it
	// with associated decoding algorithm. If the specified account is not in
	// the trie, nil will be returned. If the trie is corrupted(e.g. some nodes
	// are missing or the account blob is incorrect for decoding), an error will
	// be returned.
	GetAccount(address common.Address) (*types.StateAccount, error)

	// PrefetchAccount attempts to resolve specific accounts from the database
	// to accelerate subsequent trie operations.
	PrefetchAccount([]common.Address) error

	// GetStorage returns the value for key stored in the trie. The value bytes
	// must not be modified by the caller. If a node was not found in the database,
	// a trie.MissingNodeError is returned.
	GetStorage(addr common.Address, key []byte) ([]byte, error)

	// PrefetchStorage attempts to resolve specific storage slots from the database
	// to accelerate subsequent trie operations.
	PrefetchStorage(addr common.Address, keys [][]byte) error

	// UpdateAccount abstracts an account write to the trie. It encodes the
	// provided account object with associated algorithm and then updates it
	// in the trie with provided address.
	UpdateAccount(address common.Address, account *types.StateAccount, codeLen int) error

	// UpdateStorage associates key with value in the trie. If value has length zero,
	// any existing value is deleted from the trie. The value bytes must not be modified
	// by the caller while they are stored in the trie. If a node was not found in the
	// database, a trie.MissingNodeError is returned.
	UpdateStorage(addr common.Address, key, value []byte) error

	// DeleteAccount abstracts an account deletion from the trie.
	DeleteAccount(address common.Address) error

	// DeleteStorage removes any existing value for key from the trie. If a node
	// was not found in the database, a trie.MissingNodeError is returned.
	DeleteStorage(addr common.Address, key []byte) error

	// UpdateContractCode abstracts code write to the trie. It is expected
	// to be moved to the stateWriter interface when the latter is ready.
	UpdateContractCode(address common.Address, codeHash common.Hash, code []byte) error

	// Hash returns the root hash of the trie. It does not write to the database and
	// can be used even if the trie doesn't have one.
	Hash() common.Hash

	// Commit collects all dirty nodes in the trie and replace them with the
	// corresponding node hash. All collected nodes(including dirty leaves if
	// collectLeaf is true) will be encapsulated into a nodeset for return.
	// The returned nodeset can be nil if the trie is clean(nothing to commit).
	// Once the trie is committed, it's not usable anymore. A new trie must
	// be created with new root and updated trie database for following usage
	Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet)

	// Witness returns a set containing all trie nodes that have been accessed.
	// The returned map could be nil if the witness is empty.
	Witness() map[string][]byte

	// NodeIterator returns an iterator that returns nodes of the trie. Iteration
	// starts at the key after the given start key. And error will be returned
	// if fails to create node iterator.
	NodeIterator(startKey []byte) (trie.NodeIterator, error)

	// Prove constructs a Merkle proof for key. The result contains all encoded nodes
	// on the path to the value at key. The value itself is also included in the last
	// node and can be retrieved by verifying the proof.
	//
	// If the trie does not contain a value for key, the returned proof contains all
	// nodes of the longest existing prefix of the key (at least the root), ending
	// with the node that proves the absence of the key.
	Prove(key []byte, proofDb ethdb.KeyValueWriter) error

	// IsVerkle returns true if the trie is verkle-tree based
	IsVerkle() bool
}

// CachingDB is an implementation of Database interface, providing functionalities
// for state access and update by leveraging configured data storage layers.
type CachingDB struct {
	triedb *triedb.Database
	codedb *codedb.Database
	snap   *snapshot.Tree

	// Transition-specific fields
	TransitionStatePerRoot *lru.Cache[common.Hash, *overlay.TransitionState]
}

// NewDatabase creates a state database with the provided data sources.
func NewDatabase(triedb *triedb.Database, codedb *codedb.Database) *CachingDB {
	return &CachingDB{
		triedb:                 triedb,
		codedb:                 codedb,
		TransitionStatePerRoot: lru.NewCache[common.Hash, *overlay.TransitionState](1000),
	}
}

// NewDatabaseForTesting is similar to NewDatabase, but it initializes the caching
// db by using an ephemeral memory db with default config for testing.
func NewDatabaseForTesting() *CachingDB {
	db := rawdb.NewMemoryDatabase()
	return NewDatabase(triedb.NewDatabase(db, nil), codedb.New(db))
}

// WithSnapshot configures the provided contract code cache. Note that this
// registration must be performed before the cachingDB is used.
func (db *CachingDB) WithSnapshot(snapshot *snapshot.Tree) *CachingDB {
	db.snap = snapshot
	return db
}

// Reader returns a state reader associated with the specified state root.
func (db *CachingDB) Reader(stateRoot common.Hash) (Reader, error) {
	var readers []StateReader

	// Configure the state reader using the standalone snapshot in hash mode.
	// This reader offers improved performance but is optional and only
	// partially useful if the snapshot is not fully generated.
	if db.TrieDB().Scheme() == rawdb.HashScheme && db.snap != nil {
		snap := db.snap.Snapshot(stateRoot)
		if snap != nil {
			readers = append(readers, newFlatReader(snap))
		}
	}
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

	combined, err := newMultiStateReader(readers...)
	if err != nil {
		return nil, err
	}
	return newReader(db.codedb.Reader(), combined), nil
}

// ReadersWithCacheStats creates a pair of state readers sharing the same internal cache and
// same backing Reader, but exposing separate statistics.
// and statistics.
func (db *CachingDB) ReadersWithCacheStats(stateRoot common.Hash) (ReaderWithStats, ReaderWithStats, error) {
	reader, err := db.Reader(stateRoot)
	if err != nil {
		return nil, nil, err
	}
	shared := newReaderWithCache(reader)
	return newReaderWithCacheStats(shared), newReaderWithCacheStats(shared), nil
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *CachingDB) OpenTrie(root common.Hash) (Trie, error) {
	if db.triedb.IsVerkle() {
		ts := overlay.LoadTransitionState(db.TrieDB().Disk(), root, db.triedb.IsVerkle())
		if ts.InTransition() {
			panic("state tree transition isn't supported yet")
		}
		if ts.Transitioned() {
			// Use BinaryTrie instead of VerkleTrie when IsVerkle is set
			// (IsVerkle actually means Binary Trie mode in this codebase)
			return bintrie.NewBinaryTrie(root, db.triedb)
		}
	}
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), db.triedb)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenStorageTrie opens the storage trie of an account.
func (db *CachingDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, self Trie) (Trie, error) {
	if db.triedb.IsVerkle() {
		return self, nil
	}
	tr, err := trie.NewStateTrie(trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root), db.triedb)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *CachingDB) TrieDB() *triedb.Database {
	return db.triedb
}

// Snapshot returns the underlying state snapshot.
func (db *CachingDB) Snapshot() *snapshot.Tree {
	return db.snap
}

// Commit flushes all pending writes and finalizes the state transition,
// committing the changes to the underlying storage. It returns an error
// if the commit fails.
func (db *CachingDB) Commit(update *stateUpdate) error {
	// Short circuit if nothing to commit
	if update.empty() {
		return nil
	}
	// Commit dirty contract code if any exists
	if len(update.codes) > 0 {
		writer := db.codedb.Writer()
		for _, code := range update.codes {
			writer.Put(code.hash, code.blob)
		}
		if err := writer.Commit(); err != nil {
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

// mustCopyTrie returns a deep-copied trie.
func mustCopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.StateTrie:
		return t.Copy()
	case *transitiontrie.TransitionTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}
