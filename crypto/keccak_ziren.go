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

//go:build ziren

package crypto

import (
	"github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime"
	"github.com/ethereum/go-ethereum/common"
)

// NewKeccakState creates a new KeccakState
// This uses a Ziren-optimized implementation that leverages the zkvm_runtime.Keccak256 system call.
func NewKeccakState() KeccakState {
	return newZirenKeccakState()
}

// Keccak256 calculates and returns the Keccak256 hash using the Ziren zkvm_runtime implementation.
func Keccak256(data ...[]byte) []byte {
	// For multiple data chunks, concatenate them
	if len(data) == 0 {
		result := zkvm_runtime.Keccak256(nil)
		return result[:]
	}
	if len(data) == 1 {
		result := zkvm_runtime.Keccak256(data[0])
		return result[:]
	}

	// Concatenate multiple data chunks
	var totalLen int
	for _, d := range data {
		totalLen += len(d)
	}

	combined := make([]byte, 0, totalLen)
	for _, d := range data {
		combined = append(combined, d...)
	}

	result := zkvm_runtime.Keccak256(combined)
	return result[:]
}

// Keccak256Hash calculates and returns the Keccak256 hash as a Hash using the Ziren zkvm_runtime implementation.
func Keccak256Hash(data ...[]byte) common.Hash {
	return common.Hash(Keccak256(data...))
}
