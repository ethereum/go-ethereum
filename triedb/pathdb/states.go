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

package pathdb

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

// counter helps in tracking items and their corresponding sizes.
type counter struct {
	n    int
	size int
}

// add size to the counter and increase the item counter.
func (c *counter) add(size int) {
	c.n++
	c.size += size
}

// report uploads the cached statistics to meters.
func (c *counter) report(count, size *metrics.Meter) {
	count.Mark(int64(c.n))
	size.Mark(int64(c.size))
}

// stateSet represents a collection of state modifications associated with a
// transition (e.g., a block execution) or multiple aggregated transitions.
//
// A stateSet can only reside within a diffLayer or the buffer of a diskLayer,
// serving as the envelope for the set. Lock protection is not required for
// accessing or mutating the account set and storage set, as the associated
// envelope is always marked as stale before any mutation is applied. Any
// subsequent state access will be denied due to the stale flag. Therefore,
// state access and mutation won't happen at the same time with guarantee.
type stateSet struct {
	accountData map[common.Hash][]byte                 // Keyed accounts for direct retrieval (nil means deleted)
	storageData map[common.Hash]map[common.Hash][]byte // Keyed storage slots for direct retrieval. one per account (nil means deleted)
	size        uint64                                 // Memory size of the state data (accountData and storageData)

	accountListSorted []common.Hash                 // List of account for iteration. If it exists, it's sorted, otherwise it's nil
	storageListSorted map[common.Hash][]common.Hash // List of storage slots for iterated retrievals, one per account. Any existing lists are sorted if non-nil

	rawStorageKey bool // indicates whether the storage set uses the raw slot key or the hash

	// Lock for guarding the two lists above. These lists might be accessed
	// concurrently and lock protection is essential to avoid concurrent
	// slice or map read/write.
	listLock sync.RWMutex
}

// newStates constructs the state set with the provided account and storage data.
func newStates(accounts map[common.Hash][]byte, storages map[common.Hash]map[common.Hash][]byte, rawStorageKey bool) *stateSet {
	// Don't panic for the lazy callers, initialize the nil maps instead.
	if accounts == nil {
		accounts = make(map[common.Hash][]byte)
	}
	if storages == nil {
		storages = make(map[common.Hash]map[common.Hash][]byte)
	}
	s := &stateSet{
		accountData:       accounts,
		storageData:       storages,
		rawStorageKey:     rawStorageKey,
		storageListSorted: make(map[common.Hash][]common.Hash),
	}
	s.size = s.check()
	return s
}

// account returns the account data associated with the specified address hash.
func (s *stateSet) account(hash common.Hash) ([]byte, bool) {
	// If the account is known locally, return it
	if data, ok := s.accountData[hash]; ok {
		return data, true
	}
	return nil, false // account is unknown in this set
}

// mustAccount returns the account data associated with the specified address
// hash. The difference is this function will return an error if the account
// is not found.
func (s *stateSet) mustAccount(hash common.Hash) ([]byte, error) {
	// If the account is known locally, return it
	if data, ok := s.accountData[hash]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("account is not found, %x", hash)
}

// storage returns the storage slot associated with the specified address hash
// and storage key hash.
func (s *stateSet) storage(accountHash, storageHash common.Hash) ([]byte, bool) {
	// If the account is known locally, try to resolve the slot locally
	if storage, ok := s.storageData[accountHash]; ok {
		if data, ok := storage[storageHash]; ok {
			return data, true
		}
	}
	return nil, false // storage is unknown in this set
}

// mustStorage returns the storage slot associated with the specified address
// hash and storage key hash. The difference is this function will return an
// error if the storage slot is not found.
func (s *stateSet) mustStorage(accountHash, storageHash common.Hash) ([]byte, error) {
	// If the account is known locally, try to resolve the slot locally
	if storage, ok := s.storageData[accountHash]; ok {
		if data, ok := storage[storageHash]; ok {
			return data, nil
		}
	}
	return nil, fmt.Errorf("storage slot is not found, %x %x", accountHash, storageHash)
}

// check sanitizes accounts and storage slots to ensure the data validity.
// Additionally, it computes the total memory size occupied by the maps.
func (s *stateSet) check() uint64 {
	var size int
	for _, blob := range s.accountData {
		size += common.HashLength + len(blob)
	}
	for accountHash, slots := range s.storageData {
		if slots == nil {
			panic(fmt.Sprintf("storage %#x nil", accountHash)) // nil slots is not permitted
		}
		for _, blob := range slots {
			size += 2*common.HashLength + len(blob)
		}
	}
	return uint64(size)
}

// accountList returns a sorted list of all accounts in this state set, including
// the deleted ones.
//
// Note, the returned slice is not a copy, so do not modify it.
func (s *stateSet) accountList() []common.Hash {
	// If an old list already exists, return it
	s.listLock.RLock()
	list := s.accountListSorted
	s.listLock.RUnlock()

	if list != nil {
		return list
	}
	// No old sorted account list exists, generate a new one. It's possible that
	// multiple threads waiting for the write lock may regenerate the list
	// multiple times, which is acceptable.
	s.listLock.Lock()
	defer s.listLock.Unlock()

	list = slices.SortedFunc(maps.Keys(s.accountData), common.Hash.Cmp)
	s.accountListSorted = list
	return list
}

// StorageList returns a sorted list of all storage slot hashes in this state set
// for the given account. The returned list will include the hash of deleted
// storage slot.
//
// Note, the returned slice is not a copy, so do not modify it.
func (s *stateSet) storageList(accountHash common.Hash) []common.Hash {
	s.listLock.RLock()
	if _, ok := s.storageData[accountHash]; !ok {
		// Account not tracked by this layer
		s.listLock.RUnlock()
		return nil
	}
	// If an old list already exists, return it
	if list, exist := s.storageListSorted[accountHash]; exist {
		s.listLock.RUnlock()
		return list // the cached list can't be nil
	}
	s.listLock.RUnlock()

	// No old sorted account list exists, generate a new one. It's possible that
	// multiple threads waiting for the write lock may regenerate the list
	// multiple times, which is acceptable.
	s.listLock.Lock()
	defer s.listLock.Unlock()

	list := slices.SortedFunc(maps.Keys(s.storageData[accountHash]), common.Hash.Cmp)
	s.storageListSorted[accountHash] = list
	return list
}

// clearLists invalidates the cached account list and storage lists.
func (s *stateSet) clearLists() {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	s.accountListSorted = nil
	s.storageListSorted = make(map[common.Hash][]common.Hash)
}

// merge integrates the accounts and storages from the external set into the
// local set, ensuring the combined set reflects the combined state of both.
//
// The stateSet supplied as parameter set will not be mutated by this operation,
// as it may still be referenced by other layers.
func (s *stateSet) merge(other *stateSet) {
	var (
		delta             int
		accountOverwrites counter
		storageOverwrites counter
	)
	// Apply the updated account data
	for accountHash, data := range other.accountData {
		if origin, ok := s.accountData[accountHash]; ok {
			delta += len(data) - len(origin)
			accountOverwrites.add(common.HashLength + len(origin))
		} else {
			delta += common.HashLength + len(data)
		}
		s.accountData[accountHash] = data
	}
	// Apply all the updated storage slots (individually)
	for accountHash, storage := range other.storageData {
		// If storage didn't exist in the set, overwrite blindly
		if _, ok := s.storageData[accountHash]; !ok {
			// To prevent potential concurrent map read/write issues, allocate a
			// new map for the storage instead of claiming it directly from the
			// passed external set. Even after merging, the slots belonging to the
			// external state set remain accessible, so ownership of the map should
			// not be taken, and any mutation on it should be avoided.
			slots := make(map[common.Hash][]byte, len(storage))
			for storageHash, data := range storage {
				slots[storageHash] = data
				delta += 2*common.HashLength + len(data)
			}
			s.storageData[accountHash] = slots
			continue
		}
		// Storage exists in both local and external set, merge the slots
		slots := s.storageData[accountHash]
		for storageHash, data := range storage {
			if origin, ok := slots[storageHash]; ok {
				delta += len(data) - len(origin)
				storageOverwrites.add(2*common.HashLength + len(origin))
			} else {
				delta += 2*common.HashLength + len(data)
			}
			slots[storageHash] = data
		}
	}
	accountOverwrites.report(gcAccountMeter, gcAccountBytesMeter)
	storageOverwrites.report(gcStorageMeter, gcStorageBytesMeter)
	s.clearLists()
	s.updateSize(delta)
}

// revertTo takes the original value of accounts and storages as input and reverts
// the latest state transition applied on the state set.
//
// Notably, this operation may result in the set containing more entries after a
// revert. For example, if account x did not exist and was created during transition
// w, reverting w will retain an x=nil entry in the set. And also if account x along
// with its storage slots was deleted in the transition w, reverting w will retain
// a list of additional storage slots with their original value.
func (s *stateSet) revertTo(accountOrigin map[common.Hash][]byte, storageOrigin map[common.Hash]map[common.Hash][]byte) {
	var delta int // size tracking
	for addrHash, blob := range accountOrigin {
		data, ok := s.accountData[addrHash]
		if !ok {
			panic(fmt.Sprintf("non-existent account for reverting, %x", addrHash))
		}
		if len(data) == 0 && len(blob) == 0 {
			panic(fmt.Sprintf("invalid account mutation (null to null), %x", addrHash))
		}
		delta += len(blob) - len(data)
		s.accountData[addrHash] = blob
	}
	// Overwrite the storage data with original value blindly
	for addrHash, storage := range storageOrigin {
		slots := s.storageData[addrHash]
		if len(slots) == 0 {
			panic(fmt.Sprintf("non-existent storage set for reverting, %x", addrHash))
		}
		for storageHash, blob := range storage {
			data, ok := slots[storageHash]
			if !ok {
				panic(fmt.Sprintf("non-existent storage slot for reverting, %x-%x", addrHash, storageHash))
			}
			if len(blob) == 0 && len(data) == 0 {
				panic(fmt.Sprintf("invalid storage slot mutation (null to null), %x-%x", addrHash, storageHash))
			}
			delta += len(blob) - len(data)
			slots[storageHash] = blob
		}
	}
	s.clearLists()
	s.updateSize(delta)
}

// updateSize updates the total cache size by the given delta.
func (s *stateSet) updateSize(delta int) {
	size := int64(s.size) + int64(delta)
	if size >= 0 {
		s.size = uint64(size)
		return
	}
	log.Error("Stateset size underflow", "prev", common.StorageSize(s.size), "delta", common.StorageSize(delta))
	s.size = 0
}

// encode serializes the content of state set into the provided writer.
func (s *stateSet) encode(w io.Writer) error {
	// Encode accounts
	if err := rlp.Encode(w, s.rawStorageKey); err != nil {
		return err
	}
	type accounts struct {
		AddrHashes []common.Hash
		Accounts   [][]byte
	}
	var enc accounts
	for addrHash, blob := range s.accountData {
		enc.AddrHashes = append(enc.AddrHashes, addrHash)
		enc.Accounts = append(enc.Accounts, blob)
	}
	if err := rlp.Encode(w, enc); err != nil {
		return err
	}
	// Encode storages
	type Storage struct {
		AddrHash common.Hash
		Keys     []common.Hash
		Vals     [][]byte
	}
	storages := make([]Storage, 0, len(s.storageData))
	for addrHash, slots := range s.storageData {
		keys := make([]common.Hash, 0, len(slots))
		vals := make([][]byte, 0, len(slots))
		for key, val := range slots {
			keys = append(keys, key)
			vals = append(vals, val)
		}
		storages = append(storages, Storage{
			AddrHash: addrHash,
			Keys:     keys,
			Vals:     vals,
		})
	}
	return rlp.Encode(w, storages)
}

// decode deserializes the content from the rlp stream into the state set.
func (s *stateSet) decode(r *rlp.Stream) error {
	if err := r.Decode(&s.rawStorageKey); err != nil {
		return fmt.Errorf("load diff raw storage key flag: %v", err)
	}
	type accounts struct {
		AddrHashes []common.Hash
		Accounts   [][]byte
	}
	var (
		dec        accounts
		accountSet = make(map[common.Hash][]byte)
	)
	if err := r.Decode(&dec); err != nil {
		return fmt.Errorf("load diff accounts: %v", err)
	}
	for i := 0; i < len(dec.AddrHashes); i++ {
		accountSet[dec.AddrHashes[i]] = dec.Accounts[i]
	}
	s.accountData = accountSet

	// Decode storages
	type storage struct {
		AddrHash common.Hash
		Keys     []common.Hash
		Vals     [][]byte
	}
	var (
		storages   []storage
		storageSet = make(map[common.Hash]map[common.Hash][]byte)
	)
	if err := r.Decode(&storages); err != nil {
		return fmt.Errorf("load diff storage: %v", err)
	}
	for _, entry := range storages {
		storageSet[entry.AddrHash] = make(map[common.Hash][]byte, len(entry.Keys))
		for i := 0; i < len(entry.Keys); i++ {
			storageSet[entry.AddrHash][entry.Keys[i]] = entry.Vals[i]
		}
	}
	s.storageData = storageSet
	s.storageListSorted = make(map[common.Hash][]common.Hash)

	s.size = s.check()
	return nil
}

// reset clears all cached state data, including any optional sorted lists that
// may have been generated.
func (s *stateSet) reset() {
	s.accountData = make(map[common.Hash][]byte)
	s.storageData = make(map[common.Hash]map[common.Hash][]byte)
	s.size = 0
	s.accountListSorted = nil
	s.storageListSorted = make(map[common.Hash][]common.Hash)
}

// dbsize returns the approximate size for db write.
//
// nolint:unused
func (s *stateSet) dbsize() int {
	m := len(s.accountData) * len(rawdb.SnapshotAccountPrefix)
	for _, slots := range s.storageData {
		m += len(slots) * len(rawdb.SnapshotStoragePrefix)
	}
	return m + int(s.size)
}

// StateSetWithOrigin wraps the state set with additional original values of the
// mutated states.
type StateSetWithOrigin struct {
	*stateSet

	// accountOrigin represents the account data before the state transition,
	// corresponding to both the accountData and destructSet. It's keyed by the
	// account address. The nil value means the account was not present before.
	accountOrigin map[common.Address][]byte

	// storageOrigin represents the storage data before the state transition,
	// corresponding to storageData and deleted slots of destructSet. It's keyed
	// by the account address and slot key hash. The nil value means the slot was
	// not present.
	storageOrigin map[common.Address]map[common.Hash][]byte

	// memory size of the state data (accountOrigin and storageOrigin)
	size uint64
}

// NewStateSetWithOrigin constructs the state set with the provided data.
func NewStateSetWithOrigin(accounts map[common.Hash][]byte, storages map[common.Hash]map[common.Hash][]byte, accountOrigin map[common.Address][]byte, storageOrigin map[common.Address]map[common.Hash][]byte, rawStorageKey bool) *StateSetWithOrigin {
	// Don't panic for the lazy callers, initialize the nil maps instead.
	if accountOrigin == nil {
		accountOrigin = make(map[common.Address][]byte)
	}
	if storageOrigin == nil {
		storageOrigin = make(map[common.Address]map[common.Hash][]byte)
	}
	// Count the memory size occupied by the set. Note that each slot key here
	// uses 2*common.HashLength to keep consistent with the calculation method
	// of stateSet.
	var size int
	for _, data := range accountOrigin {
		size += common.HashLength + len(data)
	}
	for _, slots := range storageOrigin {
		for _, data := range slots {
			size += 2*common.HashLength + len(data)
		}
	}
	set := newStates(accounts, storages, rawStorageKey)
	return &StateSetWithOrigin{
		stateSet:      set,
		accountOrigin: accountOrigin,
		storageOrigin: storageOrigin,
		size:          set.size + uint64(size),
	}
}

// encode serializes the content of state set into the provided writer.
func (s *StateSetWithOrigin) encode(w io.Writer) error {
	// Encode state set
	if err := s.stateSet.encode(w); err != nil {
		return err
	}
	// Encode accounts
	type Accounts struct {
		Addresses []common.Address
		Accounts  [][]byte
	}
	var accounts Accounts
	for address, blob := range s.accountOrigin {
		accounts.Addresses = append(accounts.Addresses, address)
		accounts.Accounts = append(accounts.Accounts, blob)
	}
	if err := rlp.Encode(w, accounts); err != nil {
		return err
	}
	// Encode storages
	type Storage struct {
		Address common.Address
		Keys    []common.Hash
		Vals    [][]byte
	}
	storages := make([]Storage, 0, len(s.storageOrigin))
	for address, slots := range s.storageOrigin {
		keys := make([]common.Hash, 0, len(slots))
		vals := make([][]byte, 0, len(slots))
		for key, val := range slots {
			keys = append(keys, key)
			vals = append(vals, val)
		}
		storages = append(storages, Storage{Address: address, Keys: keys, Vals: vals})
	}
	return rlp.Encode(w, storages)
}

// decode deserializes the content from the rlp stream into the state set.
func (s *StateSetWithOrigin) decode(r *rlp.Stream) error {
	if s.stateSet == nil {
		s.stateSet = &stateSet{}
	}
	if err := s.stateSet.decode(r); err != nil {
		return err
	}
	// Decode account origin
	type Accounts struct {
		Addresses []common.Address
		Accounts  [][]byte
	}
	var (
		accounts   Accounts
		accountSet = make(map[common.Address][]byte)
	)
	if err := r.Decode(&accounts); err != nil {
		return fmt.Errorf("load diff account origin set: %v", err)
	}
	for i := 0; i < len(accounts.Accounts); i++ {
		accountSet[accounts.Addresses[i]] = accounts.Accounts[i]
	}
	s.accountOrigin = accountSet

	// Decode storage origin
	type Storage struct {
		Address common.Address
		Keys    []common.Hash
		Vals    [][]byte
	}
	var (
		storages   []Storage
		storageSet = make(map[common.Address]map[common.Hash][]byte)
	)
	if err := r.Decode(&storages); err != nil {
		return fmt.Errorf("load diff storage origin: %v", err)
	}
	for _, storage := range storages {
		storageSet[storage.Address] = make(map[common.Hash][]byte)
		for i := 0; i < len(storage.Keys); i++ {
			storageSet[storage.Address][storage.Keys[i]] = storage.Vals[i]
		}
	}
	s.storageOrigin = storageSet
	return nil
}
