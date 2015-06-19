package trie

import (
	"github.com/ethereum/go-ethereum/compression/rle"
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
	// write the data to the ldb batch
	self.batch.Put(key, rle.Compress(data))
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
