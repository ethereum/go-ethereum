// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build gc && !purego && arm64

package keccak

//go:noescape
func keccakF1600(a *[25]uint64)
