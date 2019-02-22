// Copyright 2015 The go-ethereum Authors
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

package abi

import (
	"math/big"
	"reflect"

	"github.com/ubiq/go-ubiq/common"
	"github.com/ubiq/go-ubiq/common/math"
)

var (
	big_t     = reflect.TypeOf(big.Int{})
	ubig_t    = reflect.TypeOf(big.Int{})
	byte_t    = reflect.TypeOf(byte(0))
	byte_ts   = reflect.TypeOf([]byte(nil))
	uint_t    = reflect.TypeOf(uint(0))
	uint8_t   = reflect.TypeOf(uint8(0))
	uint16_t  = reflect.TypeOf(uint16(0))
	uint32_t  = reflect.TypeOf(uint32(0))
	uint64_t  = reflect.TypeOf(uint64(0))
	int_t     = reflect.TypeOf(int(0))
	int8_t    = reflect.TypeOf(int8(0))
	int16_t   = reflect.TypeOf(int16(0))
	int32_t   = reflect.TypeOf(int32(0))
	int64_t   = reflect.TypeOf(int64(0))
	hash_t    = reflect.TypeOf(common.Hash{})
	address_t = reflect.TypeOf(common.Address{})

	uint_ts   = reflect.TypeOf([]uint(nil))
	uint8_ts  = reflect.TypeOf([]uint8(nil))
	uint16_ts = reflect.TypeOf([]uint16(nil))
	uint32_ts = reflect.TypeOf([]uint32(nil))
	uint64_ts = reflect.TypeOf([]uint64(nil))
	ubig_ts   = reflect.TypeOf([]*big.Int(nil))

	int_ts   = reflect.TypeOf([]int(nil))
	int8_ts  = reflect.TypeOf([]int8(nil))
	int16_ts = reflect.TypeOf([]int16(nil))
	int32_ts = reflect.TypeOf([]int32(nil))
	int64_ts = reflect.TypeOf([]int64(nil))
	big_ts   = reflect.TypeOf([]*big.Int(nil))
)

// U256 converts a big Int into a 256bit EVM number.
func U256(n *big.Int) []byte {
	return math.PaddedBigBytes(math.U256(n), 32)
}

// checks whether the given reflect value is signed. This also works for slices with a number type
func isSigned(v reflect.Value) bool {
	switch v.Type() {
	case int_ts, int8_ts, int16_ts, int32_ts, int64_ts, int_t, int8_t, int16_t, int32_t, int64_t:
		return true
	}
	return false
}
