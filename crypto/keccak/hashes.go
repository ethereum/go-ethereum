// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keccak

// This file provides functions for creating instances of the SHA-3
// and SHAKE hash functions, as well as utility functions for hashing
// bytes.

import (
	"hash"
)

const (
	dsbyteSHA3   = 0b00000110
	dsbyteKeccak = 0b00000001
	dsbyteShake  = 0b00011111
	dsbyteCShake = 0b00000100

	// rateK[c] is the rate in bytes for Keccak[c] where c is the capacity in
	// bits. Given the sponge size is 1600 bits, the rate is 1600 - c bits.
	rateK256  = (1600 - 256) / 8
	rateK448  = (1600 - 448) / 8
	rateK512  = (1600 - 512) / 8
	rateK768  = (1600 - 768) / 8
	rateK1024 = (1600 - 1024) / 8
)

// NewLegacyKeccak256 creates a new Keccak-256 hash.
//
// Only use this function if you require compatibility with an existing cryptosystem
// that uses non-standard padding. All other users should use New256 instead.
func NewLegacyKeccak256() hash.Hash {
	return &state{rate: rateK512, outputLen: 32, dsbyte: dsbyteKeccak}
}

// NewLegacyKeccak512 creates a new Keccak-512 hash.
//
// Only use this function if you require compatibility with an existing cryptosystem
// that uses non-standard padding. All other users should use New512 instead.
func NewLegacyKeccak512() hash.Hash {
	return &state{rate: rateK1024, outputLen: 64, dsbyte: dsbyteKeccak}
}
