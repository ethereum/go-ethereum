// Copyright 2020 The go-ethereum Authors
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

package snapshot

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
)

type leaf struct {
	key   common.Hash
	value []byte
}

// trieGenerator is a very basic hexary trie builder which uses the same Trie
// as the rest of geth, with no enhancements or optimizations
type trieGenerator struct{}

//BenchmarkTrieGeneration/4K-8         	      73	  15309586 ns/op	 6614793 B/op	   55006 allocs/op
//BenchmarkTrieGeneration/10K-8        	      28	  39538254 ns/op	16539589 B/op	  137515 allocs/op
func (gen *trieGenerator) Generate3(in chan (leaf), out chan (common.Hash)) {
	t := trie.NewHashTrie()
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}

//BenchmarkTrieGeneration/4K-6         	      94	  12598506 ns/op	 6162370 B/op	   57921 allocs/op
//BenchmarkTrieGeneration/10K-6        	      37	  33790908 ns/op	17278751 B/op	  151002 allocs/op
func (gen *trieGenerator) Generate2(in chan (leaf), out chan (common.Hash)) {
	t, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}

//BenchmarkTrieGeneration/4K-6         	     115	  12755614 ns/op	 2303051 B/op	   42678 allocs/op
//BenchmarkTrieGeneration/10K-6        	      46	  25374595 ns/op	 5754446 B/op	  106676 allocs/op
func (gen *trieGenerator) Generate(in chan (leaf), out chan (common.Hash)) {
	t := trie.NewAppendOnlyTrie()
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}
