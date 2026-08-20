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

package bal

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func u256(n uint64) *uint256.Int { return uint256.NewInt(n) }

// foldList builds a single-account list; index-carrying fields come in as-is.
func foldList(accounts ...AccountAccess) *BlockAccessList {
	list := BlockAccessList(accounts)
	return &list
}

// TestFoldLastWriteWinsAcrossBlocks pins the hazard Fold exists for: access
// indexes only order writes within one block, so a later block's low-index
// write must beat an earlier block's high-index write to the same slot.
func TestFoldLastWriteWinsAcrossBlocks(t *testing.T) {
	addr := common.Address{0x01}
	first := foldList(AccountAccess{
		Address: addr,
		StorageChanges: []encodingSlotChanges{{
			Slot:        u256(7),
			SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 10, PostValue: u256(0xaa)}},
		}},
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 9, PostBalance: u256(100)}},
	})
	second := foldList(AccountAccess{
		Address: addr,
		StorageChanges: []encodingSlotChanges{{
			Slot:        u256(7),
			SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 2, PostValue: u256(0xbb)}},
		}},
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 1, PostBalance: u256(50)}},
	})

	folded, n := Fold([]*BlockAccessList{first, second})
	if n != 2 {
		t.Fatalf("folded %d lists, want 2", n)
	}
	if len(*folded) != 1 {
		t.Fatalf("folded into %d accounts, want 1", len(*folded))
	}
	acc := (*folded)[0]
	if got := acc.StorageChanges[0].SlotChanges; len(got) != 1 || !got[0].PostValue.Eq(u256(0xbb)) || got[0].BlockAccessIndex != 0 {
		t.Fatalf("slot fold = %+v, want single index-0 write of 0xbb", got)
	}
	if got := acc.BalanceChanges; len(got) != 1 || !got[0].PostBalance.Eq(u256(50)) || got[0].BlockAccessIndex != 0 {
		t.Fatalf("balance fold = %+v, want single index-0 balance of 50", got)
	}
}

// TestFoldSplitsOnRemovalThenTouch pins the deletion guard: an account whose
// latest folded balance is zero may have been removed at that block's commit,
// so a later block touching it again must start a fresh batch.
func TestFoldSplitsOnRemovalThenTouch(t *testing.T) {
	addr := common.Address{0x02}
	first := foldList(AccountAccess{
		Address:        addr,
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 3, PostBalance: u256(0)}},
	})
	second := foldList(AccountAccess{
		Address:        addr,
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 1, PostBalance: u256(5)}},
	})

	folded, n := Fold([]*BlockAccessList{first, second})
	if n != 1 {
		t.Fatalf("folded %d lists, want 1 (split before the recreation)", n)
	}
	if got := (*folded)[0].BalanceChanges[0].PostBalance; !got.IsZero() {
		t.Fatalf("folded balance = %v, want the removal's zero", got)
	}
}

// TestFoldRemovalMarkerIsLatestOnly pins that a zero balance overwritten
// within the same block is not a removal: only the latest value counts.
func TestFoldRemovalMarkerIsLatestOnly(t *testing.T) {
	addr := common.Address{0x03}
	first := foldList(AccountAccess{
		Address: addr,
		BalanceChanges: []encodingBalanceChange{
			{BlockAccessIndex: 1, PostBalance: u256(0)},
			{BlockAccessIndex: 2, PostBalance: u256(7)},
		},
	})
	second := foldList(AccountAccess{
		Address:      addr,
		NonceChanges: []encodingAccountNonce{{BlockAccessIndex: 1, PostNonce: 1}},
	})

	folded, n := Fold([]*BlockAccessList{first, second})
	if n != 2 {
		t.Fatalf("folded %d lists, want 2 (no removal happened)", n)
	}
	acc := (*folded)[0]
	if !acc.BalanceChanges[0].PostBalance.Eq(u256(7)) || acc.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("fold = %+v, want balance 7 and nonce 1", acc)
	}
}

// TestFoldDropsReadOnlyAccounts pins that accounts with no mutation vanish:
// the folded list installs state, and reads install nothing.
func TestFoldDropsReadOnlyAccounts(t *testing.T) {
	reader := AccountAccess{
		Address:      common.Address{0x04},
		StorageReads: []*uint256.Int{u256(1)},
	}
	writer := AccountAccess{
		Address:      common.Address{0x05},
		NonceChanges: []encodingAccountNonce{{BlockAccessIndex: 1, PostNonce: 4}},
	}

	folded, n := Fold([]*BlockAccessList{foldList(reader, writer)})
	if n != 1 {
		t.Fatalf("folded %d lists, want 1", n)
	}
	if len(*folded) != 1 || (*folded)[0].Address != writer.Address {
		t.Fatalf("fold kept %+v, want only the writer", *folded)
	}
	if len((*folded)[0].StorageReads) != 0 {
		t.Fatalf("fold kept storage reads: %+v", (*folded)[0].StorageReads)
	}
}

// TestFoldOrdersAccountsAndSlots pins the output ordering the appliers and
// the encoding both assume: accounts by address, slots by value.
func TestFoldOrdersAccountsAndSlots(t *testing.T) {
	first := foldList(AccountAccess{
		Address: common.Address{0x02},
		StorageChanges: []encodingSlotChanges{{
			Slot:        u256(5),
			SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 1, PostValue: u256(1)}},
		}},
	})
	second := foldList(AccountAccess{
		Address: common.Address{0x01},
		StorageChanges: []encodingSlotChanges{
			{Slot: u256(9), SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 1, PostValue: u256(2)}}},
		},
	}, AccountAccess{
		Address: common.Address{0x02},
		StorageChanges: []encodingSlotChanges{{
			Slot:        u256(3),
			SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 1, PostValue: u256(3)}},
		}},
	})

	folded, n := Fold([]*BlockAccessList{first, second})
	if n != 2 {
		t.Fatalf("folded %d lists, want 2", n)
	}
	if len(*folded) != 2 || (*folded)[0].Address != (common.Address{0x01}) || (*folded)[1].Address != (common.Address{0x02}) {
		t.Fatalf("account order = %+v, want 0x01 then 0x02", *folded)
	}
	slots := (*folded)[1].StorageChanges
	if len(slots) != 2 || !slots[0].Slot.Eq(u256(3)) || !slots[1].Slot.Eq(u256(5)) {
		t.Fatalf("slot order = %+v, want 3 then 5", slots)
	}
}

// TestFoldPreservesEmptyCodeChange pins the 7702 clear: an empty NewCode is a
// change (back to no code) and must survive folding as one, not vanish.
func TestFoldPreservesEmptyCodeChange(t *testing.T) {
	addr := common.Address{0x06}
	list := foldList(AccountAccess{
		Address:     addr,
		CodeChanges: []encodingCodeChange{{BlockAccessIndex: 2, NewCode: []byte{}}},
	})

	folded, n := Fold([]*BlockAccessList{list})
	if n != 1 {
		t.Fatalf("folded %d lists, want 1", n)
	}
	codes := (*folded)[0].CodeChanges
	if len(codes) != 1 || codes[0].NewCode == nil || len(codes[0].NewCode) != 0 {
		t.Fatalf("code fold = %+v, want one empty (non-absent) code change", codes)
	}
}

// TestFoldEmptyInput pins the degenerate shapes: no lists folds nothing, and
// empty per-block lists are consumed without contributing accounts.
func TestFoldEmptyInput(t *testing.T) {
	if folded, n := Fold(nil); folded != nil || n != 0 {
		t.Fatalf("Fold(nil) = %v, %d, want nil, 0", folded, n)
	}
	empty := BlockAccessList{}
	folded, n := Fold([]*BlockAccessList{&empty, nil})
	if n != 2 {
		t.Fatalf("folded %d lists, want 2 (empty blocks are consumed)", n)
	}
	if len(*folded) != 0 {
		t.Fatalf("fold of empty lists has %d accounts, want 0", len(*folded))
	}
}

// TestFoldSplitsOnEmptinessConsistentTouch pins the widened guard: removal
// triggers on full emptiness, so metadata changes that are all consistent
// with emptiness - not only a zero balance - must split before a later touch.
func TestFoldSplitsOnEmptinessConsistentTouch(t *testing.T) {
	addr := common.Address{0x07}
	first := foldList(AccountAccess{
		Address:      addr,
		NonceChanges: []encodingAccountNonce{{BlockAccessIndex: 1, PostNonce: 0}},
		CodeChanges:  []encodingCodeChange{{BlockAccessIndex: 1, NewCode: []byte{}}},
	})
	second := foldList(AccountAccess{
		Address:        addr,
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 1, PostBalance: u256(5)}},
	})

	if _, n := Fold([]*BlockAccessList{first, second}); n != 1 {
		t.Fatalf("folded %d lists, want 1 (split before the recreation)", n)
	}
}

// TestFoldNoSplitOnLiveAccount pins the guard's other side: a nonzero nonce
// proves the account alive, so a later touch folds through.
func TestFoldNoSplitOnLiveAccount(t *testing.T) {
	addr := common.Address{0x08}
	first := foldList(AccountAccess{
		Address:      addr,
		NonceChanges: []encodingAccountNonce{{BlockAccessIndex: 1, PostNonce: 1}},
	})
	second := foldList(AccountAccess{
		Address:        addr,
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 1, PostBalance: u256(5)}},
	})

	if _, n := Fold([]*BlockAccessList{first, second}); n != 2 {
		t.Fatalf("folded %d lists, want 2 (the account is provably alive)", n)
	}
}
