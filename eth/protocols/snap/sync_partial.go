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

package snap

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
)

// Database key prefix for tracking intentionally skipped storage during partial sync.
// These markers allow the healing phase to know which accounts had storage intentionally
// skipped (vs. accounts that need storage healing due to sync interruption).
var skippedStoragePrefix = []byte("SnapSkipped")

// Metrics for partial sync progress tracking
var (
	storageSkippedMeter  = metrics.NewRegisteredMeter("snap/sync/storage/skipped", nil)
	bytecodeSkippedMeter = metrics.NewRegisteredMeter("snap/sync/bytecode/skipped", nil)
)

// skippedStorageKey returns the database key for a skipped storage marker.
// The key format is: skippedStoragePrefix + accountHash (32 bytes)
func skippedStorageKey(accountHash common.Hash) []byte {
	return append(skippedStoragePrefix, accountHash.Bytes()...)
}

// markStorageSkipped records that storage was intentionally skipped for an account.
// This is used during partial sync to skip storage for contracts not in the configured list.
// The storageRoot is stored so we can verify consistency if needed.
func markStorageSkipped(db ethdb.KeyValueWriter, accountHash common.Hash, storageRoot common.Hash) {
	db.Put(skippedStorageKey(accountHash), storageRoot.Bytes())
}

// isStorageSkipped checks if storage was intentionally skipped for an account.
// Returns true if this account's storage was skipped during partial sync.
func isStorageSkipped(db ethdb.KeyValueReader, accountHash common.Hash) bool {
	has, _ := db.Has(skippedStorageKey(accountHash))
	return has
}

// deleteStorageSkipped removes the skip marker for an account.
// Used during cleanup or when re-syncing with different configuration.
func deleteStorageSkipped(db ethdb.KeyValueWriter, accountHash common.Hash) {
	db.Delete(skippedStorageKey(accountHash))
}

// shouldSyncStorage returns true if storage should be synced for this account hash.
// If no filter is configured (filter == nil), all storage is synced (full node behavior).
func (s *Syncer) shouldSyncStorage(accountHash common.Hash) bool {
	if s.filter == nil {
		return true // No filter = sync everything (full node)
	}
	return s.filter.ShouldSyncStorageByHash(accountHash)
}

// shouldSyncCode returns true if bytecode should be synced for this account hash.
// If no filter is configured (filter == nil), all bytecode is synced (full node behavior).
func (s *Syncer) shouldSyncCode(accountHash common.Hash) bool {
	if s.filter == nil {
		return true // No filter = sync everything (full node)
	}
	return s.filter.ShouldSyncCodeByHash(accountHash)
}

// isPartialSync returns true if partial sync mode is active.
func (s *Syncer) isPartialSync() bool {
	return s.filter != nil
}
