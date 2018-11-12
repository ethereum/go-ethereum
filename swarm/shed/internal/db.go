// Copyright 2018 The go-ethereum Authors
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

package internal

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const openFileLimit = 128

type DB struct {
	ldb *leveldb.DB
}

func NewDB(path string) (db *DB, err error) {
	ldb, err := leveldb.OpenFile(path, &opt.Options{OpenFilesCacheCapacity: openFileLimit})
	if err != nil {
		return nil, err
	}
	db = &DB{ldb: ldb}

	if _, err = db.getSchema(); err != nil {
		if err == leveldb.ErrNotFound {
			if err = db.putSchema(schema{
				Fields:  make(map[string]fieldSpec),
				Indexes: make(map[byte]indexSpec),
			}); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return db, nil
}

func (db *DB) Put(key []byte, value []byte) (err error) {
	metrics.GetOrRegisterCounter("DB.put", nil).Inc(1)

	return db.ldb.Put(key, value, nil)
}

func (db *DB) Get(key []byte) (value []byte, err error) {
	metrics.GetOrRegisterCounter("DB.get", nil).Inc(1)

	return db.ldb.Get(key, nil)
}

func (db *DB) Delete(key []byte) error {
	return db.ldb.Delete(key, nil)
}

func (db *DB) NewIterator() iterator.Iterator {
	metrics.GetOrRegisterCounter("DB.newiterator", nil).Inc(1)

	return db.ldb.NewIterator(nil, nil)
}

func (db *DB) WriteBatch(batch *leveldb.Batch) error {
	metrics.GetOrRegisterCounter("DB.write", nil).Inc(1)

	return db.ldb.Write(batch, nil)
}

func (db *DB) Close() (err error) {
	return db.ldb.Close()
}
