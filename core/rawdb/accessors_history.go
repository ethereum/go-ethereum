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

// ReadTrienodeHistoryIndexMetadata retrieves the metadata of trienode history index.
func ReadTrienodeHistoryIndexMetadata(db ethdb.KeyValueReader) []byte {
	data, _ := db.Get(headTrienodeHistoryIndexKey)
	return data
}

// WriteTrienodeHistoryIndexMetadata stores the metadata of trienode history index
// into database.
func WriteTrienodeHistoryIndexMetadata(db ethdb.KeyValueWriter, blob []byte) {
	if err := db.Put(headTrienodeHistoryIndexKey, blob); err != nil {
		log.Crit("Failed to store the metadata of trienode history index", "err", err)
	}
}

// DeleteTrienodeHistoryIndexMetadata removes the metadata of trienode history index.
func DeleteTrienodeHistoryIndexMetadata(db ethdb.KeyValueWriter) {
	if err := db.Delete(headTrienodeHistoryIndexKey); err != nil {
		log.Crit("Failed to delete the metadata of trienode history index", "err", err)
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

// ReadTrienodeHistoryIndex retrieves the trienode history index with the provided
// account address and storage key hash.
func ReadTrienodeHistoryIndex(db ethdb.KeyValueReader, addressHash common.Hash, path []byte) []byte {
	data, err := db.Get(trienodeHistoryIndexKey(addressHash, path))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteTrienodeHistoryIndex writes the provided trienode history index into database.
func WriteTrienodeHistoryIndex(db ethdb.KeyValueWriter, addressHash common.Hash, path []byte, data []byte) {
	if err := db.Put(trienodeHistoryIndexKey(addressHash, path), data); err != nil {
		log.Crit("Failed to store trienode history index", "err", err)
	}
}

// DeleteTrienodeHistoryIndex deletes the specified trienode index from the database.
func DeleteTrienodeHistoryIndex(db ethdb.KeyValueWriter, addressHash common.Hash, path []byte) {
	if err := db.Delete(trienodeHistoryIndexKey(addressHash, path)); err != nil {
		log.Crit("Failed to delete trienode history index", "err", err)
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

// ReadTrienodeHistoryIndexBlock retrieves the index block with the provided state
// identifier along with the block id.
func ReadTrienodeHistoryIndexBlock(db ethdb.KeyValueReader, addressHash common.Hash, path []byte, blockID uint32) []byte {
	data, err := db.Get(trienodeHistoryIndexBlockKey(addressHash, path, blockID))
	if err != nil || len(data) == 0 {
		return nil
	}
	return data
}

// WriteTrienodeHistoryIndexBlock writes the provided index block into database.
func WriteTrienodeHistoryIndexBlock(db ethdb.KeyValueWriter, addressHash common.Hash, path []byte, id uint32, data []byte) {
	if err := db.Put(trienodeHistoryIndexBlockKey(addressHash, path, id), data); err != nil {
		log.Crit("Failed to store trienode index block", "err", err)
	}
}

// DeleteTrienodeHistoryIndexBlock deletes the specified index block from the database.
func DeleteTrienodeHistoryIndexBlock(db ethdb.KeyValueWriter, addressHash common.Hash, path []byte, id uint32) {
	if err := db.Delete(trienodeHistoryIndexBlockKey(addressHash, path, id)); err != nil {
		log.Crit("Failed to delete trienode index block", "err", err)
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

// DeleteStateHistoryIndexes completely removes all history indexing data, including
// indexes for accounts and storages.
func DeleteStateHistoryIndexes(db ethdb.KeyValueRangeDeleter) {
	DeleteHistoryByRange(db, StateHistoryAccountMetadataPrefix)
	DeleteHistoryByRange(db, StateHistoryStorageMetadataPrefix)
	DeleteHistoryByRange(db, StateHistoryAccountBlockPrefix)
	DeleteHistoryByRange(db, StateHistoryStorageBlockPrefix)
}

// DeleteTrienodeHistoryIndexes completely removes all trienode history indexing data.
func DeleteTrienodeHistoryIndexes(db ethdb.KeyValueRangeDeleter) {
	DeleteHistoryByRange(db, TrienodeHistoryMetadataPrefix)
	DeleteHistoryByRange(db, TrienodeHistoryBlockPrefix)
}

// DeleteHistoryByRange completely removes all database entries with the specific prefix.
// Note, this method assumes the space with the given prefix is exclusively occupied!
func DeleteHistoryByRange(db ethdb.KeyValueRangeDeleter, prefix []byte) {
	start := prefix
	limit := increaseKey(bytes.Clone(prefix))

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
