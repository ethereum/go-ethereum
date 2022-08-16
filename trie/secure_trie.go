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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// SecureTrie is the old name of StateTrie.
// Deprecated: use StateTrie.
type SecureTrie = StateTrie

// NewSecure creates a new StateTrie.
// Deprecated: use NewStateTrie.
func NewSecure(owner common.Hash, root common.Hash, db *Database) (*SecureTrie, error) {
	return NewStateTrie(owner, root, db)
}

// StateTrie wraps a trie with key hashing. In a secure trie, all
// access operations hash the key using keccak256. This prevents
// calling code from creating long chains of nodes that
// increase the access time.
//
// Contrary to a regular trie, a StateTrie can only be created with
// New and must have an attached database. The database also stores
// the preimage of each key.
//
// StateTrie is not safe for concurrent use.
type StateTrie struct {
	trie             Trie
	preimages        *preimageStore
	hashKeyBuf       [common.HashLength]byte
	secKeyCache      map[string][]byte
	secKeyCacheOwner *StateTrie // Pointer to self, replace the key cache on mismatch
}

// NewStateTrie creates a trie with an existing root node from a backing database
// and optional intermediate in-memory node pool.
//
// If root is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty. Otherwise, New will panic if db is nil
// and returns MissingNodeError if the root node cannot be found.
//
// Accessing the trie loads nodes from the database or node pool on demand.
// Loaded nodes are kept around until their 'cache generation' expires.
// A new cache generation is created by each call to Commit.
// cachelimit sets the number of past cache generations to keep.
func NewStateTrie(owner common.Hash, root common.Hash, db *Database) (*StateTrie, error) {
	if db == nil {
		panic("trie.NewSecure called without a database")
	}
	trie, err := New(owner, root, db)
	if err != nil {
		return nil, err
	}
	return &StateTrie{trie: *trie, preimages: db.preimages}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *StateTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *StateTrie) TryGet(key []byte) ([]byte, error) {
	return t.trie.TryGet(t.hashKey(key))
}

func (t *StateTrie) TryGetAccount(key []byte) (*types.StateAccount, error) {
	var ret types.StateAccount
	res, err := t.trie.TryGet(t.hashKey(key))
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
		return &ret, err
	}
	if res == nil {
		return nil, nil
	}
	err = rlp.DecodeBytes(res, &ret)
	return &ret, err
}

// TryGetAccountWithPreHashedKey does the same thing as TryGetAccount, however
// it expects a key that is already hashed. This constitutes an abstraction leak,
// since the client code needs to know the key format.
func (t *StateTrie) TryGetAccountWithPreHashedKey(key []byte) (*types.StateAccount, error) {
	var ret types.StateAccount
	res, err := t.trie.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
		return &ret, err
	}
	if res == nil {
		return nil, nil
	}
	err = rlp.DecodeBytes(res, &ret)
	return &ret, err
}

// TryGetNode attempts to retrieve a trie node by compact-encoded path. It is not
// possible to use keybyte-encoding as the path might contain odd nibbles.
func (t *StateTrie) TryGetNode(path []byte) ([]byte, int, error) {
	return t.trie.TryGetNode(path)
}

// TryUpdateAccount account will abstract the write of an account to the
// secure trie.
func (t *StateTrie) TryUpdateAccount(key []byte, acc *types.StateAccount) error {
	hk := t.hashKey(key)
	data, err := rlp.EncodeToBytes(acc)
	if err != nil {
		return err
	}
	if err := t.trie.TryUpdate(hk, data); err != nil {
		return err
	}
	t.getSecKeyCache()[string(hk)] = common.CopyBytes(key)
	return nil
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *StateTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
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
func (t *StateTrie) TryUpdate(key, value []byte) error {
	hk := t.hashKey(key)
	err := t.trie.TryUpdate(hk, value)
	if err != nil {
		return err
	}
	t.getSecKeyCache()[string(hk)] = common.CopyBytes(key)
	return nil
}

// Delete removes any existing value for key from the trie.
func (t *StateTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *StateTrie) TryDelete(key []byte) error {
	hk := t.hashKey(key)
	delete(t.getSecKeyCache(), string(hk))
	return t.trie.TryDelete(hk)
}

// TryDeleteACcount abstracts an account deletion from the trie.
func (t *StateTrie) TryDeleteAccount(key []byte) error {
	hk := t.hashKey(key)
	delete(t.getSecKeyCache(), string(hk))
	return t.trie.TryDelete(hk)
}

// GetKey returns the sha3 preimage of a hashed key that was
// previously used to store a value.
func (t *StateTrie) GetKey(shaKey []byte) []byte {
	if key, ok := t.getSecKeyCache()[string(shaKey)]; ok {
		return key
	}
	if t.preimages == nil {
		return nil
	}
	return t.preimages.preimage(common.BytesToHash(shaKey))
}

// Commit collects all dirty nodes in the trie and replace them with the
// corresponding node hash. All collected nodes(including dirty leaves if
// collectLeaf is true) will be encapsulated into a nodeset for return.
// The returned nodeset can be nil if the trie is clean(nothing to commit).
// All cached preimages will be also flushed if preimages recording is enabled.
// Once the trie is committed, it's not usable anymore. A new trie must
// be created with new root and updated trie database for following usage
func (t *StateTrie) Commit(collectLeaf bool) (common.Hash, *NodeSet, error) {
	// Write all the pre-images to the actual disk database
	if len(t.getSecKeyCache()) > 0 {
		if t.preimages != nil {
			preimages := make(map[common.Hash][]byte)
			for hk, key := range t.secKeyCache {
				preimages[common.BytesToHash([]byte(hk))] = key
			}
			t.preimages.insertPreimage(preimages)
		}
		t.secKeyCache = make(map[string][]byte)
	}
	// Commit the trie to its intermediate node database
	return t.trie.Commit(collectLeaf)
}

// Hash returns the root hash of StateTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *StateTrie) Hash() common.Hash {
	return t.trie.Hash()
}

// Copy returns a copy of StateTrie.
func (t *StateTrie) Copy() *StateTrie {
	return &StateTrie{
		trie:        *t.trie.Copy(),
		preimages:   t.preimages,
		secKeyCache: t.secKeyCache,
	}
}

// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (t *StateTrie) NodeIterator(start []byte) NodeIterator {
	return t.trie.NodeIterator(start)
}

// hashKey returns the hash of key as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (t *StateTrie) hashKey(key []byte) []byte {
	h := newHasher(false)
	h.sha.Reset()
	h.sha.Write(key)
	h.sha.Read(t.hashKeyBuf[:])
	returnHasherToPool(h)
	return t.hashKeyBuf[:]
}

// getSecKeyCache returns the current secure key cache, creating a new one if
// ownership changed (i.e. the current secure trie is a copy of another owning
// the actual cache).
func (t *StateTrie) getSecKeyCache() map[string][]byte {
	if t != t.secKeyCacheOwner {
		t.secKeyCacheOwner = t
		t.secKeyCache = make(map[string][]byte)
	}
	return t.secKeyCache
}
