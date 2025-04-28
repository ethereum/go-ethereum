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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// context wraps all fields for executing state diffs.
type context struct {
	prevRoot      common.Hash
	postRoot      common.Hash
	accounts      map[common.Address][]byte
	storages      map[common.Address]map[common.Hash][]byte
	nodes         *trienode.MergedNodeSet
	rawStorageKey bool

	// TODO (rjl493456442) abstract out the state hasher
	// for supporting verkle tree.
	accountTrie *trie.Trie
}

// apply processes the given state diffs, updates the corresponding post-state
// and returns the trie nodes that have been modified.
func apply(db database.NodeDatabase, prevRoot common.Hash, postRoot common.Hash, rawStorageKey bool, accounts map[common.Address][]byte, storages map[common.Address]map[common.Hash][]byte) (map[common.Hash]map[string]*trienode.Node, error) {
	tr, err := trie.New(trie.TrieID(postRoot), db)
	if err != nil {
		return nil, err
	}
	ctx := &context{
		prevRoot:      prevRoot,
		postRoot:      postRoot,
		accounts:      accounts,
		storages:      storages,
		accountTrie:   tr,
		rawStorageKey: rawStorageKey,
		nodes:         trienode.NewMergedNodeSet(),
	}
	for addr, account := range accounts {
		var err error
		if len(account) == 0 {
			err = deleteAccount(ctx, db, addr)
		} else {
			err = updateAccount(ctx, db, addr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to revert state, err: %w", err)
		}
	}
	root, result := tr.Commit(false)
	if root != prevRoot {
		return nil, fmt.Errorf("failed to revert state, want %#x, got %#x", prevRoot, root)
	}
	if err := ctx.nodes.Merge(result); err != nil {
		return nil, err
	}
	return ctx.nodes.Flatten(), nil
}

// updateAccount the account was present in prev-state, and may or may not
// existent in post-state. Apply the reverse diff and verify if the storage
// root matches the one in prev-state account.
func updateAccount(ctx *context, db database.NodeDatabase, addr common.Address) error {
	// The account was present in prev-state, decode it from the
	// 'slim-rlp' format bytes.
	h := newHasher()
	defer h.release()

	addrHash := h.hash(addr.Bytes())
	prev, err := types.FullAccount(ctx.accounts[addr])
	if err != nil {
		return err
	}
	// The account may or may not existent in post-state, try to
	// load it and decode if it's found.
	blob, err := ctx.accountTrie.Get(addrHash.Bytes())
	if err != nil {
		return err
	}
	post := types.NewEmptyStateAccount()
	if len(blob) != 0 {
		if err := rlp.DecodeBytes(blob, &post); err != nil {
			return err
		}
	}
	// Apply all storage changes into the post-state storage trie.
	st, err := trie.New(trie.StorageTrieID(ctx.postRoot, addrHash, post.Root), db)
	if err != nil {
		return err
	}
	for key, val := range ctx.storages[addr] {
		tkey := key
		if ctx.rawStorageKey {
			tkey = h.hash(key.Bytes())
		}
		var err error
		if len(val) == 0 {
			err = st.Delete(tkey.Bytes())
		} else {
			err = st.Update(tkey.Bytes(), val)
		}
		if err != nil {
			return err
		}
	}
	root, result := st.Commit(false)
	if root != prev.Root {
		return errors.New("failed to reset storage trie")
	}
	// The returned set can be nil if storage trie is not changed
	// at all.
	if result != nil {
		if err := ctx.nodes.Merge(result); err != nil {
			return err
		}
	}
	// Write the prev-state account into the main trie
	full, err := rlp.EncodeToBytes(prev)
	if err != nil {
		return err
	}
	return ctx.accountTrie.Update(addrHash.Bytes(), full)
}

// deleteAccount the account was not present in prev-state, and is expected
// to be existent in post-state. Apply the reverse diff and verify if the
// account and storage is wiped out correctly.
func deleteAccount(ctx *context, db database.NodeDatabase, addr common.Address) error {
	// The account must be existent in post-state, load the account.
	h := newHasher()
	defer h.release()

	addrHash := h.hash(addr.Bytes())
	blob, err := ctx.accountTrie.Get(addrHash.Bytes())
	if err != nil {
		return err
	}
	if len(blob) == 0 {
		return fmt.Errorf("account is non-existent %#x", addrHash)
	}
	var post types.StateAccount
	if err := rlp.DecodeBytes(blob, &post); err != nil {
		return err
	}
	st, err := trie.New(trie.StorageTrieID(ctx.postRoot, addrHash, post.Root), db)
	if err != nil {
		return err
	}
	for key, val := range ctx.storages[addr] {
		if len(val) != 0 {
			return errors.New("expect storage deletion")
		}
		tkey := key
		if ctx.rawStorageKey {
			tkey = h.hash(key.Bytes())
		}
		if err := st.Delete(tkey.Bytes()); err != nil {
			return err
		}
	}
	root, result := st.Commit(false)
	if root != types.EmptyRootHash {
		return errors.New("failed to clear storage trie")
	}
	// The returned set can be nil if storage trie is not changed
	// at all.
	if result != nil {
		if err := ctx.nodes.Merge(result); err != nil {
			return err
		}
	}
	// Delete the post-state account from the main trie.
	return ctx.accountTrie.Delete(addrHash.Bytes())
}
