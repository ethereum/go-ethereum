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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie/testutil"
)

func makeTrieHistories(n int) []trieHistory {
	var (
		parent    = testutil.RandomHash()
		histories []trieHistory
	)
	for i := 0; i < n; i++ {
		var (
			root  = testutil.RandomHash()
			tries []trieDiff
		)
		for j := 0; j < 10; j++ {
			entry := trieDiff{Owner: testutil.RandomHash()}
			for z := 0; z < 10; z++ {
				if rand.Intn(2) == 0 {
					entry.Nodes = append(entry.Nodes, nodeDiff{
						Path: testutil.RandBytes(30),
						Prev: testutil.RandBytes(30),
					})
				} else {
					entry.Nodes = append(entry.Nodes, nodeDiff{
						Path: testutil.RandBytes(30),
						Prev: []byte{},
					})
				}
			}
			tries = append(tries, entry)
		}
		histories = append(histories, trieHistory{
			Parent: parent,
			Root:   root,
			Tries:  tries,
		})
		parent = root
	}
	return histories
}

func TestEncodeDecodeTrieHistory(t *testing.T) {
	var hs = makeTrieHistories(10)
	for i := 0; i < len(hs); i++ {
		blob, err := hs[i].encode()
		if err != nil {
			t.Fatalf("Failed to encode trie history %v", err)
		}
		var dec trieHistory
		if err := dec.decode(blob); err != nil {
			t.Fatalf("Failed to decode trie history %v", err)
		}
		if !reflect.DeepEqual(dec, hs[i]) {
			t.Fatalf("Unexpected value")
		}
	}
}

func TestLoadStoreTrieHistory(t *testing.T) {
	var (
		hs         = makeTrieHistories(10)
		freezer, _ = openFreezer(t.TempDir(), false)
	)
	for i := 0; i < len(hs); i++ {
		blob, err := hs[i].encode()
		if err != nil {
			t.Fatalf("Failed to encode trie history %v", err)
		}
		rawdb.WriteTrieHistory(freezer, uint64(i+1), blob)
	}
	for i := 0; i < len(hs); i++ {
		h, err := loadTrieHistory(freezer, uint64(i+1))
		if err != nil {
			t.Fatalf("Failed to load trie history %v", err)
		}
		if h.Root != hs[i].Root {
			t.Fatalf("Unexpected root want %x got %x", hs[i].Root, h.Root)
		}
		if h.Parent != hs[i].Parent {
			t.Fatalf("Unexpected parent want %x got %x", hs[i].Parent, h.Parent)
		}
		if !reflect.DeepEqual(h.Tries, hs[i].Tries) {
			t.Fatal("Unexpected states")
		}
	}
}

func assertTrieHistory(t *testing.T, freezer *rawdb.ResettableFreezer, id uint64, exist bool) {
	blob := rawdb.ReadTrieHistory(freezer, id)
	if exist && len(blob) == 0 {
		t.Errorf("Failed to load trie history, %d", id)
	}
	if !exist && len(blob) != 0 {
		t.Errorf("Unexpected trie history, %d", id)
	}
}

func assertReverseDiffInRange(t *testing.T, freezer *rawdb.ResettableFreezer, from, to uint64, exist bool) {
	for i, j := from, 0; i <= to; i, j = i+1, j+1 {
		assertTrieHistory(t, freezer, i, exist)
	}
}

func TestTruncateHeadTrieHistory(t *testing.T) {
	var (
		hs         = makeTrieHistories(10)
		freezer, _ = openFreezer(t.TempDir(), false)
	)
	for i := 0; i < len(hs); i++ {
		blob, err := hs[i].encode()
		if err != nil {
			t.Fatalf("Failed to encode trie history %v", err)
		}
		rawdb.WriteTrieHistory(freezer, uint64(i+1), blob)
	}
	for size := len(hs); size > 0; size-- {
		pruned, err := truncateFromHead(freezer, uint64(size-1))
		if err != nil {
			t.Fatalf("Failed to truncate from head %v", err)
		}
		if pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, freezer, uint64(size), uint64(10), false)
		assertReverseDiffInRange(t, freezer, uint64(1), uint64(size-1), true)
	}
}

func TestTruncateTailHistory(t *testing.T) {
	var (
		hs         = makeTrieHistories(10)
		freezer, _ = openFreezer(t.TempDir(), false)
	)
	for i := 0; i < len(hs); i++ {
		blob, err := hs[i].encode()
		if err != nil {
			t.Fatalf("Failed to encode trie history %v", err)
		}
		rawdb.WriteTrieHistory(freezer, uint64(i+1), blob)
	}
	for newTail := 1; newTail < len(hs); newTail++ {
		pruned, _ := truncateFromTail(freezer, uint64(newTail))
		if pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, freezer, uint64(1), uint64(newTail), false)
		assertReverseDiffInRange(t, freezer, uint64(newTail+1), uint64(10), true)
	}
}

func TestTruncateTailTrieHistories(t *testing.T) {
	var cases = []struct {
		limit       uint64
		expPruned   int
		maxPruned   uint64
		minUnpruned uint64
		empty       bool
	}{
		{
			1, 9, 9, 10, false,
		},
		{
			0, 10, 10, 0 /* no meaning */, true,
		},
		{
			10, 0, 0, 1, false,
		},
	}
	for _, c := range cases {
		var (
			hs         = makeTrieHistories(10)
			freezer, _ = openFreezer(t.TempDir(), false)
		)
		for i := 0; i < len(hs); i++ {
			blob, err := hs[i].encode()
			if err != nil {
				t.Fatalf("Failed to encode trie history %v", err)
			}
			rawdb.WriteTrieHistory(freezer, uint64(i+1), blob)
		}
		pruned, _ := truncateFromTail(freezer, uint64(10)-c.limit)
		if pruned != c.expPruned {
			t.Error("Unexpected pruned items", "want", c.expPruned, "got", pruned)
		}
		if c.empty {
			assertReverseDiffInRange(t, freezer, uint64(1), uint64(10), false)
		} else {
			assertReverseDiffInRange(t, freezer, uint64(1), c.maxPruned, false)
			assertTrieHistory(t, freezer, c.minUnpruned, true)
		}
	}
}

// openFreezer initializes the freezer instance for storing trie histories.
func openFreezer(datadir string, readOnly bool) (*rawdb.ResettableFreezer, error) {
	return rawdb.NewTrieHistoryFreezer(datadir, readOnly)
}
