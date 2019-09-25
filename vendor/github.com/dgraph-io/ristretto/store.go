/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ristretto

import (
	"sync"
)

// store is the interface fulfilled by all hash map implementations in this
// file. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every store is safe for concurrent usage.
type store interface {
	// Get returns the value associated with the key parameter.
	Get(uint64) (interface{}, bool)
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set(uint64, interface{})
	// Del deletes the key-value pair from the Map.
	Del(uint64)
}

// newStore returns the default store implementation.
func newStore() store {
	// return newSyncMap()
	return newShardedMap()
}

type syncMap struct {
	*sync.Map
}

func newSyncMap() store {
	return &syncMap{&sync.Map{}}
}

func (m *syncMap) Get(key uint64) (interface{}, bool) {
	return m.Load(key)
}

func (m *syncMap) Set(key uint64, value interface{}) {
	m.Store(key, value)
}

func (m *syncMap) Del(key uint64) {
	m.Delete(key)
}

const numShards uint64 = 256

type shardedMap struct {
	shards []*lockedMap
}

func newShardedMap() *shardedMap {
	sm := &shardedMap{shards: make([]*lockedMap, int(numShards))}
	for i := range sm.shards {
		sm.shards[i] = newLockedMap()
	}
	return sm
}

func (sm *shardedMap) Get(key uint64) (interface{}, bool) {
	idx := key % numShards
	return sm.shards[idx].Get(key)
}

func (sm *shardedMap) Set(key uint64, value interface{}) {
	idx := key % numShards
	sm.shards[idx].Set(key, value)
}

func (sm *shardedMap) Del(key uint64) {
	idx := key % numShards
	sm.shards[idx].Del(key)
}

type lockedMap struct {
	sync.RWMutex
	data map[uint64]interface{}
}

func newLockedMap() *lockedMap {
	return &lockedMap{data: make(map[uint64]interface{})}
}

func (m *lockedMap) Get(key uint64) (interface{}, bool) {
	m.RLock()
	defer m.RUnlock()
	val, found := m.data[key]
	return val, found
}

func (m *lockedMap) Set(key uint64, value interface{}) {
	m.Lock()
	defer m.Unlock()
	m.data[key] = value
}

func (m *lockedMap) Del(key uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.data, key)
}
