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

package trie

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
)

var (
	directCacheLock      = &sync.Mutex{}
	directCacheLocked    = false
	directCacheWrites    = metrics.NewCounter("directcache/writes")
	directCacheHits      = metrics.NewCounter("directcache/hits")
	directCacheMisses    = metrics.NewCounter("directcache/misses")
	directCacheMemHits   = metrics.NewCounter("directcache/memcache/hits")
	directCacheMemMisses = metrics.NewCounter("directcache/memcache/misses")
	directCacheTimer     = metrics.NewTimer("directcache/timer")

	NotFound        = errors.New("Cache entry not found")
	MigrationPrefix = []byte("directstatecachemigration:")
)

type MigrationStatus int

const (
	NotStarted = iota
	Running    = iota
	Complete   = iota
)

// CacheValidator can check whether a certain block is in the current canonical chain.
type CacheValidator interface {
	IsCanonChainBlock(uint64, common.Hash) bool
}

type DirectCache struct {
	data     PersistentMap
	db       Database
	memcache *lru.ARCCache

	keyPrefix []byte
	blockNum  uint64
	blockHash common.Hash
	validator CacheValidator
	complete  bool
	dirty     map[string]bool
}

type NullCacheValidator struct{}

func (cv *NullCacheValidator) IsCanonChainBlock(num uint64, hash common.Hash) bool {
	return false
}

func NewDirectCache(pm PersistentMap, db Database, memcache *lru.ARCCache, keyPrefix []byte, blockNum uint64, blockHash common.Hash, validator CacheValidator, complete bool) *DirectCache {
	if memcache == nil {
		memcache, _ = lru.NewARC(0)
	}
	return &DirectCache{
		data:      pm,
		db:        db,
		memcache:  memcache,
		keyPrefix: keyPrefix,
		blockNum:  blockNum,
		blockHash: blockHash,
		validator: validator,
		complete:  complete,
		dirty:     make(map[string]bool),
	}
}

func (dc *DirectCache) Iterator() *Iterator {
	// Todo: If complete is true, implement an iterator over the DB instead.
	return dc.data.Iterator()
}

func (dc *DirectCache) Get(key []byte) []byte {
	res, err := dc.TryGet(key)
	if err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled error: %v", err)
	}
	return res
}

func (dc *DirectCache) TryGet(key []byte) ([]byte, error) {
	dirty := dc.dirty[string(key)]

	// Use the underlying object for dirty keys
	if !dirty {
		if cached, ok := dc.getCached(key); ok {
			return cached, nil
		}
	}

	value, err := dc.data.TryGet(key)
	if err != nil {
		return value, err
	}

	if !dc.dirty[string(key)] {
		// Flag the key as dirty so it gets written at commit time
		dc.dirty[string(key)] = true
	}

	// Don't count fetches of dirty data as cache misses
	if !dirty {
		directCacheMisses.Inc(1)
	}

	return value, nil
}

func (dc *DirectCache) getCached(key []byte) ([]byte, bool) {
	// Try to retrieve the object from the memcache
	var entry *cachedValue
	if cached, ok := dc.memcache.Get(string(key)); ok {
		directCacheMemHits.Inc(1)
		entry = cached.(*cachedValue)
	}
	// If the memcache missed, reach down to the database
	var err error
	if entry == nil {
		directCacheMemMisses.Inc(1)

		if entry, err = getDirectCache(dc.keyPrefix, key, dc.db); err != nil {
			if err == NotFound {
				return nil, dc.complete
			}
			glog.Errorf("Error retrieving direct cache data: %v", err)
			return nil, false
		}
		dc.memcache.Add(string(key), entry)
	}
	canonical := dc.blockNum > 0 && entry.BlockNum < dc.blockNum && dc.validator.IsCanonChainBlock(entry.BlockNum, entry.BlockHash)
	return entry.Value, canonical
}

func (dc *DirectCache) Update(key, value []byte) {
	if err := dc.TryUpdate(key, value); err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled error: %v", err)
	}
}

func (dc *DirectCache) TryUpdate(key, value []byte) error {
	dc.dirty[string(key)] = true
	return dc.data.TryUpdate(key, value)
}

func (dc *DirectCache) Delete(key []byte) {
	if err := dc.TryDelete(key); err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled error: %v", err)
	}
}

func (dc *DirectCache) TryDelete(key []byte) error {
	dc.dirty[string(key)] = true
	return dc.data.TryDelete(key)
}

func (dc *DirectCache) CommitTo(dbw DatabaseWriter) (root common.Hash, err error) {
	if err := DirectCacheTransaction(func() error {
		for k, _ := range dc.dirty {
			v, err := dc.data.TryGet([]byte(k))
			if _, ok := err.(*MissingNodeError); err != nil && !ok {
				return err
			}
			if err := WriteDirectCache(dc.keyPrefix, []byte(k), v, dc.blockNum, dc.blockHash, dbw); err != nil {
				return err
			}
			dc.memcache.Add(k, &cachedValue{v, dc.blockNum, dc.blockHash})
		}
		return nil
	}); err != nil {
		return common.Hash{}, err
	}

	dc.dirty = make(map[string]bool)
	return dc.data.CommitTo(dbw)
}

// Populate iterates over the underlying trie, filling in any unset cache entries.
// After Populate has completed, future DirectCache instances can have `complete`
// set to true, for better efficiency on cache misses.
func (dc *DirectCache) Populate() (err error) {
	i := 0
	writes := 0
	it := dc.Iterator()
	for it.Next() {
		if err := DirectCacheTransaction(func() error {
			if !HasDirectCache(dc.keyPrefix, it.Key, dc.db) {
				writes += 1
				if err = WriteDirectCache(dc.keyPrefix, it.Key, it.Value, dc.blockNum, dc.blockHash, dc.db); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}

		i += 1
		if i%10000 == 0 && glog.V(logger.Info) {
			glog.V(logger.Info).Infof("Constructing direct cache: processed %v entries, written %v", i, writes)
		}
	}
	return nil
}

type cachedValue struct {
	Value     []byte
	BlockNum  uint64
	BlockHash common.Hash
}

func DirectCacheTransaction(tx func() error) error {
	directCacheLock.Lock()
	directCacheLocked = true
	defer func() {
		directCacheLocked = false
		directCacheLock.Unlock()
	}()

	return tx()
}

// WriteDirectCache places a value node directly into the database along with
// block metadata to validate its relevancy.
//
// The method is meant to be used by code that circumvents the state database
// and its integrated cache, namely during fast sync and database upgrades.
func WriteDirectCache(prefix, key, value []byte, number uint64, hash common.Hash, dbw DatabaseWriter) error {
	directCacheWrites.Inc(1)

	if !directCacheLocked {
		return fmt.Errorf("WriteDirectCache may only be called in a DirectCacheTransaction")
	}

	enc, _ := rlp.EncodeToBytes(cachedValue{value, number, hash})
	return dbw.Put(append(prefix, key...), enc)
}

// getDirectCache retrieves a value node directly from the database along with
// block metadata to validate its relevancy.
func getDirectCache(prefix, key []byte, db Database) (*cachedValue, error) {
	defer func(start time.Time) { directCacheTimer.UpdateSince(start) }(time.Now())

	enc, _ := db.Get(append(prefix, key...))
	if len(enc) == 0 {
		return nil, NotFound
	}
	data := new(cachedValue)
	if err := rlp.DecodeBytes(enc, data); err != nil {
		return nil, fmt.Errorf("Can't decode cached object at %x: %v", key, err)
	}
	return data, nil
}

// GetDirectCache retrieves a value node directly from the database along with
// block metadata to validate its relevancy.
//
// The method is meant to be used by code that circumvents the state database
// and its integrated cache, namely during fast sync and database upgrades.
func GetDirectCache(prefix, key []byte, db Database) ([]byte, uint64, common.Hash, error) {
	data, err := getDirectCache(prefix, key, db)
	if err != nil {
		return nil, 0, common.Hash{}, err
	}
	return data.Value, data.BlockNum, data.BlockHash, nil
}

// HasDirectCache returns true iff a direct cache node exists for the specified key
func HasDirectCache(prefix, key []byte, db Database) bool {
	if enc, err := db.Get(append(prefix, key...)); err == nil && len(enc) > 0 {
		return true
	}
	return false
}

type migrationState struct {
	Status MigrationStatus
	Hash   common.Hash
}

// GetMigrationState returns the block number of the migration to the direct cache, and
// whether or not it's complete.
func GetMigrationState(prefix []byte, db Database) (common.Hash, MigrationStatus) {
	enc, _ := db.Get(append(MigrationPrefix, prefix...))
	if len(enc) == 0 {
		return common.Hash{}, NotStarted
	}

	var data migrationState
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		glog.Errorf("Could not decode migration status: %v", err)
		return common.Hash{}, NotStarted
	}

	return data.Hash, data.Status
}

// SetMigrationState updates the migration state in the database
func SetMigrationState(prefix []byte, blockHash common.Hash, status MigrationStatus, db Database) error {
	enc, _ := rlp.EncodeToBytes(migrationState{status, blockHash})
	return db.Put(append(MigrationPrefix, prefix...), enc)
}

// DirectCacheReads retrieves a global counter measuring the number of direct
// cache reads from the disk since process startup. This isn't useful for anything
// apart from trie debugging purposes.
func DirectCacheReads() int64 {
	return directCacheTimer.Count()
}

// DirectCacheWrites retrieves a global counter measuring the number of direct
// cache writes from the disk since process startup. This isn't useful for anything
// apart from trie debugging purposes.
func DirectCacheWrites() int64 {
	return directCacheWrites.Count()
}

// DirectCacheMisses retrieves a global counter measuring the number of direct
// cache misses from the disk since process startup. This isn't useful for anything
// apart from trie debugging purposes.
func DirectCacheMisses() int64 {
	return directCacheMisses.Count()
}

func DirectCacheMemcacheHits() int64 {
	return directCacheMemHits.Count()
}

func DirectCacheMemcacheMisses() int64 {
	return directCacheMemMisses.Count()
}
