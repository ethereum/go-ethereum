// Copyright 2017 The go-ethereum Authors
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

package main

import (
	"encoding/json"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
)

// JSONLogger merely contains a writer, and immediately outputs to that channel,
// instead of collecting logs
type JSONLogger struct {
	encoder *json.Encoder
}

// NewJSONLogger returns a new JSON logger
func NewJSONLogger(writer io.Writer) *JSONLogger {
	logger := &JSONLogger{
		encoder: json.NewEncoder(writer),
	}
	return logger
}

// CaptureState outputs state information on the logger
func (l *JSONLogger) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, contract *vm.Contract, depth int, err error) error {
	log := vm.StructLog{
		Pc:      pc,
		Op:      op,
		Gas:     gas + cost,
		GasCost: cost,
		Memory:  memory.Data(),
		Stack:   stack.Data(),
		Storage: nil,
		Depth:   depth,
		Err:     err}
	return l.encoder.Encode(log)
}

// CaptureEnd is triggered at end of execution
func (l *JSONLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration) error {
	type endLog struct {
		Output  string              `json:"output"`
		GasUsed math.HexOrDecimal64 `json:"gasUsed"`
		Time    time.Duration       `json:"time"`
	}

	log := endLog{common.Bytes2Hex(output), math.HexOrDecimal64(gasUsed), t}
	return l.encoder.Encode(log)

}
