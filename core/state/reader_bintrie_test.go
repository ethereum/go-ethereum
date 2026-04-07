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

// TestBintrieFlatReaderMissingAccount verifies that an account never
// touched by any commit returns (nil, nil) — the standard "account
// doesn't exist" sentinel that the merkle flatReader also returns.
func TestBintrieFlatReaderMissingAccount(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(disk, triedb.VerkleDefaults)
	sdb := NewDatabase(tdb, nil)
	state, err := New(types.EmptyVerkleHash, sdb)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	// Touch addrA so the trie has at least one stem; otherwise we'd be
	// reading from an empty disk layer where everything is trivially
	// absent.
	addrA := common.HexToAddress("0x0101010101010101010101010101010101010101")
	state.SetBalance(addrA, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	root, err := state.Commit(0, true, false)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	reader, err := sdb.StateReader(root)
	if err != nil {
		t.Fatalf("StateReader: %v", err)
	}

	missing := common.HexToAddress("0xfeedfacefeedfacefeedfacefeedfacefeedface")
	got, err := reader.Account(missing)
	if err != nil {
		t.Fatalf("Account(missing): %v", err)
	}
	if got != nil {
		t.Errorf("missing account: got %+v, want nil", got)
	}
}
