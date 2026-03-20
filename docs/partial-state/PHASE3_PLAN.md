# Phase 3: BAL Processing & State Updates for Partial Statefulness

## Overview

**Goal**: Enable partial state nodes to process blocks using Block Access Lists (BALs) instead of re-executing transactions. This allows state updates without needing full contract storage.

**Key principle**: BALs (per EIP-7928) provide state diffs that allow computing the new state root by applying changes directly to the trie, without transaction execution.

---

## Prerequisites

- Phase 1 (Configuration & Infrastructure): ✓ Complete
- Phase 2 (Snap Sync Modifications): ✓ Complete
- EIP-7928 BAL types already exist in `core/types/bal/`

---

## Design Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Block Processing Flow                         │
│                                                                   │
│  Full Node:     Block → Execute TXs → Compute State Root         │
│                                                                   │
│  Partial Node:  Block + BAL → Apply BAL Diffs → Verify Root      │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    BAL Application Flow                          │
│                                                                   │
│  1. Receive block + BAL (via Engine API)                         │
│  2. Verify: keccak256(rlp(BAL)) == header.BlockAccessListHash    │
│  3. For each AccountAccess in BAL:                               │
│     a. Load account from trie                                    │
│     b. Apply balance/nonce changes (final values)                │
│     c. Apply storage changes (tracked contracts only)            │
│     d. Update account in trie                                    │
│  4. Commit trie → Verify root matches header.stateRoot           │
│  5. Store BAL for reorg handling                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Existing Infrastructure (ALREADY EXISTS - REUSE!)

Based on agent exploration, the following infrastructure **already exists and is production-ready**:

| Component | Location | Status |
|-----------|----------|--------|
| BAL Types | `core/types/bal/bal.go` | ✅ Complete - `ConstructionBlockAccessList`, `BlockAccessList` |
| BAL Encoding | `core/types/bal/bal_encoding.go` | ✅ Complete - RLP, Hash(), Validate() |
| DB Schema | `core/rawdb/schema.go:172` | ✅ Complete - prefix `"p"` |
| DB Accessors | `core/rawdb/accessors_bal.go` | ✅ Complete - Read/Write/Delete/Prune |
| BALHistory | `core/state/partial/history.go` | ✅ Complete - wrapper over rawdb |
| PartialState | `core/state/partial/state.go` | ⚠️ Skeleton - needs `ApplyBALAndComputeRoot()` |
| ContractFilter | `core/state/partial/filter.go` | ✅ Complete - ConfiguredFilter, AllowAllFilter |
| Trie Interface | `trie/trie.go` | ✅ Standard trie operations |

**What this means:** Tasks 3.2, 3.3, 3.4 are already done! We only need to implement:
- `ApplyBALAndComputeRoot()` in PartialState
- `ProcessBlockWithBAL()` in BlockChain
- Reorg handling
- Tests

---

## Detailed Implementation Plan

### Task 3.1: Review/Extend Existing PartialState Struct

**File:** `core/state/partial/state.go` (ALREADY EXISTS!)

**Agent Finding:** PartialState skeleton already exists with correct structure:
```go
type PartialState struct {
    db        ethdb.Database
    trieDB    *triedb.Database
    filter    ContractFilter
    history   *BALHistory      // Already includes history!
    stateRoot common.Hash
}
```

**Current methods (already implemented):**
- `NewPartialState()` - Constructor ✅
- `Filter()` - Filter access ✅
- `Root()` / `SetRoot()` - Root management ✅
- `History()` - BAL history access ✅

**Key patterns from StateDB (confirmed by agent):**
- PartialState does NOT need `stateObjects` caching (applies BAL directly to trie)
- PartialState does NOT need journal/revert (BAL diffs are immutable)
- PartialState does NOT need prefetcher (not executing contracts)
- Error handling: return errors immediately (no memoization)

**What needs to be added:**
- `ApplyBALAndComputeRoot()` method (Task 3.5)
- Optional metrics fields for monitoring

**Estimated changes:** ~10 lines (mostly just adding ApplyBALAndComputeRoot)

---

### Task 3.2: ✅ ALREADY EXISTS - BAL History Database Schema

**File:** `core/rawdb/schema.go` line 172

**Agent confirmed:** Schema already exists!
```go
balHistoryPrefix = []byte("p") // balHistoryPrefix + num (uint64 big endian) -> RLP(bal.BlockAccessList)
```

**Key format:** `"p" + blockNumber(8 bytes, big-endian)` → RLP-encoded BlockAccessList

**Estimated changes:** 0 lines (already exists)

---

### Task 3.3: ✅ ALREADY EXISTS - BAL History Accessors

**File:** `core/rawdb/accessors_bal.go`

**Agent confirmed:** All accessors already implemented!
- `ReadBALHistory(db, blockNum)` ✅
- `WriteBALHistory(db, blockNum, accessList)` ✅
- `DeleteBALHistory(db, blockNum)` ✅
- `HasBALHistory(db, blockNum)` ✅
- `PruneBALHistory(db, beforeBlock)` ✅ (with safe range iteration)

**Estimated changes:** 0 lines (already exists)

---

### Task 3.4: ✅ ALREADY EXISTS - BALHistory Wrapper

**File:** `core/state/partial/history.go`

**Agent confirmed:** BALHistory wrapper already implemented!
```go
type BALHistory struct {
    db        ethdb.Database
    retention uint64
}

// Methods: Store(), Get(), Delete(), Prune(), Retention()
```

**Design note:** We have BOTH:
1. BALHistory in `partial/history.go` - for explicit BAL storage/retrieval
2. Blocks contain BALs - can also access via block

For reorgs, we'll use BALHistory since it's already built and tested.

**Estimated changes:** 0 lines (already exists)

---

### Task 3.5: Implement ApplyBALAndComputeRoot

**File:** `core/state/partial/state.go` (extend)

**Key implementation requirements (from code review and agent research):**

1. **BAL field names**: Use `Accesses` (not `Writes`) and `ValueAfter` (not `Value`) per `core/types/bal/bal_encoding.go`
2. **Commit ordering**: Storage tries → update account.Root → account trie (critical for correct state root)
3. **Account origin tracking**: Track `existed` flag to prevent incorrect EIP-161 deletion
4. **Code handling**: Update CodeHash for ALL accounts, store code bytes only for tracked contracts
5. **PathDB StateSet**: Must construct proper `triedb.StateSet` for `trieDB.Update()` call

**PathDB StateSet construction (from agent research on `core/state/statedb.go`):**

The `trieDB.Update()` signature is:
```go
func (db *Database) Update(root, parent common.Hash, block uint64, nodes *trienode.MergedNodeSet, states *StateSet) error
```

The `StateSet` structure requires:
```go
type StateSet struct {
    Accounts       map[common.Hash][]byte                    // Mutated accounts in 'slim RLP' encoding
    AccountsOrigin map[common.Address][]byte                 // Original account values (for PathDB)
    Storages       map[common.Hash]map[common.Hash][]byte    // Storage: accountHash → slotHash → value
    StoragesOrigin map[common.Address]map[common.Hash][]byte // Original storage values
    RawStorageKey  bool                                      // false = use hashed keys
}
```

**Key encoding requirements:**
- Accounts: Use `types.SlimAccountRLP(account)` for encoding
- Storage values: Use prefix-zero-trimmed RLP (`rlp.EncodeToBytes(common.TrimLeftZeroes(val[:]))`)
- Storage keys: Must be hashed (`crypto.Keccak256Hash(rawKey[:])`)
- Nil values indicate deletion

**Estimated changes:** ~250 lines (includes PathDB StateSet construction)

---

### Task 3.6: Implement ProcessBlockWithBAL

**File:** `core/blockchain_partial.go` (NEW)

**Trust Model:** Blocks via Engine API are pre-attested by the Consensus Layer. The function documents this trust model clearly in its comments, explaining why no additional attestation verification is needed (same as full nodes).

**Estimated changes:** ~100 lines

---

### Task 3.7: Implement Reorg Handling

**File:** `core/blockchain_partial.go` (extend)

**DESIGN:** Reorg handling accesses blocks directly (which contain BALs), NOT a separate BALHistory. This mirrors how full nodes handle reorgs.

**Key differences from full node reorg:**
- Full node: re-executes transactions on new chain
- Partial node: applies BALs from new chain blocks

**Estimated changes:** ~50 lines

---

### Task 3.8: Wire PartialState into BlockChain

**File:** `core/blockchain.go` (modify)

**Agent findings on BlockChain state patterns:**

**Existing state fields (lines 311-366):**
```go
type BlockChain struct {
    db            ethdb.Database           // Low-level persistent database
    snaps         *snapshot.Tree           // Snapshot tree for fast trie leaf access
    triedb        *triedb.Database         // TrieDB handler for maintaining trie nodes
    statedb       *state.CachingDB         // State database (reused between imports)
    // ... caches, processor, validator, etc.
}
```

**Add partialState alongside existing fields:**
```go
type BlockChain struct {
    // ... existing fields ...

    // Partial state management (nil if full node)
    partialState  *partial.PartialState
}
```

**Estimated changes:** ~40 lines

---

### Task 3.9: Add Unit Tests

**File:** `core/state/partial/state_test.go` (NEW)

```go
func TestApplyBALAndComputeRoot(t *testing.T) {
    // Test that BAL application produces correct state root
}

func TestApplyStorageChanges(t *testing.T) {
    // Test storage updates for tracked contracts
}

func TestApplyBalanceChanges(t *testing.T) {
    // Test balance updates from BAL
}

func TestFilteredStorageChanges(t *testing.T) {
    // Test that untracked contract storage is not applied
}
```

**Estimated changes:** ~100 lines

---

### Task 3.10: Integration Test

**File:** `core/blockchain_partial_test.go` (NEW)

Test end-to-end BAL processing:
1. Create a chain with known state
2. Generate BALs for blocks
3. Process blocks with `ProcessBlockWithBAL`
4. Verify state roots match
5. Test reorg handling

**Estimated changes:** ~200 lines

---

## Files to Modify/Create Summary

| File | Status | Changes |
|------|--------|---------|
| `docs/partial-state/PARTIAL_STATEFULNESS_PLAN.md` | NEW | Copy master plan from `.claude/plans/` |
| `docs/partial-state/PHASE3_PLAN.md` | NEW | Copy this Phase 3 plan |
| `core/state/partial/state.go` | EXTEND | Add `ApplyBALAndComputeRoot()` + StateSet (~250 lines) |
| `core/rawdb/schema.go` | ✅ EXISTS | `balHistoryPrefix` already defined |
| `core/rawdb/accessors_bal.go` | ✅ EXISTS | All accessors already implemented |
| `core/state/partial/history.go` | ✅ EXISTS | `BALHistory` wrapper already implemented |
| `core/blockchain.go` | MODIFY | Add `partialState` field, initialization (~40 lines) |
| `core/blockchain_partial.go` | NEW | `ProcessBlockWithBAL`, reorg, attestation (~150 lines) |
| `core/state/partial/state_test.go` | NEW | Unit tests (~100 lines) |
| `core/blockchain_partial_test.go` | NEW | Integration tests (~200 lines) |

**Total estimated new code:** ~710 lines
**Infrastructure already exists:** ~300 lines (schema, accessors, history)

---

## Task Summary

| Task ID | Description | Dependencies | Effort | Status |
|---------|-------------|--------------|--------|--------|
| 1 | Save master plan + Phase 3 plan to docs/partial-state/ | None | S | TODO |
| 3.1 | Review existing PartialState, add metrics | Phase 1 | S | Exists |
| 3.2 | BAL history DB schema | None | - | ✅ EXISTS |
| 3.3 | BAL history accessors | 3.2 | - | ✅ EXISTS |
| 3.4 | BALHistory wrapper | 3.3 | - | ✅ EXISTS |
| 3.5 | Implement `ApplyBALAndComputeRoot` with PathDB StateSet | 3.1 | L | TODO |
| 3.6 | Implement `ProcessBlockWithBAL` with trust model docs | 3.5 | M | TODO |
| 3.7 | Implement reorg handling (uses BALHistory) | 3.6 | M | TODO |
| 3.8 | Wire into BlockChain | 3.6 | S | TODO |
| 3.9 | Unit tests | 3.5, 3.7 | M | TODO |
| 3.10 | Integration test | 3.6, 3.7 | L | TODO |

**Effort:** S = Small (few hours), M = Medium (1-2 days), L = Large (3-5 days)

**Good news:** Tasks 3.2, 3.3, 3.4 are already implemented! Only need to implement 3.5-3.10.

---

## Dependency Graph

```
Task 1 (Save master plan + Phase 3 plan)
  ↓
3.1 (Review existing PartialState) ─── 3.2/3.3/3.4 ✅ ALREADY EXIST
  ↓
3.5 (ApplyBALAndComputeRoot with PathDB StateSet)
  ↓
3.6 (ProcessBlockWithBAL with trust model docs)
  ↓
3.7 (Reorg handling via BALHistory)
  ↓
3.8 (Wire into BlockChain)
  ↓
3.9 (Unit tests)
  ↓
3.10 (Integration test)
```

---

## Verification Checklist

**Pre-implementation (completed):**
- [x] Code review completed for ApplyBALAndComputeRoot design
- [x] BAL field names verified: `Accesses`, `ValueAfter` (from `core/types/bal/bal_encoding.go`)
- [x] Commit ordering documented: storage tries before account trie
- [x] PathDB StateSet construction researched and documented
- [x] SELFDESTRUCT handling verified: tracked in BAL per EIP-7928
- [x] Engine API delivery researched: standardized via engine_newPayloadV5, etc.

**After implementation:**
- [ ] Master plan saved to `docs/partial-state/PARTIAL_STATEFULNESS_PLAN.md`
- [ ] Phase 3 plan saved to `docs/partial-state/PHASE3_PLAN.md`
- [ ] PartialState struct follows StateDB patterns
- [ ] BAL hash verification works correctly
- [ ] Balance/nonce/codeHash changes apply correctly for ALL accounts
- [ ] Storage/code bytes stored only for tracked contracts
- [ ] Commit ordering correct: storage trie commit → update account.Root → account trie commit
- [ ] EIP-161 empty account deletion only for modified+empty+existed accounts
- [ ] PathDB StateSet properly constructed with origins
- [ ] Computed state root matches header
- [ ] Reorg handling works (via blocks, not separate BALHistory)
- [ ] All unit tests pass
- [ ] Integration test passes

---

## Local Testing Strategy

### 1. Unit Test Execution
```bash
go test ./core/state/partial/... -v
go test ./core/rawdb/... -run TestBAL -v
```

### 2. Build Verification
```bash
go build ./...
go build ./cmd/geth
```

### 3. Integration Test
```bash
go test ./core/... -run TestPartialBlock -v -timeout 5m
```

---

## Open Items

1. **Engine API Integration**: BAL delivery is **already standardized** via extended Engine API methods:
   - `engine_newPayloadV5`: Validates computed access lists match provided BAL
   - `engine_getPayloadV6`: Returns `ExecutionPayloadV4` containing RLP-encoded BAL
   - `engine_getPayloadBodiesByHashV2` / `engine_getPayloadBodiesByRangeV2`: Retrieve historical BALs
   - **Status**: No additional design needed - use existing Engine API

---

## Critical Invariants

1. **State root must match**: Computed root from BAL application MUST match header's stateRoot
2. **BAL hash verification**: Always verify BAL hash before processing
3. **Account trie complete**: All account changes apply (balance, nonce, codeHash); only storage/code bytes are filtered for untracked
4. **No execution required**: Block processing uses only BAL data, never re-executes transactions
5. **Commit ordering**: Storage tries MUST be committed BEFORE account trie (storage roots needed first)
6. **EIP-161 compliance**: Only delete accounts that were modified AND are now empty AND previously existed
7. **BAL field names**: Use `Accesses` (not `Writes`) and `ValueAfter` (not `Value`) per `core/types/bal/bal_encoding.go`
8. **PathDB StateSet**: Must construct proper `triedb.StateSet` with accounts/storage and their origins for `trieDB.Update()`

## Design Decisions

1. **SELFDESTRUCT is tracked**: Per EIP-7928, "Accounts destroyed within a transaction MUST be included in AccountChanges without nonce or code changes." Self-destructed accounts appear in BAL with balance changes but no nonce/code changes.

2. **Code handling for tracked vs untracked contracts**:
   - **All accounts**: Update `CodeHash` in account trie (required for correct state root)
   - **Tracked contracts only**: Store actual code bytes via `rawdb.WriteCode()`
   - **Untracked contracts**: Skip storing code bytes (saves storage, code not needed for partial state)

3. **Block attestation trust model** (Post-Merge architecture):
   - **CL responsibility**: Proposer signatures, sync committee attestations (2/3+ threshold), finality proofs, consensus rules
   - **EL responsibility**: Transaction execution, state root computation, receipt validation
   - **Trust boundary**: Blocks via Engine API (`engine_newPayloadV5`) are pre-attested by CL; EL trusts CL for consensus
   - **Partial state nodes**: Receive blocks via Engine API, so attestations are already verified
   - **Light client sync** (future): If blocks come from untrusted sources, use `beacon/light/CommitteeChain.VerifySignedHeader()`
