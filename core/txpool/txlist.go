// Copyright 2021 The go-ethereum Authors
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

package txpool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type txEntry struct {
	price  *big.Int
	sender common.Address
	tx     *types.Transaction
	next   *txEntry
}

// Less defines a total ordering of executable transactions.
// It panics if t is nil.
// If entry is nil, Less will return false
// Transactions from the same sender are sorted by nonce
// Transactions from different senders are sorted by gas price
// Transactions with the same gas price are sorted by arrival time
func (t *txEntry) Less(entry *txEntry) bool {
	if t == nil {
		return false
	}
	if entry == nil {
		panic("Less called with nil arg, should not happen")
	}
	if t.sender == entry.sender {
		return t.tx.Nonce() < entry.tx.Nonce()
	}
	if t.price.Cmp(entry.price) == 0 {
		return t.tx.Before(entry.tx)
	}
	return t.price.Cmp(entry.price) == -1
}

type txList struct {
	head   *txEntry
	bottom *txEntry
	len    int
	maxLen int
}

func (l *txList) LastEntry() *txEntry {
	return l.bottom
}

func (l *txList) Len() int {
	return l.len
}

// Add adds a new tx entry to the list.
// Returns true if the tx list should be pruned.
func (l *txList) Add(entry *txEntry) bool {
	if l.head == nil {
		l.head = entry
		l.bottom = entry
		return false
	}
	// If the new entry is bigger than the head, replace head
	old := l.head
	if old.Less(entry) {
		l.head = entry
		entry.next = old
		l.len++
		return l.len > l.maxLen
	}
	// Insert into the linked list
	inserted := false
	for new := old.next; new != nil; new = new.next {
		if !new.Less(entry) {
			old.next = entry
			entry.next = new
			l.len++
			inserted = true
			return l.len > l.maxLen
		}
		old = new
	}
	// Not inserted? Insert as last element
	if !inserted {
		old.next = entry
		l.bottom = entry
		l.len++
	}
	return l.len > l.maxLen
}

// Delete the first occurence where equal returns true.
// Returns true if we found the txEntry
func (l *txList) Delete(equal func(*txEntry) bool) *txEntry {
	if l.len == 0 {
		return nil
	}
	if equal(l.head) {
		if l.len == 1 {
			l.bottom = nil
		}
		h := l.head
		l.head = l.head.next
		l.len--
		return h
	}
	old := l.head
	for new := old.next; new != nil; new = new.next {
		if equal(new) {
			if l.bottom == new {
				l.bottom = old
			}
			old.next = new.next
			l.len--
			return new
		}
		old = new
	}
	return nil
}

// Peek returns the next len transactions.
// If not enough transactions are available, less are returned.
func (l *txList) Peek(len int) types.Transactions {
	size := len
	if size > l.len {
		size = l.len
	}
	t := make(types.Transactions, 0, size)
	for new := l.head; new != nil; new = new.next {
		t = append(t, new.tx)
	}
	return t
}

func (l *txList) Prune() {

}
