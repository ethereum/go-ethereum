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
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func compareProgress(a legacyProgress, b SyncProgress) bool {
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
					common.Hash{0x1}: {
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

func convertLegacy(legacy legacyProgress) SyncProgress {
	var progress SyncProgress
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
	var dec SyncProgress
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
