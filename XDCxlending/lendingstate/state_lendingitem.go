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

type lendingItemState struct {
	orderBook common.Hash
	orderId   common.Hash
	data      LendingItem
	onDirty   func(orderId common.Hash) // Callback method to mark a state object newly dirty
}

func (s *lendingItemState) empty() bool {
	return s.data.Quantity == nil || s.data.Quantity.Cmp(Zero) == 0
}

func newLendinItemState(orderBook common.Hash, orderId common.Hash, data LendingItem, onDirty func(orderId common.Hash)) *lendingItemState {
	return &lendingItemState{
		orderBook: orderBook,
		orderId:   orderId,
		data:      data,
		onDirty:   onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *lendingItemState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (self *lendingItemState) deepCopy(onDirty func(orderId common.Hash)) *lendingItemState {
	stateOrderList := newLendinItemState(self.orderBook, self.orderId, self.data, onDirty)
	return stateOrderList
}

func (self *lendingItemState) setVolume(volume *big.Int) {
	self.data.Quantity = volume
	if self.onDirty != nil {
		self.onDirty(self.orderId)
		self.onDirty = nil
	}
}

func (self *lendingItemState) Quantity() *big.Int {
	return self.data.Quantity
}
