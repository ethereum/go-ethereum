# TODO

Known gaps in the EIP-8297 binary tree (PBT) work, recorded so they are not
rediscovered by accident.

## Pin the `debug_storageRangeAt` refusal with a test

`storageRangeAt` refuses under the binary tree — `eth/api_debug.go:240`, inside
the function at `:234`:

```go
if statedb.Database().TrieDB().IsPBT() {
    return StorageRangeResult{}, errors.New("debug_storageRangeAt is not supported for the binary tree")
}
```

The guard is correct and was verified by reading, but **no test exercises it**.
It is the one entry in the capability contract index (see the table at the top
of `core/pbt_capabilities_test.go`) marked *not pinned*, and every other refusal
in that table has a test next to its guard.

Why it matters: the tree keeps no per-account storage trie to range over and the
account carries no storage root, so without the guard the empty value below it
would report **every contract as having no storage** — a wrong answer rather
than an error. That is the failure mode the rest of the capability work exists
to prevent, and it is currently held only by code review.

What the test needs: `TestStorageRangeAt` (`eth/api_debug_test.go:274`) builds
its state with `state.New(root, db)` over a merkle-patricia database, so a PBT
case needs a binary-tree `triedb` in the `eth` package. `core/state`'s
`newPBTState` helper (in `core/state/pbt_semantics_test.go`) is the shape to
copy — it uses `triedb.NewDatabase(disk, triedb.PBTDefaults)`. Assert the call
returns an error naming the binary tree, and keep the existing merkle case as
the control so the refusal is known to be tree-specific.

## The engine API cannot carry a stateless binary tree payload

`eth/catalyst/witness.go` has `ExecuteStatelessPayloadV1` through `V4`, and
`ForkchoiceUpdatedWithWitnessV1` through `V3`. There is no `V5` of either. The
binary tree activates at Amsterdam and so needs `NewPayloadV5` /
`ForkchoiceUpdatedV4`, which means a binary tree payload can neither be
*requested* with a witness nor stateless-*executed* through the engine API.

Nothing guards this. Those methods relied on the blanket refusal in
`core/stateless.go`, which is gone now that stateless execution works, so they
will attempt a binary tree payload and fail somewhere further in rather than
saying what is missing.

Shape to copy: `TestWitnessCreationAndConsumption` (`eth/catalyst/api_test.go`)
drives the whole loop, but only at V3. The binary tree fixture to extend is
`TestPBTNodeProducesAndImportsBlocks` (`eth/catalyst/pbt_test.go`), which never
touches witnesses today.

## The witness could be a multiproof

The witness ships the nodes a block resolved, which is the same shape as the
per-stem group records: about 26 KB to witness a whole 24 KiB contract, but
about 4.7 KB to witness reading one chunk of it. `trie/bintrie/multiproof.go`
answers the same single-chunk read in 672 B, roughly seven times smaller, and
real blocks read sparsely rather than densely.

Swapping it in needs three things the node-set format does not: recording which
keys a block touched, covering written stems whole (`insStem` refuses a write
into a partially shipped stem), and the absence handling the multiproof already
grew. `TestMultiproofSize` exists to measure the swap when it happens.

## Witness statistics have no binary-aware histogram

`--vmwitnessstats` is refused on the binary tree
(`core/blockchain.go triedbConfig`) because `WitnessStats` reads a node's path
as a nibble string and its depth as that string's length, bucketed into a fixed
sixteen levels. A binary path is a two-byte bit count followed by packed bits,
so the depth is wrong immediately and passes sixteen after 113 bits.

Making it work needs the bit count read out of the path encoding and a
histogram wider than `trie.LevelStats`'s fixed sixteen. Refusing only stops the
crash.

## Also deferred, for context

These are known and tracked elsewhere; listed so this file is the single place
to look.

- **The code zone (`0x01`) has no coverage against the reference vectors.**
  `trie/bintrie/multiproof_test.go`'s `TestCodeZoneKeyVerifies` proves a
  `CODE_ZONE` key against a real root, so the zone is no longer unverified
  outright. What is still missing is spec conformance: counting the leading
  zone byte of every hashed entry in `trie/bintrie/testdata/eip8297_vectors.json`
  gives 601 for accounts (`0x00`) and 266 for storage (`0xFF`), and **zero** for
  content-addressed code, which appears only under `embedding_vectors.chunks` —
  key derivation, never a root. Closing it needs a re-export from the reference
  implementation via `testdata/export_vectors.py`, so it belongs to the
  EEST/EELS integration phase.
- **One harness blocker before EEST fixtures can run.** `execBlockTest`
  (`tests/block_test.go`) runs every fixture under both the hash and path
  schemes, and the binary tree hard-fails on anything but path. It also always
  requests witness building, which used to be a second blocker; the tree
  supports that now.
- **`TestT8n`** fails on the binary tree fixtures because the prestate is
  reopened with an already-committed trie. Out of scope by instruction.
- **`StackBuilder`** (`trie/bintrie/stackbuilder.go`) has no production caller.
  Its natural consumer is the offline conversion in `cmd/geth`, which still
  inserts one stem at a time. Revisit when conversion is benchmarked; delete if
  still unwired.
