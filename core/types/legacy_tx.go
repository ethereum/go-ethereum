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

package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// LegacyTx is the transaction data of regular Ethereum transactions.
type LegacyTx struct {
	AccountNonce uint64
	Price        *big.Int
	GasLimit     uint64
	Recipient    *common.Address `rlp:"nil"` // nil means contract creation
	Amount       *big.Int
	Payload      []byte

	// Signature values.
	V *big.Int
	R *big.Int
	S *big.Int
}

// NewTransaction creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&LegacyTx{
		AccountNonce: nonce,
		Recipient:    &to,
		Amount:       amount,
		GasLimit:     gasLimit,
		Price:        gasPrice,
		Payload:      data,
	})
}

// NewContractCreation creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&LegacyTx{
		AccountNonce: nonce,
		Amount:       amount,
		GasLimit:     gasLimit,
		Price:        gasPrice,
		Payload:      data,
	})
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *LegacyTx) copy() TxData {
	cpy := &LegacyTx{
		AccountNonce: tx.AccountNonce,
		Recipient:    tx.Recipient, // TODO: copy pointed-to address
		Payload:      common.CopyBytes(tx.Payload),
		GasLimit:     tx.GasLimit,
		// These are initialized below.
		Amount: new(big.Int),
		Price:  new(big.Int),
		V:      new(big.Int),
		R:      new(big.Int),
		S:      new(big.Int),
	}
	if tx.Amount != nil {
		cpy.Amount.Set(tx.Amount)
	}
	if tx.Price != nil {
		cpy.Price.Set(tx.Price)
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

func (tx *LegacyTx) txType() byte           { return LegacyTxType }
func (tx *LegacyTx) chainID() *big.Int      { return deriveChainId(tx.V) }
func (tx *LegacyTx) accessList() AccessList { return nil }
func (tx *LegacyTx) data() []byte           { return tx.Payload }
func (tx *LegacyTx) gas() uint64            { return tx.GasLimit }
func (tx *LegacyTx) gasPrice() *big.Int     { return tx.Price }
func (tx *LegacyTx) value() *big.Int        { return tx.Amount }
func (tx *LegacyTx) nonce() uint64          { return tx.AccountNonce }
func (tx *LegacyTx) to() *common.Address    { return tx.Recipient }

func (tx *LegacyTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *LegacyTx) setSignatureValues(v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}
