// Copyright 2025 The go-ethereum Authors
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
	"bytes"
	"cmp"
	"reflect"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func makeTestConstructionBAL() *ConstructionBlockAccessList {
	return &ConstructionBlockAccessList{
		map[common.Address]*ConstructionAccountAccess{
			common.BytesToAddress([]byte{0xff, 0xff}): {
				StorageWrites: map[common.Hash]map[uint32]common.Hash{
					common.BytesToHash([]byte{0x01}): {
						1: common.BytesToHash([]byte{1, 2, 3, 4}),
						2: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
					},
					common.BytesToHash([]byte{0x10}): {
						20: common.BytesToHash([]byte{1, 2, 3, 4}),
					},
				},
				StorageReads: map[common.Hash]struct{}{
					common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7}): {},
				},
				BalanceChanges: map[uint32]*uint256.Int{
					1: uint256.NewInt(100),
					2: uint256.NewInt(500),
				},
				NonceChanges: map[uint32]uint64{
					1: 2,
					2: 6,
				},
				CodeChange: map[uint32][]byte{
					0: common.Hex2Bytes("deadbeef"),
				},
			},
			common.BytesToAddress([]byte{0xff, 0xff, 0xff}): {
				StorageWrites: map[common.Hash]map[uint32]common.Hash{
					common.BytesToHash([]byte{0x01}): {
						2: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
						3: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
					},
					common.BytesToHash([]byte{0x10}): {
						21: common.BytesToHash([]byte{1, 2, 3, 4, 5}),
					},
				},
				StorageReads: map[common.Hash]struct{}{
					common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}): {},
				},
				BalanceChanges: map[uint32]*uint256.Int{
					2: uint256.NewInt(100),
					3: uint256.NewInt(500),
				},
				NonceChanges: map[uint32]uint64{
					1: 2,
				},
				CodeChange: map[uint32][]byte{
					0: common.Hex2Bytes("deadbeef"),
				},
			},
		},
	}
}

// TestBALEncoding tests that a populated access list can be encoded/decoded correctly.
func TestBALEncoding(t *testing.T) {
	var buf bytes.Buffer
	bal := makeTestConstructionBAL()
	err := bal.EncodeRLP(&buf)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	var dec BlockAccessList
	if err := dec.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 0)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
	if dec.Hash() != bal.ToEncodingObj().Hash() {
		t.Fatalf("encoded block hash doesn't match decoded")
	}
	if !reflect.DeepEqual(bal.ToEncodingObj(), &dec) {
		t.Fatal("decoded BAL doesn't match")
	}
}

func TestConstructionBALMerge(t *testing.T) {
	var (
		addrA = common.BytesToAddress([]byte{0xAA})
		addrB = common.BytesToAddress([]byte{0xBB})
		slot1 = common.BytesToHash([]byte{0x01})
		slot2 = common.BytesToHash([]byte{0x02})
		slot3 = common.BytesToHash([]byte{0x03})
	)
	a := NewConstructionBlockAccessList()
	a.StorageWrite(1, addrA, slot1, common.BytesToHash([]byte{0x11}))
	a.StorageRead(addrA, slot2) // demoted by other's write below
	a.BalanceChange(1, addrA, uint256.NewInt(100))
	a.NonceChange(addrA, 1, 7)

	b := NewConstructionBlockAccessList()
	b.StorageWrite(2, addrA, slot1, common.BytesToHash([]byte{0x22})) // same slot, disjoint txIdx
	b.StorageWrite(2, addrA, slot2, common.BytesToHash([]byte{0x33}))
	b.StorageRead(addrA, slot3)
	b.BalanceChange(2, addrA, uint256.NewInt(200))
	b.NonceChange(addrA, 2, 8)
	b.CodeChange(addrB, 2, []byte{0xde, 0xad}) // account only in other

	a.Merge(b)

	accA := a.Accounts[addrA]
	wantWrites := map[common.Hash]map[uint32]common.Hash{
		slot1: {1: common.BytesToHash([]byte{0x11}), 2: common.BytesToHash([]byte{0x22})},
		slot2: {2: common.BytesToHash([]byte{0x33})},
	}
	if !reflect.DeepEqual(accA.StorageWrites, wantWrites) {
		t.Fatalf("storage writes mismatch: got %v, want %v", accA.StorageWrites, wantWrites)
	}
	wantReads := map[common.Hash]struct{}{slot3: {}}
	if !reflect.DeepEqual(accA.StorageReads, wantReads) {
		t.Fatalf("storage reads mismatch: got %v, want %v", accA.StorageReads, wantReads)
	}
	if accA.BalanceChanges[1].Uint64() != 100 || accA.BalanceChanges[2].Uint64() != 200 {
		t.Fatalf("balance changes mismatch: %v", accA.BalanceChanges)
	}
	if accA.NonceChanges[1] != 7 || accA.NonceChanges[2] != 8 {
		t.Fatalf("nonce changes mismatch: %v", accA.NonceChanges)
	}
	accB, ok := a.Accounts[addrB]
	if !ok {
		t.Fatal("account only present in other was not adopted")
	}
	if !bytes.Equal(accB.CodeChange[2], []byte{0xde, 0xad}) {
		t.Fatalf("code change for adopted account missing: %x", accB.CodeChange[2])
	}
}

func makeTestAccountAccess(sort bool) AccountAccess {
	var (
		storageWrites []encodingSlotWrites
		storageReads  []*uint256.Int
		balances      []encodingBalanceChange
		nonces        []encodingAccountNonce
		codes         []encodingCodeChange
	)
	randSlot := func() *uint256.Int {
		return new(uint256.Int).SetBytes(testrand.Bytes(32))
	}
	for i := 0; i < 5; i++ {
		slot := encodingSlotWrites{
			Slot: randSlot(),
		}
		for j := 0; j < 3; j++ {
			slot.Accesses = append(slot.Accesses, encodingStorageWrite{
				TxIdx:      uint32(2 * j),
				ValueAfter: randSlot(),
			})
		}
		if sort {
			slices.SortFunc(slot.Accesses, func(a, b encodingStorageWrite) int {
				return cmp.Compare[uint32](a.TxIdx, b.TxIdx)
			})
		}
		storageWrites = append(storageWrites, slot)
	}
	if sort {
		slices.SortFunc(storageWrites, func(a, b encodingSlotWrites) int {
			return a.Slot.Cmp(b.Slot)
		})
	}

	for i := 0; i < 5; i++ {
		storageReads = append(storageReads, randSlot())
	}
	if sort {
		slices.SortFunc(storageReads, func(a, b *uint256.Int) int {
			return a.Cmp(b)
		})
	}

	for i := 0; i < 5; i++ {
		balances = append(balances, encodingBalanceChange{
			TxIdx:   uint32(2 * i),
			Balance: new(uint256.Int).SetBytes(testrand.Bytes(16)),
		})
	}
	if sort {
		slices.SortFunc(balances, func(a, b encodingBalanceChange) int {
			return cmp.Compare[uint32](a.TxIdx, b.TxIdx)
		})
	}

	for i := 0; i < 5; i++ {
		nonces = append(nonces, encodingAccountNonce{
			TxIdx: uint32(2 * i),
			Nonce: uint64(i + 100),
		})
	}
	if sort {
		slices.SortFunc(nonces, func(a, b encodingAccountNonce) int {
			return cmp.Compare[uint32](a.TxIdx, b.TxIdx)
		})
	}

	for i := 0; i < 5; i++ {
		codes = append(codes, encodingCodeChange{
			TxIndex: uint32(2 * i),
			Code:    testrand.Bytes(256),
		})
	}
	if sort {
		slices.SortFunc(codes, func(a, b encodingCodeChange) int {
			return cmp.Compare[uint32](a.TxIndex, b.TxIndex)
		})
	}

	return AccountAccess{
		Address:        [20]byte(testrand.Bytes(20)),
		StorageWrites:  storageWrites,
		StorageReads:   storageReads,
		BalanceChanges: balances,
		NonceChanges:   nonces,
		CodeChanges:    codes,
	}
}

func makeTestBAL(sort bool) *BlockAccessList {
	list := make(BlockAccessList, 0, 5)
	for i := 0; i < 5; i++ {
		list = append(list, makeTestAccountAccess(sort))
	}
	if sort {
		slices.SortFunc(list, func(a, b AccountAccess) int {
			return bytes.Compare(a.Address[:], b.Address[:])
		})
	}
	return &list
}

func TestBlockAccessListCopy(t *testing.T) {
	list := makeTestBAL(true)
	cpy := list.Copy()
	cpyCpy := cpy.Copy()

	if !reflect.DeepEqual(list, cpy) {
		t.Fatal("block access mismatch")
	}
	if !reflect.DeepEqual(cpy, cpyCpy) {
		t.Fatal("block access mismatch")
	}

	// Make sure the mutations on copy won't affect the origin
	for _, aa := range *cpyCpy {
		for i := 0; i < len(aa.StorageReads); i++ {
			aa.StorageReads[i] = new(uint256.Int).SetBytes(testrand.Bytes(32))
		}
	}
	if !reflect.DeepEqual(list, cpy) {
		t.Fatal("block access mismatch")
	}
}

func TestBlockAccessListItemCount(t *testing.T) {
	empty := &BlockAccessList{}
	if got := empty.ItemCount(); got != 0 {
		t.Fatalf("empty BAL item count: got %d, want 0", got)
	}

	addr1 := [20]byte(testrand.Bytes(20))
	addr2 := [20]byte(testrand.Bytes(20))
	one := func() *uint256.Int { return new(uint256.Int).SetBytes(testrand.Bytes(32)) }
	bal := &BlockAccessList{
		AccountAccess{
			Address: addr1,
			StorageWrites: []encodingSlotWrites{
				{Slot: one(), Accesses: []encodingStorageWrite{{TxIdx: 0, ValueAfter: one()}, {TxIdx: 1, ValueAfter: one()}}},
				{Slot: one()},
			},
			StorageReads: []*uint256.Int{one()},
		},
		AccountAccess{Address: addr2}, // address-only, no slots
	}
	// 2 addresses + 2 write-slots + 1 read-slot = 5 items.
	// (Multiple TxIdx writes to the same slot count as ONE item.)
	if got := bal.ItemCount(); got != 5 {
		t.Fatalf("item count: got %d, want 5", got)
	}
}

func TestBlockAccessListValidateSize(t *testing.T) {
	// Build a BAL with exactly 30 items: 3 addresses, each with 9 storage
	// slots (some writes, some reads). 3 + 9*3 = 30.
	one := func() *uint256.Int { return new(uint256.Int).SetBytes(testrand.Bytes(32)) }
	bal := make(BlockAccessList, 3)
	for i := range bal {
		bal[i].Address = [20]byte(testrand.Bytes(20))
		for j := 0; j < 5; j++ {
			bal[i].StorageWrites = append(bal[i].StorageWrites, encodingSlotWrites{
				Slot: one(), Accesses: []encodingStorageWrite{{TxIdx: 0, ValueAfter: one()}},
			})
		}
		for j := 0; j < 4; j++ {
			bal[i].StorageReads = append(bal[i].StorageReads, one())
		}
	}
	if got := bal.ItemCount(); got != 30 {
		t.Fatalf("setup: item count = %d, want 30", got)
	}

	// limit = blockGasLimit / BALItemCost.
	// 30 items requires limit >= 30, i.e. gasLimit >= 30 * 2000 = 60_000.
	tests := []struct {
		name        string
		gasLimit    uint64
		expectError bool
	}{
		{"exactly at limit", 30 * params.BALItemCost, false},
		{"well above limit", 60_000_000, false},
		{"one below limit", 30*params.BALItemCost - 1, true},
		{"zero gas limit", 0, true},
	}
	for _, tc := range tests {
		err := bal.ValidateSize(tc.gasLimit)
		if (err != nil) != tc.expectError {
			t.Errorf("%s: got err=%v, expectError=%v", tc.name, err, tc.expectError)
		}
	}

	// Empty BAL is always valid (even with 0 gas limit).
	if err := (&BlockAccessList{}).ValidateSize(0); err != nil {
		t.Fatalf("empty BAL must pass any limit: %v", err)
	}
}

func TestBlockAccessListValidation(t *testing.T) {
	// Validate the block access list after RLP decoding
	enc := makeTestBAL(true)
	if err := enc.Validate(); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
	var buf bytes.Buffer
	if err := enc.EncodeRLP(&buf); err != nil {
		t.Fatalf("Unexpected encoding error: %v", err)
	}

	var dec BlockAccessList
	if err := dec.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 0)); err != nil {
		t.Fatalf("Unexpected RLP-decode error: %v", err)
	}
	if err := dec.Validate(); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}

	// Validate the derived block access list
	cBAL := makeTestConstructionBAL()
	listB := cBAL.ToEncodingObj()
	if err := listB.Validate(); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
}
