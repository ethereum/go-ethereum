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
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

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
	leafCallbackFn func(hash common.Hash, stat *generateStats) common.Hash
)

// GenerateAccountTrieRoot takes an account iterator and reproduces the root hash.
func GenerateAccountTrieRoot(it AccountIterator) (common.Hash, error) {
	return generateTrieRoot(it, common.Hash{}, stdGenerate, nil, &generateStats{start: time.Now()}, true)
}

// GenerateStorageTrieRoot takes a storage iterator and reproduces the root hash.
func GenerateStorageTrieRoot(account common.Hash, it StorageIterator) (common.Hash, error) {
	return generateTrieRoot(it, account, stdGenerate, nil, &generateStats{start: time.Now()}, true)
}

// VerifyState takes the whole snapshot tree as the input, traverses all the accounts
// as well as the corresponding storages and compares the re-computed hash with the
// original one(state root and the storage root).
func VerifyState(snaptree *Tree, root common.Hash) error {
	acctIt, err := snaptree.AccountIterator(root, common.Hash{})
	if err != nil {
		return err
	}
	defer acctIt.Release()

	got, err := generateTrieRoot(acctIt, common.Hash{}, stdGenerate, func(account common.Hash, stat *generateStats) common.Hash {
		storageIt, err := snaptree.StorageIterator(root, account, common.Hash{})
		if err != nil {
			return common.Hash{}
		}
		defer storageIt.Release()

		hash, err := generateTrieRoot(storageIt, account, stdGenerate, nil, stat, false)
		if err != nil {
			return common.Hash{}
		}
		return hash
	}, &generateStats{start: time.Now()}, true)

	if err != nil {
		return err
	}
	if got != root {
		return fmt.Errorf("state root hash mismatch: got %x, want %x", got, root)
	}
	return nil
}

// generateStats is a collection of statistics gathered by the trie generator
// for logging purposes.
type generateStats struct {
	accounts   uint64
	slots      uint64
	curAccount common.Hash
	curSlot    common.Hash
	start      time.Time
	lock       sync.RWMutex
}

// progress records the progress trie generator made recently.
func (stat *generateStats) progress(accounts, slots uint64, curAccount common.Hash, curSlot common.Hash) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.accounts += accounts
	stat.slots += slots
	stat.curAccount = curAccount
	stat.curSlot = curSlot
}

// report prints the cumulative progress statistic smartly.
func (stat *generateStats) report() {
	stat.lock.RLock()
	defer stat.lock.RUnlock()

	var ctx []interface{}
	if stat.curSlot != (common.Hash{}) {
		ctx = append(ctx, []interface{}{
			"in", stat.curAccount,
			"at", stat.curSlot,
		}...)
	} else {
		ctx = append(ctx, []interface{}{"at", stat.curAccount}...)
	}
	// Add the usual measurements
	ctx = append(ctx, []interface{}{"accounts", stat.accounts}...)
	if stat.slots != 0 {
		ctx = append(ctx, []interface{}{"slots", stat.slots}...)
	}
	ctx = append(ctx, []interface{}{"elapsed", common.PrettyDuration(time.Since(stat.start))}...)
	log.Info("Generating trie hash from snapshot", ctx...)
}

// reportDone prints the last log when the whole generation is finished.
func (stat *generateStats) reportDone() {
	stat.lock.RLock()
	defer stat.lock.RUnlock()

	var ctx []interface{}
	ctx = append(ctx, []interface{}{"accounts", stat.accounts}...)
	if stat.slots != 0 {
		ctx = append(ctx, []interface{}{"slots", stat.slots}...)
	}
	ctx = append(ctx, []interface{}{"elapsed", common.PrettyDuration(time.Since(stat.start))}...)
	log.Info("Generated trie hash from snapshot", ctx...)
}

// generateTrieRoot generates the trie hash based on the snapshot iterator.
// It can be used for generating account trie, storage trie or even the
// whole state which connects the accounts and the corresponding storages.
func generateTrieRoot(it Iterator, account common.Hash, generatorFn trieGeneratorFn, leafCallback leafCallbackFn, stats *generateStats, report bool) (common.Hash, error) {
	var (
		in      = make(chan trieKV)         // chan to pass leaves
		out     = make(chan common.Hash, 1) // chan to collect result
		stoplog = make(chan bool, 1)        // 1-size buffer, works when logging is not enabled
		wg      sync.WaitGroup
	)
	// Spin up a go-routine for trie hash re-generation
	wg.Add(1)
	go func() {
		defer wg.Done()
		generatorFn(in, out)
	}()

	// Spin up a go-routine for progress logging
	if report && stats != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			timer := time.NewTimer(0)
			defer timer.Stop()

			for {
				select {
				case <-timer.C:
					stats.report()
					timer.Reset(time.Second * 8)
				case success := <-stoplog:
					if success {
						stats.reportDone()
					}
					return
				}
			}
		}()
	}
	// stop is a helper function to shutdown the background threads
	// and return the re-generated trie hash.
	stop := func(success bool) common.Hash {
		close(in)
		result := <-out
		stoplog <- success
		wg.Wait()
		return result
	}
	var (
		logged    = time.Now()
		processed = uint64(0)
		leaf      trieKV
		last      common.Hash
	)
	// Start to feed leaves
	for it.Next() {
		if account == (common.Hash{}) {
			var (
				err      error
				fullData []byte
			)
			if leafCallback == nil {
				fullData, err = FullAccountRLP(it.(AccountIterator).Account())
				if err != nil {
					stop(false)
					return common.Hash{}, err
				}
			} else {
				account, err := FullAccount(it.(AccountIterator).Account())
				if err != nil {
					stop(false)
					return common.Hash{}, err
				}
				// Apply the leaf callback. Normally the callback is used to traverse
				// the storage trie and re-generate the subtrie root.
				subroot := leafCallback(it.Hash(), stats)
				if !bytes.Equal(account.Root, subroot.Bytes()) {
					stop(false)
					return common.Hash{}, fmt.Errorf("invalid subroot(%x), want %x, got %x", it.Hash(), account.Root, subroot)
				}
				fullData, err = rlp.EncodeToBytes(account)
				if err != nil {
					stop(false)
					return common.Hash{}, err
				}
			}
			leaf = trieKV{it.Hash(), fullData}
		} else {
			leaf = trieKV{it.Hash(), common.CopyBytes(it.(StorageIterator).Slot())}
		}
		in <- leaf

		// Accumulate the generaation statistic if it's required.
		processed++
		if time.Since(logged) > 3*time.Second && stats != nil {
			if account == (common.Hash{}) {
				stats.progress(processed, 0, it.Hash(), common.Hash{})
			} else {
				stats.progress(0, processed, account, it.Hash())
			}
			logged, processed = time.Now(), 0
		}
		last = it.Hash()
	}
	// Commit the last part statistic.
	if processed > 0 && stats != nil {
		if account == (common.Hash{}) {
			stats.progress(processed, 0, last, common.Hash{})
		} else {
			stats.progress(0, processed, account, last)
		}
	}
	result := stop(true)
	return result, nil
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
