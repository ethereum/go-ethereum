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
	"bytes"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	// smallBitmapLen is the JUMPDEST bitmap size of a ~100 byte contract, the
	// case that makes the per-entry bookkeeping dominate the payload.
	smallBitmapLen = 17

	// measuredEntryFootprint is the real heap cost of one cached entry holding
	// a smallBitmapLen bitmap: the 32 byte key, the value, and the map and list
	// bookkeeping around them, measured at 186-188 B for this layout. It is
	// deliberately written out rather than derived from perEntryOverhead, so
	// these tests bound the memory actually held rather than restating whatever
	// the cache has chosen to bill.
	measuredEntryFootprint = 186

	// minEntryOverhead is the low end of the bookkeeping an entry costs beyond
	// its key and value bytes.
	minEntryOverhead = 137
)

// newSizedJumpDestCache builds a cache shaped like the production one but with
// a caller supplied per-shard budget, so eviction can be exercised without
// allocating jumpDestBucketSize per bucket.
func newSizedJumpDestCache(bucketSize uint64) *shardedJumpDestCache {
	c := new(shardedJumpDestCache)
	for i := range c.buckets {
		c.buckets[i].dest = lru.NewSizeConstrainedCacheWithKeySize[common.Hash, vm.BitVec](bucketSize, jumpDestEntrySize)
	}
	return c
}

// jumpDestKey returns a unique code hash that Load and Store dispatch to the
// requested shard.
func jumpDestKey(bucket byte, n uint32) common.Hash {
	var h common.Hash
	h[0] = bucket & (jumpDestBuckets - 1)
	binary.BigEndian.PutUint32(h[1:5], n)
	return h
}

// bitmapOf returns a deterministic bitmap of the given length for a key.
func bitmapOf(n uint32, length int) vm.BitVec {
	b := make(vm.BitVec, length)
	for i := range b {
		b[i] = byte(n) ^ byte(i)
	}
	return b
}

// TestJumpDestEntrySize checks that an entry is charged for its key and for
// the bookkeeping around it, not merely for the bitmap it carries.
func TestJumpDestEntrySize(t *testing.T) {
	got := jumpDestEntrySize(common.Hash{})
	if want := uint64(common.HashLength + minEntryOverhead); got < want {
		t.Fatalf("entry size charge = %d, want at least %d (key %d + bookkeeping %d)",
			got, want, common.HashLength, minEntryOverhead)
	}
	// The charge must not depend on the key's contents, only its length.
	if other := jumpDestEntrySize(jumpDestKey(3, 42)); other != got {
		t.Fatalf("entry size charge varies with key contents: %d != %d", other, got)
	}
}

// TestJumpDestCacheStoreLoad covers the round trip and the miss path.
func TestJumpDestCacheStoreLoad(t *testing.T) {
	cache := NewJumpDestCache()

	key := jumpDestKey(0, 1)
	if _, ok := cache.Load(key); ok {
		t.Fatal("empty cache reported a hit")
	}
	want := bitmapOf(1, smallBitmapLen)
	cache.Store(key, want)

	got, ok := cache.Load(key)
	if !ok {
		t.Fatal("stored bitmap not found")
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("loaded bitmap = %x, want %x", got, want)
	}
	// A key that was never stored must still miss, and miss cleanly.
	if got, ok := cache.Load(jumpDestKey(0, 2)); ok || got != nil {
		t.Fatalf("unknown key returned (%x, %v), want (nil, false)", got, ok)
	}
}

// TestJumpDestCacheBudget is the regression test for the entry overhead not
// being billed. Small bitmaps are dominated by their bookkeeping, so a budget
// that only counts value bytes retains far more entries than it advertises.
func TestJumpDestCacheBudget(t *testing.T) {
	const budget = 64 * 1024

	cache := newSizedJumpDestCache(budget)

	// Fill a single shard well past its budget.
	const inserted = 4000
	for i := uint32(0); i < inserted; i++ {
		cache.Store(jumpDestKey(0, i), bitmapOf(i, smallBitmapLen))
	}
	// Every shard must respect its byte budget, including the untouched ones.
	for i := range cache.buckets {
		if size := cache.buckets[i].dest.Size(); size > budget {
			t.Fatalf("bucket %d holds %d bytes, over the %d budget", i, size, budget)
		}
	}
	// The retained entry count is what the overhead charge actually bounds.
	// Billing the value bytes alone lets the shard hold several times the heap
	// its budget describes, which is the overshoot this guards against.
	var retained int
	for i := uint32(0); i < inserted; i++ {
		if _, ok := cache.Load(jumpDestKey(0, i)); ok {
			retained++
		}
	}
	if retained == 0 {
		t.Fatal("shard retained nothing")
	}
	if held := uint64(retained) * measuredEntryFootprint; held > budget {
		t.Fatalf("shard retains %d entries holding ~%d bytes of heap against a %d byte budget (%.1fx overshoot)",
			retained, held, budget, float64(held)/budget)
	}
	// The survivors must be the most recently stored ones.
	for i := inserted - uint32(retained); i < inserted; i++ {
		if _, ok := cache.Load(jumpDestKey(0, i)); !ok {
			t.Fatalf("recently stored entry %d was evicted while %d entries were kept", i, retained)
		}
	}
}

// TestJumpDestCacheEvictionRefund checks that the charge taken on insert is
// handed back on eviction, so the accounted size cannot drift upwards over a
// long run of insertions.
func TestJumpDestCacheEvictionRefund(t *testing.T) {
	const budget = 8 * 1024

	cache := newSizedJumpDestCache(budget)
	for i := uint32(0); i < 5000; i++ {
		cache.Store(jumpDestKey(0, i), bitmapOf(i, smallBitmapLen))
		if size := cache.buckets[0].dest.Size(); size > budget {
			t.Fatalf("after %d stores the bucket holds %d bytes, over the %d budget", i+1, size, budget)
		}
	}
	// Re-storing a key already present must not be charged twice.
	key := jumpDestKey(0, 4999)
	before := cache.buckets[0].dest.Size()
	for range 100 {
		cache.Store(key, bitmapOf(4999, smallBitmapLen))
	}
	if after := cache.buckets[0].dest.Size(); after != before {
		t.Fatalf("re-storing an existing key changed the size from %d to %d", before, after)
	}
}

// TestJumpDestCacheSharding verifies that code hashes are dispatched by the low
// bits of their first byte, and that shards evict independently.
func TestJumpDestCacheSharding(t *testing.T) {
	const budget = 8 * 1024

	cache := newSizedJumpDestCache(budget)

	// One entry per shard, then flood shard 0 only.
	for b := byte(0); b < jumpDestBuckets; b++ {
		cache.Store(jumpDestKey(b, 0), bitmapOf(uint32(b), smallBitmapLen))
	}
	for i := uint32(1); i < 2000; i++ {
		cache.Store(jumpDestKey(0, i), bitmapOf(i, smallBitmapLen))
	}
	// Shard 0's original entry is long gone, the others are untouched.
	if _, ok := cache.Load(jumpDestKey(0, 0)); ok {
		t.Fatal("flooded shard still holds its oldest entry")
	}
	for b := byte(1); b < jumpDestBuckets; b++ {
		got, ok := cache.Load(jumpDestKey(b, 0))
		if !ok {
			t.Fatalf("entry in shard %d was evicted by traffic on shard 0", b)
		}
		if want := bitmapOf(uint32(b), smallBitmapLen); !bytes.Equal(got, want) {
			t.Fatalf("shard %d returned %x, want %x", b, got, want)
		}
	}
	// Hashes that differ only above the shard bits share a shard.
	if cache.buckets[0].dest.Size() == 0 {
		t.Fatal("shard 0 is empty after flooding it")
	}
	for b := byte(jumpDestBuckets); b < 2*jumpDestBuckets; b++ {
		key := jumpDestKey(b, 7)
		cache.Store(key, bitmapOf(7, smallBitmapLen))
		if _, ok := cache.Load(jumpDestKey(b&(jumpDestBuckets-1), 7)); !ok {
			t.Fatalf("hash prefix %d did not map onto shard %d", b, b&(jumpDestBuckets-1))
		}
	}
}

// TestJumpDestCacheConcurrent exercises the cache the way block processing and
// the parallel prefetcher share it. Meaningful under -race.
func TestJumpDestCacheConcurrent(t *testing.T) {
	cache := NewJumpDestCache()

	const (
		workers = 8
		keys    = 512
	)
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := uint32(0); i < keys; i++ {
				key := jumpDestKey(byte(i), i)
				cache.Store(key, bitmapOf(i, smallBitmapLen))
				if got, ok := cache.Load(key); ok {
					if want := bitmapOf(i, smallBitmapLen); !bytes.Equal(got, want) {
						t.Errorf("worker %d read a torn bitmap for key %d: %x, want %x", w, i, got, want)
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()

	// Everything written fits in the production budget, so nothing was evicted.
	for i := uint32(0); i < keys; i++ {
		got, ok := cache.Load(jumpDestKey(byte(i), i))
		if !ok {
			t.Fatalf("key %d missing after concurrent writes", i)
		}
		if want := bitmapOf(i, smallBitmapLen); !bytes.Equal(got, want) {
			t.Fatalf("key %d = %x, want %x", i, got, want)
		}
	}
}
