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
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"golang.org/x/sync/errgroup"
)

// wrapTrie pairs a StateTrie with an optional background prefetcher that
// preloads trie nodes ahead of mutation.
type wrapTrie struct {
	*trie.StateTrie
	prefetcher *prefetcher
}

// newWrapTrie creates a merkle trie with the optional prefetcher enabled.
func newWrapTrie(id *trie.ID, db *triedb.Database, prefetch bool, prefetchRead bool) (*wrapTrie, error) {
	t, err := trie.NewStateTrie(id, db)
	if err != nil {
		return nil, err
	}
	var p *prefetcher
	if prefetch {
		p = newPrefetcher(t, prefetchRead)
	}
	return &wrapTrie{StateTrie: t, prefetcher: p}, nil
}

// term synchronously terminates the prefetcher (no-op if nil or already done).
// After termination the prefetcher reference is nilled so subsequent calls are
// a cheap pointer check.
func (tr *wrapTrie) term() {
	if tr.prefetcher == nil {
		return
	}
	tr.prefetcher.terminate()
	tr.prefetcher = nil
}

// The methods below shadow the embedded trie.StateTrie so that any direct trie
// access auto-terminates the prefetcher first. This makes data-race freedom
// structural: callers never need to remember to call term() manually.

func (tr *wrapTrie) UpdateAccount(address common.Address, acc *types.StateAccount) error {
	tr.term()
	return tr.StateTrie.UpdateAccount(address, acc, 0)
}

func (tr *wrapTrie) DeleteAccount(address common.Address) error {
	tr.term()
	return tr.StateTrie.DeleteAccount(address)
}

func (tr *wrapTrie) UpdateStorage(address common.Address, key, value []byte) error {
	tr.term()
	return tr.StateTrie.UpdateStorage(address, key, value)
}

func (tr *wrapTrie) DeleteStorage(address common.Address, key []byte) error {
	tr.term()
	return tr.StateTrie.DeleteStorage(address, key)
}

func (tr *wrapTrie) Hash() common.Hash {
	tr.term()
	return tr.StateTrie.Hash()
}

func (tr *wrapTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet) {
	tr.term()
	return tr.StateTrie.Commit(collectLeaf)
}

func (tr *wrapTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	tr.term()
	return tr.StateTrie.Prove(key, proofDb)
}

func (tr *wrapTrie) Witness() map[string][]byte {
	tr.term()
	return tr.StateTrie.Witness()
}

// prefetchAccounts prewarms the trie with the specified account list.
func (tr *wrapTrie) prefetchAccounts(addresses []common.Address, read bool) {
	if tr.prefetcher == nil {
		return
	}
	tr.prefetcher.scheduleAccounts(addresses, read)
}

// prefetchStorage prewarms the trie with the specified storage list.
func (tr *wrapTrie) prefetchStorage(addr common.Address, keys []common.Hash, read bool) {
	if tr.prefetcher == nil {
		return
	}
	tr.prefetcher.scheduleSlots(addr, keys, read)
}

// copy returns a deep-copied state trie. Notably the prefetcher is deliberately
// not copied, as it only belongs to the original one.
func (tr *wrapTrie) copy() *wrapTrie {
	tr.term()
	return &wrapTrie{StateTrie: tr.StateTrie.Copy()}
}

// storageRootReader wraps the account trie for loading the storage root. It is
// essential to use an independent trie to prevent potential data races with
// the optional prefetcher.
//
// TODO(rjl493456442) use the flat state for better read efficiency.
type storageRootReader struct {
	tr *trie.StateTrie
}

func newStorageRootReader(root common.Hash, db *triedb.Database) (*storageRootReader, error) {
	t, err := trie.NewStateTrie(trie.StateTrieID(root), db)
	if err != nil {
		return nil, err
	}
	return &storageRootReader{tr: t}, nil
}

func (r *storageRootReader) read(address common.Address) (common.Hash, error) {
	acct, err := r.tr.GetAccount(address)
	if err != nil {
		return common.Hash{}, err
	}
	if acct == nil {
		return types.EmptyRootHash, nil
	}
	return acct.Root, nil
}

func (r *storageRootReader) copy() *storageRootReader {
	return &storageRootReader{tr: r.tr.Copy()}
}

// merkleHasher is a Hasher implementation backed by the traditional two-layer
// Merkle Patricia Trie (separate account trie and per-account storage tries).
type merkleHasher struct {
	db           *triedb.Database
	root         common.Hash
	reader       *storageRootReader
	prefetch     bool
	prefetchRead bool

	acctTrie     *wrapTrie
	storageTries map[common.Address]*wrapTrie

	// deletedTries preserves storage tries of accounts that were deleted
	// during the block keyed by address. Only the first deletion per
	// address is recorded (the pre-block incarnation).
	deletedTries map[common.Address]*wrapTrie

	// storageRoots tracks the storage root transition for each resolved
	// account. Prev is captured on first touch; Hash is updated by
	// UpdateStorage or set to EmptyRootHash on deletion.
	storageRoots map[common.Address]Hashes

	// Lock guards storage trie fields
	storageLock sync.Mutex
}

func newMerkleHasher(root common.Hash, db *triedb.Database, prefetch bool, prefetchRead bool) (*merkleHasher, error) {
	tr, err := newWrapTrie(trie.StateTrieID(root), db, prefetch, prefetchRead)
	if err != nil {
		return nil, err
	}
	r, err := newStorageRootReader(root, db)
	if err != nil {
		return nil, err
	}
	return &merkleHasher{
		db:           db,
		root:         root,
		prefetch:     prefetch,
		prefetchRead: prefetchRead,
		reader:       r,
		acctTrie:     tr,
		storageTries: make(map[common.Address]*wrapTrie),
		deletedTries: make(map[common.Address]*wrapTrie),
		storageRoots: make(map[common.Address]Hashes),
	}, nil
}

// storageRoot returns the current tracked storage root for addr. On first
// access for a given address the root is read from the account trie and
// recorded as the Prev value for the commit-time transition report.
func (h *merkleHasher) storageRoot(addr common.Address) (common.Hash, error) {
	if hashes, ok := h.storageRoots[addr]; ok {
		return hashes.Hash, nil
	}
	root, err := h.reader.read(addr)
	if err != nil {
		return common.Hash{}, err
	}
	h.storageRoots[addr] = Hashes{
		Prev: root,
		Hash: root,
	}
	return root, nil
}

// openStorageTrie returns the cached storage trie for addr, or opens one from
// the database if not already cached.
func (h *merkleHasher) openStorageTrie(address common.Address, prefetch bool) (*wrapTrie, error) {
	h.storageLock.Lock()
	defer h.storageLock.Unlock()

	if tr, ok := h.storageTries[address]; ok {
		return tr, nil
	}
	root, err := h.storageRoot(address)
	if err != nil {
		return nil, err
	}
	id := trie.StorageTrieID(h.root, crypto.Keccak256Hash(address.Bytes()), root)

	tr, err := newWrapTrie(id, h.db, h.prefetch && prefetch, h.prefetchRead)
	if err != nil {
		return nil, err
	}
	h.storageTries[address] = tr
	return tr, nil
}

// deleteAccount removes the account specified by the address from the state.
func (h *merkleHasher) deleteAccount(addr common.Address) error {
	// Capture the original storage root before modifying the trie.
	_, err := h.storageRoot(addr)
	if err != nil {
		return err
	}
	h.storageRoots[addr] = Hashes{
		Prev: h.storageRoots[addr].Prev,
		Hash: types.EmptyRootHash,
	}
	// Preserve the first deleted storage trie per address for
	// witness collection.
	if tr, ok := h.storageTries[addr]; ok && h.deletedTries[addr] == nil {
		h.deletedTries[addr] = tr
	}
	delete(h.storageTries, addr)

	return h.acctTrie.DeleteAccount(addr)
}

// update writes the account specified by the address into the state.
func (h *merkleHasher) updateAccount(addr common.Address, account AccountMut) error {
	root, err := h.storageRoot(addr)
	if err != nil {
		return err
	}
	data := &types.StateAccount{
		Nonce:    account.Account.Nonce,
		Balance:  account.Account.Balance,
		Root:     root,
		CodeHash: account.Account.CodeHash,
	}
	return h.acctTrie.UpdateAccount(addr, data)
}

// UpdateAccount implements Hasher, writing a list of account mutations
// into the state. The assumption is held all the storage changes have
// already been written beforehand.
func (h *merkleHasher) UpdateAccount(addresses []common.Address, accounts []AccountMut) error {
	var err error
	for i, addr := range addresses {
		if accounts[i].Account == nil {
			err = h.deleteAccount(addr)
		} else {
			err = h.updateAccount(addr, accounts[i])
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateStorage implements Hasher, writing a list of storage slot mutations
// into the state. This function must be invoked first before writing the
// associated account metadata into the state.
func (h *merkleHasher) UpdateStorage(address common.Address, keys []common.Hash, values []common.Hash) error {
	tr, err := h.openStorageTrie(address, false)
	if err != nil {
		return err
	}
	for i, key := range keys {
		if values[i] == (common.Hash{}) {
			err = tr.DeleteStorage(address, key[:])
		} else {
			err = tr.UpdateStorage(address, key[:], common.TrimLeftZeroes(values[i][:]))
		}
		if err != nil {
			return err
		}
	}
	// Hash outside the lock to allow full parallelism across accounts.
	hash := tr.Hash()

	// Write back the storage root back for reflecting the most recent
	// changes.
	h.storageLock.Lock()
	h.storageRoots[address] = Hashes{
		Prev: h.storageRoots[address].Prev,
		Hash: hash,
	}
	h.storageLock.Unlock()
	return nil
}

// Hash implements Hasher, computing the state root hash without committing.
func (h *merkleHasher) Hash() common.Hash {
	return h.acctTrie.Hash()
}

// Commit implements Hasher, finalizing all pending changes and returning
// the resulting state root hash, along with the set of dirty trie nodes
// generated by the updates.
func (h *merkleHasher) Commit() (common.Hash, *trienode.MergedNodeSet, map[common.Address]Hashes, error) {
	// Explicitly terminate all resolved tries. Some of them may not be
	// terminated due to read-only prefetching. This is essential to
	// prevent goroutine leaks.
	h.Close()

	var (
		eg   errgroup.Group
		root common.Hash

		lock  sync.Mutex
		nodes = trienode.NewMergedNodeSet()
		merge = func(set *trienode.NodeSet) error {
			lock.Lock()
			defer lock.Unlock()

			return nodes.Merge(set)
		}
	)
	eg.Go(func() error {
		r, set := h.acctTrie.Commit(true)
		root = r
		if set == nil {
			return nil
		}
		return merge(set)
	})
	for _, tr := range h.storageTries {
		eg.Go(func() error {
			_, set := tr.Commit(false)
			if set == nil {
				return nil
			}
			return merge(set)
		})
	}
	if err := eg.Wait(); err != nil {
		return common.Hash{}, nil, nil, err
	}
	return root, nodes, h.storageRoots, nil
}

// Copy implements Hasher, returning a deep-copied hasher instance.
func (h *merkleHasher) Copy() Hasher {
	cpy := &merkleHasher{
		db:           h.db,
		root:         h.root,
		reader:       h.reader.copy(),
		prefetch:     false,
		prefetchRead: false,
		acctTrie:     h.acctTrie.copy(),
		storageTries: make(map[common.Address]*wrapTrie, len(h.storageTries)),
		deletedTries: make(map[common.Address]*wrapTrie, len(h.deletedTries)),
		storageRoots: maps.Clone(h.storageRoots),
	}
	for addr, tr := range h.storageTries {
		cpy.storageTries[addr] = tr.copy()
	}
	for addr, tr := range h.deletedTries {
		cpy.deletedTries[addr] = tr.copy()
	}
	return cpy
}

// Close terminates all prefetcher goroutines. Safe to call multiple times.
func (h *merkleHasher) Close() {
	h.acctTrie.term()
	for _, tr := range h.storageTries {
		tr.term()
	}
	for _, tr := range h.deletedTries {
		tr.term()
	}
}

// ProveAccount implements Prover, constructing a proof for the given account.
func (h *merkleHasher) ProveAccount(addr common.Address, proofDb ethdb.KeyValueWriter) error {
	return h.acctTrie.Prove(crypto.Keccak256(addr.Bytes()), proofDb)
}

// ProveStorage implements Prover, constructing a proof for the given storage
// slot of the specified account.
func (h *merkleHasher) ProveStorage(addr common.Address, key common.Hash, proofDb ethdb.KeyValueWriter) error {
	tr, err := h.openStorageTrie(addr, false)
	if err != nil {
		return err
	}
	return tr.Prove(crypto.Keccak256(key.Bytes()), proofDb)
}

// CollectWitness implements WitnessCollector. It aggregates all trie nodes
// accessed (both read and write) across the account trie, all active storage
// tries and deleted storage tries into a single state witness.
func (h *merkleHasher) CollectWitness(witness *stateless.Witness) {
	witness.AddState(h.acctTrie.Witness(), common.Hash{})
	for addr, tr := range h.storageTries {
		witness.AddState(tr.Witness(), crypto.Keccak256Hash(addr.Bytes()))
	}
	for addr, tr := range h.deletedTries {
		witness.AddState(tr.Witness(), crypto.Keccak256Hash(addr.Bytes()))
	}
}

// PrefetchAccount implements Prefetcher, preloading the nodes of specific accounts.
func (h *merkleHasher) PrefetchAccount(addresses []common.Address, read bool) {
	if !h.prefetch {
		return
	}
	h.acctTrie.prefetchAccounts(addresses, read)
}

// PrefetchStorage implements Prefetcher. The storage trie is opened eagerly
// so the prefetcher can begin loading nodes in the background.
func (h *merkleHasher) PrefetchStorage(addr common.Address, keys []common.Hash, read bool) {
	if !h.prefetch {
		return
	}
	tr, err := h.openStorageTrie(addr, true)
	if err != nil {
		return
	}
	tr.prefetchStorage(addr, keys, read)
}
