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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type ConsolidationRequest -field-override consolidationRequestMarshaling -out gen_consolidation_request_json.go

// ConsolidationRequest represents an EIP-7251 consolidation request from source for
// the validator associated with the source public key to a target public key.
type ConsolidationRequest struct {
	Source          common.Address `json:"sourceAddress"`
	SourcePublicKey [48]byte       `json:"sourcePubkey"`
	TargetPublicKey [48]byte       `json:"targetPubkey"`
}

// field type overrides for gencodec
type consolidationRequestMarshaling struct {
	SourcePublicKey hexutil.Bytes
	TargetPublicKey hexutil.Bytes
}

func (c *ConsolidationRequest) Bytes() []byte {
	out := make([]byte, 116)
	copy(out, c.Source.Bytes())
	copy(out[20:], c.SourcePublicKey[:])
	copy(out[68:], c.TargetPublicKey[:])
	return out
}

// ConsolidationRequests implements DerivableList for consolidation requests.
type ConsolidationRequests []*ConsolidationRequest

// Len returns the length of s.
func (s ConsolidationRequests) Len() int { return len(s) }

// EncodeIndex encodes the i'th consolidation request to c.
func (s ConsolidationRequests) EncodeIndex(i int, c *bytes.Buffer) {
	rlp.Encode(c, s[i])
}

// Requests creates a deep copy of each deposit and returns a slice of the
// withdrwawal requests as Request objects.
func (s ConsolidationRequests) Requests() (reqs Requests) {
	for _, d := range s {
		reqs = append(reqs, NewRequest(d))
	}
	return
}

func (c *ConsolidationRequest) requestType() byte            { return ConsolidationRequestType }
func (c *ConsolidationRequest) encode(b *bytes.Buffer) error { return rlp.Encode(b, c) }
func (c *ConsolidationRequest) decode(input []byte) error    { return rlp.DecodeBytes(input, c) }
func (c *ConsolidationRequest) copy() RequestData {
	return &ConsolidationRequest{
		Source:          c.Source,
		SourcePublicKey: c.SourcePublicKey,
		TargetPublicKey: c.TargetPublicKey,
	}
}
