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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package rawdb

import (
	"encoding/binary"

	"github.com/xpaymentsorg/go-xpayments/common"
	"github.com/xpaymentsorg/go-xpayments/log"
	"github.com/xpaymentsorg/go-xpayments/xpsdb"
	// "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/ethdb"
	// "github.com/ethereum/go-ethereum/log"
)

// ReadSnapshotDisabled retrieves if the snapshot maintenance is disabled.
func ReadSnapshotDisabled(db xpsdb.KeyValueReader) bool {
	disabled, _ := db.Has(snapshotDisabledKey)
	return disabled
}

// WriteSnapshotDisabled stores the snapshot pause flag.
func WriteSnapshotDisabled(db xpsdb.KeyValueWriter) {
	if err := db.Put(snapshotDisabledKey, []byte("42")); err != nil {
		log.Crit("Failed to store snapshot disabled flag", "err", err)
	}
}

// DeleteSnapshotDisabled deletes the flag keeping the snapshot maintenance disabled.
func DeleteSnapshotDisabled(db xpsdb.KeyValueWriter) {
	if err := db.Delete(snapshotDisabledKey); err != nil {
		log.Crit("Failed to remove snapshot disabled flag", "err", err)
	}
}

// ReadSnapshotRoot retrieves the root of the block whose state is contained in
// the persisted snapshot.
func ReadSnapshotRoot(db xpsdb.KeyValueReader) common.Hash {
	data, _ := db.Get(SnapshotRootKey)
	if len(data) != common.HashLength {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteSnapshotRoot stores the root of the block whose state is contained in
// the persisted snapshot.
func WriteSnapshotRoot(db xpsdb.KeyValueWriter, root common.Hash) {
	if err := db.Put(SnapshotRootKey, root[:]); err != nil {
		log.Crit("Failed to store snapshot root", "err", err)
	}
}

// DeleteSnapshotRoot deletes the hash of the block whose state is contained in
// the persisted snapshot. Since snapshots are not immutable, this  method can
// be used during updates, so a crash or failure will mark the entire snapshot
// invalid.
func DeleteSnapshotRoot(db xpsdb.KeyValueWriter) {
	if err := db.Delete(SnapshotRootKey); err != nil {
		log.Crit("Failed to remove snapshot root", "err", err)
	}
}

// ReadAccountSnapshot retrieves the snapshot entry of an account trie leaf.
func ReadAccountSnapshot(db xpsdb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(accountSnapshotKey(hash))
	return data
}

// WriteAccountSnapshot stores the snapshot entry of an account trie leaf.
func WriteAccountSnapshot(db xpsdb.KeyValueWriter, hash common.Hash, entry []byte) {
	if err := db.Put(accountSnapshotKey(hash), entry); err != nil {
		log.Crit("Failed to store account snapshot", "err", err)
	}
}

// DeleteAccountSnapshot removes the snapshot entry of an account trie leaf.
func DeleteAccountSnapshot(db xpsdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(accountSnapshotKey(hash)); err != nil {
		log.Crit("Failed to delete account snapshot", "err", err)
	}
}

// ReadStorageSnapshot retrieves the snapshot entry of an storage trie leaf.
func ReadStorageSnapshot(db xpsdb.KeyValueReader, accountHash, storageHash common.Hash) []byte {
	data, _ := db.Get(storageSnapshotKey(accountHash, storageHash))
	return data
}

// WriteStorageSnapshot stores the snapshot entry of an storage trie leaf.
func WriteStorageSnapshot(db xpsdb.KeyValueWriter, accountHash, storageHash common.Hash, entry []byte) {
	if err := db.Put(storageSnapshotKey(accountHash, storageHash), entry); err != nil {
		log.Crit("Failed to store storage snapshot", "err", err)
	}
}

// DeleteStorageSnapshot removes the snapshot entry of an storage trie leaf.
func DeleteStorageSnapshot(db xpsdb.KeyValueWriter, accountHash, storageHash common.Hash) {
	if err := db.Delete(storageSnapshotKey(accountHash, storageHash)); err != nil {
		log.Crit("Failed to delete storage snapshot", "err", err)
	}
}

// IterateStorageSnapshots returns an iterator for walking the entire storage
// space of a specific account.
func IterateStorageSnapshots(db xpsdb.Iteratee, accountHash common.Hash) xpsdb.Iterator {
	return db.NewIterator(storageSnapshotsKey(accountHash), nil)
}

// ReadSnapshotJournal retrieves the serialized in-memory diff layers saved at
// the last shutdown. The blob is expected to be max a few 10s of megabytes.
func ReadSnapshotJournal(db xpsdb.KeyValueReader) []byte {
	data, _ := db.Get(snapshotJournalKey)
	return data
}

// WriteSnapshotJournal stores the serialized in-memory diff layers to save at
// shutdown. The blob is expected to be max a few 10s of megabytes.
func WriteSnapshotJournal(db xpsdb.KeyValueWriter, journal []byte) {
	if err := db.Put(snapshotJournalKey, journal); err != nil {
		log.Crit("Failed to store snapshot journal", "err", err)
	}
}

// DeleteSnapshotJournal deletes the serialized in-memory diff layers saved at
// the last shutdown
func DeleteSnapshotJournal(db xpsdb.KeyValueWriter) {
	if err := db.Delete(snapshotJournalKey); err != nil {
		log.Crit("Failed to remove snapshot journal", "err", err)
	}
}

// ReadSnapshotGenerator retrieves the serialized snapshot generator saved at
// the last shutdown.
func ReadSnapshotGenerator(db xpsdb.KeyValueReader) []byte {
	data, _ := db.Get(snapshotGeneratorKey)
	return data
}

// WriteSnapshotGenerator stores the serialized snapshot generator to save at
// shutdown.
func WriteSnapshotGenerator(db xpsdb.KeyValueWriter, generator []byte) {
	if err := db.Put(snapshotGeneratorKey, generator); err != nil {
		log.Crit("Failed to store snapshot generator", "err", err)
	}
}

// DeleteSnapshotGenerator deletes the serialized snapshot generator saved at
// the last shutdown
func DeleteSnapshotGenerator(db xpsdb.KeyValueWriter) {
	if err := db.Delete(snapshotGeneratorKey); err != nil {
		log.Crit("Failed to remove snapshot generator", "err", err)
	}
}

// ReadSnapshotRecoveryNumber retrieves the block number of the last persisted
// snapshot layer.
func ReadSnapshotRecoveryNumber(db xpsdb.KeyValueReader) *uint64 {
	data, _ := db.Get(snapshotRecoveryKey)
	if len(data) == 0 {
		return nil
	}
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// WriteSnapshotRecoveryNumber stores the block number of the last persisted
// snapshot layer.
func WriteSnapshotRecoveryNumber(db xpsdb.KeyValueWriter, number uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], number)
	if err := db.Put(snapshotRecoveryKey, buf[:]); err != nil {
		log.Crit("Failed to store snapshot recovery number", "err", err)
	}
}

// DeleteSnapshotRecoveryNumber deletes the block number of the last persisted
// snapshot layer.
func DeleteSnapshotRecoveryNumber(db xpsdb.KeyValueWriter) {
	if err := db.Delete(snapshotRecoveryKey); err != nil {
		log.Crit("Failed to remove snapshot recovery number", "err", err)
	}
}

// ReadSnapshotSyncStatus retrieves the serialized sync status saved at shutdown.
func ReadSnapshotSyncStatus(db xpsdb.KeyValueReader) []byte {
	data, _ := db.Get(snapshotSyncStatusKey)
	return data
}

// WriteSnapshotSyncStatus stores the serialized sync status to save at shutdown.
func WriteSnapshotSyncStatus(db xpsdb.KeyValueWriter, status []byte) {
	if err := db.Put(snapshotSyncStatusKey, status); err != nil {
		log.Crit("Failed to store snapshot sync status", "err", err)
	}
}
