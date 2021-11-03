// Copyright 2014 The go-ethereum Authors
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
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"io"
	"math/big"
)

type lendingTradeState struct {
	orderBook common.Hash
	tradeId   common.Hash
	data      LendingTrade
	onDirty   func(orderId common.Hash) // Callback method to mark a state object newly dirty
}

func (s *lendingTradeState) empty() bool {
	return s.data.Amount.Sign() == 0
}

func newLendingTradeState(orderBook common.Hash, tradeId common.Hash, data LendingTrade, onDirty func(orderId common.Hash)) *lendingTradeState {
	return &lendingTradeState{
		orderBook: orderBook,
		tradeId:   tradeId,
		data:      data,
		onDirty:   onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *lendingTradeState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (self *lendingTradeState) deepCopy(onDirty func(orderId common.Hash)) *lendingTradeState {
	stateOrderList := newLendingTradeState(self.orderBook, self.tradeId, self.data, onDirty)
	return stateOrderList
}

func (self *lendingTradeState) SetCollateralLockedAmount(amount *big.Int) {
	self.data.CollateralLockedAmount = amount
	if self.onDirty != nil {
		self.onDirty(self.tradeId)
		self.onDirty = nil
	}
}

func (self *lendingTradeState) SetLiquidationPrice(price *big.Int) {
	self.data.LiquidationPrice = price
	if self.onDirty != nil {
		self.onDirty(self.tradeId)
		self.onDirty = nil
	}
}

func (self *lendingTradeState) SetAmount(amount *big.Int) {
	self.data.Amount = amount
	if self.onDirty != nil {
		self.onDirty(self.tradeId)
		self.onDirty = nil
	}
}
