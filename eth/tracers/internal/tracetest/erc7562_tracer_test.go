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

package tracetest

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/stretchr/testify/require"
)

type accessedSlots struct {
	Reads           map[string][]string `json:"reads"`
	Writes          map[string]uint64   `json:"writes"`
	TransientReads  map[string]uint64   `json:"transientReads"`
	TransientWrites map[string]uint64   `json:"transientWrites"`
}
type contractSizeWithOpcode struct {
	ContractSize int       `json:"contractSize"`
	Opcode       vm.OpCode `json:"opcode"`
}

// erc7562Trace is the result of a erc7562Tracer run.
type erc7562Trace struct {
	From              common.Address                             `json:"from"`
	Gas               *hexutil.Uint64                            `json:"gas"`
	GasUsed           *hexutil.Uint64                            `json:"gasUsed"`
	To                *common.Address                            `json:"to,omitempty" rlp:"optional"`
	Input             hexutil.Bytes                              `json:"input" rlp:"optional"`
	Output            hexutil.Bytes                              `json:"output,omitempty" rlp:"optional"`
	Error             string                                     `json:"error,omitempty" rlp:"optional"`
	RevertReason      string                                     `json:"revertReason,omitempty"`
	Logs              []callLog                                  `json:"logs,omitempty" rlp:"optional"`
	Value             *hexutil.Big                               `json:"value,omitempty" rlp:"optional"`
	AccessedSlots     accessedSlots                              `json:"accessedSlots"`
	ExtCodeAccessInfo []common.Address                           `json:"extCodeAccessInfo"`
	UsedOpcodes       map[hexutil.Uint64]uint64                  `json:"usedOpcodes"`
	ContractSize      map[common.Address]*contractSizeWithOpcode `json:"contractSize"`
	OutOfGas          bool                                       `json:"outOfGas"`
	Calls             []erc7562Trace                             `json:"calls,omitempty" rlp:"optional"`
	Keccak            []hexutil.Bytes                            `json:"keccak,omitempty"`
	Type              string                                     `json:"type"`
}

// erc7562TracerTest defines a single test to check the erc7562 tracer against.
type erc7562TracerTest struct {
	tracerTestEnv
	Result *erc7562Trace `json:"result"`
}

func TestErc7562Tracer(t *testing.T) {
	dirPath := "erc7562_tracer"
	tracerName := "erc7562Tracer"
	files, err := os.ReadDir(filepath.Join("testdata", dirPath))
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		t.Run(camel(strings.TrimSuffix(file.Name(), ".json")), func(t *testing.T) {
			t.Parallel()

			var (
				test = new(erc7562TracerTest)
				tx   = new(types.Transaction)
			)
			// erc7562 tracer test found, read if from disk
			if blob, err := os.ReadFile(filepath.Join("testdata", dirPath, file.Name())); err != nil {
				t.Fatalf("failed to read testcase: %v", err)
			} else if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}
			if err := tx.UnmarshalBinary(common.FromHex(test.Input)); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}
			// Configure a blockchain with the given prestate
			var (
				signer  = types.MakeSigner(test.Genesis.Config, new(big.Int).SetUint64(uint64(test.Context.Number)), uint64(test.Context.Time))
				context = test.Context.toBlockContext(test.Genesis)
				st      = tests.MakePreState(rawdb.NewMemoryDatabase(), test.Genesis.Alloc, false, rawdb.HashScheme)
			)
			st.Close()

			tracer, err := tracers.DefaultDirectory.New(tracerName, new(tracers.Context), test.TracerConfig, test.Genesis.Config)
			if err != nil {
				t.Fatalf("failed to create erc7562 tracer: %v", err)
			}
			logState := vm.StateDB(st.StateDB)
			if tracer.Hooks != nil {
				logState = state.NewHookedState(st.StateDB, tracer.Hooks)
			}
			msg, err := core.TransactionToMessage(tx, signer, context.BaseFee)
			if err != nil {
				t.Fatalf("failed to prepare transaction for tracing: %v", err)
			}
			evm := vm.NewEVM(context, logState, test.Genesis.Config, vm.Config{Tracer: tracer.Hooks})
			tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
			vmRet, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
			if err != nil {
				t.Fatalf("failed to execute transaction: %v", err)
			}
			tracer.OnTxEnd(&types.Receipt{GasUsed: vmRet.UsedGas}, nil)
			// Retrieve the trace result and compare against the expected.
			res, err := tracer.GetResult()
			if err != nil {
				t.Fatalf("failed to retrieve trace result: %v", err)
			}
			want, err := json.Marshal(test.Result)
			if err != nil {
				t.Fatalf("failed to marshal test: %v", err)
			}
			require.JSONEq(t, string(res), string(want))
		})
	}
}
