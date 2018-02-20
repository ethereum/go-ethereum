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

// wrapper of db-s to provide mockable custom local chunk store access to syncer
type DBAPI struct {
	db  *DbStore
	loc *LocalStore
}

func NewDBAPI(loc *LocalStore) *DBAPI {
	return &DBAPI{loc.DbStore.(*DbStore), loc}
}

// to obtain the chunks from key or request db entry only
func (self *DBAPI) Get(key Key) (*Chunk, error) {
	return self.loc.Get(key)
}

// current storage counter of chunk db
func (self *DBAPI) CurrentBucketStorageIndex(po uint8) uint64 {
	return self.db.CurrentBucketStorageIndex(po)
}

// iteration storage counter and proximity order
func (self *DBAPI) Iterator(from uint64, to uint64, po uint8, f func(Key, uint64) bool) error {
	return self.db.SyncIterator(from, to, po, f)
}

// to obtain the chunks from key or request db entry only
func (self *DBAPI) GetOrCreateRequest(key Key) (*Chunk, bool) {
	return self.loc.GetOrCreateRequest(key)
}

// to obtain the chunks from key or request db entry only
func (self *DBAPI) Put(chunk *Chunk) {
	self.loc.Put(chunk)
}
