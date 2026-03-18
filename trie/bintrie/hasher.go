// Copyright 2026 go-ethereum Authors
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

package bintrie

import (
	"crypto/sha256"
	"hash"
	"sync"
)

var sha256Pool = sync.Pool{
	New: func() any {
		return sha256.New()
	},
}

func newSha256() hash.Hash {
	h := sha256Pool.Get().(hash.Hash)
	h.Reset()
	return h
}

func returnSha256(h hash.Hash) {
	sha256Pool.Put(h)
}

// sha256Sum256 computes a sha256 digest and returns it as a common.Hash.
func sha256Sum256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// parallelHashDepth controls below which depth hashing is parallelised.
const parallelHashDepth = 4
