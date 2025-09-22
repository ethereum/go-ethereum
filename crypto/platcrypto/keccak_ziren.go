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

package platcrypto

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	zkruntime "github.com/zkMIPS/zkMIPS/crates/go-runtime/zkm_runtime"
)

// zirenKeccak implements keccak256 using the ziren platform precompile
func zirenKeccak(data []byte) []byte {
	return zkruntime.Keccak(data)
}

// zirenKeccakState wraps the ziren platform keccak precompile to implement crypto.KeccakState interface
type zirenKeccakState struct {
	data []byte
}

func (k *zirenKeccakState) Reset() {
	k.data = k.data[:0]
}

func (k *zirenKeccakState) Clone() crypto.KeccakState {
	clone := &zirenKeccakState{
		data: make([]byte, len(k.data)),
	}
	copy(clone.data, k.data)
	return clone
}

func (k *zirenKeccakState) Write(data []byte) (int, error) {
	k.data = append(k.data, data...)
	return len(data), nil
}

func (k *zirenKeccakState) Read(hash []byte) (int, error) {
	if len(hash) < 32 {
		return 0, errors.New("hash slice too short")
	}
	
	result := zirenKeccak(k.data)
	copy(hash[:32], result)
	return 32, nil
}

func (k *zirenKeccakState) Sum(data []byte) []byte {
	hash := make([]byte, 32)
	k.Read(hash)
	return append(data, hash...)
}

func (k *zirenKeccakState) Size() int {
	return 32
}

func (k *zirenKeccakState) BlockSize() int {
	return 136 // keccak256 block size
}

// Keccak256 calculates and returns the Keccak256 hash using the ziren platform precompile.
func Keccak256(data ...[]byte) []byte {
	hasher := &zirenKeccakState{}
	for _, b := range data {
		hasher.Write(b)
	}
	hash := make([]byte, 32)
	hasher.Read(hash)
	return hash
}

// Keccak256Hash calculates and returns the Keccak256 hash as a Hash using the ziren platform precompile.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	hash := Keccak256(data...)
	copy(h[:], hash)
	return h
}

// NewKeccakState returns a new keccak state hasher using the ziren platform precompile.
func NewKeccakState() crypto.KeccakState {
	return &zirenKeccakState{}
}
