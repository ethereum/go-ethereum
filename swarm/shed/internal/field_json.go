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
	"encoding/json"

	"github.com/syndtr/goleveldb/leveldb"
)

type JSONField struct {
	db  *DB
	key []byte
}

func (db *DB) NewJSONField(name string) (f JSONField, err error) {
	key, err := db.schemaFieldKey(name, "json")
	if err != nil {
		return f, err
	}
	return JSONField{
		db:  db,
		key: key,
	}, nil
}

func (f JSONField) Unmarshal(val interface{}) (err error) {
	b, err := f.db.Get(f.key)
	if err != nil {
		// Q: should we ignore not found
		// if err == leveldb.ErrNotFound {
		// 	return nil
		// }
		return err
	}
	return json.Unmarshal(b, val)
}

func (f JSONField) Put(val interface{}) (err error) {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return f.db.Put(f.key, b)
}

func (f JSONField) PutInBatch(batch *leveldb.Batch, val interface{}) (err error) {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	batch.Put(f.key, b)
	return nil
}
