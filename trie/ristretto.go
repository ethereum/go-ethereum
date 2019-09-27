package trie

// Copyright 2019 The go-ethereum Authors
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

import (
	"errors"

	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/common"
)

var ErrMissingItem = errors.New("missing item")

type RistrettoCache struct {
	cache *ristretto.Cache
}

// NewRistrettoCache create a new ristretto cache with the given
// capacity in MB
func NewRistrettoCache(capacity int) (*RistrettoCache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(capacity * 10),
		MaxCost:     int64(1000 * 1000 * capacity),
		BufferItems: 64,
		Metrics:     false,
		KeyToHash: func(key interface{}) uint64 {
			h := key.(common.Hash)
			return uint64(h[7]) | uint64(h[6])<<8 | uint64(h[5])<<16 | uint64(h[4])<<24 |
				uint64(h[3])<<32 | uint64(h[2])<<40 | uint64(h[1])<<48 | uint64(h[0])<<56
		},
	})
	if err != nil {
		return nil, err
	}
	return &RistrettoCache{
		cache: cache,
	}, nil

}

func (c *RistrettoCache) Get(key common.Hash) ([]byte, error) {
	v, exist := c.cache.Get(key)
	if exist {
		return []byte(v.(string)), nil
	}
	return nil, ErrMissingItem
}

func (c *RistrettoCache) Set(key common.Hash, value []byte) error {
	c.cache.Set(key, string(value), int64(len(value)))
	return nil
}
