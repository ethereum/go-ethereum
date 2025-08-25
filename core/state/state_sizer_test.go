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
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

func TestSizeTracker(t *testing.T) {
	tempDir := t.TempDir()

	pdb, err := pebble.New(tempDir, 0, 0, "", false)
	if err != nil {
		t.Fatalf("Failed to create pebble database: %v", err)
	}
	defer pdb.Close()

	db := rawdb.NewDatabase(pdb)

	tdb := triedb.NewDatabase(db, &triedb.Config{PathDB: pathdb.Defaults})
	sdb := NewDatabase(tdb, nil)

	state, _ := New(types.EmptyRootHash, sdb)

	testAddr1 := common.HexToAddress("0x1234567890123456789012345678901234567890")
	testAddr2 := common.HexToAddress("0x2345678901234567890123456789012345678901")
	testAddr3 := common.HexToAddress("0x3456789012345678901234567890123456789012")

	state.AddBalance(testAddr1, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	state.SetNonce(testAddr1, 1, tracing.NonceChangeUnspecified)
	state.SetState(testAddr1, common.HexToHash("0x1111"), common.HexToHash("0xaaaa"))
	state.SetState(testAddr1, common.HexToHash("0x2222"), common.HexToHash("0xbbbb"))

	state.AddBalance(testAddr2, uint256.NewInt(2000), tracing.BalanceChangeUnspecified)
	state.SetNonce(testAddr2, 2, tracing.NonceChangeUnspecified)
	state.SetCode(testAddr2, []byte{0x60, 0x80, 0x60, 0x40, 0x52})

	state.AddBalance(testAddr3, uint256.NewInt(3000), tracing.BalanceChangeUnspecified)
	state.SetNonce(testAddr3, 3, tracing.NonceChangeUnspecified)

	root1, _, err := state.CommitWithUpdate(1, true, false)
	if err != nil {
		t.Fatalf("Failed to commit initial state: %v", err)
	}
	if err := tdb.Commit(root1, false); err != nil {
		t.Fatalf("Failed to commit trie: %v", err)
	}

	// Generate 50 blocks first to establish a baseline
	baselineBlockNum := uint64(50)
	currentRoot := root1

	for i := 0; i < 49; i++ { // blocks 2-50
		blockNum := uint64(i + 2)

		// Create new state from the previous committed root
		newState, err := New(currentRoot, sdb)
		if err != nil {
			t.Fatalf("Failed to create new state at block %d: %v", blockNum, err)
		}

		testAddr := common.BigToAddress(uint256.NewInt(uint64(i + 100)).ToBig())
		newState.AddBalance(testAddr, uint256.NewInt(uint64((i+1)*1000)), tracing.BalanceChangeUnspecified)
		newState.SetNonce(testAddr, uint64(i+10), tracing.NonceChangeUnspecified)

		if i%2 == 0 {
			newState.SetState(testAddr1, common.BigToHash(uint256.NewInt(uint64(i+0x1000)).ToBig()),
				common.BigToHash(uint256.NewInt(uint64(i+0x2000)).ToBig()))
		}

		if i%3 == 0 {
			newState.SetCode(testAddr, []byte{byte(i), 0x60, 0x80, byte(i + 1), 0x52})
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
	rawdb.WriteSnapshotRoot(db, baselineRoot)

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

	// Continue from where we left off (block 51+) and track those updates
	var trackedUpdates []SizeStats
	currentRoot = baselineRoot

	// Generate additional blocks beyond the baseline and track them
	for i := 49; i < 130; i++ { // blocks 51-132
		blockNum := uint64(i + 2)
		newState, _ := New(currentRoot, sdb)

		testAddr := common.BigToAddress(uint256.NewInt(uint64(i + 100)).ToBig())
		newState.AddBalance(testAddr, uint256.NewInt(uint64((i+1)*1000)), tracing.BalanceChangeUnspecified)
		newState.SetNonce(testAddr, uint64(i+10), tracing.NonceChangeUnspecified)

		if i%2 == 0 {
			newState.SetState(testAddr1, common.BigToHash(uint256.NewInt(uint64(i+0x1000)).ToBig()),
				common.BigToHash(uint256.NewInt(uint64(i+0x2000)).ToBig()))
		}

		if i%3 == 0 {
			newState.SetCode(testAddr, []byte{byte(i), 0x60, 0x80, byte(i + 1), 0x52})
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

	// Give the StateTracker time to process all the notifications we sent
	time.Sleep(100 * time.Millisecond)

	finalRoot := currentRoot

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

	// Now we have a proper test:
	// - Baseline measured at block 50 (with snapshot completion)
	// - Final state measured at block 132
	// - Tracked updates from blocks 51-132 (should show growth)

	// Verify that both baseline and final measurements show reasonable data
	if baseline.Accounts < 50 {
		t.Errorf("Expected baseline to have at least 50 accounts, got %d", baseline.Accounts)
	}
	if baseline.StorageBytes == 0 {
		t.Errorf("Expected baseline to have storage data, got 0 bytes")
	}

	if actualStats.Accounts <= baseline.Accounts {
		t.Errorf("Expected final state to have more accounts than baseline: baseline=%d, final=%d", baseline.Accounts, actualStats.Accounts)
	}

	if actualStats.StorageBytes <= baseline.StorageBytes {
		t.Errorf("Expected final state to have more storage than baseline: baseline=%d, final=%d", baseline.StorageBytes, actualStats.StorageBytes)
	}

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
	if actualStats.AccountTrienodes != expectedStats.AccountTrienodes {
		t.Errorf("Account trie nodes mismatch: expected %d, got %d", expectedStats.AccountTrienodes, actualStats.AccountTrienodes)
	}
	if actualStats.AccountTrienodeBytes != expectedStats.AccountTrienodeBytes {
		t.Errorf("Account trie node bytes mismatch: expected %d, got %d", expectedStats.AccountTrienodeBytes, actualStats.AccountTrienodeBytes)
	}
	if actualStats.StorageTrienodes != expectedStats.StorageTrienodes {
		t.Errorf("Storage trie nodes mismatch: expected %d, got %d", expectedStats.StorageTrienodes, actualStats.StorageTrienodes)
	}
	if actualStats.StorageTrienodeBytes != expectedStats.StorageTrienodeBytes {
		t.Errorf("Storage trie node bytes mismatch: expected %d, got %d", expectedStats.StorageTrienodeBytes, actualStats.StorageTrienodeBytes)
	}

	// Verify reasonable growth occurred
	accountGrowth := actualStats.Accounts - baseline.Accounts
	storageGrowth := actualStats.Storages - baseline.Storages
	codeGrowth := actualStats.ContractCodes - baseline.ContractCodes

	if accountGrowth <= 0 {
		t.Errorf("Expected account growth, got %d", accountGrowth)
	}
	if storageGrowth <= 0 {
		t.Errorf("Expected storage growth, got %d", storageGrowth)
	}
	if codeGrowth <= 0 {
		t.Errorf("Expected contract code growth, got %d", codeGrowth)
	}

	// Verify we successfully tracked updates from blocks 51-132
	expectedUpdates := 81 // blocks 51-132 (81 blocks)
	if len(trackedUpdates) < 70 || len(trackedUpdates) > expectedUpdates {
		t.Errorf("Expected 70-%d tracked updates, got %d", expectedUpdates, len(trackedUpdates))
	}

	t.Logf("Baseline stats:  Accounts=%d, AccountBytes=%d, Storages=%d, StorageBytes=%d, ContractCodes=%d",
		baseline.Accounts, baseline.AccountBytes, baseline.Storages, baseline.StorageBytes, baseline.ContractCodes)
	t.Logf("Expected stats:  Accounts=%d, AccountBytes=%d, Storages=%d, StorageBytes=%d, ContractCodes=%d",
		expectedStats.Accounts, expectedStats.AccountBytes, expectedStats.Storages, expectedStats.StorageBytes, expectedStats.ContractCodes)
	t.Logf("Final stats:     Accounts=%d, AccountBytes=%d, Storages=%d, StorageBytes=%d, ContractCodes=%d",
		actualStats.Accounts, actualStats.AccountBytes, actualStats.Storages, actualStats.StorageBytes, actualStats.ContractCodes)
	t.Logf("Growth:          Accounts=+%d, StorageSlots=+%d, ContractCodes=+%d",
		accountGrowth, storageGrowth, codeGrowth)
	t.Logf("Tracked %d state updates from %d blocks successfully", len(trackedUpdates), 81)
}
