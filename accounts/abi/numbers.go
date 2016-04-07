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

	"github.com/ethereum/go-ethereum/common"
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

// U256 will ensure unsigned 256bit on big nums
func U256(n *big.Int) []byte {
	return common.LeftPadBytes(common.U256(n).Bytes(), 32)
}

func S256(n *big.Int) []byte {
	sint := common.S256(n)
	ret := common.LeftPadBytes(sint.Bytes(), 32)
	if sint.Cmp(common.Big0) < 0 {
		for i, b := range ret {
			if b == 0 {
				ret[i] = 1
				continue
			}
			break
		}
	}

	return ret
}

// S256 will ensure signed 256bit on big nums
func U2U256(n uint64) []byte {
	return U256(big.NewInt(int64(n)))
}

func S2S256(n int64) []byte {
	return S256(big.NewInt(n))
}

// packNum packs the given number (using the reflect value) and will cast it to appropriate number representation
func packNum(value reflect.Value, to byte) []byte {
	switch kind := value.Kind(); kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if to == UintTy {
			return U2U256(value.Uint())
		} else {
			return S2S256(int64(value.Uint()))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if to == UintTy {
			return U2U256(uint64(value.Int()))
		} else {
			return S2S256(value.Int())
		}
	case reflect.Ptr:
		// This only takes care of packing and casting. No type checking is done here. It should be done prior to using this function.
		if to == UintTy {
			return U256(value.Interface().(*big.Int))
		} else {
			return S256(value.Interface().(*big.Int))
		}

	}

	return nil
}

// checks whether the given reflect value is signed. This also works for slices with a number type
func isSigned(v reflect.Value) bool {
	switch v.Type() {
	case int_ts, int8_ts, int16_ts, int32_ts, int64_ts, int_t, int8_t, int16_t, int32_t, int64_t:
		return true
	}
	return false
}
