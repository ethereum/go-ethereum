// Copyright 2016 The go-ethereum Authors
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

package math

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// wordSize is the size number of bits in a big.Int Word.
const wordSize = 32 << (uint64(^big.Word(0)) >> 63)

// Exp implement exponentiation by squaring algorithm.
//
// Exp return a new variable; base and exponent must
// not be changed under any circumstance.
//
// Courtesy @karalabe and @chfast
func Exp(base, exponent *big.Int) *big.Int {
	result := big.NewInt(1)

	for _, word := range exponent.Bits() {
		for i := 0; i < wordSize; i++ {
			if word&1 == 1 {
				common.U256(result.Mul(result, base))
			}
			common.U256(base.Mul(base, base))
			word >>= 1
		}
	}
	return result
}
