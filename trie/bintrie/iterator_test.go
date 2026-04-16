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
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

// makeTrie creates a BinaryTrie populated with the given key-value pairs.
func makeTrie(t *testing.T, entries [][2]common.Hash) *BinaryTrie {
	t.Helper()
	tr := &BinaryTrie{
		root:   NewBinaryNode(),
		tracer: trie.NewPrevalueTracer(),
	}
	for _, kv := range entries {
		var err error
		tr.root, err = tr.root.Insert(kv[0][:], kv[1][:], nil, 0)
		if err != nil {
			t.Fatal(err)
		}
	}
	return tr
}

// countLeaves iterates the trie and returns the number of leaves visited.
func countLeaves(t *testing.T, tr *BinaryTrie) int {
	t.Helper()
	it, err := newBinaryNodeIterator(tr, nil)
	if err != nil {
		t.Fatal(err)
	}
	leaves := 0
	for it.Next(true) {
		if it.Leaf() {
			leaves++
		}
	}
	if it.Error() != nil {
		t.Fatalf("iterator error: %v", it.Error())
	}
	return leaves
}

// TestIteratorEmptyTrie verifies that iterating over an empty trie returns
// no nodes and reports no error.
func TestIteratorEmptyTrie(t *testing.T) {
	tr := &BinaryTrie{
		root:   Empty{},
		tracer: trie.NewPrevalueTracer(),
	}
	it, err := newBinaryNodeIterator(tr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if it.Next(true) {
		t.Fatal("expected no iteration over empty trie")
	}
	if it.Error() != nil {
		t.Fatalf("unexpected error: %v", it.Error())
	}
}

// TestIteratorSingleStem verifies iteration over a trie with a single stem
// node containing multiple values.
func TestIteratorSingleStem(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000003"), oneKey},
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000007"), oneKey},
		{common.HexToHash("00000000000000000000000000000000000000000000000000000000000000FF"), oneKey},
	})
	if leaves := countLeaves(t, tr); leaves != 3 {
		t.Fatalf("expected 3 leaves, got %d", leaves)
	}
}

// TestIteratorTwoStems verifies iteration over a trie with two stems
// separated by internal nodes, ensuring all leaves from both stems are visited.
func TestIteratorTwoStems(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000002"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000002"), oneKey},
	})
	if leaves := countLeaves(t, tr); leaves != 4 {
		t.Fatalf("expected 4 leaves, got %d", leaves)
	}
}

// TestIteratorLeafKeyAndBlob verifies that the iterator returns correct
// leaf keys and values.
func TestIteratorLeafKeyAndBlob(t *testing.T) {
	key := common.HexToHash("0000000000000000000000000000000000000000000000000000000000000005")
	val := common.HexToHash("00000000000000000000000000000000000000000000000000000000deadbeef")
	tr := makeTrie(t, [][2]common.Hash{{key, val}})

	it, err := newBinaryNodeIterator(tr, nil)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for it.Next(true) {
		if it.Leaf() {
			found = true
			if !bytes.Equal(it.LeafKey(), key[:]) {
				t.Fatalf("leaf key mismatch: got %x, want %x", it.LeafKey(), key)
			}
			if !bytes.Equal(it.LeafBlob(), val[:]) {
				t.Fatalf("leaf blob mismatch: got %x, want %x", it.LeafBlob(), val)
			}
		}
	}
	if !found {
		t.Fatal("expected to find a leaf")
	}
}

// TestIteratorEmptyNodeBacktrack is a regression test for the Empty node
// backtracking bug. Before the fix, encountering an Empty child during
// iteration would terminate the walk prematurely instead of backtracking
// to the parent and continuing with the next sibling.
func TestIteratorEmptyNodeBacktrack(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
	})

	if _, ok := tr.root.(*InternalNode); !ok {
		t.Fatalf("expected InternalNode root, got %T", tr.root)
	}
	if leaves := countLeaves(t, tr); leaves != 2 {
		t.Fatalf("expected 2 leaves, got %d (Empty backtrack bug?)", leaves)
	}
}

// TestIteratorHashedNodeNilData is a regression test for the nil-data guard.
// When nodeResolver encounters a zero-hash HashedNode, it returns (nil, nil).
// The iterator should treat this as Empty and continue rather than panicking.
func TestIteratorHashedNodeNilData(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
	})

	root, ok := tr.root.(*InternalNode)
	if !ok {
		t.Fatalf("expected InternalNode root, got %T", tr.root)
	}

	// Replace right child with a zero-hash HashedNode. nodeResolver
	// short-circuits on common.Hash{} and returns (nil, nil), which
	// triggers the nil-data guard in the iterator.
	root.right = HashedNode(common.Hash{})

	// Should not panic; the zero-hash right child should be treated as Empty.
	if leaves := countLeaves(t, tr); leaves != 1 {
		t.Fatalf("expected 1 leaf (zero-hash right node skipped), got %d", leaves)
	}
}

// TestIteratorManyStems verifies iteration correctness with many stems,
// producing a deep tree structure.
func TestIteratorManyStems(t *testing.T) {
	entries := make([][2]common.Hash, 16)
	for i := range entries {
		var key common.Hash
		key[0] = byte(i << 4)
		key[31] = 1
		entries[i] = [2]common.Hash{key, oneKey}
	}
	tr := makeTrie(t, entries)
	if leaves := countLeaves(t, tr); leaves != 16 {
		t.Fatalf("expected 16 leaves, got %d", leaves)
	}
}

// TestIteratorDeepTree verifies iteration over a trie with stems that share
// a long common prefix, producing many intermediate InternalNodes.
func TestIteratorDeepTree(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0"), oneKey},
		{common.HexToHash("0000000000E00000000000000000000000000000000000000000000000000000"), twoKey},
	})
	if leaves := countLeaves(t, tr); leaves != 2 {
		t.Fatalf("expected 2 leaves in deep tree, got %d", leaves)
	}
}

// collectLeaves iterates the trie and returns all (key, value) pairs visited.
func collectLeaves(t *testing.T, tr *BinaryTrie, start []byte) [][2][]byte {
	t.Helper()
	it, err := newBinaryNodeIterator(tr, start)
	if err != nil {
		t.Fatal(err)
	}
	var out [][2][]byte
	for it.Next(true) {
		if it.Leaf() {
			k := slices.Clone(it.LeafKey())
			v := slices.Clone(it.LeafBlob())
			out = append(out, [2][]byte{k, v})
		}
	}
	if it.Error() != nil {
		t.Fatalf("iterator error: %v", it.Error())
	}
	return out
}

// TestSeekEmptyStart verifies that seek with a nil/empty start behaves like
// a fresh iterator (no skipping).
func TestSeekEmptyStart(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
	})
	// Both nil and empty slice should iterate everything.
	if got := len(collectLeaves(t, tr, nil)); got != 2 {
		t.Fatalf("nil start: expected 2 leaves, got %d", got)
	}
	if got := len(collectLeaves(t, tr, []byte{})); got != 2 {
		t.Fatalf("empty start: expected 2 leaves, got %d", got)
	}
}

// TestSeekToExactKey verifies that seeking to an existing leaf key positions
// the iterator at that exact leaf.
func TestSeekToExactKey(t *testing.T) {
	keys := [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000002"), twoKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
	}
	tr := makeTrie(t, keys)

	// Seek to the second key. We expect to see [key2, key3].
	start := keys[1][0]
	got := collectLeaves(t, tr, start[:])
	if len(got) != 2 {
		t.Fatalf("expected 2 leaves after seek to %x, got %d", start, len(got))
	}
	if !bytes.Equal(got[0][0], keys[1][0][:]) {
		t.Fatalf("first leaf after seek: got %x, want %x", got[0][0], keys[1][0])
	}
	if !bytes.Equal(got[1][0], keys[2][0][:]) {
		t.Fatalf("second leaf after seek: got %x, want %x", got[1][0], keys[2][0])
	}
}

// TestSeekToBetweenKeys verifies that seeking to a key that doesn't exist
// positions the iterator at the next existing key (in-order).
func TestSeekToBetweenKeys(t *testing.T) {
	keys := [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000005"), twoKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
	}
	tr := makeTrie(t, keys)

	// Seek to a key between key0 and key1: should land at key1.
	between := common.HexToHash("0000000000000000000000000000000000000000000000000000000000000003")
	got := collectLeaves(t, tr, between[:])
	if len(got) != 2 {
		t.Fatalf("expected 2 leaves after seek between, got %d", len(got))
	}
	if !bytes.Equal(got[0][0], keys[1][0][:]) {
		t.Fatalf("first leaf: got %x, want %x", got[0][0], keys[1][0])
	}
	if !bytes.Equal(got[1][0], keys[2][0][:]) {
		t.Fatalf("second leaf: got %x, want %x", got[1][0], keys[2][0])
	}
}

// TestSeekIntoEmptySubtree verifies that seeking into a subtree where the
// chosen path is empty correctly backtracks to the next populated subtree.
func TestSeekIntoEmptySubtree(t *testing.T) {
	// Build a trie with stems split across the bit-0 and bit-1 subtrees.
	keys := [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), twoKey},
	}
	tr := makeTrie(t, keys)

	// Seek to a key in a subtree that's entirely missing (e.g., 0x40...).
	// The high bit is 0, so we'd descend left, but the left subtree only has
	// keys with the FIRST bit being 0 — and the seek bit pattern would walk
	// into a position that has no leaves at or after it on the left side,
	// requiring backtrack to the right subtree.
	missing := common.HexToHash("4000000000000000000000000000000000000000000000000000000000000001")
	got := collectLeaves(t, tr, missing[:])
	// Should land at key1 (the right subtree leaf).
	if len(got) != 1 {
		t.Fatalf("expected 1 leaf after seek into missing subtree, got %d", len(got))
	}
	if !bytes.Equal(got[0][0], keys[1][0][:]) {
		t.Fatalf("leaf: got %x, want %x", got[0][0], keys[1][0])
	}
}

// TestSeekPastEnd verifies that seeking past the last key returns no leaves.
func TestSeekPastEnd(t *testing.T) {
	keys := [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000002"), oneKey},
	}
	tr := makeTrie(t, keys)

	// Seek past the maximum key.
	beyond := common.HexToHash("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	got := collectLeaves(t, tr, beyond[:])
	if len(got) != 0 {
		t.Fatalf("expected 0 leaves after seek past end, got %d: %x", len(got), got)
	}
}

// TestSeekWithinSameStem verifies that seeking within a single stem (multiple
// values at different offsets) positions correctly at the requested offset.
func TestSeekWithinSameStem(t *testing.T) {
	// All three keys share the same stem; only the last byte differs.
	keys := [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000005"), twoKey},
		{common.HexToHash("00000000000000000000000000000000000000000000000000000000000000ff"), oneKey},
	}
	tr := makeTrie(t, keys)

	// Seek to offset 5: should yield keys 1 (offset 5) and 2 (offset 0xff).
	start := common.HexToHash("0000000000000000000000000000000000000000000000000000000000000005")
	got := collectLeaves(t, tr, start[:])
	if len(got) != 2 {
		t.Fatalf("expected 2 leaves, got %d", len(got))
	}
	if got[0][0][31] != 0x05 {
		t.Fatalf("first leaf offset: got 0x%02x, want 0x05", got[0][0][31])
	}
	if got[1][0][31] != 0xff {
		t.Fatalf("second leaf offset: got 0x%02x, want 0xff", got[1][0][31])
	}

	// Seek to offset 6 (between 5 and 0xff): should yield only key 2.
	start[31] = 0x06
	got = collectLeaves(t, tr, start[:])
	if len(got) != 1 {
		t.Fatalf("expected 1 leaf after seek to offset 6, got %d", len(got))
	}
	if got[0][0][31] != 0xff {
		t.Fatalf("leaf offset: got 0x%02x, want 0xff", got[0][0][31])
	}
}

// TestSeekResumeSimulation simulates a generator interruption: iterate halfway,
// extract the last leaf key, build a fresh iterator, seek to the next key, and
// verify that the resumed iteration produces the remaining leaves.
func TestSeekResumeSimulation(t *testing.T) {
	// Construct a deterministic set of keys.
	var keys [][2]common.Hash
	for i := range 16 {
		var k common.Hash
		k[0] = byte(i << 4) // distribute across the high nibble
		k[31] = 0x01
		keys = append(keys, [2]common.Hash{k, oneKey})
	}
	tr := makeTrie(t, keys)

	// First pass: collect all leaves.
	all := collectLeaves(t, tr, nil)
	if len(all) != 16 {
		t.Fatalf("first pass: expected 16 leaves, got %d", len(all))
	}

	// Stop after the 7th leaf and resume.
	stopIdx := 7
	lastKey := all[stopIdx][0]

	// Resume: seek to the byte AFTER lastKey (we use lastKey + 1 in the last
	// byte; for our keys this is sufficient because each key's last byte is
	// 0x01 and we want to go to the NEXT stem).
	resumeKey := slices.Clone(lastKey)
	// Increment the last byte; if it overflows, that's fine for these keys
	// because all our last bytes are 0x01.
	resumeKey[31]++
	// But actually we want to start AT lastKey + 1, which for our keys means
	// we want the NEXT stem. Since each stem has only one value at offset 0x01
	// and we want everything strictly after lastKey, set offset to 0x02.
	got := collectLeaves(t, tr, resumeKey)
	if len(got) != len(all)-stopIdx-1 {
		t.Fatalf("resume: expected %d leaves, got %d", len(all)-stopIdx-1, len(got))
	}
	for i, leaf := range got {
		want := all[stopIdx+1+i]
		if !bytes.Equal(leaf[0], want[0]) {
			t.Fatalf("resume leaf %d: got %x, want %x", i, leaf[0], want[0])
		}
	}
}

// TestSeekDeepTree verifies seek works on a tree with a long shared prefix.
func TestSeekDeepTree(t *testing.T) {
	keys := [][2]common.Hash{
		{common.HexToHash("0000000000C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0"), oneKey},
		{common.HexToHash("0000000000E00000000000000000000000000000000000000000000000000000"), twoKey},
	}
	tr := makeTrie(t, keys)

	// Seek to the first key exactly.
	got := collectLeaves(t, tr, keys[0][0][:])
	if len(got) != 2 {
		t.Fatalf("seek to first: expected 2 leaves, got %d", len(got))
	}
	if !bytes.Equal(got[0][0], keys[0][0][:]) {
		t.Fatalf("first leaf: got %x, want %x", got[0][0], keys[0][0])
	}

	// Seek to the second key exactly.
	got = collectLeaves(t, tr, keys[1][0][:])
	if len(got) != 1 {
		t.Fatalf("seek to second: expected 1 leaf, got %d", len(got))
	}
	if !bytes.Equal(got[0][0], keys[1][0][:]) {
		t.Fatalf("leaf: got %x, want %x", got[0][0], keys[1][0])
	}
}

// TestIteratorNodeCount verifies the total number of Next(true) calls
// for a known tree structure.
func TestIteratorNodeCount(t *testing.T) {
	tr := makeTrie(t, [][2]common.Hash{
		{common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001"), oneKey},
		{common.HexToHash("8000000000000000000000000000000000000000000000000000000000000001"), oneKey},
	})

	it, err := newBinaryNodeIterator(tr, nil)
	if err != nil {
		t.Fatal(err)
	}

	total := 0
	leaves := 0
	for it.Next(true) {
		total++
		if it.Leaf() {
			leaves++
		}
	}
	if leaves != 2 {
		t.Fatalf("expected 2 leaves, got %d", leaves)
	}
	// Root(InternalNode) + leaf1 (from left StemNode) + leaf2 (from right StemNode) = 3
	// StemNodes are not returned as separate steps; the iterator advances
	// directly to the first non-nil value within the stem.
	if total != 3 {
		t.Fatalf("expected 3 total nodes, got %d", total)
	}
}
