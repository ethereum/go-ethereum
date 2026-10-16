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

package override

import (
	"maps"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

type precompileContract struct{}

func (p *precompileContract) RequiredGas(input []byte) uint64 { return 0 }

func (p *precompileContract) Run(input []byte) ([]byte, error) { return nil, nil }

func (p *precompileContract) Name() string {
	panic("implement me")
}

func TestStateOverrideMovePrecompile(t *testing.T) {
	db := state.NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
	statedb, err := state.New(types.EmptyRootHash, db)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}
	precompiles := map[common.Address]vm.PrecompiledContract{
		common.BytesToAddress([]byte{0x1}): &precompileContract{},
		common.BytesToAddress([]byte{0x2}): &precompileContract{},
	}
	bytes2Addr := func(b []byte) *common.Address {
		a := common.BytesToAddress(b)
		return &a
	}
	var testSuite = []struct {
		overrides           StateOverride
		expectedPrecompiles map[common.Address]struct{}
		fail                bool
	}{
		{
			overrides: StateOverride{
				common.BytesToAddress([]byte{0x1}): {
					Code:             hex2Bytes("0xff"),
					MovePrecompileTo: bytes2Addr([]byte{0x2}),
				},
				common.BytesToAddress([]byte{0x2}): {
					Code: hex2Bytes("0x00"),
				},
			},
			// 0x2 has already been touched by the moveTo.
			fail: true,
		}, {
			overrides: StateOverride{
				common.BytesToAddress([]byte{0x1}): {
					Code:             hex2Bytes("0xff"),
					MovePrecompileTo: bytes2Addr([]byte{0xff}),
				},
				common.BytesToAddress([]byte{0x3}): {
					Code:             hex2Bytes("0x00"),
					MovePrecompileTo: bytes2Addr([]byte{0xfe}),
				},
			},
			// 0x3 is not a precompile.
			fail: true,
		}, {
			overrides: StateOverride{
				common.BytesToAddress([]byte{0x1}): {
					Code:             hex2Bytes("0xff"),
					MovePrecompileTo: bytes2Addr([]byte{0xff}),
				},
				common.BytesToAddress([]byte{0x2}): {
					Code:             hex2Bytes("0x00"),
					MovePrecompileTo: bytes2Addr([]byte{0xfe}),
				},
			},
			expectedPrecompiles: map[common.Address]struct{}{common.BytesToAddress([]byte{0xfe}): {}, common.BytesToAddress([]byte{0xff}): {}},
		},
	}

	for i, tt := range testSuite {
		cpy := maps.Clone(precompiles)
		// Apply overrides
		err := tt.overrides.Apply(statedb, cpy, true)
		if tt.fail {
			if err == nil {
				t.Errorf("test %d: want error, have nothing", i)
			}
			continue
		}
		if err != nil {
			t.Errorf("test %d: want no error, have %v", i, err)
			continue
		}
		// Precompile keys
		if len(cpy) != len(tt.expectedPrecompiles) {
			t.Errorf("test %d: precompile mismatch, want %d, have %d", i, len(tt.expectedPrecompiles), len(cpy))
		}
		for k := range tt.expectedPrecompiles {
			if _, ok := cpy[k]; !ok {
				t.Errorf("test %d: precompile not found: %s", i, k.String())
			}
		}
	}
}

func hex2Bytes(str string) *hexutil.Bytes {
	rpcBytes := hexutil.Bytes(common.FromHex(str))
	return &rpcBytes
}

// TestStateOverrideEmptyAccount checks that an override does not create an empty
// account where EIP-158 is active, and creates an account that it leaves non-empty.
func TestStateOverrideEmptyAccount(t *testing.T) {
	var (
		addr  = common.Address{0xaa}
		empty = hexutil.Bytes{}
		zero  = common.Hash{}
		one   = common.HexToHash("0x01")
	)
	tests := []struct {
		name     string
		override OverrideAccount
		write    bool
		create   bool
	}{
		{name: "balance", override: OverrideAccount{Balance: (*hexutil.Big)(big.NewInt(1))}, write: true, create: true},
		{name: "empty code", override: OverrideAccount{Code: &empty}, write: true},
		{name: "zero slots", override: OverrideAccount{StateDiff: map[common.Hash]common.Hash{one: zero}}, write: true},
		{name: "empty state", override: OverrideAccount{State: map[common.Hash]common.Hash{}}, write: true},
		{name: "empty state diff", override: OverrideAccount{StateDiff: map[common.Hash]common.Hash{}}},
		{name: "storage", override: OverrideAccount{Code: &empty, StateDiff: map[common.Hash]common.Hash{zero: one}}, write: true, create: true},
		{name: "whole storage", override: OverrideAccount{State: map[common.Hash]common.Hash{zero: one}}, write: true, create: true},
	}
	for _, tt := range tests {
		for _, deleteEmptyObjects := range []bool{false, true} {
			db := state.NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
			statedb, err := state.New(types.EmptyRootHash, db)
			if err != nil {
				t.Fatalf("failed to create statedb: %v", err)
			}
			overrides := StateOverride{addr: tt.override}
			if err := overrides.Apply(statedb, nil, deleteEmptyObjects); err != nil {
				t.Fatalf("%s: failed to apply overrides: %v", tt.name, err)
			}
			want := tt.create || (tt.write && !deleteEmptyObjects)
			if exist := statedb.Exist(addr); exist != want {
				t.Errorf("%s, deleteEmptyObjects %v: account exists %v, want %v", tt.name, deleteEmptyObjects, exist, want)
			}
		}
	}
}

// TestStateOverrideEmptiedAccount checks that an existing account that the overrides
// leave empty is removed with its storage, unless the override sets a non-zero slot,
// and that a move-only override leaves its account unchanged.
func TestStateOverrideEmptiedAccount(t *testing.T) {
	var (
		contract = common.Address{0xaa}
		eoa      = common.Address{0xbb}
		kept     = common.Address{0xcc}
		source   = common.BytesToAddress([]byte{0x01})
		empty    = hexutil.Bytes{}
		zero     = hexutil.Uint64(0)
		slot     = common.HexToHash("0x01")
	)
	for _, deleteEmptyObjects := range []bool{false, true} {
		db := state.NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
		statedb, err := state.New(types.EmptyRootHash, db)
		if err != nil {
			t.Fatalf("failed to create statedb: %v", err)
		}
		for _, addr := range []common.Address{contract, kept} {
			statedb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
			statedb.SetCode(addr, []byte{0x00}, tracing.CodeChangeUnspecified)
			statedb.SetState(addr, common.Hash{}, slot)
		}
		statedb.SetBalance(eoa, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		statedb.SetState(source, common.Hash{}, slot)
		statedb.Finalise(params.Rules{})

		overrides := StateOverride{
			contract: OverrideAccount{Nonce: &zero, Code: &empty},
			eoa:      OverrideAccount{Balance: (*hexutil.Big)(new(big.Int))},
			kept:     OverrideAccount{Nonce: &zero, Code: &empty, StateDiff: map[common.Hash]common.Hash{slot: slot}},
			source:   OverrideAccount{MovePrecompileTo: &common.Address{0xdd}},
		}
		precompiles := vm.PrecompiledContracts{source: &precompileContract{}}
		if err := overrides.Apply(statedb, precompiles, deleteEmptyObjects); err != nil {
			t.Fatalf("failed to apply overrides: %v", err)
		}
		removed := deleteEmptyObjects
		if exist := statedb.Exist(contract); exist == removed {
			t.Errorf("deleteEmptyObjects %v: emptied contract exists %v", deleteEmptyObjects, exist)
		}
		if exist := statedb.Exist(eoa); exist == removed {
			t.Errorf("deleteEmptyObjects %v: emptied account exists %v", deleteEmptyObjects, exist)
		}
		wantSlot := slot
		if removed {
			wantSlot = common.Hash{}
		}
		if got := statedb.GetState(contract, common.Hash{}); got != wantSlot {
			t.Errorf("deleteEmptyObjects %v: emptied contract slot 0 = %x, want %x", deleteEmptyObjects, got, wantSlot)
		}
		if !statedb.Exist(kept) || statedb.GetState(kept, common.Hash{}) != slot || statedb.GetState(kept, slot) != slot {
			t.Errorf("deleteEmptyObjects %v: account with a storage override lost its storage", deleteEmptyObjects)
		}
		if !statedb.Exist(source) || statedb.GetState(source, common.Hash{}) != slot {
			t.Errorf("deleteEmptyObjects %v: move-only override changed the source account", deleteEmptyObjects)
		}
	}
}
