// Copyright 2020 The go-ethereum Authors
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
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBinaryKeyCreation(t *testing.T) {
	byteKey := []byte{0, 1, 2, 3}
	binKey := newBinKey(byteKey)
	exp := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1}
	if !bytes.Equal(binKey[:], exp) {
		t.Fatalf("invalid key conversion, got %x != exp %x", binKey[:], exp)
	}
}
func TestBinaryKeyCreationEmpty(t *testing.T) {
	byteKey := []byte(nil)
	binKey := newBinKey(byteKey)
	exp := []byte(nil)
	if !bytes.Equal(binKey[:], exp) {
		t.Fatalf("invalid key conversion, got %x != exp %x", binKey[:], exp)
	}
}

func TestBinaryKeyCommonLength(t *testing.T) {
	byteKey1 := []byte{0, 1, 2, 3}
	binKey1 := newBinKey(byteKey1)
	byteKey2 := []byte{0, 1, 3, 3}
	binKey2 := newBinKey(byteKey2)

	split := binKey1.commonLength(binKey2)

	if split != 23 {
		t.Fatalf("split at wrong location: got %d != exp 23", split)
	}

}

func TestBinaryKeyCommonLengthLast(t *testing.T) {
	byteKey1 := []byte{0, 1, 2, 3}
	binKey1 := newBinKey(byteKey1)
	byteKey2 := []byte{0, 1, 2, 2}
	binKey2 := newBinKey(byteKey2)

	split := binKey1.commonLength(binKey2)

	if split != 31 {
		t.Fatalf("split at wrong location: got %d != exp 31", split)
	}

}

func TestBinaryKeyCommonLengthFirst(t *testing.T) {
	byteKey1 := []byte{0, 1, 2, 3}
	binKey1 := newBinKey(byteKey1)
	byteKey2 := []byte{128, 1, 2, 3}
	binKey2 := newBinKey(byteKey2)

	split := binKey1.commonLength(binKey2)

	if split != 0 {
		t.Fatalf("split at wrong location: got %d != exp 0", split)
	}

}

func TestBinaryStoreSort(t *testing.T) {
	store := store{
		{
			key:   newBinKey([]byte{2}),
			value: []byte{10},
		},
		{
			key:   newBinKey([]byte{0}),
			value: []byte{10},
		},
		{
			key:   newBinKey([]byte{1}),
			value: []byte{10},
		},
	}

	sort.Sort(store)
	for i := range store {
		if !bytes.Equal(store[i].key, newBinKey([]byte{byte(i)})) {
			t.Fatalf("item %d out of order: %d", i, store[i])
		}
	}
}

func TestBinaryTrieEmptyHash(t *testing.T) {
	trie := NewBinaryTrie()
	got := trie.Hash()
	exp := emptyRoot[:]

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertOneLeafAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("06b42a1e2618f532aca432615e040bb1fc63fd2c3a03e94bc3a7dd8b15eb46a0")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertTwoLeavesAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{8}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("3590925ae30faa2a566bd4fd6605a205b1a1553b223f46c58eaa73646b173245")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}
func TestBinaryTrieInsertTwoLeavesAtFirstBitAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{128}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("8724dc87faa3ecd18a24f612632f7b27e02a6060d8603c378e39c04ad8b7259a")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertTwoLeavesAtEndBitAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{1}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("92f4d45186b0f0d8f737ce9c4a202b35ef897c660c16bd7dc235bffd1178d0cc")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertWithOffsetAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{8}, []byte{18})
	trie.Update([]byte{11}, []byte{20})
	got := trie.Hash()
	exp := common.FromHex("73df5c5434b663c53847bdf7ac5f67b701184152b587bfdd4d7669b6198495fe")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}
