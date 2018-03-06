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
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

/*
DPA provides the client API entrypoints Store and Retrieve to store and retrieve
It can store anything that has a byte slice representation, so files or serialised objects etc.

Storage: DPA calls the Chunker to segment the input datastream of any size to a merkle hashed tree of chunks. The key of the root block is returned to the client.

Retrieval: given the key of the root block, the DPA retrieves the block chunks and reconstructs the original data and passes it back as a lazy reader. A lazy reader is a reader with on-demand delayed processing, i.e. the chunks needed to reconstruct a large file are only fetched and processed if that particular part of the document is actually read.

As the chunker produces chunks, DPA dispatches them to its own chunk store
implementation for storage or retrieval.
*/

const (
	storeChanCapacity           = 100
	retrieveChanCapacity        = 100
	singletonSwarmDbCapacity    = 50000
	singletonSwarmCacheCapacity = 500
	maxStoreProcesses           = 8
	maxRetrieveProcesses        = 8
)

var (
	ErrChunkNotFound = errors.New("chunk not found")
	ErrFetching      = errors.New("chunk still fetching")
	// timeout interval before retrieval is timed out
	searchTimeout = 3 * time.Second
)

type DPA struct {
	ChunkStore
	storeC    chan *Chunk
	retrieveC chan *Chunk
	Chunker   Chunker

	lock    sync.Mutex
	running bool
	wg      *sync.WaitGroup
	quitC   chan bool
}

// for testing locally
func NewLocalDPA(datadir string, basekey []byte) (*DPA, error) {

	hash := MakeHashFunc("SHA3")

	dbStore, err := NewLDBStore(datadir, hash, singletonSwarmDbCapacity, func(k Key) (ret uint8) { return uint8(Proximity(basekey[:], k[:])) })
	if err != nil {
		return nil, err
	}

	return NewDPA(&LocalStore{
		memStore: NewMemStore(dbStore, singletonSwarmCacheCapacity),
		DbStore:  dbStore,
	}, NewChunkerParams()), nil
}

func NewDPA(store ChunkStore, params *ChunkerParams) *DPA {
	chunker := NewPyramidChunker(params)
	return &DPA{
		Chunker:    chunker,
		ChunkStore: store,
	}
}

// Public API. Main entry point for document retrieval directly. Used by the
// FS-aware API and httpaccess
// Chunk retrieval blocks on netStore requests with a timeout so reader will
// report error if retrieval of chunks within requested range time out.
func (self *DPA) Retrieve(key Key) LazySectionReader {
	return self.Chunker.Join(key, self.retrieveC, 0)
}

// Public API. Main entry point for document storage directly. Used by the
// FS-aware API and httpaccess
func (self *DPA) Store(data io.Reader, size int64) (key Key, wait func(), err error) {
	return self.Chunker.Split(data, size, self.storeC)
}

func (self *DPA) Start() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.running {
		return
	}
	self.running = true
	self.retrieveC = make(chan *Chunk, retrieveChanCapacity)
	self.storeC = make(chan *Chunk, storeChanCapacity)
	self.quitC = make(chan bool)
	self.storeLoop()
	self.retrieveLoop()
}

func (self *DPA) Stop() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if !self.running {
		return
	}
	self.running = false
	close(self.quitC)
}

// retrieveLoop dispatches the parallel chunk retrieval requests received on the
// retrieve channel to its ChunkStore  (NetStore or LocalStore)
func (self *DPA) retrieveLoop() {
	for i := 0; i < maxRetrieveProcesses; i++ {
		go self.retrieveWorker()
	}
	log.Trace(fmt.Sprintf("dpa: retrieve loop spawning %v workers", maxRetrieveProcesses))
}

func (self *DPA) retrieveWorker() {
	for chunk := range self.retrieveC {
		storedChunk, err := self.Get(chunk.Key)
		if err != nil {
			log.Trace(fmt.Sprintf("error retrieving chunk %v: %v", chunk.Key.Log(), err))
		} else {
			chunk.SData = storedChunk.SData
			chunk.Size = storedChunk.Size
		}
		close(chunk.C)

		select {
		case <-self.quitC:
			return
		default:
		}
	}
}

// storeLoop dispatches the parallel chunk store request processors
// received on the store channel to its ChunkStore (NetStore or LocalStore)
func (self *DPA) storeLoop() {
	for i := 0; i < maxStoreProcesses; i++ {
		go self.storeWorker()
	}
	log.Trace(fmt.Sprintf("dpa: store spawning %v workers", maxStoreProcesses))
}

func (self *DPA) storeWorker() {
	for chunk := range self.storeC {
		self.Put(chunk)
		select {
		case <-self.quitC:
			return
		default:
		}
	}
}
