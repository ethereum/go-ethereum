// Copyright 2025 go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-verkle"
)

// TransitionTrie is a trie that implements a façade design pattern, presenting
// a single interface to the old MPT trie and the new verkle/binary trie. Reads
// first from the overlay trie, and falls back to the base trie if the key isn't
// found. All writes go to the overlay trie.
type TransitionTrie struct {
	overlay *VerkleTrie
	base    *SecureTrie
	storage bool
}

// NewTransitionTrie creates a new TransitionTrie.
func NewTransitionTrie(base *SecureTrie, overlay *VerkleTrie, st bool) *TransitionTrie {
	return &TransitionTrie{
		overlay: overlay,
		base:    base,
		storage: st,
	}
}

// Base returns the base trie.
func (t *TransitionTrie) Base() *SecureTrie {
	return t.base
}

// Overlay returns the overlay trie.
func (t *TransitionTrie) Overlay() *VerkleTrie {
	return t.overlay
}

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (t *TransitionTrie) GetKey(key []byte) []byte {
	if key := t.overlay.GetKey(key); key != nil {
		return key
	}
	return t.base.GetKey(key)
}

// GetStorage returns the value for key stored in the trie. The value bytes must
// not be modified by the caller.
func (t *TransitionTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	val, err := t.overlay.GetStorage(addr, key)
	if err != nil {
		return nil, fmt.Errorf("get storage from overlay: %s", err)
	}
	if len(val) != 0 {
		return val, nil
	}
	// TODO also insert value into overlay
	return t.base.GetStorage(addr, key)
}

// PrefetchStorage attempts to resolve specific storage slots from the database
// to accelerate subsequent trie operations.
func (t *TransitionTrie) PrefetchStorage(addr common.Address, keys [][]byte) error {
	for _, key := range keys {
		if _, err := t.GetStorage(addr, key); err != nil {
			return err
		}
	}
	return nil
}

// GetAccount abstract an account read from the trie.
func (t *TransitionTrie) GetAccount(address common.Address) (*types.StateAccount, error) {
	data, err := t.overlay.GetAccount(address)
	if err != nil {
		// Post cancun, no indicator needs to be used to indicate that
		// an account was deleted in the overlay tree. If an error is
		// returned, then it's a genuine error, and not an indicator
		// that a tombstone was found.
		return nil, err
	}
	if data != nil {
		return data, nil
	}
	return t.base.GetAccount(address)
}

// PrefetchAccount attempts to resolve specific accounts from the database
// to accelerate subsequent trie operations.
func (t *TransitionTrie) PrefetchAccount(addresses []common.Address) error {
	for _, addr := range addresses {
		if _, err := t.GetAccount(addr); err != nil {
			return err
		}
	}
	return nil
}

// UpdateStorage associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie.
func (t *TransitionTrie) UpdateStorage(address common.Address, key []byte, value []byte) error {
	var v []byte
	if len(value) >= 32 {
		v = value[:32]
	} else {
		var val [32]byte
		copy(val[32-len(value):], value[:])
		v = val[:]
	}
	return t.overlay.UpdateStorage(address, key, v)
}

// UpdateAccount abstract an account write to the trie.
func (t *TransitionTrie) UpdateAccount(addr common.Address, account *types.StateAccount, codeLen int) error {
	// NOTE: before the rebase, this was saving the state root, so that OpenStorageTrie
	// could still work during a replay. This is no longer needed, as OpenStorageTrie
	// only needs to know what the account trie does now.
	return t.overlay.UpdateAccount(addr, account, codeLen)
}

// DeleteStorage removes any existing value for key from the trie. If a node was not
// found in the database, a trie.MissingNodeError is returned.
func (t *TransitionTrie) DeleteStorage(addr common.Address, key []byte) error {
	return t.overlay.DeleteStorage(addr, key)
}

// DeleteAccount abstracts an account deletion from the trie.
func (t *TransitionTrie) DeleteAccount(key common.Address) error {
	return t.overlay.DeleteAccount(key)
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (t *TransitionTrie) Hash() common.Hash {
	return t.overlay.Hash()
}

// Commit collects all dirty nodes in the trie and replace them with the
// corresponding node hash. All collected nodes(including dirty leaves if
// collectLeaf is true) will be encapsulated into a nodeset for return.
// The returned nodeset can be nil if the trie is clean(nothing to commit).
// Once the trie is committed, it's not usable anymore. A new trie must
// be created with new root and updated trie database for following usage
func (t *TransitionTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet) {
	// Just return if the trie is a storage trie: otherwise,
	// the overlay trie will be committed as many times as
	// there are storage tries. This would kill performance.
	if t.storage {
		return common.Hash{}, nil
	}
	return t.overlay.Commit(collectLeaf)
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (t *TransitionTrie) NodeIterator(startKey []byte) (NodeIterator, error) {
	panic("not implemented") // TODO: Implement
}

// Prove constructs a Merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
func (t *TransitionTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented") // TODO: Implement
}

// IsVerkle returns true if the trie is verkle-tree based
func (t *TransitionTrie) IsVerkle() bool {
	// For all intents and purposes, the calling code should treat this as a verkle trie
	return true
}

// UpdateStem updates a group of values, given the stem they are using. If
// a value already exists, it is overwritten.
func (t *TransitionTrie) UpdateStem(key []byte, values [][]byte) error {
	trie := t.overlay
	switch root := trie.root.(type) {
	case *verkle.InternalNode:
		return root.InsertValuesAtStem(key, values, t.overlay.nodeResolver)
	default:
		panic("invalid root type")
	}
}

// Copy creates a deep copy of the transition trie.
func (t *TransitionTrie) Copy() *TransitionTrie {
	return &TransitionTrie{
		overlay: t.overlay.Copy(),
		// base in immutable, so there is no need to copy it
		base:    t.base,
		storage: t.storage,
	}
}

// UpdateContractCode updates the contract code for the given address.
func (t *TransitionTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	return t.overlay.UpdateContractCode(addr, codeHash, code)
}

// Witness returns a set containing all trie nodes that have been accessed.
func (t *TransitionTrie) Witness() map[string][]byte {
	panic("not implemented")
}
