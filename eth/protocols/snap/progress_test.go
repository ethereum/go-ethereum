// Copyright 2024 The go-ethereum Authors
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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestSyncProgressV1Discarded verifies that a persisted blob written in the
// old unversioned format (raw JSON, no version prefix) is detected and
// discarded on load, that the syncer falls through to a fresh start, and
// that any orphan flat-state entries from the prior format are wiped.
func TestSyncProgressV1Discarded(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Write a raw JSON blob (no version byte) to simulate progress persisted
	// by a prior geth binary (snap/1 format).
	legacy := map[string]any{
		"Root":        common.HexToHash("0xaaaa"),
		"BlockNumber": uint64(42),
		"Tasks":       []any{},
	}
	blob, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	rawdb.WriteSnapshotSyncStatus(db, blob)

	// Pre-write orphan flat-state entries that should be wiped on fresh start.
	orphanAccountHash := common.HexToHash("0xdeadbeef")
	rawdb.WriteAccountSnapshot(db, orphanAccountHash, []byte{0xde, 0xad})
	orphanStorageAccount := common.HexToHash("0xfeedface")
	orphanStorageSlot := common.HexToHash("0xabcd")
	rawdb.WriteStorageSnapshot(db, orphanStorageAccount, orphanStorageSlot, []byte{0xff, 0xff})

	syncer := NewSyncer(db, rawdb.HashScheme)
	syncer.loadSyncStatus()

	if syncer.previousPivot != nil {
		t.Fatalf("expected previousPivot nil after discarding old format, got %+v", syncer.previousPivot)
	}
	if len(syncer.tasks) != accountConcurrency {
		t.Fatalf("expected fresh task split of %d, got %d", accountConcurrency, len(syncer.tasks))
	}
	if data := rawdb.ReadAccountSnapshot(db, orphanAccountHash); len(data) != 0 {
		t.Errorf("orphan account snapshot should be wiped, got %x", data)
	}
	if val := rawdb.ReadStorageSnapshot(db, orphanStorageAccount, orphanStorageSlot); len(val) != 0 {
		t.Errorf("orphan storage snapshot should be wiped, got %x", val)
	}
}

// TestSyncProgressV2RoundTrip verifies that the persisted blob is framed
// with the expected version byte at offset 0, and that all six status
// counters survive the round-trip.
func TestSyncProgressV2RoundTrip(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	saver := NewSyncer(db, rawdb.HashScheme)
	saver.pivot = &types.Header{Number: new(big.Int).SetUint64(123), Difficulty: common.Big0}
	saver.accountSynced = 1
	saver.accountBytes = 2
	saver.bytecodeSynced = 3
	saver.bytecodeBytes = 4
	saver.storageSynced = 5
	saver.storageBytes = 6
	saver.saveSyncStatus()

	raw := rawdb.ReadSnapshotSyncStatus(db)
	if len(raw) == 0 || raw[0] != syncProgressVersion {
		t.Fatalf("expected version byte %d at offset 0, got blob %x", syncProgressVersion, raw)
	}

	loader := NewSyncer(db, rawdb.HashScheme)
	loader.loadSyncStatus()
	for _, c := range []struct {
		name string
		got  uint64
		want uint64
	}{
		{"accountSynced", loader.accountSynced, 1},
		{"accountBytes", uint64(loader.accountBytes), 2},
		{"bytecodeSynced", loader.bytecodeSynced, 3},
		{"bytecodeBytes", uint64(loader.bytecodeBytes), 4},
		{"storageSynced", loader.storageSynced, 5},
		{"storageBytes", uint64(loader.storageBytes), 6},
	} {
		if c.got != c.want {
			t.Errorf("%s mismatch: got %d, want %d", c.name, c.got, c.want)
		}
	}
}

// TestSyncProgressCorruptPayload verifies that a persisted blob with the
// correct version byte but unparseable JSON body is discarded, triggers a
// fresh-start fall-through (not a panic or a stale-state load), and the
// orphan flat state is wiped along with the corrupt status.
func TestSyncProgressCorruptPayload(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Version byte followed by garbage that isn't valid JSON.
	rawdb.WriteSnapshotSyncStatus(db, []byte{syncProgressVersion, 0x7b, 0x7b, 0x7b})

	// Pre-write orphan flat-state entries that should be wiped on fresh start.
	orphanAccountHash := common.HexToHash("0xdeadbeef")
	rawdb.WriteAccountSnapshot(db, orphanAccountHash, []byte{0xde, 0xad})

	syncer := NewSyncer(db, rawdb.HashScheme)
	syncer.loadSyncStatus()

	if syncer.previousPivot != nil {
		t.Fatalf("expected previousPivot nil after corrupt payload, got %+v", syncer.previousPivot)
	}
	if len(syncer.tasks) != accountConcurrency {
		t.Fatalf("expected fresh task split of %d, got %d", accountConcurrency, len(syncer.tasks))
	}
	if data := rawdb.ReadAccountSnapshot(db, orphanAccountHash); len(data) != 0 {
		t.Errorf("orphan account snapshot should be wiped, got %x", data)
	}
}
