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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

// proofStateObject represents a proof commitment which is being modified.
//
// The usage pattern is as follows:
// - First you need to obtain a state object.
// - Next, call commit to return the changes.
type proofStateObject struct {
	db       *StateDB
	address  common.Address // address of ethereum account
	addrHash common.Hash    // hash of ethereum address of the account
	origin   *common.Hash   // Account original data without any change applied, nil means it was not existent
	data     common.Hash    // Account data with all mutations applied in the scope of block
}

// empty returns whether the proof data is empty.
func (s *proofStateObject) empty() bool {
	return s.data.Cmp(common.Hash{}) == 0
}

// newObject creates a state object.
func newProofStateObject(db *StateDB, address common.Address, data *common.Hash) *proofStateObject {
	if data == nil {
		data = &common.Hash{}
	}
	return &proofStateObject{
		db:       db,
		address:  address,
		addrHash: crypto.Keccak256Hash(address[:]),
	}
}

func (s *proofStateObject) touch() {
	s.db.proofJournal.touchChange(s.address)
}

func (s *proofStateObject) GetState(key common.Hash) common.Hash {
	s.db.ProofLoaded++

	start := time.Now()
	value, err := s.db.reader.Storage(s.address, key)
	if err != nil {
		s.db.setError(err)
		return common.Hash{}
	}
	s.db.ProofReads += time.Since(start)

	// Schedule the resolved storage slots for prefetching if it's enabled.
	if s.db.proofPrefetcher != nil {
		if err = s.db.proofPrefetcher.prefetch(s.addrHash, s.data, s.address, nil, []common.Hash{key}, true); err != nil {
			log.Error("Failed to prefetch storage slot", "addr", s.address, "key", key, "err", err)
		}
	}
	return value
}

// SetState updates a value in account storage.
// It returns the previous value
func (s *proofStateObject) SetState(key, value common.Hash) common.Hash {
	// If the new value is the same as old, don't set. Otherwise, track only the
	// dirty changes, supporting reverting all of it back to no change.
	prev := s.GetState(key)
	// New value is different, update and journal the change
	s.db.proofJournal.storageChange(s.address, key, prev, common.Hash{})
	s.setState(key, value)
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
			slotsToPrefetch = append(slotsToPrefetch, key) // Copy needed for closure
		}
		// Aggregate the dirty storage slots into the pending area. It might
		// be possible that the value of tracked slot here is same with the
		// one in originStorage (e.g. the slot was modified in tx_a and then
		// modified back in tx_b). We can't blindly remove it from pending
		// map as the dirty slot might have been committed already (before the
		// byzantium fork) and entry is necessary to modify the value back.
		s.pendingStorage[key] = value
	}
	if s.db.prefetcher != nil && len(slotsToPrefetch) > 0 && s.data.Root != types.EmptyRootHash {
		if err := s.db.prefetcher.prefetch(s.addrHash, s.data.Root, s.address, nil, slotsToPrefetch, false); err != nil {
			log.Error("Failed to prefetch slots", "addr", s.address, "slots", len(slotsToPrefetch), "err", err)
		}
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
	// Revoke the flag at the end of the transaction. It finalizes the status
	// of the newly-created object as it's no longer eligible for self-destruct
	// by EIP-6780. For non-newly-created objects, it's a no-op.
	s.newContract = false
}

// updateTrie is responsible for persisting cached storage changes into the
// object's storage trie. In case the storage trie is not yet loaded, this
// function will load the trie automatically. If any issues arise during the
// loading or updating of the trie, an error will be returned. Furthermore,
// this function will return the mutated storage trie, or nil if there is no
// storage change at all.
//
// It assumes all the dirty storage slots have been finalized before.
func (s *stateObject) updateTrie() (Trie, error) {
	// Short circuit if nothing was accessed, don't trigger a prefetcher warning
	if len(s.uncommittedStorage) == 0 {
		// Nothing was written, so we could stop early. Unless we have both reads
		// and witness collection enabled, in which case we need to fetch the trie.
		if s.db.witness == nil || len(s.originStorage) == 0 {
			return s.trie, nil
		}
	}
	// Retrieve a pretecher populated trie, or fall back to the database. This will
	// block until all prefetch tasks are done, which are needed for witnesses even
	// for unmodified state objects.
	tr := s.getPrefetchedTrie()
	if tr != nil {
		// Prefetcher returned a live trie, swap it out for the current one
		s.trie = tr
	} else {
		// Fetcher not running or empty trie, fallback to the database trie
		var err error
		tr, err = s.getTrie()
		if err != nil {
			s.db.setError(err)
			return nil, err
		}
	}
	// Short circuit if nothing changed, don't bother with hashing anything
	if len(s.uncommittedStorage) == 0 {
		return s.trie, nil
	}
	// Perform trie updates before deletions. This prevents resolution of unnecessary trie nodes
	// in circumstances similar to the following:
	//
	// Consider nodes `A` and `B` who share the same full node parent `P` and have no other siblings.
	// During the execution of a block:
	// - `A` is deleted,
	// - `C` is created, and also shares the parent `P`.
	// If the deletion is handled first, then `P` would be left with only one child, thus collapsed
	// into a shortnode. This requires `B` to be resolved from disk.
	// Whereas if the created node is handled first, then the collapse is avoided, and `B` is not resolved.
	var (
		deletions []common.Hash
		used      = make([]common.Hash, 0, len(s.uncommittedStorage))
	)
	for key, origin := range s.uncommittedStorage {
		// Skip noop changes, persist actual changes
		value, exist := s.pendingStorage[key]
		if value == origin {
			log.Error("Storage update was noop", "address", s.address, "slot", key)
			continue
		}
		if !exist {
			log.Error("Storage slot is not found in pending area", s.address, "slot", key)
			continue
		}
		if (value != common.Hash{}) {
			if err := tr.UpdateStorage(s.address, key[:], common.TrimLeftZeroes(value[:])); err != nil {
				s.db.setError(err)
				return nil, err
			}
			s.db.StorageUpdated.Add(1)
		} else {
			deletions = append(deletions, key)
		}
		// Cache the items for preloading
		used = append(used, key) // Copy needed for closure
	}
	for _, key := range deletions {
		if err := tr.DeleteStorage(s.address, key[:]); err != nil {
			s.db.setError(err)
			return nil, err
		}
		s.db.StorageDeleted.Add(1)
	}
	if s.db.prefetcher != nil {
		s.db.prefetcher.used(s.addrHash, s.data.Root, nil, used)
	}
	s.uncommittedStorage = make(Storage) // empties the commit markers
	return tr, nil
}

// updateRoot flushes all cached storage mutations to trie, recalculating the
// new storage trie root.
func (s *stateObject) updateRoot() {
	// Flush cached storage mutations into trie, short circuit if any error
	// is occurred or there is no change in the trie.
	tr, err := s.updateTrie()
	if err != nil || tr == nil {
		return
	}
	s.data.Root = tr.Hash()
}

// commitStorage overwrites the clean storage with the storage changes and
// fulfills the storage diffs into the given accountUpdate struct.
func (s *stateObject) commitStorage(op *accountUpdate) {
	var (
		buf    = crypto.NewKeccakState()
		encode = func(val common.Hash) []byte {
			if val == (common.Hash{}) {
				return nil
			}
			blob, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(val[:]))
			return blob
		}
	)
	for key, val := range s.pendingStorage {
		// Skip the noop storage changes, it might be possible the value
		// of tracked slot is same in originStorage and pendingStorage
		// map, e.g. the storage slot is modified in tx_a and then reset
		// back in tx_b.
		if val == s.originStorage[key] {
			continue
		}
		hash := crypto.HashData(buf, key[:])
		if op.storages == nil {
			op.storages = make(map[common.Hash][]byte)
		}
		op.storages[hash] = encode(val)
		if op.storagesOrigin == nil {
			op.storagesOrigin = make(map[common.Hash][]byte)
		}
		op.storagesOrigin[hash] = encode(s.originStorage[key])

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
func (s *stateObject) commit() (*accountUpdate, *trienode.NodeSet, error) {
	// commit the account metadata changes
	op := &accountUpdate{
		address: s.address,
		data:    types.SlimAccountRLP(s.data),
	}
	if s.origin != nil {
		op.origin = types.SlimAccountRLP(*s.origin)
	}
	// commit the contract code if it's modified
	if s.dirtyCode {
		op.code = &contractCode{
			hash: common.BytesToHash(s.CodeHash()),
			blob: s.code,
		}
		s.dirtyCode = false // reset the dirty flag
	}
	// Commit storage changes and the associated storage trie
	s.commitStorage(op)
	if len(op.storages) == 0 {
		// nothing changed, don't bother to commit the trie
		s.origin = s.data.Copy()
		return op, nil, nil
	}
	root, nodes := s.trie.Commit(false)
	s.data.Root = root
	s.origin = s.data.Copy()
	return op, nodes, nil
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
		addrHash:           s.addrHash,
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
	if s.trie != nil {
		obj.trie = mustCopyTrie(s.trie)
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

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	s.db.journal.setCode(s.address)
	s.setCode(codeHash, code)
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

func (s *stateObject) Root() common.Hash {
	return s.data.Root
}
