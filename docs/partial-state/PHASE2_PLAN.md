# Phase 2: Snap Sync Modifications for Partial Statefulness

## Pre-Execution Tasks

Before implementing Phase 2, complete these preparatory tasks:

### Task 0.1: Commit Phase 1 Changes
Commit all existing Phase 1 work (configuration, filters, BAL infrastructure):
```bash
git add cmd/geth/chaincmd.go cmd/geth/main.go cmd/utils/flags.go \
        core/rawdb/schema.go core/rawdb/accessors_bal.go \
        eth/ethconfig/config.go eth/ethconfig/gen_config.go \
        core/state/partial/
git commit -m "eth: add partial statefulness foundation (Phase 1)

Implements EIP-7928 BAL-based partial statefulness infrastructure:

- Add PartialStateConfig to eth/ethconfig with CLI flags
- Add ContractFilter interface in core/state/partial/
- Add BAL history database accessors in core/rawdb/
- Add PartialState and BALHistory managers

This enables nodes to track only configured contracts' storage
while maintaining full account trie integrity."
```

### Task 0.2: Save Plan Documentation
Create a reference document in the repo (not to be committed):
```bash
mkdir -p docs/partial-state
# Copy plan content to docs/partial-state/PHASE2_PLAN.md
```

---

## Executive Summary

This plan modifies go-ethereum's snap sync to support **partial statefulness**: downloading ALL accounts but only storage/bytecode for **configured contracts**. This enables nodes to operate with ~30-40GB instead of ~1TB+ while maintaining full account trie integrity.

---

## Snap Sync Protocol Overview

Based on comprehensive analysis of 10 different aspects of the snap sync implementation:

### Current Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Syncer.Sync()                            │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ PHASE 1: Snap Download                                    │   │
│  │  1. assignAccountTasks() → Download account ranges        │   │
│  │  2. processAccountResponse() → Analyze each account:      │   │
│  │     • CodeHash != Empty → Add to codeTasks                │   │
│  │     • Root != Empty → Add to stateTasks                   │   │
│  │  3. assignBytecodeTasks() → Download bytecodes            │   │
│  │  4. assignStorageTasks() → Download storage slots         │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ PHASE 2: Healing                                          │   │
│  │  • Fill gaps in trie structure                            │   │
│  │  • Download missing intermediate nodes                    │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Key Decision Points for Filtering

| Location            | Function                   | Decision                                                 |
| ------------------- | -------------------------- | -------------------------------------------------------- |
| `sync.go:1908-1928` | `processAccountResponse()` | Checks `CodeHash != EmptyCodeHash` → adds to `codeTasks` |
| `sync.go:1930-1969` | `processAccountResponse()` | Checks `Root != EmptyRootHash` → adds to `stateTasks`    |
| `sync.go:1117-1215` | `assignBytecodeTasks()`    | Iterates `codeTasks` map                                 |
| `sync.go:1220-1373` | `assignStorageTasks()`     | Iterates `stateTasks` map                                |

### Key Data Structures

```go
type accountTask struct {
    needCode  []bool                         // Which accounts need bytecode
    needState []bool                         // Which accounts need storage
    needHeal  []bool                         // Which accounts need healing
    codeTasks map[common.Hash]struct{}       // Pending bytecode hashes
    stateTasks map[common.Hash]common.Hash   // Account hash → storage root
    stateCompleted map[common.Hash]struct{}  // Completed storage syncs
}
```

---

## Design: Minimal-Invasion Approach

Instead of creating a separate `PartialSyncer`, we'll add **filter checks at decision points** within the existing Syncer. This is less invasive and easier to maintain.

### Changes Overview

```
┌─────────────────────────────────────────────────────────────┐
│ eth/protocols/snap/sync.go                                   │
│  • Add filter field to Syncer struct                         │
│  • Modify processAccountResponse() to check filter           │
│  • Add skip markers for intentionally skipped storage        │
│  • Modify healing to skip intentionally-skipped accounts     │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ eth/protocols/snap/sync_partial.go (NEW)                     │
│  • PartialSyncConfig struct                                  │
│  • Skip marker database functions                            │
│  • Helper functions for filter integration                   │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ eth/downloader/downloader.go                                 │
│  • Pass PartialStateConfig to snap.Syncer                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Detailed Implementation Plan

### Task 2.1: Add Filter to Syncer Struct

**File:** `eth/protocols/snap/sync.go`

Add filter field to Syncer:
```go
type Syncer struct {
    // ... existing fields ...

    // Partial state filter (nil = sync everything)
    filter partial.ContractFilter
}
```

Modify `NewSyncer()`:
```go
func NewSyncer(db ethdb.KeyValueStore, scheme string, filter partial.ContractFilter) *Syncer {
    return &Syncer{
        db:     db,
        scheme: scheme,
        filter: filter,  // May be nil for full sync
        // ... rest unchanged
    }
}
```

**Estimated changes:** ~10 lines

---

### Task 2.2: Create sync_partial.go Helper File

**File:** `eth/protocols/snap/sync_partial.go` (NEW)

```go
package snap

import (
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/rawdb"
    "github.com/ethereum/go-ethereum/core/state/partial"
    "github.com/ethereum/go-ethereum/ethdb"
)

// Database key prefix for tracking intentionally skipped storage
var skippedStoragePrefix = []byte("SnapSkipped")

// skippedStorageKey returns the database key for a skipped storage marker
func skippedStorageKey(accountHash common.Hash) []byte {
    return append(skippedStoragePrefix, accountHash.Bytes()...)
}

// markStorageSkipped records that storage was intentionally skipped for an account
func markStorageSkipped(db ethdb.KeyValueWriter, accountHash common.Hash, storageRoot common.Hash) {
    db.Put(skippedStorageKey(accountHash), storageRoot.Bytes())
}

// isStorageSkipped checks if storage was intentionally skipped for an account
func isStorageSkipped(db ethdb.KeyValueReader, accountHash common.Hash) bool {
    has, _ := db.Has(skippedStorageKey(accountHash))
    return has
}

// deleteStorageSkipped removes the skip marker (used during cleanup)
func deleteStorageSkipped(db ethdb.KeyValueWriter, accountHash common.Hash) {
    db.Delete(skippedStorageKey(accountHash))
}

// shouldSyncStorage returns true if storage should be synced for this address
func (s *Syncer) shouldSyncStorage(addr common.Address) bool {
    if s.filter == nil {
        return true  // No filter = sync everything
    }
    return s.filter.ShouldSyncStorage(addr)
}

// shouldSyncCode returns true if bytecode should be synced for this address
func (s *Syncer) shouldSyncCode(addr common.Address) bool {
    if s.filter == nil {
        return true  // No filter = sync everything
    }
    return s.filter.ShouldSyncCode(addr)
}
```

**Estimated changes:** ~50 lines

---

### Task 2.3: Modify processAccountResponse() for Filtering

**File:** `eth/protocols/snap/sync.go`

**Current code (lines 1908-1969):**
```go
// Check if the account is a contract with an unknown code
if !bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes()) {
    if !rawdb.HasCodeWithPrefix(s.db, common.BytesToHash(account.CodeHash)) {
        res.task.codeTasks[common.BytesToHash(account.CodeHash)] = struct{}{}
        res.task.needCode[i] = true
        res.task.pend++
    }
}
// Check if the account is a contract with an unknown storage trie
if account.Root != types.EmptyRootHash {
    // ... adds to stateTasks
}
```

**Modified code:**
```go
// Derive address from account hash for filter check
// Note: We have the hash, need to track address mapping
addr := s.hashToAddress(res.hashes[i])  // New helper needed

// Check if the account is a contract with an unknown code
if !bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes()) {
    if !rawdb.HasCodeWithPrefix(s.db, common.BytesToHash(account.CodeHash)) {
        // NEW: Check filter before adding to codeTasks
        if s.shouldSyncCode(addr) {
            res.task.codeTasks[common.BytesToHash(account.CodeHash)] = struct{}{}
            res.task.needCode[i] = true
            res.task.pend++
        }
        // If filtered out, bytecode just won't be fetched
    }
}

// Check if the account is a contract with an unknown storage trie
if account.Root != types.EmptyRootHash {
    // NEW: Check filter before adding to stateTasks
    if s.shouldSyncStorage(addr) {
        // ... existing logic to add to stateTasks
    } else {
        // Mark as intentionally skipped for healing phase
        markStorageSkipped(s.db, res.hashes[i], account.Root)
        res.task.stateCompleted[res.hashes[i]] = struct{}{}
        // Don't increment pend - we're not waiting for this storage
    }
}
```

**Challenge:** We have account hashes but need addresses for filter checks.

**Solution:** The filter operates on addresses, but snap sync uses hashes. Two options:
1. Store hash→address mapping during sync (memory overhead)
2. Modify filter to work with hashes (requires pre-computing hashes of configured addresses)

**Recommended: Option 2** - Pre-compute hashes in filter:
```go
type ConfiguredFilter struct {
    contracts     map[common.Address]struct{}
    contractHashes map[common.Hash]struct{}  // Pre-computed: keccak256(address)
}

func (f *ConfiguredFilter) ShouldSyncStorageByHash(hash common.Hash) bool {
    _, ok := f.contractHashes[hash]
    return ok
}
```

**Estimated changes:** ~40 lines in sync.go, ~20 lines in filter.go

---

### Task 2.4: Modify Healing to Skip Storage for Non-Tracked Contracts

**Important Clarification:** We **NEVER skip accounts** - ALL accounts are always synced (this is the core value proposition). We only skip **storage and bytecode** for contracts not in the configured filter.

**File:** `eth/protocols/snap/sync.go`

In `onHealState()` callback (lines 3071-3092), add check for **storage leaves only**:
```go
func (s *Syncer) onHealState(paths [][]byte, value []byte) error {
    if len(paths) == 1 {
        // Account trie leaf - ALWAYS process (never skip accounts)
        var account types.StateAccount
        if err := rlp.DecodeBytes(value, &account); err != nil {
            return nil
        }
        blob := types.SlimAccountRLP(account)
        rawdb.WriteAccountSnapshot(s.stateWriter, common.BytesToHash(paths[0]), blob)
        s.accountHealed += 1
        // ... rest unchanged
    }
    if len(paths) == 2 {
        // Storage trie leaf
        accountHash := common.BytesToHash(paths[0])

        // NEW: Skip STORAGE healing for non-tracked contracts
        // (accounts themselves are always synced/healed)
        if isStorageSkipped(s.db, accountHash) {
            return nil  // Don't heal storage we intentionally skipped
        }

        // ... existing storage handling
        rawdb.WriteStorageSnapshot(s.stateWriter, accountHash, ...)
    }
    return nil
}
```

Also modify healing task creation to avoid requesting storage trie nodes for non-tracked contracts.

**Key principle:** Account healing always proceeds. Only storage trie node requests are filtered.

**Estimated changes:** ~30 lines

---

### Task 2.5: Update Downloader to Pass Filter

**File:** `eth/downloader/downloader.go`

Modify `New()` to accept and pass filter:
```go
func New(stateDb ethdb.Database, mode ethconfig.SyncMode, ...,
         partialConfig *ethconfig.PartialStateConfig) *Downloader {

    var filter partial.ContractFilter
    if partialConfig != nil && partialConfig.Enabled {
        filter = partial.NewConfiguredFilter(partialConfig.Contracts)
    }

    dl := &Downloader{
        // ... existing fields
        SnapSyncer: snap.NewSyncer(stateDb, chain.TrieDB().Scheme(), filter),
    }
    // ...
}
```

**File:** `eth/handler.go`

Pass config through handler:
```go
h.downloader = downloader.New(config.Database, config.Sync, h.eventMux,
                               h.chain, h.removePeer, h.enableSyncedFeatures,
                               &config.Eth.PartialState)
```

**Estimated changes:** ~20 lines

---

### Task 2.6: Add Hash-Based Filter Methods

**File:** `core/state/partial/filter.go`

Extend ConfiguredFilter:
```go
type ConfiguredFilter struct {
    contracts      map[common.Address]struct{}
    contractHashes map[common.Hash]struct{}  // NEW: Pre-computed hashes
}

func NewConfiguredFilter(addresses []common.Address) *ConfiguredFilter {
    m := make(map[common.Address]struct{}, len(addresses))
    h := make(map[common.Hash]struct{}, len(addresses))
    for _, addr := range addresses {
        m[addr] = struct{}{}
        h[crypto.Keccak256Hash(addr.Bytes())] = struct{}{}  // Pre-compute hash
    }
    return &ConfiguredFilter{contracts: m, contractHashes: h}
}

// NEW: Hash-based filter for snap sync (which works with hashes, not addresses)
func (f *ConfiguredFilter) ShouldSyncStorageByHash(hash common.Hash) bool {
    _, ok := f.contractHashes[hash]
    return ok
}

func (f *ConfiguredFilter) ShouldSyncCodeByHash(hash common.Hash) bool {
    _, ok := f.contractHashes[hash]
    return ok
}
```

Update ContractFilter interface:
```go
type ContractFilter interface {
    ShouldSyncStorage(address common.Address) bool
    ShouldSyncCode(address common.Address) bool
    IsTracked(address common.Address) bool

    // Hash-based methods for snap sync
    ShouldSyncStorageByHash(hash common.Hash) bool
    ShouldSyncCodeByHash(hash common.Hash) bool
}
```

**Estimated changes:** ~30 lines

---

### Task 2.7: Persist Skip Markers for Resumption

**File:** `eth/protocols/snap/sync.go`

In `saveSyncStatus()`, ensure skip markers are preserved (they're already in DB, just verify):
```go
func (s *Syncer) saveSyncStatus() {
    // ... existing serialization

    // Skip markers are already in DB (written during processAccountResponse)
    // They persist across restarts automatically
}
```

In `loadSyncStatus()`, log skipped storage count for visibility:
```go
func (s *Syncer) loadSyncStatus() {
    // ... existing deserialization

    if s.filter != nil {
        log.Info("Partial state sync active",
            "trackedContracts", len(s.filter.Contracts()))
    }
}
```

**Estimated changes:** ~10 lines

---

### Task 2.8: Add Metrics for Partial Sync

**File:** `eth/protocols/snap/sync.go`

Add counters:
```go
var (
    storageSkippedGauge = metrics.NewRegisteredGauge("snap/sync/storage/skipped", nil)
    bytecodeSkippedGauge = metrics.NewRegisteredGauge("snap/sync/bytecode/skipped", nil)
)
```

Increment in processAccountResponse:
```go
if !s.shouldSyncStorage(addr) {
    storageSkippedGauge.Inc(1)
    // ...
}
```

**Estimated changes:** ~15 lines

---

### Task 2.9: Unit Tests

**File:** `eth/protocols/snap/sync_partial_test.go` (NEW)

```go
package snap

import (
    "testing"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/state/partial"
)

func TestPartialSyncFilterStorage(t *testing.T) {
    // Create filter with specific contracts
    tracked := []common.Address{
        common.HexToAddress("0x1234..."),
    }
    filter := partial.NewConfiguredFilter(tracked)

    // Verify tracked contracts pass filter
    if !filter.ShouldSyncStorage(tracked[0]) {
        t.Error("Tracked contract should pass filter")
    }

    // Verify untracked contracts are filtered
    untracked := common.HexToAddress("0xABCD...")
    if filter.ShouldSyncStorage(untracked) {
        t.Error("Untracked contract should be filtered")
    }

    // Verify hash-based filter works
    trackedHash := crypto.Keccak256Hash(tracked[0].Bytes())
    if !filter.ShouldSyncStorageByHash(trackedHash) {
        t.Error("Tracked contract hash should pass filter")
    }
}

func TestSkipMarkerPersistence(t *testing.T) {
    db := rawdb.NewMemoryDatabase()
    accountHash := common.HexToHash("0x1234...")
    storageRoot := common.HexToHash("0xABCD...")

    // Mark as skipped
    markStorageSkipped(db, accountHash, storageRoot)

    // Verify marker persists
    if !isStorageSkipped(db, accountHash) {
        t.Error("Skip marker should persist")
    }

    // Delete and verify
    deleteStorageSkipped(db, accountHash)
    if isStorageSkipped(db, accountHash) {
        t.Error("Skip marker should be deleted")
    }
}
```

**Estimated changes:** ~100 lines

---

### Task 2.10: Integration Test

**File:** `eth/protocols/snap/sync_partial_integration_test.go` (NEW)

Create end-to-end test that:
1. Sets up a mock state with multiple contracts
2. Configures partial sync with subset of contracts
3. Runs sync
4. Verifies:
   - All accounts synced
   - Only configured contracts have storage
   - Skip markers present for non-configured contracts
   - Healing doesn't try to heal skipped storage

**Estimated changes:** ~200 lines

---

## Local Testing Strategy

### 1. Unit Test Execution
```bash
cd eth/protocols/snap
go test -v -run TestPartialSync
go test -v -run TestSkipMarker
```

### 2. Build Verification
```bash
go build ./...
go build ./cmd/geth
```

### 3. Simulated Network Test

Create a test script that:
```bash
# Terminal 1: Start full node (serves as peer)
./geth --datadir /tmp/full-node --syncmode snap --port 30303

# Terminal 2: Start partial node
./geth --datadir /tmp/partial-node --syncmode snap --port 30304 \
    --partial-state \
    --partial-state.contracts 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2 \
    --bootnodes "enode://..."
```

### 4. Verification Checks

After sync completes:
```bash
# Check database size (should be significantly smaller)
du -sh /tmp/partial-node/geth/chaindata

# Query RPC to verify:
# - Account balance works for any address
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x...", "latest"],"id":1}' \
    http://localhost:8545

# - Storage works for tracked contracts
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0x0", "latest"],"id":1}' \
    http://localhost:8545

# - Storage fails for untracked contracts (once RPC phase implemented)
```

### 5. Devnet Testing

For full integration testing:
1. Use a local devnet with known state
2. Configure partial sync with specific test contracts
3. Verify sync completion and state correctness
4. Test reorg handling with BAL history

---

## Files to Modify Summary

| File                                                  | Changes                                                  | Lines |
| ----------------------------------------------------- | -------------------------------------------------------- | ----- |
| `eth/protocols/snap/sync.go`                          | Add filter field, modify processAccountResponse, healing | ~80   |
| `eth/protocols/snap/sync_partial.go`                  | NEW: Skip markers, helpers                               | ~50   |
| `core/state/partial/filter.go`                        | Add hash-based filter methods                            | ~30   |
| `eth/downloader/downloader.go`                        | Pass filter to Syncer                                    | ~15   |
| `eth/handler.go`                                      | Pass config through                                      | ~5    |
| `eth/protocols/snap/sync_partial_test.go`             | NEW: Unit tests                                          | ~100  |
| `eth/protocols/snap/sync_partial_integration_test.go` | NEW: Integration tests                                   | ~200  |

**Total estimated changes:** ~480 lines

---

## Task Summary

| Task ID | Description                      | Dependencies  | Effort |
| ------- | -------------------------------- | ------------- | ------ |
| 2.1     | Add filter to Syncer struct      | None          | S      |
| 2.2     | Create sync_partial.go helpers   | 2.1           | S      |
| 2.3     | Modify processAccountResponse    | 2.1, 2.2, 2.6 | M      |
| 2.4     | Modify healing to skip filtered  | 2.2           | S      |
| 2.5     | Update Downloader to pass filter | 2.1           | S      |
| 2.6     | Add hash-based filter methods    | None          | S      |
| 2.7     | Persist skip markers             | 2.2           | S      |
| 2.8     | Add metrics                      | 2.3           | S      |
| 2.9     | Unit tests                       | 2.1-2.8       | M      |
| 2.10    | Integration test                 | 2.9           | L      |

**Effort:** S = Small (few hours), M = Medium (1-2 days), L = Large (3-5 days)

---

## Critical Design Decisions

1. **Minimal Invasion**: Modify existing Syncer rather than creating parallel implementation
2. **Hash-Based Filtering**: Pre-compute address hashes for efficient snap sync filtering
3. **Skip Markers in DB**: Persist skip decisions for healing phase and resumption
4. **Healing Awareness**: Healing phase must respect skip markers to avoid errors

---

## Verification Checklist

After implementation:
- [ ] **ALL accounts synced** (complete account trie - this is critical)
- [ ] **Account trie fully intact** - eth_getProof works for ANY account
- [ ] Only configured contracts have storage synced
- [ ] Only configured contracts have bytecode synced
- [ ] Skip markers present for non-tracked contract storage
- [ ] Healing completes successfully (doesn't try to heal skipped storage)
- [ ] Sync can resume after restart (skip markers persist)
- [ ] Database size significantly reduced (~30-40GB vs ~1TB+)
- [ ] eth_getBalance works for ANY account (not just tracked)
- [ ] eth_getTransactionCount works for ANY account
- [ ] eth_getStorageAt works for tracked contracts
- [ ] Metrics show skipped storage/bytecode counts

**Critical Invariant:** The account trie must be 100% complete. Only storage tries for non-tracked contracts are skipped.


# After implemenmtation:
★ Insight ─────────────────────────────────────
  Phase 2 Implementation Complete:
  The key architectural change is that snap sync now supports partial statefulness through a filter mechanism. The filter is passed through
  the entire stack:
  1. ethconfig.PartialStateConfig → CLI configuration
  2. partial.ConfiguredFilter → Pre-computes keccak256 hashes for efficient lookup
  3. snap.Syncer.filter → Makes filtering decisions during sync
  4. markStorageSkipped() / isStorageSkipped() → Persists skip decisions for healing phase

  This allows a node to sync ~30-40GB (all accounts) instead of ~1TB+ (all accounts + all storage).
  ─────────────────────────────────────────────────

⏺ Bash(git diff --stat HEAD 2>&1 | tail -20)
  ⎿   core/state/partial/filter.go      | 46 +++++++++++++++++++++++++--
      eth/backend.go                    | 10 ++++++
      eth/downloader/downloader.go      |  5 +--
     … +5 lines (ctrl+o to expand)

⏺ Bash(git status --short 2>&1)
  ⎿   M core/state/partial/filter.go
      M eth/backend.go
      M eth/downloader/downloader.go
     … +9 lines (ctrl+o to expand)

⏺ Here's a summary of the Phase 2 Snap Sync Modifications that have been implemented:

  Phase 2 Implementation Summary

  Tasks Completed
  ┌──────┬───────────────────────────────────────────────┬───────────────────────────────────┐
  │ Task │                  Description                  │              Status               │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.1  │ Add filter to Syncer struct                   │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.2  │ Create sync_partial.go helpers                │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.3  │ Modify processAccountResponse() for filtering │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.4  │ Modify healing to skip storage                │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.5  │ Update Downloader to pass filter              │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.6  │ Add hash-based filter methods                 │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.7  │ Persist skip markers                          │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.8  │ Add metrics                                   │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.9  │ Unit tests                                    │ ✓                                 │
  ├──────┼───────────────────────────────────────────────┼───────────────────────────────────┤
  │ 2.10 │ Integration test                              │ Deferred (requires network setup) │
  └──────┴───────────────────────────────────────────────┴───────────────────────────────────┘
  Files Modified (7 files, +112 lines)

  - core/state/partial/filter.go - Added hash-based filter methods
  - eth/backend.go - Create and pass partial filter
  - eth/downloader/downloader.go - Accept filter parameter
  - eth/handler.go - Add PartialFilter to handlerConfig
  - eth/protocols/snap/sync.go - Filter storage/bytecode in processAccountResponse and healing

  Files Created (4 files)

  - eth/protocols/snap/sync_partial.go - Skip marker DB functions and filter helpers
  - eth/protocols/snap/sync_partial_test.go - Unit tests for partial sync
  - core/state/partial/filter_test.go - Unit tests for filter
  - docs/partial-state/PHASE2_PLAN.md - Plan documentation

  Key Design Decisions

  1. Minimal invasion: Modified existing Syncer rather than creating parallel implementation
  2. Hash-based filtering: Pre-compute keccak256 hashes for O(1) lookup during sync
  3. Skip markers: Persist decisions in DB so healing phase knows what was intentionally skipped
  4. All accounts synced: Only storage/bytecode are filtered, account trie is always complete

  Testing

  All 8 new tests pass:
  - TestPartialSyncFilterStorage
  - TestAllowAllFilter
  - TestSkipMarkerPersistence
  - TestSyncerFilterMethods
  - TestConfiguredFilterContracts
  - TestConfiguredFilterBasic
  - TestConfiguredFilterHashConsistency
  - TestAllowAllFilterInterface