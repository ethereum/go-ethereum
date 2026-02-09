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

package t8ntool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// opcodeCounter is a simple tracer that counts how many times each opcode is executed.
type opcodeCounter struct {
	counts map[vm.OpCode]uint64
}

func newOpcodeCounter() *opcodeCounter {
	return &opcodeCounter{
		counts: make(map[vm.OpCode]uint64),
	}
}

func (c *opcodeCounter) hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			c.counts[vm.OpCode(op)]++
		},
	}
}

// results returns the opcode counts keyed by opcode name.
func (c *opcodeCounter) results() map[string]uint64 {
	out := make(map[string]uint64, len(c.counts))
	for op, count := range c.counts {
		out[op.String()] = count
	}
	return out
}

// composeHooks merges two sets of hooks into one. Both sets of hooks are called
// for each event.
func composeHooks(a, b *tracing.Hooks) *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart: func(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
			if a.OnTxStart != nil {
				a.OnTxStart(vm, tx, from)
			}
			if b.OnTxStart != nil {
				b.OnTxStart(vm, tx, from)
			}
		},
		OnTxEnd: func(receipt *types.Receipt, err error) {
			if a.OnTxEnd != nil {
				a.OnTxEnd(receipt, err)
			}
			if b.OnTxEnd != nil {
				b.OnTxEnd(receipt, err)
			}
		},
		OnEnter: func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
			if a.OnEnter != nil {
				a.OnEnter(depth, typ, from, to, input, gas, value)
			}
			if b.OnEnter != nil {
				b.OnEnter(depth, typ, from, to, input, gas, value)
			}
		},
		OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if a.OnExit != nil {
				a.OnExit(depth, output, gasUsed, err, reverted)
			}
			if b.OnExit != nil {
				b.OnExit(depth, output, gasUsed, err, reverted)
			}
		},
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			if a.OnOpcode != nil {
				a.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
			}
			if b.OnOpcode != nil {
				b.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
			}
		},
		OnFault: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if a.OnFault != nil {
				a.OnFault(pc, op, gas, cost, scope, depth, err)
			}
			if b.OnFault != nil {
				b.OnFault(pc, op, gas, cost, scope, depth, err)
			}
		},
		OnSystemCallStart: func() {
			if a.OnSystemCallStart != nil {
				a.OnSystemCallStart()
			}
			if b.OnSystemCallStart != nil {
				b.OnSystemCallStart()
			}
		},
		OnSystemCallEnd: func() {
			if a.OnSystemCallEnd != nil {
				a.OnSystemCallEnd()
			}
			if b.OnSystemCallEnd != nil {
				b.OnSystemCallEnd()
			}
		},
	}
}
