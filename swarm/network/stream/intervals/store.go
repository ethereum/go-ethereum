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

// Package intervals TODO: implement LevelDB based Store.
package intervals

import (
	"errors"
	"sync"
)

// ErrNotFound is returned by the Store implementation when the Interval
// for a specific key does not exist.
var ErrNotFound = errors.New("not found")

// Store defines methods required to get and retrieve Intervals for different keys.
// It is meant to be used for intervals persistance for different streams in the
// stream package.
type Store interface {
	Get(key string) (i *Intervals, err error)
	Put(key string, i *Intervals) (err error)
	Delete(key string) (err error)
}

// MemStore is the reference implementation of Store interface that is supposed
// to be used in tests.
type MemStore struct {
	db map[string]*Intervals
	mu sync.RWMutex
}

// NewMemStore returns a new instance of MemStore.
func NewMemStore() *MemStore {
	return &MemStore{
		db: make(map[string]*Intervals),
	}
}

// Get retrieves Intervals for a specific key. If there is no Intervals
// ErrNotFound is returned.
func (s *MemStore) Get(key string) (i *Intervals, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	i, ok := s.db[key]
	if !ok {
		return nil, ErrNotFound
	}
	return i, nil
}

// Put stores Intervals for a specific key.
func (s *MemStore) Put(key string, i *Intervals) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.db[key] = i
	return nil
}

// Delete removes Intervals stored under a specific key.
func (s *MemStore) Delete(key string) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.db[key]; !ok {
		return ErrNotFound
	}
	delete(s.db, key)
	return nil
}
