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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/partial"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestPartialSyncFilterStorage(t *testing.T) {
	// Create filter with specific contracts
	tracked := []common.Address{
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), // WETH
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // USDC
	}
	filter := partial.NewConfiguredFilter(tracked)

	// Verify tracked contracts pass filter by address
	for _, addr := range tracked {
		if !filter.ShouldSyncStorage(addr) {
			t.Errorf("Tracked contract %s should pass storage filter", addr.Hex())
		}
		if !filter.ShouldSyncCode(addr) {
			t.Errorf("Tracked contract %s should pass code filter", addr.Hex())
		}
		if !filter.IsTracked(addr) {
			t.Errorf("Tracked contract %s should be marked as tracked", addr.Hex())
		}
	}

	// Verify untracked contracts are filtered
	untracked := common.HexToAddress("0x1234567890123456789012345678901234567890")
	if filter.ShouldSyncStorage(untracked) {
		t.Error("Untracked contract should be filtered for storage")
	}
	if filter.ShouldSyncCode(untracked) {
		t.Error("Untracked contract should be filtered for code")
	}
	if filter.IsTracked(untracked) {
		t.Error("Untracked contract should not be marked as tracked")
	}

	// Verify hash-based filter works
	for _, addr := range tracked {
		trackedHash := crypto.Keccak256Hash(addr.Bytes())
		if !filter.ShouldSyncStorageByHash(trackedHash) {
			t.Errorf("Tracked contract hash %s should pass storage filter", trackedHash.Hex())
		}
		if !filter.ShouldSyncCodeByHash(trackedHash) {
			t.Errorf("Tracked contract hash %s should pass code filter", trackedHash.Hex())
		}
	}

	// Verify untracked hash is filtered
	untrackedHash := crypto.Keccak256Hash(untracked.Bytes())
	if filter.ShouldSyncStorageByHash(untrackedHash) {
		t.Error("Untracked contract hash should be filtered for storage")
	}
	if filter.ShouldSyncCodeByHash(untrackedHash) {
		t.Error("Untracked contract hash should be filtered for code")
	}
}

func TestAllowAllFilter(t *testing.T) {
	filter := &partial.AllowAllFilter{}

	// Any address should pass
	testAddresses := []common.Address{
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		common.HexToAddress("0x1234567890123456789012345678901234567890"),
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
	}

	for _, addr := range testAddresses {
		if !filter.ShouldSyncStorage(addr) {
			t.Errorf("AllowAllFilter should allow storage for %s", addr.Hex())
		}
		if !filter.ShouldSyncCode(addr) {
			t.Errorf("AllowAllFilter should allow code for %s", addr.Hex())
		}
		if !filter.IsTracked(addr) {
			t.Errorf("AllowAllFilter should mark %s as tracked", addr.Hex())
		}

		hash := crypto.Keccak256Hash(addr.Bytes())
		if !filter.ShouldSyncStorageByHash(hash) {
			t.Errorf("AllowAllFilter should allow storage by hash for %s", hash.Hex())
		}
		if !filter.ShouldSyncCodeByHash(hash) {
			t.Errorf("AllowAllFilter should allow code by hash for %s", hash.Hex())
		}
	}
}

func TestSkipMarkerPersistence(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	accountHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	storageRoot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	// Initially not skipped
	if isStorageSkipped(db, accountHash) {
		t.Error("Account should not be marked as skipped initially")
	}

	// Mark as skipped
	markStorageSkipped(db, accountHash, storageRoot)

	// Verify marker persists
	if !isStorageSkipped(db, accountHash) {
		t.Error("Skip marker should persist after write")
	}

	// Delete and verify
	deleteStorageSkipped(db, accountHash)
	if isStorageSkipped(db, accountHash) {
		t.Error("Skip marker should be removed after delete")
	}
}

func TestSyncerFilterMethods(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Test with nil filter (full node mode)
	syncer := NewSyncer(db, rawdb.HashScheme, nil)
	anyHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	if !syncer.shouldSyncStorage(anyHash) {
		t.Error("Nil filter should sync all storage")
	}
	if !syncer.shouldSyncCode(anyHash) {
		t.Error("Nil filter should sync all code")
	}
	if syncer.isPartialSync() {
		t.Error("Nil filter means not in partial sync mode")
	}

	// Test with configured filter (partial mode)
	tracked := []common.Address{
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
	}
	filter := partial.NewConfiguredFilter(tracked)
	partialSyncer := NewSyncer(db, rawdb.HashScheme, filter)

	if !partialSyncer.isPartialSync() {
		t.Error("Configured filter should indicate partial sync mode")
	}

	// Tracked contract should pass
	trackedHash := crypto.Keccak256Hash(tracked[0].Bytes())
	if !partialSyncer.shouldSyncStorage(trackedHash) {
		t.Error("Tracked contract should pass storage filter")
	}
	if !partialSyncer.shouldSyncCode(trackedHash) {
		t.Error("Tracked contract should pass code filter")
	}

	// Untracked contract should be filtered
	untrackedHash := crypto.Keccak256Hash(common.HexToAddress("0x1234").Bytes())
	if partialSyncer.shouldSyncStorage(untrackedHash) {
		t.Error("Untracked contract should be filtered for storage")
	}
	if partialSyncer.shouldSyncCode(untrackedHash) {
		t.Error("Untracked contract should be filtered for code")
	}
}

func TestConfiguredFilterContracts(t *testing.T) {
	tracked := []common.Address{
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
	}
	filter := partial.NewConfiguredFilter(tracked)

	// Verify Contracts() returns all tracked addresses
	contracts := filter.Contracts()
	if len(contracts) != len(tracked) {
		t.Errorf("Expected %d contracts, got %d", len(tracked), len(contracts))
	}

	// Check all tracked are in result (order may differ)
	for _, addr := range tracked {
		found := false
		for _, c := range contracts {
			if c == addr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Contract %s not found in Contracts() result", addr.Hex())
		}
	}
}
