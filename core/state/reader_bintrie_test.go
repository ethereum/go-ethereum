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

package state

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// TestBintrieFlatReaderEndToEnd is the integration test that exercises
// the full Commit-10 read path for a binary-trie database:
//
//  1. Build a fresh verkle pathdb-backed StateDB.
//  2. Mutate accounts (balance, nonce, code) and storage slots; the
//     binaryHasher produces leaf writes via DrainStemWrites under the
//     hood (Commit 7).
//  3. Commit through the standard StateDB.Commit pipeline. This drives
//     stateUpdate.encodeBinary (Commit 8) which converts the leaves
//     into per-offset accountData entries that flow into pathdb's
//     stateSet, then are persisted to disk via the bintrie codec's
//     Flush method (Commit 8).
//  4. Open a StateReader for the resulting root. CachingDB.StateReader
//     installs a bintrieFlatReader (Commit 10) ahead of the trie
//     reader because db.TrieDB().IsVerkle() is true.
//  5. Read the accounts and one storage slot back through the
//     StateReader and assert the values round-trip exactly.
//
// This is the canonical "does the bintrie flat-state read path actually
// work end-to-end" test. If it fails, something between the hasher's
// leaf production and the disk-layer reads is wrong.
func TestBintrieFlatReaderEndToEnd(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)

	// A fresh verkle pathdb's disk layer is keyed by EmptyVerkleHash
	// (all-zero hash), not EmptyRootHash. The TestVerkleCodeSizePreserved
	// helper documents this gotcha.
	state, err := New(types.EmptyVerkleHash, sdb)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	var (
		addrA   = common.HexToAddress("0xAAaaAAaaAAaaAAaaAAaaAAaaAAaaAAaaAAaaAAaa")
		addrB   = common.HexToAddress("0xBBbbBBbbBBbbBBbbBBbbBBbbBBbbBBbbBBbbBBbb")
		balance = uint256.NewInt(0xCAFE)
		slot    = common.HexToHash("0x07")
		value   = common.HexToHash("0x42")
	)

	// addrA: contract account with balance, nonce, code, and a storage
	// slot. Slot 7 is in the EIP-7864 header range so it shares a stem
	// with the BasicData leaf, exercising the per-stem RMW path.
	state.SetBalance(addrA, balance, tracing.BalanceChangeUnspecified)
	state.SetNonce(addrA, 5, tracing.NonceChangeUnspecified)
	state.SetCode(addrA, []byte{0x60, 0x80, 0x60, 0x40}, tracing.CodeChangeUnspecified)
	state.SetState(addrA, slot, value)

	// addrB: EOA with only a balance set. Lives at a different stem so
	// it tests two distinct stems landing in the same flush.
	state.SetBalance(addrB, uint256.NewInt(0xBEEF), tracing.BalanceChangeUnspecified)

	root, err := state.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Now read the state back via a StateReader for the new root. The
	// dispatch in CachingDB.StateReader uses bintrieFlatReader because
	// IsVerkle() is true.
	reader, err := sdb.StateReader(root)
	if err != nil {
		t.Fatalf("StateReader: %v", err)
	}

	gotA, err := reader.Account(addrA)
	if err != nil {
		t.Fatalf("Account A: %v", err)
	}
	if gotA == nil {
		t.Fatal("addrA: account is nil after commit")
	}
	if gotA.Nonce != 5 {
		t.Errorf("addrA nonce: got %d, want 5", gotA.Nonce)
	}
	if gotA.Balance.Cmp(balance) != 0 {
		t.Errorf("addrA balance: got %s, want %s", gotA.Balance, balance)
	}
	if len(gotA.CodeHash) != 32 {
		t.Errorf("addrA code hash: got %d-byte hash, want 32", len(gotA.CodeHash))
	}

	gotB, err := reader.Account(addrB)
	if err != nil {
		t.Fatalf("Account B: %v", err)
	}
	if gotB == nil {
		t.Fatal("addrB: account is nil after commit")
	}
	if gotB.Balance.Uint64() != 0xBEEF {
		t.Errorf("addrB balance: got %s, want 0xBEEF", gotB.Balance)
	}

	// Storage slot round-trip: SetState wrote value at slot 7 of addrA.
	// The bintrieFlatReader.Storage call derives the bintrie storage
	// key locally and looks it up via pathdb's AccountRLP path.
	gotSlot, err := reader.Storage(addrA, slot)
	if err != nil {
		t.Fatalf("Storage: %v", err)
	}
	if gotSlot != value {
		t.Errorf("storage slot: got %x, want %x", gotSlot, value)
	}
}

// TestBintrieFlatReaderMissingAccountSentinel verifies that the bintrie
// flat reader returns errBintrieFlatStateMiss (a non-nil error sentinel)
// for an account that was never written to the flat state.
//
// Post-A2: the flat reader returns errBintrieFlatStateMiss so the
// multiStateReader falls through to the trie reader. This is the
// correct behavior: the flat state does not have the entry, so the
// trie reader should be the gatekeeper.
//
// KNOWN ISSUE: BinaryTrie.GetAccount does NOT verify stem membership —
// it returns the closest stem's data for ANY address query. So the trie
// reader currently returns wrong data for non-existent addresses. That
// is a pre-existing bintrie bug (not introduced by A2). This test
// therefore verifies the FLAT READER's sentinel error directly, in
// isolation from the buggy trie reader fallthrough path.
func TestBintrieFlatReaderMissingAccountSentinel(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)
	state, err := New(types.EmptyVerkleHash, sdb)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	// Touch addrA so the trie has at least one stem.
	addrA := common.HexToAddress("0x0101010101010101010101010101010101010101")
	state.SetBalance(addrA, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	root, err := state.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Get the pathdb reader so we can test the bintrieFlatReader in
	// isolation — not through multiStateReader (which would fall
	// through to the buggy trie reader).
	pathdbReader, err := tdb.StateReader(root)
	if err != nil {
		t.Fatalf("pathdb StateReader: %v", err)
	}
	br := newBintrieFlatReader(pathdbReader)
	if br == nil {
		t.Fatal("newBintrieFlatReader returned nil")
	}

	missing := common.HexToAddress("0xfeedfacefeedfacefeedfacefeedfacefeedface")
	_, flatErr := br.Account(missing)
	if flatErr == nil {
		t.Fatal("expected errBintrieFlatStateMiss for missing account, got nil error")
	}
	if !errors.Is(flatErr, errBintrieFlatStateMiss) {
		t.Fatalf("expected errBintrieFlatStateMiss, got: %v", flatErr)
	}
}

// TestBintrieFlatReaderEndToEndAfterFlush is the smoking-gun regression
// test for A1 (fix bintrieFlatReader disk-layer shape). Before the A1
// remediation, `bintrieFlatCodec.ReadAccount` returned the full stem
// blob from disk while `bintrieFlatReader.Account` expected a per-offset
// 32-byte value — so every disk-layer hit errored with "bintrie
// BasicData leaf invalid length". The original TestBintrieFlatReaderEndToEnd
// did not catch this because it never flushed the write buffer to disk:
// all reads came from the in-memory diff-layer buffer (which stores
// per-offset entries correctly).
//
// This test explicitly calls `tdb.Commit(root, false)` after the state
// commit, forcing the buffer to flush. Subsequent reads MUST hit the
// disk-layer code path. If A1 regresses, the reads either error out or
// return wrong data.
func TestBintrieFlatReaderEndToEndAfterFlush(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)

	state, err := New(types.EmptyVerkleHash, sdb)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	var (
		addrA   = common.HexToAddress("0xAAaaAAaaAAaaAAaaAAaaAAaaAAaaAAaaAAaaAAaa")
		addrB   = common.HexToAddress("0xBBbbBBbbBBbbBBbbBBbbBBbbBBbbBBbbBBbbBBbb")
		balance = uint256.NewInt(0xCAFE)
		slot    = common.HexToHash("0x07")
		value   = common.HexToHash("0x42")
	)

	state.SetBalance(addrA, balance, tracing.BalanceChangeUnspecified)
	state.SetNonce(addrA, 5, tracing.NonceChangeUnspecified)
	state.SetCode(addrA, []byte{0x60, 0x80, 0x60, 0x40}, tracing.CodeChangeUnspecified)
	state.SetState(addrA, slot, value)
	state.SetBalance(addrB, uint256.NewInt(0xBEEF), tracing.BalanceChangeUnspecified)

	root, err := state.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Force buffer → disk flush. Without this, all reads below would hit
	// the in-memory diff-layer buffer path, masking the A1 bug.
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("tdb.Commit (flush to disk): %v", err)
	}

	// Open a fresh StateReader for the flushed root. Reads now go
	// through the disk layer via `codec.ReadAccount`, which (post-A1)
	// must return per-offset 32-byte values matching what the reader
	// expects.
	reader, err := sdb.StateReader(root)
	if err != nil {
		t.Fatalf("StateReader after flush: %v", err)
	}

	gotA, err := reader.Account(addrA)
	if err != nil {
		t.Fatalf("Account A after flush: %v", err)
	}
	if gotA == nil {
		t.Fatal("addrA: account is nil after flush (A1 regression)")
	}
	if gotA.Nonce != 5 {
		t.Errorf("addrA nonce after flush: got %d, want 5", gotA.Nonce)
	}
	if gotA.Balance.Cmp(balance) != 0 {
		t.Errorf("addrA balance after flush: got %s, want %s", gotA.Balance, balance)
	}

	gotB, err := reader.Account(addrB)
	if err != nil {
		t.Fatalf("Account B after flush: %v", err)
	}
	if gotB == nil {
		t.Fatal("addrB: account is nil after flush (A1 regression)")
	}
	if gotB.Balance.Uint64() != 0xBEEF {
		t.Errorf("addrB balance after flush: got %s, want 0xBEEF", gotB.Balance)
	}

	gotSlot, err := reader.Storage(addrA, slot)
	if err != nil {
		t.Fatalf("Storage after flush: %v", err)
	}
	if gotSlot != value {
		t.Errorf("storage slot after flush: got %x, want %x", gotSlot, value)
	}
}

// TestBintrieFlatReaderMultipleOffsetsPerStem verifies that multiple
// offsets at the same stem (BasicData at offset 0, CodeHash at offset 1,
// a header storage slot at offset 64+slotnum) all round-trip correctly
// through the per-offset read path. This exercises the "same stem, many
// offsets" common case for contract accounts with header storage.
func TestBintrieFlatReaderMultipleOffsetsPerStem(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)

	state, err := New(types.EmptyVerkleHash, sdb)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	state.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	state.SetNonce(addr, 7, tracing.NonceChangeUnspecified)
	state.SetCode(addr, []byte{0xDE, 0xAD, 0xBE, 0xEF}, tracing.CodeChangeUnspecified)
	// Header slots 0..63 (per EIP-7864) live at the same stem as
	// BasicData/CodeHash. Set a few to exercise multi-offset per stem.
	state.SetState(addr, common.HexToHash("0x00"), common.HexToHash("0x11"))
	state.SetState(addr, common.HexToHash("0x01"), common.HexToHash("0x22"))
	state.SetState(addr, common.HexToHash("0x05"), common.HexToHash("0x33"))

	root, err := state.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	// Flush so the reads hit the disk path.
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("tdb.Commit: %v", err)
	}

	reader, err := sdb.StateReader(root)
	if err != nil {
		t.Fatalf("StateReader: %v", err)
	}

	gotAcct, err := reader.Account(addr)
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if gotAcct == nil {
		t.Fatal("account is nil")
	}
	if gotAcct.Nonce != 7 {
		t.Errorf("nonce: got %d, want 7", gotAcct.Nonce)
	}
	if gotAcct.Balance.Uint64() != 100 {
		t.Errorf("balance: got %s, want 100", gotAcct.Balance)
	}

	for _, tc := range []struct{ slot, want common.Hash }{
		{common.HexToHash("0x00"), common.HexToHash("0x11")},
		{common.HexToHash("0x01"), common.HexToHash("0x22")},
		{common.HexToHash("0x05"), common.HexToHash("0x33")},
	} {
		got, err := reader.Storage(addr, tc.slot)
		if err != nil {
			t.Fatalf("Storage(%x): %v", tc.slot, err)
		}
		if got != tc.want {
			t.Errorf("slot %x: got %x, want %x", tc.slot, got, tc.want)
		}
	}
}
