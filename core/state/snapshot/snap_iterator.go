// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package snapshot

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

// snapIter is a wrapper of underlying database iterator. It extends
// the basic iteration interface by adding Discard which can cache
// the element internally where the iterator is currently located.
type snapIter struct {
	it        ethdb.Iterator
	key       []byte
	val       []byte
	checkLast bool
}

// newSnapIter initializes the snapshot iterator.
func newSnapIter(it ethdb.Iterator) *snapIter {
	return &snapIter{it: it}
}

// Discard stores the element internally where the iterator is currently located.
func (it *snapIter) Discard() {
	if it.it.Key() == nil {
		return // nothing to cache
	}
	it.key = common.CopyBytes(it.it.Key())
	it.val = common.CopyBytes(it.it.Value())
	it.checkLast = false
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (it *snapIter) Next() bool {
	if !it.checkLast && it.key != nil {
		it.checkLast = true
	} else if it.checkLast {
		it.checkLast = false
		it.key = nil
		it.val = nil
	}
	if it.key != nil {
		return true // shift to discarded value
	}
	return it.it.Next()
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error.
func (it *snapIter) Error() error { return it.it.Error() }

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (it *snapIter) Release() {
	it.checkLast = false
	it.key = nil
	it.val = nil
	it.it.Release()
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (it *snapIter) Key() []byte {
	if it.key != nil {
		return it.key
	}
	return it.it.Key()
}

// Value returns the value of the current key/value pair, or nil if done. The
// caller should not modify the contents of the returned slice, and its contents
// may change on the next call to Next.
func (it *snapIter) Value() []byte {
	if it.val != nil {
		return it.val
	}
	return it.it.Value()
}
