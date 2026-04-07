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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// CodeMut represents a mutation to contract code.
type CodeMut struct {
	Code []byte // Null for deletion
}

// AccountMut represents a mutation to an account.
// Semantics:
// - Account == nil: delete the account
// - Code == nil:  leave code unchanged
// - Code != nil: apply the given code mutation
// - CodeSize: the account's CURRENT total code size, not just the bytes
//   carried in Code. It is used by implementations that pack the code
//   size into their on-trie account encoding (e.g. the binary trie
//   BasicData leaf). Callers must always populate this field to the
//   account's real code size, obtained via stateObject.CodeSize() or an
//   equivalent source — even on balance/nonce-only updates where the
//   code bytes themselves are not loaded. Leaving it at zero on a
//   non-code-touching update silently corrupts on-trie state for any
//   hasher that stores code size.
type AccountMut struct {
	Account  *Account // Null for deletion
	Code     *CodeMut // Null for unchanged
	CodeSize int      // Current code length (must be set by the caller)
}

// Hashes encapsulates a trie root together with its original (pre-update) root.
type Hashes struct {
	Hash common.Hash // Post-mutation root
	Prev common.Hash // Pre-mutation root
}

// StemWrite describes a single write to a bintrie stem offset. It is used
// by LeafProducer-capable hashers to report flat-state mutations derived
// from their trie updates so a downstream flat-state layer can be kept
// consistent with the hasher's on-trie view.
//
// Stem is the 31-byte common prefix of the EIP-7864 tree key. Offset is
// the index into the stem's 256-value group (0..255). Value is the
// 32-byte leaf value that was written; the caller uses the per-call
// policy documented on the binary hasher:
//   - Account create/update: two writes (BasicData, CodeHash) with
//     non-nil 32-byte values.
//   - Storage update to a non-zero value: one write with the 32-byte
//     normalized value.
//   - Storage update to zero (the bintrie's "delete" convention): one
//     write with 32 zero bytes (tombstone / present with zero).
//   - Account delete: two writes with nil values, signalling the flat
//     state to clear the corresponding offsets.
type StemWrite struct {
	Stem   [31]byte
	Offset byte
	Value  []byte
}

// LeafProducer is an optional extension to Hasher for implementations
// that track flat-state mutations alongside trie updates. Callers use it
// to harvest the set of stem writes needed to keep an out-of-band flat
// state layer consistent with the hasher's trie mutations.
//
// The binary hasher implements this interface; the merkle hasher does
// not, because merkle flat state is MPT-shaped and does not use stems.
// Callers check via a type assertion:
//
//	if lp, ok := h.(LeafProducer); ok {
//	    writes := lp.DrainStemWrites()
//	    // ... propagate writes into the state update ...
//	}
//
// DrainStemWrites is intended to be called ONCE per block, AFTER all
// UpdateAccount/UpdateStorage calls for that block have completed. The
// implementation must reset its internal buffer on drain so subsequent
// calls return only writes accumulated since the last drain.
type LeafProducer interface {
	// DrainStemWrites returns all stem writes accumulated since the last
	// drain, in the order they were produced, and resets the internal
	// buffer. The returned slice is owned by the caller; the hasher
	// allocates a fresh slice on the next update.
	DrainStemWrites() []StemWrite
}

// Hasher defines the minimal interface for computing state root hashes.
//
// It abstracts over different trie implementations, such as the traditional
// two-layer Merkle Patricia Trie (separate account and storage tries) and a
// unified single-layer binary trie (a single trie covering accounts, storages
// and contract code).
//
// This abstraction also enables alternative implementations, such as a no-op
// hasher for flat-state-only nodes (i.e. nodes that do not store trie data and
// do not perform state validation).
//
// The Hash method may be invoked multiple times and must return a hash that
// reflects all preceding state mutations. This behavior is required for
// compatibility with pre-Byzantium semantics.
type Hasher interface {
	// UpdateAccount writes a list of accounts into the state.
	UpdateAccount(addresses []common.Address, accounts []AccountMut) error

	// UpdateStorage writes a list of storage slot value.
	UpdateStorage(address common.Address, keys []common.Hash, values []common.Hash) error

	// Hash computes and returns the state root hash without committing.
	Hash() common.Hash

	// Commit finalizes all pending changes and returns the resulting state root
	// hash, along with the set of dirty trie nodes generated by the updates.
	//
	// Additionally, if the hasher uses a two-layer structure, the roots of the
	// secondary tries together with their original hashes will also be returned
	// for all mutated accounts, regardless of whether their storage was modified.
	Commit() (common.Hash, *trienode.MergedNodeSet, map[common.Address]Hashes, error)

	// Copy returns a deep-copied hasher instance.
	Copy() Hasher
}

// Prefetcher is an optional extension implemented by hashers that can
// asynchronously warm up trie/state data ahead of hashing.
type Prefetcher interface {
	// PrefetchAccount schedules the account for prefetching.
	PrefetchAccount(addresses []common.Address, read bool)

	// PrefetchStorage schedules the storage slot for prefetching.
	PrefetchStorage(addr common.Address, keys []common.Hash, read bool)

	// TermPrefetch terminates all the background prefetching activities.
	TermPrefetch()
}

// WitnessCollector is an optional extension implemented by hashers that can
// construct a state witness for the most recent committed state transition.
type WitnessCollector interface {
	// CollectWitness returns the state witness corresponding to the most recent
	// committed state transition.
	CollectWitness(*stateless.Witness)
}

// Prover is an optional extension implemented by hashers that can construct
// proofs against the current state.
type Prover interface {
	// ProveAccount constructs a proof for the given account.
	//
	// The returned proof contains all encoded nodes on the path to the account.
	// The account itself is included in the last node and can be retrieved by
	// verifying the proof.
	//
	// If the account does not exist, the returned proof contains all nodes of
	// the longest existing prefix of the account key (at least the root), ending
	// with the node that proves the absence of the account.
	ProveAccount(addr common.Address, proofDb ethdb.KeyValueWriter) error

	// ProveStorage constructs a proof for the given storage slot of the
	// specified account.
	//
	// The returned proof contains all encoded nodes on the path to the storage
	// slot. The slot value itself is included in the last node and can be
	// retrieved by verifying the proof.
	//
	// If the account or storage slot does not exist, the returned proof contains
	// the nodes required to prove its absence.
	ProveStorage(addr common.Address, key common.Hash, proofDb ethdb.KeyValueWriter) error
}

// noopHasher is a Hasher implementation that performs no work and always
// returns an empty state root.
type noopHasher struct{}

func (n *noopHasher) UpdateAccount([]common.Address, []AccountMut) error { return nil }
func (n *noopHasher) UpdateStorage(common.Address, []common.Hash, []common.Hash) error {
	return nil
}
func (n *noopHasher) Hash() common.Hash { return common.Hash{} }
func (n *noopHasher) Commit() (common.Hash, *trienode.MergedNodeSet, map[common.Address]Hashes, error) {
	return common.Hash{}, trienode.NewMergedNodeSet(), make(map[common.Address]Hashes), nil
}
func (n *noopHasher) Copy() Hasher { return &noopHasher{} }
func (n *noopHasher) Close()       {}
