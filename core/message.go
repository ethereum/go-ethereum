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
	to         *common.Address
	from       common.Address
	nonce      uint64
	amount     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	gasFeeCap  *big.Int
	gasTipCap  *big.Int
	data       []byte
	accessList types.AccessList
	isFake     bool
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, data []byte, accessList types.AccessList, isFake bool) *Message {
	return &Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gasLimit,
		gasPrice:   gasPrice,
		gasFeeCap:  gasFeeCap,
		gasTipCap:  gasTipCap,
		data:       data,
		accessList: accessList,
		isFake:     isFake,
	}
}

// AsMessage returns the transaction as a core.Message.
func AsMessage(tx *types.Transaction, s types.Signer, baseFee *big.Int) (*Message, error) {
	msg := &Message{
		nonce:      tx.Nonce(),
		gasLimit:   tx.Gas(),
		gasPrice:   new(big.Int).Set(tx.GasPrice()),
		gasFeeCap:  new(big.Int).Set(tx.GasFeeCap()),
		gasTipCap:  new(big.Int).Set(tx.GasTipCap()),
		to:         tx.To(),
		amount:     tx.Value(),
		data:       tx.Data(),
		accessList: tx.AccessList(),
		isFake:     false,
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	if baseFee != nil {
		msg.gasPrice = math.BigMin(msg.gasPrice.Add(msg.gasTipCap, baseFee), msg.gasFeeCap)
	}
	var err error
	msg.from, err = types.Sender(s, tx)
	return msg, err
}

// TODO: Get rid of these accessor methods. Message should remain a simple data-only struct whose
// values are accessed directly.
func (m *Message) From() common.Address         { return m.from }
func (m *Message) To() *common.Address          { return m.to }
func (m *Message) GasPrice() *big.Int           { return m.gasPrice }
func (m *Message) GasFeeCap() *big.Int          { return m.gasFeeCap }
func (m *Message) GasTipCap() *big.Int          { return m.gasTipCap }
func (m *Message) Value() *big.Int              { return m.amount }
func (m *Message) Gas() uint64                  { return m.gasLimit }
func (m *Message) Nonce() uint64                { return m.nonce }
func (m *Message) Data() []byte                 { return m.data }
func (m *Message) AccessList() types.AccessList { return m.accessList }
func (m *Message) IsFake() bool                 { return m.isFake }
