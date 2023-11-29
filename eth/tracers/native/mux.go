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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/directory"
)

func init() {
	directory.DefaultDirectory.Register("muxTracer", newMuxTracer, false)
}

// muxTracer is a go implementation of the Tracer interface which
// runs multiple tracers in one go.
type muxTracer struct {
	names   []string
	tracers []directory.Tracer
}

// newMuxTracer returns a new mux tracer.
func newMuxTracer(ctx *directory.Context, cfg json.RawMessage) (directory.Tracer, error) {
	var config map[string]json.RawMessage
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	objects := make([]directory.Tracer, 0, len(config))
	names := make([]string, 0, len(config))
	for k, v := range config {
		t, err := directory.DefaultDirectory.New(k, ctx, v)
		if err != nil {
			return nil, err
		}
		objects = append(objects, t)
		names = append(names, k)
	}

	return &muxTracer{names: names, tracers: objects}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *muxTracer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	for _, t := range t.tracers {
		t.CaptureStart(from, to, create, input, gas, value)
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *muxTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	for _, t := range t.tracers {
		t.CaptureEnd(output, gasUsed, err)
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *muxTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	for _, t := range t.tracers {
		t.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
	}
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *muxTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	for _, t := range t.tracers {
		t.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (t *muxTracer) CaptureKeccakPreimage(hash common.Hash, data []byte) {
	for _, t := range t.tracers {
		t.CaptureKeccakPreimage(hash, data)
	}
}

// CaptureGasConsumed is called when gas is consumed.
func (t *muxTracer) OnGasChange(old, new uint64, reason vm.GasChangeReason) {
	for _, t := range t.tracers {
		t.OnGasChange(old, new, reason)
	}
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *muxTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	for _, t := range t.tracers {
		t.CaptureEnter(typ, from, to, input, gas, value)
	}
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *muxTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	for _, t := range t.tracers {
		t.CaptureExit(output, gasUsed, err)
	}
}

func (t *muxTracer) CaptureTxStart(env *vm.EVM, tx *types.Transaction, from common.Address) {
	for _, t := range t.tracers {
		t.CaptureTxStart(env, tx, from)
	}
}

func (t *muxTracer) CaptureTxEnd(receipt *types.Receipt, err error) {
	for _, t := range t.tracers {
		t.CaptureTxEnd(receipt, err)
	}
}

func (t *muxTracer) OnBalanceChange(a common.Address, prev, new *big.Int, reason state.BalanceChangeReason) {
	for _, t := range t.tracers {
		t.OnBalanceChange(a, prev, new, reason)
	}
}

func (t *muxTracer) OnNonceChange(a common.Address, prev, new uint64) {
	for _, t := range t.tracers {
		t.OnNonceChange(a, prev, new)
	}
}

func (t *muxTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	for _, t := range t.tracers {
		t.OnCodeChange(a, prevCodeHash, prev, codeHash, code)
	}
}

func (t *muxTracer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	for _, t := range t.tracers {
		t.OnStorageChange(a, k, prev, new)
	}
}

func (t *muxTracer) OnLog(log *types.Log) {
	for _, t := range t.tracers {
		t.OnLog(log)
	}
}

func (t *muxTracer) OnNewAccount(a common.Address) {
	for _, t := range t.tracers {
		t.OnNewAccount(a)
	}
}

// GetResult returns an empty json object.
func (t *muxTracer) GetResult() (json.RawMessage, error) {
	resObject := make(map[string]json.RawMessage)
	for i, tt := range t.tracers {
		r, err := tt.GetResult()
		if err != nil {
			return nil, err
		}
		resObject[t.names[i]] = r
	}
	res, err := json.Marshal(resObject)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *muxTracer) Stop(err error) {
	for _, t := range t.tracers {
		t.Stop(err)
	}
}
