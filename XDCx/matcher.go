// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCx

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/XDCx/tradingstate"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// ErrOrderBookNotFound is returned when order book is not found
	ErrOrderBookNotFound = errors.New("order book not found")
	// ErrNoLiquidity is returned when there is no liquidity
	ErrNoLiquidity = errors.New("no liquidity")
)

// OrderBook represents a trading order book
type OrderBook struct {
	BaseToken  common.Address
	QuoteToken common.Address
	Bids       []*Order // Buy orders sorted by price desc
	Asks       []*Order // Sell orders sorted by price asc
}

// Matcher handles order matching
type Matcher struct {
	xdcx *XDCx
	lock sync.RWMutex
}

// NewMatcher creates a new matcher
func NewMatcher(xdcx *XDCx) *Matcher {
	return &Matcher{
		xdcx: xdcx,
	}
}

// GetOrderBook returns the order book for a trading pair
func (m *Matcher) GetOrderBook(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*OrderBook, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	pairKey := GetPairKey(baseToken, quoteToken)
	stateOrderBook := tradingState.GetOrderBook(pairKey)
	if stateOrderBook == nil {
		return nil, ErrOrderBookNotFound
	}

	orderBook := &OrderBook{
		BaseToken:  baseToken,
		QuoteToken: quoteToken,
		Bids:       make([]*Order, 0),
		Asks:       make([]*Order, 0),
	}

	// Convert state order book to API order book
	for _, bid := range stateOrderBook.Bids {
		orderBook.Bids = append(orderBook.Bids, bid.Clone())
	}
	for _, ask := range stateOrderBook.Asks {
		orderBook.Asks = append(orderBook.Asks, ask.Clone())
	}

	return orderBook, nil
}

// GetBestBid returns the best bid price
func (m *Matcher) GetBestBid(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*big.Int, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	pairKey := GetPairKey(baseToken, quoteToken)
	orderBook := tradingState.GetOrderBook(pairKey)
	if orderBook == nil {
		return nil, ErrOrderBookNotFound
	}

	bestBid := orderBook.GetBestBid()
	if bestBid == nil {
		return nil, ErrNoLiquidity
	}

	return new(big.Int).Set(bestBid.Price), nil
}

// GetBestAsk returns the best ask price
func (m *Matcher) GetBestAsk(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*big.Int, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	pairKey := GetPairKey(baseToken, quoteToken)
	orderBook := tradingState.GetOrderBook(pairKey)
	if orderBook == nil {
		return nil, ErrOrderBookNotFound
	}

	bestAsk := orderBook.GetBestAsk()
	if bestAsk == nil {
		return nil, ErrNoLiquidity
	}

	return new(big.Int).Set(bestAsk.Price), nil
}

// GetSpread returns the bid-ask spread
func (m *Matcher) GetSpread(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*big.Int, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	bestBid, err := m.GetBestBid(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	bestAsk, err := m.GetBestAsk(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	spread := new(big.Int).Sub(bestAsk, bestBid)
	return spread, nil
}

// GetMidPrice returns the mid price
func (m *Matcher) GetMidPrice(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*big.Int, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	bestBid, err := m.GetBestBid(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	bestAsk, err := m.GetBestAsk(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	midPrice := new(big.Int).Add(bestBid, bestAsk)
	midPrice = midPrice.Div(midPrice, big.NewInt(2))
	return midPrice, nil
}

// GetDepth returns the order book depth at each price level
func (m *Matcher) GetDepth(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB, levels int) (*Depth, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	pairKey := GetPairKey(baseToken, quoteToken)
	orderBook := tradingState.GetOrderBook(pairKey)
	if orderBook == nil {
		return nil, ErrOrderBookNotFound
	}

	depth := &Depth{
		Bids: make([]PriceLevel, 0),
		Asks: make([]PriceLevel, 0),
	}

	// Aggregate bids by price
	bidPrices := make(map[string]*big.Int)
	for _, bid := range orderBook.Bids {
		key := bid.Price.String()
		if _, exists := bidPrices[key]; !exists {
			bidPrices[key] = big.NewInt(0)
		}
		bidPrices[key] = new(big.Int).Add(bidPrices[key], bid.RemainingQuantity())
	}

	// Aggregate asks by price
	askPrices := make(map[string]*big.Int)
	for _, ask := range orderBook.Asks {
		key := ask.Price.String()
		if _, exists := askPrices[key]; !exists {
			askPrices[key] = big.NewInt(0)
		}
		askPrices[key] = new(big.Int).Add(askPrices[key], ask.RemainingQuantity())
	}

	return depth, nil
}

// Depth represents order book depth
type Depth struct {
	Bids []PriceLevel
	Asks []PriceLevel
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    *big.Int
	Quantity *big.Int
}
