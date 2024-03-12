// Copyright 2024 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

func init() {
	tracers.DefaultDirectory.Register("opcodeTracer", newOpcodeTracer, false)
}

//go:generate go run github.com/fjl/gencodec -type CallFrameEnter -field-override callFrameEnterMarshaling -out gen_callframe_enter.go

// CallFrameEnter is emitted every call frame entered.
type CallFrameEnter struct {
	Op    vm.OpCode      `json:"op"`
	From  common.Address `json:"from"`
	To    common.Address `json:"to"`
	Input []byte         `json:"input"`
	Gas   uint64         `json:"gas"`
	Value *big.Int       `json:"value"`
}

// overrides for gencodec
type callFrameEnterMarshaling struct {
	Input  hexutil.Bytes
	Gas    math.HexOrDecimal64
	Value  *hexutil.Big
	OpName string `json:"opName,omitempty"` // adds call to OpName() in MarshalJSON
}

// OpName formats the operand name in a human-readable format.
func (c *CallFrameEnter) OpName() string {
	return c.Op.String()
}

//go:generate go run github.com/fjl/gencodec -type CallFrameExit -field-override callFrameExitMarshaling -out gen_callframe_exit.go

// CallFrameExit is emitted every call frame exits.
type CallFrameExit struct {
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Output  []byte         `json:"output"`
	GasUsed uint64         `json:"gasUsed"`
	Error   *string        `json:"error,omitempty"`
}

// overrides for gencodec
type callFrameExitMarshaling struct {
	Output  hexutil.Bytes
	GasUsed math.HexOrDecimal64
}

type opcodeTracer struct {
	noopTracer
	traces    []json.RawMessage
	callStack []common.Address
	cfg       opcodeTracerConfig
	env       *vm.EVM
}

type opcodeTracerConfig struct {
	EnableMemory     bool // enable memory capture
	DisableStack     bool // disable stack capture
	EnableReturnData bool // enable return data capture
	EnableCallFrames bool // enable call frame enter capture
}

// newCallTracer returns a native go tracer which tracks
// call frames of a tx, and implements vm.EVMLogger.
func newOpcodeTracer(ctx *tracers.Context, config json.RawMessage) (tracers.Tracer, error) {
	var cfg opcodeTracerConfig
	if config != nil {
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, err
		}
	}
	// First callframe contains tx context info
	// and is populated on start and end.
	return &opcodeTracer{traces: make([]json.RawMessage, 0), callStack: make([]common.Address, 0), cfg: cfg}, nil
}

func (t *opcodeTracer) appendTrace(item interface{}) {
	data, err := json.Marshal(item)
	if err != nil {
		return
	}
	t.traces = append(t.traces, data)
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *opcodeTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	t.callStack = append(t.callStack, from, to)
	if t.cfg.EnableCallFrames {
		op := vm.CALL
		if create {
			op = vm.CREATE
		}
		t.appendTrace(CallFrameEnter{
			Op:    op,
			From:  from,
			To:    to,
			Gas:   gas,
			Value: value,
			Input: input,
		})
	}
}

func (t *opcodeTracer) CaptureFault(pc uint64, op vm.OpCode, gas uint64, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// TODO: Add rData to this interface as well
	t.CaptureState(pc, op, gas, cost, scope, nil, depth, err)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *opcodeTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	memory := scope.Memory
	stack := scope.Stack

	log := logger.StructLog{
		Pc:            pc,
		Op:            op,
		Gas:           gas,
		GasCost:       cost,
		MemorySize:    memory.Len(),
		Depth:         depth,
		RefundCounter: t.env.StateDB.GetRefund(),
		Err:           err,
	}
	if t.cfg.EnableMemory {
		log.Memory = memory.Data()
	}
	if !t.cfg.DisableStack {
		log.Stack = stack.Data()
	}
	if t.cfg.EnableReturnData {
		log.ReturnData = rData
	}
	t.appendTrace(log)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *opcodeTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	if t.cfg.EnableCallFrames {
		var errString *string
		if err != nil {
			errStr := err.Error()
			errString = &errStr
		}
		t.appendTrace(CallFrameExit{
			From:    t.callStack[len(t.callStack)-1],
			To:      t.callStack[len(t.callStack)-2],
			Output:  output,
			GasUsed: gasUsed,
			Error:   errString,
		})
	}
	t.callStack = t.callStack[:len(t.callStack)-2]
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *opcodeTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.callStack = append(t.callStack, to)
	if t.cfg.EnableCallFrames {
		t.appendTrace(CallFrameEnter{
			Op:    typ,
			From:  from,
			To:    to,
			Gas:   gas,
			Value: value,
			Input: input,
		})
	}
}

// CaptureExit is called when EVM exits a scope.
func (t *opcodeTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	if t.cfg.EnableCallFrames {
		var errString *string
		if err != nil {
			errStr := err.Error()
			errString = &errStr
		}
		t.appendTrace(CallFrameExit{
			From:    t.callStack[len(t.callStack)-1],
			To:      t.callStack[len(t.callStack)-2],
			Output:  output,
			GasUsed: gasUsed,
			Error:   errString,
		})
	}
	t.callStack = t.callStack[:len(t.callStack)-1]
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *opcodeTracer) GetResult() (json.RawMessage, error) {
	res, err := json.Marshal(t.traces)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), nil
}
