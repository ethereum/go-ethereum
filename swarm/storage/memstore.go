// Copyright 2018 The go-ethereum Authors
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

// memory storage layer for the package blockhash

package storage

import (
	"context"

	lru "github.com/hashicorp/golang-lru"
)

type MemStore struct {
	cache    *lru.Cache
	disabled bool
}

//NewMemStore is instantiating a MemStore cache keeping all frequently requested
//chunks in the `cache` LRU cache.
func NewMemStore(params *StoreParams, _ *LDBStore) (m *MemStore) {
	if params.CacheCapacity == 0 {
		return &MemStore{
			disabled: true,
		}
	}

	c, err := lru.New(int(params.CacheCapacity))
	if err != nil {
		panic(err)
	}

	return &MemStore{
		cache: c,
	}
}

func (m *MemStore) Get(_ context.Context, addr Address) (Chunk, error) {
	if m.disabled {
		return nil, ErrChunkNotFound
	}

	c, ok := m.cache.Get(string(addr))
	if !ok {
		return nil, ErrChunkNotFound
	}
	return c.(Chunk), nil
}

func (m *MemStore) Put(_ context.Context, c Chunk) error {
	if m.disabled {
		return nil
	}

	m.cache.Add(string(c.Address()), c)
	return nil
}

func (m *MemStore) setCapacity(n int) {
	if n <= 0 {
		m.disabled = true
	} else {
		c, err := lru.New(n)
		if err != nil {
			panic(err)
		}

		*m = MemStore{
			cache: c,
		}
	}
}

func (s *MemStore) Close() {}
