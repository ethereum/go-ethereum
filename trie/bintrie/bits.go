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

package bintrie

import (
	"encoding/binary"
	"math/bits"
)

// bitstr is a bit string packed MSB-first into bytes. Unused low bits of the
// last byte are always zero, so byte-wise equality equals bit-wise equality.
type bitstr struct {
	b []byte // packed bits, MSB first
	n int    // length in bits; len(b) == (n+7)/8
}

// bitAt returns bit i (MSB-first) of the packed byte string key.
func bitAt(key []byte, i int) byte {
	return key[i>>3] >> (7 - i&7) & 1
}

// bit returns bit i of the bit string.
func (p bitstr) bit(i int) byte {
	return bitAt(p.b, i)
}

func (p bitstr) equal(q bitstr) bool {
	if p.n != q.n {
		return false
	}
	for i := range p.b {
		if p.b[i] != q.b[i] {
			return false
		}
	}
	return true
}

// slice extracts n bits of key starting at bit position from, as a bitstr.
func slice(key []byte, from, n int) bitstr {
	out := bitstr{b: make([]byte, (n+7)/8), n: n}
	// Fast path: byte-aligned source range.
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
	out.maskTail()
	return out
}

// maskTail zeroes the unused low bits of the final byte, restoring the
// canonical-form invariant after byte-granular copies.
func (p *bitstr) maskTail() {
	if p.n&7 != 0 && len(p.b) > 0 {
		p.b[len(p.b)-1] &= 0xff << (8 - p.n&7)
	}
}

// concat returns p ‖ bit ‖ q, the prefix re-concatenation used when a branch
// collapses into its parent position after a deletion.
func (p bitstr) concat(bit byte, q bitstr) bitstr {
	n := p.n + 1 + q.n
	out := bitstr{b: make([]byte, (n+7)/8), n: n}
	copy(out.b, p.b)
	out.setBit(p.n, bit)
	for i := 0; i < q.n; i++ { // TODO: byte-blit once profiles demand it
		out.setBit(p.n+1+i, q.bit(i))
	}
	return out
}

func (p *bitstr) setBit(i int, bit byte) {
	if bit != 0 {
		p.b[i>>3] |= 1 << (7 - i&7)
	}
}

// matchKey returns how many bits of p agree with key starting at key bit
// position pos. The result is at most min(p.n, 8*len(key)-pos).
func (p bitstr) matchKey(key []byte, pos int) int {
	max := p.n
	if avail := 8*len(key) - pos; avail < max {
		max = avail
	}
	for i := 0; i < max; i++ {
		if p.bit(i) != bitAt(key, pos+i) {
			return i
		}
	}
	return max
}

// commonPrefixLen returns the number of bits, starting at bit position from,
// on which a and b agree. Comparison stops at the end of the shorter input.
func commonPrefixLen(a, b []byte, from int) int {
	max := 8 * min(len(a), len(b))
	i := from
	// Align to a byte boundary bit-by-bit, then compare whole bytes.
	for ; i < max && i&7 != 0; i++ {
		if bitAt(a, i) != bitAt(b, i) {
			return i - from
		}
	}
	for i < max {
		x := a[i>>3] ^ b[i>>3]
		if x != 0 {
			i += bits.LeadingZeros8(x)
			if i > max {
				i = max
			}
			return i - from
		}
		i += 8
	}
	return max - from
}

// encodeBitPrefix packs a branch prefix for hashing and storage: a two-byte
// big-endian bit count followed by the MSB-first packed bits, zero-padded to
// a byte boundary (EIP-8297 "Node merkelization").
func encodeBitPrefix(p bitstr) []byte {
	out := make([]byte, 2+len(p.b))
	binary.BigEndian.PutUint16(out, uint16(p.n))
	copy(out[2:], p.b)
	return out
}

// appendBitPrefix is encodeBitPrefix writing into dst, avoiding allocation on
// the hash hot path.
func appendBitPrefix(dst []byte, p bitstr) []byte {
	dst = append(dst, byte(p.n>>8), byte(p.n))
	return append(dst, p.b...)
}

// decodeBitPrefix parses an encodeBitPrefix payload from the front of blob,
// returning the prefix and the number of bytes consumed.
func decodeBitPrefix(blob []byte) (bitstr, int, error) {
	if len(blob) < 2 {
		return bitstr{}, 0, errInvalidSerializedLength
	}
	n := int(binary.BigEndian.Uint16(blob))
	nbytes := (n + 7) / 8
	if len(blob) < 2+nbytes {
		return bitstr{}, 0, errInvalidSerializedLength
	}
	p := bitstr{b: append([]byte{}, blob[2:2+nbytes]...), n: n}
	// Reject non-canonical padding: two different-length prefixes must never
	// decode from the same bytes.
	var check = p
	check.b = append([]byte{}, p.b...)
	check.maskTail()
	if !check.equal(p) {
		return bitstr{}, 0, errNonCanonicalPrefix
	}
	return p, 2 + nbytes, nil
}

// encodePath encodes a node position (the bits consumed above the node) as a
// database path key: a two-byte big-endian bit count followed by the packed
// bits. The root position encodes as the empty path, matching the database
// convention of storing the root record at path nil.
func encodePath(key []byte, n int) []byte {
	if n == 0 {
		return []byte{}
	}
	p := slice(key, 0, n)
	return encodeBitPrefix(p)
}

// bytesToBits expands packed bytes into one bit per byte (test bridge to the
// EELS reference representation).
func bytesToBits(data []byte) []byte {
	out := make([]byte, 8*len(data))
	for i := range out {
		out[i] = bitAt(data, i)
	}
	return out
}

// bitsToBitstr packs a one-bit-per-byte slice into a bitstr (test bridge).
func bitsToBitstr(bits []byte) bitstr {
	p := bitstr{b: make([]byte, (len(bits)+7)/8), n: len(bits)}
	for i, bit := range bits {
		p.setBit(i, bit)
	}
	return p
}
