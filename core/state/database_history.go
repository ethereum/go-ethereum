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
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// historicReader wraps a historical state reader defined in path database,
// providing historic state serving over the path scheme.
//
// TODO(rjl493456442): historicReader is not thread-safe and does not fully
// comply with the StateReader interface requirements, needs to be fixed.
// Currently, it is only used in a non-concurrent context, so it is safe for now.
type historicReader struct {
	reader *pathdb.HistoricalStateReader
}

// newHistoricReader constructs a reader for historic state serving.
func newHistoricReader(r *pathdb.HistoricalStateReader) *historicReader {
	return &historicReader{reader: r}
}

// Account implements StateReader, retrieving the account specified by the address.
//
// An error will be returned if the associated snapshot is already stale or
// the requested account is not yet covered by the snapshot.
//
// The returned account might be nil if it's not existent.
func (r *historicReader) Account(addr common.Address) (*types.StateAccount, error) {
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
func (r *historicReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
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

// HistoricDB is the implementation of Database interface, with the ability to
// access historical state.
type HistoricDB struct {
	disk          ethdb.KeyValueStore
	triedb        *triedb.Database
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
	pointCache    *utils.PointCache
}

// NewHistoricDatabase creates a historic state database.
func NewHistoricDatabase(disk ethdb.KeyValueStore, triedb *triedb.Database) *HistoricDB {
	return &HistoricDB{
		disk:          disk,
		triedb:        triedb,
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](codeSizeCacheSize),
		pointCache:    utils.NewPointCache(pointCacheSize),
	}
}

// Reader implements Database interface, returning a reader of the specific state.
func (db *HistoricDB) Reader(stateRoot common.Hash) (Reader, error) {
	hr, err := db.triedb.HistoricReader(stateRoot)
	if err != nil {
		return nil, err
	}
	return newReader(newCachingCodeReader(db.disk, db.codeCache, db.codeSizeCache), newHistoricReader(hr)), nil
}

// OpenTrie opens the main account trie. It's not supported by historic database.
func (db *HistoricDB) OpenTrie(root common.Hash) (Trie, error) {
	return nil, errors.New("not implemented")
}

// OpenStorageTrie opens the storage trie of an account. It's not supported by
// historic database.
func (db *HistoricDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, trie Trie) (Trie, error) {
	return nil, errors.New("not implemented")
}

// PointCache returns the cache holding points used in verkle tree key computation
func (db *HistoricDB) PointCache() *utils.PointCache {
	return db.pointCache
}

// TrieDB returns the underlying trie database for managing trie nodes.
func (db *HistoricDB) TrieDB() *triedb.Database {
	return db.triedb
}

// Snapshot returns the underlying state snapshot.
func (db *HistoricDB) Snapshot() *snapshot.Tree {
	return nil
}
