// Copyright 2023 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package trienode

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func BenchmarkMerge(b *testing.B) {
	b.Run("1K", func(b *testing.B) {
		benchmarkMerge(b, 1000)
	})
	b.Run("10K", func(b *testing.B) {
		benchmarkMerge(b, 10_000)
	})
}

func benchmarkMerge(b *testing.B, count int) {
	x := NewNodeSet(common.Hash{})
	y := NewNodeSet(common.Hash{})
	addNode := func(s *NodeSet) {
		path := make([]byte, 4)
		rand.Read(path)
		blob := make([]byte, 32)
		rand.Read(blob)
		hash := crypto.Keccak256Hash(blob)
		s.AddNode(path, New(hash, blob))
	}
	for i := 0; i < count; i++ {
		// Random path of 4 nibbles
		addNode(x)
		addNode(y)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Store set x into a backup
		z := NewNodeSet(common.Hash{})
		z.Merge(common.Hash{}, x.Nodes)
		// Merge y into x
		x.Merge(common.Hash{}, y.Nodes)
		x = z
	}
}
