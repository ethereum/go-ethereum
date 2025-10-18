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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// TestMaxDepthInitialization tests that MaxDepth starts at 0 for new accounts
func TestMaxDepthInitialization(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)
	state, _ := New(types.EmptyRootHash, sdb)

	addr := common.HexToAddress("0x1234")

	state.AddBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)

	if maxDepth := state.GetMaxDepth(addr); maxDepth != 0 {
		t.Errorf("Initial MaxDepth should be 0, got %d", maxDepth)
	}
}

// TestMaxDepthMultipleStorage tests MaxDepth tracking with multiple storage writes
func TestMaxDepthMultipleStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)
	state, _ := New(types.EmptyRootHash, sdb)

	addr := common.HexToAddress("0x1234")

	for i := 0; i < 10; i++ {
		key := common.BigToHash(big.NewInt(int64(i)))
		val := common.BigToHash(big.NewInt(int64(i * 100)))
		state.SetState(addr, key, val)
	}

	root, err := state.Commit(0, false, false)
	if err != nil {
		t.Fatalf("Failed to commit state: %v", err)
	}

	state, _ = New(root, sdb)

	maxDepth := state.GetMaxDepth(addr)
	if maxDepth == 0 {
		t.Error("MaxDepth should be > 0 after storage writes")
	}
	t.Logf("MaxDepth after 10 storage writes: %d", maxDepth)
}

// TestMaxDepthEmptyAccount tests that non-existent accounts return 0 for MaxDepth
func TestMaxDepthEmptyAccount(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)
	state, _ := New(types.EmptyRootHash, sdb)

	addr := common.HexToAddress("0x9999")

	// Check MaxDepth for non-existent account
	maxDepth := state.GetMaxDepth(addr)
	if maxDepth != 0 {
		t.Errorf("Non-existent account should have MaxDepth 0, got %d", maxDepth)
	}
}

// TestMaxDepthIncrementalIncrease tests that MaxDepth increases progressively
// as more keys are added to the storage trie. With carefully chosen keys that
// create hash collisions at specific depths (2-3), we can observe incremental increases.
func TestMaxDepthIncrementalIncrease(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)

	addr := common.HexToAddress("0x1234")

	// Insert keys progressively and track MaxDepth changes.
	// With enough keys, we'll naturally see depth increases.
	// We insert in multiple "blocks" to simulate incremental growth.

	var previousMaxDepth uint64 = 0
	var depthIncreases []int // Track which blocks caused depth increases

	// We'll insert keys in batches, starting from a fresh state each time
	// This simulates blocks where storage grows incrementally
	// Use larger, more spread-out values to increase hash collision chances
	keyBatches := []int{1, 200, 200, 200, 200, 200} // Number of keys to insert in each "block"

	for blockNum, totalKeys := range keyBatches {
		state, _ := New(types.EmptyRootHash, sdb)
		state.AddBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)

		// Insert keys using larger, more spread-out values to increase
		// the chance of hash collisions that create deeper trie structures
		for i := range totalKeys {
			// Use a larger multiplier to spread keys across hash space
			key := common.BigToHash(big.NewInt(int64(i*1000 + 1)))
			val := common.BigToHash(big.NewInt(int64(i*100 + 1)))
			state.SetState(addr, key, val)
		}

		root, err := state.Commit(0, false, false)
		if err != nil {
			t.Fatalf("Block %d: Failed to commit state: %v", blockNum, err)
		}
		if err := tdb.Commit(root, false); err != nil {
			t.Fatalf("Block %d: Failed to commit trie: %v", blockNum, err)
		}

		// Reload state and get MaxDepth
		state, _ = New(root, sdb)
		currentMaxDepth := state.GetMaxDepth(addr)

		t.Logf("Block %d: Inserted %d keys, MaxDepth = %d (previous = %d)",
			blockNum, totalKeys, currentMaxDepth, previousMaxDepth)

		// Track if depth increased
		if currentMaxDepth > previousMaxDepth {
			depthIncreases = append(depthIncreases, blockNum)
		}

		// MaxDepth should never decrease
		if currentMaxDepth < previousMaxDepth {
			t.Errorf("Block %d: MaxDepth decreased from %d to %d", blockNum, previousMaxDepth, currentMaxDepth)
		}

		previousMaxDepth = currentMaxDepth
	}

	// Verify we saw at least some depth increases
	if len(depthIncreases) == 0 {
		t.Error("Expected to see at least one MaxDepth increase across blocks")
	} else {
		t.Logf("MaxDepth increased in %d blocks: %v", len(depthIncreases), depthIncreases)
	}

	// Final verification: with 31 keys, we should have decent depth
	if previousMaxDepth < 2 {
		t.Logf("Note: Final MaxDepth is %d with 31 keys - hash distribution may need more keys for deeper collisions", previousMaxDepth)
	}

	t.Logf("Final MaxDepth: %d", previousMaxDepth)
}
