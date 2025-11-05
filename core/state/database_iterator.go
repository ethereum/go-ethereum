// Copyright 2025 The go-ethereum Authors
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

package state

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

// Iterator is an iterator to step over all the accounts or the specific
// storage in the specific state.
type Iterator interface {
	// Next steps the iterator forward one element, returning false if exhausted,
	// or an error if iteration failed for some reason.
	Next() bool

	// Error returns any failure that occurred during iteration, which might have
	// caused a premature iteration exit.
	Error() error

	// Hash returns the hash of the account or storage slot the iterator is
	// currently at.
	Hash() common.Hash

	// Release releases associated resources. Release should always succeed and
	// can be called multiple times without causing error.
	Release()
}

// AccountIterator is an iterator to step over all the accounts in the
// specific state.
type AccountIterator interface {
	Iterator

	// Account returns the RLP encoded account the iterator is currently at.
	// An error will be retained if the iterator becomes invalid.
	Account() []byte
}

// StorageIterator is an iterator to step over the specific storage in the
// specific state.
type StorageIterator interface {
	Iterator

	// Slot returns the storage slot the iterator is currently at. An error will
	// be retained if the iterator becomes invalid.
	Slot() []byte
}

// Iteratee wraps the NewIterator methods for traversing the accounts and
// storages of the specific state.
type Iteratee interface {
	// NewAccountIterator creates an account iterator for the state specified by
	// the given root. It begins at a specified starting position, corresponding
	// to a particular initial key (or the next key if the specified one does
	// not exist).
	//
	// The starting position here refers to the hash of the account address.
	NewAccountIterator(start common.Hash) (AccountIterator, error)

	// NewStorageIterator creates a storage iterator for the state specified by
	// the given root and the address hash. It begins at a specified starting
	// position, corresponding to a particular initial key (or the next key if
	// the specified one does not exist).
	//
	// The starting position here refers to the hash of the slot key.
	NewStorageIterator(addressHash common.Hash, storageRoot common.Hash, start common.Hash) (StorageIterator, error)
}

// flatAccountIterator is a wrapper around the underlying flat state iterator.
// Before returning data from the iterator, it performs an additional conversion
// to bridge the slim encoding with the full encoding format.
type flatAccountIterator struct {
	err error
	it  snapshot.AccountIterator
}

// newFlatAccountIterator constructs the account iterator with the provided
// flat state iterator.
func newFlatAccountIterator(it snapshot.AccountIterator) *flatAccountIterator {
	return &flatAccountIterator{it: it}
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason.
func (ai *flatAccountIterator) Next() bool {
	if ai.err != nil {
		return false
	}
	return ai.it.Next()
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit.
func (ai *flatAccountIterator) Error() error {
	if ai.err != nil {
		return ai.err
	}
	return ai.it.Error()
}

// Hash returns the hash of the account or storage slot the iterator is
// currently at.
func (ai *flatAccountIterator) Hash() common.Hash {
	return ai.it.Hash()
}

// Release releases associated resources. Release should always succeed and
// can be called multiple times without causing error.
func (ai *flatAccountIterator) Release() {
	ai.it.Release()
}

// Account returns the account data the iterator is currently at. The account
// data is encoded as slim format from the underlying iterator, the conversion
// is required.
func (ai *flatAccountIterator) Account() []byte {
	data, err := types.FullAccountRLP(ai.it.Account())
	if err != nil {
		ai.err = err
		return nil
	}
	return data
}

// merkleIterator implements the Iterator interface, providing functions to traverse
// the accounts or storages with the manner of Merkle-Patricia-Trie.
type merkleIterator struct {
	err     error
	it      *trie.Iterator
	account bool
}

// newMerkleTrieIterator constructs the iterator with the given trie and starting position.
func newMerkleTrieIterator(tr Trie, start common.Hash, account bool) (*merkleIterator, error) {
	it, err := tr.NodeIterator(start.Bytes())
	if err != nil {
		return nil, err
	}
	return &merkleIterator{
		it:      trie.NewIterator(it),
		account: account,
	}, nil
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason.
func (ti *merkleIterator) Next() bool {
	if ti.err != nil {
		return false
	}
	return ti.it.Next()
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit.
func (ti *merkleIterator) Error() error {
	if ti.err != nil {
		return ti.err
	}
	return ti.it.Err
}

// Hash returns the hash of the account or storage slot the iterator is
// currently at.
func (ti *merkleIterator) Hash() common.Hash {
	return common.BytesToHash(ti.it.Key)
}

// Release releases associated resources. Release should always succeed and
// can be called multiple times without causing error.
func (ti *merkleIterator) Release() {}

// Account returns the account data the iterator is currently at.
func (ti *merkleIterator) Account() []byte {
	if !ti.account {
		ti.err = errors.New("account data is not available")
		return nil
	}
	return ti.it.Value
}

// Slot returns the storage slot the iterator is currently at.
func (ti *merkleIterator) Slot() []byte {
	if ti.account {
		ti.err = errors.New("storage data is not available")
		return nil
	}
	return ti.it.Value
}

// stateIteratee implements Iteratee interface, providing the state traversal
// functionalities of a specific state.
type stateIteratee struct {
	merkle bool
	root   common.Hash
	triedb *triedb.Database
	snap   *snapshot.Tree
}

func newStateIteratee(merkle bool, root common.Hash, triedb *triedb.Database, snap *snapshot.Tree) (*stateIteratee, error) {
	return &stateIteratee{
		merkle: merkle,
		root:   root,
		triedb: triedb,
		snap:   snap,
	}, nil
}

// NewAccountIterator creates an account iterator for the state specified by
// the given root. It begins at a specified starting position, corresponding
// to a particular initial key (or the next key if the specified one does
// not exist).
//
// The starting position here refers to the hash of the account address.
func (si *stateIteratee) NewAccountIterator(start common.Hash) (AccountIterator, error) {
	// If the external snapshot is available (hash scheme), try to initialize
	// the account iterator from there first.
	if si.snap != nil {
		it, err := si.snap.AccountIterator(si.root, start)
		if err == nil {
			return newFlatAccountIterator(it), nil
		}
	}
	// If the external snapshot is not available, try to initialize the
	// account iterator from the trie database (path scheme)
	it, err := si.triedb.AccountIterator(si.root, start)
	if err == nil {
		return newFlatAccountIterator(it), nil
	}
	if !si.merkle {
		return nil, fmt.Errorf("state %x is not available for account traversal", si.root)
	}
	// The snapshot is not usable so far, construct the account iterator from
	// the trie as the fallback. It's not as efficient as the flat state iterator.
	tr, err := trie.NewStateTrie(trie.StateTrieID(si.root), si.triedb)
	if err != nil {
		return nil, err
	}
	return newMerkleTrieIterator(tr, start, true)
}

// NewStorageIterator creates a storage iterator for the state specified by
// the given root and the address hash. It begins at a specified starting
// position, corresponding to a particular initial key (or the next key if
// the specified one does not exist).
//
// The starting position here refers to the hash of the slot key.
func (si *stateIteratee) NewStorageIterator(addressHash common.Hash, storageRoot common.Hash, start common.Hash) (StorageIterator, error) {
	// If the external snapshot is available (hash scheme), try to initialize
	// the storage iterator from there first.
	if si.snap != nil {
		it, err := si.snap.StorageIterator(si.root, addressHash, start)
		if err == nil {
			return it, nil
		}
	}
	// If the external snapshot is not available, try to initialize the
	// storage iterator from the trie database (path scheme)
	it, err := si.triedb.StorageIterator(si.root, addressHash, start)
	if err == nil {
		return it, nil
	}
	if !si.merkle {
		return nil, fmt.Errorf("state %x is not available for account traversal", si.root)
	}
	// The snapshot is not usable so far, construct the storage iterator from
	// the trie as the fallback. It's not as efficient as the flat state iterator.
	//
	// TODO(rjl493456442) the storageRoot can be resolved from the reader
	// internally, we can probably get rid of it from the parameters.
	tr, err := trie.NewStateTrie(trie.StorageTrieID(si.root, addressHash, storageRoot), si.triedb)
	if err != nil {
		return nil, err
	}
	return newMerkleTrieIterator(tr, start, false)
}
