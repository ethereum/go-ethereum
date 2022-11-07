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
	"github.com/hashicorp/golang-lru/simplelru"
)

// SizeConstrainedLRU is a wrapper around simplelru.LRU. The simplelru.LRU is capable
// of item-count constraints, but is not capable of enforcing a byte-size constraint,
// hence this wrapper.
// OBS: This cache assumes that items are content-addressed: keys are unique per content.
// In other words: two Add(..) with the same key K, will always have the same value V.
type SizeConstrainedLRU struct {
	size    uint64
	maxSize uint64
	lru     *simplelru.LRU
	lock    sync.RWMutex
}

// NewSizeConstrainedLRU creates a new SizeConstrainedLRU.
func NewSizeConstrainedLRU(max uint64) *SizeConstrainedLRU {
	lru, err := simplelru.NewLRU(math.MaxInt, nil)
	if err != nil {
		panic(err)
	}
	return &SizeConstrainedLRU{
		size:    0,
		maxSize: max,
		lru:     lru,
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
			targetSize -= uint64(len(v.([]byte)))
		}
		c.size = targetSize
	}
	c.lru.Add(key, value)
	return evicted
}

// Get looks up a key's value from the cache.
func (c *SizeConstrainedLRU) Get(key common.Hash) []byte {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if v, ok := c.lru.Get(key); ok {
		return v.([]byte)
	}
	return nil
}
