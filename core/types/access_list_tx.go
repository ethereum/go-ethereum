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

type AccessTuple struct {
	Address     *common.Address `json:"address"        gencodec:"required"`
	StorageKeys []*common.Hash  `json:"storageKeys"    gencodec:"required"`
}

type AccessList []AccessTuple

func (al *AccessList) Addresses() int { return len(*al) }
func (al *AccessList) StorageKeys() int {
	sum := 0
	for _, tuple := range *al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

type AccessListTransaction struct {
	Chain        *big.Int        `json:"chainId"    gencodec:"required"`
	AccountNonce uint64          `json:"nonce"      gencodec:"required"`
	Price        *big.Int        `json:"gasPrice"   gencodec:"required"`
	GasLimit     uint64          `json:"gas"        gencodec:"required"`
	Recipient    *common.Address `json:"to"         rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"      gencodec:"required"`
	Payload      []byte          `json:"input"      gencodec:"required"`
	Accesses     *AccessList     `json:"accessList" gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func NewAccessListTransaction(chainId *big.Int, nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accesses *AccessList) *Transaction {
	return newAccessListTransaction(chainId, nonce, &to, amount, gasLimit, gasPrice, data, accesses)
}

func NewAccessListContractCreation(chainId *big.Int, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accesses *AccessList) *Transaction {
	return newAccessListTransaction(chainId, nonce, nil, amount, gasLimit, gasPrice, data, accesses)
}

func newAccessListTransaction(chainId *big.Int, nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accesses *AccessList) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	i := AccessListTransaction{
		Chain:        new(big.Int),
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Accesses:     &AccessList{},
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
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
	if gasPrice != nil {
		i.Price.Set(gasPrice)
	}
	if accesses != nil {
		i.Accesses = accesses
	}
	return &Transaction{
		typ:   AccessListTxId,
		inner: &i,
		time:  time.Now(),
	}
}

func (tx *AccessListTransaction) ChainId() *big.Int       { return tx.Chain }
func (tx *AccessListTransaction) Protected() bool         { return true }
func (tx *AccessListTransaction) AccessList() *AccessList { return tx.Accesses }
func (tx *AccessListTransaction) Data() []byte            { return common.CopyBytes(tx.Payload) }
func (tx *AccessListTransaction) Gas() uint64             { return tx.GasLimit }
func (tx *AccessListTransaction) GasPrice() *big.Int      { return new(big.Int).Set(tx.Price) }
func (tx *AccessListTransaction) Value() *big.Int         { return new(big.Int).Set(tx.Amount) }
func (tx *AccessListTransaction) Nonce() uint64           { return tx.AccountNonce }
func (tx *AccessListTransaction) CheckNonce() bool        { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *AccessListTransaction) To() *common.Address {
	if tx.Recipient == nil {
		return nil
	}
	to := *tx.Recipient
	return &to
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *AccessListTransaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}
