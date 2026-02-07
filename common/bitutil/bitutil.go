// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from: https://go.dev/src/crypto/subtle/xor_generic.go

// Package bitutil implements fast bitwise operations.
package bitutil

import (
	"crypto/subtle"
	"runtime"
	"unsafe"
)

const wordSize = int(unsafe.Sizeof(uintptr(0)))
const supportsUnaligned = runtime.GOARCH == "386" || runtime.GOARCH == "amd64" || runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" || runtime.GOARCH == "s390x"

// XORBytes xors the bytes in a and b. The destination is assumed to have enough
// space. Returns the number of bytes xor'd.
//
// If dst does not have length at least n,
// XORBytes panics without writing anything to dst.
//
// dst and x or y may overlap exactly or not at all,
// otherwise XORBytes may panic.
//
// Deprecated: use crypto/subtle.XORBytes
func XORBytes(dst, a, b []byte) int {
	return subtle.XORBytes(dst, a, b)
}

// ANDBytes ands the bytes in a and b. The destination is assumed to have enough
// space. Returns the number of bytes and'd.
func ANDBytes(dst, a, b []byte) int {
	if supportsUnaligned {
		return fastANDBytes(dst, a, b)
	}
	return safeANDBytes(dst, a, b)
}

// fastANDBytes ands in bulk. It only works on architectures that support
// unaligned read/writes.
func fastANDBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	w := n / wordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst))
		aw := *(*[]uintptr)(unsafe.Pointer(&a))
		bw := *(*[]uintptr)(unsafe.Pointer(&b))
		for i := 0; i < w; i++ {
			dw[i] = aw[i] & bw[i]
		}
	}
	for i := n - n%wordSize; i < n; i++ {
		dst[i] = a[i] & b[i]
	}
	return n
}

// safeANDBytes ands one by one. It works on all architectures, independent if
// it supports unaligned read/writes or not.
func safeANDBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] & b[i]
	}
	return n
}

// ORBytes ors the bytes in a and b. The destination is assumed to have enough
// space. Returns the number of bytes or'd.
//
// dst and x or y may overlap exactly or not at all,
// otherwise ORBytes may panic.
func ORBytes(dst, a, b []byte) int {
	n := min(len(a), len(b))
	if inexactOverlap(dst[:n], a[:n]) || inexactOverlap(dst[:n], b[:n]) {
		panic("ORBytes: invalid overlap")
	}
	return orBytes(dst, a, b)
}

// safeORBytes ors one by one. It works on all architectures, independent if
// it supports unaligned read/writes or not.
func safeORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] | b[i]
	}
	return n
}

// TestBytes tests whether any bit is set in the input byte slice.
func TestBytes(p []byte) bool {
	if supportsUnaligned {
		return fastTestBytes(p)
	}
	return safeTestBytes(p)
}

// fastTestBytes tests for set bits in bulk. It only works on architectures that
// support unaligned read/writes.
func fastTestBytes(p []byte) bool {
	n := len(p)
	w := n / wordSize
	if w > 0 {
		pw := *(*[]uintptr)(unsafe.Pointer(&p))
		for i := 0; i < w; i++ {
			if pw[i] != 0 {
				return true
			}
		}
	}
	for i := n - n%wordSize; i < n; i++ {
		if p[i] != 0 {
			return true
		}
	}
	return false
}

// safeTestBytes tests for set bits one byte at a time. It works on all
// architectures, independent if it supports unaligned read/writes or not.
func safeTestBytes(p []byte) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return true
		}
	}
	return false
}

// anyOverlap reports whether x and y share memory at any (not necessarily
// corresponding) index. The memory beyond the slice length is ignored.
// from: https://github.com/golang/go/blob/4a3cef2036097d323b6cc0bbe90fc4d8c7588660/src/crypto/internal/fips140/alias/alias.go#L13-L17
func anyOverlap(x, y []byte) bool {
	return len(x) > 0 && len(y) > 0 &&
		uintptr(unsafe.Pointer(&x[0])) <= uintptr(unsafe.Pointer(&y[len(y)-1])) &&
		uintptr(unsafe.Pointer(&y[0])) <= uintptr(unsafe.Pointer(&x[len(x)-1]))
}

// inexactOverlap reports whether x and y share memory at any non-corresponding
// index. The memory beyond the slice length is ignored. Note that x and y can
// have different lengths and still not have any inexact overlap.
//
// inexactOverlap can be used to implement the requirements of the crypto/cipher
// AEAD, Block, BlockMode and Stream interfaces.
// from: https://github.com/golang/go/blob/4a3cef2036097d323b6cc0bbe90fc4d8c7588660/src/crypto/internal/fips140/alias/alias.go#L25-L30
func inexactOverlap(x, y []byte) bool {
	if len(x) == 0 || len(y) == 0 || &x[0] == &y[0] {
		return false
	}
	return anyOverlap(x, y)
}
