// Copyright 2024 The go-ethereum Authors
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
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type Deposit -field-override depositMarshaling -out gen_deposit_json.go

// Deposit contians EIP-6110 deposit data.
type Deposit struct {
	PublicKey             BLSPublicKey `json:"pubkey"`
	WithdrawalCredentials common.Hash  `json:"withdrawalCredentials"`
	Amount                uint64       `json:"amount"` // in gwei
	Signature             BLSSignature `json:"signature"`
	Index                 uint64       `json:"index"`
}

type depositMarshaling struct {
	PublicKey             hexutil.Bytes
	WithdrawalCredentials hexutil.Bytes
	Amount                hexutil.Uint64
	Signature             hexutil.Bytes
	Index                 hexutil.Uint64
}

// Deposit implements DerivableList for withdrawals.
type Deposits []*Deposit

// Len returns the length of s.
func (s Deposits) Len() int { return len(s) }

// EncodeIndex encodes the i'th deposit to s.
func (s Deposits) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

// misc bls types
////

var (
	pubkeyT = reflect.TypeOf(BLSPublicKey{})
	sigT    = reflect.TypeOf(BLSSignature{})
)

type BLSPublicKey [48]byte

// UnmarshalJSON parses a hash in hex syntax.
func (h *BLSPublicKey) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(pubkeyT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h BLSPublicKey) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

type BLSSignature [96]byte

// UnmarshalJSON parses a hash in hex syntax.
func (h *BLSSignature) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(sigT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h BLSSignature) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}
