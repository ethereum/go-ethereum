// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the goevmlab library. If not, see <http://www.gnu.org/licenses/>.

// package program is a utility to create EVM bytecode for testing, but _not_ for production. As such:
//
// - There are not package guarantees. We might iterate heavily on this package, and do backwards-incompatible changes without warning
// - There are no quality-guarantees. These utilities may produce evm-code that is non-functional. YMMV.
// - There are no stability-guarantees. The utility will `panic` if the inputs do not align / make sense.
package program

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// Program is a simple bytecode container. It can be used to construct
// simple EVM programs. Errors during construction of a Program typically
// cause panics: so avoid using these programs in production settings or on
// untrusted input.
// This package is mainly meant to aid in testing. This is not a production
// -level "compiler".
type Program struct {
	code []byte
}

func New() *Program {
	return &Program{
		code: make([]byte, 0),
	}
}

// add adds the op to the code.
func (p *Program) add(op byte) *Program {
	p.code = append(p.code, op)
	return p
}

// pushBig creates a PUSHX instruction and pushes the given val.
// - If the val is nil, it pushes zero
// - If the val is bigger than 32 bytes, it panics
func (p *Program) pushBig(val *big.Int) {
	if val == nil {
		val = big.NewInt(0)
	}
	valBytes := val.Bytes()
	if len(valBytes) == 0 {
		valBytes = append(valBytes, 0)
	}
	bLen := len(valBytes)
	if bLen > 32 {
		panic(fmt.Sprintf("Push value too large, %d bytes", bLen))
	}
	p.add(byte(vm.PUSH1) - 1 + byte(bLen))
	p.Append(valBytes)

}

// Append appends the given data to the code.
func (p *Program) Append(data []byte) *Program {
	p.code = append(p.code, data...)
	return p
}

// Op appends the given opcode
func (p *Program) Op(op vm.OpCode) *Program {
	return p.add(byte(op))
}

// Ops appends the given opcodes
func (p *Program) Ops(ops ...vm.OpCode) *Program {
	for _, op := range ops {
		p.add(byte(op))
	}
	return p
}

// Push creates a PUSHX instruction with the data provided
func (p *Program) Push(val any) *Program {
	switch v := val.(type) {
	case int:
		p.pushBig(new(big.Int).SetUint64(uint64(v)))
	case uint64:
		p.pushBig(new(big.Int).SetUint64(v))
	case uint32:
		p.pushBig(new(big.Int).SetUint64(uint64(v)))
	case *big.Int:
		p.pushBig(v)
	case *uint256.Int:
		p.pushBig(v.ToBig())
	case uint256.Int:
		p.pushBig(v.ToBig())
	case []byte:
		p.pushBig(new(big.Int).SetBytes(v))
	case byte:
		p.pushBig(new(big.Int).SetUint64(uint64(v)))
	case interface{ Bytes() []byte }:
		// Here, we jump through some hovm in order to avoid depending on
		// go-ethereum types.Address and common.Hash, and instead use the
		// interface. This works on both values and pointers!
		p.pushBig(new(big.Int).SetBytes(v.Bytes()))
	case nil:
		p.pushBig(nil)
	default:
		panic(fmt.Sprintf("unsupported type %v", v))
	}
	return p
}

// Push0 implements PUSH0 (0x5f)
func (p *Program) Push0() *Program {
	return p.Op(vm.PUSH0)
}

// Bytecode returns the Program bytecode
func (p *Program) Bytecode() []byte {
	return p.code
}

// Hex returns the Program bytecode as a hex string
func (p *Program) Hex() string {
	return fmt.Sprintf("%02x", p.Bytecode())
}

// ExtcodeCopy performsa an extcodecopy invocation
func (p *Program) ExtcodeCopy(address, memOffset, codeOffset, length any) {
	p.Push(length)
	p.Push(codeOffset)
	p.Push(memOffset)
	p.Push(address)
	p.Op(vm.EXTCODECOPY)
}

// Call is a convenience function to make a call. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) Call(gas *big.Int, address, value, inOffset, inSize, outOffset, outSize any) {
	p.Push(outSize)
	p.Push(outOffset)
	p.Push(inSize)
	p.Push(inOffset)
	p.Push(value)
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.pushBig(gas)
	}
	p.Op(vm.CALL)
}

// DelegateCall is a convenience function to make a delegatecall. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) DelegateCall(gas *big.Int, address, inOffset, inSize, outOffset, outSize any) {
	p.Push(outSize)
	p.Push(outOffset)
	p.Push(inSize)
	p.Push(inOffset)
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.pushBig(gas)
	}
	p.Op(vm.DELEGATECALL)
}

// StaticCall is a convenience function to make a staticcall. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) StaticCall(gas *big.Int, address, inOffset, inSize, outOffset, outSize any) {
	p.Push(outSize)
	p.Push(outOffset)
	p.Push(inSize)
	p.Push(inOffset)
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.pushBig(gas)
	}
	p.Op(vm.STATICCALL)
}

// StaticCall is a convenience function to make a callcode. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) CallCode(gas *big.Int, address, value, inOffset, inSize, outOffset, outSize any) {
	p.Push(outSize)
	p.Push(outOffset)
	p.Push(inSize)
	p.Push(inOffset)
	p.Push(value)
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.pushBig(gas)
	}
	p.Op(vm.CALLCODE)
}

// Label returns the PC (of the next instruction)
func (p *Program) Label() uint64 {
	return uint64(len(p.code))
}

// Jumpdest adds a JUMPDEST op, and returns the PC of that instruction
func (p *Program) Jumpdest() uint64 {
	here := p.Label()
	p.Op(vm.JUMPDEST)
	return here
}

// Jump pushes the destination and adds a JUMP
func (p *Program) Jump(loc any) {
	p.Push(loc)
	p.Op(vm.JUMP)
}

// Jump pushes the destination and adds a JUMP
func (p *Program) JumpIf(loc any, condition any) {
	p.Push(condition)
	p.Push(loc)
	p.Op(vm.JUMPI)
}

func (p *Program) Size() int {
	return len(p.code)
}

// InputToMemory stores the input (calldata) to memory as address (20 bytes).
func (p *Program) InputAddressToStack(inputOffset uint32) *Program {
	p.Push(inputOffset)
	p.Op(vm.CALLDATALOAD) // Loads [n -> n + 32] of input data to stack top
	mask, ok := big.NewInt(0).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
	if !ok {
		panic("whoa")
	}
	p.Push(mask) // turn into address
	return p.Op(vm.AND)
}

// MStore stores the provided data (into the memory area starting at memStart)
func (p *Program) Mstore(data []byte, memStart uint32) *Program {
	var idx = 0
	// We need to store it in chunks of 32 bytes
	for ; idx+32 <= len(data); idx += 32 {
		chunk := data[idx : idx+32]
		// push the value
		p.Push(chunk)
		// push the memory index
		p.Push(uint32(idx) + memStart)
		p.Op(vm.MSTORE)
	}
	// Remainders become stored using MSTORE8
	for ; idx < len(data); idx++ {
		b := data[idx]
		// push the byte
		p.Push(b)
		p.Push(uint32(idx) + memStart)
		p.Op(vm.MSTORE8)
	}
	return p
}

// MemToStorage copies the given memory area into SSTORE slots,
// It expects data to be aligned to 32 byte, and does not zero out
// remainders if some data is not
// I.e, if given a 1-byte area, it will still copy the full 32 bytes to storage
func (p *Program) MemToStorage(memStart, memSize, startSlot int) *Program {
	// We need to store it in chunks of 32 bytes
	for idx := memStart; idx < (memStart + memSize); idx += 32 {
		dataStart := idx
		// Mload the chunk
		p.Push(dataStart)
		p.Op(vm.MLOAD)
		// Value is now on stack,
		p.Push(startSlot)
		p.Op(vm.SSTORE)
		startSlot++
	}
	return p
}

// Sstore stores the given byte array to the given slot.
// OBS! Does not verify that the value indeed fits into 32 bytes
// If it does not, it will panic later on via pushBig
func (p *Program) Sstore(slot any, value any) *Program {
	p.Push(value)
	p.Push(slot)
	return p.Op(vm.SSTORE)
}

// Tstore stores the given byte array to the given t-slot.
// OBS! Does not verify that the value indeed fits into 32 bytes
// If it does not, it will panic later on via pushBig
func (p *Program) Tstore(slot any, value any) *Program {
	p.Push(value)
	p.Push(slot)
	return p.Op(vm.TSTORE)
}

func (p *Program) Return(offset, len uint32) *Program {
	p.Push(len)
	p.Push(offset)
	return p.Op(vm.RETURN)
}

// ReturnData loads the given data into memory, and does a return with it
func (p *Program) ReturnData(data []byte) *Program {
	p.Mstore(data, 0)
	return p.Return(0, uint32(len(data)))
}

// Create2 uses create2 to construct a contract with the given bytecode.
// This operation leaves either '0' or address on the stack.
func (p *Program) Create2(code []byte, salt any) *Program {
	var (
		value  = 0
		offset = 0
		size   = len(code)
	)
	// Load the code into mem
	p.Mstore(code, 0)
	// Create it
	return p.Push(salt).
		Push(size).
		Push(offset).
		Push(value).
		Op(vm.CREATE2)
	// On the stack now, is either
	// zero: in case of failure
	// address: in case of success
}

// Create2AndCall calls create2 with the given initcode and salt, and then calls
// into the created contract (or calls into zero, if the creation failed).
func (p *Program) Create2AndCall(code []byte, salt any) *Program {
	p.Create2(code, salt)
	// If there happen to be a zero on the stack, it doesn't matter, we're
	// not sending any value anyway
	p.Push(0).Push(0) // mem out
	p.Push(0).Push(0) // mem in
	p.Push(0)         // value
	p.Op(vm.DUP6)     // address
	p.Op(vm.GAS)
	p.Op(vm.CALL)
	p.Op(vm.POP)        // pop the retval
	return p.Op(vm.POP) // pop the address
}

// Selfdestruct pushes beneficiary and invokes selfdestruct.
func (p *Program) Selfdestruct(beneficiary any) *Program {
	p.Push(beneficiary)
	return p.Op(vm.SELFDESTRUCT)
}
