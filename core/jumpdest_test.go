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

// TestJumpDestCacheEntryOverhead checks that the per-shard byte budget charges
// what an entry actually costs to hold. The cache bills the bitmap value
// itself and the key plus the fixed per-entry cost, so a budget filled with
// small bitmaps (the common case: contracts are mostly a few hundred bytes,
// and 100 bytes of code produce a 12-byte bitmap) must not silently hold many
// times its stated size. The bound is wide on purpose: it catches the
// per-entry charge being wrong by an order of magnitude rather than pinning a
// number that moves with the Go version.
func TestJumpDestCacheEntryOverhead(t *testing.T) {
	const entries = 20_000

	// Fill the same cache structure the jumpdest cache uses, keyed by code
	// hash and holding bitmaps for 100-byte contracts. The budget is
	// unreachable so that every entry added is still held at the second
	// reading.
	c := &shardedJumpDestCache{}
	for i := range c.buckets {
		c.buckets[i].dest = lru.NewSizeConstrainedCacheWithKeySize[common.Hash, vm.BitVec](jumpDestBucketSize, jumpDestEntrySize)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := range entries {
		var hash common.Hash
		// hash[0] is zero, so every fixture lands in bucket 0 of the sharded
		// cache and the billed size of that one bucket covers all entries.
		hash[1] = byte(i >> 8)
		hash[2] = byte(i)
		c.Store(hash, make(vm.BitVec, 100/8))
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
