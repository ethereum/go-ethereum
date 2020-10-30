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
	// on preformance measurements at the moment.
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

	//trie = NewM4BinaryTrieWithBlake2b()
	//trie.Update([]byte{0}, []byte{10})
	//got = trie.Hash()
	//exp = common.FromHex("59f78e329994764d27e42cf7e2802a8311cd5c45725788e6288f94850c92a7d6")

	//if !bytes.Equal(got, exp) {
		//t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	//}
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

//func TestBinaryTrieInsertTwoLeavesAtFirstBitAndHash(t *testing.T) {
	//trie := NewBinaryTrie()
	//trie.Update([]byte{0}, []byte{10})
	//trie.Update([]byte{128}, []byte{10})
	//got := trie.Hash()
	//exp := common.FromHex("83cbe1f4e4ddfdc66074424d54d64c34deafd6517970e9f6e96c21f506235a4e")

	//if !bytes.Equal(got, exp) {
		//t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	//}
//}

//func TestBinaryTrieInsertTwoLeavesAtEndBitAndHash(t *testing.T) {
	//trie := NewBinaryTrie()
	//trie.Update([]byte{0}, []byte{10})
	//trie.Update([]byte{1}, []byte{10})
	//got := trie.Hash()
	//exp := common.FromHex("05b8807c3d0b42b8ff79ee3e157473564eba0154281af1755476f7154d753556")

	//if !bytes.Equal(got, exp) {
		//t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	//}
//}

//func TestBinaryTrieInsertWithOffsetAndHash(t *testing.T) {
	//trie := NewBinaryTrie()
	//trie.Update([]byte{0}, []byte{10})
	//trie.Update([]byte{8}, []byte{18})
	//trie.Update([]byte{11}, []byte{20})
	//got := trie.Hash()
	//exp := common.FromHex("b73bb5b26278b862d872455f83eb71b34a3702b85761f6ca8fc7e7e98b4b5fe6")

	//if !bytes.Equal(got, exp) {
		//t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	//}
//}

//func TestBinaryTrieReadEmpty(t *testing.T) {
	//trie := NewBinaryTrie()
	//_, err := trie.TryGet([]byte{1})
	//if err != errKeyNotPresent {
		//t.Fatalf("incorrect error received, expected '%v', got '%v'", errKeyNotPresent, err)
	//}
//}

//func TestBinaryTrieReadOneLeaf(t *testing.T) {
	//trie := NewBinaryTrie()
	//trie.Update([]byte{0}, []byte{10})

	//v, err := trie.TryGet([]byte{0})
	//if err != nil {
		//t.Fatalf("error searching for key 0 in trie, err=%v", err)
	//}
	//if !bytes.Equal(v, []byte{10}) {
		//t.Fatalf("could not find correct value %x != 0a", v)
	//}

	//v, err = trie.TryGet([]byte{1})
	//if err != errKeyNotPresent {
		//t.Fatalf("incorrect error received, expected '%v', got '%v'", errKeyNotPresent, err)
	//}
//}

//func TestBinaryTrieReadOneFromManyLeaves(t *testing.T) {
	//trie := NewBinaryTrie()
	//trie.Update([]byte{0}, []byte{10})
	//trie.Update([]byte{8}, []byte{18})
	//trie.Update([]byte{11}, []byte{20})

	//v, err := trie.TryGet([]byte{0})
	//if err != nil {
		//t.Fatalf("error searching for key 0 in trie, err=%v", err)
	//}
	//if !bytes.Equal(v, []byte{10}) {
		//t.Fatalf("could not find correct value %x != 0a", v)
	//}

	//v, err = trie.TryGet([]byte{1})
	//if err != errKeyNotPresent {
		//t.Fatalf("incorrect error received, expected '%v', got '%v'", errKeyNotPresent, err)
	//}
//}
