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
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
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
func (p *LocalStoreParams) Init(path string) {
	if p.ChunkDbPath == "" {
		p.ChunkDbPath = filepath.Join(path, "chunks")
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
func (ls *LocalStore) Put(ctx context.Context, chunk *Chunk) {
	if l := len(chunk.SData); l < 9 {
		log.Debug("incomplete chunk data", "addr", chunk.Addr, "length", l)
		chunk.SetErrored(ErrChunkInvalid)
		chunk.markAsStored()
		return
	}
	valid := true
	for _, v := range ls.Validators {
		if valid = v.Validate(chunk.Addr, chunk.SData); valid {
			break
		}
	}
	if !valid {
		log.Trace("invalid content address", "addr", chunk.Addr)
		chunk.SetErrored(ErrChunkInvalid)
		chunk.markAsStored()
		return
	}

	log.Trace("localstore.put", "addr", chunk.Addr)

	ls.mu.Lock()
	defer ls.mu.Unlock()

	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	memChunk, err := ls.memStore.Get(ctx, chunk.Addr)
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

	ls.DbStore.Put(ctx, chunk)

	// chunk is no longer a request, but a chunk with data, so replace it in memStore
	newc := NewChunk(chunk.Addr, nil)
	newc.SData = chunk.SData
	newc.Size = chunk.Size
	newc.dbStoredC = chunk.dbStoredC

	ls.memStore.Put(ctx, newc)

	if memChunk != nil && memChunk.ReqC != nil {
		close(memChunk.ReqC)
	}
}

// Get(chunk *Chunk) looks up a chunk in the local stores
// This method is blocking until the chunk is retrieved
// so additional timeout may be needed to wrap this call if
// ChunkStores are remote and can have long latency
func (ls *LocalStore) Get(ctx context.Context, addr Address) (chunk *Chunk, err error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	return ls.get(ctx, addr)
}

func (ls *LocalStore) get(ctx context.Context, addr Address) (chunk *Chunk, err error) {
	chunk, err = ls.memStore.Get(ctx, addr)
	if err == nil {
		if chunk.ReqC != nil {
			select {
			case <-chunk.ReqC:
			default:
				metrics.GetOrRegisterCounter("localstore.get.errfetching", nil).Inc(1)
				return chunk, ErrFetching
			}
		}
		metrics.GetOrRegisterCounter("localstore.get.cachehit", nil).Inc(1)
		return
	}
	metrics.GetOrRegisterCounter("localstore.get.cachemiss", nil).Inc(1)
	chunk, err = ls.DbStore.Get(ctx, addr)
	if err != nil {
		metrics.GetOrRegisterCounter("localstore.get.error", nil).Inc(1)
		return
	}
	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
	ls.memStore.Put(ctx, chunk)
	return
}

// retrieve logic common for local and network chunk retrieval requests
func (ls *LocalStore) GetOrCreateRequest(ctx context.Context, addr Address) (chunk *Chunk, created bool) {
	metrics.GetOrRegisterCounter("localstore.getorcreaterequest", nil).Inc(1)

	ls.mu.Lock()
	defer ls.mu.Unlock()

	var err error
	chunk, err = ls.get(ctx, addr)
	if err == nil && chunk.GetErrored() == nil {
		metrics.GetOrRegisterCounter("localstore.getorcreaterequest.hit", nil).Inc(1)
		log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v found locally", addr))
		return chunk, false
	}
	if err == ErrFetching && chunk.GetErrored() == nil {
		metrics.GetOrRegisterCounter("localstore.getorcreaterequest.errfetching", nil).Inc(1)
		log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v hit on an existing request %v", addr, chunk.ReqC))
		return chunk, false
	}
	// no data and no request status
	metrics.GetOrRegisterCounter("localstore.getorcreaterequest.miss", nil).Inc(1)
	log.Trace(fmt.Sprintf("LocalStore.GetOrRetrieve: %v not found locally. open new request", addr))
	chunk = NewChunk(addr, make(chan bool))
	ls.memStore.Put(ctx, chunk)
	return chunk, true
}

// RequestsCacheLen returns the current number of outgoing requests stored in the cache
func (ls *LocalStore) RequestsCacheLen() int {
	return ls.memStore.requests.Len()
}

// Close the local store
func (ls *LocalStore) Close() {
	ls.DbStore.Close()
}
