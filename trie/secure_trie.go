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

	hash       hash.Hash
	secKeyBuf  []byte
	hashKeyBuf []byte
}

// NewSecure creates a trie with an existing root node from db.
//
// If root is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty. Otherwise, New will panics if db is nil
// and returns ErrMissingRoot if the root node cannpt be found.
// Accessing the trie loads nodes from db on demand.
func NewSecure(root common.Hash, db Database) (*SecureTrie, error) {
	if db == nil {
		panic("NewSecure called with nil database")
	}
	trie, err := New(root, db)
	if err != nil {
		return nil, err
	}
	return &SecureTrie{Trie: trie}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *SecureTrie) Get(key []byte) []byte {
	return t.Trie.Get(t.hashKey(key))
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *SecureTrie) Update(key, value []byte) {
	hk := t.hashKey(key)
	t.Trie.Update(hk, value)
	t.Trie.db.Put(t.secKey(hk), key)
}

// Delete removes any existing value for key from the trie.
func (t *SecureTrie) Delete(key []byte) {
	t.Trie.Delete(t.hashKey(key))
}

// GetKey returns the sha3 preimage of a hashed key that was
// previously used to store a value.
func (t *SecureTrie) GetKey(shaKey []byte) []byte {
	key, _ := t.Trie.db.Get(t.secKey(shaKey))
	return key
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
