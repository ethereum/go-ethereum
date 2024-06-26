// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

//go:build amd64 && !gccgo && !appengine

package asmhex

import (
	"encoding/hex"

	"golang.org/x/sys/cpu"
)

func Encode(dst, src []byte) int {
	if len(dst) < len(src)*2 {
		panic("dst buffer is too small")
	}
	if len(src) == 0 {
		return 0
	}
	switch {
	case cpu.X86.HasAVX:
		encodeAVX(&dst[0], &src[0], uint64(len(src)), &alphabet[0])
	case cpu.X86.HasSSE41:
		encodeSSE(&dst[0], &src[0], uint64(len(src)), &alphabet[0])
	default:
		return hex.Encode(dst, src)
	}
	return len(src) * 2
}

func Decode(dst, src []byte) (int, error) {
	if len(src)%2 != 0 {
		return 0, hex.ErrLength
	}
	if len(dst) < len(src)/2 {
		panic("dst buffer is too small")
	}
	if len(src) == 0 {
		return 0, nil
	}
	var (
		n  uint64
		ok bool
	)
	switch {
	case cpu.X86.HasAVX:
		n, ok = decodeAVX(&dst[0], &src[0], uint64(len(src)))
	case cpu.X86.HasSSE41:
		n, ok = decodeSSE(&dst[0], &src[0], uint64(len(src)))
	default:
		return hex.Decode(dst, src)
	}
	if !ok {
		return 0, hex.InvalidByteError(src[n])
	}
	return len(src) / 2, nil
}

// This function is implemented in asmhex_encode_amd64.s
//
//go:noescape
func encodeAVX(dst *byte, src *byte, len uint64, alpha *byte)

// This function is implemented in asmhex_encode_amd64.s
//
//go:noescape
func encodeSSE(dst *byte, src *byte, len uint64, alpha *byte)

// This function is implemented in asmhex_decode_amd64.s
//
//go:noescape
func decodeAVX(dst *byte, src *byte, len uint64) (n uint64, ok bool)

// This function is implemented in asmhex_decode_amd64.s
//
//go:noescape
func decodeSSE(dst *byte, src *byte, len uint64) (n uint64, ok bool)
