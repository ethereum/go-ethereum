// Copyright 2022 The go-ethereum Authors
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

package modexp

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	big2 "github.com/holiman/big"
)

// Fuzz is the fuzzing entry-point.
// The function must return
//
//   - 1 if the fuzzer should increase priority of the
//     given input during subsequent fuzzing (for example, the input is lexically
//     correct and was parsed successfully);
//   - -1 if the input must not be added to corpus even if gives new coverage; and
//   - 0 otherwise
//
// other values are reserved for future use.
func Fuzz(input []byte) int {
	if len(input) <= 96 {
		return -1
	}
	// Abort on too expensive inputs
	precomp := vm.PrecompiledContractsBerlin[common.BytesToAddress([]byte{5})]
	if gas := precomp.RequiredGas(input); gas > 40_000_000 {
		return 0
	}
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()
	)
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return -1
	}
	input = input[96:]
	// Retrieve the operands and execute the exponentiation
	var (
		base  = new(big.Int).SetBytes(getData(input, 0, baseLen))
		exp   = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod   = new(big.Int).SetBytes(getData(input, baseLen+expLen, modLen))
		base2 = new(big2.Int).SetBytes(getData(input, 0, baseLen))
		exp2  = new(big2.Int).SetBytes(getData(input, baseLen, expLen))
		mod2  = new(big2.Int).SetBytes(getData(input, baseLen+expLen, modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return -1
	}
	var a = new(big2.Int).Exp(base2, exp2, mod2).String()
	var b = new(big.Int).Exp(base, exp, mod).String()
	if a != b {
		panic(fmt.Sprintf("Inequality %#x ^ %#x mod %#x \n have %s\n want %s", base, exp, mod, a, b))
	}
	return 1
}

// getData returns a slice from the data based on the start and size and pads
// up to size with zero's. This function is overflow safe.
func getData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return common.RightPadBytes(data[start:end], int(size))
}
