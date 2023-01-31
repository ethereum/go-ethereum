// Copyright 2022 The go-ethereum Authors
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
// Message is a fully derived transaction and implements core.Message

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
)

// A Message contains the data derived from a single transaction that is relevant to state
// processing.
type Message struct {
	To         *common.Address
	From       common.Address
	Nonce      uint64
	Value      *big.Int
	GasLimit   uint64
	GasPrice   *big.Int
	GasFeeCap  *big.Int
	GasTipCap  *big.Int
	Data       []byte
	AccessList types.AccessList
	IsFake     bool
}

// AsMessage returns the transaction as a core.Message.
func AsMessage(tx *types.Transaction, s types.Signer, baseFee *big.Int) (*Message, error) {
	msg := &Message{
		Nonce:      tx.Nonce(),
		GasLimit:   tx.Gas(),
		GasPrice:   new(big.Int).Set(tx.GasPrice()),
		GasFeeCap:  new(big.Int).Set(tx.GasFeeCap()),
		GasTipCap:  new(big.Int).Set(tx.GasTipCap()),
		To:         tx.To(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
		IsFake:     false,
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	if baseFee != nil {
		msg.GasPrice = math.BigMin(msg.GasPrice.Add(msg.GasTipCap, baseFee), msg.GasFeeCap)
	}
	var err error
	msg.From, err = types.Sender(s, tx)
	return msg, err
}
