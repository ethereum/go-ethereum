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

type DynamicFeeTransaction struct {
	Chain        *big.Int        `json:"chainId"    gencodec:"required"`
	AccountNonce uint64          `json:"nonce"      gencodec:"required"`
	InclusionFee *big.Int        `json:"inclusionFee"   gencodec:"required"`
	GasFee       *big.Int        `json:"gasFee"   gencodec:"required"`
	GasLimit     uint64          `json:"gas"        gencodec:"required"`
	Recipient    *common.Address `json:"to"         rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"      gencodec:"required"`
	Payload      []byte          `json:"input"      gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func NewDynamicFeeTransaction(chainId *big.Int, nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasFee, tip *big.Int, data []byte) *Transaction {
	return newDynamicFeeTransaction(chainId, nonce, &to, amount, gasLimit, gasFee, tip, data)
}

func NewDynamicFeeContractCreation(chainId *big.Int, nonce uint64, amount *big.Int, gasLimit uint64, gasFee, tip *big.Int, data []byte, accesses *AccessList) *Transaction {
	return newDynamicFeeTransaction(chainId, nonce, nil, amount, gasLimit, gasFee, tip, data)
}

func newDynamicFeeTransaction(chainId *big.Int, nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasFee, tip *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	i := DynamicFeeTransaction{
		Chain:        new(big.Int),
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		InclusionFee: new(big.Int),
		GasFee:       new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if chainId != nil {
		i.Chain.Set(chainId)
	}
	if amount != nil {
		i.Amount.Set(amount)
	}
	if gasFee != nil {
		i.GasFee.Set(gasFee)
	}
	if tip != nil {
		i.InclusionFee.Set(tip)
	}
	return &Transaction{
		typ:   DynamicFeeTxId,
		inner: &i,
		time:  time.Now(),
	}
}

func (tx *DynamicFeeTransaction) ChainId() *big.Int       { return tx.Chain }
func (tx *DynamicFeeTransaction) Protected() bool         { return true }
func (tx *DynamicFeeTransaction) AccessList() *AccessList { return nil }
func (tx *DynamicFeeTransaction) Data() []byte            { return common.CopyBytes(tx.Payload) }
func (tx *DynamicFeeTransaction) Gas() uint64             { return tx.GasLimit }
func (tx *DynamicFeeTransaction) FeeCap() *big.Int        { return new(big.Int).Set(tx.GasFee) }
func (tx *DynamicFeeTransaction) Tip() *big.Int           { return new(big.Int).Set(tx.InclusionFee) }
func (tx *DynamicFeeTransaction) Value() *big.Int         { return new(big.Int).Set(tx.Amount) }
func (tx *DynamicFeeTransaction) Nonce() uint64           { return tx.AccountNonce }
func (tx *DynamicFeeTransaction) CheckNonce() bool        { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *DynamicFeeTransaction) To() *common.Address {
	if tx.Recipient == nil {
		return nil
	}
	to := *tx.Recipient
	return &to
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *DynamicFeeTransaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}
