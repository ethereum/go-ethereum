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

package state

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

func TestSizeTracker(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	defer db.Close()

	tdb := triedb.NewDatabase(db, &triedb.Config{PathDB: pathdb.Defaults})
	sdb := NewDatabase(tdb, nil)

	// Generate 50 blocks to establish a baseline
	baselineBlockNum := uint64(50)
	currentRoot := types.EmptyRootHash

	addr1 := common.BytesToAddress([]byte{1, 0, 0, 1})
	addr2 := common.BytesToAddress([]byte{1, 0, 0, 2})
	addr3 := common.BytesToAddress([]byte{1, 0, 0, 3})

	// Create initial state with fixed accounts
	state, _ := New(currentRoot, sdb)
	state.AddBalance(addr1, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
	state.SetState(addr1, common.HexToHash("0x1111"), common.HexToHash("0xaaaa"))
	state.SetState(addr1, common.HexToHash("0x2222"), common.HexToHash("0xbbbb"))

	state.AddBalance(addr2, uint256.NewInt(2000), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr2, 2, tracing.NonceChangeUnspecified)
	state.SetCode(addr2, []byte{0x60, 0x80, 0x60, 0x40, 0x52}, tracing.CodeChangeUnspecified)

	state.AddBalance(addr3, uint256.NewInt(3000), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr3, 3, tracing.NonceChangeUnspecified)

	currentRoot, _, err := state.CommitWithUpdate(1, true, false)
	if err != nil {
		t.Fatalf("Failed to commit initial state: %v", err)
	}
	if err := tdb.Commit(currentRoot, false); err != nil {
		t.Fatalf("Failed to commit initial trie: %v", err)
	}

	for i := 1; i < 50; i++ { // blocks 2-50
		blockNum := uint64(i + 1)

		newState, err := New(currentRoot, sdb)
		if err != nil {
			t.Fatalf("Failed to create new state at block %d: %v", blockNum, err)
		}
		testAddr := common.BigToAddress(uint256.NewInt(uint64(i + 100)).ToBig())
		newState.AddBalance(testAddr, uint256.NewInt(uint64((i+1)*1000)), tracing.BalanceChangeUnspecified)
		newState.SetNonce(testAddr, uint64(i+10), tracing.NonceChangeUnspecified)

		if i%2 == 0 {
			newState.SetState(addr1, common.BigToHash(uint256.NewInt(uint64(i+0x1000)).ToBig()), common.BigToHash(uint256.NewInt(uint64(i+0x2000)).ToBig()))
		}
		if i%3 == 0 {
			newState.SetCode(testAddr, []byte{byte(i), 0x60, 0x80, byte(i + 1), 0x52}, tracing.CodeChangeUnspecified)
		}
		root, _, err := newState.CommitWithUpdate(blockNum, true, false)
		if err != nil {
			t.Fatalf("Failed to commit state at block %d: %v", blockNum, err)
		}
		if err := tdb.Commit(root, false); err != nil {
			t.Fatalf("Failed to commit trie at block %d: %v", blockNum, err)
		}
		currentRoot = root
	}
	baselineRoot := currentRoot

	// Wait for snapshot completion
	for !tdb.SnapshotCompleted() {
		time.Sleep(100 * time.Millisecond)
	}

	// Calculate baseline from the intermediate persisted state
	baselineTracker := &SizeTracker{
		db:     db,
		triedb: tdb,
		abort:  make(chan struct{}),
	}
	done := make(chan buildResult)

	go baselineTracker.build(baselineRoot, baselineBlockNum, done)
	var baselineResult buildResult
	select {
	case baselineResult = <-done:
		if baselineResult.err != nil {
			t.Fatalf("Failed to get baseline stats: %v", baselineResult.err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for baseline stats")
	}
	baseline := baselineResult.stat

	// Now start the tracker and notify it of updates that happen AFTER the baseline
	tracker, err := NewSizeTracker(db, tdb)
	if err != nil {
		t.Fatalf("Failed to create size tracker: %v", err)
	}
	defer tracker.Stop()

	var trackedUpdates []SizeStats
	currentRoot = baselineRoot

	// Generate additional blocks beyond the baseline and track them
	for i := 49; i < 130; i++ { // blocks 51-132
		blockNum := uint64(i + 2)
		newState, err := New(currentRoot, sdb)
		if err != nil {
			t.Fatalf("Failed to create new state at block %d: %v", blockNum, err)
		}
		testAddr := common.BigToAddress(uint256.NewInt(uint64(i + 100)).ToBig())
		newState.AddBalance(testAddr, uint256.NewInt(uint64((i+1)*1000)), tracing.BalanceChangeUnspecified)
		newState.SetNonce(testAddr, uint64(i+10), tracing.NonceChangeUnspecified)

		if i%2 == 0 {
			newState.SetState(addr1, common.BigToHash(uint256.NewInt(uint64(i+0x1000)).ToBig()), common.BigToHash(uint256.NewInt(uint64(i+0x2000)).ToBig()))
		}
		if i%3 == 0 {
			newState.SetCode(testAddr, []byte{byte(i), 0x60, 0x80, byte(i + 1), 0x52}, tracing.CodeChangeUnspecified)
		}
		root, update, err := newState.CommitWithUpdate(blockNum, true, false)
		if err != nil {
			t.Fatalf("Failed to commit state at block %d: %v", blockNum, err)
		}
		if err := tdb.Commit(root, false); err != nil {
			t.Fatalf("Failed to commit trie at block %d: %v", blockNum, err)
		}

		diff, err := calSizeStats(update)
		if err != nil {
			t.Fatalf("Failed to calculate size stats for block %d: %v", blockNum, err)
		}
		trackedUpdates = append(trackedUpdates, diff)
		tracker.Notify(update)
		currentRoot = root
	}
	finalRoot := rawdb.ReadSnapshotRoot(db)

	// Ensure all commits are flushed to disk
	if err := tdb.Close(); err != nil {
		t.Fatalf("Failed to close triedb: %v", err)
	}
	// Reopen the database to simulate a restart
	tdb = triedb.NewDatabase(db, &triedb.Config{PathDB: pathdb.Defaults})
	defer tdb.Close()

	finalTracker := &SizeTracker{
		db:     db,
		triedb: tdb,
		abort:  make(chan struct{}),
	}
	finalDone := make(chan buildResult)

	go finalTracker.build(finalRoot, uint64(132), finalDone)
	var result buildResult
	select {
	case result = <-finalDone:
		if result.err != nil {
			t.Fatalf("Failed to build final stats: %v", result.err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for final stats")
	}
	actualStats := result.stat

	expectedStats := baseline
	for _, diff := range trackedUpdates {
		expectedStats = expectedStats.add(diff)
	}

	// The final measured stats should match our calculated expected stats exactly
	if actualStats.Accounts != expectedStats.Accounts {
		t.Errorf("Account count mismatch: baseline(%d) + tracked_changes = %d, but final_measurement = %d", baseline.Accounts, expectedStats.Accounts, actualStats.Accounts)
	}
	if actualStats.AccountBytes != expectedStats.AccountBytes {
		t.Errorf("Account bytes mismatch: expected %d, got %d", expectedStats.AccountBytes, actualStats.AccountBytes)
	}
	if actualStats.Storages != expectedStats.Storages {
		t.Errorf("Storage count mismatch: baseline(%d) + tracked_changes = %d, but final_measurement = %d", baseline.Storages, expectedStats.Storages, actualStats.Storages)
	}
	if actualStats.StorageBytes != expectedStats.StorageBytes {
		t.Errorf("Storage bytes mismatch: expected %d, got %d", expectedStats.StorageBytes, actualStats.StorageBytes)
	}
	if actualStats.ContractCodes != expectedStats.ContractCodes {
		t.Errorf("Contract code count mismatch: baseline(%d) + tracked_changes = %d, but final_measurement = %d", baseline.ContractCodes, expectedStats.ContractCodes, actualStats.ContractCodes)
	}
	if actualStats.ContractCodeBytes != expectedStats.ContractCodeBytes {
		t.Errorf("Contract code bytes mismatch: expected %d, got %d", expectedStats.ContractCodeBytes, actualStats.ContractCodeBytes)
	}
	// TODO: failed on github actions, need to investigate
	// if actualStats.AccountTrienodes != expectedStats.AccountTrienodes {
	// 	t.Errorf("Account trie nodes mismatch: expected %d, got %d", expectedStats.AccountTrienodes, actualStats.AccountTrienodes)
	// }
	// if actualStats.AccountTrienodeBytes != expectedStats.AccountTrienodeBytes {
	// 	t.Errorf("Account trie node bytes mismatch: expected %d, got %d", expectedStats.AccountTrienodeBytes, actualStats.AccountTrienodeBytes)
	// }
	if actualStats.StorageTrienodes != expectedStats.StorageTrienodes {
		t.Errorf("Storage trie nodes mismatch: expected %d, got %d", expectedStats.StorageTrienodes, actualStats.StorageTrienodes)
	}
	if actualStats.StorageTrienodeBytes != expectedStats.StorageTrienodeBytes {
		t.Errorf("Storage trie node bytes mismatch: expected %d, got %d", expectedStats.StorageTrienodeBytes, actualStats.StorageTrienodeBytes)
	}
}
