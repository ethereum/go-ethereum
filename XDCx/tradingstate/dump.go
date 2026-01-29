// Copyright 2019 XDC Network
// This file is part of the XDC library.

package tradingstate

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// DumpOrder represents an order for JSON export
type DumpOrder struct {
	ID              string `json:"id"`
	UserAddress     string `json:"userAddress"`
	ExchangeAddress string `json:"exchangeAddress"`
	BaseToken       string `json:"baseToken"`
	QuoteToken      string `json:"quoteToken"`
	Side            string `json:"side"`
	Type            string `json:"type"`
	Price           string `json:"price"`
	Quantity        string `json:"quantity"`
	FilledQuantity  string `json:"filledQuantity"`
	Status          string `json:"status"`
	Nonce           uint64 `json:"nonce"`
	Timestamp       uint64 `json:"timestamp"`
}

// DumpOrderBook represents an order book for JSON export
type DumpOrderBook struct {
	PairKey string       `json:"pairKey"`
	Bids    []*DumpOrder `json:"bids"`
	Asks    []*DumpOrder `json:"asks"`
}

// DumpAccount represents an account for JSON export
type DumpAccount struct {
	Address string `json:"address"`
	Nonce   uint64 `json:"nonce"`
	Balance string `json:"balance"`
}

// Dump represents the state dump
type Dump struct {
	Root       string           `json:"root"`
	Accounts   []*DumpAccount   `json:"accounts"`
	OrderBooks []*DumpOrderBook `json:"orderBooks"`
	Orders     []*DumpOrder     `json:"orders"`
}

// RawDump returns a raw state dump
func (s *TradingStateDB) RawDump() *Dump {
	s.lock.Lock()
	defer s.lock.Unlock()

	dump := &Dump{
		Root:       s.trie.Hash().Hex(),
		Accounts:   make([]*DumpAccount, 0),
		OrderBooks: make([]*DumpOrderBook, 0),
		Orders:     make([]*DumpOrder, 0),
	}

	// Dump accounts
	for _, obj := range s.stateObjects {
		dump.Accounts = append(dump.Accounts, &DumpAccount{
			Address: obj.key.Hex(),
			Nonce:   obj.data.Nonce,
			Balance: obj.data.Balance.String(),
		})
	}

	// Dump order books
	for pairKey, ob := range s.orderBooks {
		dumpOB := &DumpOrderBook{
			PairKey: pairKey.Hex(),
			Bids:    make([]*DumpOrder, 0),
			Asks:    make([]*DumpOrder, 0),
		}

		for _, bid := range ob.Bids {
			dumpOB.Bids = append(dumpOB.Bids, dumpOrderState(bid))
		}

		for _, ask := range ob.Asks {
			dumpOB.Asks = append(dumpOB.Asks, dumpOrderState(ask))
		}

		dump.OrderBooks = append(dump.OrderBooks, dumpOB)
	}

	// Dump orders
	for _, order := range s.orders {
		dump.Orders = append(dump.Orders, dumpOrderState(order))
	}

	return dump
}

// Dump returns a JSON encoded state dump
func (s *TradingStateDB) Dump() ([]byte, error) {
	dump := s.RawDump()
	return json.MarshalIndent(dump, "", "  ")
}

// dumpOrderState converts an order state to a dump order
func dumpOrderState(order *OrderState) *DumpOrder {
	sideStr := "buy"
	if order.Side == 1 {
		sideStr = "sell"
	}

	typeStr := "limit"
	if order.Type == 1 {
		typeStr = "market"
	}

	statusStr := "new"
	switch order.Status {
	case 1:
		statusStr = "partial"
	case 2:
		statusStr = "filled"
	case 3:
		statusStr = "cancelled"
	case 4:
		statusStr = "rejected"
	}

	return &DumpOrder{
		ID:              order.ID.Hex(),
		UserAddress:     order.UserAddress.Hex(),
		ExchangeAddress: order.ExchangeAddress.Hex(),
		BaseToken:       order.BaseToken.Hex(),
		QuoteToken:      order.QuoteToken.Hex(),
		Side:            sideStr,
		Type:            typeStr,
		Price:           order.Price.String(),
		Quantity:        order.Quantity.String(),
		FilledQuantity:  order.FilledQuantity.String(),
		Status:          statusStr,
		Nonce:           order.Nonce,
		Timestamp:       order.Timestamp,
	}
}

// GetTradingPairs returns all trading pairs
func (s *TradingStateDB) GetTradingPairs() []common.Hash {
	s.lock.Lock()
	defer s.lock.Unlock()

	pairs := make([]common.Hash, 0, len(s.orderBooks))
	for pairKey := range s.orderBooks {
		pairs = append(pairs, pairKey)
	}
	return pairs
}

// GetAllOrders returns all orders
func (s *TradingStateDB) GetAllOrders() []*OrderState {
	s.lock.Lock()
	defer s.lock.Unlock()

	orders := make([]*OrderState, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order.Copy())
	}
	return orders
}

// GetOrdersByUser returns all orders for a specific user
func (s *TradingStateDB) GetOrdersByUser(userAddress common.Address) []*OrderState {
	s.lock.Lock()
	defer s.lock.Unlock()

	orders := make([]*OrderState, 0)
	for _, order := range s.orders {
		if order.UserAddress == userAddress {
			orders = append(orders, order.Copy())
		}
	}
	return orders
}

// GetOrdersByPair returns all orders for a specific trading pair
func (s *TradingStateDB) GetOrdersByPair(pairKey common.Hash) []*OrderState {
	s.lock.Lock()
	defer s.lock.Unlock()

	ob := s.orderBooks[pairKey]
	if ob == nil {
		return nil
	}

	orders := make([]*OrderState, 0, len(ob.Bids)+len(ob.Asks))
	for _, bid := range ob.Bids {
		orders = append(orders, bid.Copy())
	}
	for _, ask := range ob.Asks {
		orders = append(orders, ask.Copy())
	}
	return orders
}

// GetVolume returns the total volume for a trading pair
func (s *TradingStateDB) GetVolume(pairKey common.Hash) (*big.Int, *big.Int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ob := s.orderBooks[pairKey]
	if ob == nil {
		return big.NewInt(0), big.NewInt(0)
	}

	return ob.GetBidVolume(), ob.GetAskVolume()
}
