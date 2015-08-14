// Copyright 2014 The go-ethereum Authors
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

package trie

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/syndtr/goleveldb/leveldb"
)

type Backend interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte) error
}

type Cache struct {
	batch   *leveldb.Batch
	store   map[string][]byte
	backend Backend
}

func NewCache(backend Backend) *Cache {
	return &Cache{new(leveldb.Batch), make(map[string][]byte), backend}
}

func (self *Cache) Get(key []byte) []byte {
	data := self.store[string(key)]
	if data == nil {
		data, _ = self.backend.Get(key)
	}

	return data
}

func (self *Cache) Put(key []byte, data []byte) {
	self.batch.Put(key, data)
	self.store[string(key)] = data
}

// Flush flushes the trie to the backing layer. If this is a leveldb instance
// we'll use a batched write, otherwise we'll use regular put.
func (self *Cache) Flush() {
	if db, ok := self.backend.(*ethdb.LDBDatabase); ok {
		if err := db.LDB().Write(self.batch, nil); err != nil {
			glog.Fatal("db write err:", err)
		}
	} else {
		for k, v := range self.store {
			self.backend.Put([]byte(k), v)
		}
	}
}

func (self *Cache) Copy() *Cache {
	cache := NewCache(self.backend)
	for k, v := range self.store {
		cache.store[k] = v
	}
	return cache
}

func (self *Cache) Reset() {
	//self.store = make(map[string][]byte)
}
