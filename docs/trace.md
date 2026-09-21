# Experimental trace namespace

This branch implements the nine Parity-style RPC methods against the proposed
[trace profile](https://github.com/banteg/execution-apis/blob/feat/trace/docs-api/docs/trace-profile.md).
The proposal is under discussion; this fork does not represent upstream Geth adoption.

Enable it with `--http --http.api eth,net,web3,trace` or add `trace` to `--ws.api`.
IPC exposes it alongside the other registered APIs. Existing `debug_*` methods keep
their behavior. No database migration or additional index is required.

| Method | Behavior |
| --- | --- |
| `trace_call(call, types, block?)` | Simulate on the selected block's post-state, using that block's environment. |
| `trace_callMany([[call, types], ...], block?)` | Execute in order on shared temporary state; reverted execution writes roll back. |
| `trace_rawTransaction(bytes, types)` | Validate and execute a signed transaction against latest state without broadcasting it. |
| `trace_replayTransaction(hash, types)` | Replay in original transaction context; include `transactionHash`. |
| `trace_replayBlockTransactions(block, types)` | Replay every transaction in order, one envelope per transaction. |
| `trace_transaction(hash)` | Return the localized call tree in preorder. |
| `trace_get(hash, position)` | Select a tree path: `[]` is the root, `["0x0"]` its first child. |
| `trace_block(block)` | Return transaction trees and actual historical PoW block/uncle rewards. |
| `trace_filter(filter)` | Scan a bounded canonical snapshot, match addresses, then paginate. |

`types` selects any combination of `trace`, `stateDiff`, and `vmTrace`. An empty
selection still executes and returns output. Unrequested trace is `[]`; other
unrequested families are `null`. Unknown transactions and paths return `null`.
Pending-block tracing and extra arguments are rejected. Calls accept the draft's
unsigned transaction fields; blob and authorization overrides are outside that
call profile. Signed raw and mined transactions use the chain's applicable rules.

Call trees omit nested zero-value precompile frames, retaining root precompiles
and nested frames with nonzero transferred or inherited value. Paths and child
counts describe the emitted tree. VM return-memory effects remain available even
when the child frame is omitted. Trees retain revert bytes, failed children, creation
results and selfdestruct actions. Their gas fields describe frame execution, not
transaction intrinsic gas. `stateDiff` compares pre-transaction and finalized
state, including fees and authorization changes. Deleted accounts carry deletion
markers; storage entries describe touched slots, not an enumeration of the old trie.
`vmTrace` contains executing bytecode and same-instruction stack/memory/storage
deltas, recursively nested for executed child bytecode. Precompile calls have no
child bytecode trace.

Filter address matching is OR within each list and AND between lists. Omitted or
empty lists are unrestricted. Creation recipients are successful created addresses;
selfdestruct uses the destroyed address and beneficiary; rewards have only a
recipient. Post-Merge blocks do not receive synthetic issuance rewards.

## Limits and history

- Calls use Geth's configured RPC gas cap; signed raw transactions exceeding it are rejected.
- Each execution has a five-second timeout. Batches, block replay and filter scans
  have a thirty-second timeout.
- Filter ranges are limited to 1,000 blocks and returned results to 10,000 records.
  Explicit `count` ends a page normally; exceeding the default limit returns an
  error. `trace_callMany` accepts at most 10,000 calls.
- Each execution limits captured frames/opcodes to 1,000,000 and tracked byte
  payloads to 64 MiB. This is an output guard, not a bound on total process memory.
  Limit and cancellation failures return an error, not a partial trace.
- Historical methods require retained or reconstructible state and transaction
  lookup data. Recognized unavailable-state errors use the proposed code `4444`.
  The namespace does not enable archive retention or add an address index.

## Validation

Run `go test ./eth/tracers -run TestTraceNamespace` for execution, nested calls,
state rollback, signed authorization, cancellation, fork and wire-format tests.
The [interop project](https://github.com/banteg/trace-interop) builds this fork from
source and captures the same frozen Hive corpora as the other clients. Its reports
separate measured assertions, schema checks, and untested behavior; passing them
is not proof of complete conformance.
