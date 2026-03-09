# NOMT Binary Merkle Trie on PebbleDB — Design Document

## 1. Overview

NOMT (Nearly-Optimal Merkle Trie) is a page-based binary merkle trie engine
integrated into geth as an alternative to the Merkle Patricia Trie (MPT). It
stores trie nodes in fixed-size 4KB pages optimized for SSD I/O, and uses
parallel batch updates for high throughput during block execution.

This implementation stores all data — trie pages, flat account/storage state,
and stem values — in geth's existing PebbleDB instance under dedicated key
prefixes. There is no custom storage engine; PebbleDB's LSM-tree, atomic
batches, and bloom filters handle persistence and crash safety.

```
+-----------------------------------------------------------------------+
|                        Ethereum State Layer                           |
|  (StateDB: accounts, storage slots, contract code)                   |
+----------------------------------+------------------------------------+
                                   |
                        +----------v-----------+
                        |      NomtTrie        |
                        |  (state.Trie impl)   |
                        |  trie/nomttrie/      |
                        +----------+-----------+
                                   |
              +--------------------+--------------------+
              |                                         |
   +----------v-----------+               +-------------v-----------+
   |   Canonical Root     |               |    Page Tree Engine     |
   |  BuildInternalTree   |               |    nomt/merkle/         |
   |  (248-bit tree over  |               |  (PageWalker + workers) |
   |   all stem hashes)   |               +-------------+-----------+
   +----------------------+                             |
                                          +-------------v-----------+
                                          |       nomt/db/          |
                                          |  (PebbleDB page store)  |
                                          +-------------+-----------+
                                                        |
              +--------------------+--------------------+
              |                    |                    |
   +----------v------+  +---------v--------+  +--------v---------+
   | Flat State       |  | Stem Values      |  | Trie Pages       |
   | prefix 0x01-0x02 |  | prefix 0x03      |  | prefix 0x04      |
   | (accts, storage) |  | (per-slot values)|  | (4KB RawPage)    |
   +------------------+  +------------------+  +------------------+
              |                    |                    |
              +--------------------+--------------------+
                                   |
                        +----------v-----------+
                        |      PebbleDB        |
                        |   (single instance)  |
                        +----------------------+
```

### Design Principles

1. **Single database**: All NOMT data lives in geth's PebbleDB — no custom
   hash table, no WAL, no separate files.
2. **Page-level granularity**: The merkle engine operates on 4KB page blobs,
   not individual 32-byte nodes. PebbleDB stores each page as a single KV pair.
3. **EIP-7864 compatibility**: Key derivation, stem hashing, and root
   computation produce roots identical to geth's `trie/bintrie/`.
4. **Unchanged merkle engine**: The `nomt/merkle/` package (PageWalker,
   parallel workers) has zero dependency on the storage backend — it accesses
   pages through the `PageSet` interface.

---

## 2. PebbleDB Key Schema

All NOMT data shares the same PebbleDB instance as geth's other subsystems.
Each NOMT key type uses a distinct single-byte prefix:

```
PebbleDB Key Layout
====================

Prefix  Key Format                              Value           Description
------  ---------------------------------------- --------------- ---------------------------
0x01    0x01 || accountHash[32]                  RLP(SlimAcct)   Account flat state
0x02    0x02 || accountHash[32] || slotHash[32]  raw bytes       Storage flat state
0x03    0x03 || stem[31] || suffix[1]            value[32]       Stem value slot
0x04    0x04 || PageID.Encode()[32]              RawPage[4032]   Trie page blob
0x05    0x05 || "root"                           Node[32]        Page tree root hash
```

Key properties:

- **0x01–0x02** (flat state): Used by `triedb/nomtdb/` for geth's standard
  `StateReader` interface. Accounts are RLP-encoded `SlimAccount` structs.
- **0x03** (stem values): The 256 value slots per stem node. Each slot stores
  a 32-byte value (account basic data, code hash, storage value, or code
  chunk). Key = 33 bytes, value = 32 bytes.
- **0x04** (trie pages): The binary merkle page tree. Key = 33 bytes (prefix +
  PageID encoding). Value = 4032 bytes (page contents; the trailing PageID and
  metadata within the 4096-byte `RawPage` are included).
- **0x05** (metadata): Currently stores only the page tree root hash,
  persisted atomically with page updates. Enables root recovery on restart.

---

## 3. Binary Merkle Trie Structure

### 3.1. Two-Layer Tree

The trie has two logical layers matching EIP-7864:

```
                        Root (canonical)
                          |
         BuildInternalTree over 248-bit stem paths
                          |
     +--------+-----------+-----------+--------+
     |        |           |           |        |
   stem_0  stem_1  ... stem_k  ... stem_n   (depth 248)
     |        |           |           |
  [256 slots each: SHA256 sub-tree of values]
```

**Internal tree (depth 0–247)**: Binary SHA256 tree where each "leaf" is an
opaque 32-byte stem hash. Navigated by bits 0–247 of the stem path. This is
what `BuildInternalTree(skip=0)` computes — its root is the canonical state
root returned by `Hash()`.

**Stem nodes (depth 248)**: Each stem holds 256 value slots indexed by the
last byte (suffix). The stem hash is computed as:

```
SHA256(stem_path[31] || 0x00 || subtree_root)

where subtree_root = 8-level binary SHA256 tree over SHA256(value_i) for i in 0..255
```

### 3.2. Page Tree (Persistent Storage)

For persistent storage, the 248-bit internal tree is partitioned into a tree
of 4KB pages. Each page stores a rootless sub-binary-tree of depth 6:

```
Page Tree Organization
======================

             Root Page (depth 0)           <- 1 page, 126 nodes
            /         |         \
     Child 0    ...  Child k  ... Child 63  <- up to 64 child pages
      /    \                      /    \
   ...      ...                ...      ... <- up to 64^2 pages at depth 2
                                                (max depth 42: 6*42=252 bits)

Each page:
+-----------------------------------------------+
| 126 internal nodes (levels 1-6), 32 bytes each |  4032 bytes
|                                                 |
|  Level 1:   2 nodes (left/right of root)       |
|  Level 2:   4 nodes                            |
|  Level 3:   8 nodes                            |
|  Level 4:  16 nodes                            |
|  Level 5:  32 nodes                            |
|  Level 6:  64 nodes (bottom layer)             |
|                                                 |
| The root of this sub-tree lives in the         |
| parent page's bottom layer (level 6).          |
+-------------------------------------------------+
| 24 bytes padding                                |
+-------------------------------------------------+
| ElidedChildren: 8-byte bitfield (uint64 LE)     |  Which of the 64
|   bit i = 1 means child page i is elided        |  children are stored
+-------------------------------------------------+  inline (not on disk)
| PageID: 32-byte encoded identifier              |
+-------------------------------------------------+
Total: 4096 bytes (SSD page aligned)
```

**Page elision**: If a subtree has few leaves, its page is not stored on disk.
Instead, the sub-tree data lives inline in the parent page's bottom-layer
nodes. The `ElidedChildren` bitfield tracks which of the 64 child slots are
elided. This avoids storing nearly-empty pages for sparse trie regions.

### 3.3. PageID Encoding

A `PageID` is a path through the page tree — a sequence of child indices
(each 0–63). The encoding produces a unique 32-byte key for PebbleDB:

```
PageID Encoding (shift-then-add)
=================================

For path [c_0, c_1, ..., c_n]:
  value = 0
  for each c_i:
    value = (value << 6) + (c_i + 1)

Store as big-endian 32 bytes.

Examples:
  Root page:  path=[]         -> 0x00...00
  Child 0:    path=[0]        -> 0x00...01
  Child 63:   path=[63]       -> 0x00...40  (64 decimal)
  [5, 10]:    path=[5,10]     -> 0x00...016B  ((6<<6)+11 = 395)

Properties:
  - Root encodes to all zeros
  - Lexicographic ordering: parent < children < right siblings
  - Unique: no two distinct paths produce the same encoding
  - Max depth 42 (6*42 = 252 bits, fits in 256-bit key)
```

---

## 4. Hash Functions

All hashing uses **SHA256** (EIP-7864). There is no MSB tagging — nodes are
either all-zero (terminator) or opaque 32-byte hashes.

```
Internal node:  SHA256(left[32] || right[32])
                Both children are 32-byte nodes.

Stem node:      SHA256(stem_path[31] || 0x00 || subtree_root[32])
                subtree_root = 8-level binary SHA256 tree over
                               SHA256(value_i) for i in 0..255
                (zero-hash pairs produce zero parent, pruning empty branches)

Terminator:     0x00...00 (32 zero bytes)
                Represents an empty sub-trie at any position.
```

SHA256 hashers are pooled via `sync.Pool` to avoid allocation pressure during
batch hashing.

---

## 5. Update Pipeline

### 5.1. Per-Block Flow

```
Block Execution
===============

1. StateDB accumulates changes
   (UpdateAccount, UpdateStorage, UpdateContractCode)
           |
           v
2. NomtTrie.pending collects stemUpdates
   Each update = (stem[31], suffix[1], value[32])
           |
           v
3. NomtTrie.Hash() triggers flush:
           |
           +---> groupAndHashStems()
           |       |
           |       +-- Group by stem path (stable sort)
           |       +-- For each stem:
           |       |     Load existing values from PebbleDB (prefix 0x03)
           |       |     Merge new values, compute SHA256 stem hash
           |       |     Write updated values back (batch)
           |       +-- Return sorted []StemKeyValue
           |
           +---> mergeStemKVs()
           |       Merge new stems into allStems (sorted in-place)
           |       Fast path: in-place update when no new stems added
           |
           +---> db.Update(stemKVs)
           |       Run PageWalker on page tree
           |       Persist updated pages to PebbleDB (prefix 0x04)
           |       Persist new page tree root (prefix 0x05)
           |       All writes in single atomic batch
           |
           +---> canonicalRoot()
                   BuildInternalTree(skip=0, allStems)
                   Returns 32-byte root matching bintrie exactly
```

### 5.2. Page Update Engine (nomt/merkle/)

The PageWalker processes sorted stem updates left-to-right through the page
tree. It loads pages from PebbleDB via the `PageSet` interface, modifies
nodes in place, and emits a list of `UpdatedPage` entries.

```
PageWalker Algorithm
====================

Input:  sorted [(stem_path, stem_hash)] + current page tree root
Output: new root + list of UpdatedPage entries

For each stem update:
  1. Descend through page stack to the target position
     (load pages from PebbleDB or create fresh ones)

  2. Place the stem hash at the target node position

  3. Hash upward through the page tree:
     - Compute SHA256(left || right) for modified internal nodes
     - Compact terminator pairs (both children zero -> parent zero)
     - Stop when remaining nodes will be affected by future updates

After all updates:
  4. Hash remaining nodes up to the root
  5. Return new root + all modified pages

Page Stack (in-memory during walk):
  +--------+--------+--------+
  | Root   | Child  | Grand- |  ...up to 42 pages deep
  | Page   | Page   | child  |
  +--------+--------+--------+
  depth 0    depth 1  depth 2
```

### 5.3. Parallel Workers

For batches with 64+ updates, the page tree is partitioned by root page child
index (first 6 bits of each stem path = 64 possible buckets). Independent
subtrees are processed concurrently:

```
Parallel Update (depth-7 split)
================================

                    Root Page
                   /    |    \
           child 0  child k  child 63    (64 slots)
              |        |        |
          +---+---+ +--+--+ +--+---+
          |Worker1| |W...k| |Worker|    N goroutines
          +---+---+ +--+--+ +--+---+
              |        |        |
        [pages]   [pages]   [pages]     each worker's UpdatedPages
              \        |       /
               +-------+------+
                       |
              Merge child roots into
              root page, persist all
              pages in atomic batch
```

Each worker gets an independent `PageSet` (via `pageSetFactory`) to avoid
contention. After workers complete, their child-page roots are merged into
the root page.

---

## 6. PebbleDB Page Storage (nomt/db/)

The `pebblePageSet` implements `merkle.PageSet` backed by PebbleDB:

```
pebblePageSet
=============

                     PageWalker
                         |
                    PageSet.Get(id)
                         |
            +------------+------------+
            |                         |
     cache[encoded_id]?           diskdb.Get(0x04||id)
       /          \                     |
    hit: copy      miss         +-------+-------+
    & return                    |               |
                             found:          not found:
                             copy to cache,  return fresh
                             return copy     zeroed page

IMPORTANT: Always return a COPY of cached pages.
The PageWalker mutates pages in place during updates.
A shared reference would corrupt the cache.
```

Page persistence uses PebbleDB's atomic batch writes:

```
Atomic Batch Write
==================

batch := diskdb.NewBatch()

for each UpdatedPage:
  if page was cleared:
    batch.Delete(0x04 || PageID.Encode())
  else:
    batch.Put(0x04 || PageID.Encode(), page[0:4096])

batch.Put(0x05 || "root", new_root[0:32])   // persist root atomically
batch.Write()                                // single atomic operation

No WAL needed — PebbleDB guarantees atomic batch writes.
If crash before Write(): no pages or root updated (safe).
If crash after Write():  all pages and root updated (consistent).
```

---

## 7. EIP-7864 Key Derivation

Key derivation delegates to `trie/bintrie/` to guarantee identical key
generation. The 32-byte key is split into a 31-byte stem and 1-byte suffix:

```
EIP-7864 Key Layout
====================

|<--------- stem (31 bytes, 248 bits) --------->|<- suffix (1 byte) ->|

Internal tree navigates bits 0-247 (stem path).
Stem node holds 256 value slots indexed by suffix (0-255).

Account Keys:
  key = SHA256(SHA256(address) || base_offset)
  stem = key[0:31]
  BasicData:  suffix = 0   (nonce at [8:16], balance at [16:32])
  CodeHash:   suffix = 1   (32-byte code hash)

Storage Keys:
  key = SHA256(SHA256(address) || storage_offset)
  stem = key[0:31], suffix = key[31]
  storage_offset encodes slot position within 256-slot groups

Code Chunk Keys:
  chunks = ChunkifyCode(bytecode)  (31-byte chunks, right-padded)
  For chunk number N:
    groupOffset = (N + 128) % 256
    if groupOffset == 0 or N == 0:
      offset[24:32] = uint64_le(N + 128)
      key = SHA256(SHA256(address) || offset)
      stem = key[0:31]
    suffix = groupOffset
```

---

## 8. Canonical Root vs. Page Tree Root

The system computes two related but distinct roots:

```
Root Computation
================

1. Canonical Root (returned by Hash()):
   BuildInternalTree(skip=0, allStems)
   - Pure computation over sorted (stem, hash) pairs
   - 248-bit binary tree, no page structure
   - Identical to bintrie's root for the same state
   - This is the state root in block headers

2. Page Tree Root (persisted in PebbleDB):
   merkle.ParallelUpdate(root, stemKVs, workers, pageSetFactory)
   - Partitioned into 4KB pages at 6-bit boundaries
   - Workers split at depth 7, adding SHA256(hash||zeros) wrapping
   - Root may differ from canonical due to wrapping levels
   - Used for persistent page storage and incremental updates

The page tree root is an implementation detail. Only the canonical
root (from BuildInternalTree) is externally visible.
```

---

## 9. Package Structure

```
go-ethereum/
  nomt/
    core/                        Pure data structures, no I/O
      node.go                    Node type, Terminator, NodeKind
      hasher.go                  SHA256 hashing (pooled), HashInternal, HashStem
      page.go                    RawPage [4096]byte, level-order node access
      pageid.go                  PageID encode/decode, child/parent navigation
      pagediff.go                126-bit change tracking bitfield
      triepos.go                 TriePosition: depth tracking, page boundary detection
      update.go                  StemKeyValue, BuildInternalTree, StemSharedBits

    merkle/                      Page-based update engine, storage-agnostic
      pageset.go                 PageSet interface, MemoryPageSet
      pagewalker.go              Left-to-right batch trie updates
      worker.go                  Parallel workers (partitioned at depth 7)
      elided.go                  ElidedChildren 64-bit bitfield

    db/                          PebbleDB integration layer
      db.go                      DB struct, pebblePageSet, atomic batch writes
                                 Key prefixes: 0x04 (pages), 0x05 (metadata)

  trie/
    nomttrie/                    state.Trie implementation
      trie.go                    NomtTrie: UpdateAccount/Storage/Code, Hash, Commit
      key_encoding.go            EIP-7864 key derivation (delegates to bintrie)
      stem.go                    Stem value storage, groupAndHashStems

  triedb/
    nomtdb/                      triedb backend
      config.go                  Config (NumWorkers)
      database.go                Database: NodeReader, StateReader, DiskDB, NomtDB
      reader.go                  Flat state readers, key prefix constants (0x01, 0x02)
```

---

## 10. Data Flow Diagram

Complete flow for a single block's state update:

```
+-----------+    UpdateAccount(addr, acc)     +------------+
| StateDB   | -----------------------------> | NomtTrie   |
|           |    UpdateStorage(addr, k, v)    |            |
|           | -----------------------------> | pending:   |
|           |    UpdateContractCode(addr, c)  | [{stem,    |
|           | -----------------------------> |   suffix,  |
+-----------+                                 |   value}]  |
                                              +-----+------+
                                                    |
                                              Hash() called
                                                    |
                                              +-----v------+
                                              | groupAnd   |
                                              | HashStems  |
                                              +-----+------+
                                                    |
                          +-------------------------+-------------------------+
                          |                                                   |
                    +-----v------+                                      +-----v------+
                    | Load stem  |                                      | Compute    |
                    | values     |                                      | stem hash  |
                    | from 0x03  |                                      | SHA256     |
                    | prefix     |                                      | sub-tree   |
                    +-----+------+                                      +-----+------+
                          |                                                   |
                    +-----v------+                                            |
                    | Merge new  |                                            |
                    | values     |                                            |
                    +-----+------+                                            |
                          |                                                   |
                    +-----v------+                                            |
                    | Write back |                                            |
                    | to 0x03    |                                            |
                    | (batch)    |                                            |
                    +------------+                                            |
                                                                              |
                          +---------------------------------------------------+
                          |
                    +-----v----------+
                    | []StemKeyValue |   sorted by stem path
                    +-----+----------+
                          |
              +-----------+-----------+
              |                       |
        +-----v------+         +-----v------+
        | Merge into |         | db.Update  |
        | allStems   |         +-----+------+
        | (sorted)   |               |
        +-----+------+         +-----v-----------+
              |                 | ParallelUpdate  |
              |                 | (PageWalker x N)|
              |                 +-----+-----------+
              |                       |
              |                 +-----v------+
              |                 | PebbleDB   |
              |                 | batch:     |
              |                 | put pages  |
              |                 | put root   |
              |                 +------------+
              |
        +-----v-----------------+
        | BuildInternalTree     |
        | skip=0, allStems      |
        +-----+-----------------+
              |
        +-----v------+
        | Canonical  |   <-- returned by Hash()
        | state root |       matches bintrie exactly
        +------------+
```

---

## 11. Crash Safety

All persistent state changes are made through PebbleDB's atomic batch writes:

```
Crash Safety Guarantees
========================

State changes happen in two atomic batches per Hash() call:

Batch 1 (stem values):
  Write updated stem values to prefix 0x03
  -> If crash here: stem values partially updated, but page tree
     and canonical root unchanged. Next Hash() will recompute
     stem hashes from flat state, producing correct result.

Batch 2 (page tree):
  Write updated pages to prefix 0x04
  Write new page tree root to prefix 0x05
  -> Atomic: either all pages + root update, or none do.
  -> If crash before: pages unchanged, root unchanged.
     Next block re-applies the same page updates.
  -> If crash after: consistent state, root matches pages.

Recovery on startup:
  1. Read root from PebbleDB (prefix 0x05)
  2. If found and valid (32 bytes): use as current root
  3. If not found: fresh database, root = Terminator
  No WAL replay, no file scanning, no repair needed.
```

---

## 12. Performance Characteristics

```
Operation Costs
================

Read account:       1 PebbleDB point lookup (prefix 0x03, stem slot 0)
                    + 1 PebbleDB point lookup (stem slot 1 for code hash)
                    Bloom filter makes misses fast.

Read storage:       1 PebbleDB point lookup (prefix 0x03)

Write account:      2 pending stemUpdates (basic data + code hash)
                    Deferred to Hash() — no I/O during execution.

Hash() flush:       O(S) prefix iterations for S dirty stems (load values)
                    O(S * log(S)) sort
                    O(S * 256) SHA256 hashing (stem sub-trees)
                    O(P) page reads/writes for P affected pages
                    2 PebbleDB atomic batch writes

Parallelism:        Page tree updates partitioned across N workers
                    (default: runtime.NumCPU())
                    Workers share no state — each has own PageSet.
                    Effective for 64+ stems per block.

Memory:             4KB per cached page (pebblePageSet, per-worker)
                    ~64 bytes per tracked stem (allStems slice)
                    SHA256 hasher pool (sync.Pool, reused across calls)
```

---

## 13. Cross-Validation

The implementation is validated against geth's `trie/bintrie/` which
independently implements EIP-7864. Both produce identical state roots:

```
Cross-Validation Test Matrix
==============================

Test                          Accounts   Contracts   Slots    Distributions
---------------------------   --------   ---------   ------   -------------
TestRootEquality/Small        100        50          1-20     PowerLaw
TestRootEquality/Medium       1,000      500         1-100    PowerLaw
TestRootEquality/Large        10,000     5,000       1-500    PowerLaw
TestDistributionVariants      100        50          1-20     PowerLaw,Uniform,Exp
TestIncrementalRootEquality   20         10          1-5      Uniform (per-op)
TestDeterminism               100        50          1-20     PowerLaw (2x same seed)

All tests verify: bintrie_root == nomt_root at every block boundary.
Race detector enabled on all test runs.
```
