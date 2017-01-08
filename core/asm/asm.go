// Copyright 2014, 2017 The go-ethereum Authors
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

// Provides support for dealing with EVM assembly instructions (e.g., disassembling them).
package asm

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Apply function to each disassembled EVM instruction.
func ForEachDisassembledInstruction(script []byte, fun func(uint64, vm.OpCode, []byte)) error {
	for pc := uint64(0); pc < uint64(len(script)); pc++ {
		op := vm.OpCode(script[pc])
		switch op {
		case vm.PUSH1, vm.PUSH2, vm.PUSH3, vm.PUSH4, vm.PUSH5, vm.PUSH6, vm.PUSH7, vm.PUSH8, vm.PUSH9, vm.PUSH10, vm.PUSH11, vm.PUSH12, vm.PUSH13, vm.PUSH14, vm.PUSH15, vm.PUSH16, vm.PUSH17, vm.PUSH18, vm.PUSH19, vm.PUSH20, vm.PUSH21, vm.PUSH22, vm.PUSH23, vm.PUSH24, vm.PUSH25, vm.PUSH26, vm.PUSH27, vm.PUSH28, vm.PUSH29, vm.PUSH30, vm.PUSH31, vm.PUSH32:
			a := uint64(op) - uint64(vm.PUSH1) + 1
			u := pc + 1 + a
			if uint64(len(script)) <= pc || uint64(len(script)) < u {
				return fmt.Errorf("incomplete push instruction at %v", pc)
			}
			fun(pc, op, script[pc+1:u])
			pc += a
		default:
			fun(pc, op, make([]byte, 0))
		}
	}
	return nil
}

// Pretty-print all disassembled EVM instructions to stdout.
func PrettyPrintDisassembledInstructions(code string) error {
	script, err := hex.DecodeString(code)
	if err != nil {
		return err
	}

	return ForEachDisassembledInstruction(script, func(pc uint64, op vm.OpCode, args []byte) {
		if args != nil && 0 < len(args) {
			fmt.Printf("%06v: %v 0x%x\n", pc, op, args)
		} else {
			fmt.Printf("%06v: %v\n", pc, op)
		}
	})
}

// Return all disassembled EVM instructions in human-readable format.
func PrettyPrintedDisassembledInstructions(script []byte) ([]string, error) {
	instrs := make([]string, 0)
	err := ForEachDisassembledInstruction(script, func(pc uint64, op vm.OpCode, args []byte) {
		if args != nil && 0 < len(args) {
			instrs = append(instrs, fmt.Sprintf("%06v: %v 0x%x\n", pc, op, args))
		} else {
			instrs = append(instrs, fmt.Sprintf("%06v: %v\n", pc, op))
		}
	})

	if err != nil {
		return nil, err
	}
	return instrs, nil
}
