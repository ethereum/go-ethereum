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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
)

// Rip7560AccountAbstractionTx represents an RIP-7560 transaction.
type Rip7560AccountAbstractionTx struct {
	// overlapping fields
	ChainID    *big.Int
	GasTipCap  *big.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int // a.k.a. maxFeePerGas
	Gas        uint64
	Data       []byte
	AccessList AccessList

	// extra fields
	Sender        *common.Address
	Signature     []byte
	Paymaster     *common.Address `rlp:"nil"`
	PaymasterData []byte
	Deployer      *common.Address `rlp:"nil"`
	DeployerData  []byte
	BuilderFee    *big.Int
	ValidationGas uint64
	PaymasterGas  uint64
	PostOpGas     uint64

	// removed fields
	To    *common.Address `rlp:"nil"`
	Nonce uint64
	Value *big.Int
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *Rip7560AccountAbstractionTx) copy() TxData {
	cpy := &Rip7560AccountAbstractionTx{
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Nonce: tx.Nonce,
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),

		Sender:        copyAddressPtr(tx.Sender),
		Signature:     common.CopyBytes(tx.Signature),
		Paymaster:     copyAddressPtr(tx.Paymaster),
		PaymasterData: common.CopyBytes(tx.PaymasterData),
		Deployer:      copyAddressPtr(tx.Deployer),
		DeployerData:  common.CopyBytes(tx.DeployerData),
		BuilderFee:    new(big.Int),
		ValidationGas: tx.ValidationGas,
		PaymasterGas:  tx.PaymasterGas,
		PostOpGas:     tx.PostOpGas,
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.BuilderFee != nil {
		cpy.BuilderFee.Set(tx.BuilderFee)
	}
	return cpy
}

// accessors for innerTx.
func (tx *Rip7560AccountAbstractionTx) txType() byte           { return Rip7560Type }
func (tx *Rip7560AccountAbstractionTx) chainID() *big.Int      { return tx.ChainID }
func (tx *Rip7560AccountAbstractionTx) accessList() AccessList { return tx.AccessList }
func (tx *Rip7560AccountAbstractionTx) data() []byte           { return tx.Data }
func (tx *Rip7560AccountAbstractionTx) gas() uint64            { return tx.Gas }
func (tx *Rip7560AccountAbstractionTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *Rip7560AccountAbstractionTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *Rip7560AccountAbstractionTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *Rip7560AccountAbstractionTx) value() *big.Int        { return tx.Value }
func (tx *Rip7560AccountAbstractionTx) nonce() uint64          { return 0 }
func (tx *Rip7560AccountAbstractionTx) to() *common.Address    { return tx.To }

func (tx *Rip7560AccountAbstractionTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap)
	}
	tip := dst.Sub(tx.GasFeeCap, baseFee)
	if tip.Cmp(tx.GasTipCap) > 0 {
		tip.Set(tx.GasTipCap)
	}
	return tip.Add(tip, baseFee)
}

func (tx *Rip7560AccountAbstractionTx) rawSignatureValues() (v, r, s *big.Int) {
	return new(big.Int), new(big.Int), new(big.Int)
}

func (tx *Rip7560AccountAbstractionTx) setSignatureValues(chainID, v, r, s *big.Int) {
	//tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

// encode the subtype byte and the payload-bearing bytes of the RIP-7560 transaction
func (tx *Rip7560AccountAbstractionTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

// decode the payload-bearing bytes of the encoded RIP-7560 transaction payload
func (tx *Rip7560AccountAbstractionTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

// Rip7560Transaction an equivalent of a solidity struct only used to encode the 'transaction' parameter
type Rip7560Transaction struct {
	Sender               common.Address
	Nonce                *big.Int
	ValidationGasLimit   *big.Int
	PaymasterGasLimit    *big.Int
	PostOpGasLimit       *big.Int
	CallGasLimit         *big.Int
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	BuilderFee           *big.Int
	Paymaster            *common.Address
	PaymasterData        []byte
	Deployer             *common.Address
	DeployerData         []byte
	CallData             []byte
	Signature            []byte
}

func (tx *Rip7560AccountAbstractionTx) AbiEncode() ([]byte, error) {
	structThing, _ := abi.NewType("tuple", "struct thing", []abi.ArgumentMarshaling{
		{Name: "sender", Type: "address"},
		{Name: "nonce", Type: "uint256"},
		{Name: "validationGasLimit", Type: "uint256"},
		{Name: "paymasterGasLimit", Type: "uint256"},
		{Name: "callGasLimit", Type: "uint256"},
		{Name: "maxFeePerGas", Type: "uint256"},
		{Name: "maxPriorityFeePerGas", Type: "uint256"},
		{Name: "builderFee", Type: "uint256"},
		{Name: "paymasterData", Type: "bytes"},
		{Name: "deployerData", Type: "bytes"},
		{Name: "callData", Type: "bytes"},
		{Name: "signature", Type: "bytes"},
	})

	args := abi.Arguments{
		{Type: structThing, Name: "param_one"},
	}
	record := &Rip7560Transaction{
		Sender:               *tx.Sender,
		Nonce:                big.NewInt(int64(tx.Nonce)),
		ValidationGasLimit:   big.NewInt(int64(tx.ValidationGas)),
		PaymasterGasLimit:    big.NewInt(int64(tx.PaymasterGas)),
		CallGasLimit:         big.NewInt(int64(tx.Gas)),
		MaxFeePerGas:         tx.GasFeeCap,
		MaxPriorityFeePerGas: tx.GasTipCap,
		BuilderFee:           tx.BuilderFee,
		PaymasterData:        tx.PaymasterData,
		DeployerData:         tx.DeployerData,
		CallData:             tx.Data,
		Signature:            tx.Signature,
	}
	packed, err := args.Pack(&record)
	return packed, err
}

// ExternallyReceivedBundle represents a bundle of Type 4 transactions received from a trusted 3rd party.
// The validator includes the bundle in the original order atomically or drops it completely.
type ExternallyReceivedBundle struct {
	BundlerId     string
	BundleHash    common.Hash
	ValidForBlock *big.Int
	Transactions  []*Transaction
}

// BundleReceipt represents a receipt for an ExternallyReceivedBundle successfully included in a block.
type BundleReceipt struct {
	BundleHash          common.Hash
	Count               uint64
	Status              uint64 // 0=included / 1=pending / 2=invalid / 3=unknown
	BlockNumber         uint64
	BlockHash           common.Hash
	TransactionReceipts []*Receipt
	GasUsed             uint64
	GasPaidPriority     *big.Int
	BlockTimestamp      uint64
}
