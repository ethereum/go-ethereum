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
	"io"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

type lendingTradeState struct {
	orderBook common.Hash
	tradeId   common.Hash
	data      LendingTrade
	onDirty   func(orderId common.Hash) // Callback method to mark a state object newly dirty
}

func (lt *lendingTradeState) empty() bool {
	return lt.data.Amount.Sign() == 0
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
func (lt *lendingTradeState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, lt.data)
}

func (lt *lendingTradeState) deepCopy(onDirty func(orderId common.Hash)) *lendingTradeState {
	stateOrderList := newLendingTradeState(lt.orderBook, lt.tradeId, lt.data, onDirty)
	return stateOrderList
}

func (lt *lendingTradeState) SetCollateralLockedAmount(amount *big.Int) {
	lt.data.CollateralLockedAmount = amount
	if lt.onDirty != nil {
		lt.onDirty(lt.tradeId)
		lt.onDirty = nil
	}
}

func (lt *lendingTradeState) SetLiquidationPrice(price *big.Int) {
	lt.data.LiquidationPrice = price
	if lt.onDirty != nil {
		lt.onDirty(lt.tradeId)
		lt.onDirty = nil
	}
}

func (lt *lendingTradeState) SetAmount(amount *big.Int) {
	lt.data.Amount = amount
	if lt.onDirty != nil {
		lt.onDirty(lt.tradeId)
		lt.onDirty = nil
	}
}
