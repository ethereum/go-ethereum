// Copyright 2014 The go-ethereum Authors
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

package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var (
	benchAddr     = common.HexToAddress("0x970e8128ab834e8eac17ab8e3812f010678cf791")
	benchSalt     = [32]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f}
	benchInitHash = Keccak256([]byte("test init code hash"))
)

// BenchmarkCreateAddress2 benchmarks CreateAddress2 with fixed inputs
func BenchmarkCreateAddress2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateAddress2(benchAddr, benchSalt, benchInitHash)
	}
}

// BenchmarkCreateAddress2_ZeroSalt benchmarks CreateAddress2 with zero salt
func BenchmarkCreateAddress2_ZeroSalt(b *testing.B) {
	var zeroSalt [32]byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateAddress2(benchAddr, zeroSalt, benchInitHash)
	}
}

// BenchmarkCreateAddress2_RandomSalt benchmarks CreateAddress2 with random salt each iteration
func BenchmarkCreateAddress2_RandomSalt(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var salt [32]byte
		rand.Read(salt[:])
		_ = CreateAddress2(benchAddr, salt, benchInitHash)
	}
}

// BenchmarkCreateAddress2_RandomInputs benchmarks CreateAddress2 with all random inputs
func BenchmarkCreateAddress2_RandomInputs(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var addr common.Address
		var salt [32]byte
		rand.Read(addr[:])
		rand.Read(salt[:])
		initHash := make([]byte, 32)
		rand.Read(initHash)
		_ = CreateAddress2(addr, salt, initHash)
	}
}
