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

// Less defines a ordering of executable transactions.
// Less is not a total ordering as it lacks transitivity.
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
		return t.tx.Nonce() > entry.tx.Nonce()
	}
	if t.price.Cmp(entry.price) == 0 {
		return t.tx.Before(entry.tx)
	}
	return t.price.Cmp(entry.price) == -1
}

type txList struct {
	head         *txEntry
	bottom       *txEntry
	lowestEntry  *txEntry
	highestNonce map[common.Address]uint64
	len          int
	maxLen       int
}

func newTxList(maxLen int) txList {
	return txList{
		maxLen:       maxLen,
		highestNonce: make(map[common.Address]uint64),
	}
}

func (l *txList) LowestEntry() *txEntry {
	if l.lowestEntry != nil {
		return l.lowestEntry
	}
	if l.len == 0 {
		return nil
	}
	lowest := l.head
	for new := l.head; new != nil; new = new.next {
		if new.price.Cmp(lowest.price) < 0 {
			lowest = new
		}
	}
	// Cache lowest entry
	l.lowestEntry = lowest
	return lowest
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
		l.len++
		l.highestNonce[entry.sender] = entry.tx.Nonce()
		return false
	}
	if nonce, ok := l.highestNonce[entry.sender]; ok && nonce < entry.tx.Nonce() {
		// TODO go through the whole list
		// Can only insert after the last highest entry
		return l.addHeavy(entry)
	}

	l.highestNonce[entry.sender] = entry.tx.Nonce()
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
		if new.Less(entry) {
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

// addHeavy adds a txEntry into the list.
// This method is used to add entries at the correct place in the list
// if the list already contains a transaction from this sender with a lower nonce.
// Expects the nonce to exist in the list.
// Expects head to be non-nil.
func (l *txList) addHeavy(entry *txEntry) bool {
	nonce := l.highestNonce[entry.sender]
	old := l.head
	for new := l.head; new != nil; new = new.next {
		if new.sender == entry.sender && nonce == new.tx.Nonce() {
			// Found the highest nonce, now insert afterwards
			break
		}
		old = new
	}
	var inserted bool
	for new := old.next; new != nil; new = new.next {
		if new.Less(entry) {
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
// Returns the txEntry if found.
func (l *txList) Delete(equal func(*txEntry) bool) *txEntry {
	if l.len == 0 {
		return nil
	}
	// Invalidate cache
	l.lowestEntry = nil
	var result *txEntry
	if equal(l.head) {
		if l.len == 1 {
			l.bottom = nil
		}
		h := l.head
		l.head = l.head.next
		l.len--
		result = h
	} else {
		old := l.head
		for new := old.next; new != nil; new = new.next {
			if equal(new) {
				if l.bottom == new {
					l.bottom = old
				}
				old.next = new.next
				l.len--
				result = new
				break
			}
			old = new
		}
	}
	// If the highest nonce was deleted, it's okay to delete it from the address cache,
	// since all lower transactions have been executed already.
	// If the tx is deleted because it will be updated, the highest nonce will be set again
	// during the Add function
	if result != nil && l.highestNonce[result.sender] == result.tx.Nonce() {
		delete(l.highestNonce, result.sender)
	}
	return result
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

// Prune splits the list to 3/4 of the max length.
// It returns a new list with the resulting entries
func (l *txList) Prune() *txList {
	len := (l.maxLen / 4) * 3
	if len >= l.len {
		return nil
	}
	l.lowestEntry = nil
	res := newTxList(l.maxLen)
	old := l.head
	for i := 0; i < len; i++ {
		old = old.next
	}
	res.head = old.next
	res.bottom = l.bottom
	l.bottom = old
	res.len = l.len - len
	l.len = len
	return &res
}
