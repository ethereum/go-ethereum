// Copyright 2022 The go-ethereum Authors
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

package lru

import (
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// SizeConstrainedLRU is a LRU cache where capacity is in bytes (instead of item count).
// When the cache is at capacity, and a new item is added, the older items are evicted
// until the size constraint can be met.
//
// OBS: This cache assumes that items are content-addressed: keys are unique per content.
// In other words: two Add(..) with the same key K, will always have the same value V.
type SizeConstrainedLRU struct {
	size    uint64
	maxSize uint64
	lru     Cache[common.Hash, []byte]
	lock    sync.Mutex
}

// NewSizeConstrainedLRU creates a new SizeConstrainedLRU.
func NewSizeConstrainedLRU(max uint64) *SizeConstrainedLRU {
	return &SizeConstrainedLRU{
		size:    0,
		maxSize: max,
		lru:     NewCache[common.Hash, []byte](math.MaxInt),
	}
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
// OBS: This cache assumes that items are content-addressed: keys are unique per content.
// In other words: two Add(..) with the same key K, will always have the same value V.
// OBS: The value is _not_ copied on Add, so the caller must not modify it afterwards.
func (c *SizeConstrainedLRU) Add(key common.Hash, value []byte) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Unless it is already present, might need to evict something.
	// OBS: If it is present, we still call Add internally to bump the recentness.
	if !c.lru.Contains(key) {
		targetSize := c.size + uint64(len(value))
		for targetSize > c.maxSize {
			evicted = true
			_, v, ok := c.lru.RemoveOldest()
			if !ok {
				// list is now empty. Break
				break
			}
			targetSize -= uint64(len(v))
		}
		c.size = targetSize
	}
	c.lru.Add(key, value)
	return evicted
}

// Get looks up a key's value from the cache.
func (c *SizeConstrainedLRU) Get(key common.Hash) []byte {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.lru.Get(key); ok {
		return v
	}
	return nil
}
