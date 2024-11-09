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

// Package asm provides support for dealing with EVM assembly instructions (e.g., disassembling them).
package asm

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Iterator for disassembled EVM instructions
type instructionIterator struct {
	code       []byte
	pc         uint64
	arg        []byte
	op         vm.OpCode
	error      error
	started    bool
	eofEnabled bool
}

// NewInstructionIterator creates a new instruction iterator.
func NewInstructionIterator(code []byte) *instructionIterator {
	it := new(instructionIterator)
	it.code = code
	return it
}

// NewEOFInstructionIterator creates a new instruction iterator for EOF-code.
func NewEOFInstructionIterator(code []byte) *instructionIterator {
	it := NewInstructionIterator(code)
	it.eofEnabled = true
	return it
}

// Next returns true if there is a next instruction and moves on.
func (it *instructionIterator) Next() bool {
	if it.error != nil || uint64(len(it.code)) <= it.pc {
		// We previously reached an error or the end.
		return false
	}

	if it.started {
		// Since the iteration has been already started we move to the next instruction.
		if it.arg != nil {
			it.pc += uint64(len(it.arg))
		}
		it.pc++
	} else {
		// We start the iteration from the first instruction.
		it.started = true
	}

	if uint64(len(it.code)) <= it.pc {
		// We reached the end.
		return false
	}
	it.op = vm.OpCode(it.code[it.pc])
	var a int
	if !it.eofEnabled { // Legacy code
		if it.op.IsPush() {
			a = int(it.op) - int(vm.PUSH0)
		}
	} else { // EOF code
		if it.op == vm.RJUMPV {
			// RJUMPV is unique as it has a variable sized operand. The total size is
			// determined by the count byte which immediately follows RJUMPV.
			maxIndex := int(it.code[it.pc+1])
			a = (maxIndex+1)*2 + 1
		} else {
			a = vm.Immediates(it.op)
		}
	}
	if a > 0 {
		u := it.pc + 1 + uint64(a)
		if uint64(len(it.code)) <= it.pc || uint64(len(it.code)) < u {
			it.error = fmt.Errorf("incomplete instruction at %v", it.pc)
			return false
		}
		it.arg = it.code[it.pc+1 : u]
	} else {
		it.arg = nil
	}
	return true
}

// Error returns any error that may have been encountered.
func (it *instructionIterator) Error() error {
	return it.error
}

// PC returns the PC of the current instruction.
func (it *instructionIterator) PC() uint64 {
	return it.pc
}

// Op returns the opcode of the current instruction.
func (it *instructionIterator) Op() vm.OpCode {
	return it.op
}

// Arg returns the argument of the current instruction.
func (it *instructionIterator) Arg() []byte {
	return it.arg
}

// PrintDisassembled pretty-print all disassembled EVM instructions to stdout.
func PrintDisassembled(code string) error {
	script, err := hex.DecodeString(code)
	if err != nil {
		return err
	}
	it := NewInstructionIterator(script)
	for it.Next() {
		if it.Arg() != nil && 0 < len(it.Arg()) {
			fmt.Printf("%05x: %v %#x\n", it.PC(), it.Op(), it.Arg())
		} else {
			fmt.Printf("%05x: %v\n", it.PC(), it.Op())
		}
	}
	return it.Error()
}

// Disassemble returns all disassembled EVM instructions in human-readable format.
func Disassemble(script []byte) ([]string, error) {
	instrs := make([]string, 0)

	it := NewInstructionIterator(script)
	for it.Next() {
		if it.Arg() != nil && 0 < len(it.Arg()) {
			instrs = append(instrs, fmt.Sprintf("%05x: %v %#x\n", it.PC(), it.Op(), it.Arg()))
		} else {
			instrs = append(instrs, fmt.Sprintf("%05x: %v\n", it.PC(), it.Op()))
		}
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	return instrs, nil
}
