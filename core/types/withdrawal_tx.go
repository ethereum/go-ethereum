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

// `WithdrawalTx` mirrors a withdrawal receipt committed to
// on the consensus layer.
//  This receipt is guaranteed to have some execution address
// and an amount in Gwei.
// NOTE: the amount is converted to Wei when the transaction data is
// unmarshalled into this struct.
type WithdrawalTx struct {
	To    *common.Address
	Value *big.Int
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *WithdrawalTx) copy() TxData {
	cpy := &WithdrawalTx{
		To:    copyAddressPtr(tx.To),
		Value: new(big.Int),
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	return cpy
}

// accessors for innerTx.
func (tx *WithdrawalTx) txType() byte           { return WithdrawalTxType }
func (tx *WithdrawalTx) chainID() *big.Int      { return new(big.Int) }
func (tx *WithdrawalTx) accessList() AccessList { return nil }
func (tx *WithdrawalTx) data() []byte           { return nil }
func (tx *WithdrawalTx) gas() uint64            { return 0 }
func (tx *WithdrawalTx) gasFeeCap() *big.Int    { return new(big.Int) }
func (tx *WithdrawalTx) gasTipCap() *big.Int    { return new(big.Int) }
func (tx *WithdrawalTx) gasPrice() *big.Int     { return new(big.Int) }
func (tx *WithdrawalTx) value() *big.Int        { return tx.Value }
func (tx *WithdrawalTx) nonce() uint64          { return 0 }
func (tx *WithdrawalTx) to() *common.Address    { return tx.To }

func (tx *WithdrawalTx) rawSignatureValues() (v, r, s *big.Int) {
	// return "zero" values, this method should not be called
	return new(big.Int), new(big.Int), new(big.Int)
}

func (tx *WithdrawalTx) setSignatureValues(chainID, v, r, s *big.Int) {
	//  no-op, keep for broader compatibility
}
