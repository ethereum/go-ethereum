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

package logger

import (
	"encoding/json"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

//go:generate go run github.com/fjl/gencodec -type callFrame -field-override callFrameMarshaling -out gen_callframe.go

// overrides for gencodec
type callFrameMarshaling struct {
	Input hexutil.Bytes
	Gas   math.HexOrDecimal64
	Value *hexutil.Big
	Type  string `json:"type"` // adds call to Type() in MarshalJSON
}

// callFrame is emitted every call frame entered.
type callFrame struct {
	op    vm.OpCode
	From  common.Address `json:"from"`
	To    common.Address `json:"to"`
	Input []byte         `json:"input,omitempty"`
	Gas   uint64         `json:"gas"`
	Value *big.Int       `json:"value"`
}

// Type formats the call type in a human-readable format.
func (c *callFrame) Type() string {
	return c.op.String()
}

type jsonLogger struct {
	encoder *json.Encoder
	cfg     *Config
	env     *tracing.VMContext
	hooks   *tracing.Hooks
}

// NewJSONLogger creates a new EVM tracer that prints execution steps as JSON objects
// into the provided stream.
func NewJSONLogger(cfg *Config, writer io.Writer) *tracing.Hooks {
	l := &jsonLogger{encoder: json.NewEncoder(writer), cfg: cfg}
	if l.cfg == nil {
		l.cfg = &Config{}
	}
	l.hooks = &tracing.Hooks{
		OnTxStart:         l.OnTxStart,
		OnSystemCallStart: l.onSystemCallStart,
		OnExit:            l.OnEnd,
		OnOpcode:          l.OnOpcode,
		OnFault:           l.OnFault,
	}
	return l.hooks
}

// NewJSONLoggerWithCallFrames creates a new EVM tracer that prints execution steps as JSON objects
// into the provided stream. It also includes call frames in the output.
func NewJSONLoggerWithCallFrames(cfg *Config, writer io.Writer) *tracing.Hooks {
	l := &jsonLogger{encoder: json.NewEncoder(writer), cfg: cfg}
	if l.cfg == nil {
		l.cfg = &Config{}
	}
	l.hooks = &tracing.Hooks{
		OnTxStart:         l.OnTxStart,
		OnSystemCallStart: l.onSystemCallStart,
		OnEnter:           l.OnEnter,
		OnExit:            l.OnExit,
		OnOpcode:          l.OnOpcode,
		OnFault:           l.OnFault,
	}
	return l.hooks
}

func (l *jsonLogger) OnFault(pc uint64, op byte, gas uint64, cost uint64, scope tracing.OpContext, depth int, err error) {
	// TODO: Add rData to this interface as well
	l.OnOpcode(pc, op, gas, cost, scope, nil, depth, err)
}

func (l *jsonLogger) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	memory := scope.MemoryData()
	stack := scope.StackData()

	log := StructLog{
		Pc:            pc,
		Op:            vm.OpCode(op),
		Gas:           gas,
		GasCost:       cost,
		MemorySize:    len(memory),
		Depth:         depth,
		RefundCounter: l.env.StateDB.GetRefund(),
		Err:           err,
	}
	if l.cfg.EnableMemory {
		log.Memory = memory
	}
	if !l.cfg.DisableStack {
		log.Stack = stack
	}
	if l.cfg.EnableReturnData {
		log.ReturnData = rData
	}
	l.encoder.Encode(log)
}

func (l *jsonLogger) onSystemCallStart() {
	// Process no events while in system call.
	hooks := *l.hooks
	*l.hooks = tracing.Hooks{
		OnSystemCallEnd: func() {
			*l.hooks = hooks
		},
	}
}

// OnEnter is not enabled by default.
func (l *jsonLogger) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	frame := callFrame{
		op:    vm.OpCode(typ),
		From:  from,
		To:    to,
		Gas:   gas,
		Value: value,
	}
	if l.cfg.EnableMemory {
		frame.Input = input
	}
	l.encoder.Encode(frame)
}

func (l *jsonLogger) OnEnd(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if depth > 0 {
		return
	}
	l.OnExit(depth, output, gasUsed, err, false)
}

func (l *jsonLogger) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	type endLog struct {
		Output  string              `json:"output"`
		GasUsed math.HexOrDecimal64 `json:"gasUsed"`
		Err     string              `json:"error,omitempty"`
	}
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	l.encoder.Encode(endLog{common.Bytes2Hex(output), math.HexOrDecimal64(gasUsed), errMsg})
}

func (l *jsonLogger) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	l.env = env
}
