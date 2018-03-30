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
	"io"
	"time"
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
	singletonSwarmDbCapacity    = 50000
	singletonSwarmCacheCapacity = 500
)

var (
	ErrChunkNotFound = errors.New("chunk not found")
	ErrFetching      = errors.New("chunk still fetching")
	// timeout interval before retrieval is timed out
	searchTimeout = 3 * time.Second
)

type DPA struct {
	ChunkStore
	hashFunc SwarmHasher
}

type DPAParams struct {
	Hash string
}

func NewDPAParams() *DPAParams {
	return &DPAParams{
		Hash: SHA3Hash,
	}
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
	}, NewDPAParams()), nil
}

func NewDPA(store ChunkStore, params *DPAParams) *DPA {
	hashFunc := MakeHashFunc(params.Hash)
	return &DPA{
		ChunkStore: store,
		hashFunc:   hashFunc,
	}
}

// Public API. Main entry point for document retrieval directly. Used by the
// FS-aware API and httpaccess
// Chunk retrieval blocks on netStore requests with a timeout so reader will
// report error if retrieval of chunks within requested range time out.
func (self *DPA) Retrieve(key Key) LazySectionReader {
	getter := NewHasherStore(self.ChunkStore, self.hashFunc, len(key) > self.hashFunc().Size())
	return TreeJoin(key, getter, 0)
}

// Public API. Main entry point for document storage directly. Used by the
// FS-aware API and httpaccess
func (self *DPA) Store(data io.Reader, size int64, toEncrypt bool) (key Key, wait func(), err error) {
	putter := NewHasherStore(self.ChunkStore, self.hashFunc, toEncrypt)
	return PyramidSplit(data, putter, putter)
}
