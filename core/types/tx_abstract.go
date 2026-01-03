// Copyright 2026 The go-ethereum Authors
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
	"github.com/holiman/uint256"
)

type AbstractAuthorization struct {
	Target common.Address
	Data   []byte
	Gas    uint64
}

func (a *AbstractAuthorization) copy() *AbstractAuthorization {
	if a == nil {
		return nil
	}
	return &AbstractAuthorization{
		Target: a.Target,
		Data:   common.CopyBytes(a.Data),
		Gas:    a.Gas,
	}
}

type PaymasterAuthorization struct {
	Target    common.Address
	Data      []byte
	Gas       uint64
	PostOpGas uint64
}

func (a *PaymasterAuthorization) copy() *PaymasterAuthorization {
	if a == nil {
		return nil
	}
	return &PaymasterAuthorization{
		Target:    a.Target,
		Data:      common.CopyBytes(a.Data),
		Gas:       a.Gas,
		PostOpGas: a.PostOpGas,
	}
}

type AbstractTx struct {
	ChainID *uint256.Int
	Nonce   uint64

	Sender    AbstractAuthorization
	Deployer  *AbstractAuthorization
	Paymaster *PaymasterAuthorization

	Data []byte // FromExecutionData
	Gas  uint64

	GasTipCap *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap *uint256.Int // a.k.a. maxFeePerGas

	AccessList AccessList
	AuthList   []SetCodeAuthorization
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *AbstractTx) copy() TxData {
	cpy := &AbstractTx{
		Nonce:     tx.Nonce,
		Sender:    *tx.Sender.copy(),
		Deployer:  tx.Deployer.copy(),
		Paymaster: tx.Paymaster.copy(),
		Data:      common.CopyBytes(tx.Data),
		Gas:       tx.Gas,

		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		AuthList:   make([]SetCodeAuthorization, len(tx.AuthList)),
		ChainID:    new(uint256.Int),
		GasTipCap:  new(uint256.Int),
		GasFeeCap:  new(uint256.Int),
	}
	copy(cpy.AccessList, tx.AccessList)
	copy(cpy.AuthList, tx.AuthList)
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	return cpy
}

// accessors for innerTx.
func (tx *AbstractTx) txType() byte           { return AbstractTxType }
func (tx *AbstractTx) chainID() *big.Int      { return tx.ChainID.ToBig() }
func (tx *AbstractTx) accessList() AccessList { return tx.AccessList }
func (tx *AbstractTx) data() []byte           { return tx.Data }
func (tx *AbstractTx) gas() uint64            { return tx.Gas }
func (tx *AbstractTx) gasFeeCap() *big.Int    { return tx.GasFeeCap.ToBig() }
func (tx *AbstractTx) gasTipCap() *big.Int    { return tx.GasTipCap.ToBig() }
func (tx *AbstractTx) gasPrice() *big.Int     { return tx.GasFeeCap.ToBig() }
func (tx *AbstractTx) value() *big.Int        { return big.NewInt(0) }
func (tx *AbstractTx) nonce() uint64          { return tx.Nonce }
func (tx *AbstractTx) to() *common.Address    { return nil } // TODO: is returning nil here correct?

func (tx *AbstractTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap.ToBig())
	}
	tip := dst.Sub(tx.GasFeeCap.ToBig(), baseFee)
	if tip.Cmp(tx.GasTipCap.ToBig()) > 0 {
		tip.Set(tx.GasTipCap.ToBig())
	}
	return tip.Add(tip, baseFee)
}

func (tx *AbstractTx) rawSignatureValues() (v, r, s *big.Int) {
	panic("abstract tx does not have signature")
}

func (tx *AbstractTx) setSignatureValues(chainID, v, r, s *big.Int) {
	panic("abstract tx does not have signature")
}

func (tx *AbstractTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *AbstractTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

func (tx *AbstractTx) sigHash(chainID *big.Int) common.Hash {
	panic("abstract tx does not have signature")
}
