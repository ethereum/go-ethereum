// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
package bintrie

// BitArray represents a trie path: the most significant `len` bits of a key,
// packed big-endian and MSB-first. Bit i (0 = most significant) lives at
// bytes[i/8] in mask 1<<(7-i%8). All bits at positions >= len are kept zero so
// that two paths are byte-equal iff they are logically equal.
//
// This mirrors the on-disk key layout, so path manipulation is plain slicing
// and copying: no shifting or endianness conversion is required. The maximum
// length is 248 bits (a 31-byte trie stem), and is a uint8 so the spare bits in
// the final byte are always available.
type BitArray struct {
	len   uint8
	bytes [32]byte
}

// NewBitArray creates a bit array of the given length whose bits are the `length`
// least-significant bits of val, read most-significant-first. Used by tests to
// build expected paths; the value is interpreted as a number, not raw bytes.
func NewBitArray(length uint8, val uint64) BitArray {
	var b BitArray
	b.len = length
	for p := uint8(0); p < length; p++ {
		if (val>>(length-1-p))&1 == 1 {
			b.bytes[p/8] |= 1 << (7 - p%8)
		}
	}
	return b
}

// Len returns the number of used bits.
func (b *BitArray) Len() uint8 {
	return b.len
}

// Bytes returns the packed big-endian, MSB-first representation. Bits beyond
// len are zero.
func (b *BitArray) Bytes() [32]byte {
	return b.bytes
}

// AppendBit sets the bit array to x with a single bit appended, and returns the
// receiver. Safe when b and x alias the same value.
func (b *BitArray) AppendBit(x *BitArray, bit uint8) *BitArray {
	*b = *x
	if bit&1 == 1 {
		// Position b.len is guaranteed zero by the all-bits-beyond-len-are-zero
		// invariant, so a 1 only needs setting; a 0 is already in place.
		b.bytes[b.len/8] |= 1 << (7 - b.len%8)
	}
	b.len++
	return b
}

// MSBs sets the bit array to the most significant n bits of x and returns the
// receiver. If n >= x.len it is an exact copy of x. Think of it as x[:n].
func (b *BitArray) MSBs(x *BitArray, n uint8) *BitArray {
	*b = *x
	if n < b.len {
		b.len = n
		b.maskTail()
	}
	return b
}

// Equal reports whether two bit arrays hold the same path.
func (b *BitArray) Equal(x *BitArray) bool {
	return b.len == x.len && b.bytes == x.bytes
}

// SetBytes sets the bit array to the most significant `length` bits of data,
// interpreted as big-endian bytes, and returns the receiver. At most 32 bytes
// of data are read; bits beyond length are zeroed.
func (b *BitArray) SetBytes(length uint8, data []byte) *BitArray {
	b.bytes = [32]byte{}
	copy(b.bytes[:], data)
	b.len = length
	b.maskTail()
	return b
}

// SetBit sets the bit array to a single bit and returns the receiver.
func (b *BitArray) SetBit(bit uint8) *BitArray {
	b.bytes = [32]byte{}
	b.len = 1
	if bit&1 == 1 {
		b.bytes[0] = 0x80
	}
	return b
}

// Copy returns a value copy of the bit array.
func (b *BitArray) Copy() BitArray {
	return *b
}

// Set sets the bit array to the same value as x and returns the receiver.
func (b *BitArray) Set(x *BitArray) *BitArray {
	*b = *x
	return b
}

// KeyBytes returns the path-to-DB-key encoding: the active bytes (the
// left-aligned MSB-first prefix) followed by a single trailing byte holding the
// bit-length. The trailing length disambiguates paths whose active bytes
// coincide (e.g. 1-bit "1" packs to [0x80, 0x01] and 8-bit "10000000" to
// [0x80, 0x08]). The empty path encodes as no bytes.
func (b *BitArray) KeyBytes() []byte {
	if b.len == 0 {
		return nil
	}
	bc := (int(b.len) + 7) / 8
	res := make([]byte, bc+1)
	copy(res[:bc], b.bytes[:bc])
	res[bc] = b.len
	return res
}

// PutKeyBytes writes the key encoding (active bytes followed by length byte)
// into dst and returns the populated sub-slice. The empty path returns dst[:0]
// without touching dst. For non-empty paths dst must have len >= 33 (32 packed
// bytes for 248 bits + 1 length byte).
func (b *BitArray) PutKeyBytes(dst []byte) []byte {
	if b.len == 0 {
		return dst[:0]
	}
	bc := (int(b.len) + 7) / 8
	_ = dst[bc] // bounds check hint
	copy(dst[:bc], b.bytes[:bc])
	dst[bc] = b.len
	return dst[:bc+1]
}

// maskTail zeroes every bit at a position >= len, preserving the invariant that
// equal paths are byte-equal.
func (b *BitArray) maskTail() {
	full := int(b.len / 8)
	if rem := b.len % 8; rem != 0 {
		b.bytes[full] &= byte(0xFF) << (8 - rem)
		full++
	}
	for i := full; i < len(b.bytes); i++ {
		b.bytes[i] = 0
	}
}
