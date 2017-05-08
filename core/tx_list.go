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
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
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

// txSortedMap is a nonce->transaction hash map with a heap based index to allow
// iterating over the contents in a nonce-incrementing way.
type txSortedMap struct {
	items map[uint64]*types.Transaction // Hash map storing the transaction data
	index *nonceHeap                    // Heap of nonces of all the stored transactions (non-strict mode)
	cache types.Transactions            // Cache of the transactions already sorted
}

// newTxSortedMap creates a new nonce-sorted transaction map.
func newTxSortedMap() *txSortedMap {
	return &txSortedMap{
		items: make(map[uint64]*types.Transaction),
		index: new(nonceHeap),
	}
}

// Get retrieves the current transactions associated with the given nonce.
func (m *txSortedMap) Get(nonce uint64) *types.Transaction {
	return m.items[nonce]
}

// Put inserts a new transaction into the map, also updating the map's nonce
// index. If a transaction already exists with the same nonce, it's overwritten.
func (m *txSortedMap) Put(tx *types.Transaction) {
	nonce := tx.Nonce()
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
	}
	m.items[nonce], m.cache = tx, nil
}

// Forward removes all transactions from the map with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (m *txSortedMap) Forward(threshold uint64) types.Transactions {
	var removed types.Transactions

	// Pop off heap items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		delete(m.items, nonce)
	}
	// If we had a cached order, shift the front
	if m.cache != nil {
		m.cache = m.cache[len(removed):]
	}
	return removed
}

// Filter iterates over the list of transactions and removes all of them for which
// the specified function evaluates to true.
func (m *txSortedMap) Filter(filter func(*types.Transaction) bool) types.Transactions {
	var removed types.Transactions

	// Collect all the transactions to filter out
	for nonce, tx := range m.items {
		if filter(tx) {
			removed = append(removed, tx)
			delete(m.items, nonce)
		}
	}
	// If transactions were removed, the heap and cache are ruined
	if len(removed) > 0 {
		*m.index = make([]uint64, 0, len(m.items))
		for nonce := range m.items {
			*m.index = append(*m.index, nonce)
		}
		heap.Init(m.index)

		m.cache = nil
	}
	return removed
}

// Cap places a hard limit on the number of items, returning all transactions
// exceeding that limit.
func (m *txSortedMap) Cap(threshold int) types.Transactions {
	// Short circuit if the number of items is under the limit
	if len(m.items) <= threshold {
		return nil
	}
	// Otherwise gather and drop the highest nonce'd transactions
	var drops types.Transactions

	sort.Sort(*m.index)
	for size := len(m.items); size > threshold; size-- {
		drops = append(drops, m.items[(*m.index)[size-1]])
		delete(m.items, (*m.index)[size-1])
	}
	*m.index = (*m.index)[:threshold]
	heap.Init(m.index)

	// If we had a cache, shift the back
	if m.cache != nil {
		m.cache = m.cache[:len(m.cache)-len(drops)]
	}
	return drops
}

// Remove deletes a transaction from the maintained map, returning whether the
// transaction was found.
func (m *txSortedMap) Remove(nonce uint64) bool {
	// Short circuit if no transaction is present
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	// Otherwise delete the transaction and fix the heap index
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	m.cache = nil

	return true
}

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
//
// Note, all transactions with nonces lower than start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (m *txSortedMap) Ready(start uint64) types.Transactions {
	// Short circuit if no transactions are available
	if m.index.Len() == 0 || (*m.index)[0] > start {
		return nil
	}
	// Otherwise start accumulating incremental transactions
	var ready types.Transactions
	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
		ready = append(ready, m.items[next])
		delete(m.items, next)
		heap.Pop(m.index)
	}
	m.cache = nil

	return ready
}

// Len returns the length of the transaction map.
func (m *txSortedMap) Len() int {
	return len(m.items)
}

// Flatten creates a nonce-sorted slice of transactions based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (m *txSortedMap) Flatten() types.Transactions {
	// If the sorting was not cached yet, create and cache it
	if m.cache == nil {
		m.cache = make(types.Transactions, 0, len(m.items))
		for _, tx := range m.items {
			m.cache = append(m.cache, tx)
		}
		sort.Sort(types.TxByNonce(m.cache))
	}
	// Copy the cache to prevent accidental modifications
	txs := make(types.Transactions, len(m.cache))
	copy(txs, m.cache)
	return txs
}

// txList is a "list" of transactions belonging to an account, sorted by account
// nonce. The same type can be used both for storing contiguous transactions for
// the executable/pending queue; and for storing gapped transactions for the non-
// executable/future queue, with minor behavioral changes.
type txList struct {
	strict  bool         // Whether nonces are strictly continuous or not
	txs     *txSortedMap // Heap indexed sorted hash map of the transactions
	costcap *big.Int     // Price of the highest costing transaction (reset only if exceeds balance)
}

// newTxList create a new transaction list for maintaining nonce-indexable fast,
// gapped, sortable transaction lists.
func newTxList(strict bool) *txList {
	return &txList{
		strict:  strict,
		txs:     newTxSortedMap(),
		costcap: new(big.Int),
	}
}

// Add tries to insert a new transaction into the list, returning whether the
// transaction was accepted, and if yes, any previous transaction it replaced.
//
// If the new transaction is accepted into the list, the lists' cost threshold
// is also potentially updated.
func (l *txList) Add(tx *types.Transaction) (bool, *types.Transaction) {
	// If there's an older better transaction, abort
	old := l.txs.Get(tx.Nonce())
	if old != nil && old.GasPrice().Cmp(tx.GasPrice()) >= 0 {
		return false, nil
	}
	// Otherwise overwrite the old transaction with the current one
	l.txs.Put(tx)
	if cost := tx.Cost(); l.costcap.Cmp(cost) < 0 {
		l.costcap = cost
	}
	return true, old
}

// Forward removes all transactions from the list with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (l *txList) Forward(threshold uint64) types.Transactions {
	return l.txs.Forward(threshold)
}

// Filter removes all transactions from the list with a cost higher than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance. Strict-mode invalidated transactions are also returned.
//
// This method uses the cached costcap to quickly decide if there's even a point
// in calculating all the costs or if the balance covers all. If the threshold is
// lower than the costcap, the costcap will be reset to a new high after removing
// expensive the too transactions.
func (l *txList) Filter(threshold *big.Int) (types.Transactions, types.Transactions) {
	// If all transactions are below the threshold, short circuit
	if l.costcap.Cmp(threshold) <= 0 {
		return nil, nil
	}
	l.costcap = new(big.Int).Set(threshold) // Lower the cap to the threshold

	// Filter out all the transactions above the account's funds
	removed := l.txs.Filter(func(tx *types.Transaction) bool { return tx.Cost().Cmp(threshold) > 0 })

	// If the list was strict, filter anything above the lowest nonce
	var invalids types.Transactions
	if l.strict && len(removed) > 0 {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > lowest })
	}
	return removed, invalids
}

// Cap places a hard limit on the number of items, returning all transactions
// exceeding that limit.
func (l *txList) Cap(threshold int) types.Transactions {
	return l.txs.Cap(threshold)
}

// Remove deletes a transaction from the maintained list, returning whether the
// transaction was found, and also returning any transaction invalidated due to
// the deletion (strict mode only).
func (l *txList) Remove(tx *types.Transaction) (bool, types.Transactions) {
	// Remove the transaction from the set
	nonce := tx.Nonce()
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}
	// In strict mode, filter out non-executable transactions
	if l.strict {
		return true, l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > nonce })
	}
	return true, nil
}

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
//
// Note, all transactions with nonces lower than start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (l *txList) Ready(start uint64) types.Transactions {
	return l.txs.Ready(start)
}

// Len returns the length of the transaction list.
func (l *txList) Len() int {
	return l.txs.Len()
}

// Empty returns whether the list of transactions is empty or not.
func (l *txList) Empty() bool {
	return l.Len() == 0
}

// Flatten creates a nonce-sorted slice of transactions based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (l *txList) Flatten() types.Transactions {
	return l.txs.Flatten()
}

// priceHeap is a heap.Interface implementation over big integers for retrieving
// sorted transaction prices to discard when the pool fills up.
type priceHeap []*big.Int

func (h priceHeap) Len() int           { return len(h) }
func (h priceHeap) Less(i, j int) bool { return h[i].Cmp(h[j]) < 0 }
func (h priceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *priceHeap) Push(x interface{}) {
	*h = append(*h, x.(*big.Int))
}

func (h *priceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// priceKey is the hash map key type representing an unsigned big integer.
type priceKey [256 / int(unsafe.Sizeof(uintptr(0)))]big.Word

// newPriceKey converts a bit integer to a hash map key.
func newPriceKey(x *big.Int) priceKey {
	var key priceKey
	copy(key[:], x.Bits())
	return key
}

// txPricedMap is a price->transactions hash map with a heap based index to allow
// iterating over the contents in a price-incrementing way.
type txPricedMap struct {
	items  map[priceKey]map[common.Hash]*types.Transaction // Hash map storing the transaction data
	index  *priceHeap                                      // Heap of prices of all the stored transactions
	stales int                                             // Number of stale price points to (re-heap trigger)
	logger log.Logger                                      // Logger reporting basic price pool stats
}

// newTxPricedMap creates a new price-sorted transaction map.
func newTxPricedMap() *txPricedMap {
	m := &txPricedMap{
		items: make(map[priceKey]map[common.Hash]*types.Transaction),
		index: new(priceHeap),
	}
	m.logger = log.New(
		"prices", log.Lazy{Fn: func() int { return len(*m.index) - m.stales }},
		"stales", log.Lazy{Fn: func() int { return m.stales }},
	)
	return m
}

// Put inserts a new transaction into the map, also updating the map's price index.
func (m *txPricedMap) Put(tx *types.Transaction) {
	price, hash := tx.GasPrice(), tx.Hash()

	// Generate the key and ensure we have a valid map for it
	key := newPriceKey(price)
	if m.items[key] == nil {
		m.items[key] = make(map[common.Hash]*types.Transaction)
		heap.Push(m.index, price)
	} else if len(m.items[key]) == 0 {
		m.stales--
	}
	// Add the transaction to the map
	m.items[key][hash] = tx

	m.logger.Trace("Accepted new transaction", "hash", hash, "price", price)
}

// Remove looks up a transaction and deletes it from the sorted map. Even if no
// more transactions are left with the same price point, the price point itself
// is retained to achieve hysteresis, until a staleness threshold is reached.
func (m *txPricedMap) Remove(tx *types.Transaction) {
	price, hash := tx.GasPrice(), tx.Hash()

	key := newPriceKey(price)
	if txs := m.items[key]; txs != nil {
		if txs[hash] != nil {
			// Transaction found, delete and update stale counter
			delete(m.items[key], hash)
			if len(m.items[key]) == 0 {
				m.stales++
			}
			// If the number of stale entries reached a critical threshold (25%), re-heap and clean up
			if m.stales > len(*m.index)/4 {
				m.reheap()
			}
		}
	}
	m.logger.Trace("Removed old transaction", "hash", hash, "price", price)
}

// reheap discards the currently cached price point heap and regenerates it based
// on the contents of the priced transaction map.
func (m *txPricedMap) reheap() {
	m.stales, m.index = 0, new(priceHeap)
	for key, txs := range m.items {
		if len(txs) == 0 {
			delete(m.items, key)
		} else {
			*m.index = append(*m.index, new(big.Int).SetBits(key[:]))
		}
	}
	heap.Init(m.index)
}

// Discard finds all the transactions below the given price threshold, drops them
// from the priced map and returs them for further removal from the entire pool.
func (m *txPricedMap) Cap(threshold *big.Int, local *txSet) types.Transactions {
	drop := make(types.Transactions, 0, 128) // Remote underpriced transactions to drop
	save := make(types.Transactions, 0, 64)  // Local underpriced transactions to keep

	for len(*m.index) > 0 {
		// Discard stale price points if found during cleanup
		price := []*big.Int(*m.index)[0]
		key := newPriceKey(price)

		if len(m.items[key]) == 0 {
			m.stales--
			heap.Pop(m.index)
			delete(m.items, key)
			continue
		}
		// Stop the discards if we've reached the threshold
		if price.Cmp(threshold) >= 0 {
			break
		}
		// Non stale price point found, discard its transactions
		for hash, tx := range m.items[key] {
			if local.contains(hash) {
				save = append(save, tx)
			} else {
				drop = append(drop, tx)
			}
		}
		m.items[key] = nil // will get removed in next iteration
	}
	for _, tx := range save {
		m.Put(tx)
	}
	return drop
}

// Underpriced checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced transaction currently being tracked.
func (m *txPricedMap) Underpriced(tx *types.Transaction, local *txSet) bool {
	// Local transactions cannot be underpriced
	if local.contains(tx.Hash()) {
		return false
	}
	// Discard stale price points if found at the heap start
	for len(*m.index) > 0 {
		price := []*big.Int(*m.index)[0]
		if key := newPriceKey(price); len(m.items[key]) == 0 {
			m.stales--
			heap.Pop(m.index)
			delete(m.items, key)
			continue
		}
		break
	}
	// Check if the transaction is underpriced or not
	if len(*m.index) == 0 {
		log.Error("Pricing query for empty pool") // This cannot happen, print to catch programming errors
		return false
	}
	cheapest := []*big.Int(*m.index)[0]
	return cheapest.Cmp(tx.GasPrice()) >= 0
}

// Discard finds a number of most underpriced transactions, removes them from the
// priced map and returs them for further removal from the entire pool.
func (m *txPricedMap) Discard(count int, local *txSet) types.Transactions {
	drop := make(types.Transactions, 0, count) // Remote underpriced transactions to drop
	save := make(types.Transactions, 0, 64)    // Local underpriced transactions to keep

	for len(*m.index) > 0 && count > 0 {
		// Discard stale price points if found during cleanup
		price := []*big.Int(*m.index)[0]
		key := newPriceKey(price)

		if len(m.items[key]) == 0 {
			m.stales--
			heap.Pop(m.index)
			delete(m.items, key)
			continue
		}
		// Non stale price point found, discard its transactions
		for hash, tx := range m.items[key] {
			if count == 0 {
				break
			}
			if local.contains(hash) {
				save = append(save, tx)
			} else {
				drop = append(drop, tx)
				count--
			}
			delete(m.items[key], hash)
		}
	}
	for _, tx := range save {
		m.Put(tx)
	}
	return drop
}
