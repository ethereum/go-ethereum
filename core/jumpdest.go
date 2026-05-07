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

package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	jumpDestHitMeter  = metrics.NewRegisteredMeter("chain/cache/jumpdest/hit", nil)
	jumpDestMissMeter = metrics.NewRegisteredMeter("chain/cache/jumpdest/miss", nil)
)

// jumpDestCacheSize is the total memory budget granted to the jumpdest
// analysis cache.
const jumpDestCacheSize = 128 * 1024 * 1024

// JumpDestCache is a thread-safe, byte-bounded LRU of JUMPDEST analysis
// bitmaps. It is owned by BlockChain and shared across block processing and
// prefetching, keyed by the immutable contract code hash.
type JumpDestCache struct {
	cache *lru.SizeConstrainedCache[common.Hash, vm.BitVec]
}

// NewJumpDestCache constructs the analysis cache.
func NewJumpDestCache() *JumpDestCache {
	return &JumpDestCache{
		cache: lru.NewSizeConstrainedCache[common.Hash, vm.BitVec](jumpDestCacheSize),
	}
}

// Load retrieves the cached jumpdest analysis for the given code hash.
func (c *JumpDestCache) Load(hash common.Hash) (vm.BitVec, bool) {
	v, ok := c.cache.Get(hash)
	if ok {
		jumpDestHitMeter.Mark(1)
	} else {
		jumpDestMissMeter.Mark(1)
	}
	return v, ok
}

// Store saves the jumpdest analysis for the given code hash.
func (c *JumpDestCache) Store(hash common.Hash, b vm.BitVec) {
	c.cache.Add(hash, b)
}
