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
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/transitiontrie"
	"github.com/ethereum/go-ethereum/triedb"
)

// UBTDatabase is an implementation of Database interface for Unified Binary Trie.
//
// In its plain form it uses a single binary trie database. During the
// MPT-to-binary transition, an optional MPT trie database (mpttriedb) and a
// frozen base root provide read-only access to pre-transition state. The
// wrapInTransitionTrie flag controls whether reads at this database are
// served via a TransitionTrie that overlays the binary trie on the MPT
// base; it is normally driven by chainConfig.UBTTransitionActive.
type UBTDatabase struct {
	triedb               *triedb.Database
	mpttriedb            *triedb.Database
	codedb               *CodeDB
	baseRoot             common.Hash
	wrapInTransitionTrie bool
}

// Type returns Binary, indicating this database is backed by a Universal Binary Trie.
func (db *UBTDatabase) Type() DatabaseType { return TypeUBT }

// NewUBTDatabase creates a state database with the Unified binary trie manner.
// State access is wrapped in a TransitionTrie by default (which degenerates
// to a passthrough when there is no MPT base) so callers that don't care
// about the override get sensible defaults.
func NewUBTDatabase(triedb *triedb.Database, codedb *CodeDB) *UBTDatabase {
	if codedb == nil {
		codedb = NewCodeDB(triedb.Disk())
	}
	return &UBTDatabase{
		triedb:               triedb,
		codedb:               codedb,
		wrapInTransitionTrie: true,
	}
}

// NewTransitionUBTDatabase creates a UBTDatabase for the active MPT-to-binary
// transition window. The binary trie is the primary store; reads fall through
// to the frozen MPT at baseRoot via the supplied mpttriedb, and writes go
// only to the binary trie.
func NewTransitionUBTDatabase(bintriedb, mpttriedb *triedb.Database, codedb *CodeDB, baseRoot common.Hash) *UBTDatabase {
	if codedb == nil {
		codedb = NewCodeDB(bintriedb.Disk())
	}
	return &UBTDatabase{
		triedb:               bintriedb,
		mpttriedb:            mpttriedb,
		codedb:               codedb,
		baseRoot:             baseRoot,
		wrapInTransitionTrie: true,
	}
}

// WithTransitionTreeWrap toggles whether reads at this database are wrapped
// in a TransitionTrie. Setting it to false disables the wrap regardless of
// the registry state and is intended for callers that have already crossed
// the configured UBTTransitionEndTime.
func (db *UBTDatabase) WithTransitionTreeWrap(wrap bool) *UBTDatabase {
	db.wrapInTransitionTrie = wrap
	return db
}

// StateReader returns a state reader associated with the specified state root.
func (db *UBTDatabase) StateReader(stateRoot common.Hash) (StateReader, error) {
	var readers []StateReader

	// Configure the state reader using the path database in path mode.
	// This reader offers improved performance but is optional and only
	// partially useful if the snapshot data in path database is not
	// fully generated.
	if db.TrieDB().Scheme() == rawdb.PathScheme {
		reader, err := db.triedb.StateReader(stateRoot)
		if err == nil {
			readers = append(readers, newFlatReader(reader))
		}
	}
	// Configure the trie reader, which is expected to be available as the
	// gatekeeper unless the state is corrupted.
	tr, err := newUBTTrieReader(stateRoot, db.triedb, db.mpttriedb, db.wrapInTransitionTrie)
	if err != nil {
		return nil, err
	}
	readers = append(readers, tr)

	return newMultiStateReader(readers...)
}

// Reader implements Database, returning a reader associated with the specified
// state root.
func (db *UBTDatabase) Reader(stateRoot common.Hash) (Reader, error) {
	sr, err := db.StateReader(stateRoot)
	if err != nil {
		return nil, err
	}
	return newReader(db.codedb.Reader(), sr), nil
}

// ReadersWithCacheStats creates a pair of state readers that share the same
// underlying state reader and internal state cache, while maintaining separate
// statistics respectively.
func (db *UBTDatabase) ReadersWithCacheStats(stateRoot common.Hash) (Reader, Reader, error) {
	r, err := db.StateReader(stateRoot)
	if err != nil {
		return nil, nil, err
	}
	sr := newStateReaderWithCache(r)
	ra := newReader(db.codedb.Reader(), newStateReaderWithStats(sr))
	rb := newReader(db.codedb.Reader(), newStateReaderWithStats(sr))
	return ra, rb, nil
}

// OpenTrie opens the main account trie at a specific root hash. During an
// active transition, the binary trie is wrapped in a TransitionTrie so writes
// land on the binary trie while reads fall through to the frozen MPT base.
func (db *UBTDatabase) OpenTrie(root common.Hash) (Trie, error) {
	bt, err := bintrie.NewBinaryTrie(root, db.triedb)
	if err != nil {
		return nil, err
	}
	if db.mpttriedb == nil || db.baseRoot == (common.Hash{}) {
		return transitiontrie.NewTransitionTrie(nil, bt, false), nil
	}
	base, err := trie.NewStateTrie(trie.StateTrieID(db.baseRoot), db.mpttriedb)
	if err != nil {
		return nil, err
	}
	return transitiontrie.NewTransitionTrie(base, bt, false), nil
}

// OpenStorageTrie opens the storage trie of an account. In binary trie mode
// the unified trie carries all state, so the main trie is reused. During the
// transition, an MPT storage trie is opened for accounts that have not yet
// been migrated.
func (db *UBTDatabase) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, self Trie) (Trie, error) {
	if self != nil && self.IsUBT() {
		return self, nil
	}
	if db.mpttriedb == nil {
		return nil, errors.New("no MPT trie database available for storage trie outside the transition window")
	}
	return trie.NewStateTrie(trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root), db.mpttriedb)
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *UBTDatabase) TrieDB() *triedb.Database {
	return db.triedb
}

// Commit flushes all pending writes and finalizes the state transition,
// committing the changes to the underlying storage. It returns an error
// if the commit fails.
func (db *UBTDatabase) Commit(update *StateUpdate) error {
	// Short circuit if nothing to commit
	if update.Empty() {
		return nil
	}
	// Commit dirty contract code if any exists
	if len(update.Codes) > 0 {
		batch := db.codedb.NewBatchWithSize(len(update.Codes))
		for _, code := range update.Codes {
			batch.Put(code.Hash, code.Blob)
		}
		if err := batch.Commit(); err != nil {
			return err
		}
	}
	// On the first transition block, the originRoot is the MPT base root,
	// but the binary trie's parent state is empty. Substitute the empty
	// binary hash so triedb.Update doesn't reject the mismatch.
	originRoot := update.OriginRoot
	if db.mpttriedb != nil && originRoot == db.baseRoot {
		originRoot = types.EmptyBinaryHash
	}
	// Encode the state mutations in the UBT format
	accounts, accountOrigin, storages, storageOrigin := update.EncodeUBTState()

	return db.triedb.Update(update.Root, originRoot, update.BlockNumber, update.Nodes, &triedb.StateSet{
		Accounts:       accounts,
		AccountsOrigin: accountOrigin,
		Storages:       storages,
		StoragesOrigin: storageOrigin,
		RawStorageKey:  update.StorageKeyType == StorageKeyPlain,
	})
}

// Iteratee returns a state iteratee associated with the specified state root,
// through which the account iterator and storage iterator can be created.
func (db *UBTDatabase) Iteratee(root common.Hash) (Iteratee, error) {
	return newStateIteratee(false, root, db.triedb, nil)
}
