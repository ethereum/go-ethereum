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

package legacypool

import (
	"container/heap"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// sortedMap is a nonce->transaction hash map with a heap based index to allow
// iterating over the contents in a nonce-incrementing way.
type sortedMap struct {
	items map[uint64]*types.Transaction // Hash map storing the transaction data
	tree  *AVLTree                      // AVL tree of nonces of all the stored transactions (non-strict mode)
}

func newSortedMap() *sortedMap {
	return &sortedMap{
		items: make(map[uint64]*types.Transaction),
		tree:  new(AVLTree),
	}
}

func (m *sortedMap) Get(nonce uint64) *types.Transaction {
	return m.items[nonce]
}

func (m *sortedMap) GetCost(nonce uint64) *big.Int {
	_, cost := m.tree.Search(nonce)
	return cost
}

func (m *sortedMap) Put(tx *types.Transaction) {
	nonce := tx.Nonce()
	m.items[nonce] = tx
	m.tree.Add(nonce, tx.Cost())
}

func (m *sortedMap) Forward(threshold uint64) types.Transactions {
	var remove types.Transactions
	for {
		nonce, err := m.tree.Smallest()
		if nonce >= threshold || err != nil {
			break
		}
		tx := m.items[nonce]
		remove = append(remove, tx)
		m.tree.Remove(nonce)
		delete(m.items, nonce)
	}
	return remove
}

func (m *sortedMap) Filter(filter func(*types.Transaction) bool) types.Transactions {
	var remove types.Transactions
	for nonce, tx := range m.items {
		if filter(tx) {
			remove = append(remove, tx)
			delete(m.items, nonce)
			m.tree.Remove(nonce)
		}
	}
	return remove
}

func (m *sortedMap) Cap(threshold int) types.Transactions {
	// Short circuit if the number of items is under the limit
	size := len(m.items)
	if size <= threshold {
		return nil
	}
	var remove types.Transactions
	for size > threshold {
		nonce, err := m.tree.Largest()
		if err != nil {
			break
		}
		remove = append(remove, m.items[nonce])
		delete(m.items, nonce)
		m.tree.Remove(nonce)
		size--
	}
	return remove
}

func (m *sortedMap) Remove(nonce uint64) bool {
	// Short circuit if no transaction is present
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	delete(m.items, nonce)
	m.tree.Remove(nonce)
	return true
}

// Given the provided start nonce, Ready returns
// transactions that are continous, the varible start is the virtual nonce.
func (m *sortedMap) Ready(start uint64, threshold *big.Int) types.Transactions {
	size := len(m.items)
	if size == 0 {
		return nil
	}
	smallest, err := m.tree.Smallest()
	if smallest > start || err != nil {
		return nil
	}

	var ready types.Transactions
	tx := m.items[smallest]
	total := new(big.Int).Set(tx.Cost())
	for next := smallest; size > 0 && smallest == next && total.Cmp(threshold) <= 0; next++ {
		ready = append(ready, m.items[next])
		m.tree.Remove(smallest)
		delete(m.items, smallest)
		size--

		smallest, err = m.tree.Smallest()
		if err != nil {
			break
		}

		tx = m.items[smallest]
		total = total.Add(total, tx.Cost())
	}

	return ready
}

func (m *sortedMap) PopExceeds(threshold *big.Int) types.Transactions {
	var invalid types.Transactions
	size := len(m.items)
	for total := m.tree.root.sum; size > 0 && total.Cmp(threshold) > 0; {
		largest, err := m.tree.Largest()
		if err != nil {
			break
		}
		tx := m.items[largest]
		invalid = append(invalid, tx)
		m.tree.Remove(largest)
		delete(m.items, largest)
		size--
	}
	return invalid
}

func (m *sortedMap) Len() int {
	return len(m.items)
}

func (m *sortedMap) Flatten() types.Transactions {
	nodes := m.tree.Flatten()
	cache := make(types.Transactions, 0, len(m.items))
	for _, node := range nodes {
		cache = append(cache, m.items[node.key])
	}
	return cache
}

func (m *sortedMap) LastElement() *types.Transaction {
	last, err := m.tree.Largest()
	if err != nil {
		return nil
	}
	return m.items[last]
}

// list is a "list" of transactions belonging to an account, sorted by account
// nonce. The same type can be used both for storing contiguous transactions for
// the executable/pending queue; and for storing gapped transactions for the non-
// executable/future queue, with minor behavioral changes.
type list struct {
	strict bool       // Whether nonces are strictly continuous or not
	txs    *sortedMap // Heap indexed sorted hash map of the transactions

	costcap *big.Int // Price of the highest costing transaction (reset only if exceeds balance)
	gascap  uint64   // Gas limit of the highest spending transaction (reset only if exceeds block limit)
}

// newList create a new transaction list for maintaining nonce-indexable fast,
// gapped, sortable transaction Lists.
func newList(strict bool) *list {
	return &list{
		strict:  strict,
		txs:     newSortedMap(),
		costcap: new(big.Int),
	}
}

// Contains returns whether the  list contains a transaction
// with the provided nonce.
func (l *list) Contains(nonce uint64) bool {
	return l.txs.Get(nonce) != nil
}

func (l *list) GetCost(nonce uint64) *big.Int {
	return l.txs.GetCost(nonce)
}

// Add tries to insert a new transaction into the list, returning whether the
// transaction was accepted, and if yes, any previous transaction it replaced.
//
// If the new transaction is accepted into the list, the Lists' cost and gas
// thresholds are also potentially updated.
func (l *list) Add(tx *types.Transaction, priceBump uint64) (bool, *types.Transaction) {
	// If there's an older better transaction, abort
	old := l.txs.Get(tx.Nonce())
	if old != nil {
		if old.GasFeeCapCmp(tx) >= 0 || old.GasTipCapCmp(tx) >= 0 {
			return false, nil
		}
		// thresholdFeeCap = oldFC  * (100 + priceBump) / 100
		a := big.NewInt(100 + int64(priceBump))
		aFeeCap := new(big.Int).Mul(a, old.GasFeeCap())
		aTip := a.Mul(a, old.GasTipCap())

		// thresholdTip    = oldTip * (100 + priceBump) / 100
		b := big.NewInt(100)
		thresholdFeeCap := aFeeCap.Div(aFeeCap, b)
		thresholdTip := aTip.Div(aTip, b)

		// We have to ensure that both the new fee cap and tip are higher than the
		// old ones as well as checking the percentage threshold to ensure that
		// this is accurate for low (Wei-level) gas price replacements.
		if tx.GasFeeCapIntCmp(thresholdFeeCap) < 0 || tx.GasTipCapIntCmp(thresholdTip) < 0 {
			return false, nil
		}
	}
	// Otherwise overwrite the old transaction with the current one
	// and update the total cost, gas cap and cost cap
	l.txs.Put(tx)
	if cost := tx.Cost(); l.costcap.Cmp(cost) < 0 {
		l.costcap = cost
	}
	if gas := tx.Gas(); l.gascap < gas {
		l.gascap = gas
	}
	return true, old
}

// Forward removes all transactions from the list with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (l *list) Forward(threshold uint64) types.Transactions {
	txs := l.txs.Forward(threshold)
	return txs
}

// Filter removes all transactions from the list with a cost or gas limit higher
// than the provided thresholds. Every removed transaction is returned for any
// post-removal maintenance. Strict-mode invalidated transactions are also
// returned.
//
// This method uses the cached costcap and gascap to quickly decide if there's even
// a point in calculating all the costs or if the balance covers all. If the threshold
// is lower than the costgas cap, the caps will be reset to a new high after removing
// the newly invalidated transactions.
func (l *list) Filter(costLimit *big.Int, gasLimit uint64) (types.Transactions, types.Transactions) {
	// If all transactions are below the threshold, short circuit
	if l.costcap.Cmp(costLimit) <= 0 && l.gascap <= gasLimit {
		return nil, nil
	}
	l.costcap = new(big.Int).Set(costLimit) // Lower the caps to the thresholds
	l.gascap = gasLimit

	// Filter out all the transactions above the account's funds
	removed := l.txs.Filter(func(tx *types.Transaction) bool {
		return tx.Gas() > gasLimit || tx.Cost().Cmp(costLimit) > 0
	})

	if len(removed) == 0 {
		return nil, nil
	}
	var invalids types.Transactions
	// If the list was strict, filter anything above the lowest nonce
	if l.strict {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		// TODO: we can use LastElement() here, may be more efficient
		invalids = l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > lowest })
	}
	// Reset total cost
	return removed, invalids
}

// Cap places a hard limit on the number of items, returning all transactions
// exceeding that limit.
func (l *list) Cap(threshold int) types.Transactions {
	txs := l.txs.Cap(threshold)
	return txs
}

// Remove deletes a transaction from the maintained list, returning whether the
// transaction was found, and also returning any transaction invalidated due to
// the deletion (strict mode only).
func (l *list) Remove(tx *types.Transaction) (bool, types.Transactions) {
	// Remove the transaction from the set
	nonce := tx.Nonce()
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}
	// In strict mode, filter out non-executable transactions
	if l.strict {
		// TODO: we can use LastElement() here, may be more efficient
		txs := l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > nonce })
		return true, txs
	}
	return true, nil
}

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
// The start is the virtual nonce of the account, not the first nonce of it.
// Note, all transactions with nonces lower than start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (l *list) Ready(start uint64, threshold *big.Int) types.Transactions {
	txs := l.txs.Ready(start, threshold)
	return txs
}

func (l *list) PopExceeds(threshold *big.Int) types.Transactions {
	txs := l.txs.PopExceeds(threshold)
	return txs
}

// Len returns the length of the transaction list.
func (l *list) Len() int {
	return l.txs.Len()
}

// Empty returns whether the list of transactions is empty or not.
func (l *list) Empty() bool {
	return l.Len() == 0
}

// Flatten creates a nonce-sorted slice of transactions based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (l *list) Flatten() types.Transactions {
	return l.txs.Flatten()
}

// LastElement returns the last element of a flattened list, thus, the
// transaction with the highest nonce
func (l *list) LastElement() *types.Transaction {
	return l.txs.LastElement()
}

// priceHeap is a heap.Interface implementation over transactions for retrieving
// price-sorted transactions to discard when the pool fills up. If baseFee is set
// then the heap is sorted based on the effective tip based on the given base fee.
// If baseFee is nil then the sorting is based on gasFeeCap.
type priceHeap struct {
	baseFee *big.Int // heap should always be re-sorted after baseFee is changed
	list    []*types.Transaction
}

func (h *priceHeap) Len() int      { return len(h.list) }
func (h *priceHeap) Swap(i, j int) { h.list[i], h.list[j] = h.list[j], h.list[i] }

func (h *priceHeap) Less(i, j int) bool {
	switch h.cmp(h.list[i], h.list[j]) {
	case -1:
		return true
	case 1:
		return false
	default:
		return h.list[i].Nonce() > h.list[j].Nonce()
	}
}

func (h *priceHeap) cmp(a, b *types.Transaction) int {
	if h.baseFee != nil {
		// Compare effective tips if baseFee is specified
		if c := a.EffectiveGasTipCmp(b, h.baseFee); c != 0 {
			return c
		}
	}
	// Compare fee caps if baseFee is not specified or effective tips are equal
	if c := a.GasFeeCapCmp(b); c != 0 {
		return c
	}
	// Compare tips if effective tips and fee caps are equal
	return a.GasTipCapCmp(b)
}

func (h *priceHeap) Push(x interface{}) {
	tx := x.(*types.Transaction)
	h.list = append(h.list, tx)
}

func (h *priceHeap) Pop() interface{} {
	old := h.list
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	h.list = old[0 : n-1]
	return x
}

// pricedList is a price-sorted heap to allow operating on transactions pool
// contents in a price-incrementing way. It's built upon the all transactions
// in txpool but only interested in the remote part. It means only remote transactions
// will be considered for tracking, sorting, eviction, etc.
//
// Two heaps are used for sorting: the urgent heap (based on effective tip in the next
// block) and the floating heap (based on gasFeeCap). Always the bigger heap is chosen for
// eviction. Transactions evicted from the urgent heap are first demoted into the floating heap.
// In some cases (during a congestion, when blocks are full) the urgent heap can provide
// better candidates for inclusion while in other cases (at the top of the baseFee peak)
// the floating heap is better. When baseFee is decreasing they behave similarly.
type pricedList struct {
	// Number of stale price points to (re-heap trigger).
	stales atomic.Int64

	all              *lookup    // Pointer to the map of all transactions
	urgent, floating priceHeap  // Heaps of prices of all the stored **remote** transactions
	reheapMu         sync.Mutex // Mutex asserts that only one routine is reheaping the list
}

const (
	// urgentRatio : floatingRatio is the capacity ratio of the two queues
	urgentRatio   = 4
	floatingRatio = 1
)

// newPricedList creates a new price-sorted transaction heap.
func newPricedList(all *lookup) *pricedList {
	return &pricedList{
		all: all,
	}
}

// Put inserts a new transaction into the heap.
func (l *pricedList) Put(tx *types.Transaction, local bool) {
	if local {
		return
	}
	// Insert every new transaction to the urgent heap first; Discard will balance the heaps
	heap.Push(&l.urgent, tx)
}

// Removed notifies the prices transaction list that an old transaction dropped
// from the pool. The list will just keep a counter of stale objects and update
// the heap if a large enough ratio of transactions go stale.
func (l *pricedList) Removed(count int) {
	// Bump the stale counter, but exit if still too low (< 25%)
	stales := l.stales.Add(int64(count))
	if int(stales) <= (len(l.urgent.list)+len(l.floating.list))/4 {
		return
	}
	// Seems we've reached a critical number of stale transactions, reheap
	l.Reheap()
}

// Underpriced checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction currently being tracked.
func (l *pricedList) Underpriced(tx *types.Transaction) bool {
	// Note: with two queues, being underpriced is defined as being worse than the worst item
	// in all non-empty queues if there is any. If both queues are empty then nothing is underpriced.
	return (l.underpricedFor(&l.urgent, tx) || len(l.urgent.list) == 0) &&
		(l.underpricedFor(&l.floating, tx) || len(l.floating.list) == 0) &&
		(len(l.urgent.list) != 0 || len(l.floating.list) != 0)
}

// underpricedFor checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction in the given heap.
func (l *pricedList) underpricedFor(h *priceHeap, tx *types.Transaction) bool {
	// Discard stale price points if found at the heap start
	for len(h.list) > 0 {
		head := h.list[0]
		if l.all.GetRemote(head.Hash()) == nil { // Removed or migrated
			l.stales.Add(-1)
			heap.Pop(h)
			continue
		}
		break
	}
	// Check if the transaction is underpriced or not
	if len(h.list) == 0 {
		return false // There is no remote transaction at all.
	}
	// If the remote transaction is even cheaper than the
	// cheapest one tracked locally, reject it.
	return h.cmp(h.list[0], tx) >= 0
}

// Discard finds a number of most underpriced transactions, removes them from the
// priced list and returns them for further removal from the entire pool.
// If noPending is set to true, we will only consider the floating list
//
// Note local transaction won't be considered for eviction.
func (l *pricedList) Discard(slots int, force bool) (types.Transactions, bool) {
	drop := make(types.Transactions, 0, slots) // Remote underpriced transactions to drop
	for slots > 0 {
		if len(l.urgent.list)*floatingRatio > len(l.floating.list)*urgentRatio {
			// Discard stale transactions if found during cleanup
			tx := heap.Pop(&l.urgent).(*types.Transaction)
			if l.all.GetRemote(tx.Hash()) == nil { // Removed or migrated
				l.stales.Add(-1)
				continue
			}
			// Non stale transaction found, move to floating heap
			heap.Push(&l.floating, tx)
		} else {
			if len(l.floating.list) == 0 {
				// Stop if both heaps are empty
				break
			}
			// Discard stale transactions if found during cleanup
			tx := heap.Pop(&l.floating).(*types.Transaction)
			if l.all.GetRemote(tx.Hash()) == nil { // Removed or migrated
				l.stales.Add(-1)
				continue
			}
			// Non stale transaction found, discard it
			drop = append(drop, tx)
			slots -= numSlots(tx)
		}
	}
	// If we still can't make enough room for the new transaction
	if slots > 0 && !force {
		for _, tx := range drop {
			heap.Push(&l.urgent, tx)
		}
		return nil, false
	}
	return drop, true
}

// Reheap forcibly rebuilds the heap based on the current remote transaction set.
func (l *pricedList) Reheap() {
	l.reheapMu.Lock()
	defer l.reheapMu.Unlock()
	start := time.Now()
	l.stales.Store(0)
	l.urgent.list = make([]*types.Transaction, 0, l.all.RemoteCount())
	l.all.Range(func(_ common.Hash, tx *types.Transaction, _ bool) bool {
		l.urgent.list = append(l.urgent.list, tx)
		return true
	}, false, true) // Only iterate remotes
	heap.Init(&l.urgent)

	// balance out the two heaps by moving the worse half of transactions into the
	// floating heap
	// Note: Discard would also do this before the first eviction but Reheap can do
	// is more efficiently. Also, Underpriced would work suboptimally the first time
	// if the floating queue was empty.
	floatingCount := len(l.urgent.list) * floatingRatio / (urgentRatio + floatingRatio)
	l.floating.list = make([]*types.Transaction, floatingCount)
	for i := 0; i < floatingCount; i++ {
		l.floating.list[i] = heap.Pop(&l.urgent).(*types.Transaction)
	}
	heap.Init(&l.floating)
	reheapTimer.Update(time.Since(start))
}

// SetBaseFee updates the base fee and triggers a re-heap. Note that Removed is not
// necessary to call right before SetBaseFee when processing a new block.
func (l *pricedList) SetBaseFee(baseFee *big.Int) {
	l.urgent.baseFee = baseFee
	l.Reheap()
}
