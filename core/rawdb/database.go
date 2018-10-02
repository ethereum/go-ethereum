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

package rawdb

import (
	"github.com/ethereum/go-ethereum/database"
	"github.com/ethereum/go-ethereum/internal/keyvalue"
)

// freezerdb is a databse wrapper that enabled freezer data retrievals.
type freezerdb struct {
	database.KeyValueStore
	database.Ancienter
}

// nofreezedb is a database wrapper that disables freezer data retrievals.
type nofreezedb struct {
	database.KeyValueStore
}

// Frozen returns nil as we don't have a backing chain freezer.
func (db *nofreezedb) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errOutOfBounds
}

// NewDatabase creates a high level database on top of a given key-value data
// store without a freezer moving immutable chain segments into cold storage.
func NewDatabase(db database.KeyValueStore) database.Database {
	return &nofreezedb{
		KeyValueStore: db,
	}
}

// NewDatabaseWithFreezer creates a high level database on top of a given key-
// value data store with a freezer moving immutable chain segments into cold
// storage.
func NewDatabaseWithFreezer(db database.KeyValueStore, freezer string, namespace string) (database.Database, error) {
	frdb, err := newFreezer(freezer, namespace)
	if err != nil {
		return nil, err
	}
	go frdb.freeze(db)

	return &freezerdb{
		KeyValueStore: db,
		Ancienter:     frdb,
	}, nil
}

// NewMemoryDatabase creates an ephemeral in-memory key-value database without a
// freezer moving immutable chain segments into cold storage.
func NewMemoryDatabase() database.Database {
	return NewDatabase(keyvalue.NewMemoryDatabase())
}

// NewMemoryDatabaseWithCap creates an ephemeral in-memory key-value database with
// an initial starting capacity, but without a freezer moving immutable chain
// segments into cold storage.
func NewMemoryDatabaseWithCap(size int) database.Database {
	return NewDatabase(keyvalue.NewMemoryDatabaseWithCap(size))
}

// NewLeveldbDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func NewLeveldbDatabase(file string, cache int, handles int, namespace string) (database.Database, error) {
	db, err := keyvalue.NewLeveldbDatabase(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	return NewDatabase(db), nil
}

// NewLeveldbDatabaseWithFreezer creates a persistent key-value database with a freezer
// moving immutable chain segments into cold storage.
func NewLeveldbDatabaseWithFreezer(file string, cache int, handles int, freezer string, namespace string) (database.Database, error) {
	kvdb, err := keyvalue.NewLeveldbDatabase(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	frdb, err := NewDatabaseWithFreezer(kvdb, freezer, namespace)
	if err != nil {
		kvdb.Close()
		return nil, err
	}
	return frdb, nil
}
