// Copyright 2016 The go-ethereum Authors
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

package core

import (
	"container/heap"
	"math"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/core/types"
)

// nonceHeap is a heap.Interface implementation over 64bit unsigned integers for
// retrieving sorted transactions from the possibly gapped future queue.
type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// txList is a "list" of transactions belonging to an account, sorted by account
// nonce. The same type can be used both for storing contiguous transactions for
// the executable/pending queue; and for storing gapped transactions for the non-
// executable/future queue, with minor behavoiral changes.
type txList struct {
	strict bool                          // Whether nonces are strictly continuous or not
	items  map[uint64]*types.Transaction // Hash map storing the transaction data
	cache  types.Transactions            // cache of the transactions already sorted

	first uint64     // Nonce of the lowest stored transaction (strict mode)
	last  uint64     // Nonce of the highest stored transaction (strict mode)
	index *nonceHeap // Heap of nonces of all teh stored transactions (non-strict mode)

	costcap *big.Int // Price of the highest costing transaction (reset only if exceeds balance)
}

// newTxList create a new transaction list for maintaining nonce-indexable fast,
// gapped, sortable transaction lists.
func newTxList(strict bool) *txList {
	return &txList{
		strict:  strict,
		items:   make(map[uint64]*types.Transaction),
		first:   math.MaxUint64,
		index:   &nonceHeap{},
		costcap: new(big.Int),
	}
}

// Add tries to inserts a new transaction into the list, returning whether the
// transaction was acceped, and if yes, any previous transaction it replaced.
//
// In case of strict lists (contiguous nonces) the nonce boundaries are updated
// appropriately with the new transaction. Otherwise (gapped nonces) the heap of
// nonces is expanded with the new transaction.
func (l *txList) Add(tx *types.Transaction) (bool, *types.Transaction) {
	// If an existing transaction is better, discard new one
	nonce := tx.Nonce()

	old, ok := l.items[nonce]
	if ok && old.GasPrice().Cmp(tx.GasPrice()) >= 0 {
		return false, nil
	}
	// Otherwise insert the transaction and replace any previous one
	l.items[nonce] = tx
	if cost := tx.Cost(); l.costcap.Cmp(cost) < 0 {
		l.costcap = cost
	}
	if l.strict {
		// In strict mode, maintain the nonce sequence boundaries
		if nonce < l.first {
			l.first = nonce
		}
		if nonce > l.last {
			l.last = nonce
		}
	} else {
		// In gapped mode, maintain the nonce heap
		heap.Push(l.index, nonce)
	}
	l.cache = nil

	return true, old
}

// Forward removes all transactions from the list with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (l *txList) Forward(threshold uint64) types.Transactions {
	var removed types.Transactions

	if l.strict {
		// In strict mode, push the lowest nonce forward to the threshold
		for l.first < threshold {
			if tx, ok := l.items[l.first]; ok {
				removed = append(removed, tx)
			}
			delete(l.items, l.first)
			l.first++
		}
		if l.first > l.last {
			l.last = l.first
		}
	} else {
		// In gapped mode, pop off heap items until the threshold is reached
		for l.index.Len() > 0 && (*l.index)[0] < threshold {
			nonce := heap.Pop(l.index).(uint64)
			removed = append(removed, l.items[nonce])
			delete(l.items, nonce)
		}
	}
	l.cache = nil

	return removed
}

// Filter removes all transactions from the list with a cost higher than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance. Strict-mode invalidated transactions are also returned.
//
// This method uses the cached costcap to quickly decide if there's even a point
// in calculating all the costs or if the balance covers all. If the threshold is
// loewr than the costcap, the costcap will be reset to a new high after removing
// expensive the too transactions.
func (l *txList) Filter(threshold *big.Int) (types.Transactions, types.Transactions) {
	// If all transactions are blow the threshold, short circuit
	if l.costcap.Cmp(threshold) <= 0 {
		return nil, nil
	}
	l.costcap = new(big.Int).Set(threshold) // Lower the cap to the threshold

	// Gather all the transactions needing deletion
	var removed types.Transactions
	for _, tx := range l.items {
		if cost := tx.Cost(); cost.Cmp(threshold) > 0 {
			removed = append(removed, tx)
			delete(l.items, tx.Nonce())
		}
	}
	// Readjust the nonce boundaries/indexes and gather invalidate tranactions
	var invalids types.Transactions
	if l.strict {
		// In strict mode iterate find the first gap and invalidate everything after it
		for i := l.first; i <= l.last; i++ {
			if _, ok := l.items[i]; !ok {
				// Gap found, invalidate all subsequent transactions
				for j := i + 1; j <= l.last; j++ {
					if tx, ok := l.items[j]; ok {
						invalids = append(invalids, tx)
						delete(l.items, j)
					}
				}
				// Reduce the highest transaction nonce and return
				l.last = i - 1
				break
			}
		}
	} else {
		// In gapped mode no transactions are invalid, but the heap is ruined
		l.index = &nonceHeap{}
		for nonce, _ := range l.items {
			*l.index = append(*l.index, nonce)
		}
		heap.Init(l.index)
	}
	l.cache = nil

	return removed, invalids
}

// Cap places a hard limit on the number of items, returning all transactions
// exceeding tht limit.
func (l *txList) Cap(threshold int) types.Transactions {
	// Short circuit if the number of items is under the limit
	if len(l.items) < threshold {
		return nil
	}
	// Otherwise gather and drop the highest nonce'd transactions
	var drops types.Transactions

	if l.strict {
		// In strict mode, just gather top down from last to first
		for len(l.items) > threshold {
			if tx, ok := l.items[l.last]; ok {
				drops = append(drops, tx)
				delete(l.items, l.last)
				l.last--
			}
		}
	} else {
		// In gapped mode it's expensive: we need to sort and drop like that
		sort.Sort(*l.index)
		for size := len(l.items); size > threshold; size-- {
			drops = append(drops, l.items[(*l.index)[size-1]])
			delete(l.items, (*l.index)[size-1])
			*l.index = (*l.index)[:size-1]
		}
		heap.Init(l.index)
	}
	l.cache = nil

	return drops
}

// Remove deletes a transaction from the maintained list, returning whether the
// transaction was found, and also returning any transaction invalidated due to
// the deletion (strict mode only).
func (l *txList) Remove(tx *types.Transaction) (bool, types.Transactions) {
	nonce := tx.Nonce()
	if _, ok := l.items[nonce]; ok {
		// Remove the item and invalidate the sorted cache
		delete(l.items, nonce)
		l.cache = nil

		// Remove all invalidated transactions (strict mode only!)
		invalids := make(types.Transactions, 0, l.last-nonce)
		if l.strict {
			for i := nonce + 1; i <= l.last; i++ {
				invalids = append(invalids, l.items[i])
				delete(l.items, i)
			}
			l.last = nonce - 1
		} else {
			// In gapped mode, remove the nonce from the index but honour the heap
			for i := 0; i < l.index.Len(); i++ {
				if (*l.index)[i] == nonce {
					heap.Remove(l.index, i)
					break
				}
			}
		}
		// Figure out the new highest nonce
		return true, invalids
	}
	return false, nil
}

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
//
// Note, all transactions with nonces lower that start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (l *txList) Ready(start uint64) types.Transactions {
	var txs types.Transactions
	if l.strict {
		// In strict mode make sure we have valid transaction, return all contiguous
		if l.first > start {
			return nil
		}
		for {
			if tx, ok := l.items[l.first]; ok {
				txs = append(txs, tx)
				delete(l.items, l.first)
				l.first++
				continue
			}
			break
		}
	} else {
		// In gapped mode, check the heap start and return all contiguous
		if l.index.Len() == 0 || (*l.index)[0] > start {
			return nil
		}
		next := (*l.index)[0]
		for l.index.Len() > 0 && (*l.index)[0] == next {
			txs = append(txs, l.items[next])
			delete(l.items, next)
			heap.Pop(l.index)
			next++
		}
	}
	l.cache = nil

	return txs
}

// Len returns the length of the transaction list.
func (l *txList) Len() int {
	return len(l.items)
}

// Empty returns whether the list of transactions is empty or not.
func (l *txList) Empty() bool {
	return len(l.items) == 0
}

// Flatten creates a nonce-sorted slice of transactions based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (l *txList) Flatten() types.Transactions {
	// If the sorting was not cached yet, create and cache it
	if l.cache == nil {
		l.cache = make(types.Transactions, 0, len(l.items))
		for _, tx := range l.items {
			l.cache = append(l.cache, tx)
		}
		sort.Sort(types.TxByNonce(l.cache))
	}
	// Copy the cache to prevent accidental modifications
	txs := make(types.Transactions, len(l.cache))
	copy(txs, l.cache)
	return txs
}
