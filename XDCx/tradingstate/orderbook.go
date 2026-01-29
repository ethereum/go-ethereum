// Copyright 2019 XDC Network
// This file is part of the XDC library.

package tradingstate

import (
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Order interface for XDCx orders
type Order interface {
	GetID() common.Hash
	GetPrice() *big.Int
	GetQuantity() *big.Int
	GetFilledQuantity() *big.Int
	GetSide() uint8
	RemainingQuantity() *big.Int
	IsFilled() bool
}

// OrderBook represents a trading pair order book
type OrderBook struct {
	PairKey common.Hash
	Bids    []*OrderState // Buy orders sorted by price desc
	Asks    []*OrderState // Sell orders sorted by price asc

	bidIndex map[common.Hash]int
	askIndex map[common.Hash]int

	lock sync.RWMutex
}

// NewOrderBook creates a new order book
func NewOrderBook(pairKey common.Hash) *OrderBook {
	return &OrderBook{
		PairKey:  pairKey,
		Bids:     make([]*OrderState, 0),
		Asks:     make([]*OrderState, 0),
		bidIndex: make(map[common.Hash]int),
		askIndex: make(map[common.Hash]int),
	}
}

// AddBid adds a buy order to the order book
func (ob *OrderBook) AddBid(order interface{}) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	var orderState *OrderState
	switch o := order.(type) {
	case *OrderState:
		orderState = o
	default:
		return
	}

	ob.Bids = append(ob.Bids, orderState)
	ob.sortBids()
	ob.rebuildBidIndex()
}

// AddAsk adds a sell order to the order book
func (ob *OrderBook) AddAsk(order interface{}) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	var orderState *OrderState
	switch o := order.(type) {
	case *OrderState:
		orderState = o
	default:
		return
	}

	ob.Asks = append(ob.Asks, orderState)
	ob.sortAsks()
	ob.rebuildAskIndex()
}

// RemoveBid removes a buy order from the order book
func (ob *OrderBook) RemoveBid(orderID common.Hash) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	if idx, exists := ob.bidIndex[orderID]; exists {
		ob.Bids = append(ob.Bids[:idx], ob.Bids[idx+1:]...)
		ob.rebuildBidIndex()
	}
}

// RemoveAsk removes a sell order from the order book
func (ob *OrderBook) RemoveAsk(orderID common.Hash) {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	if idx, exists := ob.askIndex[orderID]; exists {
		ob.Asks = append(ob.Asks[:idx], ob.Asks[idx+1:]...)
		ob.rebuildAskIndex()
	}
}

// GetBestBid returns the best bid order
func (ob *OrderBook) GetBestBid() *OrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if len(ob.Bids) == 0 {
		return nil
	}
	return ob.Bids[0]
}

// GetBestAsk returns the best ask order
func (ob *OrderBook) GetBestAsk() *OrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if len(ob.Asks) == 0 {
		return nil
	}
	return ob.Asks[0]
}

// GetBid returns a specific bid order
func (ob *OrderBook) GetBid(orderID common.Hash) *OrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if idx, exists := ob.bidIndex[orderID]; exists {
		return ob.Bids[idx]
	}
	return nil
}

// GetAsk returns a specific ask order
func (ob *OrderBook) GetAsk(orderID common.Hash) *OrderState {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	if idx, exists := ob.askIndex[orderID]; exists {
		return ob.Asks[idx]
	}
	return nil
}

// BidCount returns the number of bids
func (ob *OrderBook) BidCount() int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()
	return len(ob.Bids)
}

// AskCount returns the number of asks
func (ob *OrderBook) AskCount() int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()
	return len(ob.Asks)
}

// GetBidVolume returns total bid volume
func (ob *OrderBook) GetBidVolume() *big.Int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	total := big.NewInt(0)
	for _, bid := range ob.Bids {
		total = new(big.Int).Add(total, bid.RemainingQuantity())
	}
	return total
}

// GetAskVolume returns total ask volume
func (ob *OrderBook) GetAskVolume() *big.Int {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	total := big.NewInt(0)
	for _, ask := range ob.Asks {
		total = new(big.Int).Add(total, ask.RemainingQuantity())
	}
	return total
}

// Copy creates a copy of the order book
func (ob *OrderBook) Copy() *OrderBook {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	copy := &OrderBook{
		PairKey:  ob.PairKey,
		Bids:     make([]*OrderState, len(ob.Bids)),
		Asks:     make([]*OrderState, len(ob.Asks)),
		bidIndex: make(map[common.Hash]int),
		askIndex: make(map[common.Hash]int),
	}

	for i, bid := range ob.Bids {
		copy.Bids[i] = bid.Copy()
		copy.bidIndex[bid.ID] = i
	}

	for i, ask := range ob.Asks {
		copy.Asks[i] = ask.Copy()
		copy.askIndex[ask.ID] = i
	}

	return copy
}

// sortBids sorts bids by price descending
func (ob *OrderBook) sortBids() {
	sort.Slice(ob.Bids, func(i, j int) bool {
		return ob.Bids[i].Price.Cmp(ob.Bids[j].Price) > 0
	})
}

// sortAsks sorts asks by price ascending
func (ob *OrderBook) sortAsks() {
	sort.Slice(ob.Asks, func(i, j int) bool {
		return ob.Asks[i].Price.Cmp(ob.Asks[j].Price) < 0
	})
}

// rebuildBidIndex rebuilds the bid index
func (ob *OrderBook) rebuildBidIndex() {
	ob.bidIndex = make(map[common.Hash]int)
	for i, bid := range ob.Bids {
		ob.bidIndex[bid.ID] = i
	}
}

// rebuildAskIndex rebuilds the ask index
func (ob *OrderBook) rebuildAskIndex() {
	ob.askIndex = make(map[common.Hash]int)
	for i, ask := range ob.Asks {
		ob.askIndex[ask.ID] = i
	}
}
