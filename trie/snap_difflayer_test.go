// Copyright 2019 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

// randomHash generates a random blob of data and returns it as a hash.
func randomHash() common.Hash {
	var hash common.Hash
	if n, err := rand.Read(hash[:]); n != common.HashLength || err != nil {
		panic(err)
	}
	return hash
}

func randomNode() *memoryNode {
	val := randBytes(100)
	return &memoryNode{
		hash: crypto.Keccak256Hash(val),
		node: rawNode(val),
		size: 100,
	}
}

func emptyLayer() *diskLayer {
	return &diskLayer{
		db:    openSnapDatabase(rawdb.NewMemoryDatabase(), nil, nil),
		dirty: newDiskcache(defaultCacheSize, nil, 0),
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkSearch128Layers
// BenchmarkSearch128Layers-8   	  243826	      4755 ns/op
func BenchmarkSearch128Layers(b *testing.B) { benchmarkSearch(b, 0, 128) }

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkSearch512Layers
// BenchmarkSearch512Layers-8   	   49686	     24256 ns/op
func BenchmarkSearch512Layers(b *testing.B) { benchmarkSearch(b, 0, 512) }

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkSearch1Layer
// BenchmarkSearch1Layer-8   	14062725	        88.40 ns/op
func BenchmarkSearch1Layer(b *testing.B) { benchmarkSearch(b, 127, 128) }

func benchmarkSearch(b *testing.B, depth int, total int) {
	var (
		npath []byte
		nhash common.Hash
		nblob []byte
	)
	// First, we set up 128 diff layers, with 3K items each
	fill := func(parent snapshot, index int) *diffLayer {
		nodes := make(map[common.Hash]map[string]*nodeWithPrev)
		nodes[common.Hash{}] = make(map[string]*nodeWithPrev)
		for i := 0; i < 3000; i++ {
			var (
				path = randomHash().Bytes()
				node = randomNode()
				blob = node.rlp()
			)
			nodes[common.Hash{}][string(path)] = &nodeWithPrev{
				memoryNode: node,
				prev:       nil,
			}
			if npath == nil && depth == index {
				npath = common.CopyBytes(path)
				nblob = common.CopyBytes(blob)
				nhash = crypto.Keccak256Hash(blob)
			}
		}
		return newDiffLayer(parent, common.Hash{}, 0, nodes)
	}
	var layer snapshot
	layer = emptyLayer()
	for i := 0; i < total; i++ {
		layer = fill(layer, i)
	}
	b.ResetTimer()

	var (
		have []byte
		err  error
	)
	for i := 0; i < b.N; i++ {
		have, err = layer.NodeBlob(common.Hash{}, npath, nhash)
		if err != nil {
			b.Fatal(err)
		}
	}
	if !bytes.Equal(have, nblob) {
		b.Fatalf("have %x want %x", have, nblob)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkGetNode
// BenchmarkGetNode-8   	 7024152	       168.0 ns/op
func BenchmarkGetNode(b *testing.B) { benchmarkGetNode(b, false) }

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkGetNodeBlob
// BenchmarkGetNodeBlob-8   	 6826884	       170.6 ns/op
func BenchmarkGetNodeBlob(b *testing.B) { benchmarkGetNode(b, true) }

func benchmarkGetNode(b *testing.B, getBlob bool) {
	db := newTestDatabase(rawdb.NewDatabase(rawdb.NewMemoryDatabase()), rawdb.PathScheme)
	trie, _ := New(TrieID(common.Hash{}), db)

	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		binary.LittleEndian.PutUint64(k, uint64(i))
		trie.Update(k, randBytes(100))
	}
	root, nodes, _ := trie.Commit(false)
	db.Update(root, common.Hash{}, NewWithNodeSet(nodes))

	var (
		path  []byte
		hash  common.Hash
		layer = db.GetReader(root)
	)
	for p, node := range nodes.nodes {
		path = []byte(p)
		hash = node.hash
		break
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if getBlob {
			layer.NodeBlob(common.Hash{}, path, hash)
		} else {
			layer.Node(common.Hash{}, path, hash)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkPersist
// BenchmarkPersist-8   	      10	 111252975 ns/op
func BenchmarkPersist(b *testing.B) {
	// First, we set up 128 diff layers, with 3K items each
	fill := func(parent snapshot) *diffLayer {
		nodes := make(map[common.Hash]map[string]*nodeWithPrev)
		nodes[common.Hash{}] = make(map[string]*nodeWithPrev)
		for i := 0; i < 3000; i++ {
			var (
				path = randomHash().Bytes()
				node = randomNode()
			)
			nodes[common.Hash{}][string(path)] = &nodeWithPrev{
				memoryNode: node,
				prev:       nil,
			}
		}
		return newDiffLayer(parent, common.Hash{}, 0, nodes)
	}
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var layer snapshot
		layer = emptyLayer()
		for i := 1; i < 128; i++ {
			layer = fill(layer)
		}
		b.StartTimer()

		dl, ok := layer.(*diffLayer)
		if !ok {
			break
		}
		dl.persist(false)
	}
}

// BenchmarkJournal benchmarks the performance for journaling the layers.
//
// BenchmarkJournal
// BenchmarkJournal-8   	      10	 110969279 ns/op
func BenchmarkJournal(b *testing.B) {
	// First, we set up 128 diff layers, with 3K items each
	fill := func(parent snapshot) *diffLayer {
		nodes := make(map[common.Hash]map[string]*nodeWithPrev)
		nodes[common.Hash{}] = make(map[string]*nodeWithPrev)
		for i := 0; i < 3000; i++ {
			var (
				path = randomHash().Bytes()
				node = randomNode()
			)
			nodes[common.Hash{}][string(path)] = &nodeWithPrev{
				memoryNode: node,
				prev:       nil,
			}
		}
		return newDiffLayer(parent, common.Hash{}, 0, nodes)
	}
	var layer snapshot
	layer = emptyLayer()
	for i := 0; i < 128; i++ {
		layer = fill(layer)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		layer.Journal(new(bytes.Buffer))
	}
}
