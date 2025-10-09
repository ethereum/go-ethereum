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

// Package types contains common types.
package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// OverrideAccount specifies the state of an account to be overridden.
type OverrideAccount struct {
	// Nonce sets nonce of the account. Note: the nonce override will only
	// be applied when it is set to a non-zero value.
	Nonce uint64

	// Code sets the contract code. The override will be applied
	// when the code is non-nil, i.e. setting empty code is possible
	// using an empty slice.
	Code []byte

	// Balance sets the account balance.
	Balance *big.Int

	// State sets the complete storage. The override will be applied
	// when the given map is non-nil. Using an empty map wipes the
	// entire contract storage during the call.
	State map[common.Hash]common.Hash

	// StateDiff allows overriding individual storage slots.
	StateDiff map[common.Hash]common.Hash
}

func (a OverrideAccount) MarshalJSON() ([]byte, error) {
	type acc struct {
		Nonce     hexutil.Uint64              `json:"nonce,omitempty"`
		Code      string                      `json:"code,omitempty"`
		Balance   *hexutil.Big                `json:"balance,omitempty"`
		State     interface{}                 `json:"state,omitempty"`
		StateDiff map[common.Hash]common.Hash `json:"stateDiff,omitempty"`
	}

	output := acc{
		Nonce:     hexutil.Uint64(a.Nonce),
		Balance:   (*hexutil.Big)(a.Balance),
		StateDiff: a.StateDiff,
	}
	if a.Code != nil {
		output.Code = hexutil.Encode(a.Code)
	}
	if a.State != nil {
		output.State = a.State
	}
	return json.Marshal(output)
}

// BlockOverrides specifies the  set of header fields to override.
type BlockOverrides struct {
	// Number overrides the block number.
	Number *big.Int
	// Difficulty overrides the block difficulty.
	Difficulty *big.Int
	// Time overrides the block timestamp. Time is applied only when
	// it is non-zero.
	Time uint64
	// GasLimit overrides the block gas limit. GasLimit is applied only when
	// it is non-zero.
	GasLimit uint64
	// Coinbase overrides the block coinbase. Coinbase is applied only when
	// it is different from the zero address.
	Coinbase common.Address
	// Random overrides the block extra data which feeds into the RANDOM opcode.
	// Random is applied only when it is a non-zero hash.
	Random common.Hash
	// BaseFee overrides the block base fee.
	BaseFee *big.Int
}

func (o BlockOverrides) MarshalJSON() ([]byte, error) {
	type override struct {
		Number     *hexutil.Big    `json:"number,omitempty"`
		Difficulty *hexutil.Big    `json:"difficulty,omitempty"`
		Time       hexutil.Uint64  `json:"time,omitempty"`
		GasLimit   hexutil.Uint64  `json:"gasLimit,omitempty"`
		Coinbase   *common.Address `json:"feeRecipient,omitempty"`
		Random     *common.Hash    `json:"prevRandao,omitempty"`
		BaseFee    *hexutil.Big    `json:"baseFeePerGas,omitempty"`
	}

	output := override{
		Number:     (*hexutil.Big)(o.Number),
		Difficulty: (*hexutil.Big)(o.Difficulty),
		Time:       hexutil.Uint64(o.Time),
		GasLimit:   hexutil.Uint64(o.GasLimit),
		BaseFee:    (*hexutil.Big)(o.BaseFee),
	}
	if o.Coinbase != (common.Address{}) {
		output.Coinbase = &o.Coinbase
	}
	if o.Random != (common.Hash{}) {
		output.Random = &o.Random
	}
	return json.Marshal(output)
}
