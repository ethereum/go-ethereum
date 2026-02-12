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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

// OpcodeCounter is a simple tracer that counts how many times each opcode is executed.
type OpcodeCounter struct {
	counts [256]uint64
}

func (c *OpcodeCounter) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			c.counts[op]++
		},
	}
}

// Results returns the opcode counts keyed by opcode name.
func (c *OpcodeCounter) Results() map[string]uint64 {
	out := make(map[string]uint64, len(c.counts))
	for op, count := range c.counts {
		out[vm.OpCode(op).String()] = count
	}
	return out
}
