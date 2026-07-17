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

package vm

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	precompileCacheHitMeter   = metrics.NewRegisteredMeter("chain/cache/precompile/hit", nil)
	precompileCacheMissMeter  = metrics.NewRegisteredMeter("chain/cache/precompile/miss", nil)
	precompileCacheEntryGauge = metrics.NewRegisteredGauge("chain/cache/precompile/entries", nil)
)

const (
	// maxCacheablePrecompileInput bounds the input size eligible for result
	// caching. Larger inputs are rare one-offs and hashing them for the key
	// eats into the win.
	maxCacheablePrecompileInput = 8192

	// maxCacheablePrecompileOutput bounds the output size stored in the
	// cache, keeping the worst case memory use of an entry small.
	maxCacheablePrecompileOutput = 1024

	// precompileCacheEntries is the maximum number of cached results. With
	// outputs capped by maxCacheablePrecompileOutput, the worst case memory
	// use stays at a few megabytes.
	precompileCacheEntries = 4096
)

// PrecompileCache is a thread-safe LRU of precompile outputs, shared between
// the state prefetcher and block processing so the serial pass can reuse
// results the prefetcher already computed. Entries are namespaced by
// precompile set, so forks never share results across a behaviour change.
type PrecompileCache struct {
	mu     sync.RWMutex
	sets   map[*PrecompiledContracts]*lru.Cache[common.Hash, []byte]
	meters map[common.Address]*precompileCacheMeters
}

// precompileCacheMeters holds the per-address hit and miss meters.
type precompileCacheMeters struct {
	hit  *metrics.Meter
	miss *metrics.Meter
}

// NewPrecompileCache constructs a precompile result cache.
func NewPrecompileCache() *PrecompileCache {
	return &PrecompileCache{
		sets:   make(map[*PrecompiledContracts]*lru.Cache[common.Hash, []byte]),
		meters: make(map[common.Address]*precompileCacheMeters),
	}
}

// load retrieves the cached output for the given key. The returned slice is
// a private copy owned by the caller, entries cross goroutine boundaries.
func (c *PrecompileCache) load(set *PrecompiledContracts, addr common.Address, key common.Hash) ([]byte, bool) {
	c.mu.RLock()
	results := c.sets[set]
	c.mu.RUnlock()

	meters := c.metersFor(addr)
	if results != nil {
		if output, ok := results.Get(key); ok {
			precompileCacheHitMeter.Mark(1)
			meters.hit.Mark(1)
			return common.CopyBytes(output), true
		}
	}
	precompileCacheMissMeter.Mark(1)
	meters.miss.Mark(1)
	return nil, false
}

// store saves the output of a precompile run under the given key. The value
// is copied, the cache never aliases caller memory.
func (c *PrecompileCache) store(set *PrecompiledContracts, addr common.Address, key common.Hash, output []byte) {
	c.mu.RLock()
	results := c.sets[set]
	c.mu.RUnlock()

	if results == nil {
		c.mu.Lock()
		if results = c.sets[set]; results == nil {
			results = lru.NewCache[common.Hash, []byte](precompileCacheEntries)
			c.sets[set] = results
		}
		c.mu.Unlock()
	}
	results.Add(key, common.CopyBytes(output))
	precompileCacheEntryGauge.Update(int64(results.Len()))
}

// metersFor returns the hit and miss meters of the given precompile address,
// registering them on first use.
func (c *PrecompileCache) metersFor(addr common.Address) *precompileCacheMeters {
	c.mu.RLock()
	meters, ok := c.meters[addr]
	c.mu.RUnlock()
	if ok {
		return meters
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if meters, ok = c.meters[addr]; ok {
		return meters
	}
	prefix := fmt.Sprintf("chain/cache/precompile/%#x", addr.Big())
	meters = &precompileCacheMeters{
		hit:  metrics.GetOrRegisterMeter(prefix+"/hit", nil),
		miss: metrics.GetOrRegisterMeter(prefix+"/miss", nil),
	}
	c.meters[addr] = meters
	return meters
}

// CacheablePrecompile lets a precompile opt out of result caching, either
// because its output is not a pure function of the input or because it is
// cheaper to rerun than to cache.
type CacheablePrecompile interface {
	Cacheable() bool
}

// cacheablePrecompile reports whether an invocation is eligible for result
// caching.
func cacheablePrecompile(p PrecompiledContract, input []byte) bool {
	if len(input) > maxCacheablePrecompileInput {
		return false
	}
	if c, ok := p.(CacheablePrecompile); ok {
		return c.Cacheable()
	}
	return true
}

// cacheKeyHasherPool holds hashers for deriving cache keys. This is not a
// consensus hash, sha256 is picked for hardware acceleration.
var cacheKeyHasherPool = sync.Pool{
	New: func() any { return sha256.New() },
}

// precompileCacheKey derives the cache key for a precompile invocation. Fork
// discrimination is handled by the set namespacing, so the key only covers
// the address and input.
func precompileCacheKey(addr common.Address, input []byte) common.Hash {
	h := cacheKeyHasherPool.Get().(hash.Hash)
	h.Reset()
	h.Write(addr[:])
	h.Write(input)

	var key common.Hash
	h.Sum(key[:0])
	cacheKeyHasherPool.Put(h)
	return key
}
