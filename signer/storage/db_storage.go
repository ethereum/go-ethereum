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

package storage

import (
	"github.com/ethereum/go-ethereum/cmd/clef/dbutil"
)

// DBStorageAPI is a storage api which is backed by a general purpose database
type DBStorageAPI struct {
	kvstore *dbutil.KVStore
}

// Get returns the previously stored value, or an error if the key is 0-length
// or unknown.
func (api *DBStorageAPI) Get(key string) (string, error) {
	return api.kvstore.Get(key)
}

// Put stores a value by key. 0-length keys results in noop.
func (api *DBStorageAPI) Put(key, value string) {
	api.kvstore.Put(key, value)
}

// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
func (api *DBStorageAPI) Del(key string) {
	api.kvstore.Del(key)
}
