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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
)

const (
	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000

	// Cache size granted for caching clean code.
	codeCacheSize = 64 * 1024 * 1024

	// Number of address->curve point associations to keep.
	pointCacheSize = 4096
)

// Database wraps access to tries and contract code.
type Database interface {
	// Reader returns a state reader associated with the specified state root.
	Reader(root common.Hash) (Reader, error)

	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, trie Trie) (Trie, error)

	// ContractCode retrieves a particular contract's code.
	ContractCode(addr common.Address, codeHash common.Hash) ([]byte, error)

	// ContractCodeSize retrieves a particular contracts code's size.
	ContractCodeSize(addr common.Address, codeHash common.Hash) (int, error)

	// PointCache returns the cache holding points used in verkle tree key computation
	PointCache() *utils.PointCache

	// TrieDB returns the underlying trie database for managing trie nodes.
	TrieDB() *triedb.Database

	// Snapshot returns the underlying state snapshot.
	Snapshot() *snapshot.Tree
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

	// GetStorage returns the value for key stored in the trie. The value bytes
	// must not be modified by the caller. If a node was not found in the database,
	// a trie.MissingNodeError is returned.
	GetStorage(addr common.Address, key []byte) ([]byte, error)

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
	Witness() map[string]struct{}

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

// CachingDB is an implementation of Database interface. It leverages both trie and
// state snapshot to provide functionalities for state access. It's meant to be a
// long-live object and has a few caches inside for sharing between blocks.
type CachingDB struct {
	disk          ethdb.KeyValueStore
	triedb        *triedb.Database
	snap          *snapshot.Tree
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
	pointCache    *utils.PointCache
}

// NewDatabase creates a state database with the provided data sources.
func NewDatabase(triedb *triedb.Database, snap *snapshot.Tree) *CachingDB {
	return &CachingDB{
		disk:          triedb.Disk(),
		triedb:        triedb,
		snap:          snap,
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](codeSizeCacheSize),
		pointCache:    utils.NewPointCache(pointCacheSize),
	}
}

// NewDatabaseForTesting is similar to NewDatabase, but it initializes the caching
// db by using an ephemeral memory db with default config for testing.
func NewDatabaseForTesting() *CachingDB {
	return NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
}

// Reader returns a state reader associated with the specified state root.
func (db *CachingDB) Reader(stateRoot common.Hash) (Reader, error) {
	var readers []Reader

	// Set up the state snapshot reader if available. This feature
	// is optional and may be partially useful if it's not fully
	// generated.
	if db.snap != nil {
		snap := db.snap.Snapshot(stateRoot)
		if snap != nil {
			readers = append(readers, newStateReader(snap)) // snap reader is optional
		}
	}
	// Set up the trie reader, which is expected to always be available
	// as the gatekeeper unless the state is corrupted.
	tr, err := newTrieReader(stateRoot, db.triedb, db.pointCache)
	if err != nil {
		return nil, err
	}
	readers = append(readers, tr)

	return newMultiReader(readers...)
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *CachingDB) OpenTrie(root common.Hash) (Trie, error) {
	if db.triedb.IsVerkle() {
		return trie.NewVerkleTrie(root, db.triedb, db.pointCache)
	}
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), db.triedb)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenStorageTrie opens the storage trie of an account.
func (db *CachingDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, self Trie) (Trie, error) {
	// In the verkle case, there is only one tree. But the two-tree structure
	// is hardcoded in the codebase. So we need to return the same trie in this
	// case.
	if db.triedb.IsVerkle() {
		return self, nil
	}
	tr, err := trie.NewStateTrie(trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root), db.triedb)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// ContractCode retrieves a particular contract's code.
func (db *CachingDB) ContractCode(address common.Address, codeHash common.Hash) ([]byte, error) {
	code, _ := db.codeCache.Get(codeHash)
	if len(code) > 0 {
		return code, nil
	}
	code = rawdb.ReadCode(db.disk, codeHash)
	if len(code) > 0 {
		db.codeCache.Add(codeHash, code)
		db.codeSizeCache.Add(codeHash, len(code))
		return code, nil
	}
	return nil, errors.New("not found")
}

// ContractCodeWithPrefix retrieves a particular contract's code. If the
// code can't be found in the cache, then check the existence with **new**
// db scheme.
func (db *CachingDB) ContractCodeWithPrefix(address common.Address, codeHash common.Hash) ([]byte, error) {
	code, _ := db.codeCache.Get(codeHash)
	if len(code) > 0 {
		return code, nil
	}
	code = rawdb.ReadCodeWithPrefix(db.disk, codeHash)
	if len(code) > 0 {
		db.codeCache.Add(codeHash, code)
		db.codeSizeCache.Add(codeHash, len(code))
		return code, nil
	}
	return nil, errors.New("not found")
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *CachingDB) ContractCodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	if cached, ok := db.codeSizeCache.Get(codeHash); ok {
		return cached, nil
	}
	code, err := db.ContractCode(addr, codeHash)
	return len(code), err
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *CachingDB) TrieDB() *triedb.Database {
	return db.triedb
}

// PointCache returns the cache of evaluated curve points.
func (db *CachingDB) PointCache() *utils.PointCache {
	return db.pointCache
}

// Snapshot returns the underlying state snapshot.
func (db *CachingDB) Snapshot() *snapshot.Tree {
	return db.snap
}

// mustCopyTrie returns a deep-copied trie.
func mustCopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.StateTrie:
		return t.Copy()
	case *trie.VerkleTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}
