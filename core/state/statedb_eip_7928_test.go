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
	"maps"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
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
	root0, err := base.Commit(params.Rules{}, 0)
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
	wantRoot := seq.IntermediateRoot(params.Rules{IsEIP158: true})

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
	gotRoot := balState.IntermediateRoot(params.Rules{IsEIP158: true})
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
	root0, err := base.Commit(params.Rules{}, 0)
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
	wantRoot := seq.IntermediateRoot(params.Rules{IsEIP158: true})
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
	gotRoot := balState.IntermediateRoot(params.Rules{IsEIP158: true})
	balState.StopPrefetcher()

	if gotRoot != wantRoot {
		t.Fatalf("BAL apply root = %x, want %x", gotRoot, wantRoot)
	}
}

// TestAccessListNetsOverBlockAccessIndex checks that the changes recorded for
// a block access index are net of all the finalisation scopes sharing it (the
// withdrawals and the post-execution system calls, for instance), and not of
// each scope on its own.
func TestAccessListNetsOverBlockAccessIndex(t *testing.T) {
	var (
		earlier = common.HexToAddress("0x1111") // also changed at an earlier index
		only    = common.HexToAddress("0x2222") // only touched at the shared index
		again   = common.HexToAddress("0x3333") // restored, then changed once more
		slot    = common.HexToHash("0x01")
		rules   = params.Rules{IsEIP158: true, IsEIP2929: true, IsAmsterdam: true}

		codeA, codeB, codeC = []byte{0x00}, []byte{0x5f, 0x00}, []byte{0x5f, 0x5f, 0x00}
	)
	sdb, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	set := func(addr common.Address, balance, nonce uint64, code []byte, value byte) {
		sdb.SetBalance(addr, uint256.NewInt(balance), tracing.BalanceChangeUnspecified)
		sdb.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
		sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
		sdb.SetState(addr, slot, common.BytesToHash([]byte{value}))
	}
	for _, addr := range []common.Address{earlier, only, again} {
		set(addr, 5, 1, codeA, 0)
	}
	sdb.Finalise(params.Rules{IsEIP158: true})

	list := bal.NewConstructionBlockAccessList()
	scope := func(index uint32, fn func()) {
		sdb.Prepare(rules, common.Address{}, common.Address{}, nil, nil, nil)
		sdb.SetTxContext(common.Hash{}, 0, index)
		fn()
		list.Merge(sdb.Finalise(rules))
	}
	// Index 1, a single scope.
	scope(1, func() { set(earlier, 7, 2, codeB, 1) })

	// Index 2, three scopes: change, restore, and change one account again.
	scope(2, func() {
		set(earlier, 9, 3, codeC, 2)
		set(only, 9, 3, codeC, 2)
		set(again, 9, 3, codeC, 2)
	})
	scope(2, func() {
		set(earlier, 7, 2, codeB, 1)
		set(only, 5, 1, codeA, 0)
		set(again, 5, 1, codeA, 0)
	})
	scope(2, func() { set(again, 11, 4, codeB, 3) })

	check := func(addr common.Address, indices ...uint32) {
		t.Helper()
		acct := list.Accounts[addr]
		if acct == nil {
			t.Fatalf("account %x missing from the access list", addr)
		}
		have := map[string][]uint32{
			"balance": slices.Sorted(maps.Keys(acct.BalanceChanges)),
			"nonce":   slices.Sorted(maps.Keys(acct.NonceChanges)),
			"code":    slices.Sorted(maps.Keys(acct.CodeChange)),
			"storage": slices.Sorted(maps.Keys(acct.StorageWrites[slot])),
		}
		for kind, have := range have {
			if !slices.Equal(have, indices) {
				t.Errorf("account %x: %s changes at indices %v, want %v", addr, kind, have, indices)
			}
		}
		if _, read := acct.StorageReads[slot]; read != (len(indices) == 0) {
			t.Errorf("account %x: slot in storage reads = %v, want %v", addr, read, len(indices) == 0)
		}
	}
	check(earlier, 1)
	check(only)
	check(again, 2)

	acct := list.Accounts[again]
	if !acct.BalanceChanges[2].Eq(uint256.NewInt(11)) || acct.NonceChanges[2] != 4 ||
		!bytes.Equal(acct.CodeChange[2], codeB) || acct.StorageWrites[slot][2] != common.BytesToHash([]byte{3}) {
		t.Errorf("account %x: wrong post-index values: %+v", again, acct)
	}
	acct = list.Accounts[earlier]
	if !acct.BalanceChanges[1].Eq(uint256.NewInt(7)) || acct.NonceChanges[1] != 2 ||
		!bytes.Equal(acct.CodeChange[1], codeB) || acct.StorageWrites[slot][1] != common.BytesToHash([]byte{1}) {
		t.Errorf("account %x: wrong values at the earlier index: %+v", earlier, acct)
	}
}
