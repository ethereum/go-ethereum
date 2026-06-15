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

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
)

const (
	maxUint64 = uint64(math.MaxUint64) // 0xFFFFFFFFFFFFFFFF
	maxUint8  = uint8(math.MaxUint8)
)

var emptyBitArray = new(BitArray)

// BitArray represents a bit array with length representing the number of used bits.
// It uses a little endian representation to do bitwise operations of the words efficiently.
// For example, if len is 10, it means that the 2^9, 2^8, ..., 2^0 bits are used.
// The max length is 255 bits (uint8), because our use case only need up to 248 bits for a given trie key.
// Although words can be used to represent 256 bits, we don't want to add an additional byte for the length.
type BitArray struct {
	len   uint8     // number of used bits
	words [4]uint64 // little endian (i.e. words[0] is the least significant)
}

// NewBitArray creates a new bit array with the given length and value.
func NewBitArray(length uint8, val uint64) BitArray {
	var b BitArray
	b.SetUint64(length, val)
	return b
}

func (b *BitArray) Len() uint8 {
	return b.len
}

// Bytes returns the bytes representation of the bit array in big endian format
func (b *BitArray) Bytes() [32]byte {
	var res [32]byte

	binary.BigEndian.PutUint64(res[0:8], b.words[3])
	binary.BigEndian.PutUint64(res[8:16], b.words[2])
	binary.BigEndian.PutUint64(res[16:24], b.words[1])
	binary.BigEndian.PutUint64(res[24:32], b.words[0])

	return res
}

// Append sets the bit array to the concatenation of x and y and returns the bit array.
// For example:
//
//	x = 000 (len=3)
//	y = 111 (len=3)
//	Append(x,y) = 000111 (len=6)
func (b *BitArray) Append(x, y *BitArray) *BitArray {
	if x.len == 0 {
		return b.Set(y)
	}
	if y.len == 0 {
		return b.Set(x)
	}
	if x.len > maxUint8-y.len {
		panic("error on bitarray append: result would exceed maximum length of 255 bits")
	}

	// Shift left by y's length and OR with y
	return b.lsh(x, y.len).or(b, y)
}

// AppendBit sets the bit array to the concatenation of x and a single bit.
// Equivalent to Append(x, {bit}) but avoids allocating a temporary BitArray.
func (b *BitArray) AppendBit(x *BitArray, bit uint8) *BitArray {
	if x.len == 0 {
		return b.SetBit(bit)
	}
	b.lsh(x, 1)
	b.words[0] |= uint64(bit & 1)
	return b
}

// MSBs sets the bit array to the most significant 'n' bits of x, that is position 0 to n (exclusive).
// If n >= x.len, the bit array is an exact copy of x.
// Think of this method as array[0:n]
// For example:
//
//	x = 11001011 (len=8)
//	MSBs(x, 4) = 1100 (len=4)
//	MSBs(x, 10) = 11001011 (len=8, original x)
//	MSBs(x, 0) = 0 (len=0)
func (b *BitArray) MSBs(x *BitArray, n uint8) *BitArray {
	if n >= x.len {
		return b.Set(x)
	}

	return b.rsh(x, x.len-n)
}

// Equal checks if two bit arrays are equal
func (b *BitArray) Equal(x *BitArray) bool {
	return b.len == x.len && b.words == x.words
}

// SetBytes interprets the data as the big-endian bytes, sets the bit array to that value and returns it.
// If the data is larger than 32 bytes, only the first 32 bytes are used.
func (b *BitArray) SetBytes(length uint8, data []byte) *BitArray {
	switch l := len(data); l {
	case 0:
		b.clear()
	case 1:
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, uint64(data[0])
	case 2:
		_ = data[1]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, uint64(binary.BigEndian.Uint16(data[0:2]))
	case 3:
		_ = data[2]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, uint64(binary.BigEndian.Uint16(data[1:3]))|uint64(data[0])<<16
	case 4:
		_ = data[3]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, uint64(binary.BigEndian.Uint32(data[0:4]))
	case 5:
		_ = data[4]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, bigEndianUint40(data[0:5])
	case 6:
		_ = data[5]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, bigEndianUint48(data[0:6])
	case 7:
		_ = data[6]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, bigEndianUint56(data[0:7])
	case 8:
		_ = data[7]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, binary.BigEndian.Uint64(data[0:8])
	case 9:
		_ = data[8]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, uint64(data[0]), binary.BigEndian.Uint64(data[1:9])
	case 10:
		_ = data[9]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, uint64(binary.BigEndian.Uint16(data[0:2])), binary.BigEndian.Uint64(data[2:10])
	case 11:
		_ = data[10]
		b.words[3], b.words[2] = 0, 0
		b.words[1], b.words[0] = uint64(binary.BigEndian.Uint16(data[1:3]))|uint64(data[0])<<16, binary.BigEndian.Uint64(data[3:11])
	case 12:
		_ = data[11]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, uint64(binary.BigEndian.Uint32(data[0:4])), binary.BigEndian.Uint64(data[4:12])
	case 13:
		_ = data[12]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, bigEndianUint40(data[0:5]), binary.BigEndian.Uint64(data[5:13])
	case 14:
		_ = data[13]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, bigEndianUint48(data[0:6]), binary.BigEndian.Uint64(data[6:14])
	case 15:
		_ = data[14]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, bigEndianUint56(data[0:7]), binary.BigEndian.Uint64(data[7:15])
	case 16:
		_ = data[15]
		b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, binary.BigEndian.Uint64(data[0:8]), binary.BigEndian.Uint64(data[8:16])
	case 17:
		_ = data[16]
		b.words[3], b.words[2] = 0, uint64(data[0])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[1:9]), binary.BigEndian.Uint64(data[9:17])
	case 18:
		_ = data[17]
		b.words[3], b.words[2] = 0, uint64(binary.BigEndian.Uint16(data[0:2]))
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[2:10]), binary.BigEndian.Uint64(data[10:18])
	case 19:
		_ = data[18]
		b.words[3], b.words[2] = 0, uint64(binary.BigEndian.Uint16(data[1:3]))|uint64(data[0])<<16
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[3:11]), binary.BigEndian.Uint64(data[11:19])
	case 20:
		_ = data[19]
		b.words[3], b.words[2] = 0, uint64(binary.BigEndian.Uint32(data[0:4]))
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[4:12]), binary.BigEndian.Uint64(data[12:20])
	case 21:
		_ = data[20]
		b.words[3], b.words[2] = 0, bigEndianUint40(data[0:5])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[5:13]), binary.BigEndian.Uint64(data[13:21])
	case 22:
		_ = data[21]
		b.words[3], b.words[2] = 0, bigEndianUint48(data[0:6])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[6:14]), binary.BigEndian.Uint64(data[14:22])
	case 23:
		_ = data[22]
		b.words[3], b.words[2] = 0, bigEndianUint56(data[0:7])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[7:15]), binary.BigEndian.Uint64(data[15:23])
	case 24:
		_ = data[23]
		b.words[3], b.words[2] = 0, binary.BigEndian.Uint64(data[0:8])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[8:16]), binary.BigEndian.Uint64(data[16:24])
	case 25:
		_ = data[24]
		b.words[3], b.words[2] = uint64(data[0]), binary.BigEndian.Uint64(data[1:9])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[9:17]), binary.BigEndian.Uint64(data[17:25])
	case 26:
		_ = data[25]
		b.words[3], b.words[2] = uint64(binary.BigEndian.Uint16(data[0:2])), binary.BigEndian.Uint64(data[2:10])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[10:18]), binary.BigEndian.Uint64(data[18:26])
	case 27:
		_ = data[26]
		b.words[3] = uint64(binary.BigEndian.Uint16(data[1:3])) | uint64(data[0])<<16
		b.words[2] = binary.BigEndian.Uint64(data[3:11])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[11:19]), binary.BigEndian.Uint64(data[19:27])
	case 28:
		_ = data[27]
		b.words[3], b.words[2] = uint64(binary.BigEndian.Uint32(data[0:4])), binary.BigEndian.Uint64(data[4:12])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[12:20]), binary.BigEndian.Uint64(data[20:28])
	case 29:
		_ = data[28]
		b.words[3], b.words[2] = bigEndianUint40(data[0:5]), binary.BigEndian.Uint64(data[5:13])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[13:21]), binary.BigEndian.Uint64(data[21:29])
	case 30:
		_ = data[29]
		b.words[3], b.words[2] = bigEndianUint48(data[0:6]), binary.BigEndian.Uint64(data[6:14])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[14:22]), binary.BigEndian.Uint64(data[22:30])
	case 31:
		_ = data[30]
		b.words[3], b.words[2] = bigEndianUint56(data[0:7]), binary.BigEndian.Uint64(data[7:15])
		b.words[1], b.words[0] = binary.BigEndian.Uint64(data[15:23]), binary.BigEndian.Uint64(data[23:31])
	default:
		b.setBytes32(data)
	}
	b.len = length
	b.truncateToLength()
	return b
}

// SetUint64 sets the bit array to the uint64 representation of a bit array.
func (b *BitArray) SetUint64(length uint8, data uint64) *BitArray {
	b.words[0] = data
	b.words[1], b.words[2], b.words[3] = 0, 0, 0
	b.len = length
	b.truncateToLength()
	return b
}

// SetBit sets the bit array to a single bit.
func (b *BitArray) SetBit(bit uint8) *BitArray {
	b.len = 1
	b.words[0] = uint64(bit & 1)
	b.words[1], b.words[2], b.words[3] = 0, 0, 0
	return b
}

// Copy returns a deep copy of the bit array.
func (b *BitArray) Copy() BitArray {
	var res BitArray
	res.Set(b)
	return res
}

// String returns a string representation of the bit array.
// This is typically used for logging or debugging.
func (b *BitArray) String() string {
	bt := b.Bytes()
	return fmt.Sprintf("(%d) %s", b.len, hex.EncodeToString(bt[:]))
}

// Bit returns the bit value at position n, where n = 0 is MSB.
// If n is out of bounds, returns 0.
func (b *BitArray) Bit(n uint8) uint8 {
	if n >= b.Len() {
		return 0
	}

	return b.bitFromLSB(b.Len() - n - 1)
}

// Set sets the bit array to the same value as x.
func (b *BitArray) Set(x *BitArray) *BitArray {
	b.len = x.len
	b.words[0] = x.words[0]
	b.words[1] = x.words[1]
	b.words[2] = x.words[2]
	b.words[3] = x.words[3]
	return b
}

// KeyBytes returns the path-to-DB-key encoding: the active bytes in big-endian
// order followed by a single trailing byte holding the bit-length. The trailing
// length disambiguates paths whose active bytes coincide (e.g. 1-bit "1" and
// 8-bit "00000001" both pack to integer value 1, but their key encodings are
// [0x01, 0x01] and [0x01, 0x08] respectively).
//
// The empty path is encoded as no bytes: byteCount=0 is unique to len=0, so
// no disambiguation byte is needed.
//
// Example:
//
//	len = 10, words = [0x3FF, 0, 0, 0] -> [0x03, 0xFF, 0x0A]
func (b *BitArray) KeyBytes() []byte {
	if b.len == 0 {
		return nil
	}
	bc := b.byteCount()
	res := make([]byte, bc+1)
	wordsBytes := b.Bytes()
	copy(res[:bc], wordsBytes[32-bc:])
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
	_ = dst[32] // bounds check hint
	binary.BigEndian.PutUint64(dst[0:8], b.words[3])
	binary.BigEndian.PutUint64(dst[8:16], b.words[2])
	binary.BigEndian.PutUint64(dst[16:24], b.words[1])
	binary.BigEndian.PutUint64(dst[24:32], b.words[0])
	bc := b.byteCount()
	copy(dst, dst[32-bc:32])
	dst[bc] = b.len
	return dst[:bc+1]
}

// bitFromLSB returns the bit value at position n, where n = 0 is LSB.
// If n is out of bounds, returns 0.
func (b *BitArray) bitFromLSB(n uint8) uint8 {
	if n >= b.len {
		return 0
	}

	if (b.words[n/64] & (1 << (n % 64))) != 0 {
		return 1
	}

	return 0
}

// copyLsb sets the bit array to the least significant 'n' bits of x.
// n is counted from the least significant bit, starting at 0.
// If length >= x.len, the bit array is an exact copy of x.
// For example:
//
//	x = 11001011 (len=8)
//	copyLsb(x, 4) = 1011 (len=4)
//	copyLsb(x, 10) = 11001011 (len=8, original x)
//	copyLsb(x, 0) = 0 (len=0)
func (b *BitArray) copyLsb(x *BitArray, n uint8) *BitArray {
	if n >= x.len {
		return b.Set(x)
	}

	b.len = n

	switch {
	case n == 0:
		b.words = [4]uint64{0, 0, 0, 0}
	case n <= 64:
		b.words[0] = x.words[0] & (maxUint64 >> (64 - n))
		b.words[1], b.words[2], b.words[3] = 0, 0, 0
	case n <= 128:
		b.words[0] = x.words[0]
		b.words[1] = x.words[1] & (maxUint64 >> (128 - n))
		b.words[2], b.words[3] = 0, 0
	case n <= 192:
		b.words[0] = x.words[0]
		b.words[1] = x.words[1]
		b.words[2] = x.words[2] & (maxUint64 >> (192 - n))
		b.words[3] = 0
	default:
		b.words[0] = x.words[0]
		b.words[1] = x.words[1]
		b.words[2] = x.words[2]
		b.words[3] = x.words[3] & (maxUint64 >> (256 - uint16(n)))
	}

	return b
}

// lsb returns the least significant bits of `x` with `n` counted from the most significant bit, starting at 0.
// Think of this method as array[n:]
// For example:
//
//	x = 11001011 (len=8)
//	lsb(x, 1) = 1001011 (len=7)
//	lsb(x, 10) = 0 (len=0)
//	lsb(x, 0) = 11001011 (len=8, original x)
func (b *BitArray) lsb(x *BitArray, n uint8) *BitArray {
	if n == 0 {
		return b.Set(x)
	}

	if n > x.Len() {
		return b.clear()
	}

	return b.copyLsb(x, x.Len()-n)
}

// or sets the bit array to x | y and returns the bit array.
func (b *BitArray) or(x, y *BitArray) *BitArray {
	b.words[0] = x.words[0] | y.words[0]
	b.words[1] = x.words[1] | y.words[1]
	b.words[2] = x.words[2] | y.words[2]
	b.words[3] = x.words[3] | y.words[3]
	b.len = x.len
	return b
}

// rsh sets the bit array to x >> n and returns the bit array.
func (b *BitArray) rsh(x *BitArray, n uint8) *BitArray {
	if x.len == 0 {
		return b.Set(x)
	}

	if n >= x.len {
		return b.clear()
	}

	switch {
	case n == 0:
		return b.Set(x)
	case n >= 192:
		b.rsh192(x)
		b.len = x.len - n
		n -= 192
		b.words[0] >>= n
	case n >= 128:
		b.rsh128(x)
		b.len = x.len - n
		n -= 128
		b.words[0] = (b.words[0] >> n) | (b.words[1] << (64 - n))
		b.words[1] >>= n
	case n >= 64:
		b.rsh64(x)
		b.len = x.len - n
		n -= 64
		b.words[0] = (b.words[0] >> n) | (b.words[1] << (64 - n))
		b.words[1] = (b.words[1] >> n) | (b.words[2] << (64 - n))
		b.words[2] >>= n
	default:
		b.Set(x)
		b.len -= n
		b.words[0] = (b.words[0] >> n) | (b.words[1] << (64 - n))
		b.words[1] = (b.words[1] >> n) | (b.words[2] << (64 - n))
		b.words[2] = (b.words[2] >> n) | (b.words[3] << (64 - n))
		b.words[3] >>= n
	}

	b.truncateToLength()
	return b
}

// lsh sets the bit array to x << n and returns the bit array.
func (b *BitArray) lsh(x *BitArray, n uint8) *BitArray {
	if x.len == 0 || n == 0 {
		return b.Set(x)
	}

	// If the result will overflow, we set the length to the max length
	// but we still shift `n` bits
	if n > maxUint8-x.len {
		b.len = maxUint8
	} else {
		b.len = x.len + n
	}

	switch {
	case n >= 192:
		b.lsh192(x)
		n -= 192
		b.words[3] <<= n
	case n >= 128:
		b.lsh128(x)
		n -= 128
		b.words[3] = (b.words[3] << n) | (b.words[2] >> (64 - n))
		b.words[2] <<= n
	case n >= 64:
		b.lsh64(x)
		n -= 64
		b.words[3] = (b.words[3] << n) | (b.words[2] >> (64 - n))
		b.words[2] = (b.words[2] << n) | (b.words[1] >> (64 - n))
		b.words[1] <<= n
	default:
		b.words[3], b.words[2], b.words[1], b.words[0] = x.words[3], x.words[2], x.words[1], x.words[0]
		b.words[3] = (b.words[3] << n) | (b.words[2] >> (64 - n))
		b.words[2] = (b.words[2] << n) | (b.words[1] >> (64 - n))
		b.words[1] = (b.words[1] << n) | (b.words[0] >> (64 - n))
		b.words[0] <<= n
	}

	b.truncateToLength()
	return b
}

func (b *BitArray) setBytes32(data []byte) {
	_ = data[31] // bound check hint, see https://golang.org/issue/14808
	b.words[3] = binary.BigEndian.Uint64(data[0:8])
	b.words[2] = binary.BigEndian.Uint64(data[8:16])
	b.words[1] = binary.BigEndian.Uint64(data[16:24])
	b.words[0] = binary.BigEndian.Uint64(data[24:32])
}

// byteCount returns the minimum number of bytes needed to represent the bit array.
// It rounds up to the nearest byte.
func (b *BitArray) byteCount() uint {
	const bits8 = 8
	return (uint(b.len) + (bits8 - 1)) / uint(bits8)
}

func (b *BitArray) rsh64(x *BitArray) {
	b.words[3], b.words[2], b.words[1], b.words[0] = 0, x.words[3], x.words[2], x.words[1]
}

func (b *BitArray) rsh128(x *BitArray) {
	b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, x.words[3], x.words[2]
}

func (b *BitArray) rsh192(x *BitArray) {
	b.words[3], b.words[2], b.words[1], b.words[0] = 0, 0, 0, x.words[3]
}

func (b *BitArray) lsh64(x *BitArray) {
	b.words[3], b.words[2], b.words[1], b.words[0] = x.words[2], x.words[1], x.words[0], 0
}

func (b *BitArray) lsh128(x *BitArray) {
	b.words[3], b.words[2], b.words[1], b.words[0] = x.words[1], x.words[0], 0, 0
}

func (b *BitArray) lsh192(x *BitArray) {
	b.words[3], b.words[2], b.words[1], b.words[0] = x.words[0], 0, 0, 0
}

func (b *BitArray) clear() *BitArray {
	b.len = 0
	b.words[0], b.words[1], b.words[2], b.words[3] = 0, 0, 0, 0
	return b
}

// truncateToLength truncates the bit array to the specified length, ensuring that any unused bits are all zeros.
//
// Example:
//
//	b := &BitArray{
//	    len: 5,
//	    words: [4]uint64{
//	        0xFFFFFFFFFFFFFFFF,  // Before: all bits are 1
//	        0x0, 0x0, 0x0,
//	    },
//	}
//	b.truncateToLength()
//	// After: only first 5 bits remain
//	// words[0] = 0x000000000000001F
//	// words[1..3] = 0x0
func (b *BitArray) truncateToLength() {
	switch {
	case b.len == 0:
		b.words = [4]uint64{0, 0, 0, 0}
	case b.len <= 64:
		b.words[0] &= maxUint64 >> (64 - b.len)
		b.words[1], b.words[2], b.words[3] = 0, 0, 0
	case b.len <= 128:
		b.words[1] &= maxUint64 >> (128 - b.len)
		b.words[2], b.words[3] = 0, 0
	case b.len <= 192:
		b.words[2] &= maxUint64 >> (192 - b.len)
		b.words[3] = 0
	default:
		b.words[3] &= maxUint64 >> (256 - uint16(b.len))
	}
}

func bigEndianUint40(b []byte) uint64 {
	_ = b[4] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[4]) | uint64(b[3])<<8 | uint64(b[2])<<16 | uint64(b[1])<<24 |
		uint64(b[0])<<32
}

func bigEndianUint48(b []byte) uint64 {
	_ = b[5] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[5]) | uint64(b[4])<<8 | uint64(b[3])<<16 | uint64(b[2])<<24 |
		uint64(b[1])<<32 | uint64(b[0])<<40
}

func bigEndianUint56(b []byte) uint64 {
	_ = b[6] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[6]) | uint64(b[5])<<8 | uint64(b[4])<<16 | uint64(b[3])<<24 |
		uint64(b[2])<<32 | uint64(b[1])<<40 | uint64(b[0])<<48
}
