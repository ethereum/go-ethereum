// Copyright 2021 The go-ethereum Authors
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
	"errors"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var loopInterruptTests = []string{
	// infinite loop using JUMP: push(2) jumpdest dup1 jump
	"60025b8056",
	// infinite loop using JUMPI: push(1) push(4) jumpdest dup2 dup2 jumpi
	"600160045b818157",
}

func TestLoopInterrupt(t *testing.T) {
	address := common.BytesToAddress([]byte("contract"))
	vmctx := BlockContext{
		Transfer: func(StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
	}

	for i, tt := range loopInterruptTests {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		statedb.CreateAccount(address)
		statedb.SetCode(address, common.Hex2Bytes(tt), tracing.CodeChangeUnspecified)
		statedb.Finalise(params.Rules{IsEIP158: true})

		evm := NewEVM(vmctx, statedb, params.AllEthashProtocolChanges, Config{})

		errChannel := make(chan error)
		timeout := make(chan bool)

		go func(evm *EVM) {
			_, _, err := evm.Call(common.Address{}, address, nil, NewGasBudget(math.MaxUint64, 0), new(uint256.Int))
			errChannel <- err
		}(evm)

		go func() {
			<-time.After(time.Second)
			timeout <- true
		}()

		evm.Cancel()

		select {
		case <-timeout:
			t.Errorf("test %d timed out", i)
		case err := <-errChannel:
			if err != nil {
				t.Errorf("test %d failure: %v", i, err)
			}
		}
	}
}

func TestTracedStateGasCost(t *testing.T) {
	storeCost := tracing.OpCost{
		Execution: params.ColdStorageAccessAmsterdam + params.StorageWriteAmsterdam,
		State:     stateGasNewSlot,
	}
	for _, tc := range []struct {
		name      string
		code      []byte
		gas       GasBudget
		wantCosts []tracing.OpCost
		wantErr   error
	}{
		{
			name:      "state charge out of gas",
			code:      sstore(0, 1),
			gas:       NewGasBudget(20_000, 0),
			wantCosts: []tracing.OpCost{{}},
			wantErr:   ErrOutOfGas,
		},
		{
			name:      "partial state reservoir out of gas",
			code:      sstore(0, 1),
			gas:       NewGasBudget(20_000, 100),
			wantCosts: []tracing.OpCost{{}},
			wantErr:   ErrOutOfGas,
		},
		{
			name: "state charge and refill",
			code: append(sstore(0, 1), sstore(0, 0)...),
			gas:  NewGasBudget(1_000_000, 0),
			wantCosts: []tracing.OpCost{
				storeCost,
				{Execution: params.WarmStorageAccessAmsterdam, State: -stateGasNewSlot},
			},
		},
		{
			name:      "sentry failure before refill",
			code:      append(sstore(0, 1), sstore(0, 0)...),
			gas:       NewGasBudget(4*GasFastestStep+storeCost.Execution+params.SstoreSentryGasEIP2200, uint64(stateGasNewSlot)),
			wantCosts: []tracing.OpCost{storeCost, {}},
			wantErr:   ErrOutOfGas,
		},
		{
			name:      "stack error after state charge",
			code:      append(sstore(0, 1), byte(ADD)),
			gas:       hugeBudget(),
			wantCosts: []tracing.OpCost{storeCost, {}},
			wantErr:   &ErrStackUnderflow{stackLen: 0, required: 2},
		},
		{
			name:      "constant gas error after state charge",
			code:      append(sstore(0, 1), byte(PUSH1), 0),
			gas:       NewGasBudget(2*GasFastestStep+storeCost.Execution, uint64(stateGasNewSlot)),
			wantCosts: []tracing.OpCost{storeCost, {}},
			wantErr:   ErrOutOfGas,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := common.BytesToAddress([]byte("self"))
			statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
			statedb.CreateAccount(self)
			statedb.SetCode(self, tc.code, tracing.CodeChangeUnspecified)
			statedb.Finalise(params.Rules{IsEIP158: true})

			var (
				costs   []tracing.OpCost
				hookErr error
			)
			evm := amsterdam8037EVM(statedb)
			evm.Config.Tracer = &tracing.Hooks{
				OnOpcodeV2: func(pc uint64, op byte, gas tracing.Gas, cost tracing.OpCost, scope tracing.OpContext, rData []byte, depth int, err error) {
					if OpCode(op) == SSTORE || err != nil {
						costs = append(costs, cost)
						hookErr = err
					}
				},
				OnFaultV2: func(pc uint64, op byte, gas tracing.Gas, cost tracing.OpCost, scope tracing.OpContext, depth int, err error) {
					t.Errorf("unexpected fault event for %s: %v", OpCode(op), err)
				},
			}
			_, _, err := evm.Call(common.Address{}, self, nil, tc.gas, new(uint256.Int))
			for _, gotErr := range []error{err, hookErr} {
				if (gotErr == nil) != (tc.wantErr == nil) || gotErr != nil && vmErrorCodeFromErr(gotErr) != vmErrorCodeFromErr(tc.wantErr) {
					t.Errorf("error = %v, want %v", gotErr, tc.wantErr)
				}
			}
			if len(costs) != len(tc.wantCosts) {
				t.Fatalf("costs = %v, want %v", costs, tc.wantCosts)
			}
			for i, cost := range costs {
				if cost != tc.wantCosts[i] {
					t.Errorf("cost[%d] = %v, want %v", i, cost, tc.wantCosts[i])
				}
			}
		})
	}
}

func TestTracedFaultCost(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      []byte
		wantState int64
		wantErr   error
	}{
		{name: "invalid jump", code: []byte{byte(PUSH1), 0, byte(JUMP)}, wantErr: ErrInvalidJump},
		{name: "create state out of gas", code: []byte{byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(CREATE)}, wantState: stateGasNewAccount, wantErr: ErrOutOfGas},
		{name: "create2 state out of gas", code: []byte{byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(CREATE2)}, wantState: stateGasNewAccount, wantErr: ErrOutOfGas},
		{name: "revert", code: []byte{byte(PUSH1), 32, byte(PUSH0), byte(REVERT)}, wantErr: ErrExecutionReverted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := common.BytesToAddress([]byte("self"))
			statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
			statedb.CreateAccount(self)
			statedb.SetCode(self, tc.code, tracing.CodeChangeUnspecified)
			statedb.Finalise(params.Rules{IsEIP158: true})
			evm := amsterdam8037EVM(statedb)
			var opcodes, faults int
			evm.Config.Tracer = &tracing.Hooks{
				OnOpcodeV2: func(pc uint64, op byte, gas tracing.Gas, cost tracing.OpCost, scope tracing.OpContext, rData []byte, depth int, err error) {
					if pc != uint64(len(tc.code)-1) {
						return
					}
					opcodes++
					if err != nil || cost.Execution == 0 || cost.State != tc.wantState {
						t.Errorf("pre-execution event: cost %v, error %v", cost, err)
					}
				},
				OnFaultV2: func(pc uint64, op byte, gas tracing.Gas, cost tracing.OpCost, scope tracing.OpContext, depth int, err error) {
					faults++
					if cost != (tracing.OpCost{}) || !errors.Is(err, tc.wantErr) {
						t.Errorf("fault event: cost %v, error %v; want empty cost and %v", cost, err, tc.wantErr)
					}
				},
			}
			_, _, err := evm.Call(common.Address{}, self, nil, NewGasBudget(100_000, 0), new(uint256.Int))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if opcodes != 1 || faults != 1 {
				t.Errorf("opcode/fault events = %d/%d, want 1/1", opcodes, faults)
			}
		})
	}
}

func BenchmarkInterpreter(b *testing.B) {
	var (
		statedb, _        = state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		evm               = NewEVM(BlockContext{BlockNumber: big.NewInt(1), Time: 1, Random: &common.Hash{}}, statedb, params.MergedTestChainConfig, Config{})
		startGas   uint64 = 100_000_000
		value             = uint256.NewInt(0)
		stack             = newStackForTesting()
		mem               = NewMemory()
		contract          = NewContract(common.Address{}, common.Address{}, value, NewGasBudget(startGas, 0), nil)
	)
	stack.push(uint256.NewInt(123))
	stack.push(uint256.NewInt(123))
	gasSStoreEIP3529 = makeGasSStoreFunc(params.SstoreClearsScheduleRefundEIP3529)
	for b.Loop() {
		gasSStoreEIP3529(evm, contract, stack, mem, 1234)
	}
}
