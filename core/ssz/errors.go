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

import "errors"

// Named decode rejection classes. Every malformed input is rejected with an
// error wrapping exactly one of these, so callers match the category with
// errors.Is while the message carries the specifics. Together they are the
// contract FuzzSSZDecode enforces: nil, or one of these, never a panic.
//
// The set follows the hardening list of the SSZ deserialization spec:
// "Offsets: out of order, out of range, mismatching minimum element size.
// Scope: Extra unused bytes, not aligned with element size. More elements
// than a list limit allows."
var (
	// ErrSize is returned when the input length is impossible for the target
	// type (too short for the fixed part, not a multiple of the element size,
	// or a mismatch with the declared vector length).
	ErrSize = errors.New("invalid size")

	// ErrOffset is returned for any violation of the offset rules: the first
	// offset not equal to the fixed-part length, offsets decreasing, or an
	// offset pointing outside the input.
	ErrOffset = errors.New("invalid offset")

	// ErrTrailing is returned when input bytes remain after the value was
	// fully decoded.
	ErrTrailing = errors.New("trailing bytes")

	// ErrBadBoolean is returned when a boolean byte is neither 0x00 nor 0x01.
	ErrBadBoolean = errors.New("invalid boolean")

	// ErrBadBitlist is returned when a bitlist is empty (the delimiter bit is
	// mandatory) or its last byte is zero (delimiter missing).
	ErrBadBitlist = errors.New("invalid bitlist")

	// ErrExcessBits is returned when a bitvector has bits set beyond its
	// declared length in the final partial byte.
	ErrExcessBits = errors.New("excess bits set")

	// ErrTooBig is returned when a (de)serialization would exceed the SSZ
	// 2**32-1 byte envelope, or a list exceeds its declared limit.
	ErrTooBig = errors.New("exceeds size limit")
)
