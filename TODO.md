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

## Merkle witnesses do not capture every bytecode they read

`ExecuteStateless` checks the state database's latched error before trusting
its root, but only for the binary tree. Enabling the same check for merkle
fails `TestWitnessCreationAndConsumption` (`eth/catalyst/api_test.go`) with
`code is not found <hash>`: the witness is missing a bytecode the replay reads.

`Witness.AddCode` is only reached from `StateDB.GetCode` and
`StateDB.GetCodeSize`, so a `stateObject.Code()` arriving by any other route is
never recorded. The replay then substitutes empty code and carries on, because
`setError` only latches and `IntermediateRoot` never consults it.

The root still matched in that test, which is why nobody noticed - the missing
code happened not to influence that block's state. On a block where it does,
the result is a wrong root reported as a good one. Closing this means finding
every path to `stateObject.Code()` and witnessing there, after which the check
in `core/stateless.go` can drop its `IsPBT` condition.

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
