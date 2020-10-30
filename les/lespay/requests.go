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

package lespay

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"
)

var ErrNoReply = errors.New("no reply for given request")

const (
	CapacityQueryName = "cq"
)

type (
	Request struct {
		Service, Name string
		Params        []byte
	}
	Requests []Request

	Replies [][]byte

	CapacityQueryReq struct {
		Bias      uint64 // seconds
		AddTokens []uint64
	}
	CapacityQueryReply []uint64
)

func (r *Requests) Add(service, name string, val interface{}) (int, error) {
	enc, err := rlp.EncodeToBytes(val)
	if err != nil {
		return -1, err
	}
	*r = append(*r, Request{
		Service: service,
		Name:    name,
		Params:  enc,
	})
	return len(*r) - 1, nil
}

func (r Replies) Get(i int, val interface{}) error {
	if i < 0 || i >= len(r) {
		return ErrNoReply
	}
	return rlp.DecodeBytes(r[i], val)
}
