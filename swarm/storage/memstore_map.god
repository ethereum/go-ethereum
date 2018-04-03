// Copyright 2016 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/metrics"
)

//metrics variables
var (
	memstorePutCounter    = metrics.NewRegisteredCounter("storage.db.memstore.put.count", nil)
	memstoreRemoveCounter = metrics.NewRegisteredCounter("storage.db.memstore.rm.count", nil)
)

const (
	memTreeLW              = 2  // log2(subtree count) of the subtrees
	memTreeFLW             = 14 // log2(subtree count) of the root layer
	dbForceUpdateAccessCnt = 1000
	defaultCacheCapacity   = 5000
)

type MemStore struct {
	m  map[string]*Chunk
	mu sync.RWMutex
}

func NewMemStore(d *LDBStore, capacity uint) (m *MemStore) {
	return &MemStore{
		m: make(map[string]*Chunk),
	}
}

func (m *MemStore) Get(key Key) (*Chunk, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.m[string(key[:])]
	if !ok {
		return nil, ErrChunkNotFound
	}
	if !bytes.Equal(c.Key, key) {
		panic(fmt.Errorf("MemStore.Get: chunk key %s != req key %s", c.Key.Hex(), key.Hex()))
	}
	return c, nil
}

func (m *MemStore) Put(c *Chunk) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[string(c.Key[:])] = c
}

func (m *MemStore) setCapacity(n int) {

}

// Close memstore
func (s *MemStore) Close() {}
