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

// simple nonconcurrent reference implementation for hashsize segment based
// Binary Merkle tree hash on arbitrary but fixed maximum chunksize
//
// This implementation does not take advantage of any paralellisms and uses
// far more memory than necessary, but it is easy to see that it is correct.
// It can be used for generating test cases for optimized implementations.
// see testBMTHasherCorrectness function in bmt_test.go
package bmt

import (
	"hash"
)

type RBMTHasher struct {
	span    int
	section int
	cap     int
	h       hash.Hash
}

func NewRBMTHasher(hasher Hasher, count int) *RBMTHasher {
	h := hasher()
	hashsize := h.Size()
	maxsize := hashsize * count
	c := 2
	for ; c < count; c *= 2 {
	}
	if c > 2 {
		c /= 2
	}
	return &RBMTHasher{
		section: 2 * hashsize,
		span:    c * hashsize,
		cap:     maxsize,
		h:       h,
	}
}

func (self *RBMTHasher) Hash(d []byte) []byte {
	if len(d) > self.cap {
		d = d[:self.cap]
	}

	return self.hash(d, self.span)
}

func (self *RBMTHasher) hash(d []byte, s int) []byte {
	l := len(d)
	left := d
	var right []byte
	if l > self.section {
		for ; s >= l; s /= 2 {
		}
		left = self.hash(d[:s], s)
		right = d[s:]
		if l-s > self.section/2 {
			right = self.hash(right, s)
		}
	}
	defer self.h.Reset()
	self.h.Write(left)
	self.h.Write(right)
	h := self.h.Sum(nil)
	return h
}
