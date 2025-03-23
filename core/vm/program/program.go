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

// New creates a new Program
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
func (p *Program) doPush(val *uint256.Int) {
	if val == nil {
		val = new(uint256.Int)
	}
	valBytes := val.Bytes()
	if len(valBytes) == 0 {
		valBytes = append(valBytes, 0)
	}
	bLen := len(valBytes)
	p.add(byte(vm.PUSH1) - 1 + byte(bLen))
	p.Append(valBytes)
}

// Append appends the given data to the code.
func (p *Program) Append(data []byte) *Program {
	p.code = append(p.code, data...)
	return p
}

// Bytes returns the Program bytecode. OBS: This is not a copy.
func (p *Program) Bytes() []byte {
	return p.code
}

// SetBytes sets the Program bytecode. The combination of Bytes and SetBytes means
// that external callers can implement missing functionality:
//
//	...
//	prog.Push(1)
//	code := prog.Bytes()
//	manipulate(code)
//	prog.SetBytes(code)
func (p *Program) SetBytes(code []byte) {
	p.code = code
}

// Hex returns the Program bytecode as a hex string.
func (p *Program) Hex() string {
	return fmt.Sprintf("%02x", p.Bytes())
}

// Op appends the given opcode(s).
func (p *Program) Op(ops ...vm.OpCode) *Program {
	for _, op := range ops {
		p.add(byte(op))
	}
	return p
}

// Push creates a PUSHX instruction with the data provided. If zero is being pushed,
// PUSH0 will be avoided in favour of [PUSH1 0], to ensure backwards compatibility.
func (p *Program) Push(val any) *Program {
	switch v := val.(type) {
	case int:
		p.doPush(new(uint256.Int).SetUint64(uint64(v)))
	case uint64:
		p.doPush(new(uint256.Int).SetUint64(v))
	case uint32:
		p.doPush(new(uint256.Int).SetUint64(uint64(v)))
	case uint16:
		p.doPush(new(uint256.Int).SetUint64(uint64(v)))
	case *big.Int:
		p.doPush(uint256.MustFromBig(v))
	case *uint256.Int:
		p.doPush(v)
	case uint256.Int:
		p.doPush(&v)
	case []byte:
		p.doPush(new(uint256.Int).SetBytes(v))
	case byte:
		p.doPush(new(uint256.Int).SetUint64(uint64(v)))
	case interface{ Bytes() []byte }:
		// Here, we jump through some hoops in order to avoid depending on
		// go-ethereum types.Address and common.Hash, and instead use the
		// interface. This works on both values and pointers!
		p.doPush(new(uint256.Int).SetBytes(v.Bytes()))
	case nil:
		p.doPush(nil)
	default:
		panic(fmt.Sprintf("unsupported type %T", v))
	}
	return p
}

// Push0 implements PUSH0 (0x5f).
func (p *Program) Push0() *Program {
	return p.Op(vm.PUSH0)
}

// ExtcodeCopy performs an extcodecopy invocation.
func (p *Program) ExtcodeCopy(address, memOffset, codeOffset, length any) *Program {
	p.Push(length)
	p.Push(codeOffset)
	p.Push(memOffset)
	p.Push(address)
	return p.Op(vm.EXTCODECOPY)
}

// Call is a convenience function to make a call. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) Call(gas *uint256.Int, address, value, inOffset, inSize, outOffset, outSize any) *Program {
	if outOffset == outSize && inSize == outSize && inOffset == outSize && value == outSize {
		p.Push(outSize).Op(vm.DUP1, vm.DUP1, vm.DUP1, vm.DUP1)
	} else {
		p.Push(outSize).Push(outOffset).Push(inSize).Push(inOffset).Push(value)
	}
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.doPush(gas)
	}
	return p.Op(vm.CALL)
}

// DelegateCall is a convenience function to make a delegatecall. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) DelegateCall(gas *uint256.Int, address, inOffset, inSize, outOffset, outSize any) *Program {
	if outOffset == outSize && inSize == outSize && inOffset == outSize {
		p.Push(outSize).Op(vm.DUP1, vm.DUP1, vm.DUP1)
	} else {
		p.Push(outSize).Push(outOffset).Push(inSize).Push(inOffset)
	}
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.doPush(gas)
	}
	return p.Op(vm.DELEGATECALL)
}

// StaticCall is a convenience function to make a staticcall. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) StaticCall(gas *uint256.Int, address, inOffset, inSize, outOffset, outSize any) *Program {
	if outOffset == outSize && inSize == outSize && inOffset == outSize {
		p.Push(outSize).Op(vm.DUP1, vm.DUP1, vm.DUP1)
	} else {
		p.Push(outSize).Push(outOffset).Push(inSize).Push(inOffset)
	}
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.doPush(gas)
	}
	return p.Op(vm.STATICCALL)
}

// CallCode is a convenience function to make a callcode. If 'gas' is nil, the opcode GAS will
// be used to provide all gas.
func (p *Program) CallCode(gas *uint256.Int, address, value, inOffset, inSize, outOffset, outSize any) *Program {
	if outOffset == outSize && inSize == outSize && inOffset == outSize {
		p.Push(outSize).Op(vm.DUP1, vm.DUP1, vm.DUP1)
	} else {
		p.Push(outSize).Push(outOffset).Push(inSize).Push(inOffset)
	}
	p.Push(value)
	p.Push(address)
	if gas == nil {
		p.Op(vm.GAS)
	} else {
		p.doPush(gas)
	}
	return p.Op(vm.CALLCODE)
}

// Label returns the PC (of the next instruction).
func (p *Program) Label() uint64 {
	return uint64(len(p.code))
}

// Jumpdest adds a JUMPDEST op, and returns the PC of that instruction.
func (p *Program) Jumpdest() (*Program, uint64) {
	here := p.Label()
	p.Op(vm.JUMPDEST)
	return p, here
}

// Jump pushes the destination and adds a JUMP.
func (p *Program) Jump(loc any) *Program {
	p.Push(loc)
	p.Op(vm.JUMP)
	return p
}

// JumpIf implements JUMPI.
func (p *Program) JumpIf(loc any, condition any) *Program {
	p.Push(condition)
	p.Push(loc)
	p.Op(vm.JUMPI)
	return p
}

// Size returns the current size of the bytecode.
func (p *Program) Size() int {
	return len(p.code)
}

// InputAddressToStack stores the input (calldata) to memory as address (20 bytes).
func (p *Program) InputAddressToStack(inputOffset uint32) *Program {
	p.Push(inputOffset)
	p.Op(vm.CALLDATALOAD) // Loads [n -> n + 32] of input data to stack top
	mask, _ := big.NewInt(0).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
	p.Push(mask) // turn into address
	return p.Op(vm.AND)
}

// Mstore stores the provided data (into the memory area starting at memStart).
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

// MstoreSmall stores the provided data, which must be smaller than 32 bytes,
// into the memory area starting at memStart.
// The data will be LHS zero-added to align on 32 bytes.
// For example, providing data 0x1122, it will do a PUSH2:
// PUSH2 0x1122, resulting in
// stack: 0x0000000000000000000000000000000000000000000000000000000000001122
// followed by MSTORE(0,0)
// And thus, the resulting memory will be
// [ 0000000000000000000000000000000000000000000000000000000000001122 ]
func (p *Program) MstoreSmall(data []byte, memStart uint32) *Program {
	if len(data) > 32 {
		// For larger sizes, use Mstore instead.
		panic("only <=32 byte data size supported")
	}
	if len(data) == 0 {
		// Storing 0-length data smells of an error somewhere.
		panic("data is zero length")
	}
	// push the value
	p.Push(data)
	// push the memory index
	p.Push(memStart)
	p.Op(vm.MSTORE)
	return p
}

// MemToStorage copies the given memory area into SSTORE slots,
// It expects data to be aligned to 32 byte, and does not zero out
// remainders if some data is not
// I.e, if given a 1-byte area, it will still copy the full 32 bytes to storage.
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

// ReturnViaCodeCopy utilises CODECOPY to place the given data in the bytecode of
// p, loads into memory (offset 0) and returns the code.
// This is a typical "constructor".
// Note: since all indexing is calculated immediately, the preceding bytecode
// must not be expanded or shortened.
func (p *Program) ReturnViaCodeCopy(data []byte) *Program {
	p.Push(len(data))
	// For convenience, we'll use PUSH2 for the offset. Then we know we can always
	// fit, since code is limited to 0xc000
	p.Op(vm.PUSH2)
	offsetPos := p.Size()  // Need to update this position later on
	p.Append([]byte{0, 0}) // Offset of the code to be copied
	p.Push(0)              // Offset in memory (destination)
	p.Op(vm.CODECOPY)      // Copy from code[offset:offset+len] to memory[0:]
	p.Return(0, len(data)) // Return memory[0:len]
	offset := p.Size()
	p.Append(data) // And add the data

	// Now, go back and fix the offset
	p.code[offsetPos] = byte(offset >> 8)
	p.code[offsetPos+1] = byte(offset)
	return p
}

// Sstore stores the given byte array to the given slot.
// OBS! Does not verify that the value indeed fits into 32 bytes.
// If it does not, it will panic later on via doPush.
func (p *Program) Sstore(slot any, value any) *Program {
	p.Push(value)
	p.Push(slot)
	return p.Op(vm.SSTORE)
}

// Tstore stores the given byte array to the given t-slot.
// OBS! Does not verify that the value indeed fits into 32 bytes.
// If it does not, it will panic later on via doPush.
func (p *Program) Tstore(slot any, value any) *Program {
	p.Push(value)
	p.Push(slot)
	return p.Op(vm.TSTORE)
}

// Return implements RETURN
func (p *Program) Return(offset, len int) *Program {
	p.Push(len)
	p.Push(offset)
	return p.Op(vm.RETURN)
}

// ReturnData loads the given data into memory, and does a return with it
func (p *Program) ReturnData(data []byte) *Program {
	p.Mstore(data, 0)
	return p.Return(0, len(data))
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
	// - zero: in case of failure, OR
	// - address: in case of success
}

// Create2ThenCall calls create2 with the given initcode and salt, and then calls
// into the created contract (or calls into zero, if the creation failed).
func (p *Program) Create2ThenCall(code []byte, salt any) *Program {
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
