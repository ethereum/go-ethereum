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

func (*dummyStatedb) GetRefund() uint64                       { return 1337 }
func (*dummyStatedb) GetBalance(addr common.Address) *big.Int { return new(big.Int) }

type vmContext struct {
	blockCtx vm.BlockContext
	txCtx    vm.TxContext
}

func testCtx() *vmContext {
	return &vmContext{blockCtx: vm.BlockContext{BlockNumber: big.NewInt(1)}, txCtx: vm.TxContext{GasPrice: big.NewInt(100000)}}
}

func runTrace(tracer *Tracer, vmctx *vmContext) (json.RawMessage, error) {
	env := vm.NewEVM(vmctx.blockCtx, vmctx.txCtx, &dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})
	var (
		startGas uint64 = 10000
		value           = big.NewInt(0)
	)
	contract := vm.NewContract(account{}, account{}, value, startGas)
	contract.Code = []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, 0x0}

	tracer.CaptureStart(env, contract.Caller(), contract.Address(), false, []byte{}, startGas, value)
	ret, err := env.Interpreter().Run(contract, []byte{}, false)
	tracer.CaptureEnd(ret, startGas-contract.Gas, 1, err)
	if err != nil {
		return nil, err
	}
	return tracer.GetResult()
}

func TestTracer(t *testing.T) {
	execTracer := func(code string) []byte {
		t.Helper()
		ctx := &vmContext{blockCtx: vm.BlockContext{BlockNumber: big.NewInt(1)}, txCtx: vm.TxContext{GasPrice: big.NewInt(100000)}}
		tracer, err := New(code, ctx.txCtx)
		if err != nil {
			t.Fatal(err)
		}
		ret, err := runTrace(tracer, ctx)
		if err != nil {
			t.Fatal(err)
		}
		return ret
	}
	for i, tt := range []struct {
		code string
		want string
	}{
		{ // tests that we don't panic on bad arguments to memory access
			code: "{depths: [], step: function(log) { this.depths.push(log.memory.slice(-1,-2)); }, fault: function() {}, result: function() { return this.depths; }}",
			want: `[{},{},{}]`,
		}, { // tests that we don't panic on bad arguments to stack peeks
			code: "{depths: [], step: function(log) { this.depths.push(log.stack.peek(-1)); }, fault: function() {}, result: function() { return this.depths; }}",
			want: `["0","0","0"]`,
		}, { //  tests that we don't panic on bad arguments to memory getUint
			code: "{ depths: [], step: function(log, db) { this.depths.push(log.memory.getUint(-64));}, fault: function() {}, result: function() { return this.depths; }}",
			want: `["0","0","0"]`,
		}, { // tests some general counting
			code: "{count: 0, step: function() { this.count += 1; }, fault: function() {}, result: function() { return this.count; }}",
			want: `3`,
		}, { // tests that depth is reported correctly
			code: "{depths: [], step: function(log) { this.depths.push(log.stack.length()); }, fault: function() {}, result: function() { return this.depths; }}",
			want: `[0,1,2]`,
		}, { // tests to-string of opcodes
			code: "{opcodes: [], step: function(log) { this.opcodes.push(log.op.toString()); }, fault: function() {}, result: function() { return this.opcodes; }}",
			want: `["PUSH1","PUSH1","STOP"]`,
		}, { // tests intrinsic gas
			code: "{depths: [], step: function() {}, fault: function() {}, result: function(ctx) { return ctx.gasPrice+'.'+ctx.gasUsed+'.'+ctx.intrinsicGas; }}",
			want: `"100000.6.21000"`,
		},
	} {
		if have := execTracer(tt.code); tt.want != string(have) {
			t.Errorf("testcase %d: expected return value to be %s got %s\n\tcode: %v", i, tt.want, string(have), tt.code)
		}
	}
}

func TestHalt(t *testing.T) {
	t.Skip("duktape doesn't support abortion")

	timeout := errors.New("stahp")
	vmctx := testCtx()
	tracer, err := New("{step: function() { while(1); }, result: function() { return null; }}", vmctx.txCtx)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		tracer.Stop(timeout)
	}()

	if _, err = runTrace(tracer, vmctx); err.Error() != "stahp    in server-side tracer function 'step'" {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestHaltBetweenSteps(t *testing.T) {
	vmctx := testCtx()
	tracer, err := New("{step: function() {}, fault: function() {}, result: function() { return null; }}", vmctx.txCtx)
	if err != nil {
		t.Fatal(err)
	}
	env := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, vm.TxContext{}, &dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})
	scope := &vm.ScopeContext{
		Contract: vm.NewContract(&account{}, &account{}, big.NewInt(0), 0),
	}

	tracer.CaptureState(env, 0, 0, 0, 0, scope, nil, 0, nil)
	timeout := errors.New("stahp")
	tracer.Stop(timeout)
	tracer.CaptureState(env, 0, 0, 0, 0, scope, nil, 0, nil)

	if _, err := tracer.GetResult(); err.Error() != timeout.Error() {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

// TestNoStepExec tests a regular value transfer (no exec), and accessing the statedb
// in 'result'
func TestNoStepExec(t *testing.T) {
	runEmptyTrace := func(tracer *Tracer, vmctx *vmContext) (json.RawMessage, error) {
		env := vm.NewEVM(vmctx.blockCtx, vmctx.txCtx, &dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})
		startGas := uint64(10000)
		contract := vm.NewContract(account{}, account{}, big.NewInt(0), startGas)
		tracer.CaptureStart(env, contract.Caller(), contract.Address(), false, []byte{}, startGas, big.NewInt(0))
		tracer.CaptureEnd(nil, startGas-contract.Gas, 1, nil)
		return tracer.GetResult()
	}
	execTracer := func(code string) []byte {
		t.Helper()
		ctx := &vmContext{blockCtx: vm.BlockContext{BlockNumber: big.NewInt(1)}, txCtx: vm.TxContext{GasPrice: big.NewInt(100000)}}
		tracer, err := New(code, ctx.txCtx)
		if err != nil {
			t.Fatal(err)
		}
		ret, err := runEmptyTrace(tracer, ctx)
		if err != nil {
			t.Fatal(err)
		}
		return ret
	}
	for i, tt := range []struct {
		code string
		want string
	}{
		{ // tests that we don't panic on accessing the db methods
			code: "{depths: [], step: function() {}, fault: function() {},  result: function(ctx, db){ return db.getBalance(ctx.to)} }",
			want: `"0"`,
		},
	} {
		if have := execTracer(tt.code); tt.want != string(have) {
			t.Errorf("testcase %d: expected return value to be %s got %s\n\tcode: %v", i, tt.want, string(have), tt.code)
		}
	}
}
