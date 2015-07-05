// Copyright 2014 The go-ethereum Authors
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

package ethdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db map[string][]byte
}

func NewMemDatabase() (*MemDatabase, error) {
	db := &MemDatabase{db: make(map[string][]byte)}

	return db, nil
}

func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Set(key []byte, value []byte) {
	db.Put(key, value)
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	return db.db[string(key)], nil
}

func (db *MemDatabase) Keys() [][]byte {
	keys := [][]byte{}
	for key, _ := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

/*
func (db *MemDatabase) GetKeys() []*common.Key {
	data, _ := db.Get([]byte("KeyRing"))

	return []*common.Key{common.NewKeyFromBytes(data)}
}
*/

func (db *MemDatabase) Delete(key []byte) error {
	delete(db.db, string(key))

	return nil
}

func (db *MemDatabase) Print() {
	for key, val := range db.db {
		fmt.Printf("%x(%d): ", key, len(key))
		node := common.NewValueFromBytes(val)
		fmt.Printf("%q\n", node.Val)
	}
}

func (db *MemDatabase) Close() {
}

func (db *MemDatabase) LastKnownTD() []byte {
	data, _ := db.Get([]byte("LastKnownTotalDifficulty"))

	if len(data) == 0 || data == nil {
		data = []byte{0x0}
	}

	return data
}

func (db *MemDatabase) Flush() error {
	return nil
}

func (db *MemDatabase) NewBatch() Batch {
	return &memBatch{db: db}
}

type kv struct{ k, v []byte }

type memBatch struct {
	db     *MemDatabase
	writes []kv
}

func (w *memBatch) Put(key, value []byte) error {
	w.writes = append(w.writes, kv{key, common.CopyBytes(value)})
	return nil
}

func (w *memBatch) Write() error {
	for _, kv := range w.writes {
		w.db.db[string(kv.k)] = kv.v
	}
	return nil
}
