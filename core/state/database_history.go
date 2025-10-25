// Copyright 2025 The go-ethereum Authors
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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// historicalStateReader wraps a historical state reader defined in path database,
// providing historic state serving over the path scheme.
type historicalStateReader struct {
	reader *pathdb.HistoricalStateReader
	lock   sync.Mutex // Lock for protecting concurrent read

}

// newHistoricalStateReader constructs a reader for historical state serving.
func newHistoricalStateReader(r *pathdb.HistoricalStateReader) *historicalStateReader {
	return &historicalStateReader{reader: r}
}

// Account implements StateReader, retrieving the account specified by the address.
//
// An error will be returned if the associated snapshot is already stale or
// the requested account is not yet covered by the snapshot.
//
// The returned account might be nil if it's not existent.
func (r *historicalStateReader) Account(addr common.Address) (*types.StateAccount, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	account, err := r.reader.Account(addr)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil
	}
	acct := &types.StateAccount{
		Nonce:    account.Nonce,
		Balance:  account.Balance,
		CodeHash: account.CodeHash,
		Root:     common.BytesToHash(account.Root),
	}
	if len(acct.CodeHash) == 0 {
		acct.CodeHash = types.EmptyCodeHash.Bytes()
	}
	if acct.Root == (common.Hash{}) {
		acct.Root = types.EmptyRootHash
	}
	return acct, nil
}

// Storage implements StateReader, retrieving the storage slot specified by the
// address and slot key.
//
// An error will be returned if the associated snapshot is already stale or
// the requested storage slot is not yet covered by the snapshot.
//
// The returned storage slot might be empty if it's not existent.
func (r *historicalStateReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	blob, err := r.reader.Storage(addr, key)
	if err != nil {
		return common.Hash{}, err
	}
	if len(blob) == 0 {
		return common.Hash{}, nil
	}
	_, content, _, err := rlp.Split(blob)
	if err != nil {
		return common.Hash{}, err
	}
	var slot common.Hash
	slot.SetBytes(content)
	return slot, nil
}

// historicalTrieOpener is a wrapper of pathdb.HistoricalNodeReader, implementing
// the database.NodeDatabase by adding NodeReader function.
type historicalTrieOpener struct {
	root   common.Hash
	reader *pathdb.HistoricalNodeReader
}

// newHistoricalTrieOpener constructs the historical trie opener.
func newHistoricalTrieOpener(root common.Hash, reader *pathdb.HistoricalNodeReader) *historicalTrieOpener {
	return &historicalTrieOpener{
		root:   root,
		reader: reader,
	}
}

// NodeReader implements database.NodeDatabase, returning a node reader of a
// specified state.
func (o *historicalTrieOpener) NodeReader(root common.Hash) (database.NodeReader, error) {
	if root != o.root {
		return nil, fmt.Errorf("state %x is not available", root)
	}
	return o.reader, nil
}

// historicalTrieReader wraps a historical node reader defined in path database,
// providing historical node serving over the path scheme.
type historicalTrieReader struct {
	root   common.Hash
	opener *historicalTrieOpener
	tr     Trie

	subRoots map[common.Address]common.Hash // Set of storage roots, cached when the account is resolved
	subTries map[common.Address]Trie        // Group of storage tries, cached when it's resolved
	lock     sync.Mutex                     // Lock for protecting concurrent read
}

// newHistoricalTrieReader constructs a reader for historical trie node serving.
func newHistoricalTrieReader(root common.Hash, r *pathdb.HistoricalNodeReader) (*historicalTrieReader, error) {
	opener := newHistoricalTrieOpener(root, r)
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), opener)
	if err != nil {
		return nil, err
	}
	return &historicalTrieReader{
		root:     root,
		opener:   opener,
		tr:       tr,
		subRoots: make(map[common.Address]common.Hash),
		subTries: make(map[common.Address]Trie),
	}, nil
}

// account is the inner version of Account and assumes the r.lock is already held.
func (r *historicalTrieReader) account(addr common.Address) (*types.StateAccount, error) {
	account, err := r.tr.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	if account == nil {
		r.subRoots[addr] = types.EmptyRootHash
	} else {
		r.subRoots[addr] = account.Root
	}
	return account, nil
}

// Account implements StateReader, retrieving the account specified by the address.
//
// An error will be returned if the associated snapshot is already stale or
// the requested account is not yet covered by the snapshot.
//
// The returned account might be nil if it's not existent.
func (r *historicalTrieReader) Account(addr common.Address) (*types.StateAccount, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.account(addr)
}

// Storage implements StateReader, retrieving the storage slot specified by the
// address and slot key.
//
// An error will be returned if the associated snapshot is already stale or
// the requested storage slot is not yet covered by the snapshot.
//
// The returned storage slot might be empty if it's not existent.
func (r *historicalTrieReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	tr, found := r.subTries[addr]
	if !found {
		root, ok := r.subRoots[addr]

		// The storage slot is accessed without account caching. It's unexpected
		// behavior but try to resolve the account first anyway.
		if !ok {
			_, err := r.account(addr)
			if err != nil {
				return common.Hash{}, err
			}
			root = r.subRoots[addr]
		}
		var err error
		tr, err = trie.NewStateTrie(trie.StorageTrieID(r.root, crypto.Keccak256Hash(addr.Bytes()), root), r.opener)
		if err != nil {
			return common.Hash{}, err
		}
		r.subTries[addr] = tr
	}
	ret, err := tr.GetStorage(addr, key.Bytes())
	if err != nil {
		return common.Hash{}, err
	}
	var value common.Hash
	value.SetBytes(ret)
	return value, nil
}

// HistoricalStateDB is the implementation of Database interface, with the ability to
// access historical state.
type HistoricalStateDB struct {
	disk          ethdb.KeyValueStore
	triedb        *triedb.Database
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
	pointCache    *utils.PointCache
}

// NewHistoricalDatabase creates a historical state database.
func NewHistoricalDatabase(disk ethdb.KeyValueStore, triedb *triedb.Database) *HistoricalStateDB {
	return &HistoricalStateDB{
		disk:          disk,
		triedb:        triedb,
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](codeSizeCacheSize),
		pointCache:    utils.NewPointCache(pointCacheSize),
	}
}

// Reader implements Database interface, returning a reader of the specific state.
func (db *HistoricalStateDB) Reader(stateRoot common.Hash) (Reader, error) {
	var readers []StateReader
	sr, err := db.triedb.HistoricalStateReader(stateRoot)
	if err == nil {
		readers = append(readers, newHistoricalStateReader(sr))
	}
	nr, err := db.triedb.HistoricalNodeReader(stateRoot)
	if err == nil {
		tr, err := newHistoricalTrieReader(stateRoot, nr)
		if err == nil {
			readers = append(readers, tr)
		}
	}
	if len(readers) == 0 {
		return nil, fmt.Errorf("historical state %x is not available", stateRoot)
	}
	combined, err := newMultiStateReader(readers...)
	if err != nil {
		return nil, err
	}
	return newReader(newCachingCodeReader(db.disk, db.codeCache, db.codeSizeCache), combined), nil
}

// OpenTrie opens the main account trie. It's not supported by historic database.
func (db *HistoricalStateDB) OpenTrie(root common.Hash) (Trie, error) {
	// TODO(rjl493456442) reuse the same historical node reader
	nr, err := db.triedb.HistoricalNodeReader(root)
	if err != nil {
		return nil, err
	}
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), newHistoricalTrieOpener(root, nr))
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenStorageTrie opens the storage trie of an account. It's not supported by
// historic database.
func (db *HistoricalStateDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, _ Trie) (Trie, error) {
	// TODO(rjl493456442) reuse the same historical node reader
	nr, err := db.triedb.HistoricalNodeReader(stateRoot)
	if err != nil {
		return nil, err
	}
	tr, err := trie.NewStateTrie(trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root), newHistoricalTrieOpener(stateRoot, nr))
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// PointCache returns the cache holding points used in verkle tree key computation
func (db *HistoricalStateDB) PointCache() *utils.PointCache {
	return db.pointCache
}

// TrieDB returns the underlying trie database for managing trie nodes.
func (db *HistoricalStateDB) TrieDB() *triedb.Database {
	return db.triedb
}

// Snapshot returns the underlying state snapshot.
func (db *HistoricalStateDB) Snapshot() *snapshot.Tree {
	return nil
}
