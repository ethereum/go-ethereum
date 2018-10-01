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

// +build js

package ethdb

import (
	"errors"
)

var errNotSupported = errors.New("ethdb: not supported")

type LDBDatabase struct {
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	return nil, errNotSupported
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return ""
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return errNotSupported
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	return false, errNotSupported
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	return nil, errNotSupported
}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	return errNotSupported
}

func (db *LDBDatabase) Close() {
}

// Meter configures the database metrics collectors and
func (db *LDBDatabase) Meter(prefix string) {
}

func (db *LDBDatabase) NewBatch() Batch {
	return nil
}
