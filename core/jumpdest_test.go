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
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TestJumpDestCacheEntryOverhead checks that the per-shard budget bills what
// an entry actually costs, so small bitmaps (100B of code yields a 17-byte
// bitmap) cannot silently multiply the budget. The wide bound catches
// order-of-magnitude undercharging.
func TestJumpDestCacheEntryOverhead(t *testing.T) {
	const entries = 20_000

	// Fill the jumpdest cache structure with the budget unreachable, so every
	// entry is still held at the second reading.
	c := &shardedJumpDestCache{}
	for i := range c.buckets {
		c.buckets[i].dest = lru.NewSizeConstrainedCacheWithKeySize[common.Hash, vm.BitVec](jumpDestBucketSize, jumpDestEntrySize)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := range entries {
		var hash common.Hash
		// A zero hash[0] keeps every entry in bucket 0, so one bucket's
		// billed size covers all of them.
		hash[1] = byte(i >> 8)
		hash[2] = byte(i)
		c.Store(hash, make(vm.BitVec, 100/8+1+4))
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(c)

	// The whole fixture lands in a single shard (hash[0] is zero), so the
	// bucket's billed size is the charge for all entries.
	billed := c.buckets[0].dest.Size()
	charged := int64(billed) / entries
	held := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	actual := held / entries
	if charged < actual/2 {
		t.Errorf("%d entries: budget charges %d bytes each, holding one costs %d bytes", entries, charged, actual)
	} else {
		t.Logf("%d entries: budget charges %d bytes each, holding one costs %d bytes", entries, charged, actual)
	}
}

// TestJumpDestCacheEvictsWithinBudget checks that once the billed size passes
// the budget, the cache actually evicts the oldest entries instead of growing
// past it.
func TestJumpDestCacheEvictsWithinBudget(t *testing.T) {
	const budget = 1024 // ~5 entries at 199 B each
	c := lru.NewSizeConstrainedCacheWithKeySize[common.Hash, vm.BitVec](budget, jumpDestEntrySize)

	var witness common.Hash
	witness[31] = 0x7f
	c.Add(witness, make(vm.BitVec, 100/8+1+4))

	for i := 0; i < 100; i++ {
		var key common.Hash
		key[0] = byte(i)
		c.Add(key, make(vm.BitVec, 100/8+1+4))
	}
	if sz := c.Size(); sz > budget {
		t.Fatalf("cache size %d exceeds budget %d", sz, budget)
	}
	if _, ok := c.Get(witness); ok {
		t.Fatal("unbounded cache: oldest entry survived past the budget")
	}
	var last common.Hash
	last[0] = 99
	if _, ok := c.Get(last); !ok {
		t.Fatal("most recent entry unexpectedly evicted")
	}
}
