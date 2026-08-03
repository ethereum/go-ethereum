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

## Also deferred, for context

These are known and tracked elsewhere; listed so this file is the single place
to look.

- **The code zone (`0x01`) has no root-verified coverage.** Counting the leading
  zone byte of every hashed entry in `trie/bintrie/testdata/eip8297_vectors.json`
  gives 601 for accounts (`0x00`) and 266 for storage (`0xFF`), and **zero** for
  content-addressed overflow code. It appears only under `embedding_vectors.chunks`,
  which checks key derivation and never a root. Closing this needs a re-export
  from the reference implementation via `testdata/export_vectors.py`, so it
  belongs to the EEST/EELS integration phase.
- **Two harness blockers before EEST fixtures can run.** `execBlockTest`
  (`tests/block_test.go`) runs every fixture under both the hash and path
  schemes, and the binary tree hard-fails on anything but path; and it always
  requests witness building, which the tree now refuses. Both are test-harness
  changes, not implementation ones.
- **`TestT8n`** fails on the binary tree fixtures because the prestate is
  reopened with an already-committed trie. Out of scope by instruction.
- **`StackBuilder`** (`trie/bintrie/stackbuilder.go`) has no production caller.
  Its natural consumer is the offline conversion in `cmd/geth`, which still
  inserts one stem at a time. Revisit when conversion is benchmarked; delete if
  still unwired.
