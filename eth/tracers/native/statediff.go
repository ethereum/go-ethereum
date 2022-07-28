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
)

func init() {
	register("stateDiffTracer", newStateDiffTracer)
}

type diffstate = map[common.Address]*diffaccount
type diffaccount struct {
	Before account `json:"before"`
	After  account `json:"after"`
}

type stateDiffTracer struct {
	env       *vm.EVM
	diffstate diffstate
	create    bool
	from      common.Address
	to        common.Address
	gasLimit  uint64 // Amount of gas bought for the whole tx
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

func newStateDiffTracer(ctx *tracers.Context) tracers.Tracer {
	// First callframe contains tx context info
	// and is populated on start and end.
	return &stateDiffTracer{diffstate: diffstate{}}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *stateDiffTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	t.create = create
	t.from = from
	t.to = to

	t.lookupAccount(env.Context.Coinbase)
	t.lookupAccount(from)
	t.lookupAccount(to)

	// The recipient balance includes the value transferred.
	toBal := hexutil.MustDecodeBig(t.diffstate[to].Before.Balance)
	toBal = new(big.Int).Sub(toBal, value)
	t.diffstate[to].Before.Balance = hexutil.EncodeBig(toBal)

	// The sender balance is after reducing: value and gasLimit.
	// We need to re-add them to get the pre-tx balance.
	fromBal := hexutil.MustDecodeBig(t.diffstate[from].Before.Balance)
	gasPrice := env.TxContext.GasPrice
	consumedGas := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(t.gasLimit))
	fromBal.Add(fromBal, new(big.Int).Add(value, consumedGas))
	t.diffstate[from].Before.Balance = hexutil.EncodeBig(fromBal)
	t.diffstate[from].Before.Nonce--
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *stateDiffTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *stateDiffTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
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
func (t *stateDiffTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *stateDiffTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *stateDiffTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (t *stateDiffTracer) CaptureTxStart(gasLimit uint64) {
	t.gasLimit = gasLimit
}

func (t *stateDiffTracer) CaptureTxEnd(restGas uint64) {
	// Refund the from address with rest gas
	refundGas := new(big.Int).Mul(t.env.TxContext.GasPrice, new(big.Int).SetUint64(restGas))
	fromBal := hexutil.MustDecodeBig(t.diffstate[t.from].After.Balance)
	fromBal.Add(fromBal, refundGas)
	t.diffstate[t.from].After.Balance = hexutil.EncodeBig(fromBal)

	// Refund the used gas to miner
	miner := t.env.Context.Coinbase
	gasPrice := t.env.TxContext.GasPrice
	if !t.env.Config.NoBaseFee && t.env.Context.BaseFee != nil {
		gasPrice.Sub(gasPrice, t.env.Context.BaseFee)
	}
	usedGas := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(t.gasLimit-restGas))
	minerBal := hexutil.MustDecodeBig(t.diffstate[miner].After.Balance)
	minerBal.Add(minerBal, usedGas)
	t.diffstate[miner].After.Balance = hexutil.EncodeBig(minerBal)

	for addr, diff := range t.diffstate {
		for key := range diff.Before.Storage {
			t.diffstate[addr].After.Storage[key] = t.env.StateDB.GetState(addr, key)
		}
	}
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *stateDiffTracer) GetResult() (json.RawMessage, error) {
	res, err := json.Marshal(t.diffstate)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *stateDiffTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}

// lookupAccount fetches details of an account and adds it to the diffstate
// if it doesn't exist there.
func (t *stateDiffTracer) lookupAccount(addr common.Address) {
	if _, ok := t.diffstate[addr]; ok {
		return
	}
	t.diffstate[addr] = &diffaccount{
		Before: account{
			Balance: bigToHex(t.env.StateDB.GetBalance(addr)),
			Nonce:   t.env.StateDB.GetNonce(addr),
			Code:    bytesToHex(t.env.StateDB.GetCode(addr)),
			Storage: make(map[common.Hash]common.Hash),
		},
		After: account{
			Balance: bigToHex(t.env.StateDB.GetBalance(addr)),
			Nonce:   t.env.StateDB.GetNonce(addr),
			Code:    bytesToHex(t.env.StateDB.GetCode(addr)),
			Storage: make(map[common.Hash]common.Hash),
		},
	}
}

// lookupStorage fetches the requested storage slot and adds
// it to the diffstate of the given contract. It assumes `lookupAccount`
// has been performed on the contract before.
func (t *stateDiffTracer) lookupStorage(addr common.Address, key common.Hash) {
	if _, ok := t.diffstate[addr].Before.Storage[key]; ok {
		return
	}
	t.diffstate[addr].Before.Storage[key] = t.env.StateDB.GetState(addr, key)
}
