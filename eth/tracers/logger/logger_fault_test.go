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
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
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
	"github.com/stretchr/testify/require"
)

func TestStructLoggerFaults(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     string
		gas      uint64
		readOnly bool
		steps    int
		wantLast string
	}{
		{
			name:     "revert memory out of gas",
			code:     "60206000fd",
			gas:      8,
			steps:    3,
			wantLast: `{"pc":4,"op":"REVERT","gas":2,"gasCost":0,"depth":1,"error":"out of gas","stack":["0x20","0x0"]}`,
		},
		{
			name:     "static log memory expansion",
			code:     "60aa60005260406000a0",
			gas:      100000,
			readOnly: true,
			steps:    6,
			wantLast: `{
				"pc":9,"op":"LOG0","gas":99982,"gasCost":0,"depth":1,
				"error":"write protection","stack":["0x40","0x0"],
				"memory":["0x00000000000000000000000000000000000000000000000000000000000000aa"]
			}`,
		},
		{
			name:     "create state out of gas",
			code:     "5f5f5ff0",
			gas:      100000,
			steps:    4,
			wantLast: `{"pc":3,"op":"CREATE","gas":99994,"gasCost":0,"depth":1,"error":"out of gas","stack":["0x0","0x0","0x0"]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewStructLogger(&Config{EnableMemory: true})
			code := common.Hex2Bytes(tc.code)
			evm, self := faultTestEVM(t, logger, code)
			defer evm.Release()

			contract := vm.NewContract(common.Address{}, self, new(uint256.Int), vm.NewGasBudget(tc.gas, 0), nil)
			contract.SetCallCode(common.Hash{}, code)
			_, err := evm.Run(contract, nil, tc.readOnly)
			require.Error(t, err)

			logs := faultTestStructLogs(t, logger)
			require.Len(t, logs, tc.steps)
			require.JSONEq(t, tc.wantLast, string(logs[tc.steps-1]))
		})
	}
}

func TestStructLoggerFaultLimit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     string
		limit    int
		steps    int
		wantCost uint64
		wantErr  string
	}{
		{
			name:    "accepted step crosses limit",
			code:    "600056",
			limit:   100,
			steps:   2,
			wantErr: vm.ErrInvalidJump.Error(),
		},
		{
			// Both iterations reach the same JUMP pc and depth; only the first fits.
			name:     "skipped iteration cannot patch previous jump",
			code:     "60035b600190038056",
			limit:    450,
			steps:    7,
			wantCost: 8,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewStructLogger(&Config{Limit: tc.limit})
			code := common.Hex2Bytes(tc.code)
			evm, self := faultTestEVM(t, logger, code)
			defer evm.Release()

			contract := vm.NewContract(common.Address{}, self, new(uint256.Int), vm.NewGasBudget(1000, 0), nil)
			contract.SetCallCode(common.Hash{}, code)
			_, err := evm.Run(contract, nil, false)
			require.ErrorIs(t, err, vm.ErrInvalidJump)

			logs := faultTestStructLogs(t, logger)
			require.Len(t, logs, tc.steps)
			last := faultTestLog(t, logs[tc.steps-1])
			require.Equal(t, vm.JUMP.String(), last.Op)
			require.Equal(t, tc.wantCost, last.GasCost)
			require.Equal(t, tc.wantErr, last.Error)
		})
	}
}

func TestStructLoggerChildFault(t *testing.T) {
	logger := NewStructLogger(nil)
	child := common.HexToAddress("0xdead")

	// Call the child, discard its result, and stop.
	code := common.Hex2Bytes("6000600060006000600061dead5af15000")
	evm, self := faultTestEVM(t, logger, code)
	defer evm.Release()
	evm.StateDB.SetCode(child, common.Hex2Bytes("600056"), tracing.CodeChangeUnspecified)
	_, _, err := evm.Call(common.Address{}, self, nil, vm.NewGasBudget(1000000, 0), new(uint256.Int))
	require.NoError(t, err)

	logs := faultTestStructLogs(t, logger)
	require.Len(t, logs, 12)
	parent := faultTestLog(t, logs[7])
	require.Equal(t, vm.CALL.String(), parent.Op)
	require.NotZero(t, parent.GasCost)
	require.Empty(t, parent.Error)

	fault := faultTestLog(t, logs[9])
	require.Equal(t, vm.JUMP.String(), fault.Op)
	require.Equal(t, 2, fault.Depth)
	require.Zero(t, fault.GasCost)
	require.Equal(t, vm.ErrInvalidJump.Error(), fault.Error)
}

func TestStreamingStructLoggerFaults(t *testing.T) {
	for _, tc := range []struct {
		name       string
		code       string
		gas        uint64
		reservoir  uint64
		directRun  bool
		rows       int
		wantErr    error
		wantHeader string
	}{
		{
			name:       "invalid jump",
			code:       "600056",
			gas:        100,
			rows:       2,
			wantErr:    vm.ErrInvalidJump,
			wantHeader: "gas=97 cost=0 ERROR: invalid jump destination",
		},
		{
			name:       "revert",
			code:       "60206000fd",
			gas:        100,
			rows:       3,
			wantErr:    vm.ErrExecutionReverted,
			wantHeader: "gas=94 cost=3",
		},
		{
			name:       "create state out of gas",
			code:       "5f5f5ff0",
			gas:        100000,
			reservoir:  111,
			rows:       4,
			wantErr:    vm.ErrOutOfGas,
			wantHeader: "gas=99994 cost=0 stateCost=0 reservoir=111 ERROR: out of gas",
		},
		{
			name:       "direct run final step",
			code:       "00",
			gas:        100,
			directRun:  true,
			rows:       1,
			wantHeader: "gas=100 cost=0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			logger := NewStreamingStructLogger(nil, &out)
			code := common.Hex2Bytes(tc.code)
			evm, self := faultTestEVM(t, logger, code)
			defer evm.Release()

			var (
				err    error
				budget = vm.NewGasBudget(tc.gas, tc.reservoir)
			)
			if tc.directRun {
				contract := vm.NewContract(common.Address{}, self, new(uint256.Int), budget, nil)
				contract.SetCallCode(common.Hash{}, code)
				_, err = evm.Run(contract, nil, false)
				logger.Flush()
			} else {
				_, _, err = evm.Call(common.Address{}, self, nil, budget, new(uint256.Int))
			}
			if !errors.Is(err, tc.wantErr) || !errors.Is(logger.Error(), tc.wantErr) {
				t.Fatalf("execution/logger errors = %v/%v, want %v", err, logger.Error(), tc.wantErr)
			}
			if strings.Count(out.String(), "pc=") != tc.rows {
				t.Fatalf("missing or duplicated rows: %s", out.String())
			}
			var (
				record string
				op     = vm.OpCode(code[len(code)-1])
			)
			for _, row := range strings.Split(out.String(), "\n\n") {
				if strings.HasPrefix(row, op.String()+" ") {
					record = row
				}
			}
			exception := tc.wantErr != nil && tc.wantErr != vm.ErrExecutionReverted
			if !strings.Contains(record, tc.wantHeader) || strings.Contains(record, "ERROR:") != exception {
				t.Fatalf("unexpected final row: %s", record)
			}
		})
	}
}

func faultTestStructLogs(t *testing.T, logger *StructLogger) []json.RawMessage {
	t.Helper()
	result, err := logger.GetResult()
	require.NoError(t, err)

	var trace ExecutionResult
	require.NoError(t, json.Unmarshal(result, &trace))
	return trace.StructLogs
}

func faultTestLog(t *testing.T, entry json.RawMessage) structLogLegacy {
	t.Helper()
	var log structLogLegacy
	if err := json.Unmarshal(entry, &log); err != nil {
		t.Fatal(err)
	}
	return log
}

func faultTestEVM(t *testing.T, logger *StructLogger, code []byte) (*vm.EVM, common.Address) {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	self := common.HexToAddress("0xc0de")
	statedb.SetCode(self, code, tracing.CodeChangeUnspecified)
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)
	evm := vm.NewEVM(vm.BlockContext{
		CanTransfer:      core.CanTransfer,
		Transfer:         core.Transfer,
		BlockNumber:      new(big.Int),
		Random:           &common.Hash{},
		CostPerStateByte: params.CostPerStateByte,
	}, statedb, &config, vm.Config{Tracer: logger.Hooks()})
	logger.OnTxStart(evm.GetVMContext(), nil, common.Address{})
	return evm, self
}
