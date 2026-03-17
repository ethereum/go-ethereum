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
	"fmt"
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

func makeTestConstructionBAL() *AccessListBuilder {
	return &AccessListBuilder{
		FinalizedAccesses: map[common.Address]*ConstructionAccountAccesses{
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
				CodeChanges: map[uint16]CodeChange{0: {
					TxIdx: 0,
					Code:  common.Hex2Bytes("deadbeef"),
				}},
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
	balBuilder := makeTestConstructionBAL()
	bal := balBuilder.FinalizedAccesses
	err := bal.EncodeRLP(&buf)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	var dec BlockAccessList
	if err := dec.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
	if dec.Hash() != bal.ToEncodingObj().Hash() {
		t.Fatalf("encoded block hash doesn't match decoded")
	}
	if !equalBALs(bal.ToEncodingObj(), &dec) {
		t.Fatal("decoded BAL doesn't match")
	}
}

func makeTestAccountAccess(sort bool) AccountAccess {
	var (
		storageWrites []encodingSlotWrites
		storageReads  []common.Hash
		balances      []encodingBalanceChange
		nonces        []encodingAccountNonce
	)
	for i := 0; i < 5; i++ {
		slot := encodingSlotWrites{
			Slot: newEncodedStorageFromHash(testrand.Hash()),
		}
		for j := 0; j < 3; j++ {
			slot.Accesses = append(slot.Accesses, encodingStorageWrite{
				TxIdx:      uint16(2 * j),
				ValueAfter: newEncodedStorageFromHash(testrand.Hash()),
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
			return bytes.Compare(a.Slot.inner.Bytes(), b.Slot.inner.Bytes())
		})
	}

	for i := 0; i < 5; i++ {
		storageReads = append(storageReads, testrand.Hash())
	}
	if sort {
		slices.SortFunc(storageReads, func(a, b common.Hash) int {
			return bytes.Compare(a[:], b[:])
		})
	}

	for i := 0; i < 5; i++ {
		balances = append(balances, encodingBalanceChange{
			TxIdx:   uint16(2 * i),
			Balance: new(uint256.Int).SetBytes(testrand.Bytes(32)),
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

	var encodedStorageReads []*EncodedStorage
	for _, slot := range storageReads {
		encodedStorageReads = append(encodedStorageReads, newEncodedStorageFromHash(slot))
	}
	return AccountAccess{
		Address:        [20]byte(testrand.Bytes(20)),
		StorageChanges: storageWrites,
		StorageReads:   encodedStorageReads,
		BalanceChanges: balances,
		NonceChanges:   nonces,
		CodeChanges: []CodeChange{
			{
				TxIdx: 100,
				Code:  testrand.Bytes(256),
			},
		},
	}
}

func makeTestBAL(sort bool) BlockAccessList {
	list := BlockAccessList{}
	for i := 0; i < 5; i++ {
		list = append(list, makeTestAccountAccess(sort))
	}
	if sort {
		slices.SortFunc(list, func(a, b AccountAccess) int {
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
	for _, aa := range cpyCpy {
		for i := 0; i < len(aa.StorageReads); i++ {
			aa.StorageReads[i] = &EncodedStorage{new(uint256.Int).SetBytes(testrand.Bytes(32))}
		}
	}
	if !reflect.DeepEqual(list, cpy) {
		t.Fatal("block access mismatch")
	}
}

func TestBlockAccessListValidation(t *testing.T) {
	// Validate the block access list after RLP decoding
	testBALMaxIndex := 8
	enc := makeTestBAL(true)
	if err := enc.Validate(testBALMaxIndex); err != nil {
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
	if err := dec.Validate(testBALMaxIndex); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}

	// Validate the derived block access list
	cBAL := makeTestConstructionBAL().FinalizedAccesses
	listB := cBAL.ToEncodingObj()
	if err := listB.Validate(testBALMaxIndex); err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
}

// TestLazyScopeCorrectness verifies that lazy scope allocation produces
// identical results to the previous eager allocation for mixed workloads:
// precompile calls (empty scopes) interspersed with state-changing calls.
func TestLazyScopeCorrectness(t *testing.T) {
	builder := newAccessListBuilder()
	sender := common.HexToAddress("0x1234")
	contract := common.HexToAddress("0x5678")
	precompile := common.HexToAddress("0x06")

	// Tx-level: sender balance/nonce
	builder.balanceChange(sender, uint256.NewInt(1000), uint256.NewInt(900))
	builder.nonceChange(sender, 0, 1)

	// Enter contract scope
	builder.enterScope()
	builder.storageRead(contract, common.HexToHash("0x01"))

	// Precompile STATICCALL (empty scope)
	builder.enterScope()
	builder.exitScope(false)

	// Another precompile call
	builder.enterScope()
	builder.exitScope(false)

	// Contract writes storage
	builder.storageWrite(contract, common.HexToHash("0x02"), common.Hash{}, common.HexToHash("0xff"))

	// Precompile that reverts (still empty scope, reverted)
	builder.enterScope()
	builder.exitScope(true)

	// Nested call to another contract
	builder.enterScope()
	builder.balanceChange(precompile, uint256.NewInt(0), uint256.NewInt(100))
	builder.exitScope(false)

	// Exit contract scope
	builder.exitScope(false)

	diff, accesses := builder.finalise()

	// Verify sender mutations
	senderMut, ok := diff.Mutations[sender]
	if !ok {
		t.Fatal("sender not in mutations")
	}
	if senderMut.Balance == nil || !senderMut.Balance.Eq(uint256.NewInt(900)) {
		t.Fatalf("sender balance mismatch: got %v", senderMut.Balance)
	}
	if senderMut.Nonce == nil || *senderMut.Nonce != 1 {
		t.Fatalf("sender nonce mismatch: got %v", senderMut.Nonce)
	}

	// Verify contract mutations (storage write)
	contractMut, ok := diff.Mutations[contract]
	if !ok {
		t.Fatal("contract not in mutations")
	}
	if contractMut.StorageWrites == nil {
		t.Fatal("contract has no storage writes")
	}
	if contractMut.StorageWrites[common.HexToHash("0x02")] != common.HexToHash("0xff") {
		t.Fatal("contract storage write mismatch")
	}

	// Verify precompile balance change
	precompileMut, ok := diff.Mutations[precompile]
	if !ok {
		t.Fatal("precompile not in mutations")
	}
	if precompileMut.Balance == nil || !precompileMut.Balance.Eq(uint256.NewInt(100)) {
		t.Fatalf("precompile balance mismatch: got %v", precompileMut.Balance)
	}

	// Verify contract storage read is in accesses
	contractAccesses, ok := accesses[contract]
	if !ok {
		t.Fatal("contract not in accesses")
	}
	if _, ok := contractAccesses[common.HexToHash("0x01")]; !ok {
		t.Fatal("contract storage read not in accesses")
	}
}

// BenchmarkPrecompileScopes simulates a precompile-heavy transaction where
// STATICCALL is invoked thousands of times against a precompile (e.g. bn128_add).
// Each call creates a scope (EnterScope) that records no state changes (precompiles
// don't touch state), then exits (ExitScope). This benchmark measures the overhead
// of scope tracking for such workloads.
func BenchmarkPrecompileScopes(b *testing.B) {
	for _, numCalls := range []int{100, 1000, 10000, 100000} {
		b.Run(fmt.Sprintf("calls=%d", numCalls), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := newAccessListBuilder()
				// Simulate a transaction: sender balance/nonce change at depth 0,
				// then thousands of precompile STATICCALL scopes that touch no state.
				sender := common.HexToAddress("0x1234")
				builder.balanceChange(sender, uint256.NewInt(1000), uint256.NewInt(900))
				builder.nonceChange(sender, 0, 1)

				for j := 0; j < numCalls; j++ {
					builder.enterScope()
					// Precompile call: no state hooks fire
					builder.exitScope(false)
				}
				builder.finalise()
			}
		})
	}
}

// BALReader test ideas
// * BAL which doesn't have any pre-tx system contracts should return an empty state diff at idx 0
