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

package rawdb

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ReadStateHistoryIndexMetadata retrieves the metadata of state history index.
func ReadStateHistoryIndexMetadata(db ethdb.KeyValueReader) []byte {
	data, _ := db.Get(headStateHistoryIndexKey)
	return data
}

// WriteStateHistoryIndexMetadata stores the metadata of state history index
// into database.
func WriteStateHistoryIndexMetadata(db ethdb.KeyValueWriter, blob []byte) {
	if err := db.Put(headStateHistoryIndexKey, blob); err != nil {
		log.Crit("Failed to store the metadata of state history index", "err", err)
	}
}

// DeleteStateHistoryIndexMetadata removes the metadata of state history index.
func DeleteStateHistoryIndexMetadata(db ethdb.KeyValueWriter) {
	if err := db.Delete(headStateHistoryIndexKey); err != nil {
		log.Crit("Failed to delete the metadata of state history index", "err", err)
	}
}

// ReadAccountHistoryIndex retrieves the account history index with the provided
// account address.
func ReadAccountHistoryIndex(db ethdb.KeyValueReader, addressHash common.Hash) []byte {
	data, err := db.Get(accountHistoryIndexKey(addressHash))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteAccountHistoryIndex writes the provided account history index into database.
func WriteAccountHistoryIndex(db ethdb.KeyValueWriter, addressHash common.Hash, data []byte) {
	if err := db.Put(accountHistoryIndexKey(addressHash), data); err != nil {
		log.Crit("Failed to store account history index", "err", err)
	}
}

// DeleteAccountHistoryIndex deletes the specified account history index from
// the database.
func DeleteAccountHistoryIndex(db ethdb.KeyValueWriter, addressHash common.Hash) {
	if err := db.Delete(accountHistoryIndexKey(addressHash)); err != nil {
		log.Crit("Failed to delete account history index", "err", err)
	}
}

// ReadStorageHistoryIndex retrieves the storage history index with the provided
// account address and storage key hash.
func ReadStorageHistoryIndex(db ethdb.KeyValueReader, addressHash common.Hash, storageHash common.Hash) []byte {
	data, err := db.Get(storageHistoryIndexKey(addressHash, storageHash))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteStorageHistoryIndex writes the provided storage history index into database.
func WriteStorageHistoryIndex(db ethdb.KeyValueWriter, addressHash common.Hash, storageHash common.Hash, data []byte) {
	if err := db.Put(storageHistoryIndexKey(addressHash, storageHash), data); err != nil {
		log.Crit("Failed to store storage history index", "err", err)
	}
}

// DeleteStorageHistoryIndex deletes the specified state index from the database.
func DeleteStorageHistoryIndex(db ethdb.KeyValueWriter, addressHash common.Hash, storageHash common.Hash) {
	if err := db.Delete(storageHistoryIndexKey(addressHash, storageHash)); err != nil {
		log.Crit("Failed to delete storage history index", "err", err)
	}
}

// ReadAccountHistoryIndexBlock retrieves the index block with the provided
// account address along with the block id.
func ReadAccountHistoryIndexBlock(db ethdb.KeyValueReader, addressHash common.Hash, blockID uint32) []byte {
	data, err := db.Get(accountHistoryIndexBlockKey(addressHash, blockID))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteAccountHistoryIndexBlock writes the provided index block into database.
func WriteAccountHistoryIndexBlock(db ethdb.KeyValueWriter, addressHash common.Hash, blockID uint32, data []byte) {
	if err := db.Put(accountHistoryIndexBlockKey(addressHash, blockID), data); err != nil {
		log.Crit("Failed to store account index block", "err", err)
	}
}

// DeleteAccountHistoryIndexBlock deletes the specified index block from the database.
func DeleteAccountHistoryIndexBlock(db ethdb.KeyValueWriter, addressHash common.Hash, blockID uint32) {
	if err := db.Delete(accountHistoryIndexBlockKey(addressHash, blockID)); err != nil {
		log.Crit("Failed to delete account index block", "err", err)
	}
}

// ReadStorageHistoryIndexBlock retrieves the index block with the provided state
// identifier along with the block id.
func ReadStorageHistoryIndexBlock(db ethdb.KeyValueReader, addressHash common.Hash, storageHash common.Hash, blockID uint32) []byte {
	data, err := db.Get(storageHistoryIndexBlockKey(addressHash, storageHash, blockID))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteStorageHistoryIndexBlock writes the provided index block into database.
func WriteStorageHistoryIndexBlock(db ethdb.KeyValueWriter, addressHash common.Hash, storageHash common.Hash, id uint32, data []byte) {
	if err := db.Put(storageHistoryIndexBlockKey(addressHash, storageHash, id), data); err != nil {
		log.Crit("Failed to store storage index block", "err", err)
	}
}

// DeleteStorageHistoryIndexBlock deletes the specified index block from the database.
func DeleteStorageHistoryIndexBlock(db ethdb.KeyValueWriter, addressHash common.Hash, storageHash common.Hash, id uint32) {
	if err := db.Delete(storageHistoryIndexBlockKey(addressHash, storageHash, id)); err != nil {
		log.Crit("Failed to delete storage index block", "err", err)
	}
}

// increaseKey increase the input key by one bit. Return nil if the entire
// addition operation overflows.
func increaseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			return key
		}
	}
	return nil
}

// DeleteStateHistoryIndex completely removes all history indexing data, including
// indexes for accounts and storages.
//
// Note, this method assumes the storage space with prefix `StateHistoryIndexPrefix`
// is exclusively occupied by the history indexing data!
func DeleteStateHistoryIndex(db ethdb.KeyValueRangeDeleter) {
	start := StateHistoryIndexPrefix
	limit := increaseKey(bytes.Clone(StateHistoryIndexPrefix))

	// Try to remove the data in the range by a loop, as the leveldb
	// doesn't support the native range deletion.
	for {
		err := db.DeleteRange(start, limit)
		if err == nil {
			return
		}
		if errors.Is(err, ethdb.ErrTooManyKeys) {
			continue
		}
		log.Crit("Failed to delete history index range", "err", err)
	}
}
