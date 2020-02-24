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

package snapshot

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

// binaryAccountIterator is a simplistic iterator to step over the accounts in
// a snapshot, which may or may npt be composed of multiple layers. Performance
// wise this iterator is slow, it's meant for cross validating the fast one,
type binaryAccountIterator struct {
	a     *diffAccountIterator
	b     AccountIterator
	aDone bool
	bDone bool
	k     common.Hash
	fail  error
}

// newBinaryAccountIterator creates a simplistic account iterator to step over
// all the accounts in a slow, but eaily verifiable way.
func (dl *diffLayer) newBinaryAccountIterator() AccountIterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		// parent is the disk layer
		return dl.AccountIterator(common.Hash{})
	}
	l := &binaryAccountIterator{
		a: dl.AccountIterator(common.Hash{}).(*diffAccountIterator),
		b: parent.newBinaryAccountIterator(),
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason (e.g. root being iterated
// becomes stale and garbage collected).
func (it *binaryAccountIterator) Next() bool {
	if it.aDone && it.bDone {
		return false
	}
	nextB := it.b.Hash()
first:
	nextA := it.a.Hash()
	if it.aDone {
		it.bDone = !it.b.Next()
		it.k = nextB
		return true
	}
	if it.bDone {
		it.aDone = !it.a.Next()
		it.k = nextA
		return true
	}
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
func (it *binaryAccountIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the account the iterator is currently at.
func (it *binaryAccountIterator) Hash() common.Hash {
	return it.k
}

// Account returns the RLP encoded slim account the iterator is currently at, or
// nil if the iterated snapshot stack became stale (you can check Error after
// to see if it failed or not).
func (it *binaryAccountIterator) Account() []byte {
	blob, err := it.a.layer.AccountRLP(it.k)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}

// Release recursively releases all the iterators in the stack.
func (it *binaryAccountIterator) Release() {
	it.a.Release()
	it.b.Release()
}
