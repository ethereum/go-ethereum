// Copyright 2021 The go-ethereum Authors
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

package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type DynamicFeeTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int
	Data       []byte
	AccessList AccessList

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DynamicFeeTx) copy() TxData {
	cpy := &DynamicFeeTx{
		Nonce:      tx.Nonce,
		To:         tx.To, // TODO: copy pointed-to address
		Data:       common.CopyBytes(tx.Data),
		Gas:        tx.Gas,
		AccessList: make(AccessList, len(tx.AccessList)),
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.Value != nil {
		cpy.Value = new(big.Int).Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID = new(big.Int).Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap = new(big.Int).Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap = new(big.Int).Set(tx.GasFeeCap)
	}
	if tx.V != nil {
		cpy.V = new(big.Int).Set(tx.V)
	}
	if tx.R != nil {
		cpy.R = new(big.Int).Set(tx.R)
	}
	if tx.S != nil {
		cpy.S = new(big.Int).Set(tx.S)
	}
	return cpy
}

// accessors for innerTx.
func (tx *DynamicFeeTx) txType() byte           { return DynamicFeeTxType }
func (tx *DynamicFeeTx) chainID() *big.Int      { return tx.ChainID }
func (tx *DynamicFeeTx) protected() bool        { return true }
func (tx *DynamicFeeTx) accessList() AccessList { return tx.AccessList }
func (tx *DynamicFeeTx) data() []byte           { return tx.Data }
func (tx *DynamicFeeTx) gas() uint64            { return tx.Gas }
func (tx *DynamicFeeTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *DynamicFeeTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *DynamicFeeTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *DynamicFeeTx) value() *big.Int        { return tx.Value }
func (tx *DynamicFeeTx) nonce() uint64          { return tx.Nonce }
func (tx *DynamicFeeTx) to() *common.Address    { return tx.To }

func (tx *DynamicFeeTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *DynamicFeeTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}
