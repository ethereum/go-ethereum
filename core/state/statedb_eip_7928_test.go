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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
)

// TestApplyBlockAccessListConcurrentPrefetch stresses the concurrent storage
// prefetch scheduling with many contracts (each carrying a non-empty storage
// trie), so that several BAL workers schedule prefetches into the shared,
// non-thread-safe prefetcher at once. Run with -race to detect unguarded
// concurrent access. The resulting root must still match the sequential one.
func TestApplyBlockAccessListConcurrentPrefetch(t *testing.T) {
	const n = 64
	code := []byte{0x60, 0x00, 0x60, 0x00}
	addrOf := func(i int) common.Address {
		return common.BigToAddress(uint256.NewInt(uint64(0x1000 + i)).ToBig())
	}
	slot := common.HexToHash("0x01")

	// Base state: n contracts, each with a pre-existing storage slot so its
	// storage root is non-empty.
	db := NewDatabaseForTesting()
	base, _ := New(types.EmptyRootHash, db)
	for i := range n {
		addr := addrOf(i)
		base.SetBalance(addr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
		base.SetCode(addr, code, tracing.CodeChangeUnspecified)
		base.SetState(addr, slot, common.HexToHash("0xaa"))
	}
	root0, err := base.Commit(0, false, false)
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}

	mutate := func(s *StateDB) {
		for i := range n {
			addr := addrOf(i)
			s.SetBalance(addr, uint256.NewInt(uint64(200+i)), tracing.BalanceChangeUnspecified)
			s.SetState(addr, slot, common.BigToHash(uint256.NewInt(uint64(i+1)).ToBig()))
		}
	}
	seq, _ := New(root0, db)
	mutate(seq)
	wantRoot := seq.IntermediateRoot(true)

	cb := bal.NewConstructionBlockAccessList()
	for i := range n {
		addr := addrOf(i)
		cb.BalanceChange(0, addr, uint256.NewInt(uint64(200+i)))
		cb.StorageWrite(0, addr, slot, common.BigToHash(uint256.NewInt(uint64(i+1)).ToBig()))
	}
	balState, _ := New(root0, db)
	balState.StartPrefetcher("test", nil)
	if err := balState.ApplyBlockAccessList(cb.ToEncodingObj()); err != nil {
		balState.StopPrefetcher()
		t.Fatalf("apply block access list: %v", err)
	}
	gotRoot := balState.IntermediateRoot(true)
	balState.StopPrefetcher()

	if gotRoot != wantRoot {
		t.Fatalf("BAL apply root = %x, want %x", gotRoot, wantRoot)
	}
}

// TestApplyBlockAccessListMatchesSequential checks that installing a block's
// post-state through ApplyBlockAccessList yields exactly the same state root as
// applying the same mutations one by one. The BAL path warms both the account
// trie and the storage tries through the prefetcher and pulls them back at
// IntermediateRoot, so this also exercises that machinery. Run with -race to
// catch data races in the concurrent prefetch scheduling.
func TestApplyBlockAccessListMatchesSequential(t *testing.T) {
	var (
		existing = common.HexToAddress("0x1111")
		contract = common.HexToAddress("0x2222")
		fresh    = common.HexToAddress("0x3333")

		slotA = common.HexToHash("0x01")
		slotB = common.HexToHash("0x02")
		code  = []byte{0x60, 0x00, 0x60, 0x00}
	)

	// Build a base state with a plain account and a contract that already has
	// some storage (so its storage root is non-empty and the storage-trie
	// prefetch path is exercised).
	db := NewDatabaseForTesting()
	base, _ := New(types.EmptyRootHash, db)
	base.SetBalance(existing, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	base.SetNonce(existing, 1, tracing.NonceChangeUnspecified)
	base.SetBalance(contract, uint256.NewInt(50), tracing.BalanceChangeUnspecified)
	base.SetCode(contract, code, tracing.CodeChangeUnspecified)
	base.SetState(contract, slotA, common.HexToHash("0xaa"))
	base.SetState(contract, slotB, common.HexToHash("0xbb"))
	root0, err := base.Commit(0, false, false)
	if err != nil {
		t.Fatalf("commit base: %v", err)
	}

	// mutate applies the block's post-state via the ordinary setters.
	mutate := func(s *StateDB) {
		s.SetBalance(existing, uint256.NewInt(1234), tracing.BalanceChangeUnspecified)
		s.SetNonce(existing, 2, tracing.NonceChangeUnspecified)
		s.SetState(contract, slotA, common.HexToHash("0xcc")) // changed
		s.SetState(contract, slotB, common.HexToHash("0xbb")) // unchanged, must be a no-op
		s.SetBalance(fresh, uint256.NewInt(7), tracing.BalanceChangeUnspecified)
		s.SetNonce(fresh, 1, tracing.NonceChangeUnspecified)
	}

	// Sequential reference root.
	seq, _ := New(root0, db)
	mutate(seq)
	wantRoot := seq.IntermediateRoot(true)
	if wantRoot == root0 {
		t.Fatal("mutations did not change the state root")
	}

	// Same post-state expressed as a block access list.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, existing, uint256.NewInt(1234))
	cb.NonceChange(existing, 0, 2)
	cb.StorageWrite(0, contract, slotA, common.HexToHash("0xcc"))
	cb.StorageWrite(0, contract, slotB, common.HexToHash("0xbb")) // no-op write
	cb.BalanceChange(0, fresh, uint256.NewInt(7))
	cb.NonceChange(fresh, 0, 1)
	list := cb.ToEncodingObj()

	balState, _ := New(root0, db)
	balState.StartPrefetcher("test", nil)
	if err := balState.ApplyBlockAccessList(list); err != nil {
		balState.StopPrefetcher()
		t.Fatalf("apply block access list: %v", err)
	}
	gotRoot := balState.IntermediateRoot(true)
	balState.StopPrefetcher()

	if gotRoot != wantRoot {
		t.Fatalf("BAL apply root = %x, want %x", gotRoot, wantRoot)
	}
}
