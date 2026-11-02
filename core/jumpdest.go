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
	"hash/maphash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	jumpDestHitMeter  = metrics.NewRegisteredMeter("chain/cache/jumpdest/hit", nil)
	jumpDestMissMeter = metrics.NewRegisteredMeter("chain/cache/jumpdest/miss", nil)
)

const (
	// jumpDestBuckets is the number of independent LRU shards. Code hashes
	// are dispatched by a randomly seeded hash to spread load across shards
	// and reduce mutex contention from the parallel prefetcher. The seed keeps
	// the shard unpredictable, since code hashes can be ground cheaply to make
	// many contracts compete for a single shard.
	jumpDestBuckets = 8

	// jumpDestBucketSize is the per-shard byte budget.
	jumpDestBucketSize = 8 * 1024 * 1024
)

// perEntryOverhead approximates an entry's map/list cost beyond its key and
// value bytes, measured at ~137-147 B for this layout. Erring high caps the
// entry count; eviction churn can reach ~1.5x, so the budget is a soft cap.
const perEntryOverhead = 150

// jumpDestEntrySize charges an entry's key plus its overhead; the value is
// counted separately and the charge refunded on eviction.
func jumpDestEntrySize(key common.Hash) uint64 {
	return uint64(len(key)) + perEntryOverhead
}

// shardedJumpDestCache is a thread-safe, byte-bounded LRU of JUMPDEST analysis
// bitmaps, sharded into independent buckets to reduce lock contention. It is
// owned by BlockChain and shared across block processing and prefetching,
// keyed by the immutable contract code hash.
type shardedJumpDestCache struct {
	seed    maphash.Seed
	buckets [jumpDestBuckets]struct {
		dest *lru.SizeConstrainedCache[common.Hash, vm.BitVec]
	}
}

// NewJumpDestCache constructs the analysis cache.
func NewJumpDestCache() vm.JumpDestCache {
	c := &shardedJumpDestCache{seed: maphash.MakeSeed()}
	for i := range c.buckets {
		c.buckets[i].dest = lru.NewSizeConstrainedCacheWithKeySize[common.Hash, vm.BitVec](jumpDestBucketSize, jumpDestEntrySize)
	}
	return c
}

// bucket returns the shard responsible for the given code hash.
func (c *shardedJumpDestCache) bucket(hash common.Hash) *lru.SizeConstrainedCache[common.Hash, vm.BitVec] {
	return c.buckets[maphash.Bytes(c.seed, hash[:])%jumpDestBuckets].dest
}

// Load retrieves the cached jumpdest analysis for the given code hash.
func (c *shardedJumpDestCache) Load(hash common.Hash) (vm.BitVec, bool) {
	v, ok := c.bucket(hash).Get(hash)
	if ok {
		jumpDestHitMeter.Mark(1)
	} else {
		jumpDestMissMeter.Mark(1)
	}
	return v, ok
}

// Store saves the jumpdest analysis for the given code hash.
func (c *shardedJumpDestCache) Store(hash common.Hash, b vm.BitVec) {
	c.bucket(hash).Add(hash, b)
}
