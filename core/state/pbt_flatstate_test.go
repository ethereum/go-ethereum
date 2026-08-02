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
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// flatDiffAccounts is the account count used by the differential tests. Small
// enough to be quick, large enough to span many trie nodes.
const flatDiffAccounts = 512

// buildFlatDiffState populates a binary-tree state and returns the database and
// the committed root. Accounts are given two storage slots, one in the header
// stem and one in the overflow bucket, plus contract code.
func buildFlatDiffState(t *testing.T) (Database, common.Hash) {
	t.Helper()

	disk := rawdb.NewMemoryDatabase()
	db := NewDatabase(triedb.NewDatabase(disk, &triedb.Config{
		IsPBT:  true,
		PathDB: &pathdb.Config{TrieCleanSize: 1024, StateCleanSize: 1024, NoAsyncFlush: true},
	}), NewCodeDB(disk))

	statedb, err := New(types.EmptyBinaryHash, db)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < flatDiffAccounts; i++ {
		addr := flatDiffAddr(i)
		statedb.SetBalance(addr, uint256.NewInt(uint64(i)+1), tracing.BalanceChangeUnspecified)
		statedb.SetNonce(addr, uint64(i)+1, tracing.NonceChangeUnspecified)
		if i%3 == 0 {
			statedb.SetCode(addr, []byte{0x60, byte(i), 0x60, 0x00, 0x55, 0x00}, tracing.CodeChangeUnspecified)
		}
		for j := 0; j < 2; j++ {
			slot := flatDiffSlot(j)
			statedb.SetState(addr, slot, common.BigToHash(uint256.NewInt(uint64(i)+1).ToBig()))
		}
	}
	root, err := statedb.Commit(0, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.TrieDB().Commit(root, false); err != nil {
		t.Fatal(err)
	}
	return db, root
}

func flatDiffAddr(i int) common.Address {
	var a common.Address
	binary.BigEndian.PutUint64(a[12:], uint64(i)+1)
	return a
}

func flatDiffSlot(j int) common.Hash {
	var slot common.Hash
	if j == 0 {
		slot[31] = 3 // header stem
		return slot
	}
	slot[31] = 200 // overflow bucket
	return slot
}

// TestPBTFlatStateMatchesTrie is the differential harness: every account and
// slot must read identically from the flat store and from the trie.
//
// Absence is the case that matters, and the reason both directions are
// checked. Once the generation marker no longer blocks flat state, a flat miss
// stops being an error and becomes the answer "this does not exist" - returned
// with no error, so the trie reader behind it is never consulted, and cached.
// A harness comparing only accounts both readers return would pass while
// exactly that bug was live.
func TestPBTFlatStateMatchesTrie(t *testing.T) {
	db, root := buildFlatDiffState(t)

	stateReader, err := db.TrieDB().StateReader(root)
	if err != nil {
		t.Fatalf("flat state is not available: %v", err)
	}
	flat := newFlatReader(stateReader)
	trie, err := newPBTTrieReader(root, db.TrieDB())
	if err != nil {
		t.Fatal(err)
	}

	// Comparisons actually performed, checked at the end. Without this a run
	// where every flat read errored would skip its way to a green result.
	var comparedAccounts, comparedSlots, comparedAbsent int

	// Present accounts, and absent ones interleaved. The absent addresses sit
	// in the same key space as the present ones so they exercise the same
	// regions of the tree.
	for i := 0; i < flatDiffAccounts*2; i++ {
		addr := flatDiffAddr(i)
		present := i < flatDiffAccounts

		fAcct, fErr := flat.Account(addr)
		tAcct, tErr := trie.Account(addr)
		if (fErr == nil) != (tErr == nil) {
			t.Fatalf("account %x: flat err %v, trie err %v", addr, fErr, tErr)
		}
		if fErr != nil {
			continue
		}
		if (fAcct == nil) != (tAcct == nil) {
			t.Fatalf("account %x: flat has=%t, trie has=%t (present=%t)", addr, fAcct != nil, tAcct != nil, present)
		}
		if fAcct == nil {
			if present {
				t.Fatalf("account %x should exist but both readers report it absent", addr)
			}
			comparedAbsent++
			continue
		}
		if !present {
			t.Fatalf("account %x should not exist but both readers returned it", addr)
		}
		if fAcct.Nonce != tAcct.Nonce || fAcct.Balance.Cmp(tAcct.Balance) != 0 || !bytes.Equal(fAcct.CodeHash, tAcct.CodeHash) {
			t.Fatalf("account %x: flat %+v, trie %+v", addr, fAcct, tAcct)
		}
		comparedAccounts++

		for j := 0; j < 3; j++ {
			// j==2 is a slot that was never written: absent in both.
			slot := flatDiffSlot(j)
			if j == 2 {
				slot[31] = 77
			}
			fSlot, fErr := flat.Storage(addr, slot)
			tSlot, tErr := trie.Storage(addr, slot)
			if (fErr == nil) != (tErr == nil) {
				t.Fatalf("slot %x of %x: flat err %v, trie err %v", slot, addr, fErr, tErr)
			}
			if fErr != nil {
				continue
			}
			if fSlot != tSlot {
				t.Fatalf("slot %x of %x: flat %x, trie %x", slot, addr, fSlot, tSlot)
			}
			if j == 2 && fSlot != (common.Hash{}) {
				t.Fatalf("slot %x of %x was never written but reads back %x", slot, addr, fSlot)
			}
			comparedSlots++
		}
	}

	if comparedAccounts != flatDiffAccounts {
		t.Fatalf("compared %d present accounts, expected %d", comparedAccounts, flatDiffAccounts)
	}
	if comparedAbsent != flatDiffAccounts {
		t.Fatalf("compared %d absent accounts, expected %d", comparedAbsent, flatDiffAccounts)
	}
	if want := flatDiffAccounts * 3; comparedSlots != want {
		t.Fatalf("compared %d slots, expected %d", comparedSlots, want)
	}
}

// TestPBTFlatStateAttestedOnOpen pins that opening a binary-tree database
// records the attestation that the differential test above depends on.
//
// The refusal side - that a database carrying state without an attestation is
// rejected rather than served - is pinned in pathdb.TestAttestFlatState, where
// the decision can be exercised without the process-fatal open path.
func TestPBTFlatStateAttestedOnOpen(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	triedb.NewDatabase(disk, &triedb.Config{
		IsPBT:  true,
		PathDB: &pathdb.Config{TrieCleanSize: 1024, StateCleanSize: 1024, NoAsyncFlush: true},
	})
	// The attestation lives under the binary tree's table prefix, alongside
	// the rest of its data, so it must be read through the same table.
	if !rawdb.ReadPBTFlatState(rawdb.NewTable(disk, string(rawdb.PBTPrefix))) {
		t.Fatal("a fresh binary-tree database did not record its flat state attestation")
	}
}
