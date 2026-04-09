// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pathdb

import (
	"bytes"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/holiman/uint256"
)

// buildTestBintrie constructs a small in-memory bintrie containing two
// accounts and one storage slot, persists its serialized nodes into the
// supplied key-value store under the standard pathdb account-trie key
// space (which is what the bintrie reads back via diskStore), and returns
// the resulting state root.
//
// This helper sidesteps triedb.Database to avoid an import cycle: pathdb
// is a child of triedb, so the test cannot construct a triedb.Database
// here. Instead it manually persists the nodes returned by
// bintrie.Commit, mirroring what writeNodes would do in production.
func buildTestBintrie(t *testing.T, db ethdb.Database) (common.Hash, []addrAcct) {
	t.Helper()

	// Use a memory-backed NodeDatabase for the empty starting trie. The
	// trie's nodeResolver returns nil for unknown hashes, which matches
	// the empty-trie semantics expected by NewBinaryTrie.
	tr, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, &diskStore{db: db})
	if err != nil {
		t.Fatalf("new bintrie: %v", err)
	}

	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000007")
	slotValue := bytes.Repeat([]byte{0x77}, 32)

	if err := tr.UpdateAccount(addr1, &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		CodeHash: types.EmptyCodeHash[:],
	}, 0); err != nil {
		t.Fatalf("update account 1: %v", err)
	}
	if err := tr.UpdateAccount(addr2, &types.StateAccount{
		Nonce:    2,
		Balance:  uint256.NewInt(200),
		CodeHash: types.EmptyCodeHash[:],
	}, 0); err != nil {
		t.Fatalf("update account 2: %v", err)
	}
	if err := tr.UpdateStorage(addr1, slot[:], slotValue); err != nil {
		t.Fatalf("update storage: %v", err)
	}
	root, nodes := tr.Commit(false)

	// Persist all collected nodes via the standard account-trie path
	// scheme accessor — the bintrie sits in the same key space as the
	// account trie because there are no per-account storage tries in
	// EIP-7864.
	batch := db.NewBatch()
	for path, node := range nodes.Nodes {
		if node.IsDeleted() {
			rawdb.DeleteAccountTrieNode(batch, []byte(path))
			continue
		}
		rawdb.WriteAccountTrieNode(batch, []byte(path), node.Blob)
	}
	if err := batch.Write(); err != nil {
		t.Fatalf("flush trie nodes: %v", err)
	}

	return root, []addrAcct{
		{addr: addr1, hasStorage: true, slot: slot, slotVal: slotValue},
		{addr: addr2, hasStorage: false},
	}
}

// addrAcct describes a test account so the assertions phase can re-derive
// the bintrie keys it should find on disk.
type addrAcct struct {
	addr       common.Address
	hasStorage bool
	slot       common.Hash
	slotVal    []byte
}

// runTestBintrieGenerator wires up a generator with the bintrie codec and
// drives generateBinTrieStems to completion. It returns the codec and the
// underlying db so the assertions can read back stem blobs.
func runTestBintrieGenerator(t *testing.T, db ethdb.Database, root common.Hash, marker []byte) {
	t.Helper()

	codec := newBintrieFlatCodec(db)
	gen := &generator{
		db:    db,
		codec: codec,
		stats: &generatorStats{start: time.Now()},
		abort: make(chan chan struct{}, 1),
		done:  make(chan struct{}),
	}
	ctx := newBintrieGeneratorContext(root, marker, db)
	defer ctx.close()

	if err := gen.generateBinTrieStems(ctx); err != nil {
		t.Fatalf("generateBinTrieStems: %v", err)
	}
	if err := ctx.batch.Write(); err != nil {
		t.Fatalf("final batch write: %v", err)
	}
}

// TestBintrieGeneratorRebuildsStems verifies the happy-path:
//   - Build a small bintrie with two accounts and one storage slot.
//   - Run the generator on its root.
//   - Read back the stem blobs and check every offset round-trips.
//
// This is the primary "the generator works" test.
func TestBintrieGeneratorRebuildsStems(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	root, accounts := buildTestBintrie(t, db)

	// Sanity-check that the bintrie isn't trivially empty.
	if root == (common.Hash{}) || root == types.EmptyBinaryHash {
		t.Fatal("test bintrie produced an empty root")
	}

	runTestBintrieGenerator(t, db, root, nil)

	// Each test account must have its BasicData (offset 0) and CodeHash
	// (offset 1) entries on disk after generation.
	for _, a := range accounts {
		stem := bintrie.GetBinaryTreeKeyBasicData(a.addr)[:bintrie.StemSize]
		blob := rawdb.ReadBinTrieStem(db, stem)
		if len(blob) == 0 {
			t.Errorf("addr %x: stem blob missing after generation", a.addr)
			continue
		}
		basic, err := extractStemOffset(blob, bintrie.BasicDataLeafKey)
		if err != nil || len(basic) != 32 {
			t.Errorf("addr %x: BasicData missing/invalid (err=%v len=%d)", a.addr, err, len(basic))
		}
		codeHash, err := extractStemOffset(blob, bintrie.CodeHashLeafKey)
		if err != nil || !bytes.Equal(codeHash, types.EmptyCodeHash[:]) {
			t.Errorf("addr %x: CodeHash mismatch (err=%v got=%x)", a.addr, err, codeHash)
		}
	}

	// The storage slot must be present at its derived stem (which may
	// equal the account's BasicData stem for header slots, or differ for
	// out-of-header slots — slot 7 is in-header so we expect the same
	// stem as BasicData).
	a := accounts[0]
	storageKey := bintrie.GetBinaryTreeKeyStorageSlot(a.addr, a.slot[:])
	storageBlob := rawdb.ReadBinTrieStem(db, storageKey[:bintrie.StemSize])
	if len(storageBlob) == 0 {
		t.Fatal("storage stem blob missing")
	}
	got, err := extractStemOffset(storageBlob, storageKey[bintrie.StemSize])
	if err != nil {
		t.Fatalf("extract storage offset: %v", err)
	}
	if !bytes.Equal(got, a.slotVal) {
		t.Errorf("storage value mismatch: got %x want %x", got, a.slotVal)
	}
}

// TestBintrieGeneratorResumeStemBoundary verifies that a generator
// started from a stem-boundary marker (stem || offset 0) correctly
// generates only the stems at or after the marker.
func TestBintrieGeneratorResumeStemBoundary(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	root, accounts := buildTestBintrie(t, db)

	stem1 := bintrie.GetBinaryTreeKeyBasicData(accounts[0].addr)[:bintrie.StemSize]
	stem2 := bintrie.GetBinaryTreeKeyBasicData(accounts[1].addr)[:bintrie.StemSize]
	larger := stem1
	smaller := stem2
	if bytes.Compare(stem1, stem2) < 0 {
		larger, smaller = stem2, stem1
	}

	marker := make([]byte, 32)
	copy(marker, larger)

	runTestBintrieGenerator(t, db, root, marker)

	if got := rawdb.ReadBinTrieStem(db, smaller); len(got) != 0 {
		t.Errorf("smaller stem should have been skipped by resume marker, got %x", got)
	}
	if got := rawdb.ReadBinTrieStem(db, larger); len(got) == 0 {
		t.Errorf("larger stem should have been generated after resume marker")
	}
}

// TestBintrieGeneratorResumeMidStem is the regression test for review
// finding C1 (mid-stem resume drops earlier offsets). Before A3's fix,
// flushStem OVERWROTE the on-disk stem blob with only the offsets
// accumulated after the resume point. Offsets from a prior pass that
// were already on disk were silently lost.
//
// The test simulates a two-pass generation:
//
//  1. Pre-seed the disk with a stem blob containing offsets 0 and 1
//     (simulating what a prior pass wrote before being interrupted).
//  2. Run the generator with marker = stem||1 (resume INSIDE the stem,
//     past offset 0).
//  3. After the generator completes, verify that the on-disk blob
//     contains ALL offsets (0, 1, and everything else the trie has)
//     — not just the offsets from the resumed walk.
//
// Before A3: step 3 would show only the post-marker offsets.
func TestBintrieGeneratorResumeMidStem(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	root, accounts := buildTestBintrie(t, db)

	// Pick addr1 (the one with storage). It has BasicData (offset 0),
	// CodeHash (offset 1), and storage slot 7 at offset 64+7=71.
	a := accounts[0]
	stem := bintrie.GetBinaryTreeKeyBasicData(a.addr)[:bintrie.StemSize]

	// Step 1: Pre-seed the disk with a partial stem blob containing
	// only offsets 0 and 1 — as if a prior generator pass wrote them
	// before being interrupted.
	preSeed := newStemBuilder()
	preSeed.set(bintrie.BasicDataLeafKey, bytes.Repeat([]byte{0xAA}, 32))
	preSeed.set(bintrie.CodeHashLeafKey, bytes.Repeat([]byte{0xBB}, 32))
	rawdb.WriteBinTrieStem(db, stem, preSeed.encode())

	// Step 2: Resume from offset 1 — the generator should pick up at
	// offset 1 of this stem and walk forward. The builder will
	// accumulate only offset 1 + storage offset from the trie walk.
	// The RMW in flushStem must merge them with the pre-seeded disk
	// blob to preserve offset 0.
	marker := make([]byte, 32)
	copy(marker[:bintrie.StemSize], stem)
	marker[bintrie.StemSize] = bintrie.CodeHashLeafKey // resume at offset 1

	runTestBintrieGenerator(t, db, root, marker)

	// Step 3: After the full run, verify the disk blob has ALL offsets.
	blob := rawdb.ReadBinTrieStem(db, stem)
	if len(blob) == 0 {
		t.Fatal("stem blob missing after mid-stem resume")
	}

	// Offset 0 (BasicData): must survive the mid-stem resume because
	// the RMW merged the builder's new content with the existing disk
	// blob. Before A3, this offset was silently dropped.
	basic, err := extractStemOffset(blob, bintrie.BasicDataLeafKey)
	if err != nil {
		t.Fatalf("extract BasicData: %v", err)
	}
	if len(basic) != 32 {
		t.Fatalf("BasicData lost after mid-stem resume (A3 regression): got len=%d, want 32", len(basic))
	}

	// Offset 1 (CodeHash): the generator walked this offset (it's at
	// the marker), so the trie's authoritative value should overwrite
	// the pre-seeded one.
	code, err := extractStemOffset(blob, bintrie.CodeHashLeafKey)
	if err != nil {
		t.Fatalf("extract CodeHash: %v", err)
	}
	if len(code) != 32 {
		t.Fatalf("CodeHash missing after resume: got len=%d", len(code))
	}

	// Storage slot must also be present (the generator walked it as
	// part of the full stem traversal).
	storageKey := bintrie.GetBinaryTreeKeyStorageSlot(a.addr, a.slot[:])
	storageOffset := storageKey[bintrie.StemSize]
	storageStem := storageKey[:bintrie.StemSize]
	if bytes.Equal(storageStem, stem) {
		// Storage is at the same stem (header slot) — verify it's in the blob.
		storageVal, err := extractStemOffset(blob, storageOffset)
		if err != nil {
			t.Fatalf("extract storage: %v", err)
		}
		if !bytes.Equal(storageVal, a.slotVal) {
			t.Errorf("storage value: got %x, want %x", storageVal, a.slotVal)
		}
	}
}
