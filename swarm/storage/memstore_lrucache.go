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
	"bytes"
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

const (
	defaultCacheCapacity = 5000
)

type MemStore struct {
	cache    *lru.Cache
	requests *lru.Cache
	mu       sync.Mutex
	disabled bool
}

func NewMemStore(_ *LDBStore, cacheCapacity uint, requestsCapacity uint) (m *MemStore) {
	if cacheCapacity == 0 {
		return &MemStore{
			disabled: true,
		}
	}

	onEvicted := func(key interface{}, value interface{}) {
		v := value.(*Chunk)
		<-v.dbStoredC
	}
	c, err := lru.NewWithEvict(int(cacheCapacity), onEvicted)
	if err != nil {
		panic(err)
	}

	r, err := lru.New(int(requestsCapacity))
	if err != nil {
		panic(err)
	}

	return &MemStore{
		cache:    c,
		requests: r,
	}
}

func (m *MemStore) Get(key Key) (*Chunk, error) {
	if m.disabled {
		return nil, ErrChunkNotFound
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.requests.Get(string(key))
	// it is a request
	if ok {
		return r.(*Chunk), nil
	}

	// it is not a request
	c, ok := m.cache.Get(string(key))
	if !ok {
		return nil, ErrChunkNotFound
	}
	chunk := c.(*Chunk)
	if !bytes.Equal(chunk.Key, key) {
		panic(fmt.Errorf("MemStore.Get: chunk key %s != req key %s", chunk.Key.Hex(), key.Hex()))
	}
	return chunk, nil
}

func (m *MemStore) Put(c *Chunk) {
	if m.disabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// it is a request
	if c.ReqC != nil {
		select {
		case <-c.ReqC:
			ok := c.GetErrored()
			if !ok {
				m.requests.Remove(string(c.Key))
				return
			}
			m.cache.Add(string(c.Key), c)
			m.requests.Remove(string(c.Key))
		default:
			m.requests.Add(string(c.Key), c)
		}
		return
	}

	// it is not a request
	m.cache.Add(string(c.Key), c)
	m.requests.Remove(string(c.Key))
}

func (m *MemStore) setCapacity(n int) {
	if n <= 0 {
		m.disabled = true
	} else {
		m = NewMemStore(nil, uint(n), singletonSwarmDbCapacity)
	}
}

func (s *MemStore) Close() {}
