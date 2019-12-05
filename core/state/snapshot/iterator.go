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
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// AccountIterator is an iterator to step over all the accounts in a snapshot,
// which may or may npt be composed of multiple layers.
type AccountIterator interface {
	// Seek steps the iterator forward as many elements as needed, so that after
	// calling Next(), the iterator will be at a key higher than the given hash.
	Seek(hash common.Hash)

	// Next steps the iterator forward one element, returning false if exhausted,
	// or an error if iteration failed for some reason (e.g. root being iterated
	// becomes stale and garbage collected).
	Next() bool

	// Error returns any failure that occurred during iteration, which might have
	// caused a premature iteration exit (e.g. snapshot stack becoming stale).
	Error() error

	// Key returns the hash of the account the iterator is currently at.
	Key() common.Hash

	// Value returns the RLP encoded slim account the iterator is currently at.
	// An error will be returned if the iterator becomes invalid (e.g. snaph
	Value() []byte
}

// diffAccountIterator is an account iterator that steps over the accounts (both
// live and deleted) contained within a single
type diffAccountIterator struct {
	layer *diffLayer
	index int
}

func (dl *diffLayer) newAccountIterator() *diffAccountIterator {
	dl.AccountList()
	return &diffAccountIterator{layer: dl, index: -1}
}

// Seek steps the iterator forward as many elements as needed, so that after
// calling Next(), the iterator will be at a key higher than the given hash.
func (it *diffAccountIterator) Seek(key common.Hash) {
	// Search uses binary search to find and return the smallest index i
	// in [0, n) at which f(i) is true
	index := sort.Search(len(it.layer.accountList), func(i int) bool {
		return bytes.Compare(key[:], it.layer.accountList[i][:]) < 0
	})
	it.index = index - 1
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diffAccountIterator) Next() bool {
	if it.index < len(it.layer.accountList) {
		it.index++
	}
	return it.index < len(it.layer.accountList)
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. snapshot stack becoming stale).
//
// A diff layer is immutable after creation content wise and can always be fully
// iterated without error, so this method always returns nil.
func (it *diffAccountIterator) Error() error {
	return nil
}

// Key returns the hash of the account the iterator is currently at.
func (it *diffAccountIterator) Key() common.Hash {
	if it.index < len(it.layer.accountList) {
		return it.layer.accountList[it.index]
	}
	return common.Hash{}
}

// Value returns the RLP encoded slim account the iterator is currently at.
func (it *diffAccountIterator) Value() []byte {
	it.layer.lock.RLock()
	defer it.layer.lock.RUnlock()

	hash := it.layer.accountList[it.index]
	if data, ok := it.layer.accountData[hash]; ok {
		return data
	}
	panic("iterator references non-existent layer account")
}

func (dl *diffLayer) iterators() []AccountIterator {
	if parent, ok := dl.parent.(*diffLayer); ok {
		iterators := parent.iterators()
		return append(iterators, dl.newAccountIterator())
	}
	return []AccountIterator{dl.newAccountIterator()}
}
