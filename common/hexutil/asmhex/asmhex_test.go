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

package asmhex

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func BenchmarkAsmHexDecode(b *testing.B) {
	for _, size := range []int{64, 256, 1024, 4096, 16384, 65536} {
		src := bytes.Repeat([]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'B', 'c', 'D', 'e', 'f'}, size/8)
		sink := make([]byte, size)

		b.Run(fmt.Sprintf("%v", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := Decode(sink, src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkStdlibDecode(b *testing.B) {
	for _, size := range []int{64, 256, 1024, 4096, 16384, 65536} {
		src := bytes.Repeat([]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'B', 'c', 'D', 'e', 'f'}, size/8)
		sink := make([]byte, size)

		b.Run(fmt.Sprintf("%v", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := hex.Decode(sink, src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
