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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

// makeTrie creates a BinaryTrie populated with the given key-value pairs.
func makeTrie(t *testing.T, entries [][2]common.Hash) *BinaryTrie {
	t.Helper()
	store := newNodeStore()
	tr := &BinaryTrie{
		store:  store,
		tracer: trie.NewPrevalueTracer(),
	}
	for _, kv := range entries {
		if err := store.Insert(kv[0][:], kv[1][:], nil); err != nil {
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
		store:  newNodeStore(),
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

	if tr.store.root.Kind() != kindInternal {
		t.Fatalf("expected InternalNode root, got kind %d", tr.store.root.Kind())
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

	root := tr.store.root
	if root.Kind() != kindInternal {
		t.Fatalf("expected InternalNode root, got kind %d", root.Kind())
	}
	rootNode := tr.store.getInternal(root.Index())

	// Replace right child with a zero-hash HashedNode. nodeResolver
	// short-circuits on common.Hash{} and returns (nil, nil), which
	// triggers the nil-data guard in the iterator.
	rootNode.right = tr.store.newHashedRef(common.Hash{})

	// Should not panic; the zero-hash right child should be treated as Empty.
	// Since the hashed node can't be resolved (nil data -> empty deserialization),
	// only the left leaf should be counted.
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
	if leaves != 1 {
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
