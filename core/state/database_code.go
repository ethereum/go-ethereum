// Copyright 2026 The go-ethereum Authors
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

package state

import (
	"sync/atomic"

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

// CodeCache maintains cached contract code that is shared across blocks, enabling
// fast access for external calls such as RPCs and state transitions.
//
// It is thread-safe and has a bounded size.
type codeCache struct {
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
}

// newCodeCache initializes the contract code cache with the predefined capacity.
func newCodeCache() *codeCache {
	return &codeCache{
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](codeSizeCacheSize),
	}
}

// Get returns the contract code associated with the provided code hash.
func (c *codeCache) Get(hash common.Hash) ([]byte, bool) {
	return c.codeCache.Get(hash)
}

// GetSize returns the contract code size associated with the provided code hash.
func (c *codeCache) GetSize(hash common.Hash) (int, bool) {
	return c.codeSizeCache.Get(hash)
}

// Put adds the provided contract code along with its size information into the cache.
func (c *codeCache) Put(hash common.Hash, code []byte) {
	c.codeCache.Add(hash, code)
	c.codeSizeCache.Add(hash, len(code))
}

// CodeReader implements state.ContractCodeReader, accessing contract code either in
// local key-value store or the shared code cache.
//
// Reader is safe for concurrent access.
type CodeReader struct {
	db    ethdb.KeyValueReader
	cache *codeCache

	// Cache statistics
	hit       atomic.Int64 // Number of code lookups found in the cache
	miss      atomic.Int64 // Number of code lookups not found in the cache
	hitBytes  atomic.Int64 // Total number of bytes read from cache
	missBytes atomic.Int64 // Total number of bytes read from database
}

// newCodeReader constructs the code reader with provided key value store and the cache.
func newCodeReader(db ethdb.KeyValueReader, cache *codeCache) *CodeReader {
	return &CodeReader{
		db:    db,
		cache: cache,
	}
}

// Has returns the flag indicating whether the contract code with
// specified address and hash exists or not.
func (r *CodeReader) Has(addr common.Address, codeHash common.Hash) bool {
	return len(r.Code(addr, codeHash)) > 0
}

// Code implements state.ContractCodeReader, retrieving a particular contract's code.
// Null is returned if the contract code is not present.
func (r *CodeReader) Code(addr common.Address, codeHash common.Hash) []byte {
	code, _ := r.cache.Get(codeHash)
	if len(code) > 0 {
		r.hit.Add(1)
		r.hitBytes.Add(int64(len(code)))
		return code
	}
	r.miss.Add(1)

	code = rawdb.ReadCode(r.db, codeHash)
	if len(code) > 0 {
		r.cache.Put(codeHash, code)
		r.missBytes.Add(int64(len(code)))
	}
	return code
}

// CodeSize implements state.ContractCodeReader, retrieving a particular contract
// code's size. Zero is returned if the contract code is not present.
func (r *CodeReader) CodeSize(addr common.Address, codeHash common.Hash) int {
	if cached, ok := r.cache.GetSize(codeHash); ok {
		r.hit.Add(1)
		return cached
	}
	return len(r.Code(addr, codeHash))
}

// CodeWithPrefix retrieves the contract code for the specified account address
// and code hash. It is almost identical to Code, but uses rawdb.ReadCodeWithPrefix
// for database lookups. The intention is to gradually deprecate the old
// contract code scheme.
func (r *CodeReader) CodeWithPrefix(addr common.Address, codeHash common.Hash) []byte {
	code, _ := r.cache.Get(codeHash)
	if len(code) > 0 {
		r.hit.Add(1)
		r.hitBytes.Add(int64(len(code)))
		return code
	}
	r.miss.Add(1)

	code = rawdb.ReadCodeWithPrefix(r.db, codeHash)
	if len(code) > 0 {
		r.cache.Put(codeHash, code)
		r.missBytes.Add(int64(len(code)))
	}
	return code
}

// GetCodeStats implements ContractCodeReaderStater, returning the statistics
// of the code reader.
func (r *CodeReader) GetCodeStats() ContractCodeReaderStats {
	return ContractCodeReaderStats{
		CacheHit:       r.hit.Load(),
		CacheMiss:      r.miss.Load(),
		CacheHitBytes:  r.hitBytes.Load(),
		CacheMissBytes: r.missBytes.Load(),
	}
}

type CodeBatch struct {
	db         *CodeDB
	codes      [][]byte
	codeHashes []common.Hash
}

// newCodeBatch constructs the batch for writing contract code.
func newCodeBatch(db *CodeDB) *CodeBatch {
	return &CodeBatch{
		db: db,
	}
}

// newCodeBatchWithSize constructs the batch with a pre-allocated capacity.
func newCodeBatchWithSize(db *CodeDB, size int) *CodeBatch {
	return &CodeBatch{
		db:         db,
		codes:      make([][]byte, 0, size),
		codeHashes: make([]common.Hash, 0, size),
	}
}

// Put inserts the given contract code into the writer, waiting for commit.
func (b *CodeBatch) Put(codeHash common.Hash, code []byte) {
	b.codes = append(b.codes, code)
	b.codeHashes = append(b.codeHashes, codeHash)
}

// Commit flushes the accumulated dirty contract code into the database and
// also place them in the cache.
func (b *CodeBatch) Commit() error {
	batch := b.db.db.NewBatch()
	for i, code := range b.codes {
		rawdb.WriteCode(batch, b.codeHashes[i], code)
		b.db.cache.Put(b.codeHashes[i], code)
	}
	if err := batch.Write(); err != nil {
		return err
	}
	b.codes = b.codes[:0]
	b.codeHashes = b.codeHashes[:0]
	return nil
}

// CodeDB is responsible for managing the contract code and provides the access
// to it. It can be used as a global object, sharing it between multiple entities.
type CodeDB struct {
	db    ethdb.KeyValueStore
	cache *codeCache
}

// NewCodeDB constructs the contract code database with the provided key value store.
func NewCodeDB(db ethdb.KeyValueStore) *CodeDB {
	return &CodeDB{
		db:    db,
		cache: newCodeCache(),
	}
}

// Reader returns the contract code reader.
func (d *CodeDB) Reader() *CodeReader {
	return newCodeReader(d.db, d.cache)
}

// NewBatch returns the batch for flushing contract codes.
func (d *CodeDB) NewBatch() *CodeBatch {
	return newCodeBatch(d)
}

// NewBatchWithSize returns the batch with pre-allocated capacity.
func (d *CodeDB) NewBatchWithSize(size int) *CodeBatch {
	return newCodeBatchWithSize(d, size)
}
