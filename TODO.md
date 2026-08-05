# TODO

Known gaps in the EIP-8297 binary tree (PBT) work, recorded so they are not
rediscovered by accident.

## The engine API has no stateless-consume or witness-build path for the tree

`eth/catalyst/witness.go` has `NewPayloadWithWitnessV1` through **`V5`**,
`ExecuteStatelessPayloadV1` through `V4`, and `ForkchoiceUpdatedWithWitnessV1`
through `V3`. So *producing* a binary tree witness already works:
`NewPayloadWithWitnessV5` is gated on `forks.Amsterdam` and returns one on
payload insertion.

What is missing is the other two directions:

- **Consumption** — an `ExecuteStatelessPayloadV5`. `V4` stops at Bogota.
- **Requesting a build with a witness** — a `ForkchoiceUpdatedWithWitnessV4`.
  `V3` stops at Bogota too.

Both refuse cleanly meanwhile rather than misbehaving, because **Amsterdam is
absent from both fork gates**. `ExecuteStatelessPayloadV4` admits
`Prague, Osaka, BPO1-5, Bogota`, so a binary tree payload is rejected with
*"newPayloadV4 must only be called for prague/osaka payloads"* well before
reaching `ExecuteStateless`. `ForkchoiceUpdatedWithWitnessV3` admits that set
plus `Cancun`, and only checks it when payload attributes are present — which
is exactly the case that requests a build, so the gap that matters is covered.

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

## Two remaining unwitnessed paths to `stateObject.Code()`

`Witness.AddCode` is reached only from `StateDB.GetCode` and
`StateDB.GetCodeSize`, so a `stateObject.Code()` arriving by any other route
goes unrecorded. The replay then substitutes empty code and carries on, because
`setError` only latches and `IntermediateRoot` never consults it — a wrong root
reported as a good one.

The path that actually fired was `updateStateObject`, which asked for a code
size on every dirty account and loaded the whole contract to measure it. That
is fixed: the binary tree takes the size from the stem it is writing, the
merkle trie never needed it, and `core/stateless.go` now holds both trees to
the completeness check. Two dormant ones are left.

- `recordAccessListChanges` (`core/state/statedb.go`) calls `obj.Code()` under
  `state.codeSet`. Amsterdam/BAL only, and only for accounts with a journalled
  `SetCode`, whose blob is already in hand — so it does not reach the reader
  today. Note that this holds by a cache hit rather than by construction:
  `Code()` short-circuits on `len(s.code) != 0`.
- `ReaderWithBlockLevelAccessList.Code`/`CodeSize`
  (`core/state/reader_eip_7928.go`) read code below the `AddCode` layer
  entirely, falling through to the wrapped reader on an access-list miss.

Neither is reachable in a way that breaks a block today. Both would be found by
the same test shape that caught the first: a block that touches a contract it
never executes, replayed with the completeness check on.

## Is `code_size` worth putting in flat state?

Not for the write path — `UpdateAccount` preserves the resident size, which
costs an in-memory lookup on a group the write resolves anyway. The question is
the read path, where the cost is adversarially reachable.

`opExtCodeSize` (`core/vm/instructions.go`) reaches `StateDB.GetCodeSize`
(`core/state/statedb.go`), which reads the whole blob either way:

```go
if s.witness != nil {
    s.witness.AddCode(stateObject.Code())   // building a witness: the blob, always
}
return stateObject.CodeSize()               // otherwise: a full read on a cold cache
```

These are alternatives rather than two costs on one call. `Code()` caches into
`stateObject.code`, so once `AddCode` has run the following `CodeSize()` returns
from memory and never reaches the reader. With no witness being built,
`CodeReader.CodeSize` (`core/state/database_code.go`) consults a size-only cache
first, so a warm node answers cheaply and a cold one does `len(r.Code(...))` —
reading the entire bytecode to learn its length.

The witness line is the serious one, because no cache spares it: whenever a
witness is being built the blob goes in. A block whose transactions do nothing
but `EXTCODESIZE` against many large contracts drags every one of their
bytecodes into the witness while reading no code at all. At Amsterdam's
`params.MaxCodeSizeAmsterdam` (64 KiB, enforced in `core/vm/common.go`) that is
megabyte-scale amplification for a block that learned nothing but a handful of
integers.

`code_size` in flat state would let both the read and the witness carry four
bytes instead of the blob. It is not free: flat state persists
`types.SlimAccount`, so this is a stored-format change requiring regeneration
on every node.

**Measure before deciding.** Build the adversarial block — many distinct large
contracts, `EXTCODESIZE` only, nothing warm — and compare its witness size and
execution time against a block doing the same number of ordinary reads. That
ratio decides whether the format change earns itself. Do not change
`types.StateAccount` instead: PBT reads accounts through flat state first, so
a field the flat reader cannot fill arrives zero and would erase the size it
was added to carry.

## Witness statistics have no binary-aware histogram

`--vmwitnessstats` is refused on the binary tree
(`core/blockchain.go triedbConfig`) because `WitnessStats` reads a node's path
as a nibble string and its depth as that string's length, bucketed into a fixed
sixteen levels. A binary path is a two-byte bit count followed by packed bits,
so the depth is wrong immediately and passes sixteen after 113 bits.

Making it work needs the bit count read out of the path encoding and a
histogram wider than `trie.LevelStats`'s fixed sixteen. Refusing only stops the
crash.

## Nothing ever reclaims a code-zone leaf

Now that every code chunk is content-addressed, no path in this tree removes
one. `UpdateContractCode` only writes, and `DeleteAccount` deliberately leaves
the zone alone. Shorter code, replaced code and destroyed contracts all leave
their old chunks behind.

The spec requires the opposite: a `CODE_ZONE` leaf must go once no account in
the resulting state has that `code_hash`, and must be kept otherwise. The
reference does it in `remove_code_chunks` (`binary_trie/embedding.py`), called
on account deletion only, gated on a `code_hash_survives` scan of its whole
account dict, and exempting delegation indicators because those live in their
account's own header.

Geth has nothing to build that scan on: there is no code-hash index and
`rawdb.DeleteCode` has no callers. Approximating it with the block's touched
accounts is unsound — an account the block never touched may hold the same
bytecode, and dropping the chunks would take its code with it.

This is precisely the locality problem [EIP #12114] removes. With the
delegation indicator in the account header, the check becomes decidable from
the transaction alone: `SELFDESTRUCT` reaches only same-transaction creations,
so any code leaf predating the transaction belongs to an account the
transaction cannot delete. Build the reclamation on top of that change rather
than trying to engineer around the scan.

Until then the divergence is real but narrow: it needs an account with deployed
code to be deleted, which post-EIP-6780 means created and destroyed inside one
transaction.

## The MPT→PBT converter needs rewriting, not patching

`cmd/geth/bintrie_convert.go` was built for a layout that no longer exists, and
the move of code into the content-addressed zone invalidated its central
assumption rather than shifting a constant. It still converts, so this is not
urgent — but it should get a dedicated session rather than another patch.

- **Its write pattern is inverted.** Every contract's chunks used to land in
  that contract's own header stem, so writes followed the account-hash
  iteration order the loop is built around. They now scatter into `CODE_ZONE`
  stems keyed by `KeyHash(code_hash ‖ tree_index)`, interleaved with a
  commit-and-reload every 1000 accounts (`runConversionLoop`) or on memory
  pressure (`maybeCommit`). The locality that made those flushes cheap is gone.
- **Deduplication is now the common case.** Identical bytecode shares leaves
  from chunk 0 rather than only past chunk 128, so the converter rewrites the
  same chunks once per holder and the output is smaller than it plans for.
  Neither is accounted for.
- **Delegation will add a case it cannot see.** Once the EIP-7702 indicator
  moves into the account header ([EIP #12114]), a delegated account in the
  merkle source is still a 23-byte code blob and would be chunked as ordinary
  code. Converted state would then disagree with replayed state — a
  correctness break, not a slowdown.
- **Nothing would catch any of that.** `bintrie_convert_test.go` reads back
  through `GetAccount`/`StorageSlotKey` and asserts no root and no leaf counts.
  The rewrite's first job is the test that is missing: convert a fixture and
  compare its root against replaying the equivalent transactions.

[EIP #12114]: https://github.com/ethereum/EIPs/pull/12114

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
  key derivation, never a root. Re-exporting does not close it on its own:
  `testdata/export_vectors.py` builds its rooted populations from account and
  storage keys only, so the exporter has to grow one that contains code chunks.
  That belongs to the EEST/EELS integration phase.
- **One harness blocker before EEST fixtures can run.** `execBlockTest`
  (`tests/block_test.go`) runs every fixture under both the hash and path
  schemes, and the binary tree hard-fails on anything but path. It also always
  requests witness building, which used to be a second blocker; the tree
  supports that now.
- **`TestT8n`** fails on the binary tree fixtures because the prestate is
  reopened with an already-committed trie. Out of scope by instruction.
- **The encoded multiproof is malleable, though not unsound.** Sweeping every
  byte of an encoded proof and flipping it, most mutations are rejected, but a
  run of them still verify: 48 such offsets before this branch, 64 after, in
  both cases concentrated in two 32-byte spans that look like per-group
  `present` bitmaps. None of them changes a value the proof proves, which is
  the property `TestMultiproofRejectsForgery` now asserts. So this is two
  encodings of one proof rather than a forged answer — but the encoding should
  be canonical before the multiproof carries a witness, or the same statement
  gets more than one wire form. The mechanism was not chased down.
- **`StackBuilder`** (`trie/bintrie/stackbuilder.go`) has no production caller.
  Its natural consumer is the offline conversion in `cmd/geth`, which still
  inserts one stem at a time. Revisit when conversion is benchmarked; delete if
  still unwired.
