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

// Legacy sync progress definitions
type legacyStorageTask struct {
	Next common.Hash // Next account to sync in this interval
	Last common.Hash // Last account to sync in this interval
}

type legacyAccountTask struct {
	Next     common.Hash                          // Next account to sync in this interval
	Last     common.Hash                          // Last account to sync in this interval
	SubTasks map[common.Hash][]*legacyStorageTask // Storage intervals needing fetching for large contracts
}

type legacyProgress struct {
	Tasks []*legacyAccountTask // The suspended account tasks (contract tasks within)
}

func compareProgress(a legacyProgress, b syncProgress) bool {
	if len(a.Tasks) != len(b.Tasks) {
		return false
	}
	for i := 0; i < len(a.Tasks); i++ {
		if a.Tasks[i].Next != b.Tasks[i].Next {
			return false
		}
		if a.Tasks[i].Last != b.Tasks[i].Last {
			return false
		}
		// new fields are not checked here

		if len(a.Tasks[i].SubTasks) != len(b.Tasks[i].SubTasks) {
			return false
		}
		for addrHash, subTasksA := range a.Tasks[i].SubTasks {
			subTasksB, ok := b.Tasks[i].SubTasks[addrHash]
			if !ok || len(subTasksB) != len(subTasksA) {
				return false
			}
			for j := 0; j < len(subTasksA); j++ {
				if subTasksA[j].Next != subTasksB[j].Next {
					return false
				}
				if subTasksA[j].Last != subTasksB[j].Last {
					return false
				}
			}
		}
	}
	return true
}

func makeLegacyProgress() legacyProgress {
	return legacyProgress{
		Tasks: []*legacyAccountTask{
			{
				Next: common.Hash{},
				Last: common.Hash{0x77},
				SubTasks: map[common.Hash][]*legacyStorageTask{
					{0x1}: {
						{
							Next: common.Hash{},
							Last: common.Hash{0xff},
						},
					},
				},
			},
			{
				Next: common.Hash{0x88},
				Last: common.Hash{0xff},
			},
		},
	}
}

func convertLegacy(legacy legacyProgress) syncProgress {
	var progress syncProgress
	for i, task := range legacy.Tasks {
		subTasks := make(map[common.Hash][]*storageTask)
		for owner, list := range task.SubTasks {
			var cpy []*storageTask
			for i := 0; i < len(list); i++ {
				cpy = append(cpy, &storageTask{
					Next: list[i].Next,
					Last: list[i].Last,
				})
			}
			subTasks[owner] = cpy
		}
		accountTask := &accountTask{
			Next:     task.Next,
			Last:     task.Last,
			SubTasks: subTasks,
		}
		if i == 0 {
			accountTask.StorageCompleted = []common.Hash{{0xaa}, {0xbb}} // fulfill new fields
		}
		progress.Tasks = append(progress.Tasks, accountTask)
	}
	return progress
}

func TestSyncProgressCompatibility(t *testing.T) {
	// Decode serialized bytes of legacy progress, backward compatibility
	legacy := makeLegacyProgress()
	blob, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Failed to marshal progress %v", err)
	}
	var dec syncProgress
	if err := json.Unmarshal(blob, &dec); err != nil {
		t.Fatalf("Failed to unmarshal progress %v", err)
	}
	if !compareProgress(legacy, dec) {
		t.Fatal("sync progress is not backward compatible")
	}

	// Decode serialized bytes of new format progress
	progress := convertLegacy(legacy)
	blob, err = json.Marshal(progress)
	if err != nil {
		t.Fatalf("Failed to marshal progress %v", err)
	}
	var legacyDec legacyProgress
	if err := json.Unmarshal(blob, &legacyDec); err != nil {
		t.Fatalf("Failed to unmarshal progress %v", err)
	}
	if !compareProgress(legacyDec, progress) {
		t.Fatal("sync progress is not forward compatible")
	}
}

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

	syncer := newSyncerV2(db, rawdb.HashScheme)
	syncer.loadSyncStatus()

	if syncer.pivot != nil {
		t.Fatalf("expected pivot nil after discarding old format, got %+v", syncer.pivot)
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

	saver := newSyncerV2(db, rawdb.HashScheme)
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

	loader := newSyncerV2(db, rawdb.HashScheme)
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

	syncer := newSyncerV2(db, rawdb.HashScheme)
	syncer.loadSyncStatus()

	if syncer.pivot != nil {
		t.Fatalf("expected pivot nil after corrupt payload, got %+v", syncer.pivot)
	}
	if len(syncer.tasks) != accountConcurrency {
		t.Fatalf("expected fresh task split of %d, got %d", accountConcurrency, len(syncer.tasks))
	}
	if data := rawdb.ReadAccountSnapshot(db, orphanAccountHash); len(data) != 0 {
		t.Errorf("orphan account snapshot should be wiped, got %x", data)
	}
}
