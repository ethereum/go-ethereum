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
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type LegacyTransaction struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newLegacyTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newLegacyTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

func newLegacyTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	i := LegacyTransaction{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		i.Amount.Set(amount)
	}
	if gasPrice != nil {
		i.Price.Set(gasPrice)
	}
	return &Transaction{
		typ:   LegacyTxType,
		inner: &i,
		time:  time.Now(),
	}
}

func (tx *LegacyTransaction) Type() byte              { return LegacyTxType }
func (tx *LegacyTransaction) ChainId() *big.Int       { return deriveChainId(tx.V) }
func (tx *LegacyTransaction) Protected() bool         { return isProtectedV(tx.V) }
func (tx *LegacyTransaction) AccessList() *AccessList { return nil }
func (tx *LegacyTransaction) Data() []byte            { return common.CopyBytes(tx.Payload) }
func (tx *LegacyTransaction) Gas() uint64             { return tx.GasLimit }
func (tx *LegacyTransaction) GasPrice() *big.Int      { return new(big.Int).Set(tx.Price) }
func (tx *LegacyTransaction) Value() *big.Int         { return new(big.Int).Set(tx.Amount) }
func (tx *LegacyTransaction) Nonce() uint64           { return tx.AccountNonce }
func (tx *LegacyTransaction) CheckNonce() bool        { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *LegacyTransaction) To() *common.Address {
	if tx.Recipient == nil {
		return nil
	}
	to := *tx.Recipient
	return &to
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *LegacyTransaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}
