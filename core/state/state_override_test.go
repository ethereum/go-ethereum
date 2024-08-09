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

package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func initTestState() (common.Hash, *CachingDB) {
	db := NewDatabaseForTesting(rawdb.NewMemoryDatabase())
	state, _ := New(types.EmptyRootHash, db)
	state.SetNonce(testAllocAddr, 1)
	state.SetBalance(testAllocAddr, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
	state.SetCode(testAllocAddr, []byte{0x1})
	state.SetState(testAllocAddr, testAllocSlotKey, common.Hash{0x1})
	root, err := state.Commit(0, false)
	if err != nil {
		panic("failed to commit state")
	}
	return root, db
}

var (
	testAllocAddr     = common.HexToAddress("0xdeadbeef")
	testAllocAddr2    = common.HexToAddress("0xbabecafe")
	testAllocSlotKey  = common.HexToHash("0xdeadbeef")
	testAllocSlotKey2 = common.HexToHash("0xbabecafe")
)

func overrideNonce(val uint64) *uint64 {
	return &val
}

func overrideCode(code []byte) *[]byte {
	return &code
}

func TestStateOverride(t *testing.T) {
	var cases = []struct {
		overrides      map[common.Address]OverrideAccount
		expectAccounts map[common.Address]types.StateAccount
		expectStorages map[common.Address]map[common.Hash]common.Hash
		expectCodes    map[common.Address][]byte
	}{
		// override is nil, it should be allowed
		{
			nil,
			map[common.Address]types.StateAccount{
				testAllocAddr: {
					Nonce:    1,
					Balance:  uint256.NewInt(100),
					CodeHash: crypto.Keccak256([]byte{0x1}),
				},
			},
			map[common.Address]map[common.Hash]common.Hash{
				testAllocAddr: {testAllocSlotKey: {0x1}},
			},
			map[common.Address][]byte{
				testAllocAddr: {0x1},
			},
		},
		// empty override
		{
			map[common.Address]OverrideAccount{},
			map[common.Address]types.StateAccount{
				testAllocAddr: {
					Nonce:    1,
					Balance:  uint256.NewInt(100),
					CodeHash: crypto.Keccak256([]byte{0x1}),
				},
			},
			map[common.Address]map[common.Hash]common.Hash{
				testAllocAddr: {testAllocSlotKey: {0x1}},
			},
			map[common.Address][]byte{
				testAllocAddr: {0x1},
			},
		},
		// empty override, access the non-existent states
		{
			map[common.Address]OverrideAccount{},
			map[common.Address]types.StateAccount{
				testAllocAddr2: {
					Nonce:    0,
					Balance:  common.U2560,
					CodeHash: common.Hash{}.Bytes(),
				},
			},
			map[common.Address]map[common.Hash]common.Hash{
				testAllocAddr2: {testAllocSlotKey: {}},
			},
			map[common.Address][]byte{
				testAllocAddr2: nil,
			},
		},
		// override account metadata
		{
			map[common.Address]OverrideAccount{
				testAllocAddr: {
					Nonce:   overrideNonce(50),
					Code:    overrideCode([]byte{0x2}),
					Balance: uint256.NewInt(200),
				},
				testAllocAddr2: {
					Nonce:   overrideNonce(50),
					Code:    overrideCode([]byte{0x2}),
					Balance: uint256.NewInt(200),
				},
			},
			map[common.Address]types.StateAccount{
				testAllocAddr: {
					Nonce:    50,
					Balance:  uint256.NewInt(200),
					CodeHash: crypto.Keccak256([]byte{0x2}),
				},
				testAllocAddr2: {
					Nonce:    50,
					Balance:  uint256.NewInt(200),
					CodeHash: crypto.Keccak256([]byte{0x2}),
				},
			},
			map[common.Address]map[common.Hash]common.Hash{
				testAllocAddr: {testAllocSlotKey: {0x1}},
			},
			map[common.Address][]byte{
				testAllocAddr:  {0x2},
				testAllocAddr2: {0x2},
			},
		},
		// override storage by diff
		{
			map[common.Address]OverrideAccount{
				testAllocAddr: {
					StateDiff: map[common.Hash]common.Hash{
						testAllocSlotKey:  {0x2},
						testAllocSlotKey2: {0x2},
					},
				},
			},
			map[common.Address]types.StateAccount{
				testAllocAddr: {
					Nonce:    1,
					Balance:  uint256.NewInt(100),
					CodeHash: crypto.Keccak256([]byte{0x1}),
				},
			},
			map[common.Address]map[common.Hash]common.Hash{
				testAllocAddr: {
					testAllocSlotKey:  {0x2},
					testAllocSlotKey2: {0x2},
				},
			},
			map[common.Address][]byte{
				testAllocAddr: {0x1},
			},
		},
		// override storage by replacing entire storage
		{
			map[common.Address]OverrideAccount{
				testAllocAddr: {
					State: map[common.Hash]common.Hash{
						testAllocSlotKey2: {0x2},
					},
				},
			},
			map[common.Address]types.StateAccount{
				testAllocAddr: {
					Nonce:    1,
					Balance:  uint256.NewInt(100),
					CodeHash: crypto.Keccak256([]byte{0x1}),
				},
			},
			map[common.Address]map[common.Hash]common.Hash{
				testAllocAddr: {
					testAllocSlotKey:  {},
					testAllocSlotKey2: {0x2},
				},
			},
			map[common.Address][]byte{
				testAllocAddr: {0x1},
			},
		},
	}
	for _, c := range cases {
		root, db := initTestState()
		stateDb, err := New(root, NewOverrideDatabase(db, c.overrides))
		if err != nil {
			t.Fatalf("Failed to initialize state, %v", err)
		}
		for addr, expect := range c.expectAccounts {
			if got := stateDb.GetBalance(addr); got.Cmp(expect.Balance) != 0 {
				t.Fatalf("Balance is not matched, got %v, want: %v", got, expect.Balance)
			}
			if got := stateDb.GetNonce(addr); got != expect.Nonce {
				t.Fatalf("Nonce is not matched, got %v, want: %v", got, expect.Nonce)
			}
			if got := stateDb.GetCodeHash(addr); !bytes.Equal(got.Bytes(), expect.CodeHash) {
				t.Fatalf("CodeHash is not matched, got %v, want: %v", got.Bytes(), expect.CodeHash)
			}
		}
		for addr, slots := range c.expectStorages {
			for key, val := range slots {
				got := stateDb.GetState(addr, key)
				if got != val {
					t.Fatalf("Storage slot is not matched, got %v, want: %v", got, val)
				}
			}
		}
		for addr, code := range c.expectCodes {
			if got := stateDb.GetCode(addr); !bytes.Equal(got, code) {
				t.Fatalf("Code is not matched, got %v, want: %v", got, code)
			}
		}
	}
}

func TestRecursiveOverride(t *testing.T) {
	root, db := initTestState()

	overrideA := map[common.Address]OverrideAccount{
		testAllocAddr: {Nonce: overrideNonce(2)},
	}
	overrideB := map[common.Address]OverrideAccount{
		testAllocAddr: {Nonce: overrideNonce(3)},
	}
	overrideC := map[common.Address]OverrideAccount{
		testAllocAddr: {
			Nonce: overrideNonce(0),
			State: map[common.Hash]common.Hash{
				testAllocSlotKey: {},
			},
		},
	}
	odbA := NewOverrideDatabase(db, overrideA)
	stateDb, err := New(root, odbA)
	if err != nil {
		t.Fatalf("Failed to initialize state, %v", err)
	}
	nonce := stateDb.GetNonce(testAllocAddr)
	if nonce != 2 {
		t.Fatalf("Unexpected nonce, want: %d, got: %d", 2, nonce)
	}
	// recursively override
	stateDb2, err := OverrideState(stateDb, overrideB)
	if err != nil {
		t.Fatalf("Failed to initialize state, %v", err)
	}
	nonce = stateDb2.GetNonce(testAllocAddr)
	if nonce != 3 {
		t.Fatalf("Unexpected nonce, want: %d, got: %d", 3, nonce)
	}
	// recursively override again
	stateDb3, err := OverrideState(stateDb2, overrideC)
	if err != nil {
		t.Fatalf("Failed to initialize state, %v", err)
	}
	nonce = stateDb3.GetNonce(testAllocAddr)
	if nonce != 0 {
		t.Fatalf("Unexpected nonce, want: %d, got: %d", 0, nonce)
	}
	val := stateDb3.GetState(testAllocAddr, testAllocSlotKey)
	if val != (common.Hash{}) {
		t.Fatalf("Unexpected storage slot, want: %v, got: %v", common.Hash{}, val)
	}
}
