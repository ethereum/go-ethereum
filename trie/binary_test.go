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

	for i := byte(0); i < 10; i++ {
		v, err := trie.TryGet([]byte{i})
		if err != nil {
			t.Fatalf("could not read data back from simple binary trie, err=%v", err)
		}

		if !bytes.Equal(v, common.FromHex("01")) {
			t.Fatalf("Invalid value read from the binary trie: %s != %s", common.ToHex(v), "01")
		}
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
