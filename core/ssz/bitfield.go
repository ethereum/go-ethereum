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

package ssz

import (
	"fmt"
	mathbits "math/bits" // aliased: locals named `bits` are the natural term here
)

// Bitfields are packed LSB-first: bit i of the field is bit i%8 of byte i/8.
//
// A Bitvector[N] serializes as exactly (N+7)/8 bytes with any padding bits in
// the final byte required to be zero.
//
// A Bitlist serializes with one extra delimiter bit set at index len (after
// the last data bit), so the bit length is recoverable from the bytes: the
// serialization is never empty and its last byte is never zero.

// BitvectorSize returns the serialized size of a Bitvector[nbits].
func BitvectorSize(nbits uint64) int {
	return int((nbits + 7) / 8)
}

// ValidateBitvector strictly validates the serialization of a
// Bitvector[nbits]: exact byte length and no excess bits set in the final
// partial byte. The bytes themselves are the runtime representation, so
// validation is all a decoder needs.
func ValidateBitvector(data []byte, nbits uint64) error {
	if uint64(len(data)) != uint64(BitvectorSize(nbits)) {
		return fmt.Errorf("ssz: bitvector[%d] needs %d bytes, have %d: %w",
			nbits, BitvectorSize(nbits), len(data), ErrSize)
	}
	if rem := nbits % 8; rem != 0 {
		if data[len(data)-1]>>rem != 0 {
			return fmt.Errorf("ssz: bitvector[%d] has bits set beyond length: %w", nbits, ErrExcessBits)
		}
	}
	return nil
}

// BitlistSize returns the serialized size of a bitlist of nbits data bits
// (including the delimiter bit).
func BitlistSize(nbits uint64) int {
	return int(nbits/8) + 1
}

// AppendBitlist appends the serialization of a bitlist given its packed data
// bits and bit length: the data bits with the delimiter bit set at index
// nbits. bits must hold at least (nbits+7)/8 bytes with no bits set at or
// above nbits.
func AppendBitlist(dst []byte, bits []byte, nbits uint64) []byte {
	full := nbits / 8 // whole data bytes before the delimiter byte
	dst = append(dst, bits[:full]...)
	var last byte
	if nbits%8 != 0 {
		last = bits[full]
	}
	return append(dst, last|1<<(nbits%8))
}

// DecodeBitlist strictly decodes a bitlist serialization, returning the
// packed data bits (delimiter stripped) and the bit length. limit is the
// type's capacity in bits.
func DecodeBitlist(data []byte, limit uint64) (bits []byte, nbits uint64, err error) {
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("ssz: empty bitlist (delimiter bit is mandatory): %w", ErrBadBitlist)
	}
	last := data[len(data)-1]
	if last == 0 {
		return nil, 0, fmt.Errorf("ssz: bitlist last byte zero (delimiter bit missing): %w", ErrBadBitlist)
	}
	// The highest set bit of the last byte is the delimiter; bits below it
	// are data.
	delim := uint64(mathbits.Len8(last)) - 1 // bit index of delimiter within last byte
	nbits = uint64(len(data)-1)*8 + delim
	if nbits > limit {
		return nil, 0, fmt.Errorf("ssz: bitlist of %d bits exceeds limit %d: %w", nbits, limit, ErrTooBig)
	}
	// Copy and strip the delimiter bit.
	bits = make([]byte, (nbits+7)/8)
	copy(bits, data[:len(bits)])
	if delim != 0 {
		bits[len(bits)-1] &^= 1 << delim
	}
	return bits, nbits, nil
}
