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

	"github.com/ethereum/go-ethereum/common"
)

// binaryIterator is a simplistic iterator to step over the accounts or storage
// in a snapshot, which may or may not be composed of multiple layers. Performance
// wise this iterator is slow, it's meant for cross validating the fast one.
//
// This iterator cannot be used on its own; it should be wrapped with an outer
// iterator, such as accountBinaryIterator or storageBinaryIterator.
//
// This iterator can only traverse the keys of the entries stored in the layers,
// but cannot obtain the corresponding values. Besides, the deleted entry will
// also be traversed, the outer iterator must check the emptiness before returning.
type binaryIterator struct {
	a     Iterator
	b     Iterator
	aDone bool
	bDone bool
	k     common.Hash
	fail  error
}

// initBinaryAccountIterator creates a simplistic iterator to step over all the
// accounts in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryAccountIterator` as the API.
func (dl *diskLayer) initBinaryAccountIterator(seek common.Hash) *binaryIterator {
	// Create two iterators for state buffer and the persistent state in disk
	// respectively and combine them as a binary iterator.
	l := &binaryIterator{
		// The account loader function is unnecessary; the account key list
		// produced by the supplied buffer alone is sufficient for iteration.
		//
		// The account key list for iteration is deterministic once the iterator
		// is constructed, no matter the referenced disk layer is stale or not
		// later.
		a: newDiffAccountIterator(seek, dl.buffer.states, nil),
		b: newDiskAccountIterator(dl.db.diskdb, seek),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryAccountIterator creates a simplistic iterator to step over all the
// accounts in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryAccountIterator` as the API.
func (dl *diffLayer) initBinaryAccountIterator(seek common.Hash) *binaryIterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		l := &binaryIterator{
			// The account loader function is unnecessary; the account key list
			// produced by the supplied state set alone is sufficient for iteration.
			//
			// The account key list for iteration is deterministic once the iterator
			// is constructed, no matter the referenced disk layer is stale or not
			// later.
			a: newDiffAccountIterator(seek, dl.states.stateSet, nil),
			b: dl.parent.(*diskLayer).initBinaryAccountIterator(seek),
		}
		l.aDone = !l.a.Next()
		l.bDone = !l.b.Next()
		return l
	}
	l := &binaryIterator{
		// The account loader function is unnecessary; the account key list
		// produced by the supplied state set alone is sufficient for iteration.
		//
		// The account key list for iteration is deterministic once the iterator
		// is constructed, no matter the referenced disk layer is stale or not
		// later.
		a: newDiffAccountIterator(seek, dl.states.stateSet, nil),
		b: parent.initBinaryAccountIterator(seek),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryStorageIterator creates a simplistic iterator to step over all the
// storage slots in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryStorageIterator` as the API.
func (dl *diskLayer) initBinaryStorageIterator(account common.Hash, seek common.Hash) *binaryIterator {
	// Create two iterators for state buffer and the persistent state in disk
	// respectively and combine them as a binary iterator.
	l := &binaryIterator{
		// The storage loader function is unnecessary; the storage key list
		// produced by the supplied buffer alone is sufficient for iteration.
		//
		// The storage key list for iteration is deterministic once the iterator
		// is constructed, no matter the referenced disk layer is stale or not
		// later.
		a: newDiffStorageIterator(account, seek, dl.buffer.states, nil),
		b: newDiskStorageIterator(dl.db.diskdb, account, seek),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryStorageIterator creates a simplistic iterator to step over all the
// storage slots in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryStorageIterator` as the API.
func (dl *diffLayer) initBinaryStorageIterator(account common.Hash, seek common.Hash) *binaryIterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		l := &binaryIterator{
			// The storage loader function is unnecessary; the storage key list
			// produced by the supplied state set alone is sufficient for iteration.
			//
			// The storage key list for iteration is deterministic once the iterator
			// is constructed, no matter the referenced disk layer is stale or not
			// later.
			a: newDiffStorageIterator(account, seek, dl.states.stateSet, nil),
			b: dl.parent.(*diskLayer).initBinaryStorageIterator(account, seek),
		}
		l.aDone = !l.a.Next()
		l.bDone = !l.b.Next()
		return l
	}
	l := &binaryIterator{
		// The storage loader function is unnecessary; the storage key list
		// produced by the supplied state set alone is sufficient for iteration.
		//
		// The storage key list for iteration is deterministic once the iterator
		// is constructed, no matter the referenced disk layer is stale or not
		// later.
		a: newDiffStorageIterator(account, seek, dl.states.stateSet, nil),
		b: parent.initBinaryStorageIterator(account, seek),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// Next advances the iterator by one element, returning false if both iterators
// are exhausted. Note that the entry pointed to by the iterator may be null
// (e.g., when an account is deleted but still accessible for iteration).
// The outer iterator must verify emptiness before terminating the iteration.
//
// There’s no need to check for errors in the two iterators, as we only iterate
// through the entries without retrieving their values.
func (it *binaryIterator) Next() bool {
	if it.aDone && it.bDone {
		return false
	}
	for {
		if it.aDone {
			it.k = it.b.Hash()
			it.bDone = !it.b.Next()
			return true
		}
		if it.bDone {
			it.k = it.a.Hash()
			it.aDone = !it.a.Next()
			return true
		}
		nextA, nextB := it.a.Hash(), it.b.Hash()
		if diff := bytes.Compare(nextA[:], nextB[:]); diff < 0 {
			it.aDone = !it.a.Next()
			it.k = nextA
			return true
		} else if diff == 0 {
			// Now we need to advance one of them
			it.aDone = !it.a.Next()
			continue
		}
		it.bDone = !it.b.Next()
		it.k = nextB
		return true
	}
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. snapshot stack becoming stale).
func (it *binaryIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the account the iterator is currently at.
func (it *binaryIterator) Hash() common.Hash {
	return it.k
}

// Release recursively releases all the iterators in the stack.
func (it *binaryIterator) Release() {
	it.a.Release()
	it.b.Release()
}

// accountBinaryIterator is a wrapper around a binary iterator that adds functionality
// to retrieve account data from the associated layer at the current position.
type accountBinaryIterator struct {
	*binaryIterator
	layer layer
}

// newBinaryAccountIterator creates a simplistic account iterator to step over
// all the accounts in a slow, but easily verifiable way.
//
// nolint:all
func (dl *diskLayer) newBinaryAccountIterator(seek common.Hash) AccountIterator {
	return &accountBinaryIterator{
		binaryIterator: dl.initBinaryAccountIterator(seek),
		layer:          dl,
	}
}

// newBinaryAccountIterator creates a simplistic account iterator to step over
// all the accounts in a slow, but easily verifiable way.
func (dl *diffLayer) newBinaryAccountIterator(seek common.Hash) AccountIterator {
	return &accountBinaryIterator{
		binaryIterator: dl.initBinaryAccountIterator(seek),
		layer:          dl,
	}
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason (e.g. the linked layer is
// stale during the iteration).
func (it *accountBinaryIterator) Next() bool {
	for {
		if !it.binaryIterator.Next() {
			return false
		}
		// Retrieve the account data referenced by the current iterator, the
		// associated layers might be outdated due to chain progressing,
		// the relative error will be set to it.fail just in case.
		//
		// Skip the null account which was deleted before and move to the
		// next account.
		if len(it.Account()) != 0 {
			return true
		}
		// it.fail might be set if error occurs by calling it.Account().
		// Stop iteration if so.
		if it.fail != nil {
			return false
		}
	}
}

// Account returns the RLP encoded slim account the iterator is currently at, or
// nil if the iterated snapshot stack became stale (you can check Error after
// to see if it failed or not).
//
// Note the returned account is not a copy, please don't modify it.
func (it *accountBinaryIterator) Account() []byte {
	blob, err := it.layer.account(it.k, 0)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}

// storageBinaryIterator is a wrapper around a binary iterator that adds functionality
// to retrieve storage slot data from the associated layer at the current position.
type storageBinaryIterator struct {
	*binaryIterator
	account common.Hash
	layer   layer
}

// newBinaryStorageIterator creates a simplistic account iterator to step over
// all the storage slots in a slow, but easily verifiable way.
//
// nolint:all
func (dl *diskLayer) newBinaryStorageIterator(account common.Hash, seek common.Hash) StorageIterator {
	return &storageBinaryIterator{
		binaryIterator: dl.initBinaryStorageIterator(account, seek),
		account:        account,
		layer:          dl,
	}
}

// newBinaryStorageIterator creates a simplistic account iterator to step over
// all the storage slots in a slow, but easily verifiable way.
func (dl *diffLayer) newBinaryStorageIterator(account common.Hash, seek common.Hash) StorageIterator {
	return &storageBinaryIterator{
		binaryIterator: dl.initBinaryStorageIterator(account, seek),
		account:        account,
		layer:          dl,
	}
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason (e.g. the linked layer is
// stale during the iteration).
func (it *storageBinaryIterator) Next() bool {
	for {
		if !it.binaryIterator.Next() {
			return false
		}
		// Retrieve the storage data referenced by the current iterator, the
		// associated layers might be outdated due to chain progressing,
		// the relative error will be set to it.fail just in case.
		//
		// Skip the null storage which was deleted before and move to the
		// next account.
		if len(it.Slot()) != 0 {
			return true
		}
		// it.fail might be set if error occurs by calling it.Slot().
		// Stop iteration if so.
		if it.fail != nil {
			return false
		}
	}
}

// Slot returns the raw storage slot data the iterator is currently at, or
// nil if the iterated snapshot stack became stale (you can check Error after
// to see if it failed or not).
//
// Note the returned slot is not a copy, please don't modify it.
func (it *storageBinaryIterator) Slot() []byte {
	blob, err := it.layer.storage(it.account, it.k, 0)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}
