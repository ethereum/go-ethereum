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
)

// SizeVariableSlice returns the serialized size of a slice of variable-size
// elements: one 4-byte offset per element plus the element payloads.
func SizeVariableSlice[T Object](elems []T) int {
	size := BytesPerLengthOffset * len(elems)
	for _, e := range elems {
		size += e.SizeSSZ()
	}
	return size
}

// SizeFixedSlice returns the serialized size of a slice of fixed-size
// elements (plain concatenation, no offsets).
func SizeFixedSlice[T Object](elems []T) int {
	size := 0
	for _, e := range elems {
		size += e.SizeSSZ()
	}
	return size
}

// MarshalVariableSlice appends the serialization of a list or vector of
// variable-size elements: an offset table followed by the element payloads.
func MarshalVariableSlice[T Object](dst []byte, elems []T) ([]byte, error) {
	offset := BytesPerLengthOffset * len(elems)
	for _, e := range elems {
		dst = AppendOffset(dst, offset)
		offset += e.SizeSSZ()
	}
	var err error
	for _, e := range elems {
		if dst, err = e.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// MarshalFixedSlice appends the serialization of a list or vector of
// fixed-size elements: plain concatenation.
func MarshalFixedSlice[T Object](dst []byte, elems []T) ([]byte, error) {
	var err error
	for _, e := range elems {
		if dst, err = e.MarshalSSZTo(dst); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// UnmarshalVariableSlice strictly decodes a list of variable-size elements.
// The element count is implied by the first offset (the fixed part is pure
// offset table, so first/4 == count). limit is the type's capacity.
func UnmarshalVariableSlice[T any, PT interface {
	*T
	Object
}](data []byte, limit uint64) ([]T, error) {
	if len(data) == 0 {
		return nil, nil // empty list
	}
	if len(data) < BytesPerLengthOffset {
		return nil, fmt.Errorf("ssz: variable list of %d bytes cannot hold an offset: %w", len(data), ErrSize)
	}
	first := uint64(binary.LittleEndian.Uint32(data))
	if first == 0 || first%BytesPerLengthOffset != 0 {
		return nil, fmt.Errorf("ssz: first offset %d not a positive multiple of %d: %w",
			first, BytesPerLengthOffset, ErrOffset)
	}
	count := first / BytesPerLengthOffset
	if count > limit {
		return nil, fmt.Errorf("ssz: %d elements exceed list limit %d: %w", count, limit, ErrTooBig)
	}
	if first > uint64(len(data)) {
		return nil, fmt.Errorf("ssz: first offset %d beyond input %d: %w", first, len(data), ErrOffset)
	}
	elems := make([]T, count)
	prev := first
	for i := uint64(0); i < count; i++ {
		end := uint64(len(data))
		if i+1 < count {
			end = uint64(binary.LittleEndian.Uint32(data[(i+1)*BytesPerLengthOffset:]))
		}
		if end < prev || end > uint64(len(data)) {
			return nil, fmt.Errorf("ssz: offset %d out of order (prev %d, input %d): %w",
				end, prev, len(data), ErrOffset)
		}
		if err := PT(&elems[i]).UnmarshalSSZ(data[prev:end]); err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		prev = end
	}
	return elems, nil
}

// UnmarshalFixedSlice strictly decodes a list of fixed-size elements of
// elemSize bytes each. limit is the type's capacity.
func UnmarshalFixedSlice[T any, PT interface {
	*T
	Object
}](data []byte, elemSize int, limit uint64) ([]T, error) {
	if elemSize <= 0 {
		panic("ssz: fixed element size must be positive")
	}
	if len(data)%elemSize != 0 {
		return nil, fmt.Errorf("ssz: input %d not a multiple of element size %d: %w",
			len(data), elemSize, ErrSize)
	}
	count := uint64(len(data) / elemSize)
	if count > limit {
		return nil, fmt.Errorf("ssz: %d elements exceed list limit %d: %w", count, limit, ErrTooBig)
	}
	elems := make([]T, count)
	for i := range elems {
		if err := PT(&elems[i]).UnmarshalSSZ(data[i*elemSize : (i+1)*elemSize]); err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
	}
	return elems, nil
}
