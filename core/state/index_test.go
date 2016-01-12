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

// Tests that all index entries are stored in the database after a state commit.
func TestStateIndexDangling(t *testing.T) {
	testStateIndex(t, nil)
}
func TestStateIndexRooted(t *testing.T) {
	testStateIndex(t, []common.Hash{common.BytesToHash([]byte{0x01}), common.BytesToHash([]byte{0x02, 0x03})})
}

func testStateIndex(t *testing.T, referrers []common.Hash) {
	// Create some arbitrary test state to iterate
	db, root, _ := makeTestState(referrers)

	state, err := New(root, db)
	if err != nil {
		t.Fatalf("failed to create state trie at %x: %v", root, err)
	}
	// Gather all the indexes that should be present in the database
	indexes := make(map[string]struct{})
	for it := NewNodeIterator(state); it.Next(); {
		if (it.Hash != common.Hash{}) && (it.Parent != common.Hash{}) {
			indexes[string(trie.ParentReferenceIndexKey(it.Parent.Bytes(), it.Hash.Bytes()))] = struct{}{}
		}
	}
	for _, referrer := range referrers {
		indexes[string(trie.ParentReferenceIndexKey(referrer.Bytes(), root.Bytes()))] = struct{}{}
	}
	// Cross check the indexes and the database itself
	for index, _ := range indexes {
		if _, err := db.Get([]byte(index)); err != nil {
			t.Errorf("failed to retrieve reported index %x: %v", index, err)
		}
	}
	for _, key := range db.(*ethdb.MemDatabase).Keys() {
		if bytes.HasPrefix(key, trie.ParentReferenceIndexPrefix) {
			if _, ok := indexes[string(key)]; !ok {
				t.Errorf("index entry not reported %x", key)
			}
		}
	}
}
