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

package tracetest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
	From              common.Address  `json:"from"`
	Gas               *hexutil.Uint64 `json:"gas"`
	GasUsed           *hexutil.Uint64 `json:"gasUsed"`
	To                *common.Address `json:"to,omitempty" rlp:"optional"`
	Input             hexutil.Bytes   `json:"input" rlp:"optional"`
	Output            hexutil.Bytes   `json:"output,omitempty" rlp:"optional"`
	Error             string          `json:"error,omitempty" rlp:"optional"`
	RevertReason      string          `json:"revertReason,omitempty"`
	Logs              []callLog       `json:"logs,omitempty" rlp:"optional"`
	Value             *hexutil.Big    `json:"value,omitempty" rlp:"optional"`
	revertedSnapshot  bool
	AccessedSlots     accessedSlots                              `json:"accessedSlots"`
	ExtCodeAccessInfo []common.Address                           `json:"extCodeAccessInfo"`
	UsedOpcodes       map[vm.OpCode]uint64                       `json:"usedOpcodes"`
	ContractSize      map[common.Address]*contractSizeWithOpcode `json:"contractSize"`
	OutOfGas          bool                                       `json:"outOfGas"`
	Calls             []erc7562Trace                             `json:"calls,omitempty" rlp:"optional"`
	Keccak            []hexutil.Bytes                            `json:"keccak,omitempty"`
	Type              string                                     `json:"type"`
}

// erc7562TracerTest defines a single test to check the erc7562 tracer against.
type erc7562TracerTest struct {
	Genesis      *core.Genesis   `json:"genesis"`
	Context      *callContext    `json:"context"`
	Input        string          `json:"input"`
	TracerConfig json.RawMessage `json:"tracerConfig"`
	Result       *erc7562Trace   `json:"result"`
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

			// Compare JSON ignoring key order by unmarshalling both into interfaces.
			var got, expected interface{}

			if err := json.Unmarshal(res, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal(want, &expected); err != nil {
				t.Fatalf("failed to unmarshal expected result: %v", err)
			}
			got = sortKeccakArrays(got)
			expected = sortKeccakArrays(expected)
			if !reflect.DeepEqual(got, expected) {
				diff(got, expected, "root")
				t.Fatalf("trace mismatch\n have: %v\n want: %v\n", got, expected)
			}

			// Sanity check: compare top call's gas used against vm result
			type simpleResult struct {
				GasUsed hexutil.Uint64
			}
			var topCall simpleResult
			if err := json.Unmarshal(res, &topCall); err != nil {
				t.Fatalf("failed to unmarshal top calls gasUsed: %v", err)
			}
			if uint64(topCall.GasUsed) != vmRet.UsedGas {
				t.Fatalf("top call has invalid gasUsed. have: %d want: %d", topCall.GasUsed, vmRet.UsedGas)
			}
		})
	}
}

func diff(a, b interface{}, path string) {
	// If both values are deeply equal, nothing to report.
	if reflect.DeepEqual(a, b) {
		return
	}

	// If the types differ, print the mismatch and return.
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		fmt.Printf("Type mismatch at %s: %T vs %T\n", path, a, b)
		return
	}

	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal := b.(map[string]interface{})
		// Check keys present in aVal.
		for k, va := range aVal {
			newPath := fmt.Sprintf("%s.%s", path, k)
			vb, exists := bVal[k]
			if !exists {
				fmt.Printf("Key %s present in a but missing in b: %v\n", newPath, va)
			} else {
				diff(va, vb, newPath)
			}
		}
		// Check keys present in bVal but missing in aVal.
		for k, vb := range bVal {
			newPath := fmt.Sprintf("%s.%s", path, k)
			if _, exists := aVal[k]; !exists {
				fmt.Printf("Key %s present in b but missing in a: %v\n", newPath, vb)
			}
		}
	case []interface{}:
		bVal := b.([]interface{})
		if len(aVal) != len(bVal) {
			fmt.Printf("Length mismatch at %s: %d vs %d\n", path, len(aVal), len(bVal))
		}
		// Compare each element.
		min := len(aVal)
		if len(bVal) < min {
			min = len(bVal)
		}
		for i := 0; i < min; i++ {
			diff(aVal[i], bVal[i], fmt.Sprintf("%s[%d]", path, i))
		}
	default:
		// For other types, just print the difference.
		fmt.Printf("Value mismatch at %s: %v vs %v\n", path, a, b)
	}
}

// sortKeccakArrays recursively traverses the JSON object and sorts any array found under the "keccak" key.
func sortKeccakArrays(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if k == "keccak" {
				if arr, ok := child.([]interface{}); ok {
					// Convert each element to a string.
					strs := make([]string, len(arr))
					for i, elem := range arr {
						strs[i] = fmt.Sprintf("%v", elem)
					}
					// Sort the strings.
					sort.Strings(strs)
					// Replace with sorted values.
					sortedArr := make([]interface{}, len(strs))
					for i, s := range strs {
						sortedArr[i] = s
					}
					val[k] = sortedArr
				}
			} else {
				val[k] = sortKeccakArrays(child)
			}
		}
		return val
	case []interface{}:
		for i, elem := range val {
			val[i] = sortKeccakArrays(elem)
		}
		return val
	default:
		return v
	}
}
