// Copyright 2023 The go-ethereum Authors
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

package light

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// canonicalStore stores instances of the given type in a database and caches
// them in memory, associated with a continuous range of period numbers.
// Note: canonicalStore is not thread safe and it is the caller's responsibility
// to avoid concurrent access.
type canonicalStore[T any] struct {
	keyPrefix []byte
	periods   Range
	cache     *lru.Cache[uint64, T]
	encode    func(T) ([]byte, error)
	decode    func([]byte) (T, error)
}

// newCanonicalStore creates a new canonicalStore and loads all keys associated
// with the keyPrefix in order to determine the ranges available in the database.
func newCanonicalStore[T any](db ethdb.KeyValueStore, keyPrefix []byte,
	encode func(T) ([]byte, error), decode func([]byte) (T, error)) *canonicalStore[T] {
	cs := &canonicalStore[T]{
		keyPrefix: keyPrefix,
		encode:    encode,
		decode:    decode,
		cache:     lru.NewCache[uint64, T](100),
	}
	var (
		iter  = db.NewIterator(keyPrefix, nil)
		kl    = len(keyPrefix)
		first = true
	)
	for iter.Next() {
		if len(iter.Key()) != kl+8 {
			log.Warn("Invalid key length in the canonical chain database", "key", fmt.Sprintf("%#x", iter.Key()))
			continue
		}
		period := binary.BigEndian.Uint64(iter.Key()[kl : kl+8])
		if first {
			cs.periods.Start = period
		} else if cs.periods.End != period {
			log.Warn("Gap in the canonical chain database")
			break // continuity guaranteed
		}
		first = false
		cs.periods.End = period + 1
	}
	iter.Release()
	return cs
}

// databaseKey returns the database key belonging to the given period.
func (cs *canonicalStore[T]) databaseKey(period uint64) []byte {
	var (
		kl  = len(cs.keyPrefix)
		key = make([]byte, kl+8)
	)
	copy(key[:kl], cs.keyPrefix)
	binary.BigEndian.PutUint64(key[kl:], period)
	return key
}

// add adds the given item to the database. It also ensures that the range remains
// continuous. Can be used either with a batch or database backend.
func (cs *canonicalStore[T]) add(backend ethdb.KeyValueWriter, period uint64, value T) error {
	if !cs.periods.CanExpand(period) {
		return fmt.Errorf("period expansion is not allowed, first: %d, next: %d, period: %d", cs.periods.Start, cs.periods.End, period)
	}
	enc, err := cs.encode(value)
	if err != nil {
		return err
	}
	if err := backend.Put(cs.databaseKey(period), enc); err != nil {
		return err
	}
	cs.cache.Add(period, value)
	cs.periods.Expand(period)
	return nil
}

// deleteFrom removes items starting from the given period.
func (cs *canonicalStore[T]) deleteFrom(batch ethdb.Batch, fromPeriod uint64) (deleted Range) {
	keepRange, deleteRange := cs.periods.Split(fromPeriod)
	deleteRange.Each(func(period uint64) {
		batch.Delete(cs.databaseKey(period))
		cs.cache.Remove(period)
	})
	cs.periods = keepRange
	return deleteRange
}

// get returns the item at the given period or the null value of the given type
// if no item is present.
func (cs *canonicalStore[T]) get(backend ethdb.KeyValueReader, period uint64) (value T, ok bool) {
	if !cs.periods.Contains(period) {
		return
	}
	if value, ok = cs.cache.Get(period); ok {
		return
	}
	if enc, err := backend.Get(cs.databaseKey(period)); err == nil {
		if v, err := cs.decode(enc); err == nil {
			value, ok = v, true
			cs.cache.Add(period, value)
		} else {
			log.Error("Error decoding canonical store value", "error", err)
		}
	} else {
		log.Error("Canonical store value not found", "period", period, "start", cs.periods.Start, "end", cs.periods.End)
	}
	return
}
