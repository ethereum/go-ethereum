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
	"path/filepath"
	"time"
)

// NetStore implements the ChunkStore interface,
// this chunk access layer assumed 2 chunk stores
// local storage eg. LocalStore and network storage eg., NetStore
// access by calling network is blocking with a timeout
type NetStore struct {
	localStore *LocalStore
	retrieve   func(chunk *Chunk) error
}

func NewNetStore(localStore *LocalStore, retrieve func(chunk *Chunk) error) *NetStore {
	return &NetStore{localStore, retrieve}
}

// Get is the entrypoint for local retrieve requests
// waits for response or times out
func (self *NetStore) Get(key Key) (chunk *Chunk, err error) {
	if self.retrieve == nil {
		chunk, err = self.localStore.Get(key)
		if err == nil {
			return chunk, nil
		}
		if err != ErrFetching {
			return nil, err
		}
	} else {
		var created bool
		chunk, created = self.localStore.GetOrCreateRequest(key)
		if chunk.ReqC == nil {
			return chunk, nil
		}

		if created {
			if err := self.retrieve(chunk); err != nil {
				return nil, err
			}
		}
	}

	t := time.NewTicker(searchTimeout)
	defer t.Stop()

	select {
	case <-t.C:
		return nil, ErrNotFound
	case <-chunk.ReqC:
	}
	return chunk, nil
}

//this can only finally be set after all config options (file, cmd line, env vars)
//have been evaluated
func (self *StoreParams) Init(path string) {
	if self.ChunkDbPath == "" {
		self.ChunkDbPath = filepath.Join(path, "chunks")
	}
}

// Put is the entrypoint for local store requests coming from storeLoop
func (self *NetStore) Put(chunk *Chunk) {
	self.localStore.Put(chunk)
}

// Close chunk store
func (self *NetStore) Close() {}
