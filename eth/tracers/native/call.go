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

package native

import (
	"encoding/json"
	"errors"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

//go:generate go run github.com/fjl/gencodec -type callFrame -field-override callFrameMarshaling -out gen_callframe_json.go

func init() {
	register("callTracer", newCallTracer)
}

type callFrame struct {
	Type    vm.OpCode      `json:"-"`
	From    common.Address `json:"from"`
	Gas     uint64         `json:"gas"`
	GasUsed uint64         `json:"gasUsed"`
	To      common.Address `json:"to,omitempty" rlp:"optional"`
	Input   []byte         `json:"input" rlp:"optional"`
	Output  []byte         `json:"output,omitempty" rlp:"optional"`
	Error   string         `json:"error,omitempty" rlp:"optional"`
	Calls   []callFrame    `json:"calls,omitempty" rlp:"optional"`
	// Placed at end on purpose. The RLP will be decoded to 0 instead of
	// nil if there are non-empty elements after in the struct.
	Value *big.Int `json:"value,omitempty" rlp:"optional"`
}

func (f callFrame) TypeString() string {
	return f.Type.String()
}

type callFrameMarshaling struct {
	TypeString string `json:"type"`
	Gas        hexutil.Uint64
	GasUsed    hexutil.Uint64
	Value      *hexutil.Big
	Input      hexutil.Bytes
	Output     hexutil.Bytes
}

type callTracer struct {
	env       *vm.EVM
	callstack []callFrame
	config    callTracerConfig
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

type callTracerConfig struct {
	OnlyTopCall bool `json:"onlyTopCall"` // If true, call tracer won't collect any subcalls
}

// newCallTracer returns a native go tracer which tracks
// call frames of a tx, and implements vm.EVMLogger.
func newCallTracer(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	var config callTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	// First callframe contains tx context info
	// and is populated on start and end.
	return &callTracer{callstack: make([]callFrame, 1), config: config}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *callTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	t.callstack[0] = callFrame{
		Type:  vm.CALL,
		From:  from,
		To:    to,
		Input: common.CopyBytes(input),
		Gas:   gas,
		Value: value,
	}
	if create {
		t.callstack[0].Type = vm.CREATE
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *callTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
	t.callstack[0].GasUsed = gasUsed
	output = common.CopyBytes(output)
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
func (t *callTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *callTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *callTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.config.OnlyTopCall {
		return
	}
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

	call := callFrame{
		Type:  typ,
		From:  from,
		To:    to,
		Input: common.CopyBytes(input),
		Gas:   gas,
		Value: value,
	}
	t.callstack = append(t.callstack, call)
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *callTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	if t.config.OnlyTopCall {
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
		call.Output = common.CopyBytes(output)
	} else {
		call.Error = err.Error()
		if call.Type == vm.CREATE || call.Type == vm.CREATE2 {
			call.To = common.Address{}
		}
	}
	t.callstack[size-1].Calls = append(t.callstack[size-1].Calls, call)
}

func (*callTracer) CaptureTxStart(gasLimit uint64) {}

func (*callTracer) CaptureTxEnd(restGas uint64) {}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *callTracer) GetResult() (json.RawMessage, error) {
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
func (t *callTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}
