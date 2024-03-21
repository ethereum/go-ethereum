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
	"crypto/rand"
	_ "embed"
	"math/bits"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/exp/slices"
)

func TestJumpDestAnalysis(t *testing.T) {
	tests := []struct {
		code  []byte
		exp   byte
		which int
	}{
		{[]byte{byte(PUSH1), 0x01, 0x01, 0x01}, 0b0000_0010, 0},
		{[]byte{byte(PUSH1), byte(PUSH1), byte(PUSH1), byte(PUSH1)}, 0b0000_1010, 0},
		{[]byte{0x00, byte(PUSH1), 0x00, byte(PUSH1), 0x00, byte(PUSH1), 0x00, byte(PUSH1)}, 0b0101_0100, 0},
		{[]byte{byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), byte(PUSH8), 0x01, 0x01, 0x01}, bits.Reverse8(0x7F), 0},
		{[]byte{byte(PUSH8), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b0000_0001, 1},
		{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, byte(PUSH2), byte(PUSH2), byte(PUSH2), 0x01, 0x01, 0x01}, 0b1100_0000, 0},
		{[]byte{0x01, 0x01, 0x01, 0x01, 0x01, byte(PUSH2), 0x01, 0x01, 0x01, 0x01, 0x01}, 0b0000_0000, 1},
		{[]byte{byte(PUSH3), 0x01, 0x01, 0x01, byte(PUSH1), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b0010_1110, 0},
		{[]byte{byte(PUSH3), 0x01, 0x01, 0x01, byte(PUSH1), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b0000_0000, 1},
		{[]byte{0x01, byte(PUSH8), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b1111_1100, 0},
		{[]byte{0x01, byte(PUSH8), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b0000_0011, 1},
		{[]byte{byte(PUSH16), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b1111_1110, 0},
		{[]byte{byte(PUSH16), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b1111_1111, 1},
		{[]byte{byte(PUSH16), 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}, 0b0000_0001, 2},
		{[]byte{byte(PUSH8), 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, byte(PUSH1), 0x01}, 0b1111_1110, 0},
		{[]byte{byte(PUSH8), 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, byte(PUSH1), 0x01}, 0b0000_0101, 1},
		{[]byte{byte(PUSH32)}, 0b1111_1110, 0},
		{[]byte{byte(PUSH32)}, 0b1111_1111, 1},
		{[]byte{byte(PUSH32)}, 0b1111_1111, 2},
		{[]byte{byte(PUSH32)}, 0b1111_1111, 3},
		{[]byte{byte(PUSH32)}, 0b0000_0001, 4},
	}
	for i, test := range tests {
		ret := codeBitmap(test.code)
		if ret[test.which] != test.exp {
			t.Fatalf("test %d: expected %x, got %02x", i, test.exp, ret[test.which])
		}
	}
}

func TestBitVec(t *testing.T) {
	tests := []struct {
		Code []byte
		Want bitVec
	}{
		{[]byte{}, bitVec{0, 0}},
		{[]byte{byte(PUSH1), 0xff, 0x00, 0x00}, bitVec{0b00000000_00000000_00000000_00000010, 0}},
		{[]byte{byte(PUSH2), 0xff, 0xff, 0x00}, bitVec{0b00000000_00000000_00000000_00000110, 0}},
		{
			[]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, byte(PUSH2), 0xff,
				0xff,
			},
			bitVec{0b10000000_00000000_00000000_00000000, 0b00000000_00000000_00000000_00000001, 0},
		},
		{
			[]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, byte(PUSH32),
			},
			bitVec{0b00000000_00000000_00000000_00000000, 0b11111111_11111111_11111111_11111111, 0},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := newCodeBitVec(test.Code)
			if !slices.Equal(test.Want, got) {
				t.Fatalf("(-want +got)\n- %32b\n+ %32b\n", test.Want, got)
			}
		})
	}
}

func FuzzBitVec(f *testing.F) {
	f.Add([]byte{0x1})
	f.Fuzz(func(t *testing.T, code []byte) {
		newBitVec := newCodeBitVec(code)
		oldBitVec := codeBitmap(code)

		for i := range code {
			if newBitVec.isCode(uint64(i)) != oldBitVec.codeSegment(uint64(i)) {
				t.Fatalf("mismatch at %d", i)
			}
		}
	})
}

const analysisCodeSize = 1200 * 1024

func BenchmarkJumpdestAnalysis_1200k(bench *testing.B) {
	// 1.4 ms
	code := make([]byte, analysisCodeSize)
	bench.SetBytes(analysisCodeSize)
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		codeBitmap(code)
	}
}

func BenchmarkJumpdestAnalysis_rand(b *testing.B) {
	b.Run("v=old", func(b *testing.B) {
		code := make([]byte, analysisCodeSize)
		rand.Read(code)

		bv := codeBitmap(code)
		b.SetBytes(int64(len(code)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmapInternal(code, bv)
		}
	})

	b.Run("v=new", func(b *testing.B) {
		code := make([]byte, analysisCodeSize)
		rand.Read(code)

		bv := newCodeBitVec(code)
		b.SetBytes(int64(len(code)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bv.codeBitVec(code)
		}
	})
}

var (
	//go:embed testdata/weth9.bytecode
	hexCodeWETH9 string
	codeWETH9    = common.FromHex(hexCodeWETH9)
)

func BenchmarkJumpdestAnalysis_weth9(b *testing.B) {
	b.Run("v=old", func(b *testing.B) {
		bv := codeBitmap(codeWETH9)
		b.SetBytes(int64(len(codeWETH9)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			codeBitmapInternal(codeWETH9, bv)
		}
	})

	b.Run("v=new", func(b *testing.B) {
		bv := newCodeBitVec(codeWETH9)
		b.SetBytes(int64(len(codeWETH9)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bv.codeBitVec(codeWETH9)
		}
	})
}

func BenchmarkJumpdestHashing_1200k(bench *testing.B) {
	// 4 ms
	code := make([]byte, analysisCodeSize)
	bench.SetBytes(analysisCodeSize)
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		crypto.Keccak256Hash(code)
	}
}

func BenchmarkJumpdestOpAnalysis(b *testing.B) {
	b.Run("v=old", func(b *testing.B) {
		var op OpCode
		bencher := func(b *testing.B) {
			code := make([]byte, analysisCodeSize)
			b.SetBytes(analysisCodeSize)
			for i := range code {
				code[i] = byte(op)
			}
			bits := make(bitvec, len(code)/8+1+4)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := range bits {
					bits[j] = 0
				}
				codeBitmapInternal(code, bits)
			}
		}
		for op = PUSH1; op <= PUSH32; op++ {
			b.Run(op.String(), bencher)
		}
		op = JUMPDEST
		b.Run(op.String(), bencher)
		op = STOP
		b.Run(op.String(), bencher)
	})

	b.Run("v=new", func(b *testing.B) {
		bencher := func(op OpCode) (string, func(b *testing.B)) {
			return op.String(), func(b *testing.B) {
				code := make([]byte, analysisCodeSize)
				b.SetBytes(analysisCodeSize)

				for i := range code {
					code[i] = byte(op)
				}
				bv := newCodeBitVec(code)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for j := range bv {
						bv[j] = 0
					}
					bv.codeBitVec(code)
				}
			}
		}
		for op := PUSH1; op <= PUSH32; op++ {
			b.Run(bencher(op))
		}
		b.Run(bencher(JUMPDEST))
		b.Run(bencher(STOP))
	})
}
