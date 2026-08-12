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
)

// blobType is the type constraint for values stored in SizeConstrainedCache.
type blobType interface {
	~[]byte | ~string
}

// SizeConstrainedCache is a cache where capacity is in bytes (instead of item count). When the cache
// is at capacity, and a new item is added, older items are evicted until the size
// constraint is met.
//
// By default only the value bytes count towards the size constraint. If the key
// carries a meaningful share of an entry's memory, pass a key-size function to
// NewSizeConstrainedCacheWithKeySize so it is accounted too.
//
// OBS: This cache assumes that items are content-addressed: keys are unique per content.
// In other words: two Add(..) with the same key K, will always have the same value V.
type SizeConstrainedCache[K comparable, V blobType] struct {
	size    uint64
	maxSize uint64
	keySize func(K) uint64
	lru     BasicLRU[K, V]
	lock    sync.Mutex
}

// NewSizeConstrainedCache creates a new size-constrained LRU cache in which only
// the value bytes count towards the size constraint.
func NewSizeConstrainedCache[K comparable, V blobType](maxSize uint64) *SizeConstrainedCache[K, V] {
	return NewSizeConstrainedCacheWithKeySize[K, V](maxSize, nil)
}

// NewSizeConstrainedCacheWithKeySize is like NewSizeConstrainedCache but also
// charges each entry's key against the size constraint, measured by keySize.
// This matters when the key, not the value, holds the bulk of an entry's memory.
// A nil keySize counts keys as free, identical to NewSizeConstrainedCache.
func NewSizeConstrainedCacheWithKeySize[K comparable, V blobType](maxSize uint64, keySize func(K) uint64) *SizeConstrainedCache[K, V] {
	return &SizeConstrainedCache[K, V]{
		size:    0,
		maxSize: maxSize,
		keySize: keySize,
		lru:     NewBasicLRU[K, V](math.MaxInt),
	}
}

// entrySize reports the number of bytes an entry counts for against the size
// constraint: the value length, plus the key size when a measure was provided.
func (c *SizeConstrainedCache[K, V]) entrySize(key K, value V) uint64 {
	n := uint64(len(value))
	if c.keySize != nil {
		n += c.keySize(key)
	}
	return n
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
// OBS: This cache assumes that items are content-addressed: keys are unique per content.
// In other words: two Add(..) with the same key K, will always have the same value V.
// OBS: The value is _not_ copied on Add, so the caller must not modify it afterwards.
func (c *SizeConstrainedCache[K, V]) Add(key K, value V) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Unless it is already present, might need to evict something.
	// OBS: If it is present, we still call Add internally to bump the recentness.
	if !c.lru.Contains(key) {
		targetSize := c.size + c.entrySize(key, value)
		for targetSize > c.maxSize {
			evicted = true
			k, v, ok := c.lru.RemoveOldest()
			if !ok {
				// list is now empty. Break
				break
			}
			targetSize -= c.entrySize(k, v)
		}
		c.size = targetSize
	}
	c.lru.Add(key, value)
	return evicted
}

// Get looks up a key's value from the cache.
func (c *SizeConstrainedCache[K, V]) Get(key K) (V, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.lru.Get(key)
}
