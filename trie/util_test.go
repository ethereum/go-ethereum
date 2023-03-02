// Copyright 2022 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
)

// Tests if the trie diffs are tracked correctly.
func TestTrieTracer(t *testing.T) {
	db := NewDatabase(rawdb.NewMemoryDatabase())
	trie := NewEmpty(db)
	trie.tracer = newTracer()

	// Insert a batch of entries, all the nodes should be marked as inserted
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Hash()

	seen := make(map[string]struct{})
	it := trie.NodeIterator(nil)
	for it.Next(true) {
		if it.Leaf() {
			continue
		}
		seen[string(it.Path())] = struct{}{}
	}
	inserted := trie.tracer.insertList()
	if len(inserted) != len(seen) {
		t.Fatalf("Unexpected inserted node tracked want %d got %d", len(seen), len(inserted))
	}
	for _, k := range inserted {
		_, ok := seen[string(k)]
		if !ok {
			t.Fatalf("Unexpected inserted node")
		}
	}
	deleted := trie.tracer.deleteList()
	if len(deleted) != 0 {
		t.Fatalf("Unexpected deleted node tracked %d", len(deleted))
	}

	// Commit the changes and re-create with new root
	root, nodes := trie.Commit(false)
	if err := db.Update(NewWithNodeSet(nodes)); err != nil {
		t.Fatal(err)
	}
	trie, _ = New(TrieID(root), db)
	trie.tracer = newTracer()

	// Delete all the elements, check deletion set
	for _, val := range vals {
		trie.Delete([]byte(val.k))
	}
	trie.Hash()

	inserted = trie.tracer.insertList()
	if len(inserted) != 0 {
		t.Fatalf("Unexpected inserted node tracked %d", len(inserted))
	}
	deleted = trie.tracer.deleteList()
	if len(deleted) != len(seen) {
		t.Fatalf("Unexpected deleted node tracked want %d got %d", len(seen), len(deleted))
	}
	for _, k := range deleted {
		_, ok := seen[string(k)]
		if !ok {
			t.Fatalf("Unexpected inserted node")
		}
	}
}

func TestTrieTracerNoop(t *testing.T) {
	trie := NewEmpty(NewDatabase(rawdb.NewMemoryDatabase()))
	trie.tracer = newTracer()

	// Insert a batch of entries, all the nodes should be marked as inserted
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.Update([]byte(val.k), []byte(val.v))
	}
	for _, val := range vals {
		trie.Delete([]byte(val.k))
	}
	if len(trie.tracer.insertList()) != 0 {
		t.Fatalf("Unexpected inserted node tracked %d", len(trie.tracer.insertList()))
	}
	if len(trie.tracer.deleteList()) != 0 {
		t.Fatalf("Unexpected deleted node tracked %d", len(trie.tracer.deleteList()))
	}
}

func TestTrieTracePrevValue(t *testing.T) {
	db := NewDatabase(rawdb.NewMemoryDatabase())
	trie := NewEmpty(db)
	trie.tracer = newTracer()

	paths, blobs := trie.tracer.prevList()
	if len(paths) != 0 || len(blobs) != 0 {
		t.Fatalf("Nothing should be tracked")
	}
	// Insert a batch of entries, all the nodes should be marked as inserted
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.Update([]byte(val.k), []byte(val.v))
	}
	paths, blobs = trie.tracer.prevList()
	if len(paths) != 0 || len(blobs) != 0 {
		t.Fatalf("Nothing should be tracked")
	}

	// Commit the changes and re-create with new root
	root, nodes := trie.Commit(false)
	if err := db.Update(NewWithNodeSet(nodes)); err != nil {
		t.Fatal(err)
	}
	trie, _ = New(TrieID(root), db)
	trie.tracer = newTracer()
	trie.resolveAndTrack(root.Bytes(), nil)

	// Load all nodes in trie
	for _, val := range vals {
		trie.TryGet([]byte(val.k))
	}

	// Ensure all nodes are tracked by tracer with correct prev-values
	iter := trie.NodeIterator(nil)
	seen := make(map[string][]byte)
	for iter.Next(true) {
		// Embedded nodes are ignored since they are not present in
		// database.
		if iter.Hash() == (common.Hash{}) {
			continue
		}
		seen[string(iter.Path())] = common.CopyBytes(iter.NodeBlob())
	}

	paths, blobs = trie.tracer.prevList()
	if len(paths) != len(seen) || len(blobs) != len(seen) {
		t.Fatalf("Unexpected tracked values")
	}
	for i, path := range paths {
		blob := blobs[i]
		prev, ok := seen[string(path)]
		if !ok {
			t.Fatalf("Missing node %v", path)
		}
		if !bytes.Equal(blob, prev) {
			t.Fatalf("Unexpected value path: %v, want: %v, got: %v", path, prev, blob)
		}
	}

	// Re-open the trie and iterate the trie, ensure nothing will be tracked.
	// Iterator will not link any loaded nodes to trie.
	trie, _ = New(TrieID(root), db)
	trie.tracer = newTracer()

	iter = trie.NodeIterator(nil)
	for iter.Next(true) {
	}
	paths, blobs = trie.tracer.prevList()
	if len(paths) != 0 || len(blobs) != 0 {
		t.Fatalf("Nothing should be tracked")
	}

	// Re-open the trie and generate proof for entries, ensure nothing will
	// be tracked. Prover will not link any loaded nodes to trie.
	trie, _ = New(TrieID(root), db)
	trie.tracer = newTracer()
	for _, val := range vals {
		trie.Prove([]byte(val.k), 0, rawdb.NewMemoryDatabase())
	}
	paths, blobs = trie.tracer.prevList()
	if len(paths) != 0 || len(blobs) != 0 {
		t.Fatalf("Nothing should be tracked")
	}

	// Delete entries from trie, ensure all previous values are correct.
	trie, _ = New(TrieID(root), db)
	trie.tracer = newTracer()
	trie.resolveAndTrack(root.Bytes(), nil)

	for _, val := range vals {
		trie.TryDelete([]byte(val.k))
	}
	paths, blobs = trie.tracer.prevList()
	if len(paths) != len(seen) || len(blobs) != len(seen) {
		t.Fatalf("Unexpected tracked values")
	}
	for i, path := range paths {
		blob := blobs[i]
		prev, ok := seen[string(path)]
		if !ok {
			t.Fatalf("Missing node %v", path)
		}
		if !bytes.Equal(blob, prev) {
			t.Fatalf("Unexpected value path: %v, want: %v, got: %v", path, prev, blob)
		}
	}
}

func TestDeleteAll(t *testing.T) {
	db := NewDatabase(rawdb.NewMemoryDatabase())
	trie := NewEmpty(db)
	trie.tracer = newTracer()

	// Insert a batch of entries, all the nodes should be marked as inserted
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.Update([]byte(val.k), []byte(val.v))
	}
	root, set := trie.Commit(false)
	if err := db.Update(NewWithNodeSet(set)); err != nil {
		t.Fatal(err)
	}
	// Delete entries from trie, ensure all values are detected
	trie, _ = New(TrieID(root), db)
	trie.tracer = newTracer()
	trie.resolveAndTrack(root.Bytes(), nil)

	// Iterate all existent nodes
	var (
		it    = trie.NodeIterator(nil)
		nodes = make(map[string][]byte)
	)
	for it.Next(true) {
		if it.Hash() != (common.Hash{}) {
			nodes[string(it.Path())] = common.CopyBytes(it.NodeBlob())
		}
	}

	// Perform deletion to purge the entire trie
	for _, val := range vals {
		trie.Delete([]byte(val.k))
	}
	root, set = trie.Commit(false)
	if root != types.EmptyRootHash {
		t.Fatalf("Invalid trie root %v", root)
	}
	for path, blob := range set.deletes {
		prev, ok := nodes[path]
		if !ok {
			t.Fatalf("Extra node deleted %v", []byte(path))
		}
		if !bytes.Equal(prev, blob) {
			t.Fatalf("Unexpected previous value %v", []byte(path))
		}
	}
	if len(set.deletes) != len(nodes) {
		t.Fatalf("Unexpected deletion set")
	}
}
