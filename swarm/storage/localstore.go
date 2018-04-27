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
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

var (
	dbStorePutCounter = metrics.NewRegisteredCounter("storage.db.dbstore.put.count", nil)
)

type LocalStoreParams struct {
	*StoreParams
	ChunkDbPath string
	Validators  []ChunkValidator `toml:"-"`
}

func NewDefaultLocalStoreParams() *LocalStoreParams {
	return &LocalStoreParams{
		StoreParams: NewDefaultStoreParams(),
	}
}

//this can only finally be set after all config options (file, cmd line, env vars)
//have been evaluated
func (self *LocalStoreParams) Init(path string) {
	if self.ChunkDbPath == "" {
		self.ChunkDbPath = filepath.Join(path, "chunks")
	}
}

// LocalStore is a combination of inmemory db over a disk persisted db
// implements a Get/Put with fallback (caching) logic using any 2 ChunkStores
type LocalStore struct {
	Validators []ChunkValidator
	memStore   *MemStore
	DbStore    *LDBStore
	mu         sync.Mutex
}

// This constructor uses MemStore and DbStore as components
func NewLocalStore(params *LocalStoreParams, mockStore *mock.NodeStore) (*LocalStore, error) {
	ldbparams := NewLDBStoreParams(params.StoreParams, params.ChunkDbPath)
	dbStore, err := NewMockDbStore(ldbparams, mockStore)
	if err != nil {
		return nil, err
	}
	return &LocalStore{
		memStore:   NewMemStore(params.StoreParams, dbStore),
		DbStore:    dbStore,
		Validators: params.Validators,
	}, nil
}

func NewTestLocalStoreForAddr(params *LocalStoreParams) (*LocalStore, error) {
	ldbparams := NewLDBStoreParams(params.StoreParams, params.ChunkDbPath)
	dbStore, err := NewLDBStore(ldbparams)
	if err != nil {
		return nil, err
	}
	localStore := &LocalStore{
		memStore:   NewMemStore(params.StoreParams, dbStore),
		DbStore:    dbStore,
		Validators: params.Validators,
	}
	return localStore, nil
}

// Put is responsible for doing validation and storage of the chunk
// by using configured ChunkValidators, MemStore and LDBStore.
// If the chunk is not valid, its GetErrored function will
// return ErrChunkInvalid.
// This method will check if the chunk is already in the MemStore
// and it will return it if it is. If there is an error from
// the MemStore.Get, it will be returned by calling GetErrored
// on the chunk.
// This method is responsible for closing Chunk.ReqC channel
// when the chunk is stored in memstore.
// After the LDBStore.Put, it is ensured that the MemStore
// contains the chunk with the same data, but nil ReqC channel.
func (self *LocalStore) Put(chunk *Chunk) {
	valid := true
	for _, v := range self.Validators {
		if valid = v.Validate(chunk.Key, chunk.SData); valid {
			break
		}
	}
	if !valid {
		chunk.SetErrored(ErrChunkInvalid)
		chunk.markAsStored()
		return
	}

	log.Trace("localstore.put", "key", chunk.Key)
	self.mu.Lock()
	defer self.mu.Unlock()

	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	memChunk, err := self.memStore.Get(chunk.Key)
	switch err {
	case nil:
		if memChunk.ReqC == nil {
			chunk.markAsStored()
			return
		}
	case ErrChunkNotFound:
	default:
		chunk.SetErrored(err)
		return
	}

	self.memStore.Put(chunk)

	if memChunk != nil && memChunk.ReqC != nil {
		close(memChunk.ReqC)
	}

	dbStorePutCounter.Inc(1)
	self.DbStore.Put(chunk)

	newc := NewChunk(chunk.Key, nil)
	newc.SData = chunk.SData
	newc.Size = chunk.Size
	//newc.dbStored = chunk.dbStored
	newc.dbStoredC = chunk.dbStoredC
	//newc.dbStoredMu = chunk.dbStoredMu
	go func() {
		<-chunk.dbStoredC

		self.mu.Lock()
		defer self.mu.Unlock()

		self.memStore.Put(newc)
	}()
}

// Get(chunk *Chunk) looks up a chunk in the local stores
// This method is blocking until the chunk is retrieved
// so additional timeout may be needed to wrap this call if
// ChunkStores are remote and can have long latency
func (self *LocalStore) Get(key Key) (chunk *Chunk, err error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.get(key)
}

func (self *LocalStore) get(key Key) (chunk *Chunk, err error) {
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
	self.mu.Lock()
	defer self.mu.Unlock()

	var err error
	chunk, err = self.get(key)
	if err == nil && chunk.GetErrored() == nil {
		log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v found locally", key))
		return chunk, false
	}
	if err == ErrFetching && chunk.GetErrored() == nil {
		log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v hit on an existing request %v", key, chunk.ReqC))
		return chunk, false
	}
	// no data and no request status
	log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v not found locally. open new request", key))
	chunk = NewChunk(key, make(chan bool))
	self.memStore.Put(chunk)
	return chunk, true
}

// RequestsCacheLen returns the current number of outgoing requests stored in the cache
func (self *LocalStore) RequestsCacheLen() int {
	return self.memStore.requests.Len()
}

// Close the local store
func (self *LocalStore) Close() {
	self.DbStore.Close()
}
