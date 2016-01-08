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

package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Tests that all index entries are stored in the database after a trie commit.
func TestTrieIndex(t *testing.T) {
	// Create some arbitrary test trie to iterate
	db, trie, _ := makeTestTrie()

	// Gather all the indexes that should be present in the database
	indexes := make(map[string]struct{})
	for it := NewNodeIterator(trie); it.Next(); {
		if (it.Hash != common.Hash{}) && (it.Parent != common.Hash{}) {
			indexes[string(ParentReferenceIndexKey(it.Parent.Bytes(), it.Hash.Bytes()))] = struct{}{}
		}
	}
	// Cross check the indexes and the database itself
	for index, _ := range indexes {
		if _, err := db.Get([]byte(index)); err != nil {
			t.Errorf("failed to retrieve reported index %x: %v", index, err)
		}
	}
	for _, key := range db.(*ethdb.MemDatabase).Keys() {
		if bytes.HasPrefix(key, ParentReferenceIndexPrefix) {
			if _, ok := indexes[string(key)]; !ok {
				t.Errorf("index entry not reported %x", key)
			}
		}
	}
}
