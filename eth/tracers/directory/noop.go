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

package directory

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

func init() {
	DefaultDirectory.Register("noopTracer", newNoopTracer, false)
}

// NoopTracer is a go implementation of the Tracer interface which
// performs no action. It's mostly useful for testing purposes.
type NoopTracer struct{}

// newNoopTracer returns a new noop tracer.
func newNoopTracer(ctx *Context, _ json.RawMessage) (Tracer, error) {
	return &NoopTracer{}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *NoopTracer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *NoopTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *NoopTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *NoopTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (t *NoopTracer) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// OnGasChange is called when gas is either consumed or refunded.
func (t *NoopTracer) OnGasChange(old, new uint64, reason vm.GasChangeReason) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *NoopTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *NoopTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (*NoopTracer) CaptureTxStart(env *vm.EVM, tx *types.Transaction) {}

func (*NoopTracer) CaptureTxEnd(receipt *types.Receipt, err error) {}

func (*NoopTracer) OnBalanceChange(a common.Address, prev, new *big.Int, reason state.BalanceChangeReason) {
}

func (*NoopTracer) OnNonceChange(a common.Address, prev, new uint64) {}

func (*NoopTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
}

func (*NoopTracer) OnStorageChange(a common.Address, k, prev, new common.Hash) {}

func (*NoopTracer) OnLog(log *types.Log) {}

func (*NoopTracer) OnNewAccount(a common.Address) {}

// GetResult returns an empty json object.
func (t *NoopTracer) GetResult() (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *NoopTracer) Stop(err error) {
}
