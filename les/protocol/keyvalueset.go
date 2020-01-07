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

package protocol

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"
)

// ErrNonexistentEntry is returned if the specified key is non-existent.
var ErrNonexistentEntry = errors.New("the entry is non-existent")

// KeyValueEntry is the entry contained in a List or Map
// which can be extended with no limitaion.
type KeyValueEntry struct {
	Key   string
	Value rlp.RawValue
}

// KeyValueList is a set of entries in list format.
//
// Usually KeyValueList is used as the container for
// protocol handshake.
type KeyValueList []KeyValueEntry

// KeyValueMap is a set of entires in map format.
// All entires is identified with its key and saved
// in RLP-encoded format.
//
// Usually KeyValueMap is used as the container for
// protocol handshake.
type KeyValueMap map[string]rlp.RawValue

// Add adds a new entry with specified key and value into list.
func (l KeyValueList) Add(key string, val interface{}) KeyValueList {
	var entry KeyValueEntry
	entry.Key = key
	if val == nil {
		val = uint64(0) // Use empty uint64 as default value
	}
	enc, err := rlp.EncodeToBytes(val)
	if err == nil {
		entry.Value = enc
	}
	return append(l, entry)
}

// ToMap converts list format to map format. Also returns
// the total size of converted map.
func (l KeyValueList) ToMap() (KeyValueMap, uint64) {
	m := make(KeyValueMap)
	var size uint64
	for _, entry := range l {
		m[entry.Key] = entry.Value
		size += uint64(len(entry.Key)) + uint64(len(entry.Value)) + 8
	}
	return m, size
}

// Get retrieves contained entry with specified key, decode the
// retrieved data in the provided container(interface).
func (m KeyValueMap) Get(key string, val interface{}) error {
	enc, ok := m[key]
	if !ok {
		return ErrNonexistentEntry
	}
	if val == nil {
		return nil
	}
	return rlp.DecodeBytes(enc, val)
}
