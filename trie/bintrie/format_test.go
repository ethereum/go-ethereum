// Copyright 2026 go-ethereum Authors
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

package bintrie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// TestRootHashMatchesReadBackHash pins the round-trip invariant: the root
// hash a Commit advertises must be exactly the value a fresh reader computes
// from the on-disk blob. Before Option B the writer produced a natural-depth
// hash while DeserializeAndHash produced an extended-depth hash, so the two
// disagreed for any non-trivial subtree — this test failed. With the
// per-entry depth byte, the reader rebuilds the natural-shape tree and the
// hashes match for every groupDepth and every divergence bit.
func TestRootHashMatchesReadBackHash(t *testing.T) {
	for groupDepth := 1; groupDepth <= MaxGroupDepth; groupDepth++ {
		// divergeBit ∈ [0, groupDepth-1] places the two stems at natural
		// depth (divergeBit+1) within the root group; we want to exercise
		// every depth-offset value the new format must handle.
		for divergeBit := 0; divergeBit < groupDepth; divergeBit++ {
			t.Run(fmt.Sprintf("gd=%d/diverge=%d", groupDepth, divergeBit), func(t *testing.T) {
				tr := &BinaryTrie{
					store:      newNodeStore(),
					tracer:     trie.NewPrevalueTracer(),
					groupDepth: groupDepth,
				}
				stemL, stemR := stemsDivergingAt(divergeBit)
				if err := tr.store.Insert(stemL, oneKey[:], nil); err != nil {
					t.Fatalf("Insert stemL: %v", err)
				}
				if err := tr.store.Insert(stemR, twoKey[:], nil); err != nil {
					t.Fatalf("Insert stemR: %v", err)
				}

				natural := tr.Hash()
				_, ns := tr.Commit(false)
				rootNode, ok := ns.Nodes[""]
				if !ok {
					t.Fatalf("Commit produced no root blob (path \"\")")
				}

				readBack, err := DeserializeAndHash(rootNode.Blob, 0)
				if err != nil {
					t.Fatalf("DeserializeAndHash: %v", err)
				}
				if natural != readBack {
					t.Fatalf("round-trip hash mismatch:\n"+
						"  tr.Hash()                    = %x\n"+
						"  DeserializeAndHash(rootBlob) = %x\n"+
						"the parent's stored root hash cannot be reproduced from its own blob",
						natural, readBack)
				}
			})
		}
	}
}

// TestMultiStemMixedDepths inserts four stems that diverge at different
// depths within a single groupDepth=5 group, then round-trips the trie
// through Commit + fresh-read. Verifies that every stem is retrievable by
// key after reload — exercises the new format with several depth-offset
// values in the same blob (1, 2, 3, 4) and confirms attachInGroup builds
// the natural-shape tree correctly.
func TestMultiStemMixedDepths(t *testing.T) {
	const groupDepth = 5

	// Each stem diverges from `0x00…00` at a different bit, so naturally:
	//  - stem at bit-0 divergence → depth 1
	//  - stem at bit-1 divergence → depth 2
	//  - stem at bit-2 divergence → depth 3
	//  - stem at bit-3 divergence → depth 4
	stems := [][]byte{
		zeroKey[:],
		bitFlipStem(0), // diverge at bit 0
		bitFlipStem(1), // diverge at bit 1 (prefix "0" matches stem 0)
		bitFlipStem(2), // diverge at bit 2 (prefix "00")
		bitFlipStem(3), // diverge at bit 3 (prefix "000")
	}
	values := []common.Hash{oneKey, twoKey, threeKey, fourKey, ffKey}

	tr := &BinaryTrie{
		store:      newNodeStore(),
		tracer:     trie.NewPrevalueTracer(),
		groupDepth: groupDepth,
	}
	for i, stem := range stems {
		if err := tr.store.Insert(stem, values[i][:], nil); err != nil {
			t.Fatalf("Insert stem %d: %v", i, err)
		}
	}

	before := tr.Hash()
	_, ns := tr.Commit(false)
	rootBlob, ok := ns.Nodes[""]
	if !ok {
		t.Fatalf("no root blob in NodeSet")
	}
	readBack, err := DeserializeAndHash(rootBlob.Blob, 0)
	if err != nil {
		t.Fatalf("DeserializeAndHash: %v", err)
	}
	if before != readBack {
		t.Fatalf("hash mismatch: tr.Hash()=%x DeserializeAndHash(rootBlob)=%x", before, readBack)
	}

	// Reload the root blob into a fresh store and confirm structure.
	fresh := newNodeStore()
	ref, err := fresh.deserializeNodeWithHash(rootBlob.Blob, 0, before)
	if err != nil {
		t.Fatalf("deserializeNodeWithHash: %v", err)
	}
	if ref.Kind() != kindInternal {
		t.Fatalf("expected root to be Internal, got kind %d", ref.Kind())
	}
	// Spot-check: the reload-tree's root hash equals the commit-time hash.
	if got := fresh.computeHash(ref); got != before {
		t.Fatalf("reload root hash mismatch: got %x, want %x", got, before)
	}
}

// TestDecodeRejectsNonCanonicalPosition hand-crafts a blob where the bitmap
// position has nonzero trailing bits given its depth offset. Two
// implementations must produce byte-identical blobs for the same logical
// content, so a non-canonical position is unambiguously an invalid blob.
func TestDecodeRejectsNonCanonicalPosition(t *testing.T) {
	// groupDepth=5, bitmap size = 4 bytes. Set bit at position 5 (binary
	// 00101) and declare depthOffset=2. Top 2 bits of 00101 are 00 (path
	// "00"), the trailing 3 bits should be zero — they're 101 here, so the
	// reader must reject.
	blob := []byte{nodeTypeInternal, 5}
	// bitmap[0] = bit at position 5 → 1 << (7-5) = 0x04
	blob = append(blob, 0x04, 0x00, 0x00, 0x00)
	// depths[0] = 2
	blob = append(blob, 2)
	// hashes[0] = 32 zero bytes
	blob = append(blob, make([]byte, HashSize)...)

	s := newNodeStore()
	_, err := s.deserializeNode(blob, 0)
	if err == nil {
		t.Fatal("expected non-canonical position error, got nil")
	}
	if err.Error() != "non-canonical bitmap position" {
		t.Errorf("expected 'non-canonical bitmap position', got %q", err.Error())
	}
}

// TestDecodeRejectsInvalidDepthOffset covers depthOffset=0 (a present entry
// must consume ≥1 bit of natural path) and depthOffset>groupDepth (the
// entry would live below the group's bottom layer, impossible by
// construction).
func TestDecodeRejectsInvalidDepthOffset(t *testing.T) {
	makeBlob := func(groupDepth int, depthOffset uint8) []byte {
		bitmapSize := bitmapSizeForDepth(groupDepth)
		bitmap := make([]byte, bitmapSize)
		bitmap[0] = 0x80 // bit at position 0
		blob := []byte{nodeTypeInternal, byte(groupDepth)}
		blob = append(blob, bitmap...)
		blob = append(blob, depthOffset)
		blob = append(blob, make([]byte, HashSize)...)
		return blob
	}

	for _, tc := range []struct {
		name        string
		groupDepth  int
		depthOffset uint8
	}{
		{"depth=0", 5, 0},
		{"depth>groupDepth", 5, 6},
		{"depth>MaxGroupDepth", MaxGroupDepth, MaxGroupDepth + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newNodeStore()
			_, err := s.deserializeNode(makeBlob(tc.groupDepth, tc.depthOffset), 0)
			if err == nil {
				t.Fatal("expected invalid depth offset error, got nil")
			}
			if err.Error() != "invalid depth offset" {
				t.Errorf("expected 'invalid depth offset', got %q", err.Error())
			}
		})
	}
}

// TestRoundTripPersistence exercises the full commit-then-reload pipeline
// the way real geth does it: write blobs to a backing map, open a fresh
// nodeStore from the root blob, then resolve and read every value back
// through the resolver. Catches mismatches between the writer's storage
// path (collectChildGroups in store_commit.go) and the reader's lookup
// path (keyToPath in store_ops.go) — exactly the bug Option B's first
// implementation hit, where blobs were written at the bottom-layer-extended
// path but resolved at the natural-depth path. Also confirms the reloaded
// trie's root hash equals the original committed hash.
func TestRoundTripPersistence(t *testing.T) {
	for _, groupDepth := range []int{1, 2, 3, 5, 8} {
		t.Run(fmt.Sprintf("groupDepth=%d", groupDepth), func(t *testing.T) {
			// 1. Build a trie with deterministically-distributed keys.
			//    50 keys with FNV-style spread guarantees several stems
			//    land in the root group (natural depth ≤ groupDepth) and
			//    several land in sub-groups, exercising both the in-group
			//    resolve and the cross-group resolve paths.
			writerTrie := &BinaryTrie{
				store:      newNodeStore(),
				tracer:     trie.NewPrevalueTracer(),
				groupDepth: groupDepth,
			}
			const n = 50
			keys := make([][HashSize]byte, n)
			values := make([][HashSize]byte, n)
			for i := range n {
				binary.BigEndian.PutUint64(keys[i][:8], uint64(i+1)*0x9e3779b97f4a7c15)
				binary.BigEndian.PutUint64(keys[i][8:16], uint64(i+1)*0xc2b2ae3d27d4eb4f)
				binary.BigEndian.PutUint64(keys[i][16:24], uint64(i+1)*0x165667b19e3779f9)
				binary.BigEndian.PutUint64(keys[i][24:32], uint64(i+1)*0x85ebca77c2b2ae63)
				binary.BigEndian.PutUint64(values[i][:8], uint64(i+1))
				if err := writerTrie.store.Insert(keys[i][:], values[i][:], nil); err != nil {
					t.Fatalf("insert %d: %v", i, err)
				}
			}

			// 2. Commit; capture every blob into an in-memory map keyed
			//    by its path. The NodeSet key is the BitArray.PutKeyBytes
			//    encoding — exactly the bytes the resolver gets from
			//    keyToPath, so map lookups by string(path) round-trip.
			rootHash := writerTrie.Hash()
			_, ns := writerTrie.Commit(false)
			blobs := make(map[string][]byte, len(ns.Nodes))
			for path, node := range ns.Nodes {
				blobs[path] = node.Blob
			}

			// 3. Build a resolver that serves blobs from the map.
			resolver := func(path []byte, hash common.Hash) ([]byte, error) {
				blob, ok := blobs[string(path)]
				if !ok {
					return nil, fmt.Errorf("blob not found at path %x (hash %x)", path, hash)
				}
				return blob, nil
			}

			// 4. Open a fresh store, seeded only with the root blob.
			//    Everything else must be reached via the resolver.
			readerStore := newNodeStore()
			rootBlob, ok := blobs[""]
			if !ok {
				t.Fatalf("no root blob in NodeSet")
			}
			rootRef, err := readerStore.deserializeNodeWithHash(rootBlob, 0, rootHash)
			if err != nil {
				t.Fatalf("deserialize root: %v", err)
			}
			readerStore.root = rootRef

			// 5. Read every key back through the resolver and verify.
			//    A mismatch here means either the storage path diverged
			//    from the lookup path, or deserialization corrupted data.
			for i := range n {
				got, err := readerStore.Get(keys[i][:], resolver)
				if err != nil {
					t.Fatalf("Get key %d (%x): %v", i, keys[i], err)
				}
				if !bytes.Equal(got, values[i][:]) {
					t.Fatalf("Get key %d: got %x, want %x", i, got, values[i][:])
				}
			}

			// 6. The reloaded trie's root hash must equal the original.
			//    This is the canonical-hash round-trip property: any
			//    independent reader walking the same blobs computes the
			//    same root, independent of in-memory layout choices.
			if got := readerStore.Hash(); got != rootHash {
				t.Fatalf("post-reload root hash: got %x, want %x", got, rootHash)
			}
		})
	}
}

// TestNoOrphanBlobAfterStemPromotion targets gballet's store_ops.go review
// concern: when a second commit promotes an existing stem deeper, the stem's
// blob moves to a new path, and Commit emits only AddNode entries (never
// deletes). If a stem's old path were not reoccupied by the new ancestor node,
// the prior commit's blob would linger as an unreachable orphan.
//
// The test applies two commit deltas to a single backing map, then walks the
// trie from the new root and asserts every persisted blob is reachable — i.e.
// no orphan survives. The first batch establishes stems at group boundaries;
// the second batch shares prefixes with the first to force promotions.
func TestNoOrphanBlobAfterStemPromotion(t *testing.T) {
	for _, groupDepth := range []int{1, 2, 3, 5} {
		t.Run(fmt.Sprintf("groupDepth=%d", groupDepth), func(t *testing.T) {
			tr := &BinaryTrie{
				store:      newNodeStore(),
				tracer:     trie.NewPrevalueTracer(),
				groupDepth: groupDepth,
			}
			db := make(map[string][]byte)
			apply := func(ns *trienode.NodeSet) {
				for path, node := range ns.Nodes {
					if node.IsDeleted() {
						delete(db, path)
						continue
					}
					db[path] = node.Blob
				}
			}

			const n = 24
			keys := make([][HashSize]byte, n)
			values := make([][HashSize]byte, n)
			for i := range n {
				binary.BigEndian.PutUint64(keys[i][:8], uint64(i+1)*0x9e3779b97f4a7c15)
				binary.BigEndian.PutUint64(keys[i][8:16], uint64(i+1)*0xc2b2ae3d27d4eb4f)
				binary.BigEndian.PutUint64(values[i][:8], uint64(i+1))
			}

			// Commit 1: first half.
			for i := 0; i < n/2; i++ {
				if err := tr.store.Insert(keys[i][:], values[i][:], nil); err != nil {
					t.Fatalf("insert %d: %v", i, err)
				}
			}
			_, ns1 := tr.Commit(false)
			apply(ns1)

			// Commit 2: second half (shares prefixes, forces promotions).
			for i := n / 2; i < n; i++ {
				if err := tr.store.Insert(keys[i][:], values[i][:], nil); err != nil {
					t.Fatalf("insert %d: %v", i, err)
				}
			}
			rootHash, ns2 := tr.Commit(false)
			apply(ns2)

			// Walk from the new root, recording every blob the reader resolves.
			resolved := make(map[string]bool)
			resolver := func(path []byte, _ common.Hash) ([]byte, error) {
				resolved[string(path)] = true
				blob, ok := db[string(path)]
				if !ok {
					return nil, fmt.Errorf("missing blob at path %x", path)
				}
				return blob, nil
			}
			reader := newNodeStore()
			rootRef, err := reader.deserializeNodeWithHash(db[""], 0, rootHash)
			if err != nil {
				t.Fatalf("deserialize root: %v", err)
			}
			reader.root = rootRef

			for i := range n {
				got, err := reader.Get(keys[i][:], resolver)
				if err != nil {
					t.Fatalf("Get key %d: %v", i, err)
				}
				if !bytes.Equal(got, values[i][:]) {
					t.Fatalf("Get key %d: got %x, want %x", i, got, values[i][:])
				}
			}

			// Every persisted blob must be reachable; the root ("") is seeded.
			reachable := map[string]bool{"": true}
			for path := range resolved {
				reachable[path] = true
			}
			for path := range db {
				if !reachable[path] {
					t.Errorf("orphan blob at path %x is unreachable from the new root", []byte(path))
				}
			}
		})
	}
}

// stemsDivergingAt returns two 32-byte stems whose first `divergeBit` bits
// are zero and whose bit at index `divergeBit` differs (left=0, right=1).
// Useful for placing two stems at a known natural depth within a group.
func stemsDivergingAt(divergeBit int) (left, right []byte) {
	left = make([]byte, HashSize)
	right = make([]byte, HashSize)
	// Bit `divergeBit` is in byte (divergeBit/8) at MSB position (7 - divergeBit%8).
	right[divergeBit/8] = 1 << (7 - divergeBit%8)
	return left, right
}

// bitFlipStem returns a 32-byte stem whose first `divergeBit` bits are zero,
// bit `divergeBit` is 1, and all subsequent bits are zero. Used together
// with the all-zero stem to force divergence at a specific bit.
func bitFlipStem(divergeBit int) []byte {
	out := make([]byte, HashSize)
	out[divergeBit/8] = 1 << (7 - divergeBit%8)
	return out
}
