// Copyright 2026 The go-ethereum Authors
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

package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

// head encodes the gas and calldata prefix consumed by splitInput.
func head(gas uint64, input []byte) []byte {
	return append([]byte{byte(gas >> 16), byte(gas >> 8), byte(gas), byte(len(input))}, input...)
}

func TestFuzzers(t *testing.T) {
	// Count from 0 to 255.
	loop := []byte{
		byte(vm.PUSH1), 0x00, // counter = 0
		byte(vm.JUMPDEST),                  // pc 2, loop start
		byte(vm.PUSH1), 0x01, byte(vm.ADD), // counter += 1
		byte(vm.DUP1), byte(vm.PUSH1), 0xff, byte(vm.GT), // 255 > counter
		byte(vm.PUSH1), 0x02, byte(vm.JUMPI), // if so, jump to the loop start
	}
	fuzzCode(append(head(100_000, []byte{1, 2, 3, 4}), loop...))

	// 1024 items, then PUSH1 overflows the stack.
	fuzzDeepStack(append(head(10, nil), 0x04, 0x00, byte(vm.PUSH1), 0x01))

	// 5 items, then ADD costs 3 with 2 gas left.
	fuzzDeepStack(append(head(2, nil), 0x00, 0x05, byte(vm.ADD)))
}

// callCallee calls the callee with the given call opcode and returns its output.
func callCallee(op vm.OpCode) []byte {
	code := []byte{
		byte(vm.PUSH1), 0x20, // ret size
		byte(vm.PUSH1), 0x00, // ret offset
		byte(vm.PUSH1), 0x00, // args size
		byte(vm.PUSH1), 0x00, // args offset
	}
	if op == vm.CALL || op == vm.CALLCODE {
		code = append(code, byte(vm.PUSH1), 0x01) // value
	}
	code = append(code,
		byte(vm.PUSH1), 0x20, // callee address
		byte(vm.GAS), // all the gas left
		byte(op),
	)
	return append(code,
		byte(vm.RETURNDATASIZE), byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.RETURNDATACOPY), // mem[0:] = return data
		byte(vm.MSIZE), byte(vm.PUSH1), 0x00, byte(vm.RETURN), // return mem[0:msize]
	)
}

func FuzzCode(f *testing.F) {
	// 1 + 2
	f.Add(append(head(50_000, nil), byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x02, byte(vm.ADD)))

	// The callee through every call opcode.
	for _, op := range []vm.OpCode{vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL} {
		f.Add(append(head(200_000, nil), callCallee(op)...))
	}
	// keccak256 of the calldata word stored in memory.
	f.Add(append(head(100_000, []byte{0xde, 0xad}),
		byte(vm.PUSH0), byte(vm.CALLDATALOAD), // calldata[0:32]
		byte(vm.PUSH0), byte(vm.MSTORE), // mem[0:32] = that word
		byte(vm.MSIZE), byte(vm.PUSH0), byte(vm.KECCAK256), // keccak256(mem[0:msize])
	))
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzCode(data)
	})
}

func FuzzDeepStack(f *testing.F) {
	// 1023 items, DUP1 fills the stack and SWAP1 runs at the limit.
	f.Add(append(head(10, nil), 0x03, 0xff, byte(vm.DUP1), byte(vm.SWAP1)))

	// 1024 items, PUSH0 overflows from Shanghai and is undefined before.
	f.Add(append(head(100, nil), 0x04, 0x00, byte(vm.PUSH0)))
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzDeepStack(data)
	})
}
