// Copyright 2019 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ReadSnapshotBlock retrieves the number and root of the block whose state is
// contained in the persisted snapshot.
func ReadSnapshotBlock(db ethdb.KeyValueReader) (uint64, common.Hash) {
	data, _ := db.Get(snapshotBlockKey)
	if len(data) != 8+common.HashLength {
		return 0, common.Hash{}
	}
	return binary.BigEndian.Uint64(data[:8]), common.BytesToHash(data[8:])
}

// WriteSnapshotBlock stores the number and root of the block whose state is
// contained in the persisted snapshot.
func WriteSnapshotBlock(db ethdb.KeyValueWriter, number uint64, root common.Hash) {
	if err := db.Put(snapshotBlockKey, append(encodeBlockNumber(number), root.Bytes()...)); err != nil {
		log.Crit("Failed to store snapsnot block's number and root", "err", err)
	}
}

// DeleteSnapshotBlock deletes the number and hash of the block whose state is
// contained in the persisted snapshot. Since snapshots are not immutable, this
// method can be used during updates, so a crash or failure will mark the entire
// snapshot invalid.
func DeleteSnapshotBlock(db ethdb.KeyValueWriter) {
	if err := db.Delete(snapshotBlockKey); err != nil {
		log.Crit("Failed to remove snapsnot block's number and hash", "err", err)
	}
}

// ReadAccountSnapshot retrieves the snapshot entry of an account trie leaf.
func ReadAccountSnapshot(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(accountSnapshotKey(hash))
	return data
}

// WriteAccountSnapshot stores the snapshot entry of an account trie leaf.
func WriteAccountSnapshot(db ethdb.KeyValueWriter, hash common.Hash, entry []byte) {
	if err := db.Put(accountSnapshotKey(hash), entry); err != nil {
		log.Crit("Failed to store account snapshot", "err", err)
	}
}

// DeleteAccountSnapshot removes the snapshot entry of an account trie leaf.
func DeleteAccountSnapshot(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(accountSnapshotKey(hash)); err != nil {
		log.Crit("Failed to delete account snapshot", "err", err)
	}
}

// ReadStorageSnapshot retrieves the snapshot entry of an storage trie leaf.
func ReadStorageSnapshot(db ethdb.KeyValueReader, accountHash, storageHash common.Hash) []byte {
	data, _ := db.Get(storageSnapshotKey(accountHash, storageHash))
	return data
}

// WriteStorageSnapshot stores the snapshot entry of an storage trie leaf.
func WriteStorageSnapshot(db ethdb.KeyValueWriter, accountHash, storageHash common.Hash, entry []byte) {
	if err := db.Put(storageSnapshotKey(accountHash, storageHash), entry); err != nil {
		log.Crit("Failed to store storage snapshot", "err", err)
	}
}

// DeleteStorageSnapshot removes the snapshot entry of an storage trie leaf.
func DeleteStorageSnapshot(db ethdb.KeyValueWriter, accountHash, storageHash common.Hash) {
	if err := db.Delete(storageSnapshotKey(accountHash, storageHash)); err != nil {
		log.Crit("Failed to delete storage snapshot", "err", err)
	}
}

// IterateStorageSnapshots returns an iterator for walking the entire storage
// space of a specific account.
func IterateStorageSnapshots(db ethdb.Iteratee, accountHash common.Hash) ethdb.Iterator {
	return db.NewIteratorWithPrefix(storageSnapshotsKey(accountHash))
}
