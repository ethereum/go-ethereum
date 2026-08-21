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

// TestFoldRemovalSplit pins the split-before-recreation guard: split when the
// folded metadata is emptiness-consistent, fold through when provably alive.
func TestFoldRemovalSplit(t *testing.T) {
	addr := common.Address{0x02}
	touch := foldList(AccountAccess{
		Address:        addr,
		BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 1, PostBalance: u256(5)}},
	})
	for _, tc := range []struct {
		name  string
		first AccountAccess
		want  int
	}{
		{"zero balance", AccountAccess{
			Address:        addr,
			BalanceChanges: []encodingBalanceChange{{BlockAccessIndex: 3, PostBalance: u256(0)}},
		}, 1},
		{"empty code, zero nonce", AccountAccess{
			Address:      addr,
			NonceChanges: []encodingAccountNonce{{BlockAccessIndex: 1, PostNonce: 0}},
			CodeChanges:  []encodingCodeChange{{BlockAccessIndex: 1, NewCode: []byte{}}},
		}, 1},
		{"alive nonce", AccountAccess{
			Address:      addr,
			NonceChanges: []encodingAccountNonce{{BlockAccessIndex: 1, PostNonce: 1}},
		}, 2},
	} {
		if _, n := Fold([]*BlockAccessList{foldList(tc.first), touch}); n != tc.want {
			t.Fatalf("%s: folded %d lists, want %d", tc.name, n, tc.want)
		}
	}
}

// TestFoldRemovalMarkerIsLatestOnly pins that only the latest value counts.
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

// TestFoldDropsReadOnlyAccounts pins that accounts with no mutation vanish.
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
// change and must survive folding as one.
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

// TestFoldEmptyInput pins the degenerate shapes.
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
