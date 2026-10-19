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

package logger

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type dummyStatedb struct {
	state.StateDB
}

func (*dummyStatedb) GetRefund() uint64                                    { return 1337 }
func (*dummyStatedb) GetState(_ common.Address, _ common.Hash) common.Hash { return common.Hash{} }
func (*dummyStatedb) SetState(_ common.Address, _ common.Hash, _ common.Hash) common.Hash {
	return common.Hash{}
}

func (*dummyStatedb) GetStateAndCommittedState(common.Address, common.Hash) (common.Hash, common.Hash) {
	return common.Hash{}, common.Hash{}
}

func TestStoreCapture(t *testing.T) {
	var (
		logger   = NewStructLogger(nil)
		evm      = vm.NewEVM(vm.BlockContext{}, &dummyStatedb{}, params.TestChainConfig, vm.Config{Tracer: logger.Hooks()})
		contract = vm.NewContract(common.Address{}, common.Address{}, new(uint256.Int), vm.NewGasBudget(100000, 0), nil)
	)
	contract.Code = []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x0, byte(vm.SSTORE)}
	var index common.Hash
	logger.OnTxStart(evm.GetVMContext(), nil, common.Address{})
	_, err := evm.Run(contract, []byte{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(logger.storage[contract.Address()]) == 0 {
		t.Fatalf("expected exactly 1 changed value on address %x, got %d", contract.Address(),
			len(logger.storage[contract.Address()]))
	}
	exp := common.BigToHash(big.NewInt(1))
	if logger.storage[contract.Address()][index] != exp {
		t.Errorf("expected %x, got %x", exp, logger.storage[contract.Address()][index])
	}
}

// Tests that blank fields don't appear in logs when JSON marshalled, to reduce
// logs bloat and confusion. See https://github.com/ethereum/go-ethereum/issues/24487
func TestStructLogMarshalingOmitEmpty(t *testing.T) {
	tests := []struct {
		name string
		log  *StructLog
		want string
	}{
		{"empty err and no fields", &StructLog{},
			`{"pc":0,"op":0,"gas":"0x0","gasCost":"0x0","memSize":0,"stack":null,"depth":0,"refund":0,"opName":"STOP"}`},
		{"with err", &StructLog{Err: errors.New("this failed")},
			`{"pc":0,"op":0,"gas":"0x0","gasCost":"0x0","memSize":0,"stack":null,"depth":0,"refund":0,"opName":"STOP","error":"this failed"}`},
		{"with mem", &StructLog{Memory: make([]byte, 2), MemorySize: 2},
			`{"pc":0,"op":0,"gas":"0x0","gasCost":"0x0","memory":"0x0000","memSize":2,"stack":null,"depth":0,"refund":0,"opName":"STOP"}`},
		{"with 0-size mem", &StructLog{Memory: make([]byte, 0)},
			`{"pc":0,"op":0,"gas":"0x0","gasCost":"0x0","memSize":0,"stack":null,"depth":0,"refund":0,"opName":"STOP"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob, err := json.Marshal(tt.log)
			if err != nil {
				t.Fatal(err)
			}
			if have, want := string(blob), tt.want; have != want {
				t.Fatalf("mismatched results\n\thave: %v\n\twant: %v", have, want)
			}
		})
	}
}

func TestStructLogLegacyJSONSpecFormatting(t *testing.T) {
	tests := []struct {
		name string
		log  *StructLog
		want string
	}{
		{
			name: "omits empty error and pads memory/storage",
			log: &StructLog{
				Pc:         7,
				Op:         vm.SSTORE,
				Gas:        100,
				GasCost:    20,
				Memory:     []byte{0xaa, 0xbb},
				Storage:    map[common.Hash]common.Hash{common.BigToHash(big.NewInt(1)): common.BigToHash(big.NewInt(2))},
				Depth:      1,
				ReturnData: []byte{0x12, 0x34},
			},
			want: `{"pc":7,"op":"SSTORE","gas":100,"gasCost":20,"depth":1,"returnData":"0x1234","memory":["0xaabb000000000000000000000000000000000000000000000000000000000000"],"storage":{"0x0000000000000000000000000000000000000000000000000000000000000001":"0x0000000000000000000000000000000000000000000000000000000000000002"}}`,
		},
		{
			name: "includes error only when present",
			log: &StructLog{
				Pc:      1,
				Op:      vm.STOP,
				Gas:     2,
				GasCost: 3,
				Depth:   1,
				Err:     errors.New("boom"),
			},
			want: `{"pc":1,"op":"STOP","gas":2,"gasCost":3,"depth":1,"error":"boom"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			have := string(tt.log.toLegacyJSON())
			if have != tt.want {
				t.Fatalf("mismatched results\n\thave: %v\n\twant: %v", have, tt.want)
			}
		})
	}
}

// Tests that a halted SSTORE records no storage on its step and that the
// attempted value does not enter the storage snapshot of later steps.
func TestHaltedStoreCapture(t *testing.T) {
	var (
		logger = NewStructLogger(nil)
		addr   = common.HexToAddress("0xc0de")
		code   = []byte{
			// With calldata, attempt the SSTORE (inside the static call).
			byte(vm.CALLDATASIZE), byte(vm.PUSH1), 0x15, byte(vm.JUMPI),
			// Without calldata, STATICCALL self with one byte of calldata, then SLOAD another slot.
			byte(vm.PUSH1), 0x0, byte(vm.PUSH1), 0x0, byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x0,
			byte(vm.ADDRESS), byte(vm.GAS), byte(vm.STATICCALL), byte(vm.POP),
			byte(vm.PUSH1), 0x1, byte(vm.SLOAD), byte(vm.POP), byte(vm.STOP),
			byte(vm.JUMPDEST), byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x0, byte(vm.SSTORE), byte(vm.STOP),
		}
		statedb, _ = state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	)
	statedb.SetCode(addr, code, tracing.CodeChangeUnspecified)
	_, _, err := runtime.Call(addr, nil, &runtime.Config{
		State:     statedb,
		GasLimit:  1000000,
		EVMConfig: vm.Config{Tracer: logger.Hooks()},
	})
	if err != nil {
		t.Fatal(err)
	}
	var sstore, sload map[string]json.RawMessage
	for _, entry := range logger.logs {
		var step map[string]json.RawMessage
		if err := json.Unmarshal(entry, &step); err != nil {
			t.Fatal(err)
		}
		switch string(step["op"]) {
		case `"SSTORE"`:
			sstore = step
		case `"SLOAD"`:
			sload = step
		}
	}
	if sstore == nil || sload == nil {
		t.Fatalf("missing SSTORE or SLOAD step: %s", logger.logs)
	}
	if sstore["error"] == nil {
		t.Fatalf("expected SSTORE to halt, got %s", logger.logs)
	}
	if storage, ok := sstore["storage"]; ok {
		t.Errorf("halted SSTORE must not record storage, got %s", storage)
	}
	want := `{"0x0000000000000000000000000000000000000000000000000000000000000001":"0x0000000000000000000000000000000000000000000000000000000000000000"}`
	if have := string(sload["storage"]); have != want {
		t.Errorf("SLOAD storage mismatch\n\thave: %v\n\twant: %v", have, want)
	}
}

// Tests that after Amsterdam the struct logger reports the EIP-8037 state gas
// charged or refilled by an opcode and the reservoir it draws from, and omits
// both fields when they are zero.
func TestStateGasCapture(t *testing.T) {
	var (
		logger = NewStructLogger(nil)
		addr   = common.HexToAddress("0xc0de")
		config = *params.MergedTestChainConfig
		// Create slot 0, then clear it again: the second SSTORE refills the
		// creation charge of the first.
		code = []byte{
			byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x0, byte(vm.SSTORE),
			byte(vm.PUSH1), 0x0, byte(vm.PUSH1), 0x0, byte(vm.SSTORE),
			byte(vm.STOP),
		}
		gas     = params.MaxTxGas + 200_000 // reservoir covering the SSTORE state cost
		statedb = state.NewDatabaseForTesting()
	)
	config.AmsterdamTime = new(uint64)
	st, _ := state.New(types.EmptyRootHash, statedb)
	st.SetCode(addr, code, tracing.CodeChangeUnspecified)
	if _, _, err := runtime.Call(addr, nil, &runtime.Config{
		ChainConfig: &config,
		State:       st,
		GasLimit:    gas,
		EVMConfig:   vm.Config{Tracer: logger.Hooks()},
	}); err != nil {
		t.Fatal(err)
	}
	var (
		reservoir = gas - params.MaxTxGas
		stateCost = int64(params.StorageCreationSize * params.CostPerStateByte)
		want      = []struct {
			cost      int64
			reservoir uint64
		}{
			{0, reservoir},                              // PUSH1
			{0, reservoir},                              // PUSH1
			{stateCost, reservoir},                      // SSTORE, creates the slot
			{0, reservoir - uint64(stateCost)},          // PUSH1
			{0, reservoir - uint64(stateCost)},          // PUSH1
			{-stateCost, reservoir - uint64(stateCost)}, // SSTORE, clears it again
			{0, reservoir},                              // STOP
		}
	)
	if len(logger.logs) != len(want) {
		t.Fatalf("wrong number of steps: have %d, want %d", len(logger.logs), len(want))
	}
	for i, entry := range logger.logs {
		var step struct {
			StateGasCost      *int64  `json:"stateGasCost"`
			StateGasReservoir *uint64 `json:"stateGasReservoir"`
		}
		if err := json.Unmarshal(entry, &step); err != nil {
			t.Fatal(err)
		}
		if have := step.StateGasCost; (have == nil) != (want[i].cost == 0) || (have != nil && *have != want[i].cost) {
			t.Errorf("step %d: stateGasCost mismatch, want %d: %s", i, want[i].cost, entry)
		}
		if have := step.StateGasReservoir; have == nil || *have != want[i].reservoir {
			t.Errorf("step %d: stateGasReservoir mismatch, want %d: %s", i, want[i].reservoir, entry)
		}
	}
}
