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
	exp := []byte{0, 0, 0, 0, 0, 0, 0, 0,0, 0, 0, 0, 0, 0, 0, 1,0, 0, 0, 0, 0, 0, 1, 0,0, 0, 0, 0, 0, 0, 1, 1,}
	if !bytes.Equal(binKey[:],exp) {
		t.Fatalf("invalid key conversion, got %x != exp %x", binKey[:], exp)
	}
}
func TestBinaryKeyCreationEmpty(t *testing.T) {
	byteKey := []byte(nil)
	binKey := newBinKey(byteKey)
	exp := []byte(nil)
	if !bytes.Equal(binKey[:],exp) {
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
			key: newBinKey([]byte{2}),
			value: []byte{10},
		},
		{
			key: newBinKey([]byte{0}),
			value: []byte{10},
		},
		{
			key: newBinKey([]byte{1}),
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
	exp := common.FromHex("d8ead31beb79e4a13c00130997c2e0e55409bb16b1e97e02129cb8d966167171")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}
