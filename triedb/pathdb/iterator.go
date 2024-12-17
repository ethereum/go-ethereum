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
	"bytes"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Iterator is an iterator to step over all the accounts or the specific
// storage in a snapshot which may or may not be composed of multiple layers.
type Iterator interface {
	// Next steps the iterator forward one element, returning false if exhausted,
	// or an error if iteration failed for some reason (e.g. root being iterated
	// becomes stale and garbage collected).
	Next() bool

	// Error returns any failure that occurred during iteration, which might have
	// caused a premature iteration exit (e.g. layer stack becoming stale).
	Error() error

	// Hash returns the hash of the account or storage slot the iterator is
	// currently at.
	Hash() common.Hash

	// Release releases associated resources. Release should always succeed and
	// can be called multiple times without causing error.
	Release()
}

// AccountIterator is an iterator to step over all the accounts in a snapshot,
// which may or may not be composed of multiple layers.
type AccountIterator interface {
	Iterator

	// Account returns the RLP encoded slim account the iterator is currently at.
	// An error will be returned if the iterator becomes invalid
	Account() []byte
}

// StorageIterator is an iterator to step over the specific storage in a snapshot,
// which may or may not be composed of multiple layers.
type StorageIterator interface {
	Iterator

	// Slot returns the storage slot the iterator is currently at. An error will
	// be returned if the iterator becomes invalid
	Slot() []byte
}

type (
	// loadAccount is the function to retrieve the account from the associated
	// layer. An error will be returned if the associated layer is stale.
	loadAccount func(hash common.Hash) ([]byte, error)

	// loadStorage is the function to retrieve the storage slot from the associated
	// layer. An error will be returned if the associated layer is stale.
	loadStorage func(addrHash common.Hash, slotHash common.Hash) ([]byte, error)
)

// diffAccountIterator is an account iterator that steps over the accounts (both
// live and deleted) contained within a state set. Higher order iterators will
// use the deleted accounts to skip deeper iterators.
//
// This iterator could be created from the diff layer or the disk layer (the
// aggregated state buffer).
type diffAccountIterator struct {
	curHash common.Hash   // The current hash the iterator is positioned on
	keys    []common.Hash // Keys left in the layer to iterate
	fail    error         // Any failures encountered (stale)
	loadFn  loadAccount   // Function to retrieve the account from with supplied hash
}

// newDiffAccountIterator creates an account iterator over the given state set.
func newDiffAccountIterator(seek common.Hash, states *stateSet, fn loadAccount) AccountIterator {
	// Seek out the requested starting account
	hashes := states.accountList()
	index := sort.Search(len(hashes), func(i int) bool {
		return bytes.Compare(seek[:], hashes[i][:]) <= 0
	})
	// Assemble and returned the already seeked iterator
	return &diffAccountIterator{
		keys:   hashes[index:],
		loadFn: fn,
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diffAccountIterator) Next() bool {
	// If the iterator was already stale, consider it a programmer error. Although
	// we could just return false here, triggering this path would probably mean
	// somebody forgot to check for Error, so lets blow up instead of undefined
	// behavior that's hard to debug.
	if it.fail != nil {
		panic(fmt.Sprintf("called Next of failed iterator: %v", it.fail))
	}
	// Stop iterating if all keys were exhausted
	if len(it.keys) == 0 {
		return false
	}
	// Iterator seems to be still alive, retrieve and cache the live hash
	it.curHash = it.keys[0]

	// key cached, shift the iterator and notify the user of success
	it.keys = it.keys[1:]
	return true
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. the linked state set becoming stale).
func (it *diffAccountIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the account the iterator is currently at.
func (it *diffAccountIterator) Hash() common.Hash {
	return it.curHash
}

// Account returns the RLP encoded slim account the iterator is currently at.
// This method may fail if the associated state goes stale. An error will
// be set to it.fail just in case.
//
// Note the returned account is not a copy, please don't modify it.
func (it *diffAccountIterator) Account() []byte {
	blob, err := it.loadFn(it.curHash)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}

// Release is a noop for diff account iterators as there are no held resources.
func (it *diffAccountIterator) Release() {}

// diskAccountIterator is an account iterator that steps over the persistent
// accounts within the database.
//
// To simplify, the staleness of the persistent state is not tracked. The disk
// iterator is not intended to be used alone. It should always be wrapped with
// a diff iterator, as the bottom-most disk layer uses both the in-memory
// aggregated buffer and the persistent disk layer as the data sources. The
// staleness of the diff iterator is sufficient to invalidate the iterator pair.
type diskAccountIterator struct {
	it ethdb.Iterator
}

// newDiskAccountIterator creates an account iterator over the persistent state.
func newDiskAccountIterator(db ethdb.KeyValueStore, seek common.Hash) AccountIterator {
	pos := common.TrimRightZeroes(seek[:])
	return &diskAccountIterator{
		it: db.NewIterator(rawdb.SnapshotAccountPrefix, pos),
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diskAccountIterator) Next() bool {
	// If the iterator was already exhausted, don't bother
	if it.it == nil {
		return false
	}
	// Try to advance the iterator and release it if we reached the end
	for {
		if !it.it.Next() {
			it.it.Release()
			it.it = nil
			return false
		}
		if len(it.it.Key()) == len(rawdb.SnapshotAccountPrefix)+common.HashLength {
			break
		}
	}
	return true
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit. (e.g, any error occurred in the database)
func (it *diskAccountIterator) Error() error {
	if it.it == nil {
		return nil // Iterator is exhausted and released
	}
	return it.it.Error()
}

// Hash returns the hash of the account the iterator is currently at.
func (it *diskAccountIterator) Hash() common.Hash {
	return common.BytesToHash(it.it.Key()) // The prefix will be truncated
}

// Account returns the RLP encoded slim account the iterator is currently at.
func (it *diskAccountIterator) Account() []byte {
	return it.it.Value()
}

// Release releases the database snapshot held during iteration.
func (it *diskAccountIterator) Release() {
	// The iterator is auto-released on exhaustion, so make sure it's still alive
	if it.it != nil {
		it.it.Release()
		it.it = nil
	}
}

// diffStorageIterator is a storage iterator that steps over the specific storage
// (both live and deleted) contained within a state set. Higher order iterators
// will use the deleted slot to skip deeper iterators.
//
// This iterator could be created from the diff layer or the disk layer (the
// aggregated state buffer).
type diffStorageIterator struct {
	curHash common.Hash   // The current slot hash the iterator is positioned on
	account common.Hash   // The account hash the storage slots belonging to
	keys    []common.Hash // Keys left in the layer to iterate
	fail    error         // Any failures encountered (stale)
	loadFn  loadStorage   // Function to retrieve the storage slot from with supplied hash
}

// newDiffStorageIterator creates a storage iterator over a single diff layer.
func newDiffStorageIterator(account common.Hash, seek common.Hash, states *stateSet, fn loadStorage) StorageIterator {
	hashes := states.storageList(account)
	index := sort.Search(len(hashes), func(i int) bool {
		return bytes.Compare(seek[:], hashes[i][:]) <= 0
	})
	// Assemble and returned the already seeked iterator
	return &diffStorageIterator{
		account: account,
		keys:    hashes[index:],
		loadFn:  fn,
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diffStorageIterator) Next() bool {
	// If the iterator was already stale, consider it a programmer error. Although
	// we could just return false here, triggering this path would probably mean
	// somebody forgot to check for Error, so lets blow up instead of undefined
	// behavior that's hard to debug.
	if it.fail != nil {
		panic(fmt.Sprintf("called Next of failed iterator: %v", it.fail))
	}
	// Stop iterating if all keys were exhausted
	if len(it.keys) == 0 {
		return false
	}
	// Iterator seems to be still alive, retrieve and cache the live hash
	it.curHash = it.keys[0]

	// key cached, shift the iterator and notify the user of success
	it.keys = it.keys[1:]
	return true
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. the state set becoming stale).
func (it *diffStorageIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the storage slot the iterator is currently at.
func (it *diffStorageIterator) Hash() common.Hash {
	return it.curHash
}

// Slot returns the raw storage slot value the iterator is currently at.
// This method may fail if the associated state goes stale. An error will
// be set to it.fail just in case.
//
// Note the returned slot is not a copy, please don't modify it.
func (it *diffStorageIterator) Slot() []byte {
	storage, err := it.loadFn(it.account, it.curHash)
	if err != nil {
		it.fail = err
		return nil
	}
	return storage
}

// Release is a noop for diff account iterators as there are no held resources.
func (it *diffStorageIterator) Release() {}

// diskStorageIterator is a storage iterator that steps over the persistent
// storage slots contained within the database.
//
// To simplify, the staleness of the persistent state is not tracked. The disk
// iterator is not intended to be used alone. It should always be wrapped with
// a diff iterator, as the bottom-most disk layer uses both the in-memory
// aggregated buffer and the persistent disk layer as the data sources. The
// staleness of the diff iterator is sufficient to invalidate the iterator pair.
type diskStorageIterator struct {
	account common.Hash
	it      ethdb.Iterator
}

// StorageIterator creates a storage iterator over the persistent state.
func newDiskStorageIterator(db ethdb.KeyValueStore, account common.Hash, seek common.Hash) StorageIterator {
	pos := common.TrimRightZeroes(seek[:])
	return &diskStorageIterator{
		account: account,
		it:      db.NewIterator(append(rawdb.SnapshotStoragePrefix, account.Bytes()...), pos),
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diskStorageIterator) Next() bool {
	// If the iterator was already exhausted, don't bother
	if it.it == nil {
		return false
	}
	// Try to advance the iterator and release it if we reached the end
	for {
		if !it.it.Next() {
			it.it.Release()
			it.it = nil
			return false
		}
		if len(it.it.Key()) == len(rawdb.SnapshotStoragePrefix)+common.HashLength+common.HashLength {
			break
		}
	}
	return true
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. the error occurred in the database).
func (it *diskStorageIterator) Error() error {
	if it.it == nil {
		return nil // Iterator is exhausted and released
	}
	return it.it.Error()
}

// Hash returns the hash of the storage slot the iterator is currently at.
func (it *diskStorageIterator) Hash() common.Hash {
	return common.BytesToHash(it.it.Key()) // The prefix will be truncated
}

// Slot returns the raw storage slot content the iterator is currently at.
func (it *diskStorageIterator) Slot() []byte {
	return it.it.Value()
}

// Release releases the database snapshot held during iteration.
func (it *diskStorageIterator) Release() {
	// The iterator is auto-released on exhaustion, so make sure it's still alive
	if it.it != nil {
		it.it.Release()
		it.it = nil
	}
}
