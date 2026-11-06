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

package pbt

import (
	"encoding/binary"
	"math/bits"
)

// bitstr is a bit string packed MSB-first. Unused low bits of the last byte
// are always zero.
type bitstr struct {
	b []byte
	n int
}

// bitAt returns bit i (MSB-first) of key.
func bitAt(key []byte, i int) byte {
	return key[i>>3] >> (7 - i&7) & 1
}

func (p bitstr) bit(i int) byte {
	return bitAt(p.b, i)
}

// slice returns n bits of key starting at bit from.
func slice(key []byte, from, n int) bitstr {
	out := bitstr{b: make([]byte, (n+7)/8), n: n}
	if from&7 == 0 {
		copy(out.b, key[from>>3:])
	} else {
		shift := from & 7
		for i := range out.b {
			hi := key[from>>3+i] << shift
			var lo byte
			if from>>3+i+1 < len(key) {
				lo = key[from>>3+i+1] >> (8 - shift)
			}
			out.b[i] = hi | lo
		}
	}
	if n&7 != 0 && len(out.b) > 0 {
		out.b[len(out.b)-1] &= 0xff << (8 - n&7)
	}
	return out
}

func (p bitstr) head(m int) bitstr { return slice(p.b, 0, m) }
func (p bitstr) tail(m int) bitstr { return slice(p.b, m, p.n-m) }

// append returns p ‖ q.
func (p bitstr) append(q bitstr) bitstr {
	out := bitstr{b: make([]byte, (p.n+q.n+7)/8), n: p.n + q.n}
	copy(out.b, p.b)
	for i := range q.n {
		out.setBit(p.n+i, q.bit(i))
	}
	return out
}

// concat returns p ‖ bit ‖ q.
func (p bitstr) concat(bit byte, q bitstr) bitstr {
	out := bitstr{b: make([]byte, (p.n+1+q.n+7)/8), n: p.n + 1 + q.n}
	copy(out.b, p.b)
	out.setBit(p.n, bit)
	for i := range q.n {
		out.setBit(p.n+1+i, q.bit(i))
	}
	return out
}

func (p *bitstr) setBit(i int, bit byte) {
	if bit != 0 {
		p.b[i>>3] |= 1 << (7 - i&7)
	}
}

// matchKey returns how many leading bits of p agree with key from bit pos.
func (p bitstr) matchKey(key []byte, pos int) int {
	n := min(p.n, 8*len(key)-pos)
	for i := range n {
		if p.bit(i) != bitAt(key, pos+i) {
			return i
		}
	}
	return n
}

// commonPrefixLen returns the number of bits from bit from on which a and b
// agree.
func commonPrefixLen(a, b []byte, from int) int {
	end := 8 * min(len(a), len(b))
	i := from
	for ; i < end && i&7 != 0; i++ {
		if bitAt(a, i) != bitAt(b, i) {
			return i - from
		}
	}
	for ; i < end; i += 8 {
		if x := a[i>>3] ^ b[i>>3]; x != 0 {
			return min(i+bits.LeadingZeros8(x), end) - from
		}
	}
	return end - from
}

// appendBitPrefix appends encode_bit_prefix(p): a two-byte big-endian bit
// count followed by the packed bits.
func appendBitPrefix(dst []byte, p bitstr) []byte {
	return append(binary.BigEndian.AppendUint16(dst, uint16(p.n)), p.b...)
}

// decodeBitPrefix parses an encoded prefix from the front of blob, returning
// it with the number of bytes consumed.
func decodeBitPrefix(blob []byte) (bitstr, int, error) {
	if len(blob) < 2 {
		return bitstr{}, 0, errInvalidRecord
	}
	n := int(binary.BigEndian.Uint16(blob))
	size := (n + 7) / 8
	if n > maxPathBits || len(blob) < 2+size {
		return bitstr{}, 0, errInvalidRecord
	}
	p := bitstr{b: append([]byte{}, blob[2:2+size]...), n: n}
	if n&7 != 0 && p.b[size-1]<<(n&7) != 0 {
		return bitstr{}, 0, errInvalidRecord
	}
	return p, 2 + size, nil
}
