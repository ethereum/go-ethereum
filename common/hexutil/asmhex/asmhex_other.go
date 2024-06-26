// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || gccgo || appengine

package asmhex

import "encoding/hex"

func Encode(dst, src []byte) int {
	return hex.Encode(dst, src)
}

func Decode(dst, src []byte) (int, error) {
	return hex.Decode(dst, src)
}
