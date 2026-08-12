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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	precompileCacheHitMeter          = metrics.NewRegisteredMeter("chain/cache/precompile/hit", nil)
	precompileCacheMissMeter         = metrics.NewRegisteredMeter("chain/cache/precompile/miss", nil)
	precompileCachePrefetchHitMeter  = metrics.NewRegisteredMeter("chain/cache/precompile/prefetch/hit", nil)
	precompileCachePrefetchMissMeter = metrics.NewRegisteredMeter("chain/cache/precompile/prefetch/miss", nil)
)

const (
	// maxCacheablePrecompileInput bounds the normalized input size eligible for
	// result caching. The key is the input itself, so this also bounds how much
	// an entry can cost.
	maxCacheablePrecompileInput = 8192

	// maxCacheablePrecompileOutput bounds the output size stored in the
	// cache, keeping the worst case memory use of an entry small.
	maxCacheablePrecompileOutput = 1024

	// maxCacheablePrecompileBytes is the budget each precompile gets per fork,
	// counting keys and values along with what an entry costs to hold. Entries
	// run from tens of bytes to kilobytes, so a budget in entries would mean
	// very different memory depending on the mix.
	maxCacheablePrecompileBytes = 1024 * 1024
)

// PrecompileCache is a thread-safe cache of precompile outputs, shared between
// the state prefetcher and block processing so the serial pass can reuse what
// the prefetcher already computed. Each precompile gets its own cache per fork,
// so results never cross a repricing and a cheap precompile cannot evict the
// results of an expensive one.
type PrecompileCache struct {
	data *precompileCacheData

	// Meters are per handle, split between the main pass and the prefetcher
	// so the hit rate of the main pass stays readable on its own.
	prefix   string
	hit      *metrics.Meter
	miss     *metrics.Meter
	mu       sync.RWMutex
	meters   map[common.Address]*precompileCacheMeters
	prefetch *PrecompileCache
}

// precompileCacheData is the storage shared by the two cache handles.
type precompileCacheData struct {
	mu     sync.RWMutex
	caches map[precompileCacheScope]*lru.SizeConstrainedCache[string, []byte]
}

// precompileCacheScope identifies the cache of one precompile at one fork. The
// set pointer keeps forks apart, the address keeps precompiles apart.
type precompileCacheScope struct {
	set  *PrecompiledContracts
	addr common.Address
}

// precompileCacheMeters holds the per-address hit and miss meters.
type precompileCacheMeters struct {
	hit  *metrics.Meter
	miss *metrics.Meter
}

// NewPrecompileCache constructs a precompile result cache.
func NewPrecompileCache() *PrecompileCache {
	data := &precompileCacheData{
		caches: make(map[precompileCacheScope]*lru.SizeConstrainedCache[string, []byte]),
	}
	return &PrecompileCache{
		data:   data,
		prefix: "chain/cache/precompile",
		hit:    precompileCacheHitMeter,
		miss:   precompileCacheMissMeter,
		meters: make(map[common.Address]*precompileCacheMeters),

		prefetch: &PrecompileCache{
			data:   data,
			prefix: "chain/cache/precompile/prefetch",
			hit:    precompileCachePrefetchHitMeter,
			miss:   precompileCachePrefetchMissMeter,
			meters: make(map[common.Address]*precompileCacheMeters),
		},
	}
}

// PrefetchView returns a handle of the same cache that marks the prefetcher
// meters instead of the main pass ones.
func (c *PrecompileCache) PrefetchView() *PrecompileCache {
	if c == nil {
		return nil
	}
	return c.prefetch
}

// load retrieves the cached output for the given key. The returned slice is
// a private copy owned by the caller, entries cross goroutine boundaries.
func (c *PrecompileCache) load(scope precompileCacheScope, key []byte) ([]byte, bool) {
	c.data.mu.RLock()
	results := c.data.caches[scope]
	c.data.mu.RUnlock()

	meters := c.metersFor(scope.addr)
	if results != nil {
		if output, ok := results.Get(string(key)); ok {
			c.hit.Mark(1)
			meters.hit.Mark(1)
			return common.CopyBytes(output), true
		}
	}
	c.miss.Mark(1)
	meters.miss.Mark(1)
	return nil, false
}

// store saves the output of a precompile run under the given key. Both the key
// and the value are copied, the cache never aliases caller memory. That matters
// for the key in particular, it aliases the caller's memory which the EVM goes
// on to overwrite.
func (c *PrecompileCache) store(scope precompileCacheScope, key []byte, output []byte) {
	c.data.mu.RLock()
	results := c.data.caches[scope]
	c.data.mu.RUnlock()

	if results == nil {
		c.data.mu.Lock()
		if results = c.data.caches[scope]; results == nil {
			// The key is the precompile input, which dwarfs the output (up to
			// maxCacheablePrecompileInput against maxCacheablePrecompileOutput),
			// so it must count towards the budget. Without it an empty output
			// (e.g. a failed ECRECOVER) would let the cache grow without bound.
			results = lru.NewSizeConstrainedCacheWithKeySize[string, []byte](maxCacheablePrecompileBytes, func(k string) uint64 { return uint64(len(k)) })
			c.data.caches[scope] = results
		}
		c.data.mu.Unlock()
	}
	results.Add(string(key), common.CopyBytes(output))
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
	prefix := fmt.Sprintf("%s/%#x", c.prefix, addr.Big())
	meters = &precompileCacheMeters{
		hit:  metrics.GetOrRegisterMeter(prefix+"/hit", nil),
		miss: metrics.GetOrRegisterMeter(prefix+"/miss", nil),
	}
	c.meters[addr] = meters
	return meters
}

// CacheablePrecompile is implemented by precompiles that opt in to result
// caching. Anything that does not implement it is never cached, so a new
// precompile is not enrolled until someone decides it should be.
type CacheablePrecompile interface {
	Cacheable() bool
}

// NormalizingPrecompile is implemented by precompiles that can narrow an input
// down to the bytes that determine the result.
type NormalizingPrecompile interface {
	// NormalizeInput returns the bytes identifying the result, and whether the
	// invocation is cacheable at all. Two inputs that normalize alike share an
	// entry, so they must run to the same output. Returning false skips the
	// cache, which is how a precompile rejects lengths it will fail on.
	NormalizeInput(input []byte) ([]byte, bool)
}

// precompileCacheKey returns the key identifying an invocation and whether it
// is eligible for result caching.
func precompileCacheKey(p PrecompiledContract, input []byte) ([]byte, bool) {
	c, ok := p.(CacheablePrecompile)
	if !ok || !c.Cacheable() {
		return nil, false
	}
	key := input
	if n, ok := p.(NormalizingPrecompile); ok {
		if key, ok = n.NormalizeInput(input); !ok {
			return nil, false
		}
	}
	if len(key) > maxCacheablePrecompileInput {
		return nil, false
	}
	return key, true
}

// normalizeZeroPadded narrows an input for a precompile that reads a fixed
// length prefix and zero extends anything shorter. Bytes past the prefix are
// never read, and trailing zeros inside it read the same as not being there,
// so both can be dropped from the key.
func normalizeZeroPadded(input []byte, prefix int) []byte {
	if len(input) > prefix {
		input = input[:prefix]
	}
	end := len(input)
	for end > 0 && input[end-1] == 0 {
		end--
	}
	return input[:end]
}
