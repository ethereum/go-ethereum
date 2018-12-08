// Copyright 2017 The go-ethereum Authors
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

// Package bmt is a simple nonconcurrent reference implementation for hashsize segment based
// Binary Merkle tree hash on arbitrary but fixed maximum chunksize
//
// This implementation does not take advantage of any paralellisms and uses
// far more memory than necessary, but it is easy to see that it is correct.
// It can be used for generating test cases for optimized implementations.
// There is extra check on reference hasher correctness in bmt_test.go
// * TestRefHasher
// * testBMTHasherCorrectness function
package bmt

import (
	"hash"
)

// RefHasher is the non-optimized easy-to-read reference implementation of BMT
type RefHasher struct {
	maxDataLength int       // c * hashSize, where c = 2 ^ ceil(log2(count)), where count = ceil(length / hashSize)
	sectionLength int       // 2 * hashSize
	hasher        hash.Hash // base hash func (Keccak256 SHA3)
}

// NewRefHasher returns a new RefHasher
func NewRefHasher(hasher BaseHasherFunc, count int) *RefHasher {
	h := hasher()
	hashsize := h.Size()
	c := 2
	for ; c < count; c *= 2 {
	}
	return &RefHasher{
		sectionLength: 2 * hashsize,
		maxDataLength: c * hashsize,
		hasher:        h,
	}
}

// Hash returns the BMT hash of the byte slice
// implements the SwarmHash interface
func (rh *RefHasher) Hash(data []byte) []byte {
	// if data is shorter than the base length (maxDataLength), we provide padding with zeros
	d := make([]byte, rh.maxDataLength)
	length := len(data)
	if length > rh.maxDataLength {
		length = rh.maxDataLength
	}
	copy(d, data[:length])
	return rh.hash(d, rh.maxDataLength)
}

// data has length maxDataLength = segmentSize * 2^k
// hash calls itself recursively on both halves of the given slice
// concatenates the results, and returns the hash of that
// if the length of d is 2 * segmentSize then just returns the hash of that section
func (rh *RefHasher) hash(data []byte, length int) []byte {
	var section []byte
	if length == rh.sectionLength {
		// section contains two data segments (d)
		section = data
	} else {
		// section contains hashes of left and right BMT subtreea
		// to be calculated by calling hash recursively on left and right half of d
		length /= 2
		section = append(rh.hash(data[:length], length), rh.hash(data[length:], length)...)
	}
	rh.hasher.Reset()
	rh.hasher.Write(section)
	return rh.hasher.Sum(nil)
}
