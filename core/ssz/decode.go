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
	"fmt"

	"github.com/holiman/uint256"
)

// Decoder reads the fixed part of a container serialization field by field
// and concentrates every offset validation rule of the SSZ deserialization
// spec in one place. Usage:
//
//	d := ssz.NewDecoder(data)
//	c.A = d.Uint16()   // fixed field
//	d.Offset()         // variable field: offset recorded for later
//	c.C = d.Uint8()    // fixed field
//	sections, err := d.Sections() // validates everything, slices variable parts
//
// Read methods never panic on short input; they latch an error which the
// terminal call (Sections or Close) returns.
type Decoder struct {
	data []byte
	pos  int
	offs []uint64 // uint64, not int: on 32-bit platforms a hostile 4-byte
	// offset would overflow int and invert the bounds checks below.
	err error
}

// NewDecoder returns a Decoder over data.
func NewDecoder(data []byte) *Decoder {
	return &Decoder{data: data}
}

// remaining checks that n more fixed bytes exist.
func (d *Decoder) remaining(n int) bool {
	if d.err != nil {
		return false
	}
	if len(d.data)-d.pos < n {
		d.err = fmt.Errorf("ssz: fixed part needs %d bytes at position %d, have %d: %w",
			n, d.pos, len(d.data)-d.pos, ErrSize)
		return false
	}
	return true
}

// Bool reads a strict boolean.
func (d *Decoder) Bool() bool {
	if !d.remaining(1) {
		return false
	}
	b := d.data[d.pos]
	d.pos++
	if b > 1 {
		d.err = fmt.Errorf("ssz: boolean byte 0x%02x: %w", b, ErrBadBoolean)
		return false
	}
	return b == 1
}

// Uint8 reads a uint8.
func (d *Decoder) Uint8() uint8 {
	if !d.remaining(1) {
		return 0
	}
	v := d.data[d.pos]
	d.pos++
	return v
}

// Uint16 reads a little-endian uint16.
func (d *Decoder) Uint16() uint16 {
	if !d.remaining(2) {
		return 0
	}
	v := binary.LittleEndian.Uint16(d.data[d.pos:])
	d.pos += 2
	return v
}

// Uint32 reads a little-endian uint32.
func (d *Decoder) Uint32() uint32 {
	if !d.remaining(4) {
		return 0
	}
	v := binary.LittleEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	return v
}

// Uint64 reads a little-endian uint64.
func (d *Decoder) Uint64() uint64 {
	if !d.remaining(8) {
		return 0
	}
	v := binary.LittleEndian.Uint64(d.data[d.pos:])
	d.pos += 8
	return v
}

// Uint128 reads a little-endian uint128 into v.
func (d *Decoder) Uint128(v *uint256.Int) {
	if !d.remaining(16) {
		return
	}
	v[0] = binary.LittleEndian.Uint64(d.data[d.pos:])
	v[1] = binary.LittleEndian.Uint64(d.data[d.pos+8:])
	v[2], v[3] = 0, 0
	d.pos += 16
}

// Uint256 reads a little-endian uint256 into v, reusing the strict decoder
// holiman/uint256 provides.
func (d *Decoder) Uint256(v *uint256.Int) {
	if !d.remaining(32) {
		return
	}
	if err := v.UnmarshalSSZ(d.data[d.pos : d.pos+32]); err != nil {
		d.err = err // unreachable: the length is exact by construction
		return
	}
	d.pos += 32
}

// Bytes copies n fixed bytes into dst (a fixed-size byte vector field).
func (d *Decoder) Bytes(dst []byte) {
	if !d.remaining(len(dst)) {
		return
	}
	copy(dst, d.data[d.pos:])
	d.pos += len(dst)
}

// Offset reads a 4-byte offset for a variable-size field and records it for
// validation by Sections.
func (d *Decoder) Offset() {
	if !d.remaining(4) {
		return
	}
	off := binary.LittleEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	d.offs = append(d.offs, uint64(off))
}

// Err returns the currently latched error, if any.
func (d *Decoder) Err() error {
	return d.err
}

// Close finalizes a value with no variable-size fields: the input must have
// been consumed exactly.
func (d *Decoder) Close() error {
	if d.err != nil {
		return d.err
	}
	if len(d.offs) != 0 {
		panic("ssz: Close called with recorded offsets, use Sections")
	}
	if d.pos != len(d.data) {
		return fmt.Errorf("ssz: %d unconsumed bytes after fixed-size value: %w",
			len(d.data)-d.pos, ErrTrailing)
	}
	return nil
}

// Sections finalizes the fixed part and slices the variable part.
//
// It enforces the deserialization rules of the SSZ spec:
//   - the first offset equals the fixed-part length (no gap, no overlap),
//   - offsets are non-decreasing (empty sections are legal),
//   - every offset is within the input,
//
// and returns one byte slice per recorded offset, in order. The last section
// extends to the end of the input, so consuming all sections consumes the
// whole buffer.
func (d *Decoder) Sections() ([][]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	if len(d.offs) == 0 {
		panic("ssz: Sections called without recorded offsets, use Close")
	}
	if d.offs[0] != uint64(d.pos) {
		return nil, fmt.Errorf("ssz: first offset %d != fixed part length %d: %w",
			d.offs[0], d.pos, ErrOffset)
	}
	sections := make([][]byte, len(d.offs))
	for i, off := range d.offs {
		end := uint64(len(d.data))
		if i+1 < len(d.offs) {
			end = d.offs[i+1]
		}
		if off > end || end > uint64(len(d.data)) {
			return nil, fmt.Errorf("ssz: offset %d beyond next boundary %d (input %d): %w",
				off, end, len(d.data), ErrOffset)
		}
		sections[i] = d.data[off:end]
	}
	return sections, nil
}

// DecodeUint64 strictly decodes a standalone uint64 (the whole input is
// exactly one value).
func DecodeUint64(data []byte) (uint64, error) {
	if len(data) != 8 {
		return 0, fmt.Errorf("ssz: uint64 needs 8 bytes, have %d: %w", len(data), ErrSize)
	}
	return binary.LittleEndian.Uint64(data), nil
}
