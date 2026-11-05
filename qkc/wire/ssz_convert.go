// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package wire

import (
	"fmt"
	"math/big"
	"slices"
)

// ChainIDToSSZ encodes a chain ID as a little-endian uint256. It rejects nil,
// negative and oversized IDs. Zero is allowed; callers validate shard identity.
func ChainIDToSSZ(id *big.Int) ([UInt256Length]byte, error) {
	var encoded [UInt256Length]byte
	if id == nil || id.Sign() < 0 || id.BitLen() > 256 {
		return encoded, fmt.Errorf("invalid SSZ chain ID: %v", id)
	}
	id.FillBytes(encoded[:])
	slices.Reverse(encoded[:])
	return encoded, nil
}

// ChainIDFromSSZ decodes a little-endian uint256 into an independent chain ID.
// Every input is representable; callers validate shard identity, including zero.
func ChainIDFromSSZ(encoded [UInt256Length]byte) *big.Int {
	slices.Reverse(encoded[:])
	return new(big.Int).SetBytes(encoded[:])
}
