// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"io"
	"maps"
	"math/big"
	"slices"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie/trienode"
)

type Storage map[common.Hash]common.Hash

func (s Storage) Copy() Storage {
	return maps.Clone(s)
}

// stateObject represents an Ethereum account which is being modified.
//
// The usage pattern is as follows:
// - First you need to obtain a state object.
// - Account values as well as storages can be accessed and modified through the object.
// - Finally, call commit to return the changes of storage trie and update account data.
type stateObject struct {
	db       *StateDB
	address  common.Address      // address of ethereum account
	addrHash common.Hash         // hash of ethereum address of the account
	origin   *types.StateAccount // Account original data without any change applied, nil means it was not existent
	data     types.StateAccount  // Account data with all mutations applied in the scope of block

	// Write caches.
	trie Trie   // storage trie, which becomes non-nil on first access
	code []byte // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution, reset for every transaction

	// Cache flags.
	dirtyCode bool // true if the code was updated

	// Flag whether the account was marked as self-destructed. The self-destructed
	// account is still accessible in the scope of same transaction.
	selfDestructed bool

	// This is an EIP-6780 flag indicating whether the object is eligible for
	// self-destruct according to EIP-6780. The flag could be set either when
	// the contract is just created within the current transaction, or when the
	// object was previously existent and is being deployed as a contract within
	// the current transaction.
	newContract bool
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, types.EmptyCodeHash.Bytes())
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, acct *types.StateAccount) *stateObject {
	origin := acct
	if acct == nil {
		acct = types.NewEmptyStateAccount()
	}
	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		origin:         origin,
		data:           *acct,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// EncodeRLP implements rlp.Encoder.
func (s *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &s.data)
}

func (s *stateObject) markSelfdestructed() {
	s.selfDestructed = true
}

func (s *stateObject) touch() {
	s.db.journal.append(touchChange{
		account: s.address,
	})
	if s.address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		s.db.journal.dirty(s.address)
	}
}

// getTrie returns the associated storage trie. The trie will be opened
// if it's not loaded previously. An error will be returned if trie can't
// be loaded.
func (s *stateObject) getTrie() (Trie, error) {
	if s.trie == nil {
		tr, err := s.db.db.OpenStorageTrie(s.db.originalRoot, s.address, s.data.Root)
		if err != nil {
			return nil, err
		}
		s.trie = tr
	}
	return s.trie, nil
}

// GetState retrieves a value from the account storage trie.
func (s *stateObject) GetState(key common.Hash) common.Hash {
	value, _ := s.getState(key)
	return value
}

// getState retrieves a value from the account storage trie and also returns if
// the slot is already dirty or not.
func (s *stateObject) getState(key common.Hash) (common.Hash, bool) {
	// If we have a dirty value for this state entry, return it
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value, true
	}
	// Otherwise return the entry's original value
	return s.GetCommittedState(key), false
}

func (s *stateObject) GetCommittedState(key common.Hash) common.Hash {
	// If we have a pending write or clean cached, return that
	if value, pending := s.pendingStorage[key]; pending {
		return value
	}
	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If the object was destructed in *this* block (and potentially resurrected),
	// the storage has been cleared out, and we should *not* consult the previous
	// database about any storage values. The only possible alternatives are:
	//   1) resurrect happened, and new slot values were set -- those should
	//      have been handles via pendingStorage above.
	//   2) we don't have new values, and can deliver empty response back
	if _, destructed := s.db.stateObjectsDestruct[s.address]; destructed {
		return common.Hash{}
	}
	// Track the amount of time wasted on reading the storage trie
	start := time.Now()
	// Otherwise load the value from the database
	tr, err := s.getTrie()
	if err != nil {
		s.db.setError(err)
		return common.Hash{}
	}
	val, err := tr.GetStorage(s.address, key.Bytes())
	s.db.StorageReads += time.Since(start)
	if err != nil {
		s.db.setError(err)
		return common.Hash{}
	}
	var value common.Hash
	value.SetBytes(val)
	s.originStorage[key] = value
	return value
}

// SetState updates a value in account storage.
func (s *stateObject) SetState(key, value common.Hash) common.Hash {
	// If the new value is the same as old, don't set. Otherwise, track only the
	// dirty changes, supporting reverting all of it back to no change.
	prev, dirty := s.getState(key)
	if prev == value {
		return prev
	}
	var prevvalue *common.Hash
	if dirty {
		prevvalue = &prev
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:   s.address,
		key:       key,
		prevvalue: prevvalue,
	})
	s.setState(key, &value)
	return prev
}

// setState updates a value in account dirty storage. If the value being set is
// nil (assuming journal revert), the dirtiness is removed.
func (s *stateObject) setState(key common.Hash, value *common.Hash) {
	// If the first set is being reverted, undo the dirty marker
	if value == nil {
		delete(s.dirtyStorage, key)
		return
	}
	// Otherwise set/update the dirty slot value (or restore it when invoked from a revert)
	s.dirtyStorage[key] = *value
}

// finalise moves all dirty storage slots into the pending area to be hashed or
// committed later. It is invoked at the end of every transaction.
func (s *stateObject) finalise() {
	for key, value := range s.dirtyStorage {
		s.pendingStorage[key] = value
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
	// Revoke the flag at the end of the transaction. It finalizes the status
	// of the newly-created object as it's no longer eligible for self-destruct
	// by EIP-6780. For non-newly-created objects, it's a no-op.
	s.newContract = false
}

// updateTrie writes cached storage modifications into the object's storage trie.
// It will return nil if the trie has not been loaded and no changes have been
// made. An error will be returned if the trie can't be loaded/updated correctly.
func (s *stateObject) updateTrie() (Trie, error) {
	// Make sure all dirty slots are finalized into the pending storage area
	s.finalise()
	if len(s.pendingStorage) == 0 {
		return s.trie, nil
	}
	// Track the amount of time wasted on updating the storage trie
	defer func(start time.Time) { s.db.StorageUpdates += time.Since(start) }(time.Now())
	// The snapshot storage map for the object
	var (
		storage map[common.Hash][]byte
		origin  map[common.Hash][]byte
		hasher  = s.db.hasher
	)
	tr, err := s.getTrie()
	if err != nil {
		s.db.setError(err)
		return nil, err
	}
	// Insert all the pending updates into the trie
	for key, value := range s.pendingStorage {
		// Skip noop changes, persist actual changes
		if value == s.originStorage[key] {
			continue
		}
		prev := s.originStorage[key]
		s.originStorage[key] = value

		// rlp-encoded value to be used by the snapshot
		var snapshotVal []byte
		if (value == common.Hash{}) {
			if err := tr.DeleteStorage(s.address, key[:]); err != nil {
				s.db.setError(err)
				return nil, err
			}
			s.db.StorageDeleted += 1
		} else {
			trimmedVal := common.TrimLeftZeroes(value[:])
			// Encoding []byte cannot fail, ok to ignore the error.
			snapshotVal, _ = rlp.EncodeToBytes(trimmedVal)
			if err := tr.UpdateStorage(s.address, key[:], trimmedVal); err != nil {
				s.db.setError(err)
				return nil, err
			}
			s.db.StorageUpdated += 1
		}
		// Cache the mutated storage slots until commit
		if storage == nil {
			if storage = s.db.storages[s.addrHash]; storage == nil {
				storage = make(map[common.Hash][]byte)
				s.db.storages[s.addrHash] = storage
			}
		}
		khash := crypto.HashData(hasher, key[:])
		storage[khash] = snapshotVal // snapshotVal will be nil if it's deleted

		// Cache the original value of mutated storage slots
		if origin == nil {
			if origin = s.db.storagesOrigin[s.address]; origin == nil {
				origin = make(map[common.Hash][]byte)
				s.db.storagesOrigin[s.address] = origin
			}
		}
		// Track the original value of slot only if it's mutated first time
		if _, ok := origin[khash]; !ok {
			if prev == (common.Hash{}) {
				origin[khash] = nil // nil if it was not present previously
			} else {
				// Encoding []byte cannot fail, ok to ignore the error.
				b, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(prev[:]))
				origin[khash] = b
			}
		}
	}
	if len(s.pendingStorage) > 0 {
		s.pendingStorage = make(Storage)
	}
	return tr, nil
}

// UpdateRoot sets the trie root to the current root hash of. An error
// will be returned if trie root hash is not computed correctly.
func (s *stateObject) updateRoot() {
	tr, err := s.updateTrie()
	if err != nil {
		return
	}
	// If nothing changed, don't bother with hashing anything
	if tr == nil {
		return
	}
	// Track the amount of time wasted on hashing the storage trie
	defer func(start time.Time) { s.db.StorageHashes += time.Since(start) }(time.Now())
	s.data.Root = tr.Hash()
}

// commit returns the changes made in storage trie and updates the account data.
func (s *stateObject) commit() (*trienode.NodeSet, error) {
	tr, err := s.updateTrie()
	if err != nil {
		return nil, err
	}
	// If nothing changed, don't bother with hashing anything
	if tr == nil {
		s.origin = s.data.Copy()
		return nil, nil
	}
	// Track the amount of time wasted on committing the storage trie
	defer func(start time.Time) { s.db.StorageCommits += time.Since(start) }(time.Now())
	root, nodes, err := tr.Commit(false)
	if err != nil {
		return nil, err
	}
	s.data.Root = root

	// Update original account data after commit
	s.origin = s.data.Copy()
	return nodes, nil
}

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
// returns the previous balance
func (s *stateObject) AddBalance(amount *big.Int) *big.Int {
	// EIP161: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if s.empty() {
			s.touch()
		}
		return new(big.Int).Set(s.Balance())
	}
	return s.SetBalance(new(big.Int).Add(s.Balance(), amount))
}

// SetBalance sets the balance for the object, and returns the previous balance.
func (s *stateObject) SetBalance(amount *big.Int) *big.Int {
	prev := new(big.Int).Set(s.data.Balance)
	s.db.journal.append(balanceChange{
		account: s.address,
		prev:    new(big.Int).Set(s.data.Balance),
	})
	s.setBalance(amount)
	return prev
}

func (s *stateObject) setBalance(amount *big.Int) {
	s.data.Balance = amount
}

func (s *stateObject) deepCopy(db *StateDB) *stateObject {
	obj := &stateObject{
		db:       db,
		address:  s.address,
		addrHash: s.addrHash,
		origin:   s.origin,
		data:     s.data,
	}
	if s.trie != nil {
		obj.trie = db.db.CopyTrie(s.trie)
	}
	obj.code = s.code
	obj.originStorage = s.originStorage.Copy()
	obj.pendingStorage = s.pendingStorage.Copy()
	obj.dirtyStorage = s.dirtyStorage.Copy()
	obj.dirtyCode = s.dirtyCode
	obj.selfDestructed = s.selfDestructed
	obj.newContract = s.newContract
	return obj
}

//
// Attribute accessors
//

// Address returns the address of the contract/account
func (s *stateObject) Address() common.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code() []byte {
	if len(s.code) != 0 {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), types.EmptyCodeHash.Bytes()) {
		return nil
	}
	code, err := s.db.db.ContractCode(s.address, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.db.setError(fmt.Errorf("can't load code hash %x: %v", s.CodeHash(), err))
	}
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none. This method is an almost mirror of Code, but uses a cache
// inside the database to avoid loading codes seen recently.
func (s *stateObject) CodeSize() int {
	if len(s.code) != 0 {
		return len(s.code)
	}
	if bytes.Equal(s.CodeHash(), types.EmptyCodeHash.Bytes()) {
		return 0
	}
	size, err := s.db.db.ContractCodeSize(s.address, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.db.setError(fmt.Errorf("can't load code size %x: %v", s.CodeHash(), err))
	}
	return size
}

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) []byte {
	prevCode := slices.Clone(s.code)
	s.db.journal.setCode(s.address, prevCode)
	s.setCode(codeHash, code)
	return prevCode
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: s.address,
		prev:    s.data.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

func (s *stateObject) Balance() *big.Int {
	return s.data.Balance
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

func (s *stateObject) Root() common.Hash {
	return s.data.Root
}
