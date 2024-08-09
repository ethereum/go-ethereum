// Copyright 2024 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
)

// stateHasher wraps the necessary methods for state hashing.
type stateHasher interface {
	// hasAccount returns a flag indicating if the account with specified
	// address is existent in the state.
	hasAccount(addr common.Address) (bool, error)

	// updateAccount updates the account with the specified address in state.
	updateAccount(addr common.Address, account *types.StateAccount) error

	// deleteAccount removes the account with the specified address from the state.
	deleteAccount(addr common.Address) error

	// updateStorage inserts the storage slot with the specified account address
	// and storage key into state.
	updateStorage(addr common.Address, key []byte, val []byte) error

	// deleteStorage removes the storage slot with the specified account address
	// and storage key from state.
	deleteStorage(addr common.Address, key []byte) error

	// commitStorage commits the storage changes and compares if the new state
	// root is equal to the target.
	commitStorage(addr common.Address, expectRoot common.Hash) error

	// commit recomputes the state root and returns the dirty trie nodes caused
	// by state rehashing.
	commit() (common.Hash, *trienode.MergedNodeSet, error)
}

// merkleHasher implements stateHasher interface, hashing the state in merkle manner.
type merkleHasher struct {
	db            *Database
	root          common.Hash
	rawStorageKey bool
	buff          crypto.KeccakState
	accountTrie   *trie.Trie
	storageTries  map[common.Address]*trie.Trie
	nodes         *trienode.MergedNodeSet
}

// newMerkleHasher initializes the merkle hasher.
func newMerkleHasher(db *Database, root common.Hash, rawStorageKey bool) (*merkleHasher, error) {
	tr, err := trie.New(trie.TrieID(root), db)
	if err != nil {
		return nil, err
	}
	return &merkleHasher{
		db:            db,
		root:          root,
		rawStorageKey: rawStorageKey,
		buff:          crypto.NewKeccakState(),
		accountTrie:   tr,
		storageTries:  make(map[common.Address]*trie.Trie),
		nodes:         trienode.NewMergedNodeSet(),
	}, nil
}

// getAccount implements the stateHasher, retrieving the account with specified
// account address from state. Nil is returned if the account is not existent.
func (m *merkleHasher) getAccount(addr common.Address) (*types.StateAccount, error) {
	addrHash := crypto.HashData(m.buff, addr.Bytes()).Bytes()
	blob, err := m.accountTrie.Get(addrHash)
	if err != nil {
		return nil, err
	}
	if len(blob) == 0 {
		return nil, nil
	}
	var account types.StateAccount
	if err := rlp.DecodeBytes(blob, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// hasAccount implements stateHasher, returning a flag indicating if the account
// with specified address is existent in the state.
func (m *merkleHasher) hasAccount(addr common.Address) (bool, error) {
	addrHash := crypto.HashData(m.buff, addr.Bytes()).Bytes()
	blob, err := m.accountTrie.Get(addrHash)
	if err != nil {
		return false, err
	}
	return len(blob) > 0, nil
}

// updateStorage implements stateHasher, updating the account data with specified
// account address in the state.
func (m *merkleHasher) updateAccount(addr common.Address, account *types.StateAccount) error {
	blob, err := rlp.EncodeToBytes(account)
	if err != nil {
		return err
	}
	return m.accountTrie.Update(crypto.HashData(m.buff, addr.Bytes()).Bytes(), blob)
}

// deleteAccount implements stateHasher, removing the account with specified
// account address from the state.
func (m *merkleHasher) deleteAccount(addr common.Address) error {
	return m.accountTrie.Delete(crypto.HashData(m.buff, addr.Bytes()).Bytes())
}

// updateStorage implements stateHasher, updating the storage with specified
// account address and storage key in the state.
func (m *merkleHasher) updateStorage(addr common.Address, key []byte, val []byte) error {
	tr, ok := m.storageTries[addr]
	if !ok {
		acct, err := m.getAccount(addr)
		if err != nil {
			return err
		}
		root := types.EmptyRootHash
		if acct != nil {
			root = acct.Root
		}
		tr, err = trie.New(trie.StorageTrieID(m.root, crypto.HashData(m.buff, addr.Bytes()), root), m.db)
		if err != nil {
			return err
		}
		m.storageTries[addr] = tr
	}
	if m.rawStorageKey {
		return tr.Update(crypto.HashData(m.buff, key).Bytes(), val)
	}
	return tr.Update(key, val)
}

// deleteStorage implements stateHasher, removing the storage with specified
// account address and storage key from the state.
func (m *merkleHasher) deleteStorage(addr common.Address, key []byte) error {
	tr, ok := m.storageTries[addr]
	if !ok {
		acct, err := m.getAccount(addr)
		if err != nil {
			return err
		}
		if acct == nil {
			return fmt.Errorf("account %x is not found", addr)
		}
		tr, err = trie.New(trie.StorageTrieID(m.root, crypto.HashData(m.buff, addr.Bytes()), acct.Root), m.db)
		if err != nil {
			return err
		}
		m.storageTries[addr] = tr
	}
	if m.rawStorageKey {
		return tr.Delete(crypto.HashData(m.buff, key).Bytes())
	}
	return tr.Delete(key)
}

// commitStorage implements stateHasher, recomputing the storage trie root and
// ensuring it is equal to the target. Additionally, it aggregates the dirty
// trie nodes into a global set for the final commit.
func (m *merkleHasher) commitStorage(addr common.Address, expectRoot common.Hash) error {
	tr, ok := m.storageTries[addr]
	if !ok {
		return errors.New("the storage trie is not initialized yet")
	}
	root, nodes := tr.Commit(false)
	if nodes == nil {
		return errors.New("the storage trie change is empty")
	}
	expectRoot = types.TrieRootHash(expectRoot)
	if root != expectRoot {
		return fmt.Errorf("expected root %s, got %s", expectRoot, root)
	}
	if err := m.nodes.Merge(nodes); err != nil {
		return err
	}
	return nil
}

// commit implements stateHasher, committing the changes made and returning the
// new state root along with the dirty trie nodes caused by rehashing.
func (m *merkleHasher) commit() (common.Hash, *trienode.MergedNodeSet, error) {
	root, nodes := m.accountTrie.Commit(false)
	if nodes == nil {
		return common.Hash{}, nil, errors.New("the account trie change is empty")
	}
	if err := m.nodes.Merge(nodes); err != nil {
		return common.Hash{}, nil, err
	}
	return root, m.nodes, nil
}

// verkleHasher implements stateHasher, hashing the state in verkle manner.
type verkleHasher struct {
	root common.Hash
	db   *Database
	tr   *trie.VerkleTrie
}

// newVerkleHasher initializes the verkle hasher.
func newVerkleHasher(db *Database, root common.Hash) (*verkleHasher, error) {
	tr, err := trie.NewVerkleTrie(root, db, utils.NewPointCache(4096))
	if err != nil {
		return nil, err
	}
	return &verkleHasher{root: root, db: db, tr: tr}, nil
}

// hasAccount implements stateHasher, returning a flag indicating if the account
// with specified address is existent in the state.
func (v *verkleHasher) hasAccount(addr common.Address) (bool, error) {
	acct, err := v.tr.GetAccount(addr)
	if err != nil {
		return false, err
	}
	return acct != nil, nil
}

// updateStorage implements stateHasher, updating the account data with specified
// account address in the state.
func (v *verkleHasher) updateAccount(addr common.Address, account *types.StateAccount) error {
	return v.tr.UpdateAccount(addr, account)
}

// deleteAccount implements stateHasher, removing the account with specified
// account address from the state.
func (v *verkleHasher) deleteAccount(addr common.Address) error {
	return v.tr.RollBackAccount(addr)
}

// updateStorage implements stateHasher, updating the storage with specified
// account address and storage key in the state.
func (v *verkleHasher) updateStorage(addr common.Address, key []byte, val []byte) error {
	return v.tr.UpdateStorage(addr, key, val)
}

// deleteStorage implements stateHasher, removing the storage with specified
// account address and storage key from the state.
func (v *verkleHasher) deleteStorage(addr common.Address, key []byte) error {
	return v.tr.DeleteStorage(addr, key)
}

// commitStorage implements stateHasher, it's an noop in verkle hasher.
func (v *verkleHasher) commitStorage(addr common.Address, expectRoot common.Hash) error {
	return nil
}

// commit implements stateHasher, committing the changes made and returning the
// new state root along with the dirty trie nodes caused by rehashing.
func (v *verkleHasher) commit() (common.Hash, *trienode.MergedNodeSet, error) {
	root, nodes := v.tr.Commit(false)
	if nodes == nil {
		return common.Hash{}, nil, errors.New("the trie change is empty")
	}
	return root, trienode.NewWithNodeSet(nodes), nil
}

func newStateHasher(db *Database, root common.Hash, rawStorageKey bool) (stateHasher, error) {
	if db.isVerkle {
		if !rawStorageKey {
			return nil, errors.New("incompatible state history for verkle rollback")
		}
		return newVerkleHasher(db, root)
	}
	return newMerkleHasher(db, root, rawStorageKey)
}

// context wraps all fields for executing state diffs.
type context struct {
	prevRoot common.Hash
	postRoot common.Hash
	accounts map[common.Address][]byte
	storages map[common.Address]map[common.Hash][]byte
	hasher   stateHasher
}

// apply processes the given state diffs, updates the corresponding post-state
// and returns the trie nodes that have been modified.
func apply(db *Database, prevRoot common.Hash, postRoot common.Hash, rawStorageKey bool, accounts map[common.Address][]byte, storages map[common.Address]map[common.Hash][]byte) (map[common.Hash]map[string]*trienode.Node, error) {
	h, err := newStateHasher(db, postRoot, rawStorageKey)
	if err != nil {
		return nil, err
	}
	ctx := &context{
		prevRoot: prevRoot,
		postRoot: postRoot,
		accounts: accounts,
		storages: storages,
		hasher:   h,
	}
	for addr, account := range accounts {
		var err error
		if len(account) == 0 {
			err = deleteAccount(ctx, addr)
		} else {
			err = updateAccount(ctx, addr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to revert state, err: %w", err)
		}
	}
	root, nodes, err := h.commit()
	if root != prevRoot {
		return nil, fmt.Errorf("failed to revert state, want %#x, got %#x", prevRoot, root)
	}
	if err != nil {
		return nil, err
	}
	return nodes.Flatten(), nil
}

// updateAccount the account was present in prev-state, and may or may not
// existent in post-state. Apply the reverse diff and verify if the storage
// root matches the one in prev-state account.
func updateAccount(ctx *context, addr common.Address) error {
	// The account was present in prev-state, decode it from the
	// 'slim-rlp' format bytes.
	prev, err := types.FullAccount(ctx.accounts[addr])
	if err != nil {
		return err
	}
	// Apply all storage changes into the hasher
	for key, val := range ctx.storages[addr] {
		var err error
		if len(val) == 0 {
			err = ctx.hasher.deleteStorage(addr, key.Bytes())
		} else {
			err = ctx.hasher.updateStorage(addr, key.Bytes(), val)
		}
		if err != nil {
			return err
		}
	}
	if len(ctx.storages[addr]) > 0 {
		if err := ctx.hasher.commitStorage(addr, prev.Root); err != nil {
			return err
		}
	}
	// Write the prev-state account into the main trie
	return ctx.hasher.updateAccount(addr, prev)
}

// deleteAccount the account was not present in prev-state, and is expected
// to be existent in post-state. Apply the reverse diff and verify if the
// account and storage is wiped out correctly.
func deleteAccount(ctx *context, addr common.Address) error {
	// Ensure the account was indeed existent in the post-state.
	exists, err := ctx.hasher.hasAccount(addr)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("account is non-existent %#x", addr)
	}
	for key, val := range ctx.storages[addr] {
		if len(val) != 0 {
			return errors.New("expect storage deletion")
		}
		if err := ctx.hasher.deleteStorage(addr, key.Bytes()); err != nil {
			return err
		}
	}
	if len(ctx.storages[addr]) > 0 {
		if err := ctx.hasher.commitStorage(addr, common.Hash{}); err != nil {
			return err
		}
	}
	// Delete the post-state account from the main trie.
	return ctx.hasher.deleteAccount(addr)
}
