// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// DepositTxType is the transaction type of cross-shard deposits, as of the
// deposit transactions of the OP Stack.
const DepositTxType = 0x7e

// DepositTxVersion is the version of the DepositTx format.
const DepositTxVersion uint8 = 0

// DepositTx is a cross-shard deposit, a message sent to a shard by another shard
// or by the root chain. Only moves are sent for now: a move only transfers Value
// to To, running no code even if To is a contract. It has no Data, and its Gas
// is params.XShardSendGas, paid at GasPrice.
//
// A deposit is not signed: its sender is From, as the source shard checked it.
// Only the consensus layer may include deposits in a block, as they mint the
// value and gas burnt on the source shard.
//
// The source block hash and log index identify the origin of the deposit: the
// block that sent it and the index of the sending log in that block. A deposit
// from the root chain has the hash of the root block and index 0.
type DepositTx struct {
	Version         uint8
	SourceBlockHash common.Hash
	SourceLogIndex  uint64
	From            common.Address
	To              *common.Address `rlp:"nil"` // nil means contract creation
	Value           *uint256.Int
	GasPrice        *uint256.Int // gas price of the sending transaction
	Gas             uint64       // gas the deposit may use on the destination shard
	IsMove          bool         // whether the deposit only transfers Value to To
	Data            []byte
}

// Deposit returns a copy of the cross-shard deposit of the transaction, or nil
// if it is not a deposit.
func (tx *Transaction) Deposit() *DepositTx {
	if deposit, ok := tx.inner.(*DepositTx); ok {
		return deposit.copy().(*DepositTx)
	}
	return nil
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DepositTx) copy() TxData {
	cpy := &DepositTx{
		Version:         tx.Version,
		SourceBlockHash: tx.SourceBlockHash,
		SourceLogIndex:  tx.SourceLogIndex,
		From:            tx.From,
		To:              copyAddressPtr(tx.To),
		Gas:             tx.Gas,
		IsMove:          tx.IsMove,
		Data:            common.CopyBytes(tx.Data),
		// These are copied below.
		Value:    new(uint256.Int),
		GasPrice: new(uint256.Int),
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
	}
	return cpy
}

// accessors for innerTx.
func (tx *DepositTx) txType() byte           { return DepositTxType }
func (tx *DepositTx) chainID() *big.Int      { return new(big.Int) }
func (tx *DepositTx) accessList() AccessList { return nil }
func (tx *DepositTx) data() []byte           { return tx.Data }
func (tx *DepositTx) gas() uint64            { return tx.Gas }
func (tx *DepositTx) gasFeeCap() *big.Int    { return tx.GasPrice.ToBig() }
func (tx *DepositTx) gasTipCap() *big.Int    { return tx.GasPrice.ToBig() }
func (tx *DepositTx) gasPrice() *big.Int     { return tx.GasPrice.ToBig() }
func (tx *DepositTx) value() *big.Int        { return tx.Value.ToBig() }
func (tx *DepositTx) nonce() uint64          { return 0 }
func (tx *DepositTx) to() *common.Address    { return copyAddressPtr(tx.To) }

// effectiveGasPrice returns the gas price of the deposit, whatever the base fee:
// the deposit pays the gas price of its sending transaction.
func (tx *DepositTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.GasPrice.ToBig())
}

func (tx *DepositTx) rawSignatureValues() (v, r, s *big.Int) {
	return new(big.Int), new(big.Int), new(big.Int)
}

func (tx *DepositTx) setSignatureValues(chainID, v, r, s *big.Int) {
	// Deposits are not signed.
}

func (tx *DepositTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

// decode decodes a deposit, rejecting the versions it does not know.
func (tx *DepositTx) decode(input []byte) error {
	if err := rlp.DecodeBytes(input, tx); err != nil {
		return err
	}
	if tx.Version != DepositTxVersion {
		return fmt.Errorf("unsupported deposit version %d", tx.Version)
	}
	return nil
}

// sigHash returns the zero hash: deposits are not signed, and signers fail to
// sign one as they do not support its type.
func (tx *DepositTx) sigHash(*big.Int) common.Hash {
	return common.Hash{}
}
