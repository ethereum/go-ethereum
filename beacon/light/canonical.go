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
	"github.com/ethereum/go-ethereum/rlp"
)

// canonicalStore stores instances of the given type in a database and caches
// them in memory, associated with a continuous range of period numbers.
// Note: canonicalStore is not thread safe and it is the caller's responsibility
// to avoid concurrent access.
type canonicalStore[T any] struct {
	keyPrefix []byte
	periods   periodRange
	cache     *lru.Cache[uint64, T]
}

// newCanonicalStore creates a new canonicalStore and loads all keys associated
// with the keyPrefix in order to determine the ranges available in the database.
func newCanonicalStore[T any](db ethdb.Iteratee, keyPrefix []byte) (*canonicalStore[T], error) {
	cs := &canonicalStore[T]{
		keyPrefix: keyPrefix,
		cache:     lru.NewCache[uint64, T](100),
	}
	var (
		iter  = db.NewIterator(keyPrefix, nil)
		kl    = len(keyPrefix)
		first = true
	)
	defer iter.Release()

	for iter.Next() {
		if len(iter.Key()) != kl+8 {
			log.Warn("Invalid key length in the canonical chain database", "key", fmt.Sprintf("%#x", iter.Key()))
			continue
		}
		period := binary.BigEndian.Uint64(iter.Key()[kl : kl+8])
		if first {
			cs.periods.Start = period
		} else if cs.periods.End != period {
			return nil, fmt.Errorf("gap in the canonical chain database between periods %d and %d", cs.periods.End, period-1)
		}
		first = false
		cs.periods.End = period + 1
	}
	return cs, nil
}

// databaseKey returns the database key belonging to the given period.
func (cs *canonicalStore[T]) databaseKey(period uint64) []byte {
	return binary.BigEndian.AppendUint64(append([]byte{}, cs.keyPrefix...), period)
}

// add adds the given item to the database. It also ensures that the range remains
// continuous. Can be used either with a batch or database backend.
func (cs *canonicalStore[T]) add(backend ethdb.KeyValueWriter, period uint64, value T) error {
	if !cs.periods.canExpand(period) {
		return fmt.Errorf("period expansion is not allowed, first: %d, next: %d, period: %d", cs.periods.Start, cs.periods.End, period)
	}
	enc, err := rlp.EncodeToBytes(value)
	if err != nil {
		return err
	}
	if err := backend.Put(cs.databaseKey(period), enc); err != nil {
		return err
	}
	cs.cache.Add(period, value)
	cs.periods.expand(period)
	return nil
}

// deleteFrom removes items starting from the given period.
func (cs *canonicalStore[T]) deleteFrom(db ethdb.KeyValueWriter, fromPeriod uint64) (deleted periodRange) {
	keepRange, deleteRange := cs.periods.split(fromPeriod)
	deleteRange.each(func(period uint64) {
		db.Delete(cs.databaseKey(period))
		cs.cache.Remove(period)
	})
	cs.periods = keepRange
	return deleteRange
}

// get returns the item at the given period or the null value of the given type
// if no item is present.
func (cs *canonicalStore[T]) get(backend ethdb.KeyValueReader, period uint64) (T, bool) {
	var null, value T
	if !cs.periods.contains(period) {
		return null, false
	}
	if value, ok := cs.cache.Get(period); ok {
		return value, true
	}
	enc, err := backend.Get(cs.databaseKey(period))
	if err != nil {
		log.Error("Canonical store value not found", "period", period, "start", cs.periods.Start, "end", cs.periods.End)
		return null, false
	}
	if err := rlp.DecodeBytes(enc, &value); err != nil {
		log.Error("Error decoding canonical store value", "error", err)
		return null, false
	}
	cs.cache.Add(period, value)
	return value, true
}
