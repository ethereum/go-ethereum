# EIP-8347 online migration: dual-tree runtime + BAL replay

2026-08-19. Base: `pbt` @ bcf30164bf. Status: approved design.

## Context

EIP-8347 (Offline State Migration to the PBT; requires 7928/8159/8297) moves state from the
merkle-patricia trie to the partitioned binary tree off the consensus path. The offline half
ships already: `cmd/geth/bintrie_convert.go` (converter) and `cmd/geth/bintrie_import.go`
(dual-check importer, records `rawdb.PBTAnchor`). The fork modeling (#26) added
`ChainConfig.BinaryTrieTime` and accepts mid-chain schedules; activating one is what this
design builds:

1. Geth holds both state databases at once: MPT canonical, PBT shadow.
2. A BAL-replay follower advances the shadow to the tip and tracks it — no re-execution.
3. At `BinaryTrieTime` the header `stateRoot` swaps source. Both trees are maintained from
   activation until the fork finalizes (EIP transition window), then MPT maintenance stops.

## Three-mode config model

- New predicate `ChainConfig.IsBinaryTrie(num, time)` beside `IsAmsterdam`; `IsPBT()` keeps
  meaning "the fork is scheduled". `Rules` stays PBT-free.
- A tri-state resolver replaces `pbtEnabled` (core/genesis.go): `modeMPT` (no fork),
  `modePBTNative` (active at genesis), `modeMigration` (scheduled later). Genesis time comes
  from the supplied `Genesis` or the stored genesis header; empty DB + nil genesis stays
  `modeMPT`.
- `Genesis.IsPBT()` is redefined to "active at genesis": a migration chain hashes and commits
  its genesis as plain MPT (same genesis hash as an MPT chain), through the existing
  `hashAlloc`/`flushAlloc` unchanged.
- `NewBlockChain`: the "PBT state without a schedule" guard applies only in `modeMPT`;
  `modeMigration` requires the path scheme, builds the canonical triedb with the flavor active
  at head, and constructs the follower next to the txIndexer. The shadow triedb is created
  lazily by the follower (a fresh namespace self-attests its flat state on first open).

## The follower (`core/bintrie_follower.go`, `core/bintrie_replay.go`)

txIndexer-shaped: one background run at a time, woken by `ChainHeadEvent`. Events are wake-ups
only — resume points come from walking parent hashes back to the last replayed block. BALs are
read by (number, hash); they are persisted for every locally executed block including side
chains, live in the never-pruned "bals" freezer table, and the downloader backfills them for
synced ranges.

States: `idle → seeding → catchup ⇄ tracking → swapped-tracking → done`; error sink
`stalled(reason)`.

- `seeding`: an anchor (`rawdb.ReadPBTAnchor`) starts the cursor there; else a fresh namespace
  on an Amsterdam-at-genesis chain seeds the shadow genesis via `getGenesisState` +
  `flushAlloc` against the shadow triedb; else `stalled("conversion artifact required")` —
  pre-Amsterdam blocks have no BALs.
- `catchup` (head − cursor > 128): batched fold-replay; `shadow.Commit(root, false)` per batch
  flattens layers; no per-block roots inside a batch (the EIP allows this while behind).
- `tracking` (within 128): per-block replay, one diff layer per block, per-hash shadow-root
  records.
- Reorgs: the shadow reorgs natively via the pathdb layer-diff tree, exactly like the canonical
  PBT chain. Sibling branches coexist as diff layers; the tree identifies the revert point (the
  common ancestor whose layer is live); the follower walks back from the new head to the last
  replayed block with a live shadow root and replays the new branch's BALs forward from that
  layer — the mirror of `recoverAncestors`' walk-back-then-reimport, with
  `TestPBTReorgInsideWindowKeepsFlatState` as canonical precedent. No BAL inversion is needed
  or possible (post-values only): the ancestor state is already in the tree, and stale branch
  layers are flattened away by `cap()`. Empty updates short-circuit before pathdb, so empty
  blocks record cursor + root only, no layer. Depth limit = the canonical PBT chain's own: a
  fork point below the persisted disk layer is unreachable; only there does the shadow stall,
  and recovery is re-anchoring from a conversion artifact.
- Snap sync: during migration only the canonical MPT may snap-sync; the PBT namespace is never
  a sync target (structurally refused: flat state cannot be regenerated from the tree). The
  follower idles while the sync flag is set and pauses on SnapSyncStart. A snap-synced node
  cannot genesis-seed the shadow — it must be anchor-seeded.
- Side chains: lazy replay, on canonicalization; the walk-back covers them.

Crash-consistency invariants:

- Truth is the shadow pathdb itself (journal if valid, else the disk layer inside the "b"
  namespace). The persisted cursor is a hint; on restart, if the cursor root is not live, walk
  back through the per-hash records to a live root.
- Per-hash root records are written at replay time, strictly before durability, so the index is
  a superset of durable roots; resume = index[pathdb head root] + 1 (or anchor + 1 / genesis).
- Re-replay is idempotent: the same BAL over the same parent yields all no-ops, an empty
  update, and no layer.
- Clean stop: drain the follower, then journal the shadow at the follower's root (not
  `CurrentBlock().Root`), wired into the blockchain's stop path.

## Replay engine (reuse, not reimplement)

Per block B with parent shadow root R and BAL L: `state.New(R, NewPBTDatabase(shadow, codedb))`
(no prefetcher, witness, or tracker) → `ApplyBlockAccessList(L)` → `Commit(B.Num,
IsEIP158(num), IsCancun(num, time))` — the canonical import's own arguments. Commit runs
IntermediateRoot internally and flows through the existing binary-tree pass, so creation
`code_hash` leaves, delegation swaps and the deletion trigger match execution by construction.
The EIP-161 touched-empty removal is explicit in the BAL (#27: a zero balance change for
existing accounts), so BAL ⇒ exact post-state. Post-swap, the same call over
`NewMPTDatabase(shadow, codedb)` maintains the MPT (nil snapshot is fine on the path scheme).

Batching uses a new `bal.Fold` primitive. `bal.Merge` is not cross-block-safe: it merges by raw
access index while `ApplyBlockAccessList` takes the highest index per field, so block N's
index-10 write would beat block N+1's index-2 write. Fold is last-write-wins per field and slot
across ascending blocks, emitted at index 0, with a conservative deletion-split guard: when a
folded account hits the explicit-removal marker (a zero balance change) and a later block
touches it again, the batch splits there so the removal materializes (storage wipe) before the
recreation applies. Post-Cancun the guard should never fire (EIP-6780 confines selfdestruct to
the creation transaction; EIP-161 empties do not exist).

## New rawdb surface

- Shadow-root table: prefix "k" + num(8) + hash(32) → root(32), in the raw namespace — "the
  non-header root of this block" (PBT pre-fork, MPT during the window; later feeds the Phase-4
  shadow-root sidecar). Prunable below finality later.
- `PBTMigrationCursor` (num + hash + root, a hint) and `PBTMigrationPhase` (`done` marker so a
  restart after MPT disposal does not consult the disposed tree).
- `PBTAnchor` stays the immutable conversion record.

## Activation, window, termination

- One helper `bc.execStateAndRoot(parent, num, time)` → {flavor, root}: flavor by
  `IsBinaryTrie(num, time)`; root = parent.Root, except at the boundary (PBT child, MPT parent)
  → the parent's recorded shadow root. Wired into `setupExecutionState`, `StateAt`,
  `HistoricState` (refuses only PBT headers), the miner via a new `StateForBuilding`, and
  `HasBlockAndState` (else newPayload answers ACCEPTED forever at the boundary).
- Import gate: the newPayload path returns ACCEPTED/delayed while triggering a priority
  catch-up (never a synchronous stall — newPayload is serialized); the insertChain path does a
  bounded wait. A stalled follower yields a hard, actionable error. Loud readiness logging well
  before the fork.
- Role swap (under chainmu, first post-fork block): `bc.triedb` ← the PBT handle (atomic
  pointer; read-path audit at implementation), the follower takes the MPT handle and flips to
  `swapped-tracking`, recording per-hash MPT roots — post-fork headers carry PBT roots, so
  those records are the only place MPT roots live. The first PBT block's header-root validation
  is the first end-to-end check of the whole shadow lineage. A restart mid-window re-derives
  the shape from `IsBinaryTrie(head)`.
- Termination: on each head event, check the finalized block (set only by FCU; may be nil for
  long stretches — the window just stays open). Once finalized is post-fork by timestamp: stop
  MPT replay, journal and close the MPT triedb, write phase = done, never restart (a
  maintenance gap is unfixable). Until then MPT state history keeps accumulating — that is what
  "recoverable to MPT" requires.

## Task breakdown (stacked PRs)

M2 — replay engine + follower (shadow is write-only extra; no behavior change elsewhere):

1. `params`: `IsBinaryTrie(num, time)`.
2. `core/types/bal`: `Fold` + deletion-split guard (+ fuzz vs per-block replay).
3. `core/rawdb`: shadow-root table, cursor, phase key.
4. `core`: tri-state mode resolver; `Genesis.IsPBT` redefinition; NewBlockChain guard/mode
   plumbing (runtime inert: canonical stays MPT).
5. `core/bintrie_replay.go`: per-block + batched replay over a shadow handle.
6. `core/bintrie_follower.go`: lifecycle, seeding, catch-up/tracking, hash-walk reorgs, crash
   recovery, snap-sync interlock, `MigrationProgress`.
7. `core/rawdb`: freezer metrics namespace split for the PBT flavor.

M3 — activation + window:

8. `core`: `execStateAndRoot` + boundary translation (setupExecutionState / StateAt /
   HistoricState / HasBlockAndState).
9. `miner`: `StateForBuilding`.
10. `core` + `eth/catalyst`: the import gate.
11. `core`: role swap, `swapped-tracking` MPT replay, restart-mid-window, finality termination.
12. `cmd/utils` + `cmd/geth`: `MakeTrieDatabase` guard parameterization for hybrid datadirs.
13. `eth` / `internal/ethapi`: MigrationProgress RPC.

## Verification

- THE invariant (primary oracle): the shadow root at height B equals `convertState` over the
  MPT at B. The twin-chain oracle (replay vs a PBT-native execution of the same transactions)
  is unsound: EIP-2935 writes parent hashes, the two arms' block hashes necessarily differ, and
  `TestBothCommitmentsRunTheSameChain` already asserts the roots differ.
- Semantics pins against the shadow root: creations, 7702 set/clear/round-trip, EIP-161
  explicit removal, EIP-8264 selfdestructs, header-stem vs storage-zone zero-writes, no-op
  SSTORE, withdrawals-only, system-contracts-only, empty blocks.
- Batch-vs-per-block equivalence; crash recovery; reorg-within-window; reorg-past-window
  detection; missed-events rescan.
- M3: activation swap; boundary parent-root resolution; lagging gate returns SYNCING/ACCEPTED,
  never INVALID; window MPT shadow stays Recoverable; reorg back across the boundary; finality
  stop idempotence; restart mid-window; snap-sync refusal; hybrid-datadir tooling.

## Risks and accepted costs

- No pre-fork validation of the shadow lineage (the Phase-4 sidecar is future work); the
  converter-parity test is the local detector, a two-node debug comparison is cheap insurance.
- The PBT shadow writes state history it can never use for rollback (mandatory for pathdb boot
  alignment; roughly doubles history disk during migration). Accepted; a "thin history" mode is
  a later optimization.
- A boundary reorg deeper than the PBT in-memory tree after its disk layer passes the fork
  block requires re-anchoring — the canonical PBT chain has the same limit.
- Two pathdb write buffers double memory in migration mode; consider halving the shadow's.
- Snap serving/downloading for PBT roots post-activation is out of scope.
- History-pruned nodes with an anchor below the cutoff: unresolved, later milestone.

## Status addendum (2026-08-20)

Everything above shipped on `pbt-bal-replay`, and the completion sprint closed the honest
gaps: the follower runs one direction per flavour and the merkle window advances live through
the fork (proven against a reverse conversion of the binary state); shutdown journals each
handle at the newest root it holds, so mid-window reboots resume from cursors on a
persistent-datadir harness; the importer pairs with the follower end to end from an anchor;
missing access lists backfill over eth/71 with forging peers dropped; a fresh artifact
replaces a stale position through the --force wipe; MigrationWindowBlocks closes rehearsal
windows without finality; progress reports per direction; and debug_shadowStateRoot plus the
debug_shadowRoots stream expose the sidecar feed. TestFullMigrationLifecycle and
TestFullMigrationLifecycleBlocksKnob are the acceptance runs.
