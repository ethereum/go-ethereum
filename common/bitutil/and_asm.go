// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// inspired by: https://github.com/golang/go/blob/4a3cef2036097d323b6cc0bbe90fc4d8c7588660/src/crypto/internal/fips140/subtle/xor_asm.go

//go:build (amd64 || arm64) && !purego

package bitutil

func andBytes(dst, a, b []byte) int {
	n := min(len(a), len(b))
	if n == 0 {
		return 0
	}
	andBytesASM(&dst[0], &a[0], &b[0], n)
	return n
}

//go:noescape
func andBytesASM(dst, a, b *byte, n int)
