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

import (
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru"
)

const (
	set2BitsMask = uint16(0b1100_0000_0000_0000)
	set3BitsMask = uint16(0b1110_0000_0000_0000)
	set4BitsMask = uint16(0b1111_0000_0000_0000)
	set5BitsMask = uint16(0b1111_1000_0000_0000)
	set6BitsMask = uint16(0b1111_1100_0000_0000)
	set7BitsMask = uint16(0b1111_1110_0000_0000)
)

const codeBitmapCacheSize = 10000

var codeBitmapCache, _ = lru.NewARC(codeBitmapCacheSize)

// bitvec is a bit vector which maps bytes in a program.
// An unset bit means the byte is an opcode, a set bit means
// it's data (i.e. argument of PUSHxx).
type bitvec []byte

var lookup = [8]byte{
	0x80, 0x40, 0x20, 0x10, 0x8, 0x4, 0x2, 0x1,
}

func (bits bitvec) set1(pos uint64) {
	bits[pos/8] |= lookup[pos%8]
}

func (bits bitvec) setN(flag uint16, pos uint64) {
	a := flag >> (pos % 8)
	bits[pos/8] |= byte(a >> 8)
	if b := byte(a); b != 0 {
		//	If the bit-setting affects the neighbouring byte, we can assign - no need to OR it,
		//	since it's the first write to that byte
		bits[pos/8+1] = b
	}
}

func (bits bitvec) set8(pos uint64) {
	a := byte(0xFF >> (pos % 8))
	bits[pos/8] |= a
	bits[pos/8+1] = ^a
}

func (bits bitvec) set16(pos uint64) {
	a := byte(0xFF >> (pos % 8))
	bits[pos/8] |= a
	bits[pos/8+1] = 0xFF
	bits[pos/8+2] = ^a
}

// codeSegment checks if the position is in a code segment.
func (bits *bitvec) codeSegment(pos uint64) bool {
	return ((*bits)[pos/8] & (0x80 >> (pos % 8))) == 0
}

// codeBitmap collects data locations in code.
func codeBitmap(code []byte) bitvec {
	// The bitmap is 4 bytes longer than necessary, in case the code
	// ends with a PUSH32, the algorithm will push zeroes onto the
	// bitvector outside the bounds of the actual code.

	bits := make(bitvec, len(code)/8+1+4)
	return codeBitmapInternal(code, bits)
}

// codeBitmapInternal is the internal implementation of codeBitmap.
// It exists for the purpose of being able to run benchmark tests
// without dynamic allocations affecting the results.
func codeBitmapInternal(code, bits bitvec) bitvec {
	for pc := uint64(0); pc < uint64(len(code)); {
		op := OpCode(code[pc])
		pc++
		if op < PUSH1 || op > PUSH32 {
			continue
		}
		numbits := op - PUSH1 + 1
		if numbits >= 8 {
			for ; numbits >= 16; numbits -= 16 {
				bits.set16(pc)
				pc += 16
			}
			for ; numbits >= 8; numbits -= 8 {
				bits.set8(pc)
				pc += 8
			}
		}
		switch numbits {
		case 1:
			bits.set1(pc)
			pc += 1
		case 2:
			bits.setN(set2BitsMask, pc)
			pc += 2
		case 3:
			bits.setN(set3BitsMask, pc)
			pc += 3
		case 4:
			bits.setN(set4BitsMask, pc)
			pc += 4
		case 5:
			bits.setN(set5BitsMask, pc)
			pc += 5
		case 6:
			bits.setN(set6BitsMask, pc)
			pc += 6
		case 7:
			bits.setN(set7BitsMask, pc)
			pc += 7
		}
	}
	return bits
}

func codeBitmapWithCache(code []byte, codeHash common.Hash) bitvec {
	if (codeHash == emptyCodeHash || codeHash == common.Hash{}) {
		return nil
	}

	if val, ok := codeBitmapCache.Get(codeHash); ok {
		return val.(bitvec)
	} else {
		val := codeBitmap(code)
		codeBitmapCache.Add(codeHash, val)
		return val
	}
}
