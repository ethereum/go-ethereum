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
	"encoding/binary"

	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// balHistoryKey constructs the database key for a BAL at a given block number.
// Key format: balHistoryPrefix + block number (uint64 big endian)
func balHistoryKey(blockNum uint64) []byte {
	key := make([]byte, len(balHistoryPrefix)+8)
	copy(key, balHistoryPrefix)
	binary.BigEndian.PutUint64(key[len(balHistoryPrefix):], blockNum)
	return key
}

// ReadBALHistory retrieves the Block Access List for a specific block number.
// Returns nil if the BAL is not found or cannot be decoded.
func ReadBALHistory(db ethdb.KeyValueReader, blockNum uint64) *bal.BlockAccessList {
	data, err := db.Get(balHistoryKey(blockNum))
	if err != nil || len(data) == 0 {
		return nil
	}
	var accessList bal.BlockAccessList
	if err := rlp.DecodeBytes(data, &accessList); err != nil {
		log.Warn("Failed to decode BAL history", "block", blockNum, "err", err)
		return nil
	}
	return &accessList
}

// WriteBALHistory stores a Block Access List for a specific block number.
func WriteBALHistory(db ethdb.KeyValueWriter, blockNum uint64, accessList *bal.BlockAccessList) {
	data, err := rlp.EncodeToBytes(accessList)
	if err != nil {
		log.Crit("Failed to encode BAL history", "block", blockNum, "err", err)
	}
	if err := db.Put(balHistoryKey(blockNum), data); err != nil {
		log.Crit("Failed to store BAL history", "block", blockNum, "err", err)
	}
}

// DeleteBALHistory removes the Block Access List for a specific block number.
func DeleteBALHistory(db ethdb.KeyValueWriter, blockNum uint64) {
	if err := db.Delete(balHistoryKey(blockNum)); err != nil {
		log.Crit("Failed to delete BAL history", "block", blockNum, "err", err)
	}
}

// PruneBALHistory removes all BALs before the specified block number.
// This uses range iteration for safe, interruptible pruning.
func PruneBALHistory(db ethdb.Database, beforeBlock uint64) error {
	// Create iterator for BAL history range
	start := balHistoryKey(0)
	end := balHistoryKey(beforeBlock)

	// Use batch deletion for efficiency
	batch := db.NewBatch()
	it := db.NewIterator(balHistoryPrefix, start)
	defer it.Release()

	deleted := 0
	for it.Next() {
		key := it.Key()
		// Stop if we've passed the end key
		if len(key) >= len(balHistoryPrefix)+8 {
			blockNum := binary.BigEndian.Uint64(key[len(balHistoryPrefix):])
			if blockNum >= beforeBlock {
				break
			}
		}
		// Check if key is within our prefix
		if len(key) < len(balHistoryPrefix) {
			continue
		}
		for i := range balHistoryPrefix {
			if key[i] != balHistoryPrefix[i] {
				goto done
			}
		}
		batch.Delete(key)
		deleted++

		// Commit batch periodically to avoid memory buildup
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
	}
done:
	// Write remaining items
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			return err
		}
	}
	if deleted > 0 {
		log.Debug("Pruned BAL history", "deleted", deleted, "beforeBlock", beforeBlock)
	}
	_ = end // silence unused variable warning (used for documentation)
	return it.Error()
}

// HasBALHistory returns whether a BAL exists for the given block number.
func HasBALHistory(db ethdb.KeyValueReader, blockNum uint64) bool {
	has, _ := db.Has(balHistoryKey(blockNum))
	return has
}
