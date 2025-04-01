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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pathdb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// storageKey wraps a storage key with an additional flag, indicating whether
// it is the raw storage key or its hash.
type storageKey struct {
	raw bool
	key common.Hash
}

// hashKey returns the hash of raw storage key.
func (k storageKey) hashKey(h *hasher) common.Hash {
	if !k.raw {
		return k.key
	}
	return h.hash(k.key.Bytes())
}

// stateHasher defines the essential functions needed for state hashing.
type stateHasher interface {
	// UpdateAccount abstracts an account write to the hasher.
	UpdateAccount(address common.Address, blob []byte) error

	// UpdateStorage abstracts a storage slot write to the hasher.
	UpdateStorage(addr common.Address, key storageKey, value []byte) error

	// DeleteAccount abstracts an account deletion from the hasher.
	DeleteAccount(address common.Address) error

	// DeleteStorage abstracts a storage slot deletion from the hasher.
	DeleteStorage(addr common.Address, key storageKey) error

	// Commit applies the pending mutations, rehashes the state, and collects
	// all modified trie nodes, returning them along with the new state root.
	Commit() (common.Hash, *trienode.MergedNodeSet, error)
}

// merkleHasher implements stateHasher in Merkle-Patricia-Trie manner.
type merkleHasher struct {
	db          database.NodeDatabase
	sha256      *hasher
	root        common.Hash
	mainTr      *trie.Trie                     // The main account trie
	subTries    map[common.Address]*trie.Trie  // The set of modified storage tries
	newSubRoots map[common.Address]common.Hash // Expected storage trie roots after state transition
}

// newMerkleHasher constructs the merkle hasher with the given state root.
func newMerkleHasher(db database.NodeDatabase, root common.Hash, sha256 *hasher) (*merkleHasher, error) {
	tr, err := trie.New(trie.TrieID(root), db)
	if err != nil {
		return nil, err
	}
	return &merkleHasher{
		db:          db,
		sha256:      sha256,
		root:        root,
		mainTr:      tr,
		subTries:    make(map[common.Address]*trie.Trie),
		newSubRoots: make(map[common.Address]common.Hash),
	}, nil
}

// UpdateAccount implements stateHasher, writing the provided account into
// the trie.
func (h *merkleHasher) UpdateAccount(address common.Address, blob []byte) error {
	// The account was encoded in a slim format and is being converted
	// to full format for the trie update.
	acct, err := types.FullAccount(blob)
	if err != nil {
		return err
	}
	h.newSubRoots[address] = acct.Root

	data, err := rlp.EncodeToBytes(acct)
	if err != nil {
		return err
	}
	return h.mainTr.Update(h.sha256.hash(address.Bytes()).Bytes(), data)
}

// DeleteAccount implements stateHasher, deleting the specified account from
// the trie.
func (h *merkleHasher) DeleteAccount(address common.Address) error {
	key := h.sha256.hash(address.Bytes()).Bytes()
	data, err := h.mainTr.Get(key)
	if err != nil || len(data) == 0 {
		return fmt.Errorf("the account to be deleted does not exist, %s", address.Hex())
	}
	h.newSubRoots[address] = types.EmptyRootHash
	return h.mainTr.Delete(key)
}

// openStorageTrie opens the storage trie with the associated account address.
func (h *merkleHasher) openStorageTrie(address common.Address, emptyAllowed bool) (*trie.Trie, error) {
	var (
		account  = types.NewEmptyStateAccount()
		addrHash = h.sha256.hash(address.Bytes())
	)
	blob, err := h.mainTr.Get(addrHash.Bytes())
	if err != nil {
		return nil, err
	}
	if len(blob) == 0 && !emptyAllowed {
		return nil, fmt.Errorf("account %x is not found", address)
	}
	if len(blob) != 0 {
		if err := rlp.DecodeBytes(blob, &account); err != nil {
			return nil, err
		}
	}
	tr, err := trie.New(trie.StorageTrieID(h.root, addrHash, account.Root), h.db)
	if err != nil {
		return nil, err
	}
	h.subTries[address] = tr
	return tr, nil
}

// UpdateStorage implements stateHasher, writing the provided storage slot into
// the trie.
func (h *merkleHasher) UpdateStorage(address common.Address, key storageKey, value []byte) error {
	st, exist := h.subTries[address]
	if !exist {
		// Empty storage trie is allowed if the account was removed
		// before and tries to add it back.
		tr, err := h.openStorageTrie(address, true)
		if err != nil {
			return err
		}
		st = tr
	}
	return st.Update(key.hashKey(h.sha256).Bytes(), value)
}

// DeleteStorage implements stateHasher, deleting the specified storage slot from
// the trie.
func (h *merkleHasher) DeleteStorage(address common.Address, key storageKey) error {
	st, exist := h.subTries[address]
	if !exist {
		// Empty storage trie is disallowed for storage deletion.
		tr, err := h.openStorageTrie(address, false)
		if err != nil {
			return err
		}
		st = tr
	}
	return st.Delete(key.hashKey(h.sha256).Bytes())
}

// Commit implements stateHasher, gathering all modified trie nodes and returns
// along with the new state root.
func (h *merkleHasher) Commit() (common.Hash, *trienode.MergedNodeSet, error) {
	merged := trienode.NewMergedNodeSet()
	for address, tr := range h.subTries {
		// Each modified storage trie must have a corresponding post-transition
		// storage root cached.
		newRoot, exist := h.newSubRoots[address]
		if !exist {
			return common.Hash{}, nil, fmt.Errorf("dangling dirty storage trie: %x", address)
		}
		root, nodes := tr.Commit(false)
		if root != newRoot {
			return common.Hash{}, nil, fmt.Errorf("unexpected storage root, want: %x, got: %x", newRoot, root)
		}
		if nodes != nil {
			if err := merged.Merge(nodes); err != nil {
				return common.Hash{}, nil, err
			}
		}
	}
	root, nodes := h.mainTr.Commit(false)
	if nodes != nil {
		if err := merged.Merge(nodes); err != nil {
			return common.Hash{}, nil, err
		}
	}
	return root, merged, nil
}

// verkleHasher implements stateHasher in Verkle-Trie manner.
type verkleHasher struct {
	db     database.NodeDatabase
	disk   ethdb.KeyValueStore
	root   common.Hash
	tr     *trie.VerkleTrie
	sha256 *hasher
}

// newVerkleHasher constructs the verkle hasher with the given state root.
func newVerkleHasher(db database.NodeDatabase, disk ethdb.KeyValueStore, root common.Hash, sha256 *hasher) (*verkleHasher, error) {
	tr, err := trie.NewVerkleTrie(root, db, utils.NewPointCache(1024)) // TODO use the shared cache
	if err != nil {
		return nil, err
	}
	return &verkleHasher{
		db:     db,
		disk:   disk,
		root:   root,
		tr:     tr,
		sha256: sha256,
	}, nil
}

// UpdateAccount implements stateHasher, writing the provided account along with
// the associated code length into the verkle trie.
func (h *verkleHasher) UpdateAccount(address common.Address, blob []byte) error {
	acct, err := types.FullAccount(blob)
	if err != nil {
		return err
	}
	var code []byte
	if !bytes.Equal(acct.CodeHash, types.EmptyCodeHash.Bytes()) {
		// Contract code is assumed to be available because:
		//
		// - There is no account deletion in Verkle, the scenario that account was
		//   removed before and needs to be restored is unexpected;
		//
		// - The contract code in key-value store should be retained even if the
		//   account is deleted;
		code = rawdb.ReadCode(h.disk, common.BytesToHash(acct.CodeHash))
		if len(code) == 0 {
			return fmt.Errorf("account code is missing, address: %x, codeHash: %x", address, acct.CodeHash)
		}
	}
	// TODO @gballet @rjl493456442
	// (a) try to avoid unnecessary update if the code is not modified;
	// (b) make sure the leftover code chunks can be deleted;
	if err := h.tr.UpdateAccount(address, acct, len(code)); err != nil {
		return err
	}
	return h.tr.UpdateContractCode(address, h.sha256.hash(code), code)
}

// DeleteAccount implements stateHasher, deleting the specified account from
// the trie.
func (h *verkleHasher) DeleteAccount(address common.Address) error {
	return h.tr.RollBackAccount(address)
}

// UpdateStorage implements stateHasher, writing the provided storage slot into
// the trie.
func (h *verkleHasher) UpdateStorage(address common.Address, key storageKey, value []byte) error {
	if !key.raw {
		return errors.New("unexpected hashed storage key")
	}
	return h.tr.UpdateStorage(address, key.key.Bytes(), value)
}

// DeleteStorage implements stateHasher, deleting the specified storage slot from
// the trie.
func (h *verkleHasher) DeleteStorage(address common.Address, key storageKey) error {
	if !key.raw {
		return errors.New("unexpected hashed storage key")
	}
	// TODO(rjl493456442) rollback storage
	return h.tr.DeleteStorage(address, key.key.Bytes())
}

// Commit implements stateHasher, gathering all modified trie nodes and returns
// along with the new state root.
func (h *verkleHasher) Commit() (common.Hash, *trienode.MergedNodeSet, error) {
	merged := trienode.NewMergedNodeSet()
	root, nodes := h.tr.Commit(false)
	if nodes != nil {
		merged.Merge(nodes)
	}
	return root, merged, nil
}

// context wraps all fields for executing state diffs.
type context struct {
	accounts      map[common.Address][]byte
	storages      map[common.Address]map[common.Hash][]byte
	rawStorageKey bool
	hasher        stateHasher
}

// apply processes the given state diffs, updates the corresponding post-state
// and returns the trie nodes that have been modified.
func apply(isVerkle bool, disk ethdb.KeyValueStore, nodeDb database.NodeDatabase, prevRoot common.Hash, postRoot common.Hash, rawStorageKey bool, accounts map[common.Address][]byte, storages map[common.Address]map[common.Hash][]byte) (map[common.Hash]map[string]*trienode.Node, error) {
	var (
		err    error
		hr     stateHasher
		sha256 = newHasher()
	)
	defer sha256.release()

	if isVerkle {
		hr, err = newVerkleHasher(nodeDb, disk, postRoot, sha256)
	} else {
		hr, err = newMerkleHasher(nodeDb, postRoot, sha256)
	}
	if err != nil {
		return nil, err
	}
	ctx := &context{
		accounts:      accounts,
		storages:      storages,
		rawStorageKey: rawStorageKey,
		hasher:        hr,
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
	root, merged, err := ctx.hasher.Commit()
	if err != nil {
		return nil, err
	}
	if root != prevRoot {
		return nil, fmt.Errorf("failed to revert state, want %#x, got %#x", prevRoot, root)
	}
	return merged.Flatten(), nil
}

// updateAccount the account was present in prev-state, and may or may not
// existent in post-state. Apply the reverse diff and verify if the storage
// root matches the one in prev-state account.
func updateAccount(ctx *context, addr common.Address) error {
	for key, val := range ctx.storages[addr] {
		var err error
		if len(val) == 0 {
			err = ctx.hasher.DeleteStorage(addr, storageKey{key: key, raw: ctx.rawStorageKey})
		} else {
			err = ctx.hasher.UpdateStorage(addr, storageKey{key: key, raw: ctx.rawStorageKey}, val)
		}
		if err != nil {
			return err
		}
	}
	return ctx.hasher.UpdateAccount(addr, ctx.accounts[addr])
}

// deleteAccount the account was not present in prev-state, and is expected
// to be existent in post-state. Apply the reverse diff and verify if the
// account and storage is wiped out correctly.
func deleteAccount(ctx *context, addr common.Address) error {
	for key, val := range ctx.storages[addr] {
		if len(val) != 0 {
			return fmt.Errorf("unexpected storage update, addr: %x, key: %x, val: %v", addr, key, val)
		}
		if err := ctx.hasher.DeleteStorage(addr, storageKey{key: key, raw: ctx.rawStorageKey}); err != nil {
			return err
		}
	}
	return ctx.hasher.DeleteAccount(addr)
}
