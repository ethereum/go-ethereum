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

package intervals

import (
	"github.com/syndtr/goleveldb/leveldb"
)

// DBStore uses LevelDB to store intervals.
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

// Get retrieves Intervals for a specific key. If there is no Intervals
// ErrNotFound is returned.
func (s *DBStore) Get(key string) (i *Intervals, err error) {
	k := []byte(key)
	has, err := s.db.Has(k, nil)
	if err != nil {
		return nil, ErrNotFound
	}
	if !has {
		return nil, ErrNotFound
	}
	data, err := s.db.Get(k, nil)
	if err == leveldb.ErrNotFound {
		err = ErrNotFound
	}
	i = &Intervals{}
	if err = i.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return i, err
}

// Put stores Intervals for a specific key.
func (s *DBStore) Put(key string, i *Intervals) (err error) {
	data, err := i.MarshalBinary()
	if err != nil {
		return err
	}
	return s.db.Put([]byte(key), data, nil)
}

// Delete removes Intervals stored under a specific key.
func (s *DBStore) Delete(key string) (err error) {
	return s.db.Delete([]byte(key), nil)
}

// Close releases the resources used by the underlying LevelDB.
func (s *DBStore) Close() error {
	return s.db.Close()
}
