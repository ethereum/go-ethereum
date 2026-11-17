// Copyright 2026 The go-ethereum Authors
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

package logger

import (
	"encoding/json"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type faultTestStep struct {
	Type    string   `json:"type"`
	Pc      uint64   `json:"pc"`
	Op      string   `json:"op"`
	GasCost *uint64  `json:"gasCost"`
	Depth   int      `json:"depth"`
	Error   string   `json:"error"`
	Stack   []string `json:"stack"`
	Memory  []string `json:"memory"`
}

func TestStructLoggerFaultEvents(t *testing.T) {
	for _, tc := range []struct {
		name       string
		code       string
		gas        uint64
		readOnly   bool
		steps      int
		op         vm.OpCode
		wantErr    string
		wantCost   uint64
		fault      bool
		stepError  bool
		wantStack  []string
		wantMemory []string
	}{
		{
			name:      "stack underflow",
			code:      "01",
			gas:       100,
			steps:     1,
			op:        vm.ADD,
			wantErr:   "stack underflow",
			stepError: true,
		},
		{
			name:      "constant gas out of gas",
			code:      "5b5b5b",
			gas:       2,
			steps:     3,
			op:        vm.JUMPDEST,
			wantErr:   vm.ErrOutOfGas.Error(),
			stepError: true,
		},
		{
			name:      "cold call out of gas",
			code:      "6000600060006000600061dead5af1",
			gas:       150,
			steps:     8,
			op:        vm.CALL,
			wantErr:   vm.ErrOutOfGas.Error(),
			stepError: true,
			wantStack: []string{"0x0", "0x0", "0x0", "0x0", "0x0", "0xdead", "0x82"},
		},
		{
			name:      "invalid jump",
			code:      "600056",
			gas:       100,
			steps:     2,
			op:        vm.JUMP,
			wantErr:   vm.ErrInvalidJump.Error(),
			wantCost:  vm.GasMidStep,
			fault:     true,
			wantStack: []string{"0x0"},
		},
		{
			name:      "static sstore",
			code:      "6001600055",
			gas:       100000,
			readOnly:  true,
			steps:     3,
			op:        vm.SSTORE,
			wantErr:   vm.ErrWriteProtection.Error(),
			stepError: true,
			wantStack: []string{"0x1", "0x0"},
		},
		{
			name:      "static log",
			code:      "60006000a0",
			gas:       100000,
			readOnly:  true,
			steps:     3,
			op:        vm.LOG0,
			wantErr:   vm.ErrWriteProtection.Error(),
			wantCost:  params.LogGas,
			fault:     true,
			wantStack: []string{"0x0", "0x0"},
		},
		{
			name:    "invalid opcode",
			code:    "fe",
			gas:     100,
			steps:   1,
			op:      vm.INVALID,
			wantErr: "invalid opcode",
			fault:   true,
		},
		{
			name:       "static log memory expansion",
			code:       "60aa60005260406000a0",
			gas:        100000,
			readOnly:   true,
			steps:      6,
			op:         vm.LOG0,
			wantErr:    vm.ErrWriteProtection.Error(),
			wantCost:   params.LogGas + 64*params.LogDataGas + params.MemoryGas,
			fault:      true,
			wantStack:  []string{"0x40", "0x0"},
			wantMemory: []string{"0x00000000000000000000000000000000000000000000000000000000000000aa"},
		},
		{
			name:      "revert",
			code:      "60206000fd",
			gas:       100,
			steps:     3,
			op:        vm.REVERT,
			wantErr:   vm.ErrExecutionReverted.Error(),
			wantCost:  3,
			wantStack: []string{"0x20", "0x0"},
		},
		{
			name:      "revert out of gas",
			code:      "60206000fd",
			gas:       7,
			steps:     3,
			op:        vm.REVERT,
			wantErr:   vm.ErrOutOfGas.Error(),
			stepError: true,
			wantStack: []string{"0x20", "0x0"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewStructLogger(&Config{EnableMemory: true})
			code := common.Hex2Bytes(tc.code)
			evm, self := faultTestEVM(t, logger.Hooks(), code)
			defer evm.Release()

			contract := vm.NewContract(common.Address{}, self, new(uint256.Int), vm.NewGasBudget(tc.gas, 0), nil)
			contract.SetCallCode(common.Hash{}, code)
			_, err := evm.Run(contract, nil, tc.readOnly)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("execution error = %v, want %q", err, tc.wantErr)
			}

			wantEntries := tc.steps
			if tc.fault {
				wantEntries++
			}
			if len(logger.logs) != wantEntries {
				t.Fatalf("got %d entries, want %d: %s", len(logger.logs), wantEntries, logger.logs)
			}
			step := faultTestDecode(t, logger.logs[tc.steps-1])
			if step.Type != "" || step.Op != tc.op.String() || step.Pc != uint64(len(code)-1) || step.Depth != 1 || step.GasCost == nil || *step.GasCost != tc.wantCost {
				t.Fatalf("unexpected opcode entry: %s", logger.logs[tc.steps-1])
			}

			wantError := ""
			if tc.stepError {
				wantError = err.Error()
			}
			if step.Error != wantError || !slices.Equal(step.Stack, tc.wantStack) {
				t.Errorf("opcode error/stack = %q/%v, want %q/%v", step.Error, step.Stack, wantError, tc.wantStack)
			}
			if !slices.Equal(step.Memory, tc.wantMemory) {
				t.Errorf("opcode memory = %v, want %v", step.Memory, tc.wantMemory)
			}
			if tc.fault {
				faultTestCompact(t, logger.logs[tc.steps], step.Pc, tc.op.String(), step.Depth, err.Error())
			}
		})
	}
}

func TestStructLoggerNestedFault(t *testing.T) {
	child := common.HexToAddress("0xdead")
	for _, tc := range []struct {
		name string
		op   vm.OpCode
		code []byte
	}{
		{
			name: "call",
			op:   vm.CALL,
			code: append(append(common.Hex2Bytes("6000600060006000600073"), child.Bytes()...), byte(vm.GAS), byte(vm.CALL), byte(vm.POP), byte(vm.STOP)),
		},
		{
			name: "create",
			op:   vm.CREATE,
			code: common.Hex2Bytes("626000565f526003601d5ff05000"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewStructLogger(nil)
			evm, self := faultTestEVM(t, logger.Hooks(), tc.code)
			defer evm.Release()

			evm.StateDB.SetCode(child, common.Hex2Bytes("600056"), tracing.CodeChangeUnspecified)
			if _, _, err := evm.Call(common.Address{}, self, nil, vm.NewGasBudget(1000000, 0), new(uint256.Int)); err != nil {
				t.Fatal(err)
			}
			var parents, faults int
			for i, entry := range logger.logs {
				step := faultTestDecode(t, entry)
				if step.Depth == 1 && step.Op == tc.op.String() {
					parents++
					if step.Type != "" || step.Error != "" || step.GasCost == nil || *step.GasCost == 0 {
						t.Fatalf("parent opcode changed: %s", entry)
					}
				}
				if step.Type == "fault" {
					faults++
					if i == 0 {
						t.Fatal("fault has no preceding opcode")
					}
					opcode := faultTestDecode(t, logger.logs[i-1])
					if opcode.Op != vm.JUMP.String() || opcode.Error != "" || opcode.GasCost == nil || *opcode.GasCost != vm.GasMidStep || !slices.Equal(opcode.Stack, []string{"0x0"}) {
						t.Fatalf("child opcode changed: %s", logger.logs[i-1])
					}
					faultTestCompact(t, entry, opcode.Pc, opcode.Op, 2, vm.ErrInvalidJump.Error())
				}
			}
			last := faultTestDecode(t, logger.logs[len(logger.logs)-1])
			if parents != 1 || faults != 1 || last.Op != vm.STOP.String() || last.Depth != 1 {
				t.Fatalf("unexpected parent/fault counts or final step: %d/%d, %+v", parents, faults, last)
			}
		})
	}
}

func faultTestCompact(t *testing.T, entry json.RawMessage, pc uint64, op any, depth int, message string) {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(entry, &fields); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"type": "fault", "pc": float64(pc), "op": op, "depth": float64(depth), "error": message}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("fault fields = %v, want %v", fields, want)
	}
}

func faultTestDecode(t *testing.T, entry json.RawMessage) faultTestStep {
	t.Helper()
	var step faultTestStep
	if err := json.Unmarshal(entry, &step); err != nil {
		t.Fatal(err)
	}
	return step
}

func faultTestEVM(t *testing.T, hooks *tracing.Hooks, code []byte) (*vm.EVM, common.Address) {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	self := common.HexToAddress("0xc0de")
	statedb.SetCode(self, code, tracing.CodeChangeUnspecified)
	evm := vm.NewEVM(
		vm.BlockContext{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			BlockNumber: new(big.Int),
			Random:      &common.Hash{},
		}, statedb, params.MergedTestChainConfig, vm.Config{Tracer: hooks},
	)
	hooks.OnTxStart(evm.GetVMContext(), nil, common.Address{})
	return evm, self
}
