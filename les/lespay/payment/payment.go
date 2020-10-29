// Copyright 2020 The go-ethereum Authors
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

package payment

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"
)

// CurrentHeader retrieves the current header from the local chain.
type ChainReader interface {
	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header

	// GetHeaderByNumber retrieves a block header from the database by number.
	GetHeaderByNumber(number uint64) *types.Header

	// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

// PaymentPacket represents a network packet for payment.
type PaymentPacket struct {
	Identity       string
	ProofOfPayment rlp.RawValue
}

type Schema interface {
	// Identity returns string format identity of payment method.
	Identity() string

	// Load loads the specified entry with given key, returns error
	// if nothing is found.
	Load(key string) (interface{}, error)
}

type SchemaRLP struct {
	Key   string
	Value rlp.RawValue
}
