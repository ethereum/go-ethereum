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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// warpBinTrie pairs a BinaryTrie with an optional background prefetcher that
// preloads trie nodes ahead of mutation.
type warpBinTrie struct {
	*bintrie.BinaryTrie
	prefetcher *prefetcher
}

// newWrapBinTrie creates a binary trie with the optional prefetcher enabled.
func newWrapBinTrie(root common.Hash, db *triedb.Database, prefetch bool, prefetchRead bool) (*warpBinTrie, error) {
	t, err := bintrie.NewBinaryTrie(root, db)
	if err != nil {
		return nil, err
	}
	var p *prefetcher
	if prefetch {
		p = newPrefetcher(t, prefetchRead)
	}
	return &warpBinTrie{BinaryTrie: t, prefetcher: p}, nil
}

// term synchronously terminates the prefetcher (no-op if nil or already done).
// After termination the prefetcher reference is nilled so subsequent calls are
// a cheap pointer check.
func (tr *warpBinTrie) term() {
	if tr.prefetcher == nil {
		return
	}
	tr.prefetcher.terminate()
	tr.prefetcher = nil
}

// The methods below shadow the embedded bintrie.BinaryTrie so that any direct trie
// access auto-terminates the prefetcher first. This makes data-race freedom
// structural: callers never need to remember to call term() manually.

func (tr *warpBinTrie) UpdateAccount(address common.Address, acc *types.StateAccount, codeLen int) error {
	tr.term()
	return tr.BinaryTrie.UpdateAccount(address, acc, codeLen)
}

func (tr *warpBinTrie) DeleteAccount(address common.Address) error {
	tr.term()
	return tr.BinaryTrie.DeleteAccount(address)
}

func (tr *warpBinTrie) UpdateStorage(address common.Address, key, value []byte) error {
	tr.term()
	return tr.BinaryTrie.UpdateStorage(address, key, value)
}

func (tr *warpBinTrie) DeleteStorage(address common.Address, key []byte) error {
	tr.term()
	return tr.BinaryTrie.DeleteStorage(address, key)
}

func (tr *warpBinTrie) Hash() common.Hash {
	tr.term()
	return tr.BinaryTrie.Hash()
}

func (tr *warpBinTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet) {
	tr.term()
	return tr.BinaryTrie.Commit(collectLeaf)
}

func (tr *warpBinTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	tr.term()
	return tr.BinaryTrie.Prove(key, proofDb)
}

func (tr *warpBinTrie) Witness() map[string][]byte {
	tr.term()
	return tr.BinaryTrie.Witness()
}

func (tr *warpBinTrie) prefetchAccounts(addresses []common.Address, read bool) {
	if tr.prefetcher == nil {
		return
	}
	tr.prefetcher.scheduleAccounts(addresses, read)
}

func (tr *warpBinTrie) prefetchStorage(addr common.Address, keys []common.Hash, read bool) {
	if tr.prefetcher == nil {
		return
	}
	tr.prefetcher.scheduleSlots(addr, keys, read)
}

// copy returns a deep-copied state trie. Notably the prefetcher is deliberately
// not copied, as it only belongs to the original one.
func (tr *warpBinTrie) copy() *warpBinTrie {
	tr.term()
	return &warpBinTrie{BinaryTrie: tr.BinaryTrie.Copy()}
}

// binaryHasher is a Hasher implementation backed by a unified single-layer
// binary trie. Accounts, storage slots, and contract code all reside in one
// trie, keyed according to the EIP-7864 address space layout.
//
// binaryHasher also implements LeafProducer: alongside every trie mutation
// it records the corresponding (stem, offset, value) write into an
// internal buffer. StateDB.commit() drains this buffer once per block
// via LeafProducer.DrainStemWrites and hands the writes to the pathdb
// flat-state layer via stateUpdate.encodeBinary, keeping the bintrie
// trie and its flat-state mirror consistent without recomputing the
// bintrie key derivation twice.
type binaryHasher struct {
	db   *triedb.Database
	root common.Hash

	prefetch bool
	trie     *warpBinTrie

	// leaves buffers flat-state writes produced as a side-effect of
	// UpdateAccount/UpdateStorage/deleteAccount. It is cleared by
	// DrainStemWrites. Direct reads and writes to this slice are only
	// safe from the single goroutine that owns the hasher; the Hasher
	// interface already requires single-threaded use per block.
	leaves []StemWrite
}

// Compile-time assertion that binaryHasher implements LeafProducer.
var _ LeafProducer = (*binaryHasher)(nil)

func newBinaryHasher(root common.Hash, db *triedb.Database, prefetch bool, prefetchRead bool) (*binaryHasher, error) {
	tr, err := newWrapBinTrie(root, db, prefetch, prefetchRead)
	if err != nil {
		return nil, err
	}
	return &binaryHasher{
		db:       db,
		root:     root,
		prefetch: prefetch,
		trie:     tr,
	}, nil
}

// DrainStemWrites implements LeafProducer. It returns the buffered stem
// writes accumulated since the last drain and resets the buffer. The
// returned slice is owned by the caller; the hasher allocates a fresh
// backing array on the next update.
func (h *binaryHasher) DrainStemWrites() []StemWrite {
	out := h.leaves
	h.leaves = nil
	return out
}

// recordLeaf appends a single stem write to the internal buffer. The
// stem is taken from the first 31 bytes of the supplied 32-byte tree
// key, and the offset is the last byte. Value may be nil (for clearing
// a slot in the flat state, matching account deletion) or a 32-byte
// slice (for writes).
func (h *binaryHasher) recordLeaf(fullKey []byte, value []byte) {
	var w StemWrite
	copy(w.Stem[:], fullKey[:bintrie.StemSize])
	w.Offset = fullKey[bintrie.StemSize]
	if value != nil {
		w.Value = make([]byte, len(value))
		copy(w.Value, value)
	}
	h.leaves = append(h.leaves, w)
}

// deleteAccount removes the account specified by the address from the state.
//
// In addition to the trie mutation, this records two "clear" stem writes
// (one for BasicData at offset 0 and one for CodeHash at offset 1) so
// the flat-state mirror can drop the matching entries.
//
// Note: BinaryTrie.DeleteAccount is currently a no-op upstream
// (tracked as a standalone bugfix PR against ethereum/go-ethereum).
// Until that fix lands the on-trie deletion does nothing, but the
// flat-state mirror will still drop its copy — a minor temporary
// inconsistency scoped to the account-delete path. Once the trie fix
// lands the two sides converge.
//
// Storage slots and code chunks at the same or other stems are NOT
// touched by this function; callers that need a full account wipe must
// walk storage explicitly. Pre-EIP-6780 self-destruct wipe is a
// documented scope limitation.
func (h *binaryHasher) deleteAccount(addr common.Address) error {
	// Record the flat-state mutations BEFORE the trie call so the
	// buffer still reflects the intended write even if the trie layer
	// errors and we need to roll things back.
	basicDataKey := bintrie.GetBinaryTreeKeyBasicData(addr)
	codeHashKey := bintrie.GetBinaryTreeKeyCodeHash(addr)
	h.recordLeaf(basicDataKey, nil) // nil → clear the flat-state offset
	h.recordLeaf(codeHashKey, nil)

	return h.trie.DeleteAccount(addr)
}

// update writes the account specified by the address into the state.
//
// The account's code size is taken from AccountMut.CodeSize, which the
// caller (StateDB.IntermediateRoot) populates via stateObject.CodeSize().
// Per EIP-7864 the code_size field is packed into the BasicData leaf
// (bytes 5-7) and is consensus-critical; BinaryTrie.UpdateAccount rewrites
// the entire BasicData blob on every call, so passing the wrong codeLen
// would silently overwrite the stored code_size. In particular, for
// balance/nonce-only updates the new code bytes (account.Code) are nil
// and len(obj.code) is 0, yet the account may still have a non-zero code
// size that must be preserved — the caller gets this right by consulting
// the stateObject, which falls back to a reader code-size lookup when
// the bytes are not loaded.
func (h *binaryHasher) updateAccount(addr common.Address, account AccountMut) error {
	data := &types.StateAccount{
		Nonce:    account.Account.Nonce,
		Balance:  account.Account.Balance,
		CodeHash: account.Account.CodeHash,
	}
	if err := h.trie.UpdateAccount(addr, data, account.CodeSize); err != nil {
		return err
	}
	// Record the two flat-state writes that correspond to the on-trie
	// BasicData (offset 0) and CodeHash (offset 1) at the account's
	// stem. PackBasicData produces the same 32-byte blob that the trie
	// layer packs internally, so the flat-state mirror encodes
	// bit-identically.
	basicData := bintrie.PackBasicData(data.Nonce, data.Balance, account.CodeSize)
	h.recordLeaf(bintrie.GetBinaryTreeKeyBasicData(addr), basicData[:])

	// CodeHash is a 32-byte value written straight into offset 1.
	// EOAs store types.EmptyCodeHash here (a known non-zero hash) so
	// the flat-state offset is always set after any non-delete update.
	h.recordLeaf(bintrie.GetBinaryTreeKeyCodeHash(addr), data.CodeHash)

	// Write chunked code into the trie when dirty.
	if account.Code != nil && len(account.Code.Code) > 0 {
		codeHash := common.BytesToHash(account.Account.CodeHash)
		if err := h.trie.UpdateContractCode(addr, codeHash, account.Code.Code); err != nil {
			return err
		}
	}
	return nil
}

// UpdateAccount implements Hasher, writing a list of account mutations
// into the state. The assumption is held all the storage changes have
// already been written beforehand.
func (h *binaryHasher) UpdateAccount(addresses []common.Address, accounts []AccountMut) error {
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
//
// Each mutation is also recorded as a flat-state stem write. A zero value
// is the bintrie's "delete" convention: the trie writes 32 zero bytes at
// the slot, and the flat-state mirror does the same (a present-with-zero
// tombstone) rather than removing the offset from its bitmap. This keeps
// the trie and flat-state views bit-identical for the slot.
func (h *binaryHasher) UpdateStorage(address common.Address, keys []common.Hash, values []common.Hash) error {
	var err error
	for i, key := range keys {
		// BinaryTrie.UpdateStorage right-justifies a shorter input into
		// 32 bytes; for a non-zero common.Hash the input is already 32
		// bytes so the normalization is a no-op. For the zero-value
		// case we emit 32 zero bytes explicitly to match the trie's
		// tombstone convention.
		var blob [bintrie.HashSize]byte
		if values[i] == (common.Hash{}) {
			err = h.trie.DeleteStorage(address, key[:])
		} else {
			copy(blob[:], values[i][:])
			err = h.trie.UpdateStorage(address, key[:], blob[:])
		}
		if err != nil {
			return err
		}
		// Record the flat-state mirror write regardless of zero/non-zero:
		// the blob is 32 zero bytes in the delete case and the value in
		// the non-delete case.
		storageKey := bintrie.GetBinaryTreeKeyStorageSlot(address, key[:])
		h.recordLeaf(storageKey, blob[:])
	}
	return nil
}

// Hash implements Hasher, computing the state root hash without committing.
func (h *binaryHasher) Hash() common.Hash {
	return h.trie.Hash()
}

// Commit implements Hasher, finalizing all pending changes and returning
// the resulting state root hash, along with the set of dirty trie nodes
// generated by the updates.
func (h *binaryHasher) Commit() (common.Hash, *trienode.MergedNodeSet, map[common.Address]Hashes, error) {
	nodes := trienode.NewMergedNodeSet()
	root, set := h.trie.Commit(false)
	if set != nil {
		if err := nodes.Merge(set); err != nil {
			return common.Hash{}, nil, nil, err
		}
	}
	// The binary trie is a single unified structure with no per-account
	// storage sub-tries, so there are no secondary hashes to report.
	return root, nodes, nil, nil
}

// Copy implements Hasher, returning a deep-copied hasher instance.
func (h *binaryHasher) Copy() Hasher {
	return &binaryHasher{
		db:       h.db,
		root:     h.root,
		prefetch: false,
		trie:     h.trie.copy(),
	}
}

// ProveAccount implements Prover, constructing a proof for the given account.
func (h *binaryHasher) ProveAccount(addr common.Address, proofDb ethdb.KeyValueWriter) error {
	return h.trie.Prove(crypto.Keccak256(addr.Bytes()), proofDb)
}

// ProveStorage implements Prover, constructing a proof for the given storage
// slot of the specified account.
func (h *binaryHasher) ProveStorage(addr common.Address, key common.Hash, proofDb ethdb.KeyValueWriter) error {
	return h.trie.Prove(crypto.Keccak256(key.Bytes()), proofDb)
}

// CollectWitness implements WitnessCollector. It aggregates all trie nodes
// accessed (both read and write) across the account trie, all active storage
// tries and deleted storage tries into a single state witness.
func (h *binaryHasher) CollectWitness(witness *stateless.Witness) {
	witness.AddState(h.trie.Witness(), common.Hash{})
}

// PrefetchAccount implements Prefetcher, preloading the nodes of specific accounts.
func (h *binaryHasher) PrefetchAccount(addresses []common.Address, read bool) {
	if !h.prefetch {
		return
	}
	h.trie.prefetchAccounts(addresses, read)
}

// PrefetchStorage implements Prefetcher. The storage trie is opened eagerly
// so the prefetcher can begin loading nodes in the background.
func (h *binaryHasher) PrefetchStorage(addr common.Address, keys []common.Hash, read bool) {
	if !h.prefetch {
		return
	}
	h.trie.prefetchStorage(addr, keys, read)
}

// TermPrefetch terminates all prefetcher goroutines. Safe to call multiple times.
func (h *binaryHasher) TermPrefetch() {
	h.trie.term()
}
