// Copyright 2026 The go-ethereum Authors
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
	"maps"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// merkleHasher is a Hasher implementation backed by the traditional two-layer
// Merkle Patricia Trie (separate account trie and per-account storage tries).
type merkleHasher struct {
	db   *triedb.Database
	root common.Hash

	accountTrie  *trie.StateTrie
	storageTries map[common.Address]*trie.StateTrie // lazily opened

	// storageRoots tracks the storage root transition for every mutated
	// account. Prev is recorded once (first touch) and Hash is updated
	// on each UpdateAccount call.
	storageRoots map[common.Address]Hashes

	lock sync.Mutex // guards storageTries (concurrent updateTrie)
}

func newMerkleHasher(root common.Hash, db *triedb.Database) (*merkleHasher, error) {
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), db)
	if err != nil {
		return nil, err
	}
	return &merkleHasher{
		db:           db,
		root:         root,
		accountTrie:  tr,
		storageTries: make(map[common.Address]*trie.StateTrie),
		storageRoots: make(map[common.Address]Hashes),
	}, nil
}

// accountStorageRoot reads the storage root of account from the account trie.
func (h *merkleHasher) accountStorageRoot(addr common.Address) common.Hash {
	if acc, _ := h.accountTrie.GetAccount(addr); acc != nil {
		return acc.Root
	}
	return types.EmptyRootHash
}

// recordOrigin records the original (pre-mutation) storage root for addr.
// Only the first call per address has any effect.
func (h *merkleHasher) recordOrigin(addr common.Address) {
	if _, ok := h.storageRoots[addr]; !ok {
		root := h.accountStorageRoot(addr)
		h.storageRoots[addr] = Hashes{
			Prev: root,
			Hash: root,
		}
	}
}

// openStorageTrie returns the cached storage trie for the given address,
// or opens one from the database if not already cached.
func (h *merkleHasher) openStorageTrie(address common.Address) (*trie.StateTrie, error) {
	if st, ok := h.storageTries[address]; ok {
		return st, nil
	}
	// Record the original storage trie root if it has not already been tracked
	// when the storage trie is loaded.
	h.recordOrigin(address)

	id := trie.StorageTrieID(h.root, crypto.Keccak256Hash(address.Bytes()), h.accountStorageRoot(address))
	st, err := trie.NewStateTrie(id, h.db)
	if err != nil {
		return nil, err
	}
	h.storageTries[address] = st
	return st, nil
}

func (h *merkleHasher) UpdateStorage(address common.Address, keys []common.Hash, values []common.Hash) error {
	h.lock.Lock()
	st, err := h.openStorageTrie(address)
	if err != nil {
		h.lock.Unlock()
		return err
	}
	h.lock.Unlock()

	for i, key := range keys {
		if values[i] == (common.Hash{}) {
			if err := st.DeleteStorage(address, key[:]); err != nil {
				return err
			}
		} else {
			if err := st.UpdateStorage(address, key[:], common.TrimLeftZeroes(values[i][:])); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *merkleHasher) UpdateAccount(addresses []common.Address, accounts []AccountMut) error {
	for i, addr := range addresses {
		h.recordOrigin(addr)
		acct := accounts[i]

		// Deletion: remove from account trie and evict any cached
		// storage trie so a re-created account starts fresh.
		if acct.Account == nil {
			if err := h.accountTrie.DeleteAccount(addr); err != nil {
				return err
			}
			delete(h.storageTries, addr)

			h.storageRoots[addr] = Hashes{
				Prev: h.storageRoots[addr].Prev,
				Hash: types.EmptyRootHash,
			}
			continue
		}
		// Determine storage root from the cached trie (if storage was
		// modified) or from the account trie (unchanged storage).
		storageRoot := h.accountStorageRoot(addr)
		if st, ok := h.storageTries[addr]; ok {
			storageRoot = st.Hash()
		}
		sa := &types.StateAccount{
			Nonce:    acct.Account.Nonce,
			Balance:  acct.Account.Balance,
			Root:     storageRoot,
			CodeHash: acct.Account.CodeHash,
		}
		if err := h.accountTrie.UpdateAccount(addr, sa, 0); err != nil {
			return err
		}
		h.storageRoots[addr] = Hashes{
			Prev: h.storageRoots[addr].Prev,
			Hash: storageRoot,
		}
	}
	return nil
}

func (h *merkleHasher) Hash() common.Hash {
	return h.accountTrie.Hash()
}

func (h *merkleHasher) Commit() (common.Hash, *trienode.MergedNodeSet, map[common.Address]Hashes, error) {
	nodes := trienode.NewMergedNodeSet()

	// Commit all dirty storage tries.
	for _, st := range h.storageTries {
		if _, set := st.Commit(false); set != nil {
			if err := nodes.Merge(set); err != nil {
				return common.Hash{}, nil, nil, err
			}
		}
	}
	// Commit the account trie. collectLeaf must be true so that hashdb
	// can link account trie leaves to their storage trie roots.
	root, set := h.accountTrie.Commit(true)
	if set != nil {
		if err := nodes.Merge(set); err != nil {
			return common.Hash{}, nil, nil, err
		}
	}
	return root, nodes, h.storageRoots, nil
}

func (h *merkleHasher) Copy() Hasher {
	cpy := &merkleHasher{
		db:           h.db,
		root:         h.root,
		accountTrie:  h.accountTrie.Copy(),
		storageTries: make(map[common.Address]*trie.StateTrie, len(h.storageTries)),
		storageRoots: maps.Clone(h.storageRoots),
	}
	for addr, st := range h.storageTries {
		cpy.storageTries[addr] = st.Copy()
	}
	return cpy
}

// ProveAccount implements Prover by constructing a Merkle proof for the
// given account against the current account trie.
func (h *merkleHasher) ProveAccount(addr common.Address, proofDb ethdb.KeyValueWriter) error {
	return h.accountTrie.Prove(crypto.Keccak256(addr.Bytes()), proofDb)
}

// ProveStorage implements Prover by constructing a Merkle proof for the given
// storage slot. The storage trie is opened lazily if not already cached.
func (h *merkleHasher) ProveStorage(addr common.Address, key common.Hash, proofDb ethdb.KeyValueWriter) error {
	st, err := h.openStorageTrie(addr)
	if err != nil {
		return err
	}
	return st.Prove(crypto.Keccak256(key.Bytes()), proofDb)
}
