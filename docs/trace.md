# Experimental trace namespace

This branch implements the nine Parity-style RPC methods against the proposed
[trace profile](https://github.com/banteg/execution-apis/blob/feat/trace/docs-api/docs/trace-profile.md).
The proposal is under discussion; this fork does not represent upstream Geth adoption.

Enable it with `--http --http.api eth,net,web3,trace` or add `trace` to `--ws.api`.
IPC exposes it alongside the other registered APIs. Existing `debug_*` methods keep
their behavior. No database migration or additional index is required.

| Method | Behavior |
| --- | --- |
| `trace_call(call, types, block?, stateOverrides?, blockOverrides?)` | Simulate on the selected block's post-state, using that block's environment. |
| `trace_callMany([[call, types], ...], block?, stateOverrides?, blockOverrides?)` | Execute in order on shared temporary state; reverted execution writes roll back. |
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
Pending-block tracing and extra arguments are rejected. Calls accept the standard
unsigned transaction fields, including chain ID, blob context and authorization
lists under the selected fork's rules. Unknown call fields are ignored; known
fields and conflicting transaction types are validated. Fees follow `eth_call`:
omitted fee fields default to zero, a zero effective gas price runs with `BASEFEE`
0, and a supplied or defaulted zero `maxFeePerBlobGas` runs with `BLOBBASEFEE` 0.
Positive prices are validated against the base fee, funded and charged. A supplied
nonce is accepted but neither validated nor used; CREATE addresses derive from the
sender's state nonce. Signed raw and mined transactions use the chain's applicable
rules. Malformed parameters return `-32602` and unknown selected blocks `-32001`. Rejected unsigned calls use the
`eth_simulateV1` codes: `-38012` fee cap below base fee, `-38013` intrinsic gas,
`-38014` insufficient funds, `-38025` initcode size and `-38026` gas above the RPC
gas cap; a priority fee above the fee cap and other invalid calls are `-32602`.
Rejected signed transactions use the `eth_sendRawTransaction` groups: `1` nonce too
low, `2` nonce too high, `800` intrinsic gas, `804` priority fee above fee cap,
`806` fee cap below base fee, `809` insufficient funds and `-32003` otherwise.
Valid transactions that revert or halt in the EVM return execution results.

Call trees omit nested zero-value precompile frames, retaining root precompiles
and nested frames with nonzero transferred or inherited value. Paths and child
counts describe the emitted tree. VM return-memory effects remain available even
when the child frame is omitted. Trees retain revert bytes, failed children, creation
results and selfdestruct actions. Their gas fields describe frame execution, not
transaction intrinsic gas. `stateDiff` compares pre-transaction and finalized
state, including fees and authorization changes. Deleted accounts carry deletion
markers; storage entries describe touched slots, not an enumeration of the old trie.
Storage slots use creation or deletion markers only when the account itself is
created or deleted; on an account present before and after, every changed slot,
including one set from or to zero, is a `*` change between 32-byte words.
`vmTrace` contains executing bytecode and same-instruction stack/memory/storage
deltas, recursively nested as `sub` for every entered child frame. A selected
`vmTrace` is never null: a frame in which no instruction ran, such as a call to an
account without code or to a precompile, is `{"code":"0x","ops":[]}`. Calls that
fail the depth or balance precondition enter no frame and keep `sub: null`.
The `cost` of call and create instructions includes the gas made available to the
child frame, excluding the value-transfer stipend, and `used` is the gas left after
the child's unused gas is returned.

Filter address matching is OR within each list and AND between lists, as `eth_getLogs`
composes topic positions. Omitted, null or empty lists are unrestricted. Optional
`mode` accepts `intersection` (the default) and `union`, which matches either populated
list; other values are invalid parameters. Creation recipients are successful created addresses;
selfdestruct uses the destroyed address and beneficiary; rewards have only a
recipient. Post-Merge blocks do not receive synthetic issuance rewards.
Omitted filter bounds both mean `latest`, resolved against one head; historical
searches must set `fromBlock`. As for `eth_getLogs`, a bound beyond the head, a
`pending` bound or a `fromBlock` above `toBlock` returns `-32602` and the range is
never clamped. The `earliest` tag selects the lowest block with available history,
as it does for `eth_*`; explicit block numbers below a node's history cutoff return
`4444`.
Calls and callMany default to latest; available safe and finalized tags select
their corresponding blocks, and EIP-1898 block hashes (optionally with
`requireCanonical`) select a block by hash. Optional state and block overrides use
the `eth_call`/`eth_simulateV1` objects and apply once before the first call: block
overrides replace fields of the selected block environment, and the zero-fee rule
then uses the overridden base fee. Invalid overrides return `-32602`.

## Limits and history

- Calls without `gas` use Geth's configured RPC gas cap. Calls and signed raw
  transactions whose gas exceeds it are rejected with `-38026`, never capped.
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
  Missing transaction hashes also return `4444` when lookup pruning prevents a
  definitive absence check; they return `null` with complete lookup history.
  The namespace does not enable archive retention or add an address index.

## Validation

Run `go test ./eth/tracers -run TestTraceNamespace` for execution, nested calls,
state rollback, signed authorization, cancellation, fork and wire-format tests.
The [interop project](https://github.com/banteg/trace-interop) builds this fork from
source and captures the same frozen Hive corpora as the other clients. Its reports
separate measured assertions, schema checks, and untested behavior; passing them
is not proof of complete conformance.
