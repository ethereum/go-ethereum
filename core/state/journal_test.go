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

	{
		// If the journal performs the rollback in the wrong order, this
		// will cause a panic.
		id := j.snapshot()
		statedb.AddSlotToAccessList(common.Address{0x1}, common.Hash{0x4})
		statedb.AddSlotToAccessList(common.Address{0x3}, common.Hash{0x4})
		statedb.RevertToSnapshot(id)
	}
	{
		id := j.snapshot()
		statedb.AddAddressToAccessList(common.Address{0x2})
		statedb.AddAddressToAccessList(common.Address{0x3})
		statedb.AddAddressToAccessList(common.Address{0x4})
		statedb.RevertToSnapshot(id)
		if statedb.accessList.ContainsAddress(common.Address{0x2}) {
			t.Fatal("should be missing")
		}
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
	zero := j.snapshot()
	j.refundChange(0)
	j.refundChange(1)
	{
		id := j.snapshot()
		j.refundChange(2)
		j.refundChange(3)
		j.revertToSnapshot(id, statedb)
		if have, want := statedb.refund, uint64(2); have != want {
			t.Fatalf("have %d want %d", have, want)
		}
	}
	{
		id := j.snapshot()
		j.refundChange(2)
		j.refundChange(3)
		j.DiscardSnapshot(id)
	}
	j.revertToSnapshot(zero, statedb)
	if have, want := statedb.refund, uint64(0); have != want {
		t.Fatalf("have %d want %d", have, want)
	}
}
