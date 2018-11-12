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
	"encoding/binary"

	"github.com/syndtr/goleveldb/leveldb"
)

type Uint64Field struct {
	db  *DB
	key []byte
}

func (db *DB) NewUint64Field(name string) (f Uint64Field, err error) {
	key, err := db.schemaFieldKey(name, "uint64")
	if err != nil {
		return f, err
	}
	return Uint64Field{
		db:  db,
		key: key,
	}, nil
}

func (f Uint64Field) Get() (val uint64, err error) {
	b, err := f.db.Get(f.key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

func (f Uint64Field) Put(val uint64) (err error) {
	return f.db.Put(f.key, encodeUint64(val))
}

func (f Uint64Field) PutInBatch(batch *leveldb.Batch, val uint64) {
	batch.Put(f.key, encodeUint64(val))
}

func (f Uint64Field) Inc() (val uint64, err error) {
	val, err = f.Get()
	if err != nil {
		if err == leveldb.ErrNotFound {
			val = 0
		} else {
			return 0, err
		}
	}
	val++
	return val, f.Put(val)
}

func (f Uint64Field) IncInBatch(batch *leveldb.Batch) (val uint64, err error) {
	val, err = f.Get()
	if err != nil {
		if err == leveldb.ErrNotFound {
			val = 0
		} else {
			return 0, err
		}
	}
	val++
	f.PutInBatch(batch, val)
	return val, nil
}

func encodeUint64(val uint64) (b []byte) {
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, val)
	return b
}
