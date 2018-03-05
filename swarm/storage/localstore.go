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

package storage

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

var (
	dbStorePutCounter = metrics.NewRegisteredCounter("storage.db.dbstore.put.count", nil)
)

type StoreParams struct {
	ChunkDbPath   string
	DbCapacity    uint64
	CacheCapacity uint
	Radius        int
}

//create params with default values
func NewDefaultStoreParams() (self *StoreParams) {
	return &StoreParams{
		DbCapacity:    defaultDbCapacity,
		CacheCapacity: defaultCacheCapacity,
	}
}

// LocalStore is a combination of inmemory db over a disk persisted db
// implements a Get/Put with fallback (caching) logic using any 2 ChunkStores
type LocalStore struct {
	memStore ChunkStore
	DbStore  ChunkStore
}

// This constructor uses MemStore and DbStore as components
func NewLocalStore(hash SwarmHasher, params *StoreParams, basekey []byte, mockStore *mock.NodeStore) (*LocalStore, error) {
	dbStore, err := NewMockDbStore(params.ChunkDbPath, hash, params.DbCapacity, func(k Key) (ret uint8) { return uint8(Proximity(basekey[:], k[:])) }, mockStore)
	if err != nil {
		return nil, err
	}
	return &LocalStore{
		memStore: NewMemStore(dbStore, params.CacheCapacity),
		DbStore:  dbStore,
	}, nil
}

func NewTestLocalStore(path string) (*LocalStore, error) {
	basekey := make([]byte, 32)
	hasher := MakeHashFunc("SHA3")
	dbStore, err := NewLDBStore(path, hasher, singletonSwarmDbCapacity, func(k Key) (ret uint8) { return uint8(Proximity(basekey[:], k[:])) })
	if err != nil {
		return nil, err
	}
	localStore := &LocalStore{
		memStore: NewMemStore(dbStore, singletonSwarmDbCapacity),
		DbStore:  dbStore,
	}
	return localStore, nil
}

func NewTestLocalStoreForAddr(path string, basekey []byte) (*LocalStore, error) {
	hasher := MakeHashFunc("SHA3")
	dbStore, err := NewLDBStore(path, hasher, singletonSwarmDbCapacity, func(k Key) (ret uint8) { return uint8(Proximity(basekey[:], k[:])) })
	if err != nil {
		return nil, err
	}
	localStore := &LocalStore{
		memStore: NewMemStore(dbStore, singletonSwarmDbCapacity),
		DbStore:  dbStore,
	}
	return localStore, nil
}

func (self *LocalStore) CacheCounter() uint64 {
	return uint64(self.memStore.(*MemStore).Counter())
}

// LocalStore is itself a chunk store
// unsafe, in that the data is not integrity checked
func (self *LocalStore) Put(chunk *Chunk) {
	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
	c := &Chunk{
		Key:      Key(append([]byte{}, chunk.Key...)),
		SData:    append([]byte{}, chunk.SData...),
		Size:     chunk.Size,
		dbStored: chunk.dbStored,
	}

	dbStorePutCounter.Inc(1)
	self.memStore.Put(c)
	self.DbStore.Put(c)
}

// Get(chunk *Chunk) looks up a chunk in the local stores
// This method is blocking until the chunk is retrieved
// so additional timeout may be needed to wrap this call if
// ChunkStores are remote and can have long latency
func (self *LocalStore) Get(key Key) (chunk *Chunk, err error) {
	chunk, err = self.memStore.Get(key)
	if err == nil {
		if chunk.ReqC != nil {
			select {
			case <-chunk.ReqC:
			default:
				return chunk, ErrFetching
			}
		}
		return
	}
	chunk, err = self.DbStore.Get(key)
	if err != nil {
		return
	}
	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
	self.memStore.Put(chunk)
	return
}

// retrieve logic common for local and network chunk retrieval requests
func (self *LocalStore) GetOrCreateRequest(key Key) (chunk *Chunk, created bool) {
	var err error
	chunk, err = self.Get(key)
	if err == nil {
		log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v found locally", key))
		return chunk, false
	}
	if err == ErrFetching {
		log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v hit on an existing request %v", key, chunk.ReqC))
		return chunk, false
	}
	// no data and no request status
	log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v not found locally. open new request", key))
	chunk = NewChunk(key, make(chan bool))
	self.memStore.Put(chunk)
	return chunk, true
}

// Close the local store
func (self *LocalStore) Close() {
	self.DbStore.Close()
}
