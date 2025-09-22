// Copyright 2025 The go-ethereum Authors
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

//go:build !ziren

package platcrypto

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Keccak256 calculates and returns the Keccak256 hash using the standard implementation.
// This is used for geth, evm, and other regular programs.
func Keccak256(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}

// Keccak256Hash calculates and returns the Keccak256 hash as a Hash using the standard implementation.
// This is used for geth, evm, and other regular programs.
func Keccak256Hash(data ...[]byte) common.Hash {
	return crypto.Keccak256Hash(data...)
}

// NewKeccakState returns a new keccak state hasher using the standard implementation.
// This is used for geth, evm, and other regular programs.
func NewKeccakState() crypto.KeccakState {
	return crypto.NewKeccakState()
}
