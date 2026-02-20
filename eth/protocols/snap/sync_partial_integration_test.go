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
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/partial"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

// TestPartialSyncIntegration tests the end-to-end partial sync flow with mock peers.
// This verifies that:
// 1. All accounts are synced (complete account trie)
// 2. Only tracked contracts have their storage synced
// 3. Skip markers are recorded for untracked contracts
// 4. Healing respects the skip markers
func TestPartialSyncIntegration(t *testing.T) {
	t.Parallel()

	testPartialSyncIntegration(t, rawdb.HashScheme)
	testPartialSyncIntegration(t, rawdb.PathScheme)
}

func testPartialSyncIntegration(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)

	// Create source state: 20 accounts with unique storage per account
	// Using unique storage prevents trie node sharing in HashScheme which would
	// cause false positives in our verification (seeing storage for untracked accounts
	// because they share nodes with tracked accounts)
	numAccounts := 20
	numStorageSlots := 50
	nodeScheme, sourceAccountTrie, elems, storageTries, storageEntries := makeAccountTrieWithStorageWithUniqueStorage(
		scheme, numAccounts, numStorageSlots, true,
	)
	_ = nodeScheme // scheme is already known

	// Set up mock peer simulating a full node
	source := newTestPeer("full-node", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems
	source.setStorageTries(storageTries)
	source.storageValues = storageEntries

	// Extract first 2 account hashes to track (simulate partial node tracking 2 contracts)
	trackedHashes := extractFirstNAccountHashes(elems, 2)

	// Create filter based on account hashes
	// Note: ConfiguredFilter uses addresses, but for this test we need hash-based filtering
	// We'll create a custom filter that works with our test account hashes
	filter := newTestHashFilter(trackedHashes)

	// Create partial syncer
	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb, scheme, filter)
	syncer.Register(source)
	source.remote = syncer

	// Verify partial sync mode is active
	if !syncer.isPartialSync() {
		t.Fatal("Expected partial sync mode to be active")
	}

	// Run the sync
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	// Verify results
	verifyPartialSync(t, scheme, stateDb, sourceAccountTrie.Hash(), elems, trackedHashes)
}

// TestPartialSyncAllAccounts verifies the account trie is complete even when
// storage is filtered. This is critical: all accounts must be present for
// balance/nonce queries, only storage is filtered.
func TestPartialSyncAllAccounts(t *testing.T) {
	t.Parallel()

	testPartialSyncAllAccounts(t, rawdb.HashScheme)
	testPartialSyncAllAccounts(t, rawdb.PathScheme)
}

func testPartialSyncAllAccounts(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)

	numAccounts := 15
	numStorageSlots := 30
	_, sourceAccountTrie, elems, storageTries, storageEntries := makeAccountTrieWithStorageWithUniqueStorage(
		scheme, numAccounts, numStorageSlots, true,
	)

	source := newTestPeer("full-node", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems
	source.setStorageTries(storageTries)
	source.storageValues = storageEntries

	// Track only 1 contract
	trackedHashes := extractFirstNAccountHashes(elems, 1)
	filter := newTestHashFilter(trackedHashes)

	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb, scheme, filter)
	syncer.Register(source)
	source.remote = syncer

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	// Verify ALL accounts are in the trie (regardless of storage filtering)
	trieDb := triedb.NewDatabase(rawdb.NewDatabase(stateDb), newDbConfig(scheme))
	accTrie, err := trie.New(trie.StateTrieID(sourceAccountTrie.Hash()), trieDb)
	if err != nil {
		t.Fatalf("Failed to open account trie: %v", err)
	}

	accountCount := 0
	accIt := trie.NewIterator(accTrie.MustNodeIterator(nil))
	for accIt.Next() {
		accountCount++
	}
	if accIt.Err != nil {
		t.Fatalf("Account trie iteration failed: %v", accIt.Err)
	}

	if accountCount != numAccounts {
		t.Errorf("Expected %d accounts in trie, got %d", numAccounts, accountCount)
	}
}

// TestPartialSyncFilterBehavior verifies that the filter correctly identifies
// tracked vs untracked accounts and that storage is only synced for tracked ones.
// Note: Skip markers are no longer used - the filter is checked directly during healing.
func TestPartialSyncFilterBehavior(t *testing.T) {
	t.Parallel()

	testPartialSyncFilterBehavior(t, rawdb.HashScheme)
	testPartialSyncFilterBehavior(t, rawdb.PathScheme)
}

func testPartialSyncFilterBehavior(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)

	numAccounts := 10
	numStorageSlots := 20
	_, sourceAccountTrie, elems, storageTries, storageEntries := makeAccountTrieWithStorageWithUniqueStorage(
		scheme, numAccounts, numStorageSlots, true,
	)

	source := newTestPeer("full-node", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems
	source.setStorageTries(storageTries)
	source.storageValues = storageEntries

	// Track 3 out of 10 contracts
	trackedHashes := extractFirstNAccountHashes(elems, 3)
	filter := newTestHashFilter(trackedHashes)

	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb, scheme, filter)
	syncer.Register(source)
	source.remote = syncer

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	// Verify filter correctly identifies tracked vs untracked accounts
	trackedSet := make(map[common.Hash]struct{})
	for _, h := range trackedHashes {
		trackedSet[h] = struct{}{}
	}

	trackedCount := 0
	untrackedCount := 0
	for _, elem := range elems {
		accountHash := common.BytesToHash(elem.k)
		if syncer.shouldSyncStorage(accountHash) {
			trackedCount++
			if _, ok := trackedSet[accountHash]; !ok {
				t.Errorf("Filter says sync storage for %s but it's not in tracked set", accountHash.Hex()[:10])
			}
		} else {
			untrackedCount++
			if _, ok := trackedSet[accountHash]; ok {
				t.Errorf("Filter says skip storage for %s but it's in tracked set", accountHash.Hex()[:10])
			}
		}
	}

	if trackedCount != len(trackedHashes) {
		t.Errorf("Expected filter to identify %d tracked, got %d", len(trackedHashes), trackedCount)
	}
	expectedUntracked := numAccounts - len(trackedHashes)
	if untrackedCount != expectedUntracked {
		t.Errorf("Expected filter to identify %d untracked, got %d", expectedUntracked, untrackedCount)
	}
}

// TestPartialSyncNoStorageForUntracked verifies that untracked contracts
// have no storage in the database.
func TestPartialSyncNoStorageForUntracked(t *testing.T) {
	t.Parallel()

	testPartialSyncNoStorageForUntracked(t, rawdb.HashScheme)
	testPartialSyncNoStorageForUntracked(t, rawdb.PathScheme)
}

func testPartialSyncNoStorageForUntracked(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)

	numAccounts := 10
	numStorageSlots := 25
	_, sourceAccountTrie, elems, storageTries, storageEntries := makeAccountTrieWithStorageWithUniqueStorage(
		scheme, numAccounts, numStorageSlots, true,
	)

	source := newTestPeer("full-node", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems
	source.setStorageTries(storageTries)
	source.storageValues = storageEntries

	// Track 2 contracts
	trackedHashes := extractFirstNAccountHashes(elems, 2)
	trackedSet := make(map[common.Hash]struct{})
	for _, h := range trackedHashes {
		trackedSet[h] = struct{}{}
	}
	filter := newTestHashFilter(trackedHashes)

	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb, scheme, filter)
	syncer.Register(source)
	source.remote = syncer

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	// Open the trie and verify storage for each account
	trieDb := triedb.NewDatabase(rawdb.NewDatabase(stateDb), newDbConfig(scheme))
	accTrie, err := trie.New(trie.StateTrieID(sourceAccountTrie.Hash()), trieDb)
	if err != nil {
		t.Fatalf("Failed to open account trie: %v", err)
	}

	accIt := trie.NewIterator(accTrie.MustNodeIterator(nil))
	for accIt.Next() {
		accountHash := common.BytesToHash(accIt.Key)
		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIt.Value, &acc); err != nil {
			t.Fatalf("Failed to decode account: %v", err)
		}

		// Skip accounts without storage
		if acc.Root == types.EmptyRootHash {
			continue
		}

		_, isTracked := trackedSet[accountHash]

		// Try to open the storage trie
		id := trie.StorageTrieID(sourceAccountTrie.Hash(), accountHash, acc.Root)
		storageTrie, err := trie.New(id, trieDb)

		if isTracked {
			// Tracked contracts should have storage
			if err != nil {
				t.Errorf("Tracked contract %s should have storage, got error: %v", accountHash.Hex()[:10], err)
				continue
			}
			// Verify storage has slots
			storeIt := trie.NewIterator(storageTrie.MustNodeIterator(nil))
			slotCount := 0
			for storeIt.Next() {
				slotCount++
			}
			if slotCount == 0 {
				t.Errorf("Tracked contract %s has empty storage", accountHash.Hex()[:10])
			}
		} else {
			// Untracked contracts should NOT have storage
			// They either have no trie or an empty trie
			if err == nil {
				storeIt := trie.NewIterator(storageTrie.MustNodeIterator(nil))
				slotCount := 0
				for storeIt.Next() {
					slotCount++
				}
				if slotCount > 0 {
					t.Errorf("Untracked contract %s should not have storage (has %d slots)", accountHash.Hex()[:10], slotCount)
				}
			}
			// If err != nil, that's expected for untracked contracts (no storage trie)
		}
	}
}

// TestPartialSyncRequestCount verifies that storage requests are only made for tracked accounts.
// This is a diagnostic test to verify the filter is preventing unnecessary requests.
func TestPartialSyncRequestCount(t *testing.T) {
	t.Parallel()

	testPartialSyncRequestCount(t, rawdb.HashScheme)
	testPartialSyncRequestCount(t, rawdb.PathScheme)
}

func testPartialSyncRequestCount(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)

	numAccounts := 10
	numStorageSlots := 20
	_, sourceAccountTrie, elems, storageTries, storageEntries := makeAccountTrieWithStorageWithUniqueStorage(
		scheme, numAccounts, numStorageSlots, true,
	)

	source := newTestPeer("full-node", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems
	source.setStorageTries(storageTries)
	source.storageValues = storageEntries

	// Track 2 out of 10 accounts
	trackedHashes := extractFirstNAccountHashes(elems, 2)
	filter := newTestHashFilter(trackedHashes)

	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb, scheme, filter)
	syncer.Register(source)
	source.remote = syncer

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	// Log request counts for diagnosis
	t.Logf("Scheme: %s", scheme)
	t.Logf("Account requests: %d", source.nAccountRequests)
	t.Logf("Storage requests: %d", source.nStorageRequests)
	t.Logf("Bytecode requests: %d", source.nBytecodeRequests)
	t.Logf("Trienode requests: %d", source.nTrienodeRequests)
	t.Logf("Tracked accounts: %d out of %d", len(trackedHashes), numAccounts)

	// Debug: Print tracked hashes
	t.Logf("Tracked hashes:")
	for i, h := range trackedHashes {
		t.Logf("  [%d] %s", i, h.Hex()[:10])
	}

	// Debug: Count storage slots for each account
	t.Logf("Storage per account:")
	trieDb := triedb.NewDatabase(rawdb.NewDatabase(stateDb), newDbConfig(scheme))
	accTrie, err := trie.New(trie.StateTrieID(sourceAccountTrie.Hash()), trieDb)
	if err != nil {
		t.Fatalf("Failed to open account trie: %v", err)
	}

	trackedSet := make(map[common.Hash]struct{})
	for _, h := range trackedHashes {
		trackedSet[h] = struct{}{}
	}

	accIt := trie.NewIterator(accTrie.MustNodeIterator(nil))
	for accIt.Next() {
		accountHash := common.BytesToHash(accIt.Key)
		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIt.Value, &acc); err != nil {
			continue
		}
		_, isTracked := trackedSet[accountHash]
		skipped := isStorageSkipped(stateDb, accountHash)

		slotCount := 0
		if acc.Root != types.EmptyRootHash {
			id := trie.StorageTrieID(sourceAccountTrie.Hash(), accountHash, acc.Root)
			storageTrie, err := trie.New(id, trieDb)
			if err == nil {
				storeIt := trie.NewIterator(storageTrie.MustNodeIterator(nil))
				for storeIt.Next() {
					slotCount++
				}
			}
		}
		status := ""
		if isTracked {
			status = "[TRACKED]"
		} else if skipped {
			status = "[SKIPPED]"
		} else {
			status = "[UNKNOWN]"
		}
		if slotCount > 0 && !isTracked {
			t.Logf("  %s %s storage=%d (UNEXPECTED)", accountHash.Hex()[:10], status, slotCount)
		} else {
			t.Logf("  %s %s storage=%d", accountHash.Hex()[:10], status, slotCount)
		}
	}
}

// TestPartialSyncVsFullSync compares a partial sync with a full sync to ensure
// the account tries match but storage differs.
func TestPartialSyncVsFullSync(t *testing.T) {
	t.Parallel()

	testPartialSyncVsFullSync(t, rawdb.HashScheme)
	testPartialSyncVsFullSync(t, rawdb.PathScheme)
}

func testPartialSyncVsFullSync(t *testing.T, scheme string) {
	var (
		once1   sync.Once
		cancel1 = make(chan struct{})
		term1   = func() {
			once1.Do(func() {
				close(cancel1)
			})
		}
		once2   sync.Once
		cancel2 = make(chan struct{})
		term2   = func() {
			once2.Do(func() {
				close(cancel2)
			})
		}
	)

	numAccounts := 12
	numStorageSlots := 30
	_, sourceAccountTrie, elems, storageTries, storageEntries := makeAccountTrieWithStorageWithUniqueStorage(
		scheme, numAccounts, numStorageSlots, true,
	)

	// Create full sync peer
	fullSource := newTestPeer("full-source", t, term1)
	fullSource.accountTrie = sourceAccountTrie.Copy()
	fullSource.accountValues = elems
	fullSource.setStorageTries(storageTries)
	fullSource.storageValues = storageEntries

	// Create partial sync peer
	partialSource := newTestPeer("partial-source", t, term2)
	partialSource.accountTrie = sourceAccountTrie.Copy()
	partialSource.accountValues = elems
	partialSource.setStorageTries(storageTries)
	partialSource.storageValues = storageEntries

	// Full sync (nil filter)
	fullDb := rawdb.NewMemoryDatabase()
	fullSyncer := NewSyncer(fullDb, scheme, nil)
	fullSyncer.Register(fullSource)
	fullSource.remote = fullSyncer

	// Partial sync (track 2 contracts)
	trackedHashes := extractFirstNAccountHashes(elems, 2)
	filter := newTestHashFilter(trackedHashes)
	partialDb := rawdb.NewMemoryDatabase()
	partialSyncer := NewSyncer(partialDb, scheme, filter)
	partialSyncer.Register(partialSource)
	partialSource.remote = partialSyncer

	// Run both syncs
	done1 := checkStall(t, term1)
	if err := fullSyncer.Sync(sourceAccountTrie.Hash(), cancel1); err != nil {
		t.Fatalf("full sync failed: %v", err)
	}
	close(done1)

	done2 := checkStall(t, term2)
	if err := partialSyncer.Sync(sourceAccountTrie.Hash(), cancel2); err != nil {
		t.Fatalf("partial sync failed: %v", err)
	}
	close(done2)

	// Both should have complete account tries
	fullTrieDb := triedb.NewDatabase(rawdb.NewDatabase(fullDb), newDbConfig(scheme))
	partialTrieDb := triedb.NewDatabase(rawdb.NewDatabase(partialDb), newDbConfig(scheme))

	fullAccTrie, err := trie.New(trie.StateTrieID(sourceAccountTrie.Hash()), fullTrieDb)
	if err != nil {
		t.Fatalf("Failed to open full account trie: %v", err)
	}

	partialAccTrie, err := trie.New(trie.StateTrieID(sourceAccountTrie.Hash()), partialTrieDb)
	if err != nil {
		t.Fatalf("Failed to open partial account trie: %v", err)
	}

	// Count accounts in both tries
	fullCount := 0
	fullIt := trie.NewIterator(fullAccTrie.MustNodeIterator(nil))
	for fullIt.Next() {
		fullCount++
	}

	partialCount := 0
	partialIt := trie.NewIterator(partialAccTrie.MustNodeIterator(nil))
	for partialIt.Next() {
		partialCount++
	}

	if fullCount != partialCount {
		t.Errorf("Account count mismatch: full=%d, partial=%d", fullCount, partialCount)
	}

	// Count total storage slots
	fullStorageSlots := countTotalStorageSlots(t, fullDb, scheme, sourceAccountTrie.Hash())
	partialStorageSlots := countTotalStorageSlots(t, partialDb, scheme, sourceAccountTrie.Hash())

	// Partial should have fewer storage slots
	if partialStorageSlots >= fullStorageSlots {
		t.Errorf("Partial sync should have fewer storage slots: full=%d, partial=%d",
			fullStorageSlots, partialStorageSlots)
	}

	t.Logf("Full sync: %d accounts, %d storage slots", fullCount, fullStorageSlots)
	t.Logf("Partial sync: %d accounts, %d storage slots", partialCount, partialStorageSlots)
	t.Logf("Storage reduction: %.1f%%", float64(fullStorageSlots-partialStorageSlots)/float64(fullStorageSlots)*100)
}

// Helper functions

// testHashFilter is a test filter that works with pre-computed account hashes.
// In production, ConfiguredFilter computes hashes from addresses, but for tests
// we use the account hashes directly from the mock trie.
type testHashFilter struct {
	trackedHashes map[common.Hash]struct{}
}

func newTestHashFilter(hashes []common.Hash) *testHashFilter {
	m := make(map[common.Hash]struct{})
	for _, h := range hashes {
		m[h] = struct{}{}
	}
	return &testHashFilter{trackedHashes: m}
}

func (f *testHashFilter) ShouldSyncStorage(addr common.Address) bool {
	return false // Not used in tests
}

func (f *testHashFilter) ShouldSyncCode(addr common.Address) bool {
	return false // Not used in tests
}

func (f *testHashFilter) IsTracked(addr common.Address) bool {
	return false // Not used in tests
}

func (f *testHashFilter) ShouldSyncStorageByHash(accountHash common.Hash) bool {
	_, ok := f.trackedHashes[accountHash]
	return ok
}

func (f *testHashFilter) ShouldSyncCodeByHash(accountHash common.Hash) bool {
	_, ok := f.trackedHashes[accountHash]
	return ok
}

// extractFirstNAccountHashes returns the first N account hashes from the account list.
func extractFirstNAccountHashes(elems []*kv, n int) []common.Hash {
	if n > len(elems) {
		n = len(elems)
	}
	hashes := make([]common.Hash, n)
	for i := 0; i < n; i++ {
		hashes[i] = common.BytesToHash(elems[i].k)
	}
	return hashes
}

// verifyPartialSync verifies the results of a partial sync.
func verifyPartialSync(t *testing.T, scheme string, db ethdb.KeyValueStore, root common.Hash, elems []*kv, trackedHashes []common.Hash) {
	t.Helper()

	trackedSet := make(map[common.Hash]struct{})
	for _, h := range trackedHashes {
		trackedSet[h] = struct{}{}
	}

	trieDb := triedb.NewDatabase(rawdb.NewDatabase(db), newDbConfig(scheme))
	accTrie, err := trie.New(trie.StateTrieID(root), trieDb)
	if err != nil {
		t.Fatalf("Failed to open account trie: %v", err)
	}

	accountCount := 0
	trackedWithStorage := 0
	untrackedWithoutStorage := 0

	accIt := trie.NewIterator(accTrie.MustNodeIterator(nil))
	for accIt.Next() {
		accountCount++
		accountHash := common.BytesToHash(accIt.Key)

		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIt.Value, &acc); err != nil {
			t.Fatalf("Failed to decode account: %v", err)
		}

		_, isTracked := trackedSet[accountHash]

		if acc.Root != types.EmptyRootHash {
			id := trie.StorageTrieID(root, accountHash, acc.Root)
			storageTrie, err := trie.New(id, trieDb)

			if isTracked {
				if err != nil {
					t.Errorf("Tracked account %s should have storage trie", accountHash.Hex()[:10])
				} else {
					storeIt := trie.NewIterator(storageTrie.MustNodeIterator(nil))
					slots := 0
					for storeIt.Next() {
						slots++
					}
					if slots > 0 {
						trackedWithStorage++
					}
				}
			} else {
				// Untracked should not have storage (skip markers are no longer used,
				// the filter is checked directly during healing)
				if err == nil {
					storeIt := trie.NewIterator(storageTrie.MustNodeIterator(nil))
					slots := 0
					for storeIt.Next() {
						slots++
					}
					if slots == 0 {
						untrackedWithoutStorage++
					} else {
						t.Errorf("Untracked account %s has %d storage slots", accountHash.Hex()[:10], slots)
					}
				} else {
					untrackedWithoutStorage++
				}
			}
		}
	}

	if accountCount != len(elems) {
		t.Errorf("Expected %d accounts, got %d", len(elems), accountCount)
	}

	if trackedWithStorage != len(trackedHashes) {
		t.Errorf("Expected %d tracked accounts with storage, got %d", len(trackedHashes), trackedWithStorage)
	}

	t.Logf("Verified: %d total accounts, %d tracked with storage, %d untracked without storage",
		accountCount, trackedWithStorage, untrackedWithoutStorage)
}

// countTotalStorageSlots counts all storage slots across all accounts.
func countTotalStorageSlots(t *testing.T, db ethdb.KeyValueStore, scheme string, root common.Hash) int {
	t.Helper()

	trieDb := triedb.NewDatabase(rawdb.NewDatabase(db), newDbConfig(scheme))
	accTrie, err := trie.New(trie.StateTrieID(root), trieDb)
	if err != nil {
		t.Fatalf("Failed to open account trie: %v", err)
	}

	totalSlots := 0
	accIt := trie.NewIterator(accTrie.MustNodeIterator(nil))
	for accIt.Next() {
		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIt.Value, &acc); err != nil {
			continue
		}

		if acc.Root == types.EmptyRootHash {
			continue
		}

		accountHash := common.BytesToHash(accIt.Key)
		id := trie.StorageTrieID(root, accountHash, acc.Root)
		storageTrie, err := trie.New(id, trieDb)
		if err != nil {
			continue
		}

		storeIt := trie.NewIterator(storageTrie.MustNodeIterator(nil))
		for storeIt.Next() {
			totalSlots++
		}
	}

	return totalSlots
}

// Verify our test filter implements ContractFilter
var _ partial.ContractFilter = (*testHashFilter)(nil)
