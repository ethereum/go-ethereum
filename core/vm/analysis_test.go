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
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

var codeVsDataTests = []struct {
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

func TestJumpDestAnalysis(t *testing.T) {
	for _, test := range codeVsDataTests {
		ret := codeBitmap(test.code)
		if ret[test.which] != test.exp {
			t.Fatalf("expected %x, got %02x", test.exp, ret[test.which])
		}
	}
}

func TestShadowAnalysisCodeAndData(t *testing.T) {
	for i, test := range codeVsDataTests {
		shadow := shadowMap(test.code)
		ret := codeBitmap(test.code)
		for c, _ := range test.code {
			exp := ret.codeSegment(uint64(c))
			got := shadow.IsCode(uint16(c))
			if got != exp {
				t.Fatalf("test %d: loc %d: expected %v, got %v", i, c, exp, got)
			}
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

func TestLEB(t *testing.T) {
	inputs := []uint16{0, 1, 31, 64, 1023, 1024, 0xc000, 0xffff}
	exp := []string{"000000", "010000", "1f0000", "400100", "7f0f00", "401000", "40400c", "7f7f0f"}
	for i, num := range inputs {
		bitmap := make([]byte, 3)
		lebEncode(num, bitmap)
		if expb := common.FromHex(exp[i]); !bytes.Equal(bitmap, expb) {
			t.Errorf("testcase %d: expected %x, got %x", i, expb, bitmap)
		}
	}
}

func TestLebEncodeDecode(t *testing.T) {
	data := make([]byte, 3)
	zero := make([]byte, 3)
	for i := 0; i < 65536; i++ {
		// clear buf
		copy(data, zero)
		exp := uint16(i)
		lebEncode(exp, data)
		got := lebDecode(data)
		if exp != got {
			t.Fatalf("exp %d, got %d: %x", exp, got, data)
		}
	}
}

func TestShadowAnalysis(t *testing.T) {

	type testcase struct {
		code string
		rout string
	}
	cases := []testcase{
		{
			code: "6001600160015c60015c5c5c",
			rout: "000000000000111111223344",
		},
		{
			code: "5c5c5c5c5c5c5c5c5c5c5c5c",
			rout: "112233445566778899aabbcc",
		},
		{
			code: "7f5c5c5c5c5c5c5c5c5c5c5c",
			rout: "000000000000000000000000",
		},
		{
			code: "615c5c5c00605c5c7f5c5c5c",
			rout: "000000111111112222222222",
		},
	}
	for i, tc := range cases {
		code := common.FromHex(tc.code)
		shadow := shadowMap(code)
		routines := common.FromHex(tc.rout)
		substart := 0
		prev := byte(0)
		for x := 0; x < len(code); x++ {
			if routines[x] != prev {
				substart = x
				prev = routines[x]
			}
			for y := 0; y < len(code); y++ {
				got := shadow.isSameSubroutine(uint16(substart), uint16(y))
				exp := routines[substart] == routines[y]
				if got != exp {
					t.Fatalf("test %d: is %d at subroutine %d? got %v exp %v",
						i, y, substart, got, exp)
				}
			}
		}
	}
}

//BenchmarkLEB/encode-6         	346602476	         3.06 ns/op	       0 B/op	       0 allocs/op
//BenchmarkLEB/decode-6         	362383606	         3.48 ns/op	       0 B/op	       0 allocs/op
func BenchmarkLEB(b *testing.B) {
	bitmap := make([]byte, 20)
	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			lebEncode(0xffff, bitmap)
			lebEncode(0xff, bitmap)
			lebEncode(0x0, bitmap)
		}
	})
	var (
		x = make([]byte, 20)
		y = make([]byte, 20)
		z = make([]byte, 20)
	)
	lebEncode(0xffff, x)
	lebEncode(0xff, y)
	lebEncode(0x0, z)
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			lebDecode(x)
			lebDecode(y)
			lebDecode(z)
		}
	})
}

// BenchmarkJumpdestHashing_1200k benchmarks a segment of code consisting of
// 1.2M bytes
func BenchmarkJumpdestAnalysis_1200k(b *testing.B) {
	benchJumpdestAnalysis(1200000, b)
}

func BenchmarkJumpdestAnalysis_49152(b *testing.B) {
	benchJumpdestAnalysis(49152, b)
}
func benchJumpdestAnalysis(size int, bench *testing.B) {
	// 1.4 ms
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

func BenchmarkShadowAnalysis_49152(b *testing.B) {
	benchShadowAnalysis(49152, b)
}

func benchShadowAnalysis(size int, bench *testing.B) {
	// 1.4 ms
	bench.Run("zeroes", func(b *testing.B) {
		code := codeFill(size, STOP)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			shadowMap(code)
		}
		b.StopTimer()
	})

	bench.Run("jumpdests", func(b *testing.B) {
		code := codeFill(size, JUMPDEST)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			shadowMap(code)
		}
		b.StopTimer()
	})

	bench.Run("push32", func(b *testing.B) {
		code := codeFill(size, PUSH32)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			shadowMap(code)
		}
		b.StopTimer()
	})
	// This is the worst case for current implementation
	bench.Run("push1", func(b *testing.B) {
		code := codeFill(size, PUSH1)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			shadowMap(code)
		}
		b.StopTimer()
	})
	bench.Run("beginsub", func(b *testing.B) {
		code := codeFill(size, BEGINSUB)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			shadowMap(code)
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
			shadowMap(code)
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

func BenchmarkHashing_49152(bench *testing.B) {
	// 4 ms
	code := make([]byte, 49152)
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
