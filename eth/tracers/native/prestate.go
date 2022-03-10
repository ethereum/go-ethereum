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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	register("prestateTracer", newPrestateTracer)
}

type prestate = map[common.Address]*account
type account struct {
	Balance string                      `json:"balance"`
	Nonce   uint64                      `json:"nonce"`
	Code    string                      `json:"code"`
	Storage map[common.Hash]common.Hash `json:"storage"`
}

type prestateTracer struct {
	env       *vm.EVM
	prestate  prestate
	from      common.Address
	create    bool
	input     []byte
	gasLimit  uint64
	value     *big.Int
	to        common.Address
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

func newPrestateTracer() tracers.Tracer {
	// First callframe contains tx context info
	// and is populated on start and end.
	return &prestateTracer{prestate: prestate{}}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *prestateTracer) CaptureStart(to common.Address, gas uint64) {
	t.to = to

	// The recipient balance includes the value transferred.
	t.lookupAccount(to)
	toBal := hexutil.MustDecodeBig(t.prestate[to].Balance)
	toBal = new(big.Int).Sub(toBal, t.value)
	t.prestate[to].Balance = hexutil.EncodeBig(toBal)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *prestateTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
	if t.create {
		// Exclude created contract.
		delete(t.prestate, t.to)
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *prestateTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	stack := scope.Stack
	stackData := stack.Data()
	stackLen := len(stackData)
	switch {
	case stackLen >= 1 && (op == vm.SLOAD || op == vm.SSTORE):
		slot := common.Hash(stackData[stackLen-1].Bytes32())
		t.lookupStorage(scope.Contract.Address(), slot)
	case stackLen >= 1 && (op == vm.EXTCODECOPY || op == vm.EXTCODEHASH || op == vm.EXTCODESIZE || op == vm.BALANCE || op == vm.SELFDESTRUCT):
		addr := common.Address(stackData[stackLen-1].Bytes20())
		t.lookupAccount(addr)
	case stackLen >= 5 && (op == vm.DELEGATECALL || op == vm.CALL || op == vm.STATICCALL || op == vm.CALLCODE):
		addr := common.Address(stackData[stackLen-2].Bytes20())
		t.lookupAccount(addr)
	case op == vm.CREATE:
		addr := scope.Contract.Address()
		nonce := t.env.StateDB.GetNonce(addr)
		t.lookupAccount(crypto.CreateAddress(addr, nonce))
	case stackLen >= 4 && op == vm.CREATE2:
		offset := stackData[stackLen-2]
		size := stackData[stackLen-3]
		init := scope.Memory.GetCopy(int64(offset.Uint64()), int64(size.Uint64()))
		inithash := crypto.Keccak256(init)
		salt := stackData[stackLen-4]
		t.lookupAccount(crypto.CreateAddress2(scope.Contract.Address(), salt.Bytes32(), inithash))
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

func (t *prestateTracer) CaptureTxStart(env *vm.EVM, from common.Address, create bool, input []byte, gasLimit uint64, value *big.Int, rules params.Rules) {
	t.env = env
	t.from = from
	t.create = create
	t.input = input
	t.gasLimit = gasLimit
	t.value = value
	t.lookupAccount(t.from)
}

func (*prestateTracer) CaptureTxEnd(remainingGas uint64, err error) {}

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

// lookupAccount fetches details of an account and adds it to the prestate
// if it doesn't exist there.
func (t *prestateTracer) lookupAccount(addr common.Address) {
	if _, ok := t.prestate[addr]; ok {
		return
	}
	t.prestate[addr] = &account{
		Balance: bigToHex(t.env.StateDB.GetBalance(addr)),
		Nonce:   t.env.StateDB.GetNonce(addr),
		Code:    bytesToHex(t.env.StateDB.GetCode(addr)),
		Storage: make(map[common.Hash]common.Hash),
	}
}

// lookupStorage fetches the requested storage slot and adds
// it to the prestate of the given contract. It assumes `lookupAccount`
// has been performed on the contract before.
func (t *prestateTracer) lookupStorage(addr common.Address, key common.Hash) {
	if _, ok := t.prestate[addr].Storage[key]; ok {
		return
	}
	t.prestate[addr].Storage[key] = t.env.StateDB.GetState(addr, key)
}
