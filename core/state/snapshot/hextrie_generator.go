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
	"sync"
)

type leaf struct {
	key   common.Hash
	value []byte
}

type trieGeneratorFn func(in chan (leaf), out chan (common.Hash))

// GenerateTrieRoot takes an account iterator and reproduces the root hash.
func GenerateTrieRoot(it AccountIterator) common.Hash {
	return generateTrieRoot(it, StackGenerate)
}

func generateTrieRoot(it AccountIterator, generatorFn trieGeneratorFn) common.Hash {
	var (
		in  = make(chan leaf)        // chan to pass leaves
		out = make(chan common.Hash) // chan to collect result
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		generatorFn(in, out)
		wg.Done()
	}()
	// Feed leaves
	for it.Next() {
		in <- leaf{it.Hash(), it.Account()}
	}
	close(in)
	result := <-out
	wg.Wait()
	return result
}

// StackGenerate is a hexary trie builder which is built from the bottom-up as
// keys are added.
func StackGenerate(in chan (leaf), out chan (common.Hash)) {
	t := trie.NewStackTrie()
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}

// PruneGenerate is a hexary trie builder which collapses old nodes, but is still
// based on more or less the ordinary trie builder
func PruneGenerate(in chan (leaf), out chan (common.Hash)) {
	t := trie.NewHashTrie()
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}

// StdGenerate is a very basic hexary trie builder which uses the same Trie
// as the rest of geth, with no enhancements or optimizations
func StdGenerate(in chan (leaf), out chan (common.Hash)) {
	t, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}

// AppendOnlyGenerate is a very basic hexary trie builder which uses the same Trie
// as the rest of geth, but does not create as many objects while expanding the trie
func AppendOnlyGenerate(in chan (leaf), out chan (common.Hash)) {
	t := trie.NewAppendOnlyTrie()
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}
