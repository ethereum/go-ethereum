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
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
	"golang.org/x/sync/errgroup"
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

// accountRevert holds an account's prev-state value and the result of its
// storage revert.
type accountRevert struct {
	addr     common.Address
	addrHash common.Hash
	prev     *types.StateAccount // nil if the account is absent in prev-state

	root  common.Hash       // storage root after the revert, set by the worker
	nodes *trienode.NodeSet // modified storage trie nodes, set by the worker
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
	reverts, err := revertAccounts(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to revert state, err: %w", err)
	}
	// Write the accounts present in prev-state first and delete the others
	// afterwards. The modified nodes depend on the order: a deletion can
	// collapse a node that a later insertion recreates unchanged.
	var updates, deletes []*accountRevert
	for _, r := range reverts {
		if r.prev == nil {
			deletes = append(deletes, r)
		} else {
			updates = append(updates, r)
		}
	}
	for _, r := range append(updates, deletes...) {
		if err := finishRevert(ctx, r); err != nil {
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
	return ctx.nodes.Nodes(), nil
}

// revertAccounts reads the accounts after resolving their paths in the account
// trie in parallel, and reverts their storage tries concurrently.
func revertAccounts(ctx *context, db database.NodeDatabase) ([]*accountRevert, error) {
	keys := make([][]byte, 0, len(ctx.accounts))
	for addr := range ctx.accounts {
		keys = append(keys, crypto.Keccak256(addr.Bytes()))
	}
	if err := ctx.accountTrie.Prefetch(keys); err != nil {
		return nil, err
	}
	var (
		reverts = make([]*accountRevert, 0, len(ctx.accounts))
		workers errgroup.Group
	)
	workers.SetLimit(runtime.GOMAXPROCS(0))
	for addr := range ctx.accounts {
		r, postStorageRoot, err := prepareRevert(ctx, addr)
		if err != nil {
			workers.Wait()
			return nil, err
		}
		reverts = append(reverts, r)
		workers.Go(func() error {
			return revertStorage(ctx, db, r, postStorageRoot)
		})
	}
	if err := workers.Wait(); err != nil {
		return nil, err
	}
	return reverts, nil
}

// prepareRevert decodes the account's prev-state value and resolves its
// post-state storage root. An account absent in prev-state must exist in
// post-state, and its storage changes must all be deletions.
func prepareRevert(ctx *context, addr common.Address) (*accountRevert, common.Hash, error) {
	r := &accountRevert{
		addr:     addr,
		addrHash: crypto.Keccak256Hash(addr.Bytes()),
	}
	if blob := ctx.accounts[addr]; len(blob) != 0 {
		// The account was present in prev-state, decode it from the
		// 'slim-rlp' format bytes.
		prev, err := types.FullAccount(blob)
		if err != nil {
			return nil, common.Hash{}, err
		}
		r.prev = prev
	}
	blob, err := ctx.accountTrie.Get(r.addrHash.Bytes())
	if err != nil {
		return nil, common.Hash{}, err
	}
	post := types.NewEmptyStateAccount()
	if len(blob) != 0 {
		if err := rlp.DecodeBytes(blob, &post); err != nil {
			return nil, common.Hash{}, err
		}
	}
	if r.prev == nil {
		if len(blob) == 0 {
			return nil, common.Hash{}, fmt.Errorf("account is non-existent %#x", r.addrHash)
		}
		for _, val := range ctx.storages[addr] {
			if len(val) != 0 {
				return nil, common.Hash{}, errors.New("expect storage deletion")
			}
		}
	}
	return r, post.Root, nil
}

// revertStorage applies the account's storage diffs to its post-state storage
// trie and commits it.
func revertStorage(ctx *context, db database.NodeDatabase, r *accountRevert, postStorageRoot common.Hash) error {
	st, err := trie.New(trie.StorageTrieID(ctx.postRoot, r.addrHash, postStorageRoot), db)
	if err != nil {
		return err
	}
	var deletes []common.Hash
	for key, val := range ctx.storages[r.addr] {
		tkey := key
		if ctx.rawStorageKey {
			tkey = crypto.Keccak256Hash(key.Bytes())
		}
		if len(val) == 0 {
			deletes = append(deletes, tkey)
		} else {
			err := st.Update(tkey.Bytes(), val)
			if err != nil {
				return err
			}
		}
	}
	for _, tkey := range deletes {
		err := st.Delete(tkey.Bytes())
		if err != nil {
			return err
		}
	}
	r.root, r.nodes = st.Commit(false)
	return nil
}

// finishRevert verifies the reverted storage root and writes the prev-state
// account into the account trie, or deletes it if absent in prev-state.
func finishRevert(ctx *context, r *accountRevert) error {
	if r.prev == nil {
		if r.root != types.EmptyRootHash {
			return errors.New("failed to clear storage trie")
		}
	} else if r.root != r.prev.Root {
		return errors.New("failed to reset storage trie")
	}
	// The returned set can be nil if storage trie is not changed
	// at all.
	if r.nodes != nil {
		if err := ctx.nodes.Merge(r.nodes); err != nil {
			return err
		}
	}
	if r.prev == nil {
		// Delete the post-state account from the main trie.
		return ctx.accountTrie.Delete(r.addrHash.Bytes())
	}
	// Write the prev-state account into the main trie
	full, err := rlp.EncodeToBytes(r.prev)
	if err != nil {
		return err
	}
	return ctx.accountTrie.Update(r.addrHash.Bytes(), full)
}
