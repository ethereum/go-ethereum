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
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// conversionAccount is used for converting between full and slim format. When
// doing this, we can consider 'balance' as a byte array, as it has already
// been converted from big.Int into an rlp-byteslice.
type conversionAccount struct {
	Nonce    uint64
	Balance  []byte
	Root     []byte
	CodeHash []byte
}

// SlimToFull converts data on the 'slim RLP' format into the full RLP-format.
// Besides, this function accepts another parameter "subRoot". If the root is
// not empty, apply it to account. Usually the subRoot is specified if we want
// to verify the whole state or re-generate state root with different trie algo.
func SlimToFull(data []byte, subRoot common.Hash) ([]byte, error) {
	acc := &conversionAccount{}
	if err := rlp.DecodeBytes(data, acc); err != nil {
		return nil, err
	}
	if len(acc.Root) == 0 {
		acc.Root = emptyRoot[:]
	}
	if subRoot != (common.Hash{}) {
		acc.Root = subRoot.Bytes()
	}
	if len(acc.CodeHash) == 0 {
		acc.CodeHash = emptyCode[:]
	}
	fullData, err := rlp.EncodeToBytes(acc)
	if err != nil {
		return nil, err
	}
	return fullData, nil
}

// trieKV represents a trie key-value pair
type trieKV struct {
	key   common.Hash
	value []byte
}

type (
	// trieGeneratorFn is the interface of trie generation which can
	// be implemented by different trie algorithm.
	trieGeneratorFn func(in chan (trieKV), out chan (common.Hash))

	// leafCallbackFn is the callback invoked at the leaves of the trie,
	// returns the subtrie root with the specified subtrie identifier.
	leafCallbackFn func(hash common.Hash) common.Hash
)

// GenerateAccountTrieRoot takes an account iterator and reproduces the root hash.
func GenerateAccountTrieRoot(it AccountIterator) common.Hash {
	return generateTrieRoot(it, true, stdGenerate, nil, true)
}

// GenerateStorageTrieRoot takes a storage iterator and reproduces the root hash.
func GenerateStorageTrieRoot(it StorageIterator) common.Hash {
	return generateTrieRoot(it, false, stdGenerate, nil, true)
}

// VerifyState takes the whole snapshot tree as the input, traverses all the accounts
// as well as the corresponding storages and compares the re-computed hash with the
// original one(state root and the storage root).
func VerifyState(snaptree *Tree, root common.Hash) error {
	acctIt, err := snaptree.AccountIterator(root, common.Hash{})
	if err != nil {
		return err
	}
	got := generateTrieRoot(acctIt, true, stdGenerate, func(account common.Hash) common.Hash {
		storageIt, err := snaptree.StorageIterator(root, account, common.Hash{})
		if err != nil {
			return common.Hash{}
		}
		return generateTrieRoot(storageIt, false, stdGenerate, nil, false)
	}, true)

	if got != root {
		return fmt.Errorf("State root hash mismatch, got %x, want %x", got, root)
	}
	return nil
}

func generateTrieRoot(it Iterator, accountIterator bool, generatorFn trieGeneratorFn, leafCallback leafCallbackFn, report bool) common.Hash {
	var (
		in  = make(chan trieKV)      // chan to pass leaves
		out = make(chan common.Hash) // chan to collect result
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		generatorFn(in, out)
		wg.Done()
	}()

	var (
		start   = time.Now()
		logged  = time.Now()
		entries = 0
	)
	// Start to feed leaves
	for it.Next() {
		// Apply the leaf callback first. Normally the callback is used
		// to traverse the storage trie and re-generate the subtrie root.
		// If the callback is specified, then replace the original storage
		// root hash with new one.
		var subRoot common.Hash
		if leafCallback != nil {
			subRoot = leafCallback(it.Hash())
		}
		var l trieKV
		if accountIterator {
			fullData, _ := SlimToFull(it.(AccountIterator).Account(), subRoot)
			l = trieKV{it.Hash(), fullData}
		} else {
			l = trieKV{it.Hash(), it.(StorageIterator).Slot()}
		}
		in <- l
		if time.Since(logged) > 8*time.Second && report {
			log.Info("Generating trie hash from snapshot", "at", l.key, "entries", entries, "elapsed", time.Since(start))
			logged = time.Now()
		}
		entries++
	}
	close(in)
	result := <-out
	wg.Wait()

	if report {
		log.Info("Generated trie hash from snapshot", "entries", entries, "elapsed", time.Since(start))
	}
	return result
}

// stdGenerate is a very basic hexary trie builder which uses the same Trie
// as the rest of geth, with no enhancements or optimizations
func stdGenerate(in chan (trieKV), out chan (common.Hash)) {
	t, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	for leaf := range in {
		t.TryUpdate(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}
