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
	"math"
	"math/big"

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
		AddTokens []IntOrInf
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

const (
	IntNonNegative = iota
	IntNegative
	IntPlusInf
	IntMinusInf //TODO is this needed?
)

type IntOrInf struct {
	Type  uint8
	Value big.Int
}

func (i *IntOrInf) BigInt() *big.Int {
	switch i.Type {
	case IntNonNegative:
		return new(big.Int).Set(&i.Value)
	case IntNegative:
		return new(big.Int).Neg(&i.Value)
	case IntPlusInf:
		panic(nil) // caller should check Inf() before trying to convert to big.Int
	case IntMinusInf:
		panic(nil)
	}
	return &big.Int{} // invalid type decodes to 0 value
}

func (i *IntOrInf) Inf() int {
	switch i.Type {
	case IntPlusInf:
		return 1
	case IntMinusInf:
		return -1
	}
	return 0 // invalid type decodes to 0 value
}

func (i *IntOrInf) Int64() int64 {
	switch i.Type {
	case IntNonNegative:
		if i.Value.IsInt64() {
			return i.Value.Int64()
		} else {
			return math.MaxInt64
		}
	case IntNegative:
		if i.Value.IsInt64() {
			return -i.Value.Int64()
		} else {
			return math.MinInt64
		}
	case IntPlusInf:
		return math.MaxInt64
	case IntMinusInf:
		return math.MinInt64
	}
	return 0 // invalid type decodes to 0 value
}

func (i *IntOrInf) SetBigInt(v *big.Int) {
	if v.Sign() >= 0 {
		i.Type = IntNonNegative
		i.Value.Set(v)
	} else {
		i.Type = IntNegative
		i.Value.Neg(v)
	}
}

func (i *IntOrInf) SetInt64(v int64) {
	if v >= 0 {
		if v == math.MaxInt64 {
			i.Type = IntPlusInf
		} else {
			i.Type = IntNonNegative
			i.Value.SetInt64(v)
		}
	} else {
		if v == math.MinInt64 {
			i.Type = IntMinusInf
		} else {
			i.Type = IntNegative
			i.Value.SetInt64(-v)
		}
	}
}

func (i *IntOrInf) SetInf(sign int) {
	if sign == 1 {
		i.Type = IntPlusInf
	} else {
		i.Type = IntMinusInf
	}
}
