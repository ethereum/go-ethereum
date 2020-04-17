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

func TestBinaryLeafReadEmpty(t *testing.T) {
	trie, err := NewBinary(nil)
	if err != nil {
		t.Fatalf("error creating binary trie: %v", err)
	}

	_, err = trie.TryGet(common.FromHex("00"))
	if err == nil {
		t.Fatalf("should have returned an error trying to get from an empty binry trie, err=%v", err)
	}
}

func TestBinaryReadPrefix(t *testing.T) {
	trieLeaf := &BinaryTrie{
		prefix:   []byte("croissants"),
		startBit: 0,
		endBit:   8 * len("croissants"),
		left:     nil,
		right:    nil,
		value:    []byte("baguette"),
	}

	res, err := trieLeaf.TryGet([]byte("croissants"))
	if !bytes.Equal(res, []byte("baguette")) {
		t.Fatalf("should have returned an error trying to get from an empty binry trie, err=%v", err)
	}

	trieExtLeaf := &BinaryTrie{
		prefix:   []byte("crois"),
		startBit: 0,
		endBit:   8 * len("crois"),
		left: &BinaryTrie{
			prefix:   []byte("sants"),
			startBit: 1,
			endBit:   8 * len("sants"),
			value:    []byte("baguette"),
			left:     nil,
			right:    nil,
		},
		right: nil,
	}

	res, err = trieExtLeaf.TryGet([]byte("croissants"))
	if !bytes.Equal(res, []byte("baguette")) {
		t.Fatalf("should not have returned err=%v", err)
	}

	// Same test as above but the break isn't on a byte boundary
	trieExtLeaf = &BinaryTrie{
		prefix:   []byte("crois"),
		startBit: 0,
		endBit:   8*len("crois") - 3,
		left: &BinaryTrie{
			prefix:   []byte("ssants"),
			startBit: 6,
			endBit:   8 * len("ssants"),
			value:    []byte("baguette"),
			left:     nil,
			right:    nil,
		},
		right: nil,
	}

	res, err = trieExtLeaf.TryGet([]byte("croissants"))
	if !bytes.Equal(res, []byte("baguette")) {
		t.Fatalf("should not have returned err=%v", err)
	}
}

func TestBinaryLeafInsert(t *testing.T) {
	trie, err := NewBinary(nil)
	if err != nil {
		t.Fatalf("error creating binary trie: %v", err)
	}

	err = trie.TryUpdate(common.FromHex("00"), common.FromHex("00"))
	if err != nil {
		t.Fatalf("could not insert (0x00, 0x00) into an empty binary trie, err=%v", err)
	}

}

func TestBinaryLeafInsertRead(t *testing.T) {
	trie, err := NewBinary(nil)
	if err != nil {
		t.Fatalf("error creating binary trie: %v", err)
	}

	err = trie.TryUpdate(common.FromHex("00"), common.FromHex("01"))
	if err != nil {
		t.Fatalf("could not insert (0x00, 0x01) into an empty binary trie, err=%v", err)
	}

	v, err := trie.TryGet(common.FromHex("00"))
	if err != nil {
		t.Fatalf("could not read data back from simple binary trie, err=%v", err)
	}

	if !bytes.Equal(v, common.FromHex("01")) {
		t.Fatalf("Invalid value read from the binary trie: %s != %s", common.ToHex(v), "01")
	}
}

func TestBinaryForkInsertRead(t *testing.T) {
	trie, err := NewBinary(nil)
	if err != nil {
		t.Fatalf("error creating binary trie: %v", err)
	}

	for i := byte(0); i < 10; i++ {
		err = trie.TryUpdate([]byte{i}, common.FromHex("01"))
		if err != nil {
			t.Fatalf("could not insert (%#x, 0x01) into an empty binary trie, err=%v", i, err)
		}
	}

	v, err := trie.TryGet([]byte{9})
	if err != nil {
		t.Fatalf("could not read data back from simple binary trie, err=%v", err)
	}

	if !bytes.Equal(v, common.FromHex("01")) {
		t.Fatalf("Invalid value read from the binary trie: %s != %s", common.ToHex(v), "01")
	}

}

func TestBinaryInsertLeftRight(t *testing.T) {
	trie, err := NewBinary(nil)
	if err != nil {
		t.Fatalf("error creating binary trie: %v", err)
	}

	trie.TryUpdate([]byte{0}, []byte{0})
	trie.TryUpdate([]byte{128}, []byte{1})

	// Trie is expected to look like this:
	//         /\
	//        / /
	//       / /
	//      / /
	//     / /
	//    / /
	//   / /
	//  / /

	// Check there is a left branch
	if trie.left == nil {
		t.Fatal("empty left branch")
	}

	// Check that the left branch has already been hashed
	if _, ok := trie.left.(hashBinaryNode); !ok {
		t.Fatalf("left branch should have been hashed!")
	}

	// Check there is a right branch
	if trie.right == nil {
		t.Fatal("empty right branch")
	}

	// Check that the right branch has only lefts after the
	// first right.
	for i, tr := 1, trie.right; i < 8; i++ {
		if tr == nil {
			t.Fatal("invalid trie structure")
		}
		tr = tr.(*BinaryTrie).left
	}
}

func TestPrefixBitLen(t *testing.T) {
	btrie := new(BinaryTrie)

	got := btrie.getPrefixLen()
	if got != 0 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 0)
	}

	btrie.prefix = []byte("croissants")
	got = btrie.getPrefixLen()
	if got != 0 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 0)
	}

	btrie.endBit = 5
	got = btrie.getPrefixLen()
	if got != 5 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 5)
	}

	btrie.endBit = 12
	got = btrie.getPrefixLen()
	if got != 12 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 12)
	}

	btrie.endBit = 27
	got = btrie.getPrefixLen()
	if got != 27 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 27)
	}

	btrie.startBit = 25
	got = btrie.getPrefixLen()
	if got != 2 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 2)
	}

	btrie.endBit = 33
	got = btrie.getPrefixLen()
	if got != 8 {
		t.Fatalf("Invalid prefix length, got %d != exp %d", got, 8)
	}
}

func TestPrefixBitAccess(t *testing.T) {
	btrie := new(BinaryTrie)
	btrie.prefix = []byte{0x55, 0x55}
	btrie.startBit = 0
	btrie.endBit = 15

	for i := 0; i < btrie.getPrefixLen(); i += 2 {
		if btrie.getPrefixBit(i) != false {
			t.Fatal("Got the wrong bit value")
		}
		if btrie.getPrefixBit(i+1) != true {
			t.Fatal("Got the wrong bit value")
		}
	}
}
