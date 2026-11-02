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

package metrics

import (
	"github.com/ethereum/go-ethereum/common/lru"
)

// MeteredCache is an LRU cache exposing hit metrics.
// This type is safe for concurrent use.
type MeteredCache[K comparable, V any] struct {
	lru.Cache[K, V]
	hit  *Meter
	miss *Meter
}

// NewMeteredCache creates a new metered LRU cache.
func NewMeteredCache[K comparable, V any](capacity int, name string, registry Registry) *MeteredCache[K, V] {
	return &MeteredCache[K, V]{
		Cache: *lru.NewCache[K, V](capacity),
		hit:   NewRegisteredMeter(name+"/hit", registry),
		miss:  NewRegisteredMeter(name+"/miss", registry),
	}
}

// Contains reports whether the given key exists in the cache.
func (c *MeteredCache[K, V]) Contains(key K) bool {
	ret := c.Cache.Contains(key)
	if c.hit != nil && c.miss != nil {
		if ret {
			c.hit.Mark(1)
		} else {
			c.miss.Mark(1)
		}
	}
	return ret
}

// Get retrieves a value from the cache. This marks the key as recently used.
func (c *MeteredCache[K, V]) Get(key K) (value V, ok bool) {
	ret, ok := c.Cache.Get(key)
	if c.hit != nil && c.miss != nil {
		if ok {
			c.hit.Mark(1)
		} else {
			c.miss.Mark(1)
		}
	}
	return ret, ok
}
