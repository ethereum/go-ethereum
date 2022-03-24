// Copyright 2022 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// preimageStore is the store for caching preimages of node key.
type preimageStore struct {
	lock          sync.RWMutex
	disk          ethdb.KeyValueStore
	preimages     map[common.Hash][]byte // Preimages of nodes from the secure trie
	preimagesSize common.StorageSize     // Storage size of the preimages cache
}

// newPreimageStore initializes the store for caching preimages.
func newPreimageStore(disk ethdb.KeyValueStore) *preimageStore {
	return &preimageStore{
		disk:      disk,
		preimages: make(map[common.Hash][]byte),
	}
}

// insertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will NOT make a copy of the slice, only use if the
// preimage will NOT be changed later on.
func (db *preimageStore) insertPreimage(preimages map[common.Hash][]byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	for hash, preimage := range preimages {
		if _, ok := db.preimages[hash]; ok {
			continue
		}
		db.preimages[hash] = preimage
		db.preimagesSize += common.StorageSize(common.HashLength + len(preimage))
	}
}

// preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *preimageStore) preimage(hash common.Hash) []byte {
	db.lock.RLock()
	preimage := db.preimages[hash]
	db.lock.RUnlock()

	if preimage != nil {
		return preimage
	}
	return rawdb.ReadPreimage(db.disk, hash)
}

// commit flushes the cached preimages into the disk.
func (db *preimageStore) commit() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.preimagesSize <= 4*1024*1024 {
		return nil
	}
	batch := db.disk.NewBatch()
	rawdb.WritePreimages(batch, db.preimages)
	if err := batch.Write(); err != nil {
		return err
	}
	db.preimages, db.preimagesSize = make(map[common.Hash][]byte), 0
	return nil
}
