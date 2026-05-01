# Partial Statefulness Design - Final Plan

## Overview

**Goal**: Enable Ethereum nodes to operate with reduced storage by keeping:
- Full account trie (all accounts + intermediate nodes)
- Selective storage (only configured contracts' storage)
- BAL-based state updates (per EIP-7928)

**Source**: [ethresear.ch - Partial Statefulness](https://ethresear.ch/t/the-future-of-state-part-2-beyond-the-myth-of-partial-statefulness-the-reality-of-zkevms/23396)

---

## Design Decisions (Confirmed)

### Core Model
| Decision | Choice | Notes |
|----------|--------|-------|
| Account trie | ALL accounts + ALL intermediate nodes | Full trie structure with compression |
| Storage | Only configured contracts | User specifies which contracts in config file |
| BAL source | Per EIP-7928 | BALs come with blocks, hash committed in header |
| Validation | Trust BAL, apply diffs | Same trust model as light clients (signing committee) |
| Block history | 256-1024 blocks | Support BLOCKHASH opcode, configurable BAL retention |

### Storage Approach
| Component | Size | Notes |
|-----------|------|-------|
| Account leaves | ~14 GB | 300M accounts × ~45 bytes (slim RLP) |
| Intermediate nodes | ~15-25 GB | With delta encoding + bitmap compression |
| **Total account trie** | **~30-40 GB** | |
| Configured storage | Variable | Depends on tracked contracts |
| BAL history | ~1-2 GB | 256-1024 blocks |

### Operations
| Operation | Approach |
|-----------|----------|
| Initial sync | Account trie first (snap sync), then configured storage |
| Block processing | Apply BAL diffs → update trie → verify state root matches header |
| Reorgs | Revert using stored BAL history; deeper reorgs request from full peers |
| eth_getProof (accounts) | Supported for ALL accounts |
| eth_getProof (storage) | Only for configured contracts; error otherwise |
| Mempool validation | Fully supported (only needs account data) |
| Serving peers | Account proofs + tracked contract storage |

---

## EIP-7928 BAL Integration

### BAL Format (from EIP-7928)
```
BlockAccessList = [AccountAccess, ...]

AccountAccess = [
  Address,
  StorageWrites,    // map[slot] -> map[txIdx] -> value
  StorageReads,     // list of read slots
  BalanceChanges,   // map[txIdx] -> balance
  NonceChanges,     // map[txIdx] -> nonce
  CodeChanges       // map[txIdx] -> bytecode
]
```

### Key EIP-7928 Facts
- **Header commitment**: `block_access_list_hash = keccak256(rlp.encode(bal))`
- **Propagation**: Via Engine API (ExecutionPayloadV4), not in block body
- **Retention**: Full nodes must keep WSP (~5 months); partial nodes: configurable (256-1024 blocks)
- **Validation**: Deterministic - wrong BAL = wrong header hash = invalid block

### BAL Processing Flow
```
1. Receive block + BAL via Engine API
2. Verify: keccak256(rlp.encode(bal)) == header.block_access_list_hash
3. For each AccountAccess in BAL:
   a. Load current account from trie
   b. Apply balance/nonce changes (final values per block)
   c. Apply storage root update (from BAL storage writes for tracked contracts)
   d. Update account in trie
4. Commit trie changes
5. Verify: trie.Root() == header.stateRoot
6. If mismatch: reject block (consensus failure elsewhere)
```

---

## State Root Verification

### How It Works Without Re-execution

Partial nodes can verify state root because:

1. **Full account trie stored**: All intermediate nodes available
2. **BAL provides final values**: Post-block account state (not deltas)
3. **Trie update is deterministic**: Same inputs → same output
4. **Cross-check with header**: header.stateRoot must match computed root

### Trust Model

Same as beacon chain light clients:
- Trust signing committee (attestations)
- Verify header commitments (state root, BAL hash)
- Detect inconsistencies via hash mismatches

If BAL is incorrect:
- State root won't match → block rejected
- Fork choice rejects the block
- Partial node follows canonical chain

---

## Snap Sync Adaptation

### Current Snap Sync (Full Node)
```
Phase 1: Sync account ranges (GetAccountRangeMsg)
Phase 2: Sync all storage for all contracts
Phase 3: Sync all bytecode
Phase 4: Healing (fill gaps)
```

### Partial Statefulness Snap Sync
```
Phase 1: Sync COMPLETE account trie (same as full node)
  - All accounts
  - All intermediate nodes
  - ~30-40 GB

Phase 2: Sync storage ONLY for configured contracts
  - Filter: Only request storage for contracts in config
  - Skip: All other contracts' storage

Phase 3: Sync bytecode ONLY for configured contracts
  - Same filtering as storage

Phase 4: Healing (account trie only)
  - No healing needed for skipped storage
```

### Implementation Changes Needed
1. Add `PartialStateConfig` to ethconfig
2. Modify `storageRequest` creation in snap syncer to check config
3. Skip storage/bytecode tasks for non-configured contracts
4. Track sync progress separately for account trie vs. storage

---

## Configuration

### Config Structure
```go
type PartialStateConfig struct {
    Enabled          bool
    Contracts        []common.Address  // Tracked contracts
    ContractsFile    string            // Or load from JSON file
    BALRetention     uint64            // Blocks to keep (default: 256)
}
```

### Example Config (TOML)
```toml
[Eth.PartialState]
Enabled = true
BALRetention = 256
Contracts = [
    "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",  # WETH
    "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",  # USDC
]
```

---

## RPC Behavior

| Method | Behavior |
|--------|----------|
| `eth_getBalance` | ✅ Works (have account data) |
| `eth_getTransactionCount` | ✅ Works (have nonce) |
| `eth_getCode` | ✅ For tracked contracts; ❌ error for others |
| `eth_getStorageAt` | ✅ For tracked contracts; ❌ error for others |
| `eth_getProof` (account) | ✅ Works for ANY account |
| `eth_getProof` (storage) | ✅ For tracked contracts; ❌ error for others |
| `eth_call` | ✅ If touches only tracked contracts; ❌ if touches untracked |
| `eth_estimateGas` | Same as eth_call |
| `eth_sendRawTransaction` | ✅ Mempool validation works (only needs account data) |

---

## Binary Trie (EIP-7864) Compatibility

### Will This Design Work With Binary Trie?

**Yes**, with minimal changes:

| Aspect | MPT | Binary Trie | Compatibility |
|--------|-----|-------------|---------------|
| Account data | StateAccount struct | Same struct | ✅ Compatible |
| Trie interface | `Trie` interface | Same interface | ✅ Compatible |
| BAL format | Per EIP-7928 | Same format | ✅ Compatible |
| Selective storage | Skip storage tries | Skip stem suffixes | ✅ Compatible |
| Proof generation | Merkle proofs | Path proofs | ✅ Use interface |

### Adaptation Needed
Only the storage size estimates change:
- Binary Trie total: ~48 GB (vs. MPT ~30-40 GB with compression)
- Binary Trie has simpler structure, no compression needed

**Recommendation**: Use go-ethereum's `Trie` interface which abstracts over both.

---

## Implementation Phases

### Phase 1: Configuration & Infrastructure
- Add `PartialStateConfig` to `eth/ethconfig/config.go`
- Create `core/state/partial/` package with `ContractFilter` interface
- Add CLI flags for partial state mode

### Phase 2: Snap Sync Modifications
- Modify `eth/protocols/snap/sync.go` for selective storage sync
- Add filter checks in `processAccountResponse` and `processStorageResponse`
- Track separate progress for account trie vs. storage

### Phase 3: BAL Processing
- Implement BAL diff application in block import pipeline
- Modify `core/blockchain.go` to use BAL for state updates
- Add state root verification without re-execution

### Phase 4: RPC & Operations
- Modify `internal/ethapi/api.go` for partial state awareness
- Add appropriate errors for untracked contract queries
- Implement BAL history management and reorg handling

---

## Key Files to Modify

| File | Changes |
|------|---------|
| `eth/ethconfig/config.go` | Add `PartialStateConfig` |
| `core/state/partial/filter.go` | New: `ContractFilter` interface |
| `eth/protocols/snap/sync.go` | Filter storage sync by config |
| `core/blockchain.go` | BAL-based state updates |
| `internal/ethapi/api.go` | Partial state RPC handling |
| `cmd/utils/flags.go` | CLI flags for partial state |

---

## Open Items for Implementation

1. **BLOCKHASH opcode**: Verify 256 blocks of history is sufficient; check if other opcodes need block history

2. **Storage root verification**: When applying BAL storage diffs for tracked contracts, verify computed storage root matches account's storageRoot field

3. **Compression implementation**: Implement delta encoding + bitmap optimization for intermediate nodes (existing pathdb patterns can be adapted)

4. **Selective snap sync protocol**: Research if snap protocol needs extension or if filtering can be done client-side

---

## Verification Checklist

After implementation, verify:
- [ ] Can sync account trie completely via snap sync
- [ ] Can sync only configured contracts' storage
- [ ] BAL diffs apply correctly, state root matches header
- [ ] eth_getProof works for any account (proof generation)
- [ ] eth_getProof returns error for untracked storage
- [ ] Mempool accepts/validates transactions correctly
- [ ] Reorgs up to BAL retention depth work
- [ ] Deeper reorgs trigger recovery from full peers
- [ ] Total storage matches estimates (~30-40 GB + configured storage)

---

# DETAILED SPECIFICATIONS

---

## SPEC 1: Snap Sync Refactoring for Selective Storage

### Overview

The snap sync protocol in go-ethereum downloads account data and contract storage in parallel. For partial statefulness, we need to:
1. Download ALL accounts (unchanged behavior)
2. Download storage ONLY for configured contracts (new filtering)
3. Download bytecode ONLY for configured contracts (new filtering)

**Design Principle**: Keep original `Syncer` implementation untouched. Create a separate syncer implementation using a strategy/interface pattern that allows selection at runtime.

### Architecture: Strategy Pattern

```
                    ┌─────────────────────┐
                    │   SyncStrategy      │  (interface)
                    │   interface         │
                    └─────────┬───────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
    ┌─────────▼─────┐  ┌──────▼──────┐  ┌─────▼───────┐
    │ FullSyncer    │  │PartialSyncer│  │ (future)    │
    │ (wraps orig)  │  │(new impl)   │  │             │
    └───────────────┘  └─────────────┘  └─────────────┘
```

### Key Files

| File | Purpose |
|------|---------|
| `eth/protocols/snap/sync.go` | **UNCHANGED** - Original Syncer |
| `eth/protocols/snap/strategy.go` | **NEW** - SyncStrategy interface |
| `eth/protocols/snap/partial_sync.go` | **NEW** - PartialSyncer implementation |
| `core/state/partial/filter.go` | **NEW** - ContractFilter interface |
| `eth/downloader/downloader.go` | **MODIFIED** - Strategy selection |

---

## SPEC 2: Compression + Root Recomputation

### Overview

For partial statefulness, we store the full account trie (~300M accounts + intermediate nodes) but need efficient storage. This spec covers:
1. **REUSE** existing delta encoding infrastructure from pathdb
2. State root recomputation from BAL diffs

### Existing Compression Infrastructure (REUSE - DO NOT REIMPLEMENT)

**Location**: `triedb/pathdb/nodes.go` (lines 431-691)

go-ethereum **already has production-grade compression** we must reuse:

| Function | Purpose | Status |
|----------|---------|--------|
| `encodeNodeCompressed()` | Delta encoding with bitmap | **REUSE** |
| `decodeNodeCompressed()` | Decode compressed format | **REUSE** |
| `encodeNodeFull()` | Full-value encoding | **REUSE** |
| `encodeNodeHistory()` | Checkpoint + delta chains | **REUSE** |

---

## SPEC 3: BAL Processing Pipeline

### Overview

Block Access Lists (BALs) per EIP-7928 provide state diffs that allow partial nodes to update state without re-executing transactions.

### Existing BAL Implementation (Already in Geth)

**Location**: `core/types/bal/`

BAL types are already implemented in go-ethereum master:

| File | Contents |
|------|----------|
| `bal.go` | `ConstructionBlockAccessList`, `ConstructionAccountAccess`, builder methods |
| `bal_encoding.go` | `BlockAccessList`, `AccountAccess`, RLP encoding, hash computation |
| `bal_encoding_rlp_generated.go` | Generated RLP encoder/decoder |

---

## SPEC 4: RPC Modifications

### Overview

Partial state nodes can answer some RPC queries but not others. This spec defines the behavior.

### Error Codes

```go
var (
    ErrStorageNotTracked = errors.New("storage not tracked for this contract")
    ErrCodeNotTracked = errors.New("code not tracked for this contract")
)

const (
    ErrCodeStorageNotTracked = -32001
    ErrCodeNotTracked        = -32002
)
```

---

## SPEC 5: Configuration System

### CLI Flags

```go
var (
    PartialStateFlag = &cli.BoolFlag{
        Name:     "partial-state",
        Usage:    "Enable partial statefulness mode (reduced storage)",
        Category: flags.EthCategory,
    }

    PartialStateContractsFlag = &cli.StringSliceFlag{
        Name:     "partial-state.contracts",
        Usage:    "Contracts to track storage for (comma-separated addresses)",
        Category: flags.EthCategory,
    }

    PartialStateContractsFileFlag = &cli.StringFlag{
        Name:     "partial-state.contracts-file",
        Usage:    "JSON file containing contracts to track",
        Category: flags.EthCategory,
    }

    PartialStateBALRetentionFlag = &cli.Uint64Flag{
        Name:     "partial-state.bal-retention",
        Usage:    "Number of blocks to retain BAL history (default: 256)",
        Value:    256,
        Category: flags.EthCategory,
    }
)
```

---

## Implementation Task Breakdown

### Phase 1: Core Infrastructure (Foundation)

| Task ID | Task | Dependencies | Effort |
|---------|------|--------------|--------|
| 1.1 | Create `core/state/partial/` package structure | None | S |
| 1.2 | Implement `ContractFilter` interface | 1.1 | S |
| 1.3 | Add `PartialStateConfig` to ethconfig | None | S |
| 1.4 | Add CLI flags for partial state | 1.3 | S |
| 1.5 | Implement config loading (file + direct) | 1.3, 1.4 | M |

### Phase 2: Snap Sync Modifications (Selective Sync via Strategy Pattern)

| Task ID | Task | Dependencies | Effort |
|---------|------|--------------|--------|
| 2.1 | Create `SyncStrategy` interface in `strategy.go` | None | S |
| 2.2 | Create `FullSyncStrategy` wrapper (embeds original Syncer) | 2.1 | S |
| 2.3 | Create `PartialSyncer` struct in `partial_sync.go` | 1.2, 2.1 | M |
| 2.4 | Implement account processing with storage filtering | 2.3 | M |
| 2.5 | Add `markStorageSkipped` / `isStorageSkipped` helpers | 2.3 | S |
| 2.6 | Implement healing with skip checks | 2.5 | M |
| 2.7 | Modify Downloader to use `SyncStrategy` interface | 2.1, 2.2 | S |
| 2.8 | Add strategy selection based on config | 2.7 | S |
| 2.9 | Unit tests for PartialSyncer | 2.4, 2.6 | M |
| 2.10 | Integration test with partial filter | 2.9 | L |

### Phase 3: BAL Processing (State Updates)

| Task ID | Task | Dependencies | Effort |
|---------|------|--------------|--------|
| 3.1 | Add BAL key schema to `core/rawdb/schema.go` | None | S |
| 3.2 | Create `core/rawdb/accessors_bal.go` (following existing pattern) | 3.1 | S |
| 3.3 | Create thin `BALHistory` wrapper in `core/state/partial/history.go` | 3.2 | S |
| 3.4 | Implement `ApplyBALAndComputeRoot` using existing BAL types + trie | Phase 2 | L |
| 3.5 | Implement `applyStorageChanges` for tracked contracts | 3.4 | M |
| 3.6 | Add `ProcessBlockWithBAL` to BlockChain | 3.4, 3.3 | L |
| 3.7 | Implement reorg handling with BAL history | 3.3, 3.6 | L |
| 3.8 | Engine API integration for BAL delivery | 3.6 | M |
| 3.9 | BAL processing tests | 3.6, 3.7 | L |

### Phase 4: RPC Modifications (API Layer)

| Task ID | Task | Dependencies | Effort |
|---------|------|--------------|--------|
| 4.1 | Add `PartialStateError` and error codes | None | S |
| 4.2 | Add `PartialStateEnabled`, `IsContractTracked` to Backend | 1.2 | S |
| 4.3 | Modify `GetStorageAt` for partial state | 4.1, 4.2 | S |
| 4.4 | Modify `GetCode` for partial state | 4.1, 4.2 | S |
| 4.5 | Modify `GetProof` (account ok, storage filtered) | 4.1, 4.2 | M |
| 4.6 | Modify `Call` / `EstimateGas` with pre-check | 4.1, 4.2 | M |
| 4.7 | RPC behavior tests | 4.3-4.6 | M |

### Phase 5: Integration & Testing

| Task ID | Task | Dependencies | Effort |
|---------|------|--------------|--------|
| 5.1 | End-to-end partial sync test | Phase 2, Phase 3 | L |
| 5.2 | Verify storage size meets estimates | 5.1 | M |
| 5.3 | Reorg recovery test | Phase 3 | M |
| 5.4 | RPC integration test | Phase 4, 5.1 | M |
| 5.5 | Documentation updates | All | M |

### Effort Legend

- **S** = Small (few hours)
- **M** = Medium (1-2 days)
- **L** = Large (3-5 days)

---

## Critical Path

The critical path for minimum viable partial statefulness:

1. **Phase 1**: Configuration infrastructure
2. **Phase 2**: Selective snap sync via strategy pattern (accounts + filtered storage)
3. **Phase 3**: BAL processing (state updates without re-execution, using existing BAL types)
4. **Phase 4**: RPC modifications (proper error handling)
5. **Phase 5**: End-to-end test

This enables a working partial stateful node. Compression and full reorg handling can be added incrementally.

## Key Design Decisions Summary

| Decision | Approach | Rationale |
|----------|----------|-----------|
| Snap sync | Strategy pattern with separate `PartialSyncer` | Keep original `Syncer` untouched |
| BAL types | Use existing `core/types/bal/` | Already implemented in geth master |
| Filter interface | `ContractFilter` interface | Flexible, testable |
| Skip tracking | DB markers + in-memory map | Persist across restarts |
| RPC errors | Custom error codes | Clear user feedback |

---

## Reuse vs. New Code Summary

### REUSING (Do Not Reimplement)

| Component | Existing Location | How We Use It |
|-----------|-------------------|---------------|
| **BAL Types** | `core/types/bal/` | Import directly |
| **Compression** | `triedb/pathdb/nodes.go` | `encodeNodeCompressed()`, `encodeNodeHistory()` |
| **Delta Encoding** | `trie/node.go` | `NodeDifference()` |
| **Checkpoint Mechanism** | `triedb/pathdb/config.go` | `FullValueCheckpoint` config |
| **Diff Layers** | `triedb/pathdb/difflayer.go` | `nodeSetWithOrigin`, `StateSetWithOrigin` |
| **History Key Patterns** | `core/rawdb/schema.go` | Follow `StateHistoryAccountBlockPrefix` pattern |
| **History Accessors** | `core/rawdb/accessors_history.go` | Follow Read/Write/Delete triplet pattern |
| **Safe Deletion** | `core/rawdb/database.go` | `SafeDeleteRange()` for pruning |
| **Filter Patterns** | `eth/filters/filter.go` | Reference for contract filtering |
| **Trie Interface** | `trie/trie.go` | Standard trie operations |

### CREATING NEW

| Component | New Location | Purpose |
|-----------|--------------|---------|
| `SyncStrategy` interface | `eth/protocols/snap/strategy.go` | Abstract sync implementations |
| `PartialSyncer` | `eth/protocols/snap/partial_sync.go` | Filtered storage sync |
| `ContractFilter` | `core/state/partial/filter.go` | Contract tracking interface |
| `PartialState` | `core/state/partial/state.go` | BAL application + root computation |
| BAL key schema | `core/rawdb/schema.go` | Add `balHistoryPrefix` |
| BAL accessors | `core/rawdb/accessors_bal.go` | Read/Write/Delete following pattern |
| `BALHistory` wrapper | `core/state/partial/history.go` | Thin layer over rawdb |
| `ProcessBlockWithBAL` | `core/blockchain_partial.go` | Block processing entry point |
| RPC error codes | `internal/ethapi/` | Partial state errors |
| Config | `eth/ethconfig/config.go` | `PartialStateConfig` |
| CLI flags | `cmd/utils/flags.go` | Partial state flags |
