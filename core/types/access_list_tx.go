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

//go:generate gencodec -type AccessTuple -out gen_access_tuple.go

type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
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

type AccessListTx struct {
	Chain        *big.Int
	AccountNonce uint64
	Price        *big.Int
	GasLimit     uint64
	Recipient    *common.Address `rlp:"nil"` // nil means contract creation
	Amount       *big.Int
	Payload      []byte
	Accesses     AccessList

	// Signature values
	V *big.Int
	R *big.Int
	S *big.Int
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *AccessListTx) copy() TxData {
	cpy := &AccessListTx{
		AccountNonce: tx.AccountNonce,
		Recipient:    tx.Recipient, // TODO: copy pointed-to address
		Payload:      common.CopyBytes(tx.Payload),
		GasLimit:     tx.GasLimit,
		// These are copied below.
		Accesses: make(AccessList, len(tx.Accesses)),
		Amount:   new(big.Int),
		Chain:    new(big.Int),
		Price:    new(big.Int),
		V:        new(big.Int),
		R:        new(big.Int),
		S:        new(big.Int),
	}
	copy(cpy.Accesses, tx.Accesses)
	if tx.Amount != nil {
		cpy.Amount.Set(tx.Amount)
	}
	if tx.Chain != nil {
		cpy.Chain.Set(tx.Chain)
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

func (tx *AccessListTx) txType() byte           { return AccessListTxType }
func (tx *AccessListTx) chainID() *big.Int      { return tx.Chain }
func (tx *AccessListTx) protected() bool        { return true }
func (tx *AccessListTx) accessList() AccessList { return tx.Accesses }
func (tx *AccessListTx) data() []byte           { return tx.Payload }
func (tx *AccessListTx) gas() uint64            { return tx.GasLimit }
func (tx *AccessListTx) gasPrice() *big.Int     { return tx.Price }
func (tx *AccessListTx) value() *big.Int        { return tx.Amount }
func (tx *AccessListTx) nonce() uint64          { return tx.AccountNonce }
func (tx *AccessListTx) to() *common.Address    { return tx.Recipient }

func (tx *AccessListTx) rawSignatureValues() (v, r, s *big.Int) { return tx.V, tx.R, tx.S }
