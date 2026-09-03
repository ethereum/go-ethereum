// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package native

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

// opcodeCounter is a simple tracer that counts how many times each opcode is executed.
type opcodeCounter struct {
	counts [256]uint64
}

// NewOpcodeCounter returns a new opcodeCounter tracer.
func NewOpcodeCounter() *tracers.Tracer {
	c := &opcodeCounter{}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
				c.counts[op]++
			},
		},
		GetResult: c.getResult,
		Stop:      func(err error) {},
	}
}

// getResult returns the opcode counts keyed by opcode name.
func (c *opcodeCounter) getResult() (json.RawMessage, error) {
	out := make(map[string]uint64)
	for op, count := range c.counts {
		if count != 0 {
			out[vm.OpCode(op).String()] = count
		}
	}
	return json.Marshal(out)
}
