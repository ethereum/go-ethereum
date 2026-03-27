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
	"maps"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
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
	db          *StateDB
	address     common.Address // address of ethereum account
	addressHash *common.Hash   // hash of ethereum address of the account
	origin      *Account       // Account original data without any change applied, nil means it was not existent
	data        Account        // Account data with all mutations applied in the scope of block

	// Write caches.
	code []byte // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage entries that have been accessed within the current block
	dirtyStorage   Storage // Storage entries that have been modified within the current transaction
	pendingStorage Storage // Storage entries that have been modified within the current block

	// uncommittedStorage tracks a set of storage entries that have been modified
	// but not yet committed since the "last commit operation", along with their
	// original values before mutation.
	//
	// Specifically, the commit will be performed after each transaction before
	// the byzantium fork, therefore the map is already reset at the transaction
	// boundary; however post the byzantium fork, the commit will only be performed
	// at the end of block, this set essentially tracks all the modifications
	// made within the block.
	uncommittedStorage Storage

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
	return s.data.Nonce == 0 && s.data.Balance.IsZero() && bytes.Equal(s.data.CodeHash, types.EmptyCodeHash.Bytes())
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, acct *Account) *stateObject {
	origin := acct
	if acct == nil {
		acct = newEmptyAccount()
	}
	return &stateObject{
		db:                 db,
		address:            address,
		origin:             origin,
		data:               *acct,
		originStorage:      make(Storage),
		dirtyStorage:       make(Storage),
		pendingStorage:     make(Storage),
		uncommittedStorage: make(Storage),
	}
}

func (s *stateObject) addrHash() common.Hash {
	if s.addressHash == nil {
		h := crypto.Keccak256Hash(s.address[:])
		s.addressHash = &h
	}
	return *s.addressHash
}

func (s *stateObject) markSelfdestructed() {
	s.selfDestructed = true
}

func (s *stateObject) touch() {
	s.db.journal.touchChange(s.address)
}

// GetState retrieves a value associated with the given storage key.
func (s *stateObject) GetState(key common.Hash) common.Hash {
	value, _ := s.getState(key)
	return value
}

// getState retrieves a value associated with the given storage key, along with
// its original value.
func (s *stateObject) getState(key common.Hash) (common.Hash, common.Hash) {
	origin := s.GetCommittedState(key)
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value, origin
	}
	return origin, origin
}

// GetCommittedState retrieves the value associated with the specific key
// without any mutations caused in the current execution.
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
		// Invoke the reader regardless and discard the returned value.
		// The returned value may not be empty, as it could belong to a
		// self-destructed contract.
		//
		// The read operation is still essential for correctly building
		// the block-level access list.
		//
		// TODO(rjl493456442) the reader interface can be extended with
		// Touch, recording the read access without the actual disk load.
		_, err := s.db.reader.Storage(s.address, key)
		if err != nil {
			s.db.setError(err)
		}
		s.originStorage[key] = common.Hash{} // track the empty slot as origin value
		return common.Hash{}
	}
	start := time.Now()
	value, err := s.db.reader.Storage(s.address, key)
	if err != nil {
		s.db.setError(err)
		return common.Hash{}
	}
	s.db.StorageLoaded++
	s.db.StorageReads += time.Since(start)

	s.originStorage[key] = value

	// Schedule the resolved storage slots for prefetching if it's enabled.
	prefetch, ok := s.db.hasher.(Prefetcher)
	if ok {
		prefetch.PrefetchStorage(s.address, []common.Hash{key}, true)
	}
	return value
}

// SetState updates a value in account storage.
// It returns the previous value
func (s *stateObject) SetState(key, value common.Hash) common.Hash {
	// If the new value is the same as old, don't set. Otherwise, track only the
	// dirty changes, supporting reverting all of it back to no change.
	prev, origin := s.getState(key)
	if prev == value {
		return prev
	}
	// New value is different, update and journal the change
	s.db.journal.storageChange(s.address, key, prev, origin)
	s.setState(key, value, origin)
	return prev
}

// setState updates a value in account dirty storage. The dirtiness will be
// removed if the value being set equals to the original value.
func (s *stateObject) setState(key common.Hash, value common.Hash, origin common.Hash) {
	// Storage slot is set back to its original value, undo the dirty marker
	if value == origin {
		delete(s.dirtyStorage, key)
		return
	}
	s.dirtyStorage[key] = value
}

// finalise moves all dirty storage slots into the pending area to be hashed or
// committed later. It is invoked at the end of every transaction.
func (s *stateObject) finalise() {
	slotsToPrefetch := make([]common.Hash, 0, len(s.dirtyStorage))
	for key, value := range s.dirtyStorage {
		if origin, exist := s.uncommittedStorage[key]; exist && origin == value {
			// The slot is reverted to its original value, delete the entry
			// to avoid thrashing the data structures.
			delete(s.uncommittedStorage, key)
		} else if exist {
			// The slot is modified to another value and the slot has been
			// tracked for commit, do nothing here.
		} else {
			// The slot is different from its original value and hasn't been
			// tracked for commit yet.
			s.uncommittedStorage[key] = s.GetCommittedState(key)
			slotsToPrefetch = append(slotsToPrefetch, key)
		}
		// Aggregate the dirty storage slots into the pending area. It might
		// be possible that the value of tracked slot here is same with the
		// one in originStorage (e.g. the slot was modified in tx_a and then
		// modified back in tx_b). We can't blindly remove it from pending
		// map as the dirty slot might have been committed already (before the
		// byzantium fork) and entry is necessary to modify the value back.
		s.pendingStorage[key] = value
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
	// Revoke the flag at the end of the transaction. It finalizes the status
	// of the newly-created object as it's no longer eligible for self-destruct
	// by EIP-6780. For non-newly-created objects, it's a no-op.
	s.newContract = false

	// Schedule the resolved storage slots for prefetching if it's enabled.
	prefetch, ok := s.db.hasher.(Prefetcher)
	if ok {
		prefetch.PrefetchStorage(s.address, slotsToPrefetch, false)
	}
}

// updateTrie is responsible for persisting cached storage changes into the
// state hasher. It assumes all the dirty storage slots have been finalized
// before.
func (s *stateObject) updateTrie() error {
	// Short circuit if nothing was accessed
	if len(s.uncommittedStorage) == 0 {
		return nil
	}
	var (
		updates int64
		deletes int64
		keys    = make([]common.Hash, 0, len(s.uncommittedStorage))
		vals    = make([]common.Hash, 0, len(s.uncommittedStorage))
	)
	for key, origin := range s.uncommittedStorage {
		// Skip noop changes, persist actual changes
		value, exist := s.pendingStorage[key]
		if value == origin {
			log.Error("Storage update was noop", "address", s.address, "slot", key)
			continue
		}
		if !exist {
			log.Error("Storage slot is not found in pending area", "address", s.address, "slot", key)
			continue
		}
		if value == (common.Hash{}) {
			deletes += 1
		} else {
			updates += 1
		}
		keys = append(keys, key)
		vals = append(vals, value)
	}
	s.uncommittedStorage = make(Storage) // empties the commit markers
	s.db.StorageUpdated.Add(updates)
	s.db.StorageDeleted.Add(deletes)

	return s.db.hasher.UpdateStorage(s.address, keys, vals)
}

// commitStorage overwrites the clean storage with the storage changes and
// fulfills the storage diffs into the given accountUpdate struct.
func (s *stateObject) commitStorage(op *accountUpdate) {
	for key, val := range s.pendingStorage {
		// Skip the noop storage changes, it might be possible the value
		// of tracked slot is same in originStorage and pendingStorage
		// map, e.g. the storage slot is modified in tx_a and then reset
		// back in tx_b.
		if val == s.originStorage[key] {
			continue
		}
		hash := crypto.Keccak256Hash(key[:])
		if op.storages == nil {
			op.storages = make(map[common.Hash]common.Hash)
		}
		op.storages[hash] = val

		if op.storagesOriginByKey == nil {
			op.storagesOriginByKey = make(map[common.Hash]common.Hash)
		}
		if op.storagesOriginByHash == nil {
			op.storagesOriginByHash = make(map[common.Hash]common.Hash)
		}
		origin := s.originStorage[key]
		op.storagesOriginByKey[key] = origin
		op.storagesOriginByHash[hash] = origin

		// Overwrite the clean value of storage slots
		s.originStorage[key] = val
	}
	s.pendingStorage = make(Storage)
}

// commit obtains the account changes (metadata, storage slots, code) caused by
// state execution along with the dirty storage trie nodes.
//
// Note, commit may run concurrently across all the state objects. Do not assume
// thread-safe access to the statedb.
func (s *stateObject) commit() (*accountUpdate, error) {
	// commit the account metadata changes
	op := &accountUpdate{
		address: s.address,
		data:    &s.data,
		origin:  s.origin,
	}
	// commit the contract code if it's modified
	if s.dirtyCode {
		s.dirtyCode = false // reset the dirty flag

		op.code = &contractCode{
			hash: common.BytesToHash(s.CodeHash()),
			blob: s.code,
		}
		if s.origin == nil {
			op.code.originHash = types.EmptyCodeHash
		} else {
			op.code.originHash = common.BytesToHash(s.origin.CodeHash)
		}
	}
	// Commit storage changes and the associated storage trie
	s.commitStorage(op)
	s.origin = s.data.copy()
	return op, nil
}

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
// returns the previous balance
func (s *stateObject) AddBalance(amount *uint256.Int) uint256.Int {
	// EIP161: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.IsZero() {
		if s.empty() {
			s.touch()
		}
		return *(s.Balance())
	}
	return s.SetBalance(new(uint256.Int).Add(s.Balance(), amount))
}

// SetBalance sets the balance for the object, and returns the previous balance.
func (s *stateObject) SetBalance(amount *uint256.Int) uint256.Int {
	prev := *s.data.Balance
	s.db.journal.balanceChange(s.address, s.data.Balance)
	s.setBalance(amount)
	return prev
}

func (s *stateObject) setBalance(amount *uint256.Int) {
	s.data.Balance = amount
}

func (s *stateObject) deepCopy(db *StateDB) *stateObject {
	obj := &stateObject{
		db:                 db,
		address:            s.address,
		addressHash:        nil,
		origin:             s.origin,
		data:               s.data,
		code:               s.code,
		originStorage:      s.originStorage.Copy(),
		pendingStorage:     s.pendingStorage.Copy(),
		dirtyStorage:       s.dirtyStorage.Copy(),
		uncommittedStorage: s.uncommittedStorage.Copy(),
		dirtyCode:          s.dirtyCode,
		selfDestructed:     s.selfDestructed,
		newContract:        s.newContract,
	}
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
	defer func(start time.Time) {
		s.db.CodeLoaded += 1
		s.db.CodeReads += time.Since(start)
		s.db.CodeLoadBytes += len(s.code)
	}(time.Now())

	code := s.db.reader.Code(s.address, common.BytesToHash(s.CodeHash()))
	if len(code) == 0 {
		s.db.setError(fmt.Errorf("code is not found %x", s.CodeHash()))
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
	defer func(start time.Time) {
		s.db.CodeLoaded += 1
		s.db.CodeReads += time.Since(start)
	}(time.Now())

	size := s.db.reader.CodeSize(s.address, common.BytesToHash(s.CodeHash()))
	if size == 0 {
		s.db.setError(fmt.Errorf("code is not found %x", s.CodeHash()))
	}
	return size
}

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) (prev []byte) {
	prev = slices.Clone(s.code)
	s.db.journal.setCode(s.address, prev)
	s.setCode(codeHash, code)
	return prev
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.nonceChange(s.address, s.data.Nonce)
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

func (s *stateObject) Balance() *uint256.Int {
	return s.data.Balance
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}
