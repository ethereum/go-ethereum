// Copyright 2015 The go-ethereum Authors
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

package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverage(t *testing.T) {
	// Create some arbitrary test state to iterate
	db, root, _ := makeTestState(nil)

	state, err := New(root, db)
	if err != nil {
		t.Fatalf("failed to create state trie at %x: %v", root, err)
	}
	// Gather all the node hashes found by the iterator
	hashes := make(map[common.Hash]struct{})
	for it := NewNodeIterator(state); it.Next(); {
		if it.Hash != (common.Hash{}) {
			hashes[it.Hash] = struct{}{}
		}
	}
	// Cross check the hashes and the database itself
	for hash, _ := range hashes {
		if _, err := db.Get(hash.Bytes()); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", hash, err)
		}
	}
	for _, key := range db.(*ethdb.MemDatabase).Keys() {
		if bytes.HasPrefix(key, []byte("secure-key-")) {
			continue
		}
		if bytes.HasPrefix(key, trie.ParentReferenceIndexPrefix) {
			continue
		}
		if _, ok := hashes[common.BytesToHash(key)]; !ok {
			t.Errorf("state entry not reported %x", key)
		}
	}
}

// Tests that the node iterator hook is invoked for all the nodes of the state,
// also testing that the iterator indeed avoids visiting aborted branches.
func TestNodeIteratorHookCoverageFull(t *testing.T)  { testNodeIteratorHookCoverage(t, false) }
func TestNodeIteratorHookCoverageDedup(t *testing.T) { testNodeIteratorHookCoverage(t, true) }

func testNodeIteratorHookCoverage(t *testing.T, dedup bool) {
	// Create some arbitrary test state to iterate
	db, root, _ := makeTestState(nil)

	state, err := New(root, db)
	if err != nil {
		t.Fatalf("failed to create state trie at %x: %v", root, err)
	}
	// Gather all the node hashes found by the iterator
	hashes := make(map[common.Hash]int)

	it := NewNodeIterator(state)
	it.PreOrderHook = func(hash, parent common.Hash) bool {
		if !dedup {
			return true
		}
		return hashes[hash] == 0
	}
	for it.Next() {
		if it.Hash != (common.Hash{}) {
			hashes[it.Hash]++
		}
	}
	// Cross check the hashes and the database itself
	for hash, _ := range hashes {
		if _, err := db.Get(hash.Bytes()); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", hash, err)
		}
	}
	for _, key := range db.(*ethdb.MemDatabase).Keys() {
		if bytes.HasPrefix(key, []byte("secure-key-")) {
			continue
		}
		if bytes.HasPrefix(key, trie.ParentReferenceIndexPrefix) {
			continue
		}
		if _, ok := hashes[common.BytesToHash(key)]; !ok {
			t.Errorf("state entry not reported %x", key)
		}
	}
	// Check whether duplicates were avoided or not
	duplicates := 0
	for hash, count := range hashes {
		if count > 1 {
			duplicates++
			if dedup {
				t.Errorf("duplicate (%d) iteration: %x", count, hash)
			}
		}
	}
	if !dedup && duplicates == 0 {
		t.Errorf("iterator didn't traverse common subtrees")
	}
}
