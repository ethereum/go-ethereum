// Copyright 2017 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/abi"
)

var (
	// MaxUint256 is the maximum value that can be represented by a uint256.
	MaxUint256 = abi.MaxUint256
	// MaxInt256 is the maximum value that can be represented by a int256.
	MaxInt256 = abi.MaxInt256
)

// ReadInteger reads the integer based on its kind and returns the appropriate value.
func ReadInteger(typ Type, b []byte) interface{} {
	return abi.ReadInteger(typ, b)
}

// ReadFixedBytes uses reflection to create a fixed array to be read from.
func ReadFixedBytes(t Type, word []byte) (interface{}, error) {
	return abi.ReadFixedBytes(t, word)
}
