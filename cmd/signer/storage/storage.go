// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
//

package storage

import (
	"fmt"
)

type Storage interface {
	// Put stores a value by key. 0-length keys results in no-op
	Put(key, value string)
	// Get returns the previously stored value, or the empty string if it does not exist or key is of 0-length
	Get(key string) string
	// New creates a new (sub) namespace for the storage
	New(namespace string) Storage
}

// EphemeralStorage is an in-memory storage that does
// not persist values to disk. Mainly used for testing
type EphemeralStorage struct {
	data      map[string]string
	namespace string
}

func (s *EphemeralStorage) Put(key, value string) {
	if len(key) == 0 {
		return
	}
	key = fmt.Sprintf("%s.%s", s.namespace, key)
	fmt.Printf("storage: put %v -> %v\n", key, value)
	s.data[key] = value
}

func (s *EphemeralStorage) Get(key string) string {

	if len(key) == 0 {
		return ""
	}
	key = fmt.Sprintf("%s.%s", s.namespace, key)
	fmt.Printf("storage: get %v\n", key)
	if v, exist := s.data[key]; exist {
		return v
	}
	return ""
}

func (s *EphemeralStorage) New(namespace string) Storage {
	child := &EphemeralStorage{
		data:      make(map[string]string),
		namespace: fmt.Sprintf("%s.%s", s.namespace, namespace),
	}
	return child
}

func NewEphemeralStorage() Storage {
	s := &EphemeralStorage{
		data:      make(map[string]string),
		namespace: "root",
	}
	return s
}
