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
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// DelegateTx represents an EIP-5806 transaction.
type DelegateTx struct {
	ChainID    *big.Int        // destination chain ID
	Nonce      uint64          // nonce of sender account
	GasPrice   *big.Int        // wei per gas
	Gas        uint64          // gas limit
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Data       []byte          // contract invocation input data
	AccessList AccessList      // EIP-2930 access list
	V, R, S    *big.Int        // signature values
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DelegateTx) copy() TxData {
	cpy := &DelegateTx{
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		ChainID:    new(big.Int),
		GasPrice:   new(big.Int),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	return cpy
}

// accessors for innerTx.
func (tx *DelegateTx) txType() byte              { return DelegateTxType }
func (tx *DelegateTx) chainID() *big.Int         { return tx.ChainID }
func (tx *DelegateTx) accessList() AccessList    { return tx.AccessList }
func (tx *DelegateTx) data() []byte              { return tx.Data }
func (tx *DelegateTx) gas() uint64               { return tx.Gas }
func (tx *DelegateTx) gasPrice() *big.Int        { return tx.GasPrice }
func (tx *DelegateTx) gasTipCap() *big.Int       { return tx.GasPrice }
func (tx *DelegateTx) gasFeeCap() *big.Int       { return tx.GasPrice }
func (tx *DelegateTx) value() *big.Int           { return big.NewInt(0) }
func (tx *DelegateTx) nonce() uint64             { return tx.Nonce }
func (tx *DelegateTx) to() *common.Address       { return tx.To }

func (tx *DelegateTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.GasPrice)
}

func (tx *DelegateTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *DelegateTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

func (tx *DelegateTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *DelegateTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}
