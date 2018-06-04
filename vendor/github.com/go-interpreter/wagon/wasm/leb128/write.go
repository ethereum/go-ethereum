// Copyright 2018 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package leb128

import "io"

// Copied from cmd/internal/dwarf/dwarf.go

// AppendUleb128 appends v to b using unsigned LEB128 encoding.
func AppendUleb128(b []byte, v uint64) []byte {
	for {
		c := uint8(v & 0x7f)
		v >>= 7
		if v != 0 {
			c |= 0x80
		}
		b = append(b, c)
		if c&0x80 == 0 {
			break
		}
	}
	return b
}

// AppendSleb128 appends v to b using signed LEB128 encoding.
func AppendSleb128(b []byte, v int64) []byte {
	for {
		c := uint8(v & 0x7f)
		s := uint8(v & 0x40)
		v >>= 7
		if (v != -1 || s == 0) && (v != 0 || s != 0) {
			c |= 0x80
		}
		b = append(b, c)
		if c&0x80 == 0 {
			break
		}
	}
	return b
}

// WriteVarUint32 writes a LEB128 encoded unsigned 32-bit integer to w.
// It returns the integer value, the size of the encoded value (in bytes), and
// the error (if any).
func WriteVarUint32(w io.Writer, cur uint32) (int, error) {
	var buf []byte
	buf = AppendUleb128(buf, uint64(cur))
	return w.Write(buf)
}

// WriteVarint64 writes a LEB128 encoded signed 64-bit integer to w, and
// returns the integer value, the size of the encoded value, and the error
// (if any)
func WriteVarint64(w io.Writer, cur int64) (int, error) {
	var buf []byte
	buf = AppendSleb128(buf, cur)
	return w.Write(buf)
}
