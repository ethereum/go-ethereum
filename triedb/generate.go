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

package triedb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb/internal"
)

// kvAccountIterator wraps an ethdb.Iterator to iterate over account snapshot
// entries in the database, implementing internal.AccountIterator.
type kvAccountIterator struct {
	it   ethdb.Iterator
	hash common.Hash
}

func newKVAccountIterator(db ethdb.Iteratee) *kvAccountIterator {
	it := rawdb.NewKeyLengthIterator(
		db.NewIterator(rawdb.SnapshotAccountPrefix, nil),
		len(rawdb.SnapshotAccountPrefix)+common.HashLength,
	)
	return &kvAccountIterator{it: it}
}

func (it *kvAccountIterator) Next() bool {
	if !it.it.Next() {
		return false
	}
	key := it.it.Key()
	copy(it.hash[:], key[len(rawdb.SnapshotAccountPrefix):])
	return true
}

func (it *kvAccountIterator) Hash() common.Hash { return it.hash }
func (it *kvAccountIterator) Account() []byte   { return it.it.Value() }
func (it *kvAccountIterator) Error() error      { return it.it.Error() }
func (it *kvAccountIterator) Release()          { it.it.Release() }

// kvStorageIterator wraps an ethdb.Iterator to iterate over storage snapshot
// entries for a specific account, implementing internal.StorageIterator.
type kvStorageIterator struct {
	it   ethdb.Iterator
	hash common.Hash
}

func newKVStorageIterator(db ethdb.Iteratee, accountHash common.Hash) *kvStorageIterator {
	it := rawdb.IterateStorageSnapshots(db, accountHash)
	return &kvStorageIterator{it: it}
}

func (it *kvStorageIterator) Next() bool {
	if !it.it.Next() {
		return false
	}
	key := it.it.Key()
	copy(it.hash[:], key[len(rawdb.SnapshotStoragePrefix)+common.HashLength:])
	return true
}

func (it *kvStorageIterator) Hash() common.Hash { return it.hash }
func (it *kvStorageIterator) Slot() []byte      { return it.it.Value() }
func (it *kvStorageIterator) Error() error      { return it.it.Error() }
func (it *kvStorageIterator) Release()          { it.it.Release() }

// GenerateTrie rebuilds all tries (storage + account) from flat snapshot data
// in the database. It reads account and storage snapshots from the KV store,
// builds tries using StackTrie with streaming node writes, and verifies the
// computed state root matches the expected root.
func GenerateTrie(db ethdb.Database, scheme string, root common.Hash) error {
	acctIt := newKVAccountIterator(db)
	defer acctIt.Release()

	got, err := internal.GenerateTrieRoot(db, scheme, acctIt, common.Hash{}, internal.StackTrieGenerate, func(dst ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *internal.GenerateStats) (common.Hash, error) {
		storageIt := newKVStorageIterator(db, accountHash)
		defer storageIt.Release()

		hash, err := internal.GenerateTrieRoot(dst, scheme, storageIt, accountHash, internal.StackTrieGenerate, nil, stat, false)
		if err != nil {
			return common.Hash{}, err
		}
		return hash, nil
	}, internal.NewGenerateStats(), true)
	if err != nil {
		return err
	}
	if got != root {
		return fmt.Errorf("state root mismatch: got %x, want %x", got, root)
	}
	return nil
}
