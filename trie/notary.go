// Copyright 2020 The go-ethereum Authors
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

package trie

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

// KeyValueNotary tracks which keys have been accessed through a key-value reader
// with te scope of verifying if certain proof datasets are maliciously bloated.
type KeyValueNotary struct {
	ethdb.KeyValueReader
	reads map[string]struct{}
}

// NewKeyValueNotary wraps a key-value database with an access notary to track
// which items have bene accessed.
func NewKeyValueNotary(db ethdb.KeyValueReader) *KeyValueNotary {
	return &KeyValueNotary{
		KeyValueReader: db,
		reads:          make(map[string]struct{}),
	}
}

// Get retrieves an item from the underlying database, but also tracks it as an
// accessed slot for bloat checks.
func (k *KeyValueNotary) Get(key []byte) ([]byte, error) {
	k.reads[string(key)] = struct{}{}
	return k.KeyValueReader.Get(key)
}

// Accessed returns s snapshot of the original key-value store containing only the
// data accessed through the notary.
func (k *KeyValueNotary) Accessed() ethdb.KeyValueStore {
	db := memorydb.New()
	for keystr := range k.reads {
		key := []byte(keystr)
		val, _ := k.KeyValueReader.Get(key)
		db.Put(key, val)
	}
	return db
}
