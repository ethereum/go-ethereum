// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package leb128 provides functions for reading integer values encoded in the
// Little Endian Base 128 (LEB128) format: https://en.wikipedia.org/wiki/LEB128
package leb128

import (
	"io"
)

// ReadVarUint32Size reads a LEB128 encoded unsigned 32-bit integer from r.
// It returns the integer value, the size of the encoded value (in bytes), and
// the error (if any).
func ReadVarUint32Size(r io.Reader) (res uint32, size uint, err error) {
	b := make([]byte, 1)
	var shift uint
	for {
		if _, err = io.ReadFull(r, b); err != nil {
			return
		}

		size++

		cur := uint32(b[0])
		res |= (cur & 0x7f) << (shift)
		if cur&0x80 == 0 {
			return res, size, nil
		}
		shift += 7
	}
}

// ReadVarUint32 reads a LEB128 encoded unsigned 32-bit integer from r, and
// returns the integer value, and the error (if any).
func ReadVarUint32(r io.Reader) (uint32, error) {
	n, _, err := ReadVarUint32Size(r)
	return n, err
}

// ReadVarint32Size reads a LEB128 encoded signed 32-bit integer from r, and
// returns the integer value, the size of the encoded value, and the error
// (if any)
func ReadVarint32Size(r io.Reader) (res int32, size uint, err error) {
	res64, size, err := ReadVarint64Size(r)
	res = int32(res64)
	return
}

// ReadVarint32 reads a LEB128 encoded signed 32-bit integer from r, and
// returns the integer value, and the error (if any).
func ReadVarint32(r io.Reader) (int32, error) {
	n, _, err := ReadVarint32Size(r)
	return n, err
}

// ReadVarint64Size reads a LEB128 encoded signed 64-bit integer from r, and
// returns the integer value, the size of the encoded value, and the error
// (if any)
func ReadVarint64Size(r io.Reader) (res int64, size uint, err error) {
	var shift uint
	var sign int64 = -1
	b := make([]byte, 1)

	for {
		if _, err = io.ReadFull(r, b); err != nil {
			return
		}
		size++

		cur := int64(b[0])
		res |= (cur & 0x7f) << shift
		shift += 7
		sign <<= 7
		if cur&0x80 == 0 {
			break
		}
	}

	if ((sign >> 1) & res) != 0 {
		res |= sign
	}
	return res, size, nil
}

// ReadVarint64 reads a LEB128 encoded signed 64-bit integer from r, and
// returns the integer value, and the error (if any).
func ReadVarint64(r io.Reader) (int64, error) {
	n, _, err := ReadVarint64Size(r)
	return n, err
}
