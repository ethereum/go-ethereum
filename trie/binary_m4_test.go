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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBinaryTrieM4EmptyHash(t *testing.T) {
	trie := NewM4BinaryTrie()
	got := trie.Hash()
	exp := emptyRoot[:]

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}

	trie = NewM4BinaryTrieWithBlake2b()
	got = trie.Hash()
	// This is the wrong empty root for blake2b. We are only focused
	// on performance measurements at the moment.
	exp = emptyRoot[:]

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieM4InsertOneLeafAndHash(t *testing.T) {
	trie := NewM4BinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("8d789541cbd874b968cc419721f727d3b77594cf002464c5e256512616c30cd4")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieM4InsertTwoLeavesAndHash(t *testing.T) {
	trie := NewM4BinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{8}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("ab0e00b26edf69db6bcdc92e40b78e01c243535539788e634c3c8dfb01c28b55")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}

	trie = NewM4BinaryTrieWithBlake2b()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{8}, []byte{10})
	got = trie.Hash()
	exp = common.FromHex("29b4bf14cc632e6d0cb8778203d9baef3ca23fb765b642dc6ce9c618ede23070")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}
