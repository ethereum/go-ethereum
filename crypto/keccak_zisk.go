// Copyright 2026 The go-ethereum Authors
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

//go:build zisk

package crypto

import (
	"github.com/0xPolygonHermez/zisk/ziskasm/lang/go"
	"github.com/ethereum/go-ethereum/common"
)

// ziskKeccakState implements KeccakState by buffering the input.
type ziskKeccakState struct {
	buf    []byte
	result common.Hash
	hashed bool // result holds the hash of buf
}

func (s *ziskKeccakState) Write(p []byte) (int, error) {
	s.buf = append(s.buf, p...)
	s.hashed = false
	return len(p), nil
}

func (s *ziskKeccakState) Sum(b []byte) []byte {
	s.hash()
	return append(b, s.result[:]...)
}

func (s *ziskKeccakState) Reset() {
	s.buf = s.buf[:0]
	s.hashed = false
}

func (s *ziskKeccakState) Size() int      { return 32 }
func (s *ziskKeccakState) BlockSize() int { return 136 }

// Read gives the 32-byte digest; a KeccakState is never squeezed further.
func (s *ziskKeccakState) Read(p []byte) (int, error) {
	s.hash()
	return copy(p, s.result[:]), nil
}

func (s *ziskKeccakState) hash() {
	if !s.hashed {
		s.result = zkvm.Keccak256(s.buf)
		s.hashed = true
	}
}

// NewKeccakState creates a new KeccakState.
func NewKeccakState() KeccakState {
	return &ziskKeccakState{buf: make([]byte, 0, 256)}
}

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	h := Keccak256Hash(data...)
	return h[:]
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) common.Hash {
	if len(data) == 1 {
		return zkvm.Keccak256(data[0])
	}
	n := 0
	for _, d := range data {
		n += len(d)
	}
	buf := make([]byte, 0, n)
	for _, d := range data {
		buf = append(buf, d...)
	}
	return zkvm.Keccak256(buf)
}
