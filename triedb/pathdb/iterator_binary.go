// Copyright 2019 The go-ethereum Authors
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
// in a layer, which may or may not be composed of multiple layers. Performance
// wise this iterator is slow, it's meant for cross validating the fast one.
type binaryIterator struct {
	a       Iterator
	b       Iterator
	aDone   bool
	bDone   bool
	k       common.Hash
	account common.Hash
	fail    error
}

// initBinaryAccountIterator creates a simplistic iterator to step over all the
// accounts in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryAccountIterator` as the API.
func (dl *diskLayer) initBinaryAccountIterator() *binaryIterator {
	l := &binaryIterator{
		a: newDiffAccountIterator(common.Hash{}, dl.buffer.states, dl.isStale),
		b: newDiskAccountIterator(dl.db.diskdb, common.Hash{}),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryAccountIterator creates a simplistic iterator to step over all the
// accounts in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryAccountIterator` as the API.
func (dl *diffLayer) initBinaryAccountIterator() *binaryIterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		l := &binaryIterator{
			a: newDiffAccountIterator(common.Hash{}, dl.states.stateSet, nil),
			b: dl.parent.(*diskLayer).initBinaryAccountIterator(),
		}
		l.aDone = !l.a.Next()
		l.bDone = !l.b.Next()
		return l
	}
	l := &binaryIterator{
		a: newDiffAccountIterator(common.Hash{}, dl.states.stateSet, nil),
		b: parent.initBinaryAccountIterator(),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryStorageIterator creates a simplistic iterator to step over all the
// storage slots in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryStorageIterator` as the API.
func (dl *diskLayer) initBinaryStorageIterator(account common.Hash) *binaryIterator {
	a, destructed := newDiffStorageIterator(account, common.Hash{}, dl.buffer.states, dl.isStale)
	if destructed {
		l := &binaryIterator{
			a:       a,
			account: account,
		}
		l.aDone = !l.a.Next()
		l.bDone = true
		return l
	}
	l := &binaryIterator{
		a:       a,
		b:       newDiskStorageIterator(dl.db.diskdb, account, common.Hash{}),
		account: account,
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryStorageIterator creates a simplistic iterator to step over all the
// storage slots in a slow, but easily verifiable way. Note this function is used
// for initialization, use `newBinaryStorageIterator` as the API.
func (dl *diffLayer) initBinaryStorageIterator(account common.Hash) *binaryIterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		// If the storage in this layer is already destructed, discard all
		// deeper layers but still return a valid single-branch iterator.
		a, destructed := newDiffStorageIterator(account, common.Hash{}, dl.states.stateSet, nil)
		if destructed {
			l := &binaryIterator{
				a:       a,
				account: account,
			}
			l.aDone = !l.a.Next()
			l.bDone = true
			return l
		}
		// The parent is disk layer
		l := &binaryIterator{
			a:       a,
			b:       dl.parent.(*diskLayer).initBinaryStorageIterator(account),
			account: account,
		}
		l.aDone = !l.a.Next()
		l.bDone = !l.b.Next()
		return l
	}
	// If the storage in this layer is already destructed, discard all
	// deeper layers but still return a valid single-branch iterator.
	a, destructed := newDiffStorageIterator(account, common.Hash{}, dl.states.stateSet, nil)
	if destructed {
		l := &binaryIterator{
			a:       a,
			account: account,
		}
		l.aDone = !l.a.Next()
		l.bDone = true
		return l
	}
	l := &binaryIterator{
		a:       a,
		b:       parent.initBinaryStorageIterator(account),
		account: account,
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason (e.g. root being iterated
// becomes stale and garbage collected).
func (it *binaryIterator) Next() bool {
	if it.aDone && it.bDone {
		return false
	}
first:
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
		goto first
	}
	it.bDone = !it.b.Next()
	it.k = nextB
	return true
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
//nolint:all
func (dl *diskLayer) newBinaryAccountIterator() AccountIterator {
	return &accountBinaryIterator{
		binaryIterator: dl.initBinaryAccountIterator(),
		layer:          dl,
	}
}

// newBinaryAccountIterator creates a simplistic account iterator to step over
// all the accounts in a slow, but easily verifiable way.
func (dl *diffLayer) newBinaryAccountIterator() AccountIterator {
	return &accountBinaryIterator{
		binaryIterator: dl.initBinaryAccountIterator(),
		layer:          dl,
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
	layer layer
}

// newBinaryStorageIterator creates a simplistic account iterator to step over
// all the storage slots in a slow, but easily verifiable way.
//
//nolint:all
func (dl *diskLayer) newBinaryStorageIterator(account common.Hash) StorageIterator {
	return &storageBinaryIterator{
		binaryIterator: dl.initBinaryStorageIterator(account),
		layer:          dl,
	}
}

// newBinaryStorageIterator creates a simplistic account iterator to step over
// all the storage slots in a slow, but easily verifiable way.
func (dl *diffLayer) newBinaryStorageIterator(account common.Hash) StorageIterator {
	return &storageBinaryIterator{
		binaryIterator: dl.initBinaryStorageIterator(account),
		layer:          dl,
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
