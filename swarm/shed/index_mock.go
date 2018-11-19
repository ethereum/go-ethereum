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
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/syndtr/goleveldb/leveldb"
)

// MockIndex provides a way to inject a mock.NodeStore to store
// data centrally instead on provided DB. DB is used just for schema
// validation and identifying byte prefix for the particular index
// that is mocked.
// Iterator functions are not implemented and MockIndex can not replace
// indexes that rely on them.
// It implements IndexField interface.
type MockIndex struct {
	store           *mock.NodeStore
	prefix          []byte
	encodeKeyFunc   func(fields IndexItem) (key []byte, err error)
	decodeKeyFunc   func(key []byte) (e IndexItem, err error)
	encodeValueFunc func(fields IndexItem) (value []byte, err error)
	decodeValueFunc func(value []byte) (e IndexItem, err error)
}

// NewMockIndex returns a new MockIndex instance with defined name and
// encoding functions. The name must be unique and will be validated
// on database schema for a key prefix byte.
// The data will not be saved on the DB itself, but using a provided
// mock.NodeStore.
func (db *DB) NewMockIndex(store *mock.NodeStore, name string, funcs IndexFuncs) (f MockIndex, err error) {
	id, err := db.schemaIndexPrefix(name)
	if err != nil {
		return f, err
	}
	return MockIndex{
		store:           store,
		prefix:          []byte{id},
		encodeKeyFunc:   newIndexEncodeKeyFunc(funcs.EncodeKey, id),
		decodeKeyFunc:   newDecodeKeyFunc(funcs.DecodeKey),
		encodeValueFunc: funcs.EncodeValue,
		decodeValueFunc: funcs.DecodeValue,
	}, nil
}

// Get accepts key fields represented as IndexItem to retrieve a
// value from the index and return maximum available information
// from the index represented as another IndexItem.
func (f MockIndex) Get(keyFields IndexItem) (out IndexItem, err error) {
	key, err := f.encodeKeyFunc(keyFields)
	if err != nil {
		return out, err
	}
	value, err := f.store.Get(key)
	if err != nil {
		return out, err
	}
	out, err = f.decodeValueFunc(value)
	if err != nil {
		return out, err
	}
	return out.Join(keyFields), nil
}

// Put accepts IndexItem to encode information from it
// and save it to the database.
func (f MockIndex) Put(i IndexItem) (err error) {
	key, err := f.encodeKeyFunc(i)
	if err != nil {
		return err
	}
	value, err := f.encodeValueFunc(i)
	if err != nil {
		return err
	}
	return f.store.Put(key, value)
}

// PutInBatch is the same as Put method.
// Batch is ignored and the data is saved to the mock store instantly.
func (f MockIndex) PutInBatch(_ *leveldb.Batch, i IndexItem) (err error) {
	return f.Put(i)
}

// Delete accepts IndexItem to remove a key/value pair
// form the database based on its fields.
func (f MockIndex) Delete(keyFields IndexItem) (err error) {
	key, err := f.encodeKeyFunc(keyFields)
	if err != nil {
		return err
	}
	return f.store.Delete(key)
}

// DeleteInBatch is the same as Delete just the operation.
// Batch is ignored and the data is deleted on the mock store instantly.
func (f MockIndex) DeleteInBatch(_ *leveldb.Batch, keyFields IndexItem) (err error) {
	return f.Delete(keyFields)
}
