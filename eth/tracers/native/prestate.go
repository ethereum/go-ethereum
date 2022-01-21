// Copyright 2022 The go-ethereum Authors
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

package native

import (
	"encoding/json"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
	register("nativePrestateTracer", newPrestateTracer)
}

type prestate = map[common.Address]account
type account struct {
	Balance *big.Int                    `json:"balance"`
	Nonce   uint64                      `json:"nonce"`
	Code    string                      `json:"code"`
	Storage map[common.Hash]common.Hash `json:"storage"`
}

type prestateTracer struct {
	env       *vm.EVM
	prestate  prestate
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

func newPrestateTracer() tracers.Tracer {
	// First callframe contains tx context info
	// and is populated on start and end.
	return &prestateTracer{prestate: prestate{}}
}

func (t *prestateTracer) lookupAccount(addr common.Address) {
	if _, ok := t.prestate[addr]; ok {
		return
	}
	t.prestate[addr] = account{
		Balance: t.env.StateDB.GetBalance(addr),
		Nonce:   t.env.StateDB.GetNonce(addr),
		Code:    bytesToHex(t.env.StateDB.GetCode(addr)),
		Storage: make(map[common.Hash]common.Hash),
	}
}

func (t *prestateTracer) lookupStorage(addr common.Address, key common.Hash) {
	if _, ok := t.prestate[addr].Storage[key]; ok {
		return
	}
	t.prestate[addr].Storage[key] = t.env.StateDB.GetState(addr, key)
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *prestateTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	// TODO: check balance is after value deduction
	//t.lookupAccount(from)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *prestateTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *prestateTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if len(t.prestate) == 0 {
		t.lookupAccount(scope.Contract.Address())
	}
	stack := scope.Stack
	stackData := stack.Data()
	stackLen := len(stackData)
	if (op == vm.SLOAD || op == vm.SSTORE) && stackLen >= 1 {
		slot := common.Hash(stackData[stackLen-1].Bytes32())
		t.lookupStorage(scope.Contract.Address(), slot)
	}
	if (op == vm.EXTCODECOPY || op == vm.EXTCODEHASH || op == vm.EXTCODESIZE || op == vm.BALANCE || op == vm.SELFDESTRUCT) && stackLen >= 1 {
		addr := common.Address(stackData[stackLen-1].Bytes20())
		t.lookupAccount(addr)
	}
	if (op == vm.DELEGATECALL || op == vm.CALL || op == vm.STATICCALL || op == vm.CALLCODE) && stackLen >= 5 {
		addr := common.Address(stackData[stackLen-2].Bytes20())
		t.lookupAccount(addr)
	}
	if op == vm.CREATE {
		addr := scope.Contract.Address()
		nonce := t.env.StateDB.GetNonce(addr)
		t.lookupAccount(crypto.CreateAddress(addr, nonce))
	}
	if op == vm.CREATE2 {
		/*addr := scope.Contract.Address()
		offset := stackData[stackLen-2]
		size := stackData[stackLen-3]*/
		// TODO
		panic("unimplemented")
	}
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *prestateTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *prestateTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *prestateTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *prestateTracer) GetResult() (json.RawMessage, error) {
	res, err := json.Marshal(t.prestate)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *prestateTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}
