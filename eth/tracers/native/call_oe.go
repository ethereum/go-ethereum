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

/*
* This tracer aims to replicate the data extracted from openethereum.
* Use in combination with geth-tracer to achieve same format.
* Differences between this tracer and the normal callTracer in call.go:
*
* - Internal calls (to addresses 0x0000...0001 - 0x0000...0009) are skipped
* - STATICCALL and DELEGATECALL frames will assume the value of their parent frame;
*
* This way we get data more resembling openethereum's tracer
 */

package native

import (
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"sync/atomic"
)

func init() {
	tracers.DefaultDirectory.Register("callTracerOE", newOpenEthereumTracer, false)
}

type callTracerNoInternals struct {
	env       *vm.EVM
	callstack []callFrame
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
	skipExit  bool   // flag whether to skip an CaptureExit call (used to skip internal calls)
}

// newCallTracer returns a native go tracer which tracks
// call frames of a tx, and implements vm.EVMLogger.
func newOpenEthereumTracer(_ *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	// First callframe contains tx context info
	// and is populated on start and end.
	return &callTracerNoInternals{callstack: make([]callFrame, 1)}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *callTracerNoInternals) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env

	t.callstack[0] = callFrame{
		Type:  vm.CALL,
		From:  from,
		To:    &to,
		Input: input,
		Gas:   gas,
		Value: value,
	}
	if create {
		t.callstack[0].Type = vm.CREATE
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *callTracerNoInternals) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.callstack[0].GasUsed = gasUsed
	if err != nil {
		t.callstack[0].Error = err.Error()
		if err.Error() == "execution reverted" && len(output) > 0 {
			t.callstack[0].Output = output
		}
	} else {
		t.callstack[0].Output = output
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *callTracerNoInternals) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *callTracerNoInternals) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *callTracerNoInternals) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Skip if tracing was interrupted

	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

	// check if target address is a precompiled contract (OpenEthereum omits such calls)
	// https://ethereum.stackexchange.com/questions/15479/list-of-pre-compiled-contracts
	if _, ok := vm.PrecompiledContractsBerlin[to]; ok {
		t.skipExit = true
		return
	}

	if typ == vm.DELEGATECALL {
		// OpenEthereum's DELEGATECALL logic will assume the value of parent call frame
		// immediate successor frames of delegate call get 0 as value (e.g. nested DELEGATECALL)
		i := len(t.callstack) - 1
		for t.callstack[i].Type == vm.DELEGATECALL {
			i-- // don't need to check for IndexOutOfBounds because root call is always of type 'CALL'
		}
		value = t.callstack[i].Value
	} else if typ == vm.STATICCALL {
		// OpenEthereum STATICCALLs are simply 0 valued
		value = big.NewInt(0)
	}

	call := callFrame{
		Type:  typ,
		From:  from,
		To:    &to,
		Input: input,
		Gas:   gas,
		Value: value,
	}
	t.callstack = append(t.callstack, call)

}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *callTracerNoInternals) CaptureExit(output []byte, gasUsed uint64, err error) {

	// skip exit frame if we are dealing with an internal call
	if t.skipExit {
		t.skipExit = false
		return
	}

	size := len(t.callstack)
	if size <= 1 {
		return
	}
	// pop call
	call := t.callstack[size-1]
	t.callstack = t.callstack[:size-1]
	size -= 1

	call.GasUsed = gasUsed
	if err == nil {
		call.Output = output
	} else {
		call.Error = err.Error()
		if call.Type == vm.CREATE || call.Type == vm.CREATE2 {
			call.To = nil
		}
	}
	t.callstack[size-1].Calls = append(t.callstack[size-1].Calls, call)
}

func (*callTracerNoInternals) CaptureTxStart(gasLimit uint64) {}

func (*callTracerNoInternals) CaptureTxEnd(restGas uint64) {}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *callTracerNoInternals) GetResult() (json.RawMessage, error) {
	if len(t.callstack) != 1 {
		return nil, errors.New("incorrect number of top-level calls")
	}
	res, err := json.Marshal(t.callstack[0])
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *callTracerNoInternals) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}
