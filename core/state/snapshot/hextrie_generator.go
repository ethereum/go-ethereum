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
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

type leaf struct {
	key   common.Hash
	value []byte
}

type trieGeneratorFn func(in chan (leaf), out chan (common.Hash))

func GenerateBinaryTree(it AccountIterator) common.Hash {
	db, err := rawdb.NewLevelDBDatabase("./bintrie", 128, 1024, "")
	if err != nil {
		panic(fmt.Sprintf("error opening bintrie db, err=%v", err))
	}
	btrie := new(trie.BinaryTrie)
	btrie.CommitCh = make(chan trie.BinaryHashPreimage)

	var nodeCount uint64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for kv := range btrie.CommitCh {
			nodeCount++
			db.Put(kv.Key, kv.Value)
		}
	}()
	counter := 0
	for it.Next() {
		counter++
		// Don't get the entire expanded account at this
		// stage - NOTE
		btrie.TryUpdate(it.Hash().Bytes(), it.Account())
	}
	log.Info("Inserted all leaves", "count", counter)

	err = btrie.Commit()
	if err != nil {
		panic(fmt.Sprintf("error committing trie, err=%v", err))
	}
	close(btrie.CommitCh)
	wg.Wait()
	btrie.CommitCh = nil
	log.Info("Done writing nodes to the DB", "count", nodeCount)
	log.Info("Calculated binary hash", "hash", common.ToHex(btrie.Hash()))

	return common.BytesToHash(btrie.Hash())
}

// GenerateTrieRoot takes an account iterator and reproduces the root hash.
func GenerateTrieRoot(it AccountIterator) common.Hash {
	//return generateTrieRoot(it, StackGenerate)
	return generateTrieRoot(it, ReStackGenerate)
}

func CrosscheckTriehasher(it AccountIterator, begin, end int) bool {
	return verifyHasher(it, ReStackGenerate, begin, end)
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
	start := time.Now()
	logged := time.Now()
	accounts := 0
	for it.Next() {
		slimData := it.Account()
		fullData := SlimToFull(slimData)
		l := leaf{it.Hash(), fullData}
		in <- l
		if time.Since(logged) > 8*time.Second {
			log.Info("Generating trie hash from snapshot",
				"at", l.key, "accounts", accounts, "elapsed", time.Since(start))
			logged = time.Now()
		}
		accounts++
	}
	close(in)
	result := <-out
	log.Info("Generated trie hash from snapshot", "accounts", accounts, "elapsed", time.Since(start))
	wg.Wait()
	return result
}

// ReStackGenerate is a hexary trie builder which is built from the
// bottom-up as keys are added. It attempts to save memory by doing
// the RLP encoding on the fly during hashing.
func ReStackGenerate(in chan (leaf), out chan (common.Hash)) {
	t := trie.NewReStackTrie()
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}

func verifyHasher(it AccountIterator, generatorFn trieGeneratorFn, begin, end int) bool {
	var (
		referenceFn = StdGenerate

		inA  = make(chan leaf)        // chan to pass leaves
		outA = make(chan common.Hash) // chan to collect result

		inB  = make(chan leaf)        // chan to pass leaves
		outB = make(chan common.Hash) // chan to collect result
		wg   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		referenceFn(inA, outA)
		wg.Done()
	}()
	go func() {
		generatorFn(inB, outB)
		wg.Done()
	}()
	// Feed leaves
	start := time.Now()
	logged := time.Now()
	accounts := 0
	for it.Next() {
		if accounts < begin {
			accounts++
			continue
		}
		if end > 0 && accounts > end {
			break
		}
		slimData := it.Account()
		fullData := SlimToFull(slimData)
		l := leaf{it.Hash(), fullData}
		inA <- l
		inB <- l
		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Generating trie hash from snapshot",
				"at", l.key, "accounts", accounts, "elapsed", time.Since(start))
			logged = time.Now()
		}
	}
	close(inA)
	close(inB)
	resultA := <-outA
	resultB := <-outB
	log.Info("Generated trie hash from snapshot", "accounts", accounts,
		"elapsed", time.Since(start),
		"start", begin,
		"end", end,
		"exp", resultA,
		"got", resultB)
	wg.Wait()
	return resultA == resultB
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
