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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	root, nodes, _ := trie.Commit(false)
	if err := db.Update(NewWithNodeSet(nodes)); err != nil {
		t.Fatal(err)
	}
	trie, _ = New(common.Hash{}, root, db)
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
