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
	"encoding/binary"

	"github.com/holiman/uint256"
)

// Append-style primitive encoders. All multi-byte values are little-endian
// per the SSZ specification. Encoders cannot fail; size bounds are enforced
// once at the Encode entry point.

// AppendBool appends the SSZ encoding of a boolean (0x00 or 0x01).
func AppendBool(dst []byte, v bool) []byte {
	if v {
		return append(dst, 1)
	}
	return append(dst, 0)
}

// AppendUint8 appends a uint8.
func AppendUint8(dst []byte, v uint8) []byte {
	return append(dst, v)
}

// AppendUint16 appends a little-endian uint16.
func AppendUint16(dst []byte, v uint16) []byte {
	return binary.LittleEndian.AppendUint16(dst, v)
}

// AppendUint32 appends a little-endian uint32.
func AppendUint32(dst []byte, v uint32) []byte {
	return binary.LittleEndian.AppendUint32(dst, v)
}

// AppendUint64 appends a little-endian uint64.
func AppendUint64(dst []byte, v uint64) []byte {
	return binary.LittleEndian.AppendUint64(dst, v)
}

// AppendUint128 appends a little-endian uint128 (16 bytes). The two upper
// limbs of v must be zero; they are ignored.
func AppendUint128(dst []byte, v *uint256.Int) []byte {
	dst = binary.LittleEndian.AppendUint64(dst, v[0])
	return binary.LittleEndian.AppendUint64(dst, v[1])
}

// AppendUint256 appends a little-endian uint256 (32 bytes). It delegates to
// the SSZ support that holiman/uint256 already ships, kept here so container
// encoders read as a uniform Append* field listing.
func AppendUint256(dst []byte, v *uint256.Int) []byte {
	dst, _ = v.MarshalSSZAppend(dst) // cannot fail
	return dst
}

// AppendOffset appends a 4-byte little-endian variable-part offset.
func AppendOffset(dst []byte, offset int) []byte {
	return binary.LittleEndian.AppendUint32(dst, uint32(offset))
}
