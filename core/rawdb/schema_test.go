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

package rawdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var benchHash = common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

func BenchmarkHeaderKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		headerKey(123456789, benchHash)
	}
}

func BenchmarkHeaderHashKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		headerHashKey(123456789)
	}
}

func BenchmarkHeaderKeyPrefix(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		headerKeyPrefix(123456789)
	}
}

func BenchmarkBlockBodyKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		blockBodyKey(123456789, benchHash)
	}
}

func BenchmarkBlockReceiptsKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		blockReceiptsKey(123456789, benchHash)
	}
}

func BenchmarkSkeletonHeaderKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		skeletonHeaderKey(123456789)
	}
}

func BenchmarkEncodeBlockNumber(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		encodeBlockNumber(123456789)
	}
}
