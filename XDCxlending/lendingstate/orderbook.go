// Copyright 2019 XDC Network
// This file is part of the XDC library.

package lendingstate

import (
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// LendingOrderBook represents a lending pair order book
type LendingOrderBook struct {
	PairKey  common.Hash
	Borrows  []*LendingOrderState // Borrow orders sorted by interest rate desc
	Lends    []*LendingOrderState // Lend orders sorted by interest rate asc

	borrowIndex map[common.Hash]int
	lendIndex   map[common.Hash]int

	lock sync.RWMutex
}

// NewLendingOrderBook creates a new lending order book
func NewLendingOrderBook(pairKey common.Hash) *LendingOrderBook {
	return &LendingOrderBook{
		PairKey:     pairKey,
		Borrows:     make([]*LendingOrderState, 0),
		Lends:       make([]*LendingOrderState, 0),
		borrowIndex: make(map[common.Hash]int),
		lendIndex:   make(map[common.Hash]int),
	}
}

// AddBorrow adds a borrow order to the order book
func (ob *LendingOrderBook) AddBorrow(order interface{}) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	var orderState *LendingOrderState
	switch o := order.(type) {
	case *LendingOrderState:
		orderState = o
	default:
		return
	}

	ob.Borrows = append(ob.Borrows, orderState)
	ob.sortBorrows()
	ob.rebuildBorrowIndex()
}

// AddLend adds a lend order to the order book
func (ob *LendingOrderBook) AddLend(order interface{}) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	var orderState *LendingOrderState
	switch o := order.(type) {
	case *LendingOrderState:
		orderState = o
	default:
		return
	}

	ob.Lends = append(ob.Lends, orderState)
	ob.sortLends()
	ob.rebuildLendIndex()
}

// RemoveBorrow removes a borrow order from the order book
func (ob *LendingOrderBook) RemoveBorrow(orderID common.Hash) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	if idx, exists := ob.borrowIndex[orderID]; exists {
		ob.Borrows = append(ob.Borrows[:idx], ob.Borrows[idx+1:]...)
		ob.rebuildBorrowIndex()
	}
}

// RemoveLend removes a lend order from the order book
func (ob *LendingOrderBook) RemoveLend(orderID common.Hash) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	if idx, exists := ob.lendIndex[orderID]; exists {
		ob.Lends = append(ob.Lends[:idx], ob.Lends[idx+1:]...)
		ob.rebuildLendIndex()
	}
}

// GetBestBorrow returns the best borrow order (highest interest rate)
func (ob *LendingOrderBook) GetBestBorrow() *LendingOrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if len(ob.Borrows) == 0 {
		return nil
	}
	return ob.Borrows[0]
}

// GetBestLend returns the best lend order (lowest interest rate)
func (ob *LendingOrderBook) GetBestLend() *LendingOrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if len(ob.Lends) == 0 {
		return nil
	}
	return ob.Lends[0]
}

// GetBorrow returns a specific borrow order
func (ob *LendingOrderBook) GetBorrow(orderID common.Hash) *LendingOrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if idx, exists := ob.borrowIndex[orderID]; exists {
		return ob.Borrows[idx]
	}
	return nil
}

// GetLend returns a specific lend order
func (ob *LendingOrderBook) GetLend(orderID common.Hash) *LendingOrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if idx, exists := ob.lendIndex[orderID]; exists {
		return ob.Lends[idx]
	}
	return nil
}

// BorrowCount returns the number of borrow orders
func (ob *LendingOrderBook) BorrowCount() int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()
	return len(ob.Borrows)
}

// LendCount returns the number of lend orders
func (ob *LendingOrderBook) LendCount() int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()
	return len(ob.Lends)
}

// GetBorrowVolume returns total borrow volume
func (ob *LendingOrderBook) GetBorrowVolume() *big.Int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	total := big.NewInt(0)
	for _, borrow := range ob.Borrows {
		total = new(big.Int).Add(total, borrow.RemainingQuantity())
	}
	return total
}

// GetLendVolume returns total lend volume
func (ob *LendingOrderBook) GetLendVolume() *big.Int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	total := big.NewInt(0)
	for _, lend := range ob.Lends {
		total = new(big.Int).Add(total, lend.RemainingQuantity())
	}
	return total
}

// Copy creates a copy of the order book
func (ob *LendingOrderBook) Copy() *LendingOrderBook {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	copy := &LendingOrderBook{
		PairKey:     ob.PairKey,
		Borrows:     make([]*LendingOrderState, len(ob.Borrows)),
		Lends:       make([]*LendingOrderState, len(ob.Lends)),
		borrowIndex: make(map[common.Hash]int),
		lendIndex:   make(map[common.Hash]int),
	}

	for i, borrow := range ob.Borrows {
		copy.Borrows[i] = borrow.Copy()
		copy.borrowIndex[borrow.ID] = i
	}

	for i, lend := range ob.Lends {
		copy.Lends[i] = lend.Copy()
		copy.lendIndex[lend.ID] = i
	}

	return copy
}

// sortBorrows sorts borrows by interest rate descending
func (ob *LendingOrderBook) sortBorrows() {
	sort.Slice(ob.Borrows, func(i, j int) bool {
		return ob.Borrows[i].InterestRate.Cmp(ob.Borrows[j].InterestRate) > 0
	})
}

// sortLends sorts lends by interest rate ascending
func (ob *LendingOrderBook) sortLends() {
	sort.Slice(ob.Lends, func(i, j int) bool {
		return ob.Lends[i].InterestRate.Cmp(ob.Lends[j].InterestRate) < 0
	})
}

// rebuildBorrowIndex rebuilds the borrow index
func (ob *LendingOrderBook) rebuildBorrowIndex() {
	ob.borrowIndex = make(map[common.Hash]int)
	for i, borrow := range ob.Borrows {
		ob.borrowIndex[borrow.ID] = i
	}
}

// rebuildLendIndex rebuilds the lend index
func (ob *LendingOrderBook) rebuildLendIndex() {
	ob.lendIndex = make(map[common.Hash]int)
	for i, lend := range ob.Lends {
		ob.lendIndex[lend.ID] = i
	}
}

// GetMarketRate returns the market interest rate
func (ob *LendingOrderBook) GetMarketRate() *big.Int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	bestBorrow := ob.GetBestBorrow()
	bestLend := ob.GetBestLend()

	if bestBorrow == nil && bestLend == nil {
		return nil
	}

	if bestBorrow == nil {
		return new(big.Int).Set(bestLend.InterestRate)
	}

	if bestLend == nil {
		return new(big.Int).Set(bestBorrow.InterestRate)
	}

	// Return midpoint
	sum := new(big.Int).Add(bestBorrow.InterestRate, bestLend.InterestRate)
	return sum.Div(sum, big.NewInt(2))
}
