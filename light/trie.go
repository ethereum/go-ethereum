// Copyright 2015 The go-ethereum Authors
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

package light

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

var (
	sha3Nil = crypto.Keccak256Hash(nil)
)

func NewState(ctx context.Context, head *types.Header, odr OdrBackend) *state.StateDB {
	state, _ := state.New(head.Root, NewStateDatabase(ctx, head, odr))
	return state
}

func NewStateDatabase(ctx context.Context, head *types.Header, odr OdrBackend) state.Database {
	return &odrDatabase{ctx, StateTrieID(head), odr}
}

// odrStateReader implements interface state.StateReader, providing the methods
// to read accounts and storage slots.
type odrStateReader struct {
	root         common.Hash                   // The specific state root the reader belongs to
	db           *odrDatabase                  // The associated odr database
	accountTrie  state.Trie                    // The cached account trie
	storageTries map[common.Address]state.Trie // The group of storage tries cached
}

// newOdrStateReader constructs the state reader with specific state root.
func newOdrStateReader(root common.Hash, db *odrDatabase) *odrStateReader {
	return &odrStateReader{
		root:         root,
		db:           db,
		storageTries: make(map[common.Address]state.Trie),
	}
}

// Account implements StateReader, retrieving the account specified by the address
// from the associated state.
func (r *odrStateReader) Account(addr common.Address) (*types.StateAccount, error) {
	if r.accountTrie == nil {
		r.accountTrie, _ = r.db.OpenTrie(r.root)
	}
	return r.accountTrie.GetAccount(addr)
}

// storageTrie returns the associated storage trie with the provided account
// address. The trie will be opened and cached locally if it's not loaded yet.
func (r *odrStateReader) storageTrie(addr common.Address) (state.Trie, error) {
	if t, ok := r.storageTries[addr]; ok {
		return t, nil
	}
	acct, err := r.Account(addr)
	if err != nil {
		return nil, err
	}
	var t state.Trie
	if acct == nil {
		t, err = r.db.OpenStorageTrie(r.root, addr, types.EmptyRootHash)
	} else {
		t, err = r.db.OpenStorageTrie(r.root, addr, acct.Root)
	}
	if err != nil {
		return nil, err
	}
	r.storageTries[addr] = t
	return t, nil
}

// Storage implements StateReader, retrieving the storage slot specified by the
// address and slot key from the associated state.
func (r *odrStateReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	t, err := r.storageTrie(addr)
	if err != nil {
		return common.Hash{}, err
	}
	ret, err := t.GetStorage(addr, key.Bytes())
	if err != nil {
		return common.Hash{}, err
	}
	var slot common.Hash
	slot.SetBytes(ret)
	return slot, nil
}

// merkleDB is the implementation of state.Database interface, designed for
// providing functionalities to read and write states in light client.
type odrDatabase struct {
	ctx     context.Context
	id      *TrieID
	backend OdrBackend
}

// StateReader constructs a reader for the specific state.
func (db *odrDatabase) StateReader(root common.Hash) (state.StateReader, error) {
	return newOdrStateReader(root, db), nil
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *odrDatabase) OpenTrie(root common.Hash) (state.Trie, error) {
	return &odrTrie{db: db, id: db.id}, nil
}

// OpenStorageTrie opens the storage trie of an account.
func (db *odrDatabase) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash) (state.Trie, error) {
	return &odrTrie{db: db, id: StorageTrieID(db.id, address, root)}, nil
}

// CopyTrie returns an independent copy of the given trie.
func (db *odrDatabase) CopyTrie(t state.Trie) state.Trie {
	switch t := t.(type) {
	case *odrTrie:
		cpy := &odrTrie{db: t.db, id: t.id}
		if t.trie != nil {
			cpy.trie = t.trie.Copy()
		}
		return cpy
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

// ReadCode implements CodeReader, retrieving a particular contract's code.
func (db *odrDatabase) ReadCode(addr common.Address, codeHash common.Hash) ([]byte, error) {
	if codeHash == sha3Nil {
		return nil, nil
	}
	code := rawdb.ReadCode(db.backend.Database(), codeHash)
	if len(code) != 0 {
		return code, nil
	}
	id := *db.id
	id.AccountAddress = addr[:]
	req := &CodeRequest{Id: &id, Hash: codeHash}
	err := db.backend.Retrieve(db.ctx, req)
	return req.Data, err
}

// ReadCodeSize implements CodeReader, retrieving a particular contracts
// code's size.
func (db *odrDatabase) ReadCodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	code, err := db.ReadCode(addr, codeHash)
	return len(code), err
}

// WriteCodes implements CodeWriter, writing the provided a list of contract
// codes into database.
func (db *odrDatabase) WriteCodes(addresses []common.Address, hashes []common.Hash, codes [][]byte) error {
	panic("not implemented")
}

// TrieDB returns the associated trie database.
func (db *odrDatabase) TrieDB() *trie.Database {
	return nil
}

// Snapshot returns the associated state snapshot, it may be nil if not configured.
func (db *odrDatabase) Snapshot() *snapshot.Tree {
	return nil
}

type odrTrie struct {
	db   *odrDatabase
	id   *TrieID
	trie *trie.Trie
}

func (t *odrTrie) GetStorage(_ common.Address, key []byte) ([]byte, error) {
	key = crypto.Keccak256(key)
	var enc []byte
	err := t.do(key, func() (err error) {
		enc, err = t.trie.Get(key)
		return err
	})
	if err != nil || len(enc) == 0 {
		return nil, err
	}
	_, content, _, err := rlp.Split(enc)
	return content, err
}

func (t *odrTrie) GetAccount(address common.Address) (*types.StateAccount, error) {
	var (
		enc []byte
		key = crypto.Keccak256(address.Bytes())
	)
	err := t.do(key, func() (err error) {
		enc, err = t.trie.Get(key)
		return err
	})
	if err != nil || len(enc) == 0 {
		return nil, err
	}
	acct := new(types.StateAccount)
	if err := rlp.DecodeBytes(enc, acct); err != nil {
		return nil, err
	}
	return acct, nil
}

func (t *odrTrie) UpdateAccount(address common.Address, acc *types.StateAccount) error {
	key := crypto.Keccak256(address.Bytes())
	value, err := rlp.EncodeToBytes(acc)
	if err != nil {
		return fmt.Errorf("decoding error in account update: %w", err)
	}
	return t.do(key, func() error {
		return t.trie.Update(key, value)
	})
}

func (t *odrTrie) UpdateContractCode(_ common.Address, _ common.Hash, _ []byte) error {
	return nil
}

func (t *odrTrie) UpdateStorage(_ common.Address, key, value []byte) error {
	key = crypto.Keccak256(key)
	v, _ := rlp.EncodeToBytes(value)
	return t.do(key, func() error {
		return t.trie.Update(key, v)
	})
}

func (t *odrTrie) DeleteStorage(_ common.Address, key []byte) error {
	key = crypto.Keccak256(key)
	return t.do(key, func() error {
		return t.trie.Delete(key)
	})
}

// DeleteAccount abstracts an account deletion from the trie.
func (t *odrTrie) DeleteAccount(address common.Address) error {
	key := crypto.Keccak256(address.Bytes())
	return t.do(key, func() error {
		return t.trie.Delete(key)
	})
}

func (t *odrTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet, error) {
	if t.trie == nil {
		return t.id.Root, nil, nil
	}
	return t.trie.Commit(collectLeaf)
}

func (t *odrTrie) Hash() common.Hash {
	if t.trie == nil {
		return t.id.Root
	}
	return t.trie.Hash()
}

func (t *odrTrie) NodeIterator(startkey []byte) (trie.NodeIterator, error) {
	return newNodeIterator(t, startkey), nil
}

func (t *odrTrie) GetKey(sha []byte) []byte {
	return nil
}

func (t *odrTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	return errors.New("not implemented, needs client/server interface split")
}

// do tries and retries to execute a function until it returns with no error or
// an error type other than MissingNodeError
func (t *odrTrie) do(key []byte, fn func() error) error {
	for {
		var err error
		if t.trie == nil {
			var id *trie.ID
			if len(t.id.AccountAddress) > 0 {
				id = trie.StorageTrieID(t.id.StateRoot, crypto.Keccak256Hash(t.id.AccountAddress), t.id.Root)
			} else {
				id = trie.StateTrieID(t.id.StateRoot)
			}
			triedb := trie.NewDatabase(t.db.backend.Database(), trie.HashDefaults)
			t.trie, err = trie.New(id, triedb)
		}
		if err == nil {
			err = fn()
		}
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
		r := &TrieRequest{Id: t.id, Key: key}
		if err := t.db.backend.Retrieve(t.db.ctx, r); err != nil {
			return err
		}
	}
}

type nodeIterator struct {
	trie.NodeIterator
	t   *odrTrie
	err error
}

func newNodeIterator(t *odrTrie, startkey []byte) trie.NodeIterator {
	it := &nodeIterator{t: t}
	// Open the actual non-ODR trie if that hasn't happened yet.
	if t.trie == nil {
		it.do(func() error {
			var id *trie.ID
			if len(t.id.AccountAddress) > 0 {
				id = trie.StorageTrieID(t.id.StateRoot, crypto.Keccak256Hash(t.id.AccountAddress), t.id.Root)
			} else {
				id = trie.StateTrieID(t.id.StateRoot)
			}
			triedb := trie.NewDatabase(t.db.backend.Database(), trie.HashDefaults)
			t, err := trie.New(id, triedb)
			if err == nil {
				it.t.trie = t
			}
			return err
		})
	}
	it.do(func() error {
		var err error
		it.NodeIterator, err = it.t.trie.NodeIterator(startkey)
		if err != nil {
			return err
		}
		return it.NodeIterator.Error()
	})
	return it
}

func (it *nodeIterator) Next(descend bool) bool {
	var ok bool
	it.do(func() error {
		ok = it.NodeIterator.Next(descend)
		return it.NodeIterator.Error()
	})
	return ok
}

// do runs fn and attempts to fill in missing nodes by retrieving.
func (it *nodeIterator) do(fn func() error) {
	var lasthash common.Hash
	for {
		it.err = fn()
		missing, ok := it.err.(*trie.MissingNodeError)
		if !ok {
			return
		}
		if missing.NodeHash == lasthash {
			it.err = fmt.Errorf("retrieve loop for trie node %x", missing.NodeHash)
			return
		}
		lasthash = missing.NodeHash
		r := &TrieRequest{Id: it.t.id, Key: nibblesToKey(missing.Path)}
		if it.err = it.t.db.backend.Retrieve(it.t.db.ctx, r); it.err != nil {
			return
		}
	}
}

func (it *nodeIterator) Error() error {
	if it.err != nil {
		return it.err
	}
	return it.NodeIterator.Error()
}

func nibblesToKey(nib []byte) []byte {
	if len(nib) > 0 && nib[len(nib)-1] == 0x10 {
		nib = nib[:len(nib)-1] // drop terminator
	}
	if len(nib)&1 == 1 {
		nib = append(nib, 0) // make even
	}
	key := make([]byte, len(nib)/2)
	for bi, ni := 0, 0; ni < len(nib); bi, ni = bi+1, ni+2 {
		key[bi] = nib[ni]<<4 | nib[ni+1]
	}
	return key
}
