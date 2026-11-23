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
	"math"
	"reflect"
	"slices"
	"strings"
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

// TestConstructionBALMergeRestores checks that the entries restored to their
// value from before a tx index revoke what was recorded at that index, and
// nothing else, also when the restoring list is merged in several hops.
func TestConstructionBALMergeRestores(t *testing.T) {
	var (
		addr  = common.BytesToAddress([]byte{0xAA})
		slot1 = common.BytesToHash([]byte{0x01}) // written at an earlier index too
		slot2 = common.BytesToHash([]byte{0x02}) // only written at the shared index
		slot3 = common.BytesToHash([]byte{0x03}) // restored, then written again
		value = common.BytesToHash([]byte{0x11})
	)
	block := NewConstructionBlockAccessList()
	for _, idx := range []uint32{1, 2} {
		block.StorageWrite(idx, addr, slot1, value)
		block.BalanceChange(idx, addr, uint256.NewInt(100))
		block.NonceChange(addr, idx, 7)
		block.CodeChange(addr, idx, []byte{0xde, 0xad})
	}
	block.StorageWrite(2, addr, slot2, value)
	block.StorageWrite(2, addr, slot3, value)

	restore := NewConstructionBlockAccessList()
	restore.StorageRestore(2, addr, slot1)
	restore.StorageRestore(2, addr, slot2)
	restore.StorageRestore(2, addr, slot3)
	restore.BalanceRestore(2, addr)
	restore.NonceRestore(addr, 2)
	restore.CodeRestore(addr, 2)

	again := NewConstructionBlockAccessList()
	again.StorageWrite(2, addr, slot3, value)
	again.BalanceChange(2, addr, uint256.NewInt(300))

	// The scopes of an index are gathered first, as the pre- and
	// post-execution do, and then handed over to the block-level list.
	post := NewConstructionBlockAccessList()
	post.AccountRead(addr)
	post.Merge(restore.Copy())
	post.Merge(again)
	block.Merge(post)

	acc := block.Accounts[addr]
	wantWrites := map[common.Hash]map[uint32]common.Hash{
		slot1: {1: value},
		slot3: {2: value},
	}
	if !reflect.DeepEqual(acc.StorageWrites, wantWrites) {
		t.Fatalf("storage writes mismatch: got %v, want %v", acc.StorageWrites, wantWrites)
	}
	wantReads := map[common.Hash]struct{}{slot2: {}}
	if !reflect.DeepEqual(acc.StorageReads, wantReads) {
		t.Fatalf("storage reads mismatch: got %v, want %v", acc.StorageReads, wantReads)
	}
	if len(acc.BalanceChanges) != 2 || acc.BalanceChanges[1].Uint64() != 100 || acc.BalanceChanges[2].Uint64() != 300 {
		t.Fatalf("balance changes mismatch: %v", acc.BalanceChanges)
	}
	if len(acc.NonceChanges) != 1 || acc.NonceChanges[1] != 7 {
		t.Fatalf("nonce changes mismatch: %v", acc.NonceChanges)
	}
	if len(acc.CodeChange) != 1 || !bytes.Equal(acc.CodeChange[1], []byte{0xde, 0xad}) {
		t.Fatalf("code changes mismatch: %v", acc.CodeChange)
	}
}

func makeTestAccountAccess(sort bool) AccountAccess {
	var (
		storageWrites []encodingSlotChanges
		storageReads  []*uint256.Int
		balances      []encodingBalanceChange
		nonces        []encodingAccountNonce
		codes         []encodingCodeChange
	)
	randSlot := func() *uint256.Int {
		return new(uint256.Int).SetBytes(testrand.Bytes(32))
	}
	for i := 0; i < 5; i++ {
		slot := encodingSlotChanges{
			Slot: randSlot(),
		}
		for j := 0; j < 3; j++ {
			slot.SlotChanges = append(slot.SlotChanges, encodingStorageWrite{
				BlockAccessIndex: uint32(2 * j),
				PostValue:        randSlot(),
			})
		}
		if sort {
			slices.SortFunc(slot.SlotChanges, func(a, b encodingStorageWrite) int {
				return cmp.Compare(a.BlockAccessIndex, b.BlockAccessIndex)
			})
		}
		storageWrites = append(storageWrites, slot)
	}
	if sort {
		slices.SortFunc(storageWrites, func(a, b encodingSlotChanges) int {
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
			BlockAccessIndex: uint32(2 * i),
			PostBalance:      new(uint256.Int).SetBytes(testrand.Bytes(16)),
		})
	}
	if sort {
		slices.SortFunc(balances, func(a, b encodingBalanceChange) int {
			return cmp.Compare(a.BlockAccessIndex, b.BlockAccessIndex)
		})
	}

	for i := 0; i < 5; i++ {
		nonces = append(nonces, encodingAccountNonce{
			BlockAccessIndex: uint32(2 * i),
			PostNonce:        uint64(i + 100),
		})
	}
	if sort {
		slices.SortFunc(nonces, func(a, b encodingAccountNonce) int {
			return cmp.Compare(a.BlockAccessIndex, b.BlockAccessIndex)
		})
	}

	for i := 0; i < 5; i++ {
		codes = append(codes, encodingCodeChange{
			BlockAccessIndex: uint32(2 * i),
			NewCode:          testrand.Bytes(256),
		})
	}
	if sort {
		slices.SortFunc(codes, func(a, b encodingCodeChange) int {
			return cmp.Compare(a.BlockAccessIndex, b.BlockAccessIndex)
		})
	}

	return AccountAccess{
		Address:        common.Address(testrand.Bytes(20)),
		StorageChanges: storageWrites,
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
	if got := empty.itemCount(); got != 0 {
		t.Fatalf("empty BAL item count: got %d, want 0", got)
	}

	addr1 := common.Address(testrand.Bytes(20))
	addr2 := common.Address(testrand.Bytes(20))
	one := func() *uint256.Int { return new(uint256.Int).SetBytes(testrand.Bytes(32)) }
	bal := &BlockAccessList{
		AccountAccess{
			Address: addr1,
			StorageChanges: []encodingSlotChanges{
				{Slot: one(), SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 0, PostValue: one()}, {BlockAccessIndex: 1, PostValue: one()}}},
				{Slot: one()},
			},
			StorageReads: []*uint256.Int{one()},
		},
		AccountAccess{Address: addr2}, // address-only, no slots
	}
	// 2 addresses + 2 write-slots + 1 read-slot = 5 items.
	// (Multiple TxIdx writes to the same slot count as ONE item.)
	if got := bal.itemCount(); got != 5 {
		t.Fatalf("item count: got %d, want 5", got)
	}
}

func TestBlockAccessListValidateSize(t *testing.T) {
	// Build a BAL with exactly 30 items: 3 addresses, each with 9 storage
	// slots (some writes, some reads). 3 + 9*3 = 30.
	one := func() *uint256.Int { return new(uint256.Int).SetBytes(testrand.Bytes(32)) }
	bal := make(BlockAccessList, 3)
	for i := range bal {
		bal[i].Address = common.Address(testrand.Bytes(20))
		for j := 0; j < 5; j++ {
			bal[i].StorageChanges = append(bal[i].StorageChanges, encodingSlotChanges{
				Slot: one(), SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 0, PostValue: one()}},
			})
		}
		for j := 0; j < 4; j++ {
			bal[i].StorageReads = append(bal[i].StorageReads, one())
		}
	}
	if got := bal.itemCount(); got != 30 {
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
	if err := enc.Validate(math.MaxUint64, 10000); err != nil {
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
	if err := dec.Validate(math.MaxUint64, 10000); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}

	// Validate the derived block access list
	cBAL := makeTestConstructionBAL()
	listB := cBAL.ToEncodingObj()
	if err := listB.Validate(math.MaxUint64, 10000); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
}

func TestBlockAccessListValidationErrors(t *testing.T) {
	var (
		addr1 = common.HexToAddress("0x01")
		addr2 = common.HexToAddress("0x02")
		u     = uint256.NewInt
	)
	write := func(slot uint64, indices ...uint32) encodingSlotChanges {
		sc := encodingSlotChanges{Slot: u(slot)}
		for _, idx := range indices {
			sc.SlotChanges = append(sc.SlotChanges, encodingStorageWrite{BlockAccessIndex: idx, PostValue: u(1)})
		}
		return sc
	}
	tests := []struct {
		name string
		bal  BlockAccessList
		want string
	}{
		{"unsorted accounts", BlockAccessList{{Address: addr2}, {Address: addr1}}, "not in lexicographic order"},
		{"duplicate accounts", BlockAccessList{{Address: addr1}, {Address: addr1}}, "not in lexicographic order"},
		{"unsorted write slots", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(2, 1), write(1, 1)}}}, "storage write slots must be unique and sorted"},
		{"duplicate write slots", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(1, 1), write(1, 2)}}}, "storage write slots must be unique and sorted"},
		{"empty slot changes", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(1)}}}, "empty slot changes"},
		{"unsorted slot change indices", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(1, 2, 1)}}}, "storage write indexes must be unique and sorted"},
		{"slot change index too high", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(1, 4)}}}, "storage write index exceeds limit"},
		{"unsorted reads", BlockAccessList{{Address: addr1, StorageReads: []*uint256.Int{u(2), u(1)}}}, "storage read slots must be unique and sorted"},
		{"duplicate reads", BlockAccessList{{Address: addr1, StorageReads: []*uint256.Int{u(1), u(1)}}}, "storage read slots must be unique and sorted"},
		{"first slot read and written", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(1, 1), write(5, 1)}, StorageReads: []*uint256.Int{u(1), u(7)}}}, "both read/write sets"},
		{"middle slot read and written", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(2, 1), write(5, 1), write(9, 1)}, StorageReads: []*uint256.Int{u(1), u(5), u(8)}}}, "both read/write sets"},
		{"last slot read and written", BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(9, 1)}, StorageReads: []*uint256.Int{u(1), u(9)}}}, "both read/write sets"},
		{"unsorted balance changes", BlockAccessList{{Address: addr1, BalanceChanges: []encodingBalanceChange{{2, u(1)}, {1, u(1)}}}}, "balance changes must be unique and sorted"},
		{"balance change index too high", BlockAccessList{{Address: addr1, BalanceChanges: []encodingBalanceChange{{4, u(1)}}}}, "balance change index exceeds limit"},
		{"duplicate nonce changes", BlockAccessList{{Address: addr1, NonceChanges: []encodingAccountNonce{{1, 1}, {1, 2}}}}, "nonce changes must be unique and sorted"},
		{"nonce change index too high", BlockAccessList{{Address: addr1, NonceChanges: []encodingAccountNonce{{4, 1}}}}, "nonce change index exceeds limit"},
		{"unsorted code changes", BlockAccessList{{Address: addr1, CodeChanges: []encodingCodeChange{{2, nil}, {1, nil}}}}, "code changes must be unique and sorted"},
		{"code change index too high", BlockAccessList{{Address: addr1, CodeChanges: []encodingCodeChange{{4, nil}}}}, "code change index exceeds limit"},
		{"oversized code", BlockAccessList{{Address: addr1, CodeChanges: []encodingCodeChange{{1, make([]byte, params.MaxCodeSizeAmsterdam+1)}}}}, "oversized code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Two transactions, so block access indices 0 to 3 are allowed.
			err := tt.bal.Validate(math.MaxUint64, 2)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("have error %v, want %q", err, tt.want)
			}
		})
	}
	// Interleaved, disjoint read and write sets are valid.
	valid := BlockAccessList{{Address: addr1, StorageChanges: []encodingSlotChanges{write(2, 0, 3), write(4, 1)}, StorageReads: []*uint256.Int{u(1), u(3), u(5)}}}
	if err := valid.Validate(math.MaxUint64, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// BenchmarkBlockAccessListValidate measures validation of an access list with
// 10000 items, where each account has a few written and read slots.
func BenchmarkBlockAccessListValidate(b *testing.B) {
	c := NewConstructionBlockAccessList()
	for i := range 2500 {
		addr := common.BytesToAddress(testrand.Bytes(20))
		c.StorageWrite(uint32(i%100+1), addr, testrand.Hash(), testrand.Hash())
		c.StorageRead(addr, testrand.Hash())
		c.StorageRead(addr, testrand.Hash())
		c.BalanceChange(uint32(i%100+1), addr, uint256.NewInt(uint64(i)))
	}
	list := c.ToEncodingObj()

	b.ReportAllocs()
	for b.Loop() {
		if err := list.Validate(math.MaxUint64, 100); err != nil {
			b.Fatal(err)
		}
	}
}
