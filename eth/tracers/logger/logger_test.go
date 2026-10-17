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
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type dummyStatedb struct {
	state.StateDB
	nonce uint64
}

func (*dummyStatedb) GetRefund() uint64                                    { return 1337 }
func (db *dummyStatedb) GetNonce(common.Address) uint64                    { return db.nonce }
func (*dummyStatedb) GetState(_ common.Address, _ common.Hash) common.Hash { return common.Hash{} }
func (*dummyStatedb) SetState(_ common.Address, _ common.Hash, _ common.Hash) common.Hash {
	return common.Hash{}
}

func (*dummyStatedb) GetStateAndCommittedState(common.Address, common.Hash) (common.Hash, common.Hash) {
	return common.Hash{}, common.Hash{}
}

type testOpContext struct {
	memory  []byte
	stack   []uint256.Int
	address common.Address
}

func (c testOpContext) MemoryData() []byte       { return c.memory }
func (c testOpContext) StackData() []uint256.Int { return c.stack }
func (c testOpContext) Caller() common.Address   { return common.Address{} }
func (c testOpContext) Address() common.Address  { return c.address }
func (c testOpContext) CallValue() *uint256.Int  { return new(uint256.Int) }
func (c testOpContext) CallInput() []byte        { return nil }
func (c testOpContext) ContractCode() []byte     { return nil }

func testStack(values ...uint64) []uint256.Int {
	stack := make([]uint256.Int, len(values))
	for i, value := range values {
		stack[i].SetUint64(value)
	}
	return stack
}

func TestStructLoggerCapturesCreateAddress(t *testing.T) {
	var (
		caller = common.HexToAddress("0x1234")
		db     = &dummyStatedb{nonce: 7}
		logger = NewStructLogger(nil)
	)
	logger.OnTxStart(&tracing.VMContext{StateDB: db}, nil, common.Address{})
	logger.OnOpcode(0, byte(vm.CREATE), 0, 0, testOpContext{
		address: caller,
		stack:   testStack(0, 0, 0),
	}, nil, 0, nil)

	var result struct {
		CreateAddr *common.Address `json:"createAddr"`
	}
	if err := json.Unmarshal(logger.logs[0], &result); err != nil {
		t.Fatal(err)
	}
	want := crypto.CreateAddress(caller, db.nonce)
	if result.CreateAddr == nil || *result.CreateAddr != want {
		t.Fatalf("unexpected create address: have %v want %v", result.CreateAddr, want)
	}
}

func TestJSONLoggerCapturesCreate2Address(t *testing.T) {
	var (
		caller = common.HexToAddress("0x1234")
		memory = []byte{0xaa, 0xbb}
		salt   = uint64(42)
		output bytes.Buffer
		db     = new(dummyStatedb)
		hooks  = NewJSONLogger(nil, &output)
	)
	hooks.OnTxStart(&tracing.VMContext{StateDB: db}, nil, common.Address{})
	hooks.OnOpcode(0, byte(vm.CREATE2), 0, 0, testOpContext{
		address: caller,
		memory:  memory,
		stack:   testStack(salt, uint64(len(memory)), 0, 0),
	}, nil, 0, nil)

	var result struct {
		CreateAddr *common.Address `json:"createAddr"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	want := crypto.CreateAddress2(caller, testStack(salt)[0].Bytes32(), crypto.Keccak256(memory))
	if result.CreateAddr == nil || *result.CreateAddr != want {
		t.Fatalf("unexpected create2 address: have %v want %v", result.CreateAddr, want)
	}
}

func TestJSONLoggerCapturesCreate2AddressWithZeroSize(t *testing.T) {
	var (
		caller = common.HexToAddress("0x1234")
		salt   = uint64(42)
		output bytes.Buffer
		db     = new(dummyStatedb)
		hooks  = NewJSONLogger(nil, &output)
		stack  = testStack(salt, 0, 0, 0)
	)
	stack[2].SetBytes(bytes.Repeat([]byte{0xff}, 32))
	hooks.OnTxStart(&tracing.VMContext{StateDB: db}, nil, common.Address{})
	hooks.OnOpcode(0, byte(vm.CREATE2), 0, 0, testOpContext{
		address: caller,
		stack:   stack,
	}, nil, 0, nil)

	var result struct {
		CreateAddr *common.Address `json:"createAddr"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	want := crypto.CreateAddress2(caller, stack[0].Bytes32(), crypto.Keccak256(nil))
	if result.CreateAddr == nil || *result.CreateAddr != want {
		t.Fatalf("unexpected create2 address: have %v want %v", result.CreateAddr, want)
	}
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
