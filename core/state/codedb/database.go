// Copyright 2025 The go-ethereum Authors
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

package codedb

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	// Number of codeHash->size associations to keep.
	codeSizeCacheSize = 1_000_000

	// Cache size granted for caching clean code.
	codeCacheSize = 256 * 1024 * 1024
)

// Cache maintains cached contract code that is shared across blocks, enabling
// fast access for external calls such as RPCs and state transitions.
//
// It is thread-safe and has a bounded size.
type Cache struct {
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
}

// NewCache initializes the contract code cache with the predefined capacity.
func NewCache() *Cache {
	return &Cache{
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](codeSizeCacheSize),
	}
}

// Get returns the contract code associated with the provided code hash.
func (c *Cache) Get(hash common.Hash) ([]byte, bool) {
	return c.codeCache.Get(hash)
}

// GetSize returns the contract code size associated with the provided code hash.
func (c *Cache) GetSize(hash common.Hash) (int, bool) {
	return c.codeSizeCache.Get(hash)
}

// Put adds the provided contract code along with its size information into the cache.
func (c *Cache) Put(hash common.Hash, code []byte) {
	c.codeCache.Add(hash, code)
	c.codeSizeCache.Add(hash, len(code))
}

// Reader implements state.ContractCodeReader, accessing contract code either in
// local key-value store or the shared code cache.
//
// Reader is safe for concurrent access.
type Reader struct {
	db    ethdb.KeyValueReader
	cache *Cache
}

// newReader constructs the code reader with provided key value store and the cache.
func newReader(db ethdb.KeyValueReader, cache *Cache) *Reader {
	return &Reader{
		db:    db,
		cache: cache,
	}
}

// Has returns the flag indicating whether the contract code with
// specified address and hash exists or not.
func (r *Reader) Has(addr common.Address, codeHash common.Hash) bool {
	return len(r.Code(addr, codeHash)) > 0
}

// Code implements state.ContractCodeReader, retrieving a particular contract's code.
// Null is returned if the contract code is not present.
func (r *Reader) Code(addr common.Address, codeHash common.Hash) []byte {
	code, _ := r.cache.Get(codeHash)
	if len(code) > 0 {
		return code
	}
	code = rawdb.ReadCode(r.db, codeHash)
	if len(code) > 0 {
		r.cache.Put(codeHash, code)
	}
	return code
}

// CodeSize implements state.ContractCodeReader, retrieving a particular contract
// code's size. Zero is returned if the contract code is not present.
func (r *Reader) CodeSize(addr common.Address, codeHash common.Hash) int {
	if cached, ok := r.cache.GetSize(codeHash); ok {
		return cached
	}
	return len(r.Code(addr, codeHash))
}

// CodeWithPrefix retrieves the contract code for the specified account address
// and code hash. It is almost identical to Code, but uses rawdb.ReadCodeWithPrefix
// for database lookups. The intention is to gradually deprecate the old
// contract code scheme.
func (r *Reader) CodeWithPrefix(addr common.Address, codeHash common.Hash) []byte {
	code, _ := r.cache.Get(codeHash)
	if len(code) > 0 {
		return code
	}
	code = rawdb.ReadCodeWithPrefix(r.db, codeHash)
	if len(code) > 0 {
		r.cache.Put(codeHash, code)
	}
	return code
}

// Writer implements the state.ContractCodeWriter for committing the dirty contract
// code into database.
type Writer struct {
	db         *Database
	codes      [][]byte
	codeHashes []common.Hash
}

// newWriter constructs the code writer.
func newWriter(db *Database) *Writer {
	return &Writer{
		db: db,
	}
}

// Put inserts the given contract code into the writer, waiting for commit.
func (w *Writer) Put(codeHash common.Hash, code []byte) {
	w.codes = append(w.codes, code)
	w.codeHashes = append(w.codeHashes, codeHash)
}

// Commit flushes the accumulated dirty contract code into the database and
// also place them in the cache.
func (w *Writer) Commit() error {
	batch := w.db.db.NewBatch()
	for i, code := range w.codes {
		rawdb.WriteCode(batch, w.codeHashes[i], code)
		w.db.cache.Put(w.codeHashes[i], code)
	}
	if err := batch.Write(); err != nil {
		return err
	}
	w.codes = w.codes[:0]
	w.codeHashes = w.codeHashes[:0]
	return nil
}

// Database is responsible for managing the contract code and provides the access
// to it. It can be used as a global object, sharing it between multiple entities.
type Database struct {
	db    ethdb.KeyValueStore
	cache *Cache
}

// New constructs the contract code database with the provided key value store.
func New(db ethdb.KeyValueStore) *Database {
	return &Database{
		db:    db,
		cache: NewCache(),
	}
}

// Reader returns the contract code reader.
func (s *Database) Reader() *Reader {
	return newReader(s.db, s.cache)
}

// Writer returns the contract code writer.
func (s *Database) Writer() *Writer {
	return newWriter(s)
}
