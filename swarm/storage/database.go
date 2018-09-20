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

// this is a clone of an earlier state of the ethereum ethdb/database
// no need for queueing/caching

import (
	"fmt"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const openFileLimit = 128

type LDBDatabase struct {
	db *leveldb.DB
}

func NewLDBDatabase(file string) (*LDBDatabase, error) {
	// Open the db
	db, err := leveldb.OpenFile(file, &opt.Options{OpenFilesCacheCapacity: openFileLimit})
	if err != nil {
		return nil, err
	}

	database := &LDBDatabase{db: db}

	return database, nil
}

func (db *LDBDatabase) Put(key []byte, value []byte) {
	metrics.GetOrRegisterCounter("ldbdatabase.put", nil).Inc(1)

	err := db.db.Put(key, value, nil)
	if err != nil {
		fmt.Println("Error put", err)
	}
}

func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	metrics.GetOrRegisterCounter("ldbdatabase.get", nil).Inc(1)

	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LDBDatabase) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *LDBDatabase) LastKnownTD() []byte {
	data, _ := db.Get([]byte("LTD"))

	if len(data) == 0 {
		data = []byte{0x0}
	}

	return data
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	metrics.GetOrRegisterCounter("ldbdatabase.newiterator", nil).Inc(1)

	return db.db.NewIterator(nil, nil)
}

func (db *LDBDatabase) Write(batch *leveldb.Batch) error {
	metrics.GetOrRegisterCounter("ldbdatabase.write", nil).Inc(1)

	return db.db.Write(batch, nil)
}

func (db *LDBDatabase) Close() {
	// Close the leveldb database
	db.db.Close()
}
