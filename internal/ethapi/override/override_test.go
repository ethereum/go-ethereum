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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
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
		err := tt.overrides.Apply(statedb, cpy)
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

func TestStateOverrideTransientStorage(t *testing.T) {
	db := state.NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
	statedb, err := state.New(types.EmptyRootHash, db)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	addr := common.BytesToAddress([]byte{0x1})
	key1 := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	key2 := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")
	value1 := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	value2 := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")

	// Verify initial state is empty
	if got := statedb.GetTransientState(addr, key1); got != (common.Hash{}) {
		t.Fatalf("expected initial transient state to be empty, got %s", got.Hex())
	}
	if got := statedb.GetTransientState(addr, key2); got != (common.Hash{}) {
		t.Fatalf("expected initial transient state to be empty, got %s", got.Hex())
	}

	// Apply override with transient storage
	override := StateOverride{
		addr: OverrideAccount{
			TransientStorage: map[common.Hash]common.Hash{
				key1: value1,
				key2: value2,
			},
		},
	}

	if err := override.Apply(statedb, nil); err != nil {
		t.Fatalf("failed to apply override: %v", err)
	}

	statedb.Prepare(params.Rules{}, common.Address{}, common.Address{}, nil, nil, nil)

	// Verify transient storage was set
	if got := statedb.GetTransientState(addr, key1); got != value1 {
		t.Errorf("expected transient state for key1 to be %s, got %s", value1.Hex(), got.Hex())
	}
	if got := statedb.GetTransientState(addr, key2); got != value2 {
		t.Errorf("expected transient state for key2 to be %s, got %s", value2.Hex(), got.Hex())
	}

	// Verify other addresses/keys remain empty
	otherAddr := common.BytesToAddress([]byte{0x2})
	if got := statedb.GetTransientState(otherAddr, key1); got != (common.Hash{}) {
		t.Errorf("expected transient state for different address to be empty, got %s", got.Hex())
	}

	otherKey := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003")
	if got := statedb.GetTransientState(addr, otherKey); got != (common.Hash{}) {
		t.Errorf("expected transient state for different key to be empty, got %s", got.Hex())
	}
}
