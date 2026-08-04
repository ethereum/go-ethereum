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
