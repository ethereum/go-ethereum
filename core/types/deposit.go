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
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type Deposit -field-override depositMarshaling -out gen_deposit_json.go

// Deposit contains EIP-6110 deposit data.
type Deposit struct {
	PublicKey             [48]byte    `json:"pubkey"`                // public key of validator
	WithdrawalCredentials common.Hash `json:"withdrawalCredentials"` // beneficiary of the validator funds
	Amount                uint64      `json:"amount"`                // deposit size in Gwei
	Signature             [96]byte    `json:"signature"`             // signature over deposit msg
	Index                 uint64      `json:"index"`                 // deposit count value
}

// field type overrides for gencodec
type depositMarshaling struct {
	PublicKey             hexutil.Bytes
	WithdrawalCredentials hexutil.Bytes
	Amount                hexutil.Uint64
	Signature             hexutil.Bytes
	Index                 hexutil.Uint64
}

// Deposits implements DerivableList for requests.
type Deposits []*Deposit

// Len returns the length of s.
func (s Deposits) Len() int { return len(s) }

// EncodeIndex encodes the i'th deposit to s.
func (s Deposits) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

// UnpackIntoDeposit unpacks a serialized DepositEvent.
func UnpackIntoDeposit(data []byte) (*Deposit, error) {
	if len(data) != 576 {
		return nil, fmt.Errorf("deposit wrong length: want 576, have %d", len(data))
	}
	var d Deposit
	// The ABI encodes the position of dynamic elements first. Since there are 5
	// elements, skip over the positional data. The first 32 bytes of dynamic
	// elements also encode their actual length. Skip over that value too.
	b := 32*5 + 32
	// PublicKey is the first element. ABI encoding pads values to 32 bytes, so
	// despite BLS public keys being length 48, the value length here is 64. Then
	// skip over the next length value.
	copy(d.PublicKey[:], data[b:b+48])
	b += 48 + 16 + 32
	// WithdrawalCredentials is 32 bytes. Read that value then skip over next
	// length.
	copy(d.WithdrawalCredentials[:], data[b:b+32])
	b += 32 + 32
	// Amount is 8 bytes, but it is padded to 32. Skip over it and the next
	// length.
	d.Amount = binary.LittleEndian.Uint64(data[b : b+8])
	b += 8 + 24 + 32
	// Signature is 96 bytes. Skip over it and the next length.
	copy(d.Signature[:], data[b:b+96])
	b += 96 + 32
	// Amount is 8 bytes.
	d.Index = binary.LittleEndian.Uint64(data[b : b+8])

	return &d, nil
}

func (d *Deposit) requestType() byte            { return DepositRequestType }
func (d *Deposit) encode(b *bytes.Buffer) error { return rlp.Encode(b, d) }
func (d *Deposit) decode(input []byte) error    { return rlp.DecodeBytes(input, d) }
func (d *Deposit) copy() RequestData {
	return &Deposit{
		PublicKey:             d.PublicKey,
		WithdrawalCredentials: d.WithdrawalCredentials,
		Amount:                d.Amount,
		Signature:             d.Signature,
		Index:                 d.Index,
	}
}
