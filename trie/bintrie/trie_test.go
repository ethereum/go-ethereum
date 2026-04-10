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

// newEmptyTestTrie creates a fresh BinaryTrie with an empty root and a
// default prevalue tracer. Use this for tests that populate the trie
// incrementally via Update*; for tests that want a pre-populated trie with
// a fixed entry set, use makeTrie (in iterator_test.go) instead.
func newEmptyTestTrie(t *testing.T) *BinaryTrie {
	t.Helper()
	return &BinaryTrie{
		root:   NewBinaryNode(),
		tracer: trie.NewPrevalueTracer(),
	}
}

// makeAccount constructs a StateAccount with the given fields. The Root is
// zeroed out because the bintrie has no per-account storage root.
func makeAccount(nonce uint64, balance uint64, codeHash common.Hash) *types.StateAccount {
	return &types.StateAccount{
		Nonce:    nonce,
		Balance:  uint256.NewInt(balance),
		CodeHash: codeHash.Bytes(),
	}
}

// TestDeleteAccountRoundTrip verifies the basic delete path: create an
// account, read it back, delete it, confirm subsequent reads return nil.
// Regression test for the no-op DeleteAccount bug where the deletion was
// silently ignored and the old values remained in the trie.
func TestDeleteAccountRoundTrip(t *testing.T) {
	tr := newEmptyTestTrie(t)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	codeHash := common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")

	// Create: write account, verify round-trip.
	acc := makeAccount(42, 1000, codeHash)
	if err := tr.UpdateAccount(addr, acc, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	got, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount returned nil after UpdateAccount")
	}
	if got.Nonce != 42 {
		t.Fatalf("Nonce: got %d, want 42", got.Nonce)
	}
	if got.Balance.Uint64() != 1000 {
		t.Fatalf("Balance: got %s, want 1000", got.Balance)
	}
	if !bytes.Equal(got.CodeHash, codeHash[:]) {
		t.Fatalf("CodeHash: got %x, want %x", got.CodeHash, codeHash)
	}

	// Delete: verify GetAccount returns nil afterwards.
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	got, err = tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("GetAccount after delete: %v", err)
	}
	if got != nil {
		t.Fatalf("GetAccount after delete: got %+v, want nil", got)
	}
}

// TestDeleteAccountOnMissingAccount verifies that deleting an account that
// was never created does not error and subsequent reads still return nil.
func TestDeleteAccountOnMissingAccount(t *testing.T) {
	tr := newEmptyTestTrie(t)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	// Delete without any prior create. Should not panic or error on an
	// empty root, and GetAccount should still return nil.
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatalf("DeleteAccount on empty trie: %v", err)
	}
	got, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("GetAccount after delete on empty trie: %v", err)
	}
	if got != nil {
		t.Fatalf("GetAccount on deleted missing account: got %+v, want nil", got)
	}
}

// TestDeleteAccountPreservesOtherAccounts verifies that deleting one account
// does not affect accounts at different stems.
func TestDeleteAccountPreservesOtherAccounts(t *testing.T) {
	tr := newEmptyTestTrie(t)
	addrA := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	addrB := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	codeHashA := common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	codeHashB := common.HexToHash("f0f1f2f3f4f5f6f7f8f9fafbfcfdfeff0102030405060708090a0b0c0d0e0f10")

	// Create two distinct accounts.
	if err := tr.UpdateAccount(addrA, makeAccount(1, 100, codeHashA), 0); err != nil {
		t.Fatalf("UpdateAccount(A): %v", err)
	}
	if err := tr.UpdateAccount(addrB, makeAccount(2, 200, codeHashB), 0); err != nil {
		t.Fatalf("UpdateAccount(B): %v", err)
	}

	// Delete A.
	if err := tr.DeleteAccount(addrA); err != nil {
		t.Fatalf("DeleteAccount(A): %v", err)
	}

	// A should be gone.
	if got, err := tr.GetAccount(addrA); err != nil {
		t.Fatalf("GetAccount(A): %v", err)
	} else if got != nil {
		t.Fatalf("GetAccount(A) after delete: got %+v, want nil", got)
	}

	// B should still be readable with its original values.
	got, err := tr.GetAccount(addrB)
	if err != nil {
		t.Fatalf("GetAccount(B): %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount(B) returned nil after unrelated delete")
	}
	if got.Nonce != 2 {
		t.Fatalf("Account B Nonce: got %d, want 2", got.Nonce)
	}
	if got.Balance.Uint64() != 200 {
		t.Fatalf("Account B Balance: got %s, want 200", got.Balance)
	}
	if !bytes.Equal(got.CodeHash, codeHashB[:]) {
		t.Fatalf("Account B CodeHash: got %x, want %x", got.CodeHash, codeHashB)
	}
}

// TestDeleteAccountThenRecreate verifies that an account can be deleted and
// then recreated with different values; the second read must return the new
// values, not the stale ones from before deletion.
func TestDeleteAccountThenRecreate(t *testing.T) {
	tr := newEmptyTestTrie(t)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	codeHash1 := common.HexToHash("1111111111111111111111111111111111111111111111111111111111111111")
	codeHash2 := common.HexToHash("2222222222222222222222222222222222222222222222222222222222222222")

	// Create.
	if err := tr.UpdateAccount(addr, makeAccount(1, 100, codeHash1), 0); err != nil {
		t.Fatalf("UpdateAccount #1: %v", err)
	}
	// Delete.
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	// Recreate with new values.
	if err := tr.UpdateAccount(addr, makeAccount(7, 9999, codeHash2), 0); err != nil {
		t.Fatalf("UpdateAccount #2: %v", err)
	}
	// Read: must observe the new values, not the originals.
	got, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount returned nil after recreate")
	}
	if got.Nonce != 7 {
		t.Fatalf("Nonce: got %d, want 7", got.Nonce)
	}
	if got.Balance.Uint64() != 9999 {
		t.Fatalf("Balance: got %s, want 9999", got.Balance)
	}
	if !bytes.Equal(got.CodeHash, codeHash2[:]) {
		t.Fatalf("CodeHash: got %x, want %x", got.CodeHash, codeHash2)
	}
}

// TestDeleteAccountDoesNotAffectMainStorage verifies that DeleteAccount only
// clears the account's BasicData and CodeHash, leaving main storage slots
// untouched. Main storage slots live at different stems entirely (their
// keys route through the non-header branch in GetBinaryTreeKeyStorageSlot),
// so this test exercises the inter-stem isolation. Header-range storage
// slots share the same stem and are covered separately by
// TestDeleteAccountPreservesHeaderStorage.
//
// Wiping storage on self-destruct is a separate concern handled at the
// StateDB level.
func TestDeleteAccountDoesNotAffectMainStorage(t *testing.T) {
	tr := newEmptyTestTrie(t)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	codeHash := common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")

	// Create account.
	if err := tr.UpdateAccount(addr, makeAccount(1, 100, codeHash), 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	// Write a main storage slot — i.e. key[31] >= 64 or key[:31] != 0 — so
	// it lives at a different stem from the account header.
	slot := common.HexToHash("0000000000000000000000000000000000000000000000000000000000000080")
	value := common.TrimLeftZeroes(common.HexToHash("00000000000000000000000000000000000000000000000000000000deadbeef").Bytes())
	if err := tr.UpdateStorage(addr, slot[:], value); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}

	// Delete the account.
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// Account should be absent.
	got, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("GetAccount after delete: %v", err)
	}
	if got != nil {
		t.Fatalf("GetAccount after delete: got %+v, want nil", got)
	}

	// Main storage slot should still be readable — DeleteAccount must not
	// have touched it.
	stored, err := tr.GetStorage(addr, slot[:])
	if err != nil {
		t.Fatalf("GetStorage after DeleteAccount: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("main storage slot was wiped by DeleteAccount, expected it to survive")
	}
	var expected [HashSize]byte
	copy(expected[HashSize-len(value):], value)
	if !bytes.Equal(stored, expected[:]) {
		t.Fatalf("main storage slot: got %x, want %x", stored, expected)
	}
}

// TestDeleteAccountPreservesHeaderStorage verifies that DeleteAccount does
// not clobber header-range storage slots (key[31] < 64), which live at the
// SAME stem as BasicData/CodeHash but at offsets 64-127. The safety here
// relies on StemNode.InsertValuesAtStem treating nil entries in the values
// slice as "do not overwrite"; this test pins that invariant so a future
// change cannot silently corrupt slots 0-63 of any contract.
func TestDeleteAccountPreservesHeaderStorage(t *testing.T) {
	tr := newEmptyTestTrie(t)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	codeHash := common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")

	// Create account.
	if err := tr.UpdateAccount(addr, makeAccount(1, 100, codeHash), 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	// Create a second, unrelated account so the root promotes from StemNode
	// to InternalNode. BinaryTrie.GetStorage walks via root.Get, which is
	// only implemented on InternalNode/Empty — calling it with a StemNode
	// root panics. The existing main-storage test gets away with this because
	// the main-storage slot lands on a separate stem and forces the same
	// promotion implicitly; here we want a same-stem header slot, so the
	// promotion has to come from a second account.
	other := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	if err := tr.UpdateAccount(other, makeAccount(0, 0, common.Hash{}), 0); err != nil {
		t.Fatalf("UpdateAccount(other): %v", err)
	}

	// Write a header-range storage slot — key[:31] == 0 and key[31] < 64
	// — which routes through the header branch in GetBinaryTreeKeyStorageSlot
	// and lands on the same stem as BasicData/CodeHash.
	var slot [HashSize]byte
	slot[31] = 5
	value := []byte{0xde, 0xad, 0xbe, 0xef}
	if err := tr.UpdateStorage(addr, slot[:], value); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}

	// Delete the account.
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// Account metadata should be gone.
	got, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("GetAccount after delete: %v", err)
	}
	if got != nil {
		t.Fatalf("GetAccount after delete: got %+v, want nil", got)
	}

	// Header storage slot must survive — DeleteAccount only writes offsets
	// BasicDataLeafKey, CodeHashLeafKey, and accountDeletedMarkerKey, leaving
	// the header-storage offsets (64-127) untouched.
	stored, err := tr.GetStorage(addr, slot[:])
	if err != nil {
		t.Fatalf("GetStorage after DeleteAccount: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("header storage slot was wiped by DeleteAccount, expected it to survive")
	}
	var expected [HashSize]byte
	copy(expected[HashSize-len(value):], value)
	if !bytes.Equal(stored, expected[:]) {
		t.Fatalf("header storage slot: got %x, want %x", stored, expected)
	}
}

func TestDeleteAccountHashIsDeterministic(t *testing.T) {
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	codeHash := common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	acc := makeAccount(42, 1000, codeHash)

	run := func() common.Hash {
		tr := newEmptyTestTrie(t)
		if err := tr.UpdateAccount(addr, acc, 0); err != nil {
			t.Fatalf("UpdateAccount: %v", err)
		}
		if err := tr.DeleteAccount(addr); err != nil {
			t.Fatalf("DeleteAccount: %v", err)
		}
		return tr.Hash()
	}

	first := run()
	second := run()
	if first != second {
		t.Fatalf("non-deterministic root after Update+Delete: first=%x second=%x", first, second)
	}

	empty := newEmptyTestTrie(t).Hash()
	if first == empty {
		t.Fatalf("post-delete root unexpectedly equals empty-trie root %x", empty)
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
