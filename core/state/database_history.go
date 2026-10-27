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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// historicStateReader implements StateReader, wrapping a historical state reader
// defined in path database and provide historic state serving over the path scheme.
type historicStateReader struct {
	reader *pathdb.HistoricalStateReader
	lock   sync.Mutex // Lock for protecting concurrent read
}

// newHistoricStateReader constructs a reader for historical state serving.
func newHistoricStateReader(r *pathdb.HistoricalStateReader) *historicStateReader {
	return &historicStateReader{reader: r}
}

// Account implements StateReader, retrieving the account specified by the address.
func (r *historicStateReader) Account(addr common.Address) (*types.StateAccount, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	account, err := r.reader.Account(addr)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil
	}
	acct := &types.StateAccount{
		Nonce:    account.Nonce,
		Balance:  account.Balance,
		CodeHash: account.CodeHash,
		Root:     common.BytesToHash(account.Root),
	}
	if len(acct.CodeHash) == 0 {
		acct.CodeHash = types.EmptyCodeHash.Bytes()
	}
	if acct.Root == (common.Hash{}) {
		acct.Root = types.EmptyRootHash
	}
	return acct, nil
}

// Storage implements StateReader, retrieving the storage slot specified by the
// address and slot key.
//
// An error will be returned if the associated snapshot is already stale or
// the requested storage slot is not yet covered by the snapshot.
//
// The returned storage slot might be empty if it's not existent.
func (r *historicStateReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	blob, err := r.reader.Storage(addr, key)
	if err != nil {
		return common.Hash{}, err
	}
	if len(blob) == 0 {
		return common.Hash{}, nil
	}
	_, content, _, err := rlp.Split(blob)
	if err != nil {
		return common.Hash{}, err
	}
	var slot common.Hash
	slot.SetBytes(content)
	return slot, nil
}

// historicTrieOpener is a wrapper of pathdb.HistoricalNodeReader, implementing
// the database.NodeDatabase by adding NodeReader function.
//
// It also serves the preimages from the given trie database, if any, so that
// a trie opened on it can resolve the raw keys of the hashed ones. Preimages
// are not versioned, so the live store serves every historical state.
type historicTrieOpener struct {
	root      common.Hash
	reader    *pathdb.HistoricalNodeReader
	preimages *triedb.Database
}

// newHistoricTrieOpener constructs the historical trie opener.
func newHistoricTrieOpener(root common.Hash, reader *pathdb.HistoricalNodeReader, preimages *triedb.Database) *historicTrieOpener {
	return &historicTrieOpener{
		root:      root,
		reader:    reader,
		preimages: preimages,
	}
}

// Preimage retrieves the preimage of the specified hash.
func (o *historicTrieOpener) Preimage(hash common.Hash) []byte {
	if o.preimages == nil {
		return nil
	}
	return o.preimages.Preimage(hash)
}

// InsertPreimage is a no-op, as the historical trie is read-only.
func (o *historicTrieOpener) InsertPreimage(map[common.Hash][]byte) {}

// PreimageEnabled returns true if the preimage store is enabled.
func (o *historicTrieOpener) PreimageEnabled() bool {
	return o.preimages != nil && o.preimages.PreimageEnabled()
}

// NodeReader implements database.NodeDatabase, returning a node reader of a
// specified state.
func (o *historicTrieOpener) NodeReader(root common.Hash) (database.NodeReader, error) {
	if root != o.root {
		return nil, fmt.Errorf("state %x is not available", root)
	}
	return o.reader, nil
}

// historicalTrieReader wraps a historical node reader defined in path database,
// providing historical node serving over the path scheme.
type historicalTrieReader struct {
	root   common.Hash
	opener *historicTrieOpener
	tr     Trie

	subRoots map[common.Address]common.Hash // Set of storage roots, cached when the account is resolved
	subTries map[common.Address]Trie        // Group of storage tries, cached when it's resolved
	lock     sync.Mutex                     // Lock for protecting concurrent read
}

// newHistoricalTrieReader constructs a reader for historical trie node serving.
func newHistoricalTrieReader(root common.Hash, r *pathdb.HistoricalNodeReader) (*historicalTrieReader, error) {
	opener := newHistoricTrieOpener(root, r, nil)
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), opener)
	if err != nil {
		return nil, err
	}
	return &historicalTrieReader{
		root:     root,
		opener:   opener,
		tr:       tr,
		subRoots: make(map[common.Address]common.Hash),
		subTries: make(map[common.Address]Trie),
	}, nil
}

// account is the inner version of Account and assumes the r.lock is already held.
func (r *historicalTrieReader) account(addr common.Address) (*types.StateAccount, error) {
	account, err := r.tr.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	if account == nil {
		r.subRoots[addr] = types.EmptyRootHash
	} else {
		r.subRoots[addr] = account.Root
	}
	return account, nil
}

// Account implements StateReader, retrieving the account specified by the address.
//
// An error will be returned if the associated snapshot is already stale or
// the requested account is not yet covered by the snapshot.
//
// The returned account might be nil if it's not existent.
func (r *historicalTrieReader) Account(addr common.Address) (*types.StateAccount, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.account(addr)
}

// Storage implements StateReader, retrieving the storage slot specified by the
// address and slot key.
//
// An error will be returned if the associated snapshot is already stale or
// the requested storage slot is not yet covered by the snapshot.
//
// The returned storage slot might be empty if it's not existent.
func (r *historicalTrieReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	tr, found := r.subTries[addr]
	if !found {
		root, ok := r.subRoots[addr]

		// The storage slot is accessed without account caching. It's unexpected
		// behavior but try to resolve the account first anyway.
		if !ok {
			_, err := r.account(addr)
			if err != nil {
				return common.Hash{}, err
			}
			root = r.subRoots[addr]
		}
		var err error
		tr, err = trie.NewStateTrie(trie.StorageTrieID(r.root, crypto.Keccak256Hash(addr.Bytes()), root), r.opener)
		if err != nil {
			return common.Hash{}, err
		}
		r.subTries[addr] = tr
	}
	ret, err := tr.GetStorage(addr, key.Bytes())
	if err != nil {
		return common.Hash{}, err
	}
	var value common.Hash
	value.SetBytes(ret)
	return value, nil
}

// HistoricDB is the implementation of Database interface, with the ability to
// access historical state.
type HistoricDB struct {
	triedb *triedb.Database
	codedb *CodeDB
}

// Type returns the trie type of the underlying database.
func (db *HistoricDB) Type() DatabaseType {
	// TODO(rjl493456442) support UBT in the future
	return TypeMPT
}

// NewHistoricDatabase creates a historic state database.
func NewHistoricDatabase(triedb *triedb.Database, codedb *CodeDB) *HistoricDB {
	return &HistoricDB{
		triedb: triedb,
		codedb: codedb,
	}
}

// Reader implements Database interface, returning a reader of the specific state.
func (db *HistoricDB) Reader(stateRoot common.Hash) (Reader, error) {
	var readers []StateReader
	sr, err := db.triedb.HistoricStateReader(stateRoot)
	if err == nil {
		readers = append(readers, newHistoricStateReader(sr))
	}
	nr, err := db.triedb.HistoricNodeReader(stateRoot)
	if err == nil {
		tr, err := newHistoricalTrieReader(stateRoot, nr)
		if err == nil {
			readers = append(readers, tr)
		}
	}
	if len(readers) == 0 {
		return nil, fmt.Errorf("historical state %x is not available", stateRoot)
	}
	combined, err := newMultiStateReader(readers...)
	if err != nil {
		return nil, err
	}
	return newReader(db.codedb.Reader(), combined), nil
}

// OpenTrie opens the main account trie. It's not supported by historic database.
func (db *HistoricDB) OpenTrie(root common.Hash) (Trie, error) {
	nr, err := db.triedb.HistoricNodeReader(root)
	if err != nil {
		return nil, err
	}
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), newHistoricTrieOpener(root, nr, db.triedb))
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenStorageTrie opens the storage trie of an account. It's not supported by
// historic database.
func (db *HistoricDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, _ Trie) (Trie, error) {
	nr, err := db.triedb.HistoricNodeReader(stateRoot)
	if err != nil {
		return nil, err
	}
	id := trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root)
	tr, err := trie.NewStateTrie(id, newHistoricTrieOpener(stateRoot, nr, db.triedb))
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// TrieDB returns the underlying trie database for managing trie nodes.
func (db *HistoricDB) TrieDB() *triedb.Database {
	return db.triedb
}

// Commit flushes all pending writes and finalizes the state transition,
// committing the changes to the underlying storage. It returns an error
// if the commit fails.
func (db *HistoricDB) Commit(update *StateUpdate) error {
	return errors.New("not implemented")
}

// Iteratee returns a state iteratee associated with the specified state root,
// through which the account iterator and storage iterator can be created.
func (db *HistoricDB) Iteratee(root common.Hash) (Iteratee, error) {
	if _, err := db.triedb.HistoricNodeReader(root); err != nil {
		return nil, err
	}
	return &historicIteratee{db: db, root: root}, nil
}

// historicIteratee implements Iteratee, traversing a historical state through
// its tries, whose nodes the path database rebuilds from the trienode history.
// No flat state exists for a historical state, so the traversal is much slower
// than the one of a live state.
type historicIteratee struct {
	db   *HistoricDB
	root common.Hash
}

// NewAccountIterator creates an account iterator for the state, beginning at
// the given account hash, or the next one if it does not exist.
func (it *historicIteratee) NewAccountIterator(start common.Hash) (AccountIterator, error) {
	tr, err := it.db.OpenTrie(it.root)
	if err != nil {
		return nil, err
	}
	return newMerkleTrieIterator(tr, start, true)
}

// NewStorageIterator creates a storage iterator for the account specified by
// the address hash, beginning at the given slot hash, or the next one if it
// does not exist.
func (it *historicIteratee) NewStorageIterator(addressHash common.Hash, start common.Hash) (StorageIterator, error) {
	nr, err := it.db.triedb.HistoricNodeReader(it.root)
	if err != nil {
		return nil, err
	}
	opener := newHistoricTrieOpener(it.root, nr, it.db.triedb)
	tr, err := trie.NewStateTrie(trie.StateTrieID(it.root), opener)
	if err != nil {
		return nil, err
	}
	acct, err := tr.GetAccountByHash(addressHash)
	if err != nil {
		return nil, err
	}
	if acct == nil || acct.Root == types.EmptyRootHash {
		return &exhaustedIterator{}, nil
	}
	storageTr, err := trie.NewStateTrie(trie.StorageTrieID(it.root, addressHash, acct.Root), opener)
	if err != nil {
		return nil, err
	}
	return newMerkleTrieIterator(storageTr, start, false)
}
