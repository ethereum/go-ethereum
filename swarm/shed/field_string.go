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
	"github.com/syndtr/goleveldb/leveldb"
)

// StringField is the most simple field implementation
// that stores an arbitrary string under a specific LevelDB key.
type StringField struct {
	db  *DB
	key []byte
}

// NewStringField retruns a new Instance of StringField.
// It validates its name and type against the database schema.
func (db *DB) NewStringField(name string) (f StringField, err error) {
	key, err := db.schemaFieldKey(name, "string")
	if err != nil {
		return f, err
	}
	return StringField{
		db:  db,
		key: key,
	}, nil
}

// Get returns a string value from database.
// If the value is not found, an empty string is returned
// an no error.
func (f StringField) Get() (val string, err error) {
	b, err := f.db.Get(f.key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// Put stores a string in the database.
func (f StringField) Put(val string) (err error) {
	return f.db.Put(f.key, []byte(val))
}

// PutInBatch stores a string in a batch that can be
// saved later in database.
func (f StringField) PutInBatch(batch *leveldb.Batch, val string) {
	batch.Put(f.key, []byte(val))
}
