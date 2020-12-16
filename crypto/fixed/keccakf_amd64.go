// This file is copied from golang/x/crypto/sha3/keccakf_amd64.go

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64,!purego,gc

package fixed

// This function is implemented in keccakf_amd64.s.

//go:noescape

func keccakF1600(state *[25]uint64)
