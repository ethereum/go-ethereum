// Copyright 2015 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var secureKeyPrefix = []byte("secure-key-")

const secureKeyLength = 11 + 32 // Length of the above prefix + 32byte hash

type PersistentMap interface {
	Iterator() *Iterator
	Get(key []byte) []byte
	TryGet(key []byte) ([]byte, error)
	Update(key, value []byte)
	TryUpdate(key, value []byte) error
	Delete(key []byte)
	TryDelete(key []byte) error
	Commit() (root common.Hash, err error)
	CommitTo(db DatabaseWriter) (root common.Hash, err error)
}

// SecureTrie wraps a trie with key hashing. In a secure trie, all
// access operations hash the key using keccak256. This prevents
// calling code from creating long chains of nodes that
// increase the access time.
//
// Contrary to a regular trie, a SecureTrie can only be created with
// New and must have an attached database. The database also stores
// the preimage of each key.
//
// SecureTrie is not safe for concurrent use.
type SecureTrie struct {
	data             PersistentMap
	db               Database
	hashKeyBuf       [secureKeyLength]byte
	secKeyBuf        [200]byte
	secKeyCache      map[string][]byte
	secKeyCacheOwner *SecureTrie // Pointer to self, replace the key cache on mismatch
}

// NewSecure creates a secure persistent map from an existing map.
func NewSecure(pm PersistentMap, db Database) *SecureTrie {
	if pm == nil {
		panic("NewSecure called with nil persistent map")
	}
	if db == nil {
		panic("NewSecure called with nil database")
	}
	return &SecureTrie{data: pm, db: db}
}

// Get returns the value for key stored in the map.
// The value bytes must not be modified by the caller.
func (t *SecureTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled persistent map error: %v", err)
	}
	return res
}

// TryGet returns the value for key stored in the map.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureTrie) TryGet(key []byte) ([]byte, error) {
	return t.data.TryGet(t.hashKey(key))
}

// Update associates key with value in the map. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the map and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the map.
func (t *SecureTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled persistent map error: %v", err)
	}
}

// TryUpdate associates key with value in the map. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the map and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the map.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureTrie) TryUpdate(key, value []byte) error {
	hk := t.hashKey(key)
	err := t.data.TryUpdate(hk, value)
	if err != nil {
		return err
	}
	t.getSecKeyCache()[string(hk)] = common.CopyBytes(key)
	return nil
}

// Delete removes any existing value for key from the map.
func (t *SecureTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled persistent map error: %v", err)
	}
}

// TryDelete removes any existing value for key from the map.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureTrie) TryDelete(key []byte) error {
	hk := t.hashKey(key)
	delete(t.getSecKeyCache(), string(hk))
	return t.data.TryDelete(hk)
}

// GetKey returns the sha3 preimage of a hashed key that was
// previously used to store a value.
func (t *SecureTrie) GetKey(shaKey []byte) []byte {
	if key, ok := t.getSecKeyCache()[string(shaKey)]; ok {
		return key
	}
	key, _ := t.db.Get(t.secKey(shaKey))
	return key
}

// Commit writes all nodes and the secure hash pre-images to the database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *SecureTrie) Commit() (root common.Hash, err error) {
	if err := t.CommitPreimages(); err != nil {
		return common.Hash{}, err
	}
	return t.data.Commit()
}

func (t *SecureTrie) Iterator() *Iterator {
	return t.data.Iterator()
}

// CommitTo writes all nodes and the secure hash pre-images to the given database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes from
// the database. Calling code must ensure that the changes made to db are
// written back to the attached database before using the map.
func (t *SecureTrie) CommitTo(db DatabaseWriter) (root common.Hash, err error) {
	if err := t.CommitPreimages(); err != nil {
		return common.Hash{}, err
	}
	return t.data.CommitTo(db)
}

func (t *SecureTrie) CommitPreimages() error {
	if len(t.getSecKeyCache()) > 0 {
		for hk, key := range t.secKeyCache {
			if err := t.db.Put(t.secKey([]byte(hk)), key); err != nil {
				return err
			}
		}
		t.secKeyCache = make(map[string][]byte)
	}
	return nil
}

// secKey returns the database key for the preimage of key, as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (t *SecureTrie) secKey(key []byte) []byte {
	buf := append(t.secKeyBuf[:0], secureKeyPrefix...)
	buf = append(buf, key...)
	return buf
}

// hashKey returns the hash of key as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (t *SecureTrie) hashKey(key []byte) []byte {
	h := newHasher(0, 0)
	h.sha.Reset()
	h.sha.Write(key)
	buf := h.sha.Sum(t.hashKeyBuf[:0])
	returnHasherToPool(h)
	return buf
}

// getSecKeyCache returns the current secure key cache, creating a new one if
// ownership changed (i.e. the current secure map is a copy of another owning
// the actual cache).
func (t *SecureTrie) getSecKeyCache() map[string][]byte {
	if t != t.secKeyCacheOwner {
		t.secKeyCacheOwner = t
		t.secKeyCache = make(map[string][]byte)
	}
	return t.secKeyCache
}
