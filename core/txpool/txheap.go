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
	"container/heap"
	"errors"
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

// txHeap is a nonce->transaction hash map with a heap based index to allow
// iterating over the contents in a nonce-incrementing way.
type txHeap struct {
	items map[uint64]*txEntry // Hash map storing the transaction data
	index *nonceHeap          // Heap of nonces of all the stored transactions (non-strict mode)
}

// newTxHeap creates a new nonce-sorted transaction map.
func newTxHeap() *txHeap {
	return &txHeap{
		items: make(map[uint64]*txEntry),
		index: new(nonceHeap),
	}
}

// Get retrieves the current transactions associated with the given nonce.
func (m *txHeap) Get(nonce uint64) *txEntry {
	return m.items[nonce]
}

func (m *txHeap) Remove(nonce uint64) *txEntry {
	// Short circuit if no transaction is present
	_, ok := m.items[nonce]
	if !ok {
		return nil
	}
	// Otherwise delete the transaction and fix the heap index
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {
			heap.Remove(m.index, i)
			break
		}
	}
	item := m.items[nonce]
	delete(m.items, nonce)
	return item
}

func (m *txHeap) Len() int {
	return m.index.Len()
}

// Put inserts a new transaction into the map, also updating the map's nonce
// index. If a transaction already exists with the same nonce, it's overwritten.
func (m *txHeap) Put(tx *txEntry) {
	nonce := tx.tx.Nonce()
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
	}
	m.items[nonce] = tx
}

// LowestNonce retrieves the smallest Nonce in the heap.
func (m *txHeap) LowestNonce() (uint64, error) {
	if m == nil || len(*m.index) == 0 {
		return 0, errors.New("no lowest nonce found")
	}
	return (*m.index)[0], nil
}

// Pop retrieves the tx with the lowest nonce from the heap.
func (m *txHeap) Pop() *txEntry {
	nonce := heap.Pop(m.index).(uint64)
	tx := m.items[nonce]
	delete(m.items, nonce)
	return tx
}
