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

package rawdb

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ReadShadowStateRoot returns the recorded shadow root of the given block: the
// root of the tree its header does not commit. The empty binary root is the
// zero hash, so presence is reported separately.
func ReadShadowStateRoot(db ethdb.KeyValueReader, hash common.Hash, number uint64) (common.Hash, bool) {
	data, _ := db.Get(shadowRootKey(number, hash))
	if len(data) != common.HashLength {
		return common.Hash{}, false
	}
	return common.BytesToHash(data), true
}

// WriteShadowStateRoot records the shadow root of the given block.
func WriteShadowStateRoot(db ethdb.KeyValueWriter, hash common.Hash, number uint64, root common.Hash) {
	if err := db.Put(shadowRootKey(number, hash), root.Bytes()); err != nil {
		log.Crit("Failed to store shadow state root", "err", err)
	}
}

// MigrationCursor is a follower direction's persisted replay position.
type MigrationCursor struct {
	Number uint64
	Hash   common.Hash
	Root   common.Hash
}

// ReadMigrationCursor returns the direction's cursor. A never-written cursor
// is a valid virgin state; a present but unreadable one errors, because
// mistaking it for virgin would let the seeding fallbacks run.
func ReadMigrationCursor(db ethdb.KeyValueReader, pbt bool) (MigrationCursor, bool, error) {
	key := migrationCursorKey(pbt)
	if has, err := db.Has(key); err != nil || !has {
		return MigrationCursor{}, false, err
	}
	data, err := db.Get(key)
	if err != nil {
		return MigrationCursor{}, false, err
	}
	if len(data) != 8+2*common.HashLength {
		return MigrationCursor{}, false, fmt.Errorf("migration cursor is %d bytes", len(data))
	}
	return MigrationCursor{
		Number: binary.BigEndian.Uint64(data),
		Hash:   common.BytesToHash(data[8 : 8+common.HashLength]),
		Root:   common.BytesToHash(data[8+common.HashLength:]),
	}, true, nil
}

// WriteMigrationCursor records the direction's replay position.
func WriteMigrationCursor(db ethdb.KeyValueWriter, pbt bool, c MigrationCursor) {
	data := append(append(encodeBlockNumber(c.Number), c.Hash.Bytes()...), c.Root.Bytes()...)
	if err := db.Put(migrationCursorKey(pbt), data); err != nil {
		log.Crit("Failed to store migration cursor", "err", err)
	}
}

func migrationCursorKey(pbt bool) []byte {
	if pbt {
		return pbtMigrationCursorKey
	}
	return mptMigrationCursorKey
}

// ReadPBTMigrationDone reports whether the migration finished.
func ReadPBTMigrationDone(db ethdb.KeyValueReader) bool {
	data, _ := db.Get(pbtMigrationDoneKey)
	return len(data) == 1 && data[0] == 1
}

// WritePBTMigrationDone marks the migration finished.
func WritePBTMigrationDone(db ethdb.KeyValueWriter) {
	if err := db.Put(pbtMigrationDoneKey, []byte{1}); err != nil {
		log.Crit("Failed to store migration done marker", "err", err)
	}
}

// ReadPBTMerkleDisposed reports whether the merkle state has been disposed
// of. Written before the deletion starts, so it means "gone or going".
func ReadPBTMerkleDisposed(db ethdb.KeyValueReader) bool {
	data, _ := db.Get(pbtMerkleDisposedKey)
	return len(data) == 1 && data[0] == 1
}

// WritePBTMerkleDisposed condemns the merkle state. It must land before the
// first key goes: a half-deleted state that still looks intact is the one
// shape a reader cannot tell from a complete one.
func WritePBTMerkleDisposed(db ethdb.KeyValueWriter) {
	if err := db.Put(pbtMerkleDisposedKey, []byte{1}); err != nil {
		log.Crit("Failed to store merkle disposal marker", "err", err)
	}
}

// DeleteMerkleState removes the merkle-patricia state: the snapshot markers
// first, so a half-deleted store cannot pass for a complete one, then the key
// families they blessed. Interrupt stops it between ranges; resuming is safe
// because the caller's marker outlives it. History freezers and the journal
// file belong to the caller.
func DeleteMerkleState(db ethdb.KeyValueStore, scheme string, interrupt <-chan struct{}) error {
	batch := db.NewBatch()
	DeleteSnapshotRoot(batch)
	DeleteSnapshotJournal(batch)
	DeleteSnapshotGenerator(batch)
	DeleteSnapshotRecoveryNumber(batch)
	DeleteSnapshotSyncStatus(batch)
	DeleteSnapshotDisabled(batch)
	// The merkle pathdb's singletons: history heads and the in-kv journal.
	DeleteStateHistoryIndexMetadata(batch)
	DeleteTrienodeHistoryIndexMetadata(batch)
	if err := batch.Delete(trieJournalKey); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		return err
	}
	stop := func(bool) bool {
		select {
		case <-interrupt:
			return true
		default:
			return false
		}
	}
	// On the hash scheme a trie node or code blob is keyed by its bare hash
	// and may begin with any family byte; SafeDeleteRange skips those.
	for _, prefix := range MerkleKeyFamilies {
		if err := SafeDeleteRange(db, prefix, increaseKey(bytes.Clone(prefix)), scheme == HashScheme, stop); err != nil {
			return err
		}
	}
	return deleteStateIDs(db, stop)
}

// deleteStateIDs removes the root-to-id mappings and the persistent id. Their
// leading byte is shared with every "Last..." chain head pointer, so they go
// by exact key length rather than by range.
func deleteStateIDs(db ethdb.KeyValueStore, stop func(bool) bool) error {
	it := NewKeyLengthIterator(db.NewIterator(stateIDPrefix, nil), len(stateIDPrefix)+common.HashLength)
	defer it.Release()

	batch := db.NewBatch()
	for it.Next() {
		if err := batch.Delete(common.CopyBytes(it.Key())); err != nil {
			return err
		}
		if batch.ValueSize() < ethdb.IdealBatchSize {
			continue
		}
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
		if stop(true) {
			return ErrDeleteRangeInterrupted
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	if err := batch.Delete(persistentStateIDKey); err != nil {
		return err
	}
	return batch.Write()
}

// WipeMigrationState clears the migration bookkeeping, done marker first,
// so a crash prefix leaves state the next boot detects, never a stale
// position a fresh anchor would lose to.
func WipeMigrationState(db ethdb.KeyValueStore) error {
	for _, key := range [][]byte{pbtMigrationDoneKey, pbtMigrationCursorKey, mptMigrationCursorKey} {
		if err := db.Delete(key); err != nil {
			return err
		}
	}
	var (
		it    = db.NewIterator(shadowRootPrefix, nil)
		batch = db.NewBatch()
	)
	defer it.Release()
	for it.Next() {
		if len(it.Key()) != len(shadowRootPrefix)+8+common.HashLength {
			continue
		}
		if err := batch.Delete(common.CopyBytes(it.Key())); err != nil {
			return err
		}
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	return batch.Write()
}
