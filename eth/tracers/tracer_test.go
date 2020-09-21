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

package tracers

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type account struct{}

func (account) SubBalance(amount *big.Int)                          {}
func (account) AddBalance(amount *big.Int)                          {}
func (account) SetAddress(common.Address)                           {}
func (account) Value() *big.Int                                     { return nil }
func (account) SetBalance(*big.Int)                                 {}
func (account) SetNonce(uint64)                                     {}
func (account) Balance() *big.Int                                   { return nil }
func (account) Address() common.Address                             { return common.Address{} }
func (account) ReturnGas(*big.Int)                                  {}
func (account) SetCode(common.Hash, []byte)                         {}
func (account) ForEachStorage(cb func(key, value common.Hash) bool) {}

type dummyStatedb struct {
	state.StateDB
}

func (*dummyStatedb) GetRefund() uint64 { return 1337 }

func runTrace(tracer Tracer) (json.RawMessage, error) {
	env := vm.NewEVM(vm.Context{BlockNumber: big.NewInt(1)}, &dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})

	contract := vm.NewContract(account{}, account{}, big.NewInt(0), 10000)
	contract.Code = []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, 0x0}

	_, err := env.Interpreter().Run(contract, []byte{}, false)
	if err != nil {
		return nil, err
	}
	return tracer.GetResult()
}

// TestRegressionPanicSlice tests that we don't panic on bad arguments to memory access
func TestRegressionPanicSlice(t *testing.T) {
	tracer, err := New("{depths: [], step: function(log) { this.depths.push(log.memory.slice(-1,-2)); }, fault: function() {}, result: function() { return this.depths; }}")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runTrace(tracer); err != nil {
		t.Fatal(err)
	}
}

// TestRegressionPanicSlice tests that we don't panic on bad arguments to stack peeks
func TestRegressionPanicPeek(t *testing.T) {
	tracer, err := New("{depths: [], step: function(log) { this.depths.push(log.stack.peek(-1)); }, fault: function() {}, result: function() { return this.depths; }}")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runTrace(tracer); err != nil {
		t.Fatal(err)
	}
}

// TestRegressionPanicSlice tests that we don't panic on bad arguments to memory getUint
func TestRegressionPanicGetUint(t *testing.T) {
	tracer, err := New("{ depths: [], step: function(log, db) { this.depths.push(log.memory.getUint(-64));}, fault: function() {}, result: function() { return this.depths; }}")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runTrace(tracer); err != nil {
		t.Fatal(err)
	}
}

func TestTracing(t *testing.T) {
	tracer, err := New("{count: 0, step: function() { this.count += 1; }, fault: function() {}, result: function() { return this.count; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("3")) {
		t.Errorf("Expected return value to be 3, got %s", string(ret))
	}
}

// wasmTraceCode is the []byte-serialized compiled WASM output of the
// following C code:
// #define WASM_EXPORT __attribute__((visibility("default")))
//
// static int count = 0;
//
// WASM_EXPORT
// void step() {
//   count++;
// }
//
// WASM_EXPORT
// void fault() {
// }
//
// WASM_EXPORT
// unsigned char *result() {
//   return &count;
// }
var wasmTraceCode []byte = []byte{0, 97, 115, 109, 1, 0, 0, 0, 1, 8, 2, 96, 0, 0, 96, 0, 1, 127, 3, 5, 4, 0, 0, 0, 1, 4, 5, 1, 112, 1, 1, 1, 5, 3, 1, 0, 2, 6, 21, 3, 127, 1, 65, 144, 136, 4, 11, 127, 0, 65, 144, 136, 4, 11, 127, 0, 65, 132, 8, 11, 7, 61, 6, 4, 115, 116, 101, 112, 0, 1, 5, 102, 97, 117, 108, 116, 0, 2, 6, 114, 101, 115, 117, 108, 116, 0, 3, 6, 109, 101, 109, 111, 114, 121, 2, 0, 11, 95, 95, 104, 101, 97, 112, 95, 98, 97, 115, 101, 3, 1, 10, 95, 95, 100, 97, 116, 97, 95, 101, 110, 100, 3, 2, 10, 31, 4, 2, 0, 11, 17, 0, 65, 0, 65, 0, 40, 2, 128, 8, 65, 1, 106, 54, 2, 128, 8, 11, 2, 0, 11, 5, 0, 65, 128, 8, 11, 11, 11, 1, 0, 65, 128, 8, 11, 4, 0, 0, 0, 0, 0, 59, 4, 110, 97, 109, 101, 1, 41, 4, 0, 17, 95, 95, 119, 97, 115, 109, 95, 99, 97, 108, 108, 95, 99, 116, 111, 114, 115, 1, 4, 115, 116, 101, 112, 2, 5, 102, 97, 117, 108, 116, 3, 6, 114, 101, 115, 117, 108, 116, 2, 9, 4, 0, 0, 1, 0, 2, 0, 3, 0}

func TestWasmTracing(t *testing.T) {
	tracer, err := New(string(wasmTraceCode))
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("3")) {
		t.Errorf("Expected return value to be 3, got %s", string(ret))
	}
}

func TestStack(t *testing.T) {
	tracer, err := New("{depths: [], step: function(log) { this.depths.push(log.stack.length()); }, fault: function() {}, result: function() { return this.depths; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("[0,1,2]")) {
		t.Errorf("Expected return value to be [0,1,2], got %s", string(ret))
	}
}

func TestOpcodes(t *testing.T) {
	tracer, err := New("{opcodes: [], step: function(log) { this.opcodes.push(log.op.toString()); }, fault: function() {}, result: function() { return this.opcodes; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("[\"PUSH1\",\"PUSH1\",\"STOP\"]")) {
		t.Errorf("Expected return value to be [\"PUSH1\",\"PUSH1\",\"STOP\"], got %s", string(ret))
	}
}

func TestHalt(t *testing.T) {
	t.Skip("duktape doesn't support abortion")

	timeout := errors.New("stahp")
	tracer, err := New("{step: function() { while(1); }, result: function() { return null; }}")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		tracer.Stop(timeout)
	}()

	if _, err = runTrace(tracer); err.Error() != "stahp    in server-side tracer function 'step'" {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestHaltBetweenSteps(t *testing.T) {
	tracer, err := New("{step: function() {}, fault: function() {}, result: function() { return null; }}")
	if err != nil {
		t.Fatal(err)
	}

	env := vm.NewEVM(vm.Context{BlockNumber: big.NewInt(1)}, &dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})
	contract := vm.NewContract(&account{}, &account{}, big.NewInt(0), 0)

	tracer.CaptureState(env, 0, 0, 0, 0, nil, nil, nil, nil, contract, 0, nil)
	timeout := errors.New("stahp")
	tracer.Stop(timeout)
	tracer.CaptureState(env, 0, 0, 0, 0, nil, nil, nil, nil, contract, 0, nil)

	if _, err := tracer.GetResult(); err.Error() != timeout.Error() {
		t.Errorf("Expected timeout error, got %v", err)
	}
}
