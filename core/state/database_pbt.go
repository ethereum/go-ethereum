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
	"github.com/ethereum/go-ethereum/trie/pbt"
	"github.com/ethereum/go-ethereum/triedb"
)

// PBTDatabase is an implementation of Database interface for the EIP-8297
// Partitioned Binary Tree, a single tree holding accounts, storage and code.
type PBTDatabase struct {
	triedb *triedb.Database
	codedb *CodeDB
}

// NewPBTDatabase creates a state database backed by the partitioned binary tree.
func NewPBTDatabase(triedb *triedb.Database, codedb *CodeDB) *PBTDatabase {
	if codedb == nil {
		codedb = NewCodeDB(triedb.Disk())
	}
	return &PBTDatabase{triedb: triedb, codedb: codedb}
}

// Type returns TypePBT.
func (db *PBTDatabase) Type() DatabaseType { return TypePBT }

// StateReader returns a state reader serving from the flat state first and
// the tree second.
func (db *PBTDatabase) StateReader(stateRoot common.Hash) (StateReader, error) {
	var readers []StateReader
	if reader, err := db.triedb.StateReader(stateRoot); err == nil {
		readers = append(readers, newFlatReader(reader))
	}
	tr, err := pbt.New(stateRoot, db.triedb)
	if err != nil {
		return nil, err
	}
	readers = append(readers, &ubtTrieReader{root: stateRoot, db: db.triedb, tr: tr})
	return newMultiStateReader(readers...)
}

// Reader returns a state reader associated with the specified state root.
func (db *PBTDatabase) Reader(stateRoot common.Hash) (Reader, error) {
	sr, err := db.StateReader(stateRoot)
	if err != nil {
		return nil, err
	}
	return newReader(db.codedb.Reader(), sr), nil
}

// ReadersWithCacheStats creates a pair of state readers sharing the same
// underlying state reader and cache, with separate statistics.
func (db *PBTDatabase) ReadersWithCacheStats(stateRoot common.Hash) (Reader, Reader, error) {
	r, err := db.StateReader(stateRoot)
	if err != nil {
		return nil, nil, err
	}
	sr := newStateReaderWithCache(r)
	return newReader(db.codedb.Reader(), newStateReaderWithStats(sr)), newReader(db.codedb.Reader(), newStateReaderWithStats(sr)), nil
}

// OpenTrie opens the tree at the given root.
func (db *PBTDatabase) OpenTrie(root common.Hash) (Trie, error) {
	return pbt.New(root, db.triedb)
}

// OpenStorageTrie returns the tree itself, which also holds all storage.
func (db *PBTDatabase) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, self Trie) (Trie, error) {
	return self, nil
}

// TrieDB returns the underlying trie database.
func (db *PBTDatabase) TrieDB() *triedb.Database {
	return db.triedb
}

// Commit writes the dirty code and hands the state update to the trie
// database.
func (db *PBTDatabase) Commit(update *StateUpdate) error {
	if update.Empty() {
		return nil
	}
	if len(update.Codes) > 0 {
		batch := db.codedb.NewBatchWithSize(len(update.Codes))
		for _, code := range update.Codes {
			batch.Put(code.Hash, code.Blob)
		}
		if err := batch.Commit(); err != nil {
			return err
		}
	}
	accounts, accountOrigin, storages, storageOrigin := update.EncodeMPTState()
	return db.triedb.Update(update.Root, update.OriginRoot, update.BlockNumber, update.Nodes, &triedb.StateSet{
		Accounts:       accounts,
		AccountsOrigin: accountOrigin,
		Storages:       storages,
		StoragesOrigin: storageOrigin,
		RawStorageKey:  update.StorageKeyType == StorageKeyPlain,
	})
}

// Iteratee returns a flat state iteratee of the given root.
func (db *PBTDatabase) Iteratee(root common.Hash) (Iteratee, error) {
	return newStateIteratee(false, root, db.triedb, nil)
}
