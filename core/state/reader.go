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

package state

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// ContractCodeReader defines the interface for accessing contract code.
type ContractCodeReader interface {
	// Code retrieves a particular contract's code.
	//
	// - Returns nil code along with nil error if the requested contract code
	//   doesn't exist
	// - Returns an error only if an unexpected issue occurs
	Code(addr common.Address, codeHash common.Hash) ([]byte, error)

	// CodeSize retrieves a particular contracts code's size.
	//
	// - Returns zero code size along with nil error if the requested contract code
	//   doesn't exist
	// - Returns an error only if an unexpected issue occurs
	CodeSize(addr common.Address, codeHash common.Hash) (int, error)
}

// StateReader defines the interface for accessing accounts and storage slots
// associated with a specific state.
type StateReader interface {
	// Account retrieves the account associated with a particular address.
	//
	// - Returns a nil account if it does not exist
	// - Returns an error only if an unexpected issue occurs
	// - The returned account is safe to modify after the call
	Account(addr common.Address) (*types.StateAccount, error)

	// Storage retrieves the storage slot associated with a particular account
	// address and slot key.
	//
	// - Returns an empty slot if it does not exist
	// - Returns an error only if an unexpected issue occurs
	// - The returned storage slot is safe to modify after the call
	Storage(addr common.Address, slot common.Hash) (common.Hash, error)
}

// Reader defines the interface for accessing accounts, storage slots and contract
// code associated with a specific state.
type Reader interface {
	ContractCodeReader
	StateReader
}

// cachingCodeReader implements ContractCodeReader, accessing contract code either in
// local key-value store or the shared code cache.
type cachingCodeReader struct {
	db ethdb.KeyValueReader

	// These caches could be shared by multiple code reader instances,
	// they are natively thread-safe.
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
}

// newCachingCodeReader constructs the code reader.
func newCachingCodeReader(db ethdb.KeyValueReader, codeCache *lru.SizeConstrainedCache[common.Hash, []byte], codeSizeCache *lru.Cache[common.Hash, int]) *cachingCodeReader {
	return &cachingCodeReader{
		db:            db,
		codeCache:     codeCache,
		codeSizeCache: codeSizeCache,
	}
}

// Code implements ContractCodeReader, retrieving a particular contract's code.
// If the contract code doesn't exist, no error will be returned.
func (r *cachingCodeReader) Code(addr common.Address, codeHash common.Hash) ([]byte, error) {
	code, _ := r.codeCache.Get(codeHash)
	if len(code) > 0 {
		return code, nil
	}
	code = rawdb.ReadCode(r.db, codeHash)
	if len(code) > 0 {
		r.codeCache.Add(codeHash, code)
		r.codeSizeCache.Add(codeHash, len(code))
	}
	return code, nil
}

// CodeSize implements ContractCodeReader, retrieving a particular contracts code's size.
// If the contract code doesn't exist, no error will be returned.
func (r *cachingCodeReader) CodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	if cached, ok := r.codeSizeCache.Get(codeHash); ok {
		return cached, nil
	}
	code, err := r.Code(addr, codeHash)
	if err != nil {
		return 0, err
	}
	return len(code), nil
}

// flatReader wraps a database state reader.
type flatReader struct {
	reader database.StateReader
	buff   crypto.KeccakState
}

// newFlatReader constructs a state reader with on the given state root.
func newFlatReader(reader database.StateReader) *flatReader {
	return &flatReader{
		reader: reader,
		buff:   crypto.NewKeccakState(),
	}
}

// Account implements StateReader, retrieving the account specified by the address.
//
// An error will be returned if the associated snapshot is already stale or
// the requested account is not yet covered by the snapshot.
//
// The returned account might be nil if it's not existent.
func (r *flatReader) Account(addr common.Address) (*types.StateAccount, error) {
	account, err := r.reader.Account(crypto.HashData(r.buff, addr.Bytes()))
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
func (r *flatReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	addrHash := crypto.HashData(r.buff, addr.Bytes())
	slotHash := crypto.HashData(r.buff, key.Bytes())
	ret, err := r.reader.Storage(addrHash, slotHash)
	if err != nil {
		return common.Hash{}, err
	}
	if len(ret) == 0 {
		return common.Hash{}, nil
	}
	// Perform the rlp-decode as the slot value is RLP-encoded in the state
	// snapshot.
	_, content, _, err := rlp.Split(ret)
	if err != nil {
		return common.Hash{}, err
	}
	var value common.Hash
	value.SetBytes(content)
	return value, nil
}

// trieReader implements the StateReader interface, providing functions to access
// state from the referenced trie.
type trieReader struct {
	root     common.Hash                    // State root which uniquely represent a state
	db       *triedb.Database               // Database for loading trie
	buff     crypto.KeccakState             // Buffer for keccak256 hashing
	mainTrie Trie                           // Main trie, resolved in constructor
	subRoots map[common.Address]common.Hash // Set of storage roots, cached when the account is resolved
	subTries map[common.Address]Trie        // Group of storage tries, cached when it's resolved
}

// trieReader constructs a trie reader of the specific state. An error will be
// returned if the associated trie specified by root is not existent.
func newTrieReader(root common.Hash, db *triedb.Database, cache *utils.PointCache) (*trieReader, error) {
	var (
		tr  Trie
		err error
	)
	if !db.IsVerkle() {
		tr, err = trie.NewStateTrie(trie.StateTrieID(root), db)
	} else {
		tr, err = trie.NewVerkleTrie(root, db, cache)
	}
	if err != nil {
		return nil, err
	}
	return &trieReader{
		root:     root,
		db:       db,
		buff:     crypto.NewKeccakState(),
		mainTrie: tr,
		subRoots: make(map[common.Address]common.Hash),
		subTries: make(map[common.Address]Trie),
	}, nil
}

// Account implements StateReader, retrieving the account specified by the address.
//
// An error will be returned if the trie state is corrupted. An nil account
// will be returned if it's not existent in the trie.
func (r *trieReader) Account(addr common.Address) (*types.StateAccount, error) {
	account, err := r.mainTrie.GetAccount(addr)
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

// Storage implements StateReader, retrieving the storage slot specified by the
// address and slot key.
//
// An error will be returned if the trie state is corrupted. An empty storage
// slot will be returned if it's not existent in the trie.
func (r *trieReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	var (
		tr    Trie
		found bool
		value common.Hash
	)
	if r.db.IsVerkle() {
		tr = r.mainTrie
	} else {
		tr, found = r.subTries[addr]
		if !found {
			root, ok := r.subRoots[addr]

			// The storage slot is accessed without account caching. It's unexpected
			// behavior but try to resolve the account first anyway.
			if !ok {
				_, err := r.Account(addr)
				if err != nil {
					return common.Hash{}, err
				}
				root = r.subRoots[addr]
			}
			var err error
			tr, err = trie.NewStateTrie(trie.StorageTrieID(r.root, crypto.HashData(r.buff, addr.Bytes()), root), r.db)
			if err != nil {
				return common.Hash{}, err
			}
			r.subTries[addr] = tr
		}
	}
	ret, err := tr.GetStorage(addr, key.Bytes())
	if err != nil {
		return common.Hash{}, err
	}
	value.SetBytes(ret)
	return value, nil
}

// multiStateReader is the aggregation of a list of StateReader interface,
// providing state access by leveraging all readers. The checking priority
// is determined by the position in the reader list.
type multiStateReader struct {
	readers []StateReader // List of state readers, sorted by checking priority
}

// newMultiStateReader constructs a multiStateReader instance with the given
// readers. The priority among readers is assumed to be sorted. Note, it must
// contain at least one reader for constructing a multiStateReader.
func newMultiStateReader(readers ...StateReader) (*multiStateReader, error) {
	if len(readers) == 0 {
		return nil, errors.New("empty reader set")
	}
	return &multiStateReader{
		readers: readers,
	}, nil
}

// Account implementing StateReader interface, retrieving the account associated
// with a particular address.
//
// - Returns a nil account if it does not exist
// - Returns an error only if an unexpected issue occurs
// - The returned account is safe to modify after the call
func (r *multiStateReader) Account(addr common.Address) (*types.StateAccount, error) {
	var errs []error
	for _, reader := range r.readers {
		acct, err := reader.Account(addr)
		if err == nil {
			return acct, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}

// Storage implementing StateReader interface, retrieving the storage slot
// associated with a particular account address and slot key.
//
// - Returns an empty slot if it does not exist
// - Returns an error only if an unexpected issue occurs
// - The returned storage slot is safe to modify after the call
func (r *multiStateReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	var errs []error
	for _, reader := range r.readers {
		slot, err := reader.Storage(addr, slot)
		if err == nil {
			return slot, nil
		}
		errs = append(errs, err)
	}
	return common.Hash{}, errors.Join(errs...)
}

// reader is the wrapper of ContractCodeReader and StateReader interface.
type reader struct {
	ContractCodeReader
	StateReader
}

// newReader constructs a reader with the supplied code reader and state reader.
func newReader(codeReader ContractCodeReader, stateReader StateReader) *reader {
	return &reader{
		ContractCodeReader: codeReader,
		StateReader:        stateReader,
	}
}
