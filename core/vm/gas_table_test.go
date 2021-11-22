// Copyright 2017 The go-ethereum Authors
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

package vm

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

func TestMemoryGasCost(t *testing.T) {
	tests := []struct {
		size     uint64
		cost     uint64
		overflow bool
	}{
		{0x1fffffffe0, 36028809887088637, false},
		{0x1fffffffe1, 0, true},
	}
	for i, tt := range tests {
		v, err := memoryGasCost(&Memory{}, tt.size)
		if (err == ErrGasUintOverflow) != tt.overflow {
			t.Errorf("test %d: overflow mismatch: have %v, want %v", i, err == ErrGasUintOverflow, tt.overflow)
		}
		if v != tt.cost {
			t.Errorf("test %d: gas cost mismatch: have %v, want %v", i, v, tt.cost)
		}
	}
}

var eip2200Tests = []struct {
	original byte
	gaspool  uint64
	input    string
	used     uint64
	refund   uint64
	failure  error
}{
	{0, math.MaxUint64, "0x60006000556000600055", 1612, 0, nil},                // 0 -> 0 -> 0
	{0, math.MaxUint64, "0x60006000556001600055", 20812, 0, nil},               // 0 -> 0 -> 1
	{0, math.MaxUint64, "0x60016000556000600055", 20812, 19200, nil},           // 0 -> 1 -> 0
	{0, math.MaxUint64, "0x60016000556002600055", 20812, 0, nil},               // 0 -> 1 -> 2
	{0, math.MaxUint64, "0x60016000556001600055", 20812, 0, nil},               // 0 -> 1 -> 1
	{1, math.MaxUint64, "0x60006000556000600055", 5812, 15000, nil},            // 1 -> 0 -> 0
	{1, math.MaxUint64, "0x60006000556001600055", 5812, 4200, nil},             // 1 -> 0 -> 1
	{1, math.MaxUint64, "0x60006000556002600055", 5812, 0, nil},                // 1 -> 0 -> 2
	{1, math.MaxUint64, "0x60026000556000600055", 5812, 15000, nil},            // 1 -> 2 -> 0
	{1, math.MaxUint64, "0x60026000556003600055", 5812, 0, nil},                // 1 -> 2 -> 3
	{1, math.MaxUint64, "0x60026000556001600055", 5812, 4200, nil},             // 1 -> 2 -> 1
	{1, math.MaxUint64, "0x60026000556002600055", 5812, 0, nil},                // 1 -> 2 -> 2
	{1, math.MaxUint64, "0x60016000556000600055", 5812, 15000, nil},            // 1 -> 1 -> 0
	{1, math.MaxUint64, "0x60016000556002600055", 5812, 0, nil},                // 1 -> 1 -> 2
	{1, math.MaxUint64, "0x60016000556001600055", 1612, 0, nil},                // 1 -> 1 -> 1
	{0, math.MaxUint64, "0x600160005560006000556001600055", 40818, 19200, nil}, // 0 -> 1 -> 0 -> 1
	{1, math.MaxUint64, "0x600060005560016000556000600055", 10818, 19200, nil}, // 1 -> 0 -> 1 -> 0
	{1, 2306, "0x6001600055", 2306, 0, ErrOutOfGas},                            // 1 -> 1 (2300 sentry + 2xPUSH)
	{1, 2307, "0x6001600055", 806, 0, nil},                                     // 1 -> 1 (2301 sentry + 2xPUSH)
}

func TestEIP2200(t *testing.T) {
	for i, tt := range eip2200Tests {
		address := common.BytesToAddress([]byte("contract"))

		statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		statedb.CreateAccount(address)
		statedb.SetCode(address, hexutil.MustDecode(tt.input))
		statedb.SetState(address, common.Hash{}, common.BytesToHash([]byte{tt.original}))
		statedb.Finalise(true) // Push the state into the "original" slot

		vmctx := BlockContext{
			CanTransfer: func(StateDB, common.Address, *big.Int) bool { return true },
			Transfer:    func(StateDB, common.Address, common.Address, *big.Int) {},
		}
		vmenv := NewEVM(vmctx, TxContext{}, statedb, params.AllEthashProtocolChanges, Config{ExtraEips: []int{2200}})

		_, gas, err := vmenv.Call(AccountRef(common.Address{}), address, nil, tt.gaspool, new(big.Int))
		if err != tt.failure {
			t.Errorf("test %d: failure mismatch: have %v, want %v", i, err, tt.failure)
		}
		if used := tt.gaspool - gas; used != tt.used {
			t.Errorf("test %d: gas used mismatch: have %v, want %v", i, used, tt.used)
		}
		if refund := vmenv.StateDB.GetRefund(); refund != tt.refund {
			t.Errorf("test %d: gas refund mismatch: have %v, want %v", i, refund, tt.refund)
		}
	}
}

var createGasTests = []struct {
	code    string
	eip3860 bool
	gasUsed uint64
}{
	// create(0, 0, 0xc000)
	{"0x61C00060006000f0", false, 41225},
	// create(0, 0, 0xc000)
	{"0x61C00060006000f0", true, 44297},
	// create2(0, 0, 0xc000, 0)
	{"0x600061C00060006000f5", false, 50444},
	// create2(0, 0, 0xc000, 0)
	{"0x600061C00060006000f5", true, 53516},
}

func TestCreateGas(t *testing.T) {
	for i, tt := range createGasTests {
		address := common.BytesToAddress([]byte("contract"))

		statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		statedb.CreateAccount(address)
		statedb.SetCode(address, hexutil.MustDecode(tt.code))
		statedb.Finalise(true)

		vmctx := BlockContext{
			CanTransfer: func(StateDB, common.Address, *big.Int) bool { return true },
			Transfer:    func(StateDB, common.Address, common.Address, *big.Int) {},
			BlockNumber: big.NewInt(0),
		}
		config := Config{}
		if tt.eip3860 {
			config.ExtraEips = []int{3860}
		}

		vmenv := NewEVM(vmctx, TxContext{}, statedb, params.AllEthashProtocolChanges, config)

		var startGas uint64 = math.MaxUint64
		_, gas, err := vmenv.Call(AccountRef(common.Address{}), address, nil, startGas, new(big.Int))
		if err != nil {
			t.Errorf("test %d execution failed: %v", i, err)
		}
		if gasUsed := startGas - gas; gasUsed != tt.gasUsed {
			t.Errorf("test %d: gas used mismatch: have %v, want %v", i, gasUsed, tt.gasUsed)
		}
	}
}
