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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var directCacheWrites = metrics.NewCounter("directcache/writes")
var directCacheHitTimer = metrics.NewTimer("directcache/timer/hits")
var directCacheMissTimer = metrics.NewTimer("directcache/timer/misses")

type cachedValue struct {
	Value     []byte
	BlockNum  uint64
	BlockHash common.Hash
}

// CacheValidator can check whether a certain block is in the current canonical chain.
type CacheValidator interface {
	IsCanonChainBlock(uint64, common.Hash) bool
}

type DirectCache struct {
	data      PersistentMap
	db        Database
	keyPrefix []byte
	blockNum  uint64
	blockHash common.Hash
	validator CacheValidator
	complete  bool
	dirty     map[string]bool
}

type NullCacheValidator struct {}

func (cv *NullCacheValidator) IsCanonChainBlock(num uint64, hash common.Hash) bool {
	return false
}

func NewDirectCache(pm PersistentMap, db Database, keyPrefix []byte, validator CacheValidator, complete bool) *DirectCache {
	return &DirectCache{
		data: pm,
		db: db,
		keyPrefix: keyPrefix,
		validator: validator,
		complete: complete,
		dirty: make(map[string]bool),
	}
}

func (dc *DirectCache) Iterator() *Iterator {
	// Todo: If complete is true, implement an iterator over the DB instead.
	return dc.data.Iterator()
}

func (dc *DirectCache) makeKey(key []byte) []byte {
	return append(dc.keyPrefix, key...)
}

func (dc *DirectCache) Get(key []byte) []byte {
	res, err := dc.TryGet(key)
	if err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled error: %v", err)
	}
	return res
}

func (dc *DirectCache) TryGet(key []byte) ([]byte, error) {
	start := time.Now()

	// Use the underlying object for dirty keys
	if !dc.dirty[string(key)] {
		cacheKey := dc.makeKey(key)
		if cached, ok := dc.getCached(cacheKey); ok {
			directCacheHitTimer.UpdateSince(start)
			return cached, nil
		}
	}

	value, err := dc.data.TryGet(key)
	if err != nil {
		return value, err
	}

	// If the cache isn't comprehensive, update it
	if !dc.complete && !dc.dirty[string(key)] {
		if err := dc.putCache(dc.db, key, value); err != nil && glog.V(logger.Error) {
			glog.Errorf("Error updating cache: %v", err)
		}
	}

	directCacheMissTimer.UpdateSince(start)
	return value, nil
}

func (dc *DirectCache) getCached(key []byte) ([]byte, bool) {
	enc, _ := dc.db.Get(key)
	if len(enc) == 0 {
		return nil, dc.complete
	}

	var data cachedValue
	if err := rlp.DecodeBytes(enc, &data); err != nil && glog.V(logger.Error) {
		glog.Errorf("Can't decode cached object at %x: %v", key, err)
		return nil, false
	}

	return data.Value, dc.validator.IsCanonChainBlock(data.BlockNum, data.BlockHash)
}

func (dc *DirectCache) putCache(dbw DatabaseWriter, key, value []byte) error {
	enc, _ := rlp.EncodeToBytes(cachedValue{value, dc.blockNum, dc.blockHash})
	if err := dbw.Put(append(dc.keyPrefix, key...), enc); err != nil {
		return err
	}
	return nil
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
	directCacheWrites.Inc(int64(len(dc.dirty)))
	for k, _ := range dc.dirty {
		v, err := dc.data.TryGet([]byte(k))
		if err, ok := err.(*MissingNodeError); err != nil && !ok {
			return common.Hash{}, err
		}
		if err := dc.putCache(dbw, []byte(k), v); err != nil {
			return common.Hash{}, err
		}
	}
	dc.dirty = make(map[string]bool)
	return dc.data.CommitTo(dbw)
}
