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

// TestBintrieFlatReaderMissingAccountAuthoritative verifies that the flat
// reader returns (nil, nil) for absent accounts after generation completes.
func TestBintrieFlatReaderMissingAccountAuthoritative(t *testing.T) {
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
	// Flush to disk so the generator completes (genMarker → nil).
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("tdb.Commit (flush to disk): %v", err)
	}

	// Get the pathdb reader so we can test the bintrieFlatReader in
	// isolation.
	pathdbReader, err := tdb.StateReader(root)
	if err != nil {
		t.Fatalf("pathdb StateReader: %v", err)
	}
	br := newBintrieFlatReader(pathdbReader)
	if br == nil {
		t.Fatal("newBintrieFlatReader returned nil")
	}

	missing := common.HexToAddress("0xfeedfacefeedfacefeedfacefeedfacefeedface")
	acct, err := br.Account(missing)
	if err != nil {
		t.Fatalf("expected authoritative nil for missing account, got error: %v", err)
	}
	if acct != nil {
		t.Fatalf("expected nil account for missing address, got: %+v", acct)
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

// TestBintrieFlatReaderStorageTombstone verifies the bintrie "tombstone"
// convention: a storage slot set to zero is present-with-32-zero-bytes,
// which must be distinguishable from "never written" (absent). This is
// the A16/T8 integration test.
func TestBintrieFlatReaderStorageTombstone(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)

	addr := common.HexToAddress("0xABCDEF0123456789ABCDEF0123456789ABCDEF01")
	slot := common.HexToHash("0x07")
	nonZero := common.HexToHash("0x42")

	// Block 1: set slot to non-zero.
	state1, _ := New(types.EmptyVerkleHash, sdb)
	state1.SetBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	state1.SetState(addr, slot, nonZero)
	root1, err := state1.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit block 1: %v", err)
	}

	// Block 2: set the same slot to zero (the bintrie writes 32 zero
	// bytes as a tombstone rather than deleting the offset).
	state2, _ := New(root1, sdb)
	state2.SetState(addr, slot, common.Hash{})
	root2, err := state2.Commit(1, true, false)
	if err != nil {
		t.Fatalf("commit block 2: %v", err)
	}

	// Read at block 2: should be the zero hash.
	reader2, err := sdb.StateReader(root2)
	if err != nil {
		t.Fatalf("StateReader(block2): %v", err)
	}
	got2, err := reader2.Storage(addr, slot)
	if err != nil {
		t.Fatalf("Storage(block2): %v", err)
	}
	if got2 != (common.Hash{}) {
		t.Errorf("block 2 slot: got %x, want zero", got2)
	}

	// Read at block 1: should still be the non-zero value.
	reader1, err := sdb.StateReader(root1)
	if err != nil {
		t.Fatalf("StateReader(block1): %v", err)
	}
	got1, err := reader1.Storage(addr, slot)
	if err != nil {
		t.Fatalf("Storage(block1): %v", err)
	}
	if got1 != nonZero {
		t.Errorf("block 1 slot: got %x, want %x", got1, nonZero)
	}
}

// TestBintrieFlatReaderMultiBlockEvolution verifies that diff-layer
// chaining works correctly across multiple blocks for the bintrie path.
// This is the A16/T9 integration test.
func TestBintrieFlatReaderMultiBlockEvolution(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)

	addr := common.HexToAddress("0xDeaDBeefDeaDBeefDeaDBeefDeaDBeefDeaDBeef")

	// Block 1: nonce=1, balance=100
	state1, _ := New(types.EmptyVerkleHash, sdb)
	state1.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	state1.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	root1, err := state1.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit block 1: %v", err)
	}

	// Block 2: nonce=2 (balance unchanged at 100)
	state2, _ := New(root1, sdb)
	state2.SetNonce(addr, 2, tracing.NonceChangeUnspecified)
	root2, err := state2.Commit(1, true, false)
	if err != nil {
		t.Fatalf("commit block 2: %v", err)
	}

	// Block 3: balance=200 (nonce unchanged at 2)
	state3, _ := New(root2, sdb)
	state3.SetBalance(addr, uint256.NewInt(200), tracing.BalanceChangeUnspecified)
	root3, err := state3.Commit(2, true, false)
	if err != nil {
		t.Fatalf("commit block 3: %v", err)
	}

	// Read at each root and verify the expected snapshot.
	for _, tc := range []struct {
		name    string
		root    common.Hash
		nonce   uint64
		balance uint64
	}{
		{"block1", root1, 1, 100},
		{"block2", root2, 2, 100},
		{"block3", root3, 2, 200},
	} {
		reader, err := sdb.StateReader(tc.root)
		if err != nil {
			t.Fatalf("%s StateReader: %v", tc.name, err)
		}
		got, err := reader.Account(addr)
		if err != nil {
			t.Fatalf("%s Account: %v", tc.name, err)
		}
		if got == nil {
			t.Fatalf("%s: account is nil", tc.name)
		}
		if got.Nonce != tc.nonce {
			t.Errorf("%s nonce: got %d, want %d", tc.name, got.Nonce, tc.nonce)
		}
		if got.Balance.Uint64() != tc.balance {
			t.Errorf("%s balance: got %d, want %d", tc.name, got.Balance.Uint64(), tc.balance)
		}
	}
}
