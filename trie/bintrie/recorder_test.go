// Copyright 2026 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

func newRecorderTestTrie() *BinaryTrie {
	return &BinaryTrie{
		store:  newNodeStore(),
		tracer: trie.NewPrevalueTracer(),
	}
}

// TestRecorderCapturesAccountWrite verifies the recorder mirrors a single
// UpdateAccount call into the resulting GenesisAlloc.
func TestRecorderCapturesAccountWrite(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	acc := &types.StateAccount{
		Nonce:    7,
		Balance:  uint256.NewInt(42),
		CodeHash: common.HexToHash("aa").Bytes(),
	}
	if err := tr.UpdateAccount(addr, acc, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	alloc := rec.Alloc()
	got, ok := alloc[addr]
	if !ok {
		t.Fatalf("address %x missing from alloc", addr)
	}
	if got.Nonce != 7 {
		t.Errorf("nonce: got %d want 7", got.Nonce)
	}
	if got.Balance == nil || got.Balance.Uint64() != 42 {
		t.Errorf("balance: got %v want 42", got.Balance)
	}
}

// TestRecorderStorageRoundTrip verifies that storage writes are recorded with
// the original (unhashed) slot keys.
func TestRecorderStorageRoundTrip(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	acc := &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1)}
	if err := tr.UpdateAccount(addr, acc, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	slot := common.HexToHash("00000000000000000000000000000000000000000000000000000000000000ff")
	value := common.HexToHash("00000000000000000000000000000000000000000000000000000000deadbeef")
	if err := tr.UpdateStorage(addr, slot[:], value[:]); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}

	alloc := rec.Alloc()
	got := alloc[addr]
	if got.Storage == nil {
		t.Fatalf("storage map nil")
	}
	if got.Storage[slot] != value {
		t.Errorf("storage[%x] = %x, want %x", slot, got.Storage[slot], value)
	}
}

// TestRecorderDeleteStorage verifies that writing a zero value (or calling
// DeleteStorage) removes the slot from the recorded set, matching MPT-dump
// semantics.
func TestRecorderDeleteStorage(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addr := common.HexToAddress("0x3333333333333333333333333333333333333333")
	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1)}, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	slotKept := common.HexToHash("00000000000000000000000000000000000000000000000000000000000000ff")
	slotGone := common.HexToHash("0100000000000000000000000000000000000000000000000000000000000001")
	if err := tr.UpdateStorage(addr, slotKept[:], common.HexToHash("01").Bytes()); err != nil {
		t.Fatalf("UpdateStorage(kept): %v", err)
	}
	if err := tr.UpdateStorage(addr, slotGone[:], common.HexToHash("02").Bytes()); err != nil {
		t.Fatalf("UpdateStorage(gone): %v", err)
	}
	if err := tr.DeleteStorage(addr, slotGone[:]); err != nil {
		t.Fatalf("DeleteStorage: %v", err)
	}

	alloc := rec.Alloc()
	if _, exists := alloc[addr].Storage[slotGone]; exists {
		t.Errorf("deleted slot still present")
	}
	if _, exists := alloc[addr].Storage[slotKept]; !exists {
		t.Errorf("retained slot missing")
	}
}

// TestRecorderDeleteAccount verifies an account removed via DeleteAccount
// disappears from the alloc entirely, including its storage.
func TestRecorderDeleteAccount(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addrKept := common.HexToAddress("0x4444444444444444444444444444444444444444")
	addrGone := common.HexToAddress("0x5555555555555555555555555555555555555555")
	for _, a := range []common.Address{addrKept, addrGone} {
		if err := tr.UpdateAccount(a, &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1)}, 0); err != nil {
			t.Fatalf("UpdateAccount(%x): %v", a, err)
		}
	}
	slot := common.HexToHash("0100000000000000000000000000000000000000000000000000000000000001")
	if err := tr.UpdateStorage(addrGone, slot[:], common.HexToHash("0a").Bytes()); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}
	if err := tr.DeleteAccount(addrGone); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	alloc := rec.Alloc()
	if _, exists := alloc[addrGone]; exists {
		t.Errorf("deleted account still present in alloc")
	}
	if _, exists := alloc[addrKept]; !exists {
		t.Errorf("untouched account missing from alloc")
	}
}

// TestRecorderDeleteThenRecreate verifies that recreating an account after a
// delete starts from a fresh entry — old storage and code do not bleed into
// the new account.
func TestRecorderDeleteThenRecreate(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addr := common.HexToAddress("0x6666666666666666666666666666666666666666")
	slot := common.HexToHash("0100000000000000000000000000000000000000000000000000000000000001")

	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(100)}, 0); err != nil {
		t.Fatalf("UpdateAccount #1: %v", err)
	}
	if err := tr.UpdateStorage(addr, slot[:], common.HexToHash("0a").Bytes()); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}
	if err := tr.UpdateContractCode(addr, common.Hash{}, []byte{0x60, 0x00}); err != nil {
		t.Fatalf("UpdateContractCode: %v", err)
	}
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 7, Balance: uint256.NewInt(9999)}, 0); err != nil {
		t.Fatalf("UpdateAccount #2: %v", err)
	}

	alloc := rec.Alloc()
	got := alloc[addr]
	if got.Nonce != 7 {
		t.Errorf("nonce after recreate: got %d want 7", got.Nonce)
	}
	if got.Balance == nil || got.Balance.Uint64() != 9999 {
		t.Errorf("balance after recreate: got %v want 9999", got.Balance)
	}
	if len(got.Storage) != 0 {
		t.Errorf("recreated account has stale storage: %v", got.Storage)
	}
	if len(got.Code) != 0 {
		t.Errorf("recreated account has stale code: %x", got.Code)
	}
}

// TestRecorderCodeOverwrite verifies that a second UpdateContractCode call
// replaces the previously-recorded code.
func TestRecorderCodeOverwrite(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addr := common.HexToAddress("0x7777777777777777777777777777777777777777")
	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1)}, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	first := []byte{0x60, 0x01}
	second := []byte{0x60, 0x02, 0x60, 0x03}
	if err := tr.UpdateContractCode(addr, common.Hash{}, first); err != nil {
		t.Fatalf("UpdateContractCode #1: %v", err)
	}
	if err := tr.UpdateContractCode(addr, common.Hash{}, second); err != nil {
		t.Fatalf("UpdateContractCode #2: %v", err)
	}

	alloc := rec.Alloc()
	if !bytes.Equal(alloc[addr].Code, second) {
		t.Errorf("code: got %x want %x", alloc[addr].Code, second)
	}
}

// TestRecorderPartialUpdatePreservesStorage verifies that a nonce/balance
// update on an account does not wipe its previously-recorded storage or code.
func TestRecorderPartialUpdatePreservesStorage(t *testing.T) {
	tr := newRecorderTestTrie()
	rec := NewRecorder()
	tr.SetRecorder(rec)

	addr := common.HexToAddress("0x8888888888888888888888888888888888888888")
	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1)}, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	slot := common.HexToHash("0100000000000000000000000000000000000000000000000000000000000001")
	if err := tr.UpdateStorage(addr, slot[:], common.HexToHash("0a").Bytes()); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}
	code := []byte{0x60, 0x05}
	if err := tr.UpdateContractCode(addr, common.Hash{}, code); err != nil {
		t.Fatalf("UpdateContractCode: %v", err)
	}
	// Bump nonce only; storage and code should survive.
	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 2, Balance: uint256.NewInt(1)}, len(code)); err != nil {
		t.Fatalf("UpdateAccount #2: %v", err)
	}

	alloc := rec.Alloc()
	got := alloc[addr]
	if got.Nonce != 2 {
		t.Errorf("nonce: got %d want 2", got.Nonce)
	}
	if got.Storage[slot] == (common.Hash{}) {
		t.Errorf("storage was cleared by partial update")
	}
	if !bytes.Equal(got.Code, code) {
		t.Errorf("code was cleared by partial update")
	}
}

// TestRecorderDisabledByDefault confirms that without SetRecorder the trie
// performs no recording (sanity check that hooks are gated).
func TestRecorderDisabledByDefault(t *testing.T) {
	tr := newRecorderTestTrie()
	if tr.Recorder() != nil {
		t.Fatal("Recorder() should be nil before SetRecorder")
	}
	addr := common.HexToAddress("0x9999999999999999999999999999999999999999")
	if err := tr.UpdateAccount(addr, &types.StateAccount{Nonce: 1, Balance: uint256.NewInt(1)}, 0); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
}
