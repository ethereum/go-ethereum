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

package lendingstate

import (
	"github.com/XinFinOrg/XDPoSChain/common"
	"math/big"
)

type journalEntry interface {
	undo(db *LendingStateDB)
}

type journal []journalEntry

type (
	// Changes to the account trie.
	insertOrder struct {
		orderBook common.Hash
		orderId   common.Hash
		order     *LendingItem
	}
	insertTrading struct {
		orderBook common.Hash
		tradeId   uint64
		prvTrade  *LendingTrade
	}
	cancelOrder struct {
		orderBook common.Hash
		orderId   common.Hash
		order     LendingItem
	}
	cancelTrading struct {
		orderBook common.Hash
		tradeId   uint64
		order     LendingTrade
	}
	subAmountOrder struct {
		orderBook common.Hash
		orderId   common.Hash
		order     LendingItem
		amount    *big.Int
	}
	nonceChange struct {
		hash common.Hash
		prev uint64
	}
	tradeNonceChange struct {
		hash common.Hash
		prev uint64
	}
	liquidationPriceChange struct {
		orderBook common.Hash
		tradeId   common.Hash
		prev      *big.Int
	}
	collateralLockedAmount struct {
		orderBook common.Hash
		tradeId   common.Hash
		prev      *big.Int
	}
)

func (ch insertOrder) undo(s *LendingStateDB) {
	s.CancelLendingOrder(ch.orderBook, ch.order)
}
func (ch cancelOrder) undo(s *LendingStateDB) {
	s.InsertLendingItem(ch.orderBook, ch.orderId, ch.order)
}
func (ch subAmountOrder) undo(s *LendingStateDB) {
	interestHash := common.BigToHash(ch.order.Interest)
	stateOrderBook := s.getLendingExchange(ch.orderBook)
	var stateOrderList *itemListState
	switch ch.order.Side {
	case Investing:
		stateOrderList = stateOrderBook.getInvestingOrderList(s.db, interestHash)
	case Borrowing:
		stateOrderList = stateOrderBook.getBorrowingOrderList(s.db, interestHash)
	default:
		return
	}
	stateOrderItem := stateOrderBook.getLendingItem(s.db, ch.orderId)
	newAmount := new(big.Int).Add(stateOrderItem.Quantity(), ch.amount)
	stateOrderItem.setVolume(newAmount)
	stateOrderList.insertLendingItem(s.db, ch.orderId, common.BigToHash(newAmount))
	stateOrderList.AddVolume(ch.amount)
}
func (ch nonceChange) undo(s *LendingStateDB) {
	s.SetNonce(ch.hash, ch.prev)
}

func (ch tradeNonceChange) undo(s *LendingStateDB) {
	s.SetTradeNonce(ch.hash, ch.prev)
}
func (ch cancelTrading) undo(s *LendingStateDB) {
	s.InsertTradingItem(ch.orderBook, ch.tradeId, ch.order)
}
func (ch insertTrading) undo(s *LendingStateDB) {
	s.InsertTradingItem(ch.orderBook, ch.tradeId, *ch.prvTrade)
}

func (ch liquidationPriceChange) undo(s *LendingStateDB) {
	stateOrderBook := s.getLendingExchange(ch.orderBook)
	if stateOrderBook == nil {
		return
	}
	stateLendingTrade := stateOrderBook.getLendingTrade(s.db, ch.tradeId)
	if stateLendingTrade == nil {
		return
	}
	stateLendingTrade.SetLiquidationPrice(ch.prev)
}

func (ch collateralLockedAmount) undo(s *LendingStateDB) {
	stateOrderBook := s.getLendingExchange(ch.orderBook)
	if stateOrderBook == nil {
		return
	}
	stateLendingTrade := stateOrderBook.getLendingTrade(s.db, ch.tradeId)
	if stateLendingTrade == nil {
		return
	}
	stateLendingTrade.SetCollateralLockedAmount(ch.prev)
}
