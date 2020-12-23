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

package shed

import (
	"encoding/binary"

	"github.com/syndtr/goleveldb/leveldb"
)

// Uint64Field provides a way to have a simple counter in the database.
// It transparently encodes uint64 type value to bytes.
type Uint64Field struct {
	db  *DB
	key []byte
}

// NewUint64Field returns a new Uint64Field.
// It validates its name and type against the database schema.
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

// Get retrieves a uint64 value from the database.
// If the value is not found in the database a 0 value
// is returned and no error.
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

// Put encodes uin64 value and stores it in the database.
func (f Uint64Field) Put(val uint64) (err error) {
	return f.db.Put(f.key, encodeUint64(val))
}

// PutInBatch stores a uint64 value in a batch
// that can be saved later in the database.
func (f Uint64Field) PutInBatch(batch *leveldb.Batch, val uint64) {
	batch.Put(f.key, encodeUint64(val))
}

// Inc increments a uint64 value in the database.
// This operation is not goroutine save.
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

// IncInBatch increments a uint64 value in the batch
// by retreiving a value from the database, not the same batch.
// This operation is not goroutine save.
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

// Dec decrements a uint64 value in the database.
// This operation is not goroutine save.
// The field is protected from overflow to a negative value.
func (f Uint64Field) Dec() (val uint64, err error) {
	val, err = f.Get()
	if err != nil {
		if err == leveldb.ErrNotFound {
			val = 0
		} else {
			return 0, err
		}
	}
	if val != 0 {
		val--
	}
	return val, f.Put(val)
}

// DecInBatch decrements a uint64 value in the batch
// by retreiving a value from the database, not the same batch.
// This operation is not goroutine save.
// The field is protected from overflow to a negative value.
func (f Uint64Field) DecInBatch(batch *leveldb.Batch) (val uint64, err error) {
	val, err = f.Get()
	if err != nil {
		if err == leveldb.ErrNotFound {
			val = 0
		} else {
			return 0, err
		}
	}
	if val != 0 {
		val--
	}
	f.PutInBatch(batch, val)
	return val, nil
}

// encode transforms uint64 to 8 byte long
// slice in big endian encoding.
func encodeUint64(val uint64) (b []byte) {
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, val)
	return b
}
