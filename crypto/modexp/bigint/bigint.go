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

package bigint

import (
	"math/big"
)

// ModExp performs modular exponentiation using Go's big.Int
// result = base^exp mod mod
func ModExp(base, exp, mod []byte) ([]byte, error) {
	baseBig := new(big.Int).SetBytes(base)
	expBig := new(big.Int).SetBytes(exp)
	modBig := new(big.Int).SetBytes(mod)

	// Handle special cases
	if modBig.BitLen() == 0 {
		// Modulo 0 is undefined, return empty bytes
		return []byte{}, nil
	}

	if baseBig.BitLen() == 1 {
		// If base == 1, then we can just return base % mod
		result := baseBig.Mod(baseBig, modBig)
		return result.Bytes(), nil
	}

	// Perform modular exponentiation
	result := new(big.Int).Exp(baseBig, expBig, modBig)
	return result.Bytes(), nil
}
