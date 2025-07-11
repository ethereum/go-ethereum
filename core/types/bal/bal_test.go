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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func equalBALs(a *BlockAccessList, b *BlockAccessList) bool {
	if !reflect.DeepEqual(a, b) {
		return false
	}
	return true
}

func makeTestConstructionBAL() *ConstructionBlockAccessList {
	return &ConstructionBlockAccessList{
		map[common.Address]*ConstructionAccountAccess{
			common.BytesToAddress([]byte{0xff, 0xff}): {
				StorageWrites: map[common.Hash]map[uint16]common.Hash{
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
				BalanceChanges: map[uint16]*uint256.Int{
					1: uint256.NewInt(100),
					2: uint256.NewInt(500),
				},
				NonceChanges: map[uint16]uint64{
					1: 2,
					2: 6,
				},
				CodeChange: &CodeChange{
					TxIndex: 0,
					Code:    common.Hex2Bytes("deadbeef"),
				},
			},
			common.BytesToAddress([]byte{0xff, 0xff, 0xff}): {
				StorageWrites: map[common.Hash]map[uint16]common.Hash{
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
				BalanceChanges: map[uint16]*uint256.Int{
					2: uint256.NewInt(100),
					3: uint256.NewInt(500),
				},
				NonceChanges: map[uint16]uint64{
					1: 2,
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
	if err := dec.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
	if dec.Hash() != bal.toEncodingObj().Hash() {
		t.Fatalf("encoded block hash doesn't match decoded")
	}
	if !equalBALs(bal.toEncodingObj(), &dec) {
		t.Fatal("decoded BAL doesn't match")
	}
}

func makeTestAccountAccess(sort bool) AccountAccess {
	var (
		storageWrites []encodingSlotWrites
		storageReads  [][32]byte
		balances      []encodingBalanceChange
		nonces        []encodingAccountNonce
	)
	for i := 0; i < 5; i++ {
		slot := encodingSlotWrites{
			Slot: testrand.Hash(),
		}
		for j := 0; j < 3; j++ {
			slot.Accesses = append(slot.Accesses, encodingStorageWrite{
				TxIdx:      uint16(2 * j),
				ValueAfter: testrand.Hash(),
			})
		}
		if sort {
			slices.SortFunc(slot.Accesses, func(a, b encodingStorageWrite) int {
				return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
			})
		}
		storageWrites = append(storageWrites, slot)
	}
	if sort {
		slices.SortFunc(storageWrites, func(a, b encodingSlotWrites) int {
			return bytes.Compare(a.Slot[:], b.Slot[:])
		})
	}

	for i := 0; i < 5; i++ {
		storageReads = append(storageReads, testrand.Hash())
	}
	if sort {
		slices.SortFunc(storageReads, func(a, b [32]byte) int {
			return bytes.Compare(a[:], b[:])
		})
	}

	for i := 0; i < 5; i++ {
		balances = append(balances, encodingBalanceChange{
			TxIdx:   uint16(2 * i),
			Balance: [16]byte(testrand.Bytes(16)),
		})
	}
	if sort {
		slices.SortFunc(balances, func(a, b encodingBalanceChange) int {
			return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
		})
	}

	for i := 0; i < 5; i++ {
		nonces = append(nonces, encodingAccountNonce{
			TxIdx: uint16(2 * i),
			Nonce: uint64(i + 100),
		})
	}
	if sort {
		slices.SortFunc(nonces, func(a, b encodingAccountNonce) int {
			return cmp.Compare[uint16](a.TxIdx, b.TxIdx)
		})
	}

	return AccountAccess{
		Address:        [20]byte(testrand.Bytes(20)),
		StorageWrites:  storageWrites,
		StorageReads:   storageReads,
		BalanceChanges: balances,
		NonceChanges:   nonces,
		Code: []CodeChange{
			{
				TxIndex: 100,
				Code:    testrand.Bytes(256),
			},
		},
	}
}

func makeTestBAL(sort bool) BlockAccessList {
	list := BlockAccessList{}
	for i := 0; i < 5; i++ {
		list.Accesses = append(list.Accesses, makeTestAccountAccess(sort))
	}
	if sort {
		slices.SortFunc(list.Accesses, func(a, b AccountAccess) int {
			return bytes.Compare(a.Address[:], b.Address[:])
		})
	}
	return list
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
	for _, aa := range cpyCpy.Accesses {
		for i := 0; i < len(aa.StorageReads); i++ {
			aa.StorageReads[i] = [32]byte(testrand.Bytes(32))
		}
	}
	if !reflect.DeepEqual(list, cpy) {
		t.Fatal("block access mismatch")
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
	listB := cBAL.toEncodingObj()
	if err := listB.Validate(); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
}
