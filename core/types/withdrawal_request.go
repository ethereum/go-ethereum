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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type WithdrawalRequest -field-override withdrawalRequestMarshaling -out gen_withdrawal_request_json.go

// WithdrawalRequest represents an EIP-7002 withdrawal request from source for
// the validator associated with the public key for amount.
type WithdrawalRequest struct {
	Source    common.Address `json:"sourceAddress"`
	PublicKey [48]byte       `json:"validatorPublicKey"`
	Amount    uint64         `json:"amount"`
}

// field type overrides for gencodec
type withdrawalRequestMarshaling struct {
	Amount hexutil.Uint64
}

func (w *WithdrawalRequest) Bytes() []byte {
	out := make([]byte, 76)
	copy(out, w.Source.Bytes())
	copy(out[20:], w.PublicKey[:])
	binary.LittleEndian.PutUint64(out, w.Amount)
	return out
}

// WithdrawalRequests implements DerivableList for withdrawal requests.
type WithdrawalRequests []*WithdrawalRequest

// Len returns the length of s.
func (s WithdrawalRequests) Len() int { return len(s) }

// EncodeIndex encodes the i'th withdrawal request to w.
func (s WithdrawalRequests) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

// Requests creates a deep copy of each deposit and returns a slice of the
// withdrwawal requests as Request objects.
func (s WithdrawalRequests) Requests() (reqs Requests) {
	for _, d := range s {
		reqs = append(reqs, NewRequest(d))
	}
	return
}

func (w *WithdrawalRequest) requestType() byte            { return WithdrawalRequestType }
func (w *WithdrawalRequest) encode(b *bytes.Buffer) error { return rlp.Encode(b, w) }
func (w *WithdrawalRequest) decode(input []byte) error    { return rlp.DecodeBytes(input, w) }
func (w *WithdrawalRequest) copy() RequestData {
	return &WithdrawalRequest{
		Source:    w.Source,
		PublicKey: w.PublicKey,
		Amount:    w.Amount,
	}
}
