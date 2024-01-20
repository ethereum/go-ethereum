// Copyright 2021 go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-verkle"
)

type TransitionTrie struct {
	overlay *VerkleTrie
	base    *SecureTrie
	storage bool
}

func NewTransitionTree(base *SecureTrie, overlay *VerkleTrie, st bool) *TransitionTrie {
	return &TransitionTrie{
		overlay: overlay,
		base:    base,
		storage: st,
	}
}

func (t *TransitionTrie) Base() *SecureTrie {
	return t.base
}

// TODO(gballet/jsign): consider removing this API.
func (t *TransitionTrie) Overlay() *VerkleTrie {
	return t.overlay
}

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
//
// TODO(fjl): remove this when StateTrie is removed
func (t *TransitionTrie) GetKey(key []byte) []byte {
	if key := t.overlay.GetKey(key); key != nil {
		return key
	}
	return t.base.GetKey(key)
}

// Get returns the value for key stored in the trie. The value bytes must
// not be modified by the caller. If a node was not found in the database, a
// trie.MissingNodeError is returned.
func (t *TransitionTrie) GetStorage(addr common.Address, key []byte) ([]byte, error) {
	if val, err := t.overlay.GetStorage(addr, key); len(val) != 0 || err != nil {
		return val, nil
	}
	// TODO also insert value into overlay
	return t.base.GetStorage(addr, key)
}

// GetAccount abstract an account read from the trie.
func (t *TransitionTrie) GetAccount(address common.Address) (*types.StateAccount, error) {
	data, err := t.overlay.GetAccount(address)
	if err != nil {
		// WORKAROUND, see the definition of errDeletedAccount
		// for an explainer of why this if is needed.
		if err == errDeletedAccount {
			return nil, nil
		}
		return nil, err
	}
	if data != nil {
		if t.overlay.db.HasStorageRootConversion(address) {
			data.Root = t.overlay.db.StorageRootConversion(address)
		}
		return data, nil
	}
	// TODO also insert value into overlay
	return t.base.GetAccount(address)
}

// Update associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
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
func (t *TransitionTrie) UpdateAccount(addr common.Address, account *types.StateAccount) error {
	if account.Root != (common.Hash{}) && account.Root != types.EmptyRootHash {
		t.overlay.db.SetStorageRootConversion(addr, account.Root)
	}
	return t.overlay.UpdateAccount(addr, account)
}

// Delete removes any existing value for key from the trie. If a node was not
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
func (t *TransitionTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet, error) {
	// Just return if the trie is a storage trie: otherwise,
	// the overlay trie will be committed as many times as
	// there are storage tries. This would kill performance.
	if t.storage {
		return common.Hash{}, nil, nil
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

func (t *TransitionTrie) UpdateStem(key []byte, values [][]byte) error {
	trie := t.overlay
	switch root := trie.root.(type) {
	case *verkle.InternalNode:
		return root.InsertValuesAtStem(key, values, t.overlay.FlatdbNodeResolver)
	default:
		panic("invalid root type")
	}
}

func (t *TransitionTrie) Copy() *TransitionTrie {
	return &TransitionTrie{
		overlay: t.overlay.Copy(),
		base:    t.base.Copy(),
		storage: t.storage,
	}
}

func (t *TransitionTrie) UpdateContractCode(addr common.Address, codeHash common.Hash, code []byte) error {
	return t.overlay.UpdateContractCode(addr, codeHash, code)
}
