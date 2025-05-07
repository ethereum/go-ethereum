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
	"github.com/ethereum/go-ethereum/triedb"
)

type precompileContract struct{}

func (p *precompileContract) RequiredGas(input []byte) uint64 { return 0 }

func (p *precompileContract) Run(input []byte) ([]byte, error) { return nil, nil }

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
