// Copyright 2025 go-ethereum Authors
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

package bintrie

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

var (
	zeroKey  = [HashSize]byte{}
	oneKey   = common.HexToHash("0101010101010101010101010101010101010101010101010101010101010101")
	twoKey   = common.HexToHash("0202020202020202020202020202020202020202020202020202020202020202")
	threeKey = common.HexToHash("0303030303030303030303030303030303030303030303030303030303030303")
	fourKey  = common.HexToHash("0404040404040404040404040404040404040404040404040404040404040404")
	ffKey    = common.HexToHash("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

func TestSingleEntry(t *testing.T) {
	tree := NewBinaryNode()
	tree, err := tree.Insert(zeroKey[:], oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tree.GetHeight() != 1 {
		t.Fatal("invalid depth")
	}
	expected := common.HexToHash("aab1060e04cb4f5dc6f697ae93156a95714debbf77d54238766adc5709282b6f")
	got := tree.Hash()
	if got != expected {
		t.Fatalf("invalid tree root, got %x, want %x", got, expected)
	}
}

func TestTwoEntriesDiffFirstBit(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	tree, err = tree.Insert(zeroKey[:], oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(common.HexToHash("8000000000000000000000000000000000000000000000000000000000000000").Bytes(), twoKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tree.GetHeight() != 2 {
		t.Fatal("invalid height")
	}
	if tree.Hash() != common.HexToHash("dfc69c94013a8b3c65395625a719a87534a7cfd38719251ad8c8ea7fe79f065e") {
		t.Fatal("invalid tree root")
	}
}

func TestOneStemColocatedValues(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	tree, err = tree.Insert(common.HexToHash("0000000000000000000000000000000000000000000000000000000000000003").Bytes(), oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(common.HexToHash("0000000000000000000000000000000000000000000000000000000000000004").Bytes(), twoKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(common.HexToHash("0000000000000000000000000000000000000000000000000000000000000009").Bytes(), threeKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(common.HexToHash("00000000000000000000000000000000000000000000000000000000000000FF").Bytes(), fourKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tree.GetHeight() != 1 {
		t.Fatal("invalid height")
	}
}

func TestTwoStemColocatedValues(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	// stem: 0...0
	tree, err = tree.Insert(common.HexToHash("0000000000000000000000000000000000000000000000000000000000000003").Bytes(), oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(common.HexToHash("0000000000000000000000000000000000000000000000000000000000000004").Bytes(), twoKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	// stem: 10...0
	tree, err = tree.Insert(common.HexToHash("8000000000000000000000000000000000000000000000000000000000000003").Bytes(), oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(common.HexToHash("8000000000000000000000000000000000000000000000000000000000000004").Bytes(), twoKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tree.GetHeight() != 2 {
		t.Fatal("invalid height")
	}
}

func TestTwoKeysMatchFirst42Bits(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	// key1 and key 2 have the same prefix of 42 bits (b0*42+b1+b1) and differ after.
	key1 := common.HexToHash("0000000000C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0C0").Bytes()
	key2 := common.HexToHash("0000000000E00000000000000000000000000000000000000000000000000000").Bytes()
	tree, err = tree.Insert(key1, oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(key2, twoKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tree.GetHeight() != 1+42+1 {
		t.Fatal("invalid height")
	}
}
func TestInsertDuplicateKey(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	tree, err = tree.Insert(oneKey[:], oneKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tree, err = tree.Insert(oneKey[:], twoKey[:], nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tree.GetHeight() != 1 {
		t.Fatal("invalid height")
	}
	// Verify that the value is updated
	if !bytes.Equal(tree.(*StemNode).Values[1], twoKey[:]) {
		t.Fatal("invalid height")
	}
}
func TestLargeNumberOfEntries(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	for i := range StemNodeWidth {
		var key [HashSize]byte
		key[0] = byte(i)
		tree, err = tree.Insert(key[:], ffKey[:], nil, 0)
		if err != nil {
			t.Fatal(err)
		}
	}
	height := tree.GetHeight()
	if height != 1+8 {
		t.Fatalf("invalid height, wanted %d, got %d", 1+8, height)
	}
}

func TestMerkleizeMultipleEntries(t *testing.T) {
	var err error
	tree := NewBinaryNode()
	keys := [][]byte{
		zeroKey[:],
		common.HexToHash("8000000000000000000000000000000000000000000000000000000000000000").Bytes(),
		common.HexToHash("0100000000000000000000000000000000000000000000000000000000000000").Bytes(),
		common.HexToHash("8100000000000000000000000000000000000000000000000000000000000000").Bytes(),
	}
	for i, key := range keys {
		var v [HashSize]byte
		binary.LittleEndian.PutUint64(v[:8], uint64(i))
		tree, err = tree.Insert(key, v[:], nil, 0)
		if err != nil {
			t.Fatal(err)
		}
	}
	got := tree.Hash()
	expected := common.HexToHash("9317155862f7a3867660ddd0966ff799a3d16aa4df1e70a7516eaa4a675191b5")
	if got != expected {
		t.Fatalf("invalid root, expected=%x, got = %x", expected, got)
	}
}

// TestStorageRoundTrip verifies that GetStorage and DeleteStorage use the same
// key mapping as UpdateStorage (GetBinaryTreeKeyStorageSlot). This is a regression
// test: previously GetStorage and DeleteStorage used GetBinaryTreeKey directly,
// which produced different tree keys and broke the read/delete path.
func TestStorageRoundTrip(t *testing.T) {
	tracer := trie.NewPrevalueTracer()
	tr := &BinaryTrie{
		root:   NewBinaryNode(),
		tracer: tracer,
	}
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	// Create an account first so the root becomes an InternalNode,
	// which is the realistic state when storage operations happen.
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(1000),
		CodeHash: common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
	}
	if err := tr.UpdateAccount(addr, acc, 0); err != nil {
		t.Fatalf("UpdateAccount error: %v", err)
	}

	// Test main storage slots (key[31] >= 64 or key[:31] != 0).
	// These produce a different stem than the account data, so after
	// UpdateAccount + UpdateStorage the root is an InternalNode.
	// Note: header slots (key[31] < 64, key[:31] == 0) share the same
	// stem as account data and are covered by GetAccount/UpdateAccount path.
	slots := []common.Hash{
		common.HexToHash("00000000000000000000000000000000000000000000000000000000000000FF"), // main storage (slot 255)
		common.HexToHash("0100000000000000000000000000000000000000000000000000000000000001"), // main storage (non-zero prefix)
	}
	val := common.TrimLeftZeroes(common.HexToHash("00000000000000000000000000000000000000000000000000000000deadbeef").Bytes())

	for _, slot := range slots {
		// Write
		if err := tr.UpdateStorage(addr, slot[:], val); err != nil {
			t.Fatalf("UpdateStorage(%x) error: %v", slot, err)
		}
		// Read back
		got, err := tr.GetStorage(addr, slot[:])
		if err != nil {
			t.Fatalf("GetStorage(%x) error: %v", slot, err)
		}
		if len(got) == 0 {
			t.Fatalf("GetStorage(%x) returned empty, expected value", slot)
		}
		// Verify value (right-justified in 32 bytes)
		var expected [HashSize]byte
		copy(expected[HashSize-len(val):], val)
		if !bytes.Equal(got, expected[:]) {
			t.Fatalf("GetStorage(%x) = %x, want %x", slot, got, expected)
		}
		// Delete
		if err := tr.DeleteStorage(addr, slot[:]); err != nil {
			t.Fatalf("DeleteStorage(%x) error: %v", slot, err)
		}
		// Verify deleted (should read as zero, not the old value)
		got, err = tr.GetStorage(addr, slot[:])
		if err != nil {
			t.Fatalf("GetStorage(%x) after delete error: %v", slot, err)
		}
		if len(got) > 0 && !bytes.Equal(got, zero[:]) {
			t.Fatalf("GetStorage(%x) after delete = %x, expected zero", slot, got)
		}
	}
}

func TestBinaryTrieWitness(t *testing.T) {
	tracer := trie.NewPrevalueTracer()

	tr := &BinaryTrie{
		root:   NewBinaryNode(),
		tracer: tracer,
	}
	if w := tr.Witness(); len(w) != 0 {
		t.Fatal("expected empty witness for fresh trie")
	}

	tracer.Put([]byte("path1"), []byte("blob1"))
	tracer.Put([]byte("path2"), []byte("blob2"))

	witness := tr.Witness()
	if len(witness) != 2 {
		t.Fatalf("expected 2 witness entries, got %d", len(witness))
	}
	if !bytes.Equal(witness[string([]byte("path1"))], []byte("blob1")) {
		t.Fatal("unexpected witness value for path1")
	}
	if !bytes.Equal(witness[string([]byte("path2"))], []byte("blob2")) {
		t.Fatal("unexpected witness value for path2")
	}
}
