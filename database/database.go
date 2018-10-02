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

// Package database defines the interfaces for an Ethereum data store.
package database

import "io"

// Reader wraps the Has and Get method of a backing data store.
type Reader interface {
	// Has retrieves if a key is present in the key-value data store.
	Has(key []byte) (bool, error)

	// Get retrieves the given key if it's present in the key-value data store.
	Get(key []byte) ([]byte, error)
}

// Writer wraps the Put method of a backing data store.
type Writer interface {
	// Put inserts the given value into the key-value data store.
	Put(key []byte, value []byte) error
}

// Deleter wraps the Delete method of a backing data store.
type Deleter interface {
	// Delete removes the key from the key-value data store.
	Delete(key []byte) error
}

// Stater wraps the Stat method of a backing data store.
type Stater interface {
	// Stat returns a particular internal stat of the database.
	Stat(property string) (string, error)
}

// KeyValueStore contains all the methods required to allow handling different
// key-value data stores backing the high level database.
type KeyValueStore interface {
	Reader
	Writer
	Deleter
	Batcher
	Iteratorer
	Stater
	io.Closer
}

// Ancienter wraps the Ancient method for a backing immutable chain data store.
type Ancienter interface {
	// Ancient retrieves an ancient binary blob from the append-only immutable files.
	Ancient(kind string, number uint64) ([]byte, error)
}

// AncientReader contains the methods required to access both key-value as well as
// immutable ancient data.
type AncientReader interface {
	Reader
	Ancienter
}

// Database contains all the methods required by the high level database to not
// only access
type Database interface {
	AncientReader
	Writer
	Deleter
	Batcher
	Iteratorer
	Stater
	io.Closer
}
