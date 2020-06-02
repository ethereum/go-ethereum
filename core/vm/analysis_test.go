// Copyright 2017 The go-ethereum Authors
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

package vm

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestJumpDestAnalysis(t *testing.T) {
	tests := []struct {
		code  []byte
		exp   byte
		which int
	}{
		{[]byte{byte(PUSH1), 0x01, 0x01, 0x01}, 0x40, 0},
		{[]byte{byte(PUSH1), byte(PUSH1), byte(PUSH1), byte(PUSH1)}, 0x50, 0},
		{[]byte{byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), 0x01, 0x01, 0x01}, 0x7F, 0},
		{[]byte{byte(PUSH8), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0x80, 1},
		{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, byte(PUSH2), byte(PUSH2), byte(PUSH2), 0x01, 0x01, 0x01}, 0x03, 0},
		{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, byte(PUSH2), 0x01, 0x01, 0x01, 0x01, 0x01}, 0x00, 1},
		{[]byte{byte(PUSH3), 0x01, 0x01, 0x01, byte(PUSH1), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0x74, 0},
		{[]byte{byte(PUSH3), 0x01, 0x01, 0x01, byte(PUSH1), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0x00, 1},
		{[]byte{0x01, byte(PUSH8), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0x3F, 0},
		{[]byte{0x01, byte(PUSH8), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0xC0, 1},
		{[]byte{byte(PUSH16), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0x7F, 0},
		{[]byte{byte(PUSH16), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0xFF, 1},
		{[]byte{byte(PUSH16), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0x80, 2},
		{[]byte{byte(PUSH8), 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, byte(PUSH1), 0x01}, 0x7f, 0},
		{[]byte{byte(PUSH8), 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, byte(PUSH1), 0x01}, 0xA0, 1},
		{[]byte{byte(PUSH32)}, 0x7F, 0},
		{[]byte{byte(PUSH32)}, 0xFF, 1},
		{[]byte{byte(PUSH32)}, 0xFF, 2},
	}
	for _, test := range tests {
		ret := codeBitmap(test.code)
		if ret[test.which] != test.exp {
			t.Fatalf("expected %x, got %02x", test.exp, ret[test.which])
		}
	}
}

// Helper functions to create worst-case scenarios for the jumpdest analyzer
func codeEmpty(size int) []byte { return make([]byte, size) }

func codeFill(size int, op OpCode) []byte {
	code := make([]byte, size)
	for index, _ := range code {
		code[index] = byte(op)
	}
	return code
}

// BenchmarkJumpdestHashing_1200k benchmarks a segment of code consisting of
// 1.2M bytes
func BenchmarkJumpdestAnalysis_1200k(bench *testing.B) {
	// 1.4 ms
	size := 1200000
	bench.Run("zeroes", func(b *testing.B) {
		code := codeFill(size, STOP)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmap(code)
		}
		b.StopTimer()
	})

	bench.Run("jumpdests", func(b *testing.B) {
		code := codeFill(size, JUMPDEST)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmap(code)
		}
		b.StopTimer()
	})

	bench.Run("push32", func(b *testing.B) {
		code := codeFill(size, PUSH32)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmap(code)
		}
		b.StopTimer()
	})
	// This is the worst case for current implementation
	bench.Run("push1", func(b *testing.B) {
		code := codeFill(size, PUSH1)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmap(code)
		}
		b.StopTimer()
	})
	bench.Run("beginsub", func(b *testing.B) {
		code := codeFill(size, BEGINSUB)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmap(code)
		}
		b.StopTimer()
	})
	bench.Run("beginsub_push", func(b *testing.B) {
		// Combine both worst cases
		code := codeFill(size, PUSH1)
		for index := 0; index < len(code); index += 32 {
			code[index] = byte(BEGINSUB)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmap(code)
		}
		b.StopTimer()
	})
}

func BenchmarkJumpdestValidation(b *testing.B) {
	size := 24000
	b.Run("jumpdests", func(b *testing.B) {
		code := codeFill(size, JUMPDEST)
		analysis := codeBitmap(code)
		//dest := new(big.Int)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for index, _ := range code {
				//dest.SetUint64(uint64(len(code)))
				analysis.codeSegment(uint64(index))
			}
		}
		b.StopTimer()
	})
}

func BenchmarkHashing_1200k(bench *testing.B) {
	// 4 ms
	code := make([]byte, 1200000)
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		crypto.Keccak256Hash(code)
	}
	bench.StopTimer()
}

func TestMemCost(t *testing.T) {
	words := 1024 * 1024 / 32
	cost, _ := memoryGasCost(NewMemory(), 32*uint64(words))
	fmt.Printf("Cost: %d\n", cost)
}
