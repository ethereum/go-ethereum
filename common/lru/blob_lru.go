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

	"github.com/hashicorp/golang-lru/simplelru"
)

// SizeConstrainedLRU is a wrapper around simplelru.LRU, which
// adds a byte-size constraint.
type SizeConstrainedLRU struct {
	size    uint64
	maxSize uint64
	lru     *simplelru.LRU
}

// NewSizeConstraiedLRU creates a new SizeConstrainedLRU.
func NewSizeConstraiedLRU(max uint64) *SizeConstrainedLRU {
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

// Set adds a value to the cache.  Returns true if an eviction occurred.
func (c *SizeConstrainedLRU) Set(key []byte, value []byte) (evicted bool) {
	return c.Add(string(key), string(value))
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *SizeConstrainedLRU) Add(key string, value string) (evicted bool) {
	targetSize := c.size + uint64(len(value))
	for targetSize > c.maxSize {
		evicted = true
		_, v, ok := c.lru.RemoveOldest()
		if !ok {
			// list is now empty. Break
			break
		}
		targetSize -= uint64(len(v.(string)))
	}
	c.size = targetSize
	c.lru.Add(key, value)
	return evicted
}

// Get looks up a key's value from the cache.
func (c *SizeConstrainedLRU) Get(key []byte) []byte {
	if v, ok := c.lru.Get(string(key)); ok {
		return []byte(v.(string))
	}
	return nil
}
