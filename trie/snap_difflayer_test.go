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

	"github.com/VictoriaMetrics/fastcache"
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

func randomNode() *cachedNode {
	val := randBytes(100)
	return &cachedNode{
		hash: crypto.Keccak256Hash(val),
		node: rawNode(val),
		size: 100,
	}
}

func randomEmptyNode() *cachedNode {
	return &cachedNode{
		hash: common.Hash{},
		node: nil,
		size: 0,
	}
}

func emptyLayer() *diskLayer {
	return &diskLayer{
		diskdb: rawdb.NewDatabase(rawdb.NewMemoryDatabase()),
		clean:  fastcache.New(500 * 1024),
		dirty:  newDiskcache(nil, 0),
	}
}

func benchmarkSearch(b *testing.B, depth int) {
	var (
		target     []byte
		targetHash common.Hash
		want       []byte
	)
	// First, we set up 128 diff layers, with 3K items each
	fill := func(parent snapshot, index int) *diffLayer {
		var nodes = make(map[string]*nodeWithPreValue)
		for i := 0; i < 3000; i++ {
			var (
				path    = randomHash().Bytes()
				storage = EncodeStorageKey(common.Hash{}, path)
				val     = randomNode()
			)
			nodes[string(storage)] = &nodeWithPreValue{
				cachedNode: val,
				pre:        nil,
			}
			if target == nil && depth == index {
				want = val.rlp()
				target = append([]byte{}, storage...)
				targetHash = val.hash
			}
		}
		return newDiffLayer(parent, common.Hash{}, 0, nodes)
	}
	var layer snapshot
	layer = emptyLayer()
	for i := 0; i < 128; i++ {
		layer = fill(layer, i)
	}
	b.ResetTimer()
	var (
		have []byte
		err  error
	)
	for i := 0; i < b.N; i++ {
		have, err = layer.NodeBlob(target, targetHash)
		if err != nil {
			b.Fatal(err)
		}
	}
	if !bytes.Equal(have, want) {
		b.Fatalf("have %x want %x", have, want)
	}
}

// BenchmarkSearchBottom benchmarks the search hits in the bottom diff layer.
//
// BenchmarkSearchBottom
// BenchmarkSearchBottom-4   	  222717	      6167 ns/op
func BenchmarkSearchBottom(b *testing.B) { benchmarkSearch(b, 0) }

// BenchmarkSearchBottom benchmarks the search hits in the top diff layer.
//
// BenchmarkSearchTop
// BenchmarkSearchTop-4   	10910677	       111.8 ns/op
func BenchmarkSearchTop(b *testing.B) { benchmarkSearch(b, 127) }

func benchmarkGetNode(b *testing.B, getBlob bool) {
	db := NewDatabase(rawdb.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	trie, _ := New(common.Hash{}, common.Hash{}, common.Hash{}, db)

	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		binary.LittleEndian.PutUint64(k, uint64(i))
		trie.Update(k, randBytes(100))
	}
	root, nodes, _ := trie.Commit(nil)
	db.Commit(root, common.Hash{}, nodes)

	var (
		target     []byte
		targetHash common.Hash
	)
	for k, n := range nodes.nodes {
		target, targetHash = []byte(k), n.hash
		break
	}
	layer := db.GetReader(root).(snapshot)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if getBlob {
			layer.NodeBlob(target, targetHash)
		} else {
			layer.Node(target, targetHash)
		}
	}
}

// BenchmarkGetNode
// BenchmarkGetNode-4   	 3279104	       349.2 ns/op
func BenchmarkGetNode(b *testing.B) { benchmarkGetNode(b, false) }

// BenchmarkGetNodeBlob
// BenchmarkGetNodeBlob-4   	 2166842	       479.8 ns/op
func BenchmarkGetNodeBlob(b *testing.B) { benchmarkGetNode(b, true) }

// BenchmarkPersist-8   	       2	 538105875 ns/op
func BenchmarkPersist(b *testing.B) {
	// First, we set up 128 diff layers, with 3K items each
	fill := func(parent snapshot) *diffLayer {
		var nodes = make(map[string]*nodeWithPreValue)
		for i := 0; i < 3000; i++ {
			var (
				path    = randomHash().Bytes()
				storage = EncodeStorageKey(common.Hash{}, path)
				val     = randomNode()
			)
			nodes[string(storage)] = &nodeWithPreValue{
				cachedNode: val,
				pre:        nil,
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
		dl.persist(nil, 0, false)
		b.StopTimer()
	}
}

// BenchmarkJournal benchmarks the performance for journaling the layers.
//
// BenchmarkJournal
// BenchmarkJournal-8   	      10	 110969279 ns/op
func BenchmarkJournal(b *testing.B) {
	// First, we set up 128 diff layers, with 3K items each
	fill := func(parent snapshot) *diffLayer {
		var nodes = make(map[string]*nodeWithPreValue)
		for i := 0; i < 3000; i++ {
			var (
				path    = randomHash().Bytes()
				storage = EncodeStorageKey(common.Hash{}, path)
				val     = randomNode()
			)
			nodes[string(storage)] = &nodeWithPreValue{
				cachedNode: val,
				pre:        nil,
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
