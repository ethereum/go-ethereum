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

package light

import (
	"context"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// LightTrie is an ODR-capable wrapper around trie.SecureTrie
type LightTrie struct {
	trie *trie.SecureTrie
	id   *TrieID
	odr  OdrBackend
	db   ethdb.Database
}

// NewLightTrie creates a new LightTrie instance. It doesn't instantly try to
// access the db or network and retrieve the root node, it only initializes its
// encapsulated SecureTrie at the first actual operation.
func NewLightTrie(id *TrieID, odr OdrBackend, useFakeMap bool) *LightTrie {
	return &LightTrie{
		// SecureTrie is initialized before first request
		id:  id,
		odr: odr,
		db:  odr.Database(),
	}
}

// retrieveKey retrieves a single key, returns true and stores nodes in local
// database if successful
func (t *LightTrie) retrieveKey(ctx context.Context, key []byte) bool {
	r := &TrieRequest{Id: t.id, Key: crypto.Keccak256(key)}
	return t.odr.Retrieve(ctx, r) == nil
}

// do tries and retries to execute a function until it returns with no error or
// an error type other than MissingNodeError
func (t *LightTrie) do(ctx context.Context, key []byte, fn func() error) error {
	err := fn()
	for err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
		if !t.retrieveKey(ctx, key) {
			break
		}
		err = fn()
	}
	return err
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *LightTrie) Get(ctx context.Context, key []byte) (res []byte, err error) {
	err = t.do(ctx, key, func() (err error) {
		if t.trie == nil {
			t.trie, err = trie.NewSecure(t.id.Root, t.db, 0)
		}
		if err == nil {
			res, err = t.trie.TryGet(key)
		}
		return
	})
	return
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *LightTrie) Update(ctx context.Context, key, value []byte) (err error) {
	err = t.do(ctx, key, func() (err error) {
		if t.trie == nil {
			t.trie, err = trie.NewSecure(t.id.Root, t.db, 0)
		}
		if err == nil {
			err = t.trie.TryUpdate(key, value)
		}
		return
	})
	return
}

// Delete removes any existing value for key from the trie.
func (t *LightTrie) Delete(ctx context.Context, key []byte) (err error) {
	err = t.do(ctx, key, func() (err error) {
		if t.trie == nil {
			t.trie, err = trie.NewSecure(t.id.Root, t.db, 0)
		}
		if err == nil {
			err = t.trie.TryDelete(key)
		}
		return
	})
	return
}
