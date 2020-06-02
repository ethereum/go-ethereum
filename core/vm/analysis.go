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
type bitvec []byte

func (bits *bitvec) set(pos uint64) {
	(*bits)[pos/8] |= 0x80 >> (pos % 8)
}
func (bits *bitvec) set8(pos uint64) {
	(*bits)[pos/8] |= 0xFF >> (pos % 8)
	(*bits)[pos/8+1] |= ^(0xFF >> (pos % 8))
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
	for pc := uint64(0); pc < uint64(len(code)); {
		op := OpCode(code[pc])

		if op >= PUSH1 && op <= PUSH32 {
			numbits := op - PUSH1 + 1
			pc++
			for ; numbits >= 8; numbits -= 8 {
				bits.set8(pc) // 8
				pc += 8
			}
			for ; numbits > 0; numbits-- {
				bits.set(pc)
				pc++
			}
		} else {
			pc++
		}
	}
	return bits
}

type shadowmap []byte

func (shadow *shadowmap) IsCode(pos uint16) bool {
	return (*shadow)[pos]&0x80 == 0
}

func (shadow *shadowmap) set(pos uint64) {
	(*shadow)[pos] |= 0x80
}

func (shadow *shadowmap) set8(pos uint64) {
	(*shadow)[pos+7] |= 0x80
	(*shadow)[pos+6] |= 0x80
	(*shadow)[pos+5] |= 0x80
	(*shadow)[pos+4] |= 0x80
	(*shadow)[pos+3] |= 0x80
	(*shadow)[pos+2] |= 0x80
	(*shadow)[pos+1] |= 0x80
	(*shadow)[pos] |= 0x80
}

// isSameSubroutine returns true if 'loc' is within the subroutine started
// at 'subStart'.
func (shadow *shadowmap) isSameSubroutine(subStart, loc uint16) bool {
	if loc < subStart {
		return false
	}
	srSize := lebDecode((*shadow)[subStart:])
	return loc < subStart+srSize
}

// shadowMap creates a 'shadow map' of the code. The shadow-map is an implementation of
// the analysis to verify JUMP restructions for subroutines. It uses a backing
// array of the same size as the analyzed code.
// - The MSB in each byte is `0` if the opcode is 'code', `1` for 'data'.
// - If the op is a BEGINSUB, the size of the subroutines is LEB-encoded, starting
//   at 'loc', possibly covering a span of 3 bytes. This is encoded into the
//   7 least significant bits of the bytes in question.
func shadowMap(code []byte) shadowmap {
	shadow := make(shadowmap, len(code)+32)
	// TODO: Check if we need to make it longer than the code, in case it
	// ends on a PUSHX
	curStart := uint16(0) // start of current subroutine
	for pc := uint64(0); pc < uint64(len(code)); {
		op := OpCode(code[pc])
		if op >= PUSH1 && op <= PUSH32 {
			numbits := op - PUSH1 + 1
			pc++
			for ; numbits >= 8; numbits -= 8 {
				shadow.set8(pc) // 8
				pc += 8
			}
			for ; numbits > 0; numbits-- {
				shadow.set(pc)
				pc++
			}
		} else {
			if op == BEGINSUB {
				srSize := uint16(pc) - curStart
				// encode the size of the subroutine into the shadowmap
				lebEncode(srSize, shadow[curStart:])
				curStart = uint16(pc)
			}
			pc++
		}
	}
	// Also need to set the final size
	srSize := uint16((uint16(len(code)) - curStart))
	lebEncode(srSize, shadow[curStart:])
	return shadow
}

// lebEncode writes n into the out slice, as 7-bit LEB-encoded values.
// All writes are OR:ed into the destination buffer
// https://en.wikipedia.org/wiki/LEB128
// This encoding differs from the one on wikipedia: we use 6+1 bits for encoding,
// to allow the MSB bit to be used as a code/or/data-marker
func lebEncode(n uint16, out []byte) {
	var (
		b1 = byte(n & 0x3F)
		b2 = byte(n >> 6 & 0x3f)
		b3 = byte(n >> 12)
	)
	if b3 != 0 {
		out[2] |= byte(b3)
		out[1] |= b2 | 0x40
		out[0] |= b1 | 0x40
		return
	}
	if b2 != 0 {
		out[1] |= b2
		out[0] |= b1 | 0x40
		return
	}
	out[0] |= b1
	return
}

// lebDecode decodes the LEB-encoded int16.
func lebDecode(in []byte) uint16 {
	var res uint16
	b := in[0]
	res |= uint16(0x3f & b)
	if b&0x40 == 0 {
		return res
	}
	b = in[1]
	res |= (uint16(0x3f&b) << 6)
	if b&0x40 == 0 {
		return res
	}
	b = in[2]
	res |= (uint16(0x3f&b) << 12)
	return res
}
