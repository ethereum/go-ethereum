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

package state

import (
	"encoding"
	"encoding/json"
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

// ErrNotFound is returned when no results are returned from the database
var ErrNotFound = errors.New("ErrorNotFound")

// ErrInvalidArgument is returned when the argument type does not match the expected type
var ErrInvalidArgument = errors.New("ErrorInvalidArgument")

// Store defines methods required to get, set, delete values for different keys
// and close the underlying resources.
type Store interface {
	Get(key string, i interface{}) (err error)
	Put(key string, i interface{}) (err error)
	Delete(key string) (err error)
	Close() error
}

// DBStore uses LevelDB to store values.
type DBStore struct {
	db *leveldb.DB
}

// NewDBStore creates a new instance of DBStore.
func NewDBStore(path string) (s *DBStore, err error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &DBStore{
		db: db,
	}, nil
}

// NewInmemoryStore returns a new instance of DBStore. To be used only in tests and simulations.
func NewInmemoryStore() *DBStore {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		panic(err)
	}
	return &DBStore{
		db: db,
	}
}

// Get retrieves a persisted value for a specific key. If there is no results
// ErrNotFound is returned. The provided parameter should be either a byte slice or
// a struct that implements the encoding.BinaryUnmarshaler interface
func (s *DBStore) Get(key string, i interface{}) (err error) {
	has, err := s.db.Has([]byte(key), nil)
	if err != nil || !has {
		return ErrNotFound
	}

	data, err := s.db.Get([]byte(key), nil)
	if err == leveldb.ErrNotFound {
		return ErrNotFound
	}

	unmarshaler, ok := i.(encoding.BinaryUnmarshaler)
	if !ok {
		return json.Unmarshal(data, i)
	}
	return unmarshaler.UnmarshalBinary(data)
}

// Put stores an object that implements Binary for a specific key.
func (s *DBStore) Put(key string, i interface{}) (err error) {
	var bytes []byte

	marshaler, ok := i.(encoding.BinaryMarshaler)
	if !ok {
		if bytes, err = json.Marshal(i); err != nil {
			return err
		}
	} else {
		if bytes, err = marshaler.MarshalBinary(); err != nil {
			return err
		}
	}

	return s.db.Put([]byte(key), bytes, nil)
}

// Delete removes entries stored under a specific key.
func (s *DBStore) Delete(key string) (err error) {
	return s.db.Delete([]byte(key), nil)
}

// Close releases the resources used by the underlying LevelDB.
func (s *DBStore) Close() error {
	return s.db.Close()
}
