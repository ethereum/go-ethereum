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

package parity

import (
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
)

func TestErrorLabelPrecheckFailures(t *testing.T) {
	for err, want := range map[error]string{
		vm.ErrInsufficientBalance: "Insufficient balance for transfer",
		vm.ErrDepth:               "Max call depth exceeded",
		vm.ErrNonceUintOverflow:   "Nonce overflow",
	} {
		if have := errorLabel(err, false); have != want {
			t.Errorf("%v: have %q, want %q", err, have, want)
		}
	}
}

var contract = common.HexToAddress("0xc0de")

// newDirectoryTracer constructs the parityTracer of the tracer directory.
func newDirectoryTracer(t *testing.T, config string) *tracers.Tracer {
	t.Helper()
	var cfg json.RawMessage
	if config != "" {
		cfg = json.RawMessage(config)
	}
	tracer, err := tracers.DefaultDirectory.New("parityTracer", new(tracers.Context), cfg, params.AllEthashProtocolChanges)
	if err != nil {
		t.Fatal(err)
	}
	return tracer
}

// runContract calls the contract with the given code under the hooks.
func runContract(hooks *tracing.Hooks, chainConfig *params.ChainConfig, code []byte, gas uint64) error {
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.SetCode(contract, code, tracing.CodeChangeUnspecified)
	_, _, err := runtime.Call(contract, nil, &runtime.Config{ChainConfig: chainConfig, GasLimit: gas, State: statedb, EVMConfig: vm.Config{Tracer: hooks}})
	return err
}

// Without a selected output family the directory tracer records the call
// frames only. The runtime does not hook state changes, so the selected state
// diff is checked for presence only.
func TestDirectoryTracerSelection(t *testing.T) {
	// Store 1 at slot 0.
	code := []byte{byte(vm.PUSH1), 1, byte(vm.PUSH1), 0, byte(vm.SSTORE)}
	for config, want := range map[string]struct{ frames, vmTrace, stateDiff bool }{
		"":                   {frames: true},
		`{}`:                 {frames: true},
		`{"vmTrace":true}`:   {vmTrace: true},
		`{"stateDiff":true}`: {stateDiff: true},
	} {
		tracer := newDirectoryTracer(t, config)
		if err := runContract(tracer.Hooks, params.AllEthashProtocolChanges, code, 100_000); err != nil {
			t.Fatal(err)
		}
		output, err := tracer.GetResult()
		if err != nil {
			t.Fatal(err)
		}
		var result Result
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatal(err)
		}
		if have := len(result.Trace) == 1; have != want.frames {
			t.Errorf("%q: frames %v, want %v", config, result.Trace, want.frames)
		}
		if have := result.VMTrace != nil && len(result.VMTrace.Ops) == 3; have != want.vmTrace {
			t.Errorf("%q: vmTrace %v, want %v", config, result.VMTrace, want.vmTrace)
		}
		if have := result.StateDiff != nil; have != want.stateDiff {
			t.Errorf("%q: stateDiff %v, want %v", config, result.StateDiff, want.stateDiff)
		}
	}
}

// Stop may come from another goroutine while the transaction executes, as
// the timeout of the debug API does, and the result then reports it.
func TestDirectoryTracerStop(t *testing.T) {
	tracer := newDirectoryTracer(t, `{"trace":true,"vmTrace":true,"stateDiff":true}`)
	// Signal the first instruction of a loop that runs until out of gas.
	hooks := *tracer.Hooks
	started := make(chan struct{})
	var once sync.Once
	hooks.OnOpcode = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
		once.Do(func() { close(started) })
		tracer.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
	}
	loop := []byte{byte(vm.JUMPDEST), byte(vm.PUSH1), 0, byte(vm.JUMP)}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runContract(&hooks, params.AllEthashProtocolChanges, loop, 10_000_000)
	}()
	<-started
	stopped := errors.New("stopped")
	tracer.Stop(stopped)
	<-done
	if _, err := tracer.GetResult(); !errors.Is(err, stopped) {
		t.Fatalf("have %v, want %v", err, stopped)
	}
}

// The EIP-8024 instructions push the words to the depth of their immediate.
func TestVMTraceImmediateDepth(t *testing.T) {
	chainConfig := *params.MergedTestChainConfig
	chainConfig.AmsterdamTime = new(uint64)

	// The stack holds 1 to 18, with 18 on top.
	words := make([]uint64, 0, 18)
	for i := uint64(1); i <= 18; i++ {
		words = append(words, i)
	}
	for name, test := range map[string]struct {
		op        vm.OpCode
		immediate byte
		push      []uint64
	}{
		// DUPN 17 duplicates the 17th word.
		"DUPN": {vm.DUPN, 0x80, append(slices.Clone(words[1:]), 2)},
		// SWAPN 17 swaps the top and the 18th word.
		"SWAPN": {vm.SWAPN, 0x80, append(append([]uint64{18}, words[1:17]...), 1)},
		// EXCHANGE 1 17 swaps the 2nd and the 18th word.
		"EXCHANGE": {vm.EXCHANGE, 0x4f, append(append([]uint64{17}, words[1:16]...), 1, 18)},
	} {
		var code []byte
		for _, word := range words {
			code = append(code, byte(vm.PUSH1), byte(word))
		}
		code = append(code, byte(test.op), test.immediate)

		tracer := New(Config{VMTrace: true}, &chainConfig, nil)
		if err := runContract(tracer.Hooks(), &chainConfig, code, 100_000); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		result, err := tracer.Result()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ops := result.VMTrace.Ops
		var have []uint64
		for _, word := range ops[len(ops)-1].Ex.Push {
			have = append(have, word.ToInt().Uint64())
		}
		if !slices.Equal(have, test.push) {
			t.Errorf("%s: push %v, want %v", name, have, test.push)
		}
	}
}
