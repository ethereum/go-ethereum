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
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var secureKeyPrefix = []byte("secure-key-")

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
	*Trie

	hash        hash.Hash
	hashKeyBuf  []byte
	secKeyBuf   []byte
	secKeyCache map[string][]byte
}

// NewSecure creates a trie with an existing root node from db.
//
// If root is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty. Otherwise, New will panic if db is nil
// and returns MissingNodeError if the root node cannot be found.
// Accessing the trie loads nodes from db on demand.
func NewSecure(root common.Hash, db Database) (*SecureTrie, error) {
	if db == nil {
		panic("NewSecure called with nil database")
	}
	trie, err := New(root, db)
	if err != nil {
		return nil, err
	}
	return &SecureTrie{
		Trie:        trie,
		secKeyCache: make(map[string][]byte),
	}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *SecureTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled trie error: %v", err)
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureTrie) TryGet(key []byte) ([]byte, error) {
	return t.Trie.TryGet(t.hashKey(key))
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *SecureTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled trie error: %v", err)
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureTrie) TryUpdate(key, value []byte) error {
	hk := t.hashKey(key)
	err := t.Trie.TryUpdate(hk, value)
	if err != nil {
		return err
	}
	t.secKeyCache[string(hk)] = common.CopyBytes(key)
	return nil
}

// Delete removes any existing value for key from the trie.
func (t *SecureTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil && glog.V(logger.Error) {
		glog.Errorf("Unhandled trie error: %v", err)
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureTrie) TryDelete(key []byte) error {
	hk := t.hashKey(key)
	delete(t.secKeyCache, string(hk))
	return t.Trie.TryDelete(hk)
}

// GetKey returns the sha3 preimage of a hashed key that was
// previously used to store a value.
func (t *SecureTrie) GetKey(shaKey []byte) []byte {
	if key, ok := t.secKeyCache[string(shaKey)]; ok {
		return key
	}
	key, _ := t.Trie.db.Get(t.secKey(shaKey))
	return key
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *SecureTrie) Commit() (root common.Hash, err error) {
	return t.CommitTo(t.db)
}

// CommitTo writes all nodes and the secure hash pre-images to the given database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes from
// the trie's database. Calling code must ensure that the changes made to db are
// written back to the trie's attached database before using the trie.
func (t *SecureTrie) CommitTo(db DatabaseWriter) (root common.Hash, err error) {
	if len(t.secKeyCache) > 0 {
		for hk, key := range t.secKeyCache {
			if err := db.Put(t.secKey([]byte(hk)), key); err != nil {
				return common.Hash{}, err
			}
		}
		t.secKeyCache = make(map[string][]byte)
	}
	n, err := t.hashRoot(db)
	if err != nil {
		return (common.Hash{}), err
	}
	t.root = n
	return common.BytesToHash(n.(hashNode)), nil
}

func (t *SecureTrie) secKey(key []byte) []byte {
	t.secKeyBuf = append(t.secKeyBuf[:0], secureKeyPrefix...)
	t.secKeyBuf = append(t.secKeyBuf, key...)
	return t.secKeyBuf
}

func (t *SecureTrie) hashKey(key []byte) []byte {
	if t.hash == nil {
		t.hash = sha3.NewKeccak256()
		t.hashKeyBuf = make([]byte, 32)
	}
	t.hash.Reset()
	t.hash.Write(key)
	t.hashKeyBuf = t.hash.Sum(t.hashKeyBuf[:0])
	return t.hashKeyBuf
}
