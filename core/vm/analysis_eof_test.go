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

package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEOFAnalysis(t *testing.T) {
	tests := []struct {
		code  []byte
		exp   byte
		which int
	}{
		{[]byte{byte(RJUMP), 0x01, 0x01, 0x01}, 0b0000_0110, 0},
		{[]byte{byte(RJUMPI), byte(RJUMP), byte(RJUMP), byte(RJUMPI)}, 0b0011_0110, 0},
		{[]byte{byte(RJUMPV), 0x02, byte(RJUMP), 0x00, byte(RJUMPI), 0x00}, 0b0011_1110, 0},
	}
	for i, test := range tests {
		ret := eofCodeBitmap(test.code)
		if ret[test.which] != test.exp {
			t.Fatalf("test %d: expected %x, got %02x", i, test.exp, ret[test.which])
		}
	}
}

func BenchmarkJumpdestOpEOFAnalysis(bench *testing.B) {
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
			clear(bits)
			eofCodeBitmapInternal(code, bits)
		}
	}
	for op = PUSH1; op <= PUSH32; op++ {
		bench.Run(op.String(), bencher)
	}
	op = JUMPDEST
	bench.Run(op.String(), bencher)
	op = STOP
	bench.Run(op.String(), bencher)
	op = RJUMPV
	bench.Run(op.String(), bencher)
	op = EOFCREATE
	bench.Run(op.String(), bencher)
}

func FuzzCodeAnalysis(f *testing.F) {
	f.Add(common.FromHex("5e30303030"))
	f.Fuzz(func(t *testing.T, data []byte) {
		eofCodeBitmap(data)
	})
}
