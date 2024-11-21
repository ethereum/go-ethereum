// Copyright 2024 The go-ethereum Authors
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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func TestLinearJournalDirty(t *testing.T) {
	testJournalDirty(t, newLinearJournal())
}

func TestSparseJournalDirty(t *testing.T) {
	testJournalDirty(t, newSparseJournal())
}

// This test verifies some basics around journalling: the ability to
// deliver a dirty-set.
func testJournalDirty(t *testing.T, j journal) {
	acc := &types.StateAccount{
		Nonce:    1,
		Balance:  new(uint256.Int),
		Root:     common.Hash{},
		CodeHash: nil,
	}
	{
		j.nonceChange(common.Address{0x1}, acc, false, false)
		if have, want := len(j.dirtyAccounts()), 1; have != want {
			t.Errorf("wrong size of dirty accounts, have %v want %v", have, want)
		}
	}
	{
		j.storageChange(common.Address{0x2}, common.Hash{0x1}, common.Hash{0x1}, common.Hash{})
		if have, want := len(j.dirtyAccounts()), 2; have != want {
			t.Errorf("wrong size of dirty accounts, have %v want %v", have, want)
		}
	}
	{ // The previous scopes should also be accounted for
		j.snapshot()
		if have, want := len(j.dirtyAccounts()), 2; have != want {
			t.Errorf("wrong size of dirty accounts, have %v want %v", have, want)
		}
	}
}

func TestLinearJournalAccessList(t *testing.T) {
	testJournalAccessList(t, newLinearJournal())
}

func TestSparseJournalAccessList(t *testing.T) {
	testJournalAccessList(t, newSparseJournal())
}

func testJournalAccessList(t *testing.T, j journal) {
	var statedb = &StateDB{}
	statedb.accessList = newAccessList()
	statedb.journal = j

	j.snapshot()
	{
		// If the journal performs the rollback in the wrong order, this
		// will cause a panic.
		statedb.AddSlotToAccessList(common.Address{0x1}, common.Hash{0x4})
		statedb.AddSlotToAccessList(common.Address{0x3}, common.Hash{0x4})
	}
	statedb.RevertSnapshot()
	j.snapshot()
	{
		statedb.AddAddressToAccessList(common.Address{0x2})
		statedb.AddAddressToAccessList(common.Address{0x3})
		statedb.AddAddressToAccessList(common.Address{0x4})
	}
	statedb.RevertSnapshot()
	if statedb.accessList.ContainsAddress(common.Address{0x2}) {
		t.Fatal("should be missing")
	}
}

func TestLinearJournalRefunds(t *testing.T) {
	testJournalRefunds(t, newLinearJournal())
}

func TestSparseJournalRefunds(t *testing.T) {
	testJournalRefunds(t, newSparseJournal())
}

func testJournalRefunds(t *testing.T, j journal) {
	var statedb = &StateDB{}
	statedb.accessList = newAccessList()
	statedb.journal = j
	j.snapshot()
	{
		j.refundChange(0)
		j.refundChange(1)
		j.snapshot()
		{
			j.refundChange(2)
			j.refundChange(3)
		}
		j.revertSnapshot(statedb)
		if have, want := statedb.refund, uint64(2); have != want {
			t.Fatalf("have %d want %d", have, want)
		}
		j.snapshot()
		{
			j.refundChange(2)
			j.refundChange(3)
		}
		j.discardSnapshot()
	}
	j.revertSnapshot(statedb)
	if have, want := statedb.refund, uint64(0); have != want {
		t.Fatalf("have %d want %d", have, want)
	}
}

func FuzzJournals(f *testing.F) {

	randByte := func() byte {
		return byte(rand.Int())
	}
	randBool := func() bool {
		return rand.Int()%2 == 0
	}
	randAccount := func() *types.StateAccount {
		return &types.StateAccount{
			Nonce:    uint64(randByte()),
			Balance:  uint256.NewInt(uint64(randByte())),
			Root:     types.EmptyRootHash,
			CodeHash: types.EmptyCodeHash[:],
		}
	}

	f.Fuzz(func(t *testing.T, operations []byte) {
		var (
			statedb1, _ = New(types.EmptyRootHash, NewDatabaseForTesting())
			statedb2, _ = New(types.EmptyRootHash, NewDatabaseForTesting())
			linear      = newLinearJournal()
			sparse      = newSparseJournal()
		)
		statedb1.journal = linear
		statedb2.journal = sparse
		linear.snapshot()
		sparse.snapshot()

		for _, o := range operations {
			switch o {
			case 0:
				addr := randByte()
				linear.accessListAddAccount(common.Address{addr})
				sparse.accessListAddAccount(common.Address{addr})
				statedb1.accessList.AddAddress(common.Address{addr})
				statedb2.accessList.AddAddress(common.Address{addr})
			case 1:
				addr := randByte()
				slot := randByte()
				linear.accessListAddSlot(common.Address{addr}, common.Hash{slot})
				sparse.accessListAddSlot(common.Address{addr}, common.Hash{slot})
				statedb1.accessList.AddSlot(common.Address{addr}, common.Hash{slot})
				statedb2.accessList.AddSlot(common.Address{addr}, common.Hash{slot})
			case 2:
				addr := randByte()
				account := randAccount()
				destructed := randBool()
				newContract := randBool()
				linear.balanceChange(common.Address{addr}, account, destructed, newContract)
				sparse.balanceChange(common.Address{addr}, account, destructed, newContract)
			case 3:
				linear = linear.copy().(*linearJournal)
				sparse = sparse.copy().(*sparseJournal)
			case 4:
				addr := randByte()
				account := randAccount()
				linear.createContract(common.Address{addr}, account)
				sparse.createContract(common.Address{addr}, account)
			case 5:
				addr := randByte()
				linear.createObject(common.Address{addr})
				sparse.createObject(common.Address{addr})
			case 6:
				addr := randByte()
				account := randAccount()
				linear.destruct(common.Address{addr}, account)
				sparse.destruct(common.Address{addr}, account)
			case 7:
				txHash := randByte()
				linear.logChange(common.Hash{txHash})
				sparse.logChange(common.Hash{txHash})
			case 8:
				addr := randByte()
				account := randAccount()
				destructed := randBool()
				newContract := randBool()
				linear.nonceChange(common.Address{addr}, account, destructed, newContract)
				sparse.nonceChange(common.Address{addr}, account, destructed, newContract)
			case 9:
				refund := randByte()
				linear.refundChange(uint64(refund))
				sparse.refundChange(uint64(refund))
			case 10:
				addr := randByte()
				account := randAccount()
				linear.setCode(common.Address{addr}, account)
				sparse.setCode(common.Address{addr}, account)
			case 11:
				addr := randByte()
				key := randByte()
				prev := randByte()
				origin := randByte()
				linear.storageChange(common.Address{addr}, common.Hash{key}, common.Hash{prev}, common.Hash{origin})
				sparse.storageChange(common.Address{addr}, common.Hash{key}, common.Hash{prev}, common.Hash{origin})
			case 12:
				addr := randByte()
				account := randAccount()
				destructed := randBool()
				newContract := randBool()
				linear.touchChange(common.Address{addr}, account, destructed, newContract)
				sparse.touchChange(common.Address{addr}, account, destructed, newContract)
			case 13:
				addr := randByte()
				key := randByte()
				prev := randByte()
				linear.transientStateChange(common.Address{addr}, common.Hash{key}, common.Hash{prev})
				sparse.transientStateChange(common.Address{addr}, common.Hash{key}, common.Hash{prev})
			case 14:
				linear.reset()
				sparse.reset()
			case 15:
				linear.snapshot()
				sparse.snapshot()
			case 16:
				linear.discardSnapshot()
				sparse.discardSnapshot()
			case 17:
				linear.revertSnapshot(statedb1)
				sparse.revertSnapshot(statedb2)
			case 18:
				accs1 := linear.dirtyAccounts()
				accs2 := linear.dirtyAccounts()
				if len(accs1) != len(accs2) {
					panic(fmt.Sprintf("mismatched accounts: %v %v", accs1, accs2))

				}
				for _, val := range accs1 {
					found := false
					for _, val2 := range accs2 {
						if val == val2 {
							if found {
								panic(fmt.Sprintf("account found twice: %v %v account %v", accs1, accs2, val))
							}
							found = true
						}
					}
					if !found {
						panic(fmt.Sprintf("missing account: %v %v account %v", accs1, accs2, val))
					}
				}
			}
		}
		// After all operations have been processed, verify equality
		accs1 := linear.dirtyAccounts()
		accs2 := linear.dirtyAccounts()
		for _, val := range accs1 {
			found := false
			for _, val2 := range accs2 {
				if val == val2 {
					if found {
						panic(fmt.Sprintf("account found twice: %v %v account %v", accs1, accs2, val))
					}
					found = true
				}
			}
			if !found {
				panic(fmt.Sprintf("missing account: %v %v account %v", accs1, accs2, val))
			}
		}
		h1, err1 := statedb1.Commit(0, false)
		h2, err2 := statedb2.Commit(0, false)
		if err1 != err2 {
			panic(fmt.Sprintf("mismatched errors: %v %v", err1, err2))
		}
		if h1 != h2 {
			panic(fmt.Sprintf("mismatched roots: %v %v", h1, h2))
		}
	})
}
