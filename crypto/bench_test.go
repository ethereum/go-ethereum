// Copyright 2022 The go-ethereum Authors
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
	"bytes"
	"testing"
)

var rows = [][][]byte{
	{[]byte("abcdef"), []byte("ghijklm")},
	{[]byte("ABCDEF"), []byte("GHIJKLM")},
	{[]byte("123456789"), []byte("XXXXXXX")},
	{[]byte("1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"), bytes.Repeat([]byte("abcdef"), 10), bytes.Repeat([]byte("a"), 26)},
	{[]byte("1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"), bytes.Repeat([]byte("a"), 101), bytes.Repeat([]byte("a"), 31)},
	{[]byte("1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"), bytes.Repeat([]byte("a"), 100), bytes.Repeat([]byte("a"), 256)},
}

var sink interface{}

func BenchmarkKeccak256(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, row := range rows {
			hash := Keccak256(row...)
			b.SetBytes(int64(len(hash)))
			sink = hash
		}
	}

	if sink == nil {
		b.Fatal("Benchmark did not run")
	}

	sink = (interface{})(nil)
}

func BenchmarkKeccak256Hash(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, row := range rows {
			hash := Keccak256Hash(row...)
			b.SetBytes(int64(len(hash)))
			sink = hash
		}
	}

	if sink == nil {
		b.Fatal("Benchmark did not run")
	}

	sink = (interface{})(nil)
}

func BenchmarkKeccak512(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, row := range rows {
			hash := Keccak512(row...)
			b.SetBytes(int64(len(hash)))
			sink = hash
		}
	}

	if sink == nil {
		b.Fatal("Benchmark did not run")
	}

	sink = (interface{})(nil)
}
