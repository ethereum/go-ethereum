// Copyright 2026 The go-ethereum Authors
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

//go:build !ziren

package crypto

import (
	"bytes"
	"hash/maphash"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
)

const (
	keccakCacheBits     = 17 // 128K entries, about 16 MB
	keccakCacheMaxInput = 85 // longest input the cache holds, sized for CREATE2 preimages
)

// keccakCacheEntry holds one input and its hash, guarded by the busy flag.
type keccakCacheEntry struct {
	busy atomic.Uint32
	size uint8 // input length plus one, zero marks an unused entry so it never matches the empty input
	data [keccakCacheMaxInput]byte
	hash common.Hash
}

var (
	keccakCache     [1 << keccakCacheBits]keccakCacheEntry
	keccakCacheSeed = maphash.MakeSeed()
)

// cachedKeccak256 returns the hash of a short input, reusing an earlier result for the same input.
func cachedKeccak256(data []byte) common.Hash {
	e := &keccakCache[maphash.Bytes(keccakCacheSeed, data)&(1<<keccakCacheBits-1)]
	if !e.busy.CompareAndSwap(0, 1) {
		// Another goroutine holds the entry, hash without it.
		return keccak256(data)
	}
	if int(e.size) == len(data)+1 && bytes.Equal(e.data[:len(data)], data) {
		h := e.hash
		e.busy.Store(0)
		return h
	}
	h := keccak256(data)
	e.size = uint8(len(data) + 1)
	copy(e.data[:], data)
	e.hash = h
	e.busy.Store(0)
	return h
}

// keccak256 hashes a single input without the cache.
func keccak256(data []byte) (h common.Hash) {
	d := hasherPool.Get().(KeccakState)
	d.Reset()
	d.Write(data)
	d.Read(h[:])
	hasherPool.Put(d)
	return h
}
