// Copyright 2014 The go-ethereum Authors
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

package vm

// bitvec is a bit vector which maps bytes in a program.
// An unset bit means the byte is an opcode, a set bit means
// it's data (i.e. argument of PUSHxx).
type bitvec []uint32

func (bv bitvec) isCode(pc uint64) bool {
	return (bv[pc/32] & (1 << (pc % 32))) == 0
}

// newCodeBitVec collects data locations in code.
func newCodeBitVec(code []byte) (bv bitvec) {
	bv = make(bitvec, len(code)/32+2)
	bv.codeBitvecInternal(code)
	return bv
}

// codeBitvecInternal is the internal implementation of codeBitmap.
// It exists for the purpose of being able to run benchmark tests
// without dynamic allocations affecting the results.
func (bv bitvec) codeBitvecInternal(code []byte) bitvec {
	var pc uint64
	for pc < uint64(len(code)) {
		op := code[pc]
		if int8(op) < int8(PUSH1) {
			pc++
			continue // continue if the OpCode is not PUSH1..32
		}
		numBytes := op - uint8(PUSH0) // number of data bytes pushed
		shift := uint8((pc + 1) % 32)
		i := (pc + 1) / 32

		switch numBytes {
		case 1:
			bv[i] |= 1 << shift
			pc += 2
		case 32:
			a := uint32(0xffffffff) << shift
			bv[i] |= a
			bv[i+1] = ^a
			pc += 33
		default:
			a := (uint64(1<<numBytes) - 1) << shift
			bv[i] |= uint32(a)
			bv[i+1] = uint32(a >> 32)
			pc += uint64(numBytes + 1)
		}
	}
	return bv
}
