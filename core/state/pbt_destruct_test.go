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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// TestPBTDestroyedAccountLeavesNoFlatStorage pins that destroying an account
// removes its slots from the flat store, not only from the tree.
//
// The tree drops the whole storage bucket structurally, so the state root is
// right either way - which is exactly why a root comparison against a reference
// implementation cannot see this. The flat store is keyed per slot and is
// consulted ahead of the tree, so a slot left behind is served in preference to
// the tree's correct answer.
//
// The damage needs a later block to show up. Within the block that destroys it
// GetCommittedState short-circuits on stateObjectsDestruct and reports empty,
// but that set is cleared per block, so an address reoccupied afterwards reads
// whatever the flat store kept.
func TestPBTDestroyedAccountLeavesNoFlatStorage(t *testing.T) {
	var (
		sdb, db = newPBTState(t)
		addr    = common.Address{0xde, 0xad}
		header  = common.Hash{31: 7}    // slot below 64: the account header stem
		bucket  = common.Hash{30: 4}    // slot 1024: the overflow bucket
		valA    = common.Hash{31: 0xaa} // header slot value
		valB    = common.Hash{31: 0xbb} // overflow slot value
	)
	// Block 1: an account holding storage in both of its homes.
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	sdb.SetState(addr, header, valA)
	sdb.SetState(addr, bucket, valB)
	sdb = reopenPBT(t, sdb, db, 1)

	if got := sdb.GetState(addr, header); got != valA {
		t.Fatalf("header slot did not survive the first commit: got %x, want %x", got, valA)
	}
	if got := sdb.GetState(addr, bucket); got != valB {
		t.Fatalf("overflow slot did not survive the first commit: got %x, want %x", got, valB)
	}

	// Block 2: destroy it. The balance is zero, so the account goes away rather
	// than being kept as a husk.
	sdb.SelfDestruct(addr)
	sdb = reopenPBT(t, sdb, db, 2)

	if sdb.Exist(addr) {
		t.Fatal("the account survived its own destruction")
	}

	// Block 3: the address is occupied again, as CREATE2 to the same address
	// would. Its storage must start empty.
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	sdb.SetBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)

	if got := sdb.GetState(addr, header); got != (common.Hash{}) {
		t.Errorf("a destroyed account's header slot was served to its successor: got %x", got)
	}
	if got := sdb.GetState(addr, bucket); got != (common.Hash{}) {
		t.Errorf("a destroyed account's overflow slot was served to its successor: got %x", got)
	}
}
