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

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type Deposit -field-override depositMarshaling -out gen_deposit_json.go

// Deposit contians EIP-6110 deposit data.
type Deposit struct {
	PublicKey             [48]byte     `json:"pubkey"`                // public key of validator
	WithdrawalCredentials common.Hash  `json:"withdrawalCredentials"` // beneficiary of the validator funds
	Amount                uint64       `json:"amount"`                // deposit size in Gwei
	Signature             [96]byte     `json:"signature"`             // signature over deposit msg
	Index                 uint64       `json:"index"`                 // deposit count value
}

// field type overrides for gencodec
type depositMarshaling struct {
	PublicKey             hexutil.Bytes
	WithdrawalCredentials hexutil.Bytes
	Amount                hexutil.Uint64
	Signature             hexutil.Bytes
	Index                 hexutil.Uint64
}

// field type overrides for abi upacking
type depositUnpacking struct {
	Pubkey                []byte
	WithdrawalCredentials []byte
	Amount                []byte
	Signature             []byte
	Index                 []byte
}

// Deposits implements DerivableList for requests.
type Deposits []*Deposit

// Len returns the length of s.
func (s Deposits) Len() int { return len(s) }

// EncodeIndex encodes the i'th deposit to s.
func (s Deposits) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

// Requests creates a deep copy of each deposit and returns a slice of Request
// objects.
func (s Deposits) Requests() (reqs Requests) {
	for _, d := range s {
		reqs = append(reqs, NewRequest(d))
	}
	return
}

var (
	// DepositABI is an ABI instance of beacon chain deposit events.
	DepositABI   = abi.ABI{Events: map[string]abi.Event{"DepositEvent": depositEvent}}
	bytesT, _    = abi.NewType("bytes", "", nil)
	depositEvent = abi.NewEvent("DepositEvent", "DepositEvent", false, abi.Arguments{
		{Name: "pubkey", Type: bytesT, Indexed: false},
		{Name: "withdrawal_credentials", Type: bytesT, Indexed: false},
		{Name: "amount", Type: bytesT, Indexed: false},
		{Name: "signature", Type: bytesT, Indexed: false},
		{Name: "index", Type: bytesT, Indexed: false}},
	)
)

// UnpackIntoDeposit unpacks a serialized DepositEvent.
func UnpackIntoDeposit(data []byte) (*Deposit, error) {
	var du depositUnpacking
	if err := DepositABI.UnpackIntoInterface(&du, "DepositEvent", data); err != nil {
		return nil, err
	}
	var d Deposit
	copy(d.PublicKey[:], du.Pubkey)
	copy(d.WithdrawalCredentials[:], du.WithdrawalCredentials)
	d.Amount = binary.LittleEndian.Uint64(du.Amount)
	copy(d.Signature[:], du.Signature)
	d.Index = binary.LittleEndian.Uint64(du.Index)

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
