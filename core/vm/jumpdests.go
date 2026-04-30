// Copyright 2024 The go-ethereum Authors
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

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
)

// globalJumpDestCacheSize caps the global cache. Worst case ~48MB (24KB code
// → ~3KB bitmap × 16384 entries).
const globalJumpDestCacheSize = 16384

// JumpDestCache represents the cache of jumpdest analysis results.
type JumpDestCache interface {
	// Load retrieves the cached jumpdest analysis for the given code hash.
	// Returns the BitVec and true if found, or nil and false if not cached.
	Load(codeHash common.Hash) (BitVec, bool)

	// Store saves the jumpdest analysis for the given code hash.
	Store(codeHash common.Hash, vec BitVec)
}

// mapJumpDests is the default implementation of JumpDests using a map.
// This implementation is not thread-safe and is meant to be used per EVM instance.
type mapJumpDests map[common.Hash]BitVec

// newMapJumpDests creates a new map-based JumpDests implementation.
func newMapJumpDests() JumpDestCache {
	return make(mapJumpDests)
}

func (j mapJumpDests) Load(codeHash common.Hash) (BitVec, bool) {
	vec, ok := j[codeHash]
	return vec, ok
}

func (j mapJumpDests) Store(codeHash common.Hash, vec BitVec) {
	j[codeHash] = vec
}

// globalJumpDests is a process-global LRU of JUMPDEST bitmaps, shared across
// every EVM instance and keyed by the immutable contract code hash.
var globalJumpDests = &lruJumpDests{cache: lru.NewCache[common.Hash, BitVec](globalJumpDestCacheSize)}

type lruJumpDests struct {
	cache *lru.Cache[common.Hash, BitVec]
}

func (j *lruJumpDests) Load(codeHash common.Hash) (BitVec, bool) {
	return j.cache.Get(codeHash)
}

func (j *lruJumpDests) Store(codeHash common.Hash, vec BitVec) {
	j.cache.Add(codeHash, vec)
}
