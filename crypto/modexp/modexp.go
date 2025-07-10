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

package modexp

import (
	"github.com/ethereum/go-ethereum/crypto/modexp/gmp"
)

// ModExp performs modular exponentiation on byte arrays
// result = base^exp mod mod
// This uses the gmp implementation by default.
// To use bigint implementation, import crypto/modexp/bigint directly.
func ModExp(base, exp, mod []byte) ([]byte, error) {
	return gmp.ModExp(base, exp, mod)
}