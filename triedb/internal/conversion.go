// Copyright 2026 The go-ethereum Authors
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

// Package internal contains shared trie generation utilities used by both
// triedb and triedb/pathdb. All code is ported from
// core/state/snapshot/conversion.go (with exported names) unless noted.
package internal

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Iterator is an iterator to step over all the accounts or the specific
// storage in a snapshot which may or may not be composed of multiple layers.
type Iterator interface {
	// Next steps the iterator forward one element, returning false if exhausted,
	// or an error if iteration failed for some reason (e.g. root being iterated
	// becomes stale and garbage collected).
	Next() bool

	// Error returns any failure that occurred during iteration, which might have
	// caused a premature iteration exit (e.g. snapshot stack becoming stale).
	Error() error

	// Hash returns the hash of the account or storage slot the iterator is
	// currently at.
	Hash() common.Hash

	// Release releases associated resources. Release should always succeed and
	// can be called multiple times without causing error.
	Release()
}

// AccountIterator is an iterator to step over all the accounts in a snapshot,
// which may or may not be composed of multiple layers.
type AccountIterator interface {
	Iterator

	// Account returns the RLP encoded slim account the iterator is currently at.
	// An error will be returned if the iterator becomes invalid
	Account() []byte
}

// StorageIterator is an iterator to step over the specific storage in a snapshot,
// which may or may not be composed of multiple layers.
type StorageIterator interface {
	Iterator

	// Slot returns the storage slot the iterator is currently at. An error will
	// be returned if the iterator becomes invalid
	Slot() []byte
}

// TrieKV represents a trie key-value pair.
type TrieKV struct {
	Key   common.Hash
	Value []byte
}

type (
	// TrieGeneratorFn is the interface of trie generation which can
	// be implemented by different trie algorithm.
	TrieGeneratorFn func(db ethdb.KeyValueWriter, scheme string, owner common.Hash, in chan (TrieKV), out chan (common.Hash))

	// LeafCallbackFn is the callback invoked at the leaves of the trie,
	// returns the subtrie root with the specified subtrie identifier.
	LeafCallbackFn func(db ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *GenerateStats) (common.Hash, error)
)

// GenerateStats is a collection of statistics gathered by the trie generator
// for logging purposes.
type GenerateStats struct {
	head  common.Hash
	start time.Time

	accounts uint64 // Number of accounts done (including those being crawled)
	slots    uint64 // Number of storage slots done (including those being crawled)

	slotsStart map[common.Hash]time.Time   // Start time for account slot crawling
	slotsHead  map[common.Hash]common.Hash // Slot head for accounts being crawled

	lock sync.RWMutex
}

// NewGenerateStats creates a new generator stats.
func NewGenerateStats() *GenerateStats {
	return &GenerateStats{
		slotsStart: make(map[common.Hash]time.Time),
		slotsHead:  make(map[common.Hash]common.Hash),
		start:      time.Now(),
	}
}

// ProgressAccounts updates the generator stats for the account range.
func (stat *GenerateStats) ProgressAccounts(account common.Hash, done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.accounts += done
	stat.head = account
}

// FinishAccounts updates the generator stats for the finished account range.
func (stat *GenerateStats) FinishAccounts(done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.accounts += done
}

// ProgressContract updates the generator stats for a specific in-progress contract.
func (stat *GenerateStats) ProgressContract(account common.Hash, slot common.Hash, done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.slots += done
	stat.slotsHead[account] = slot
	if _, ok := stat.slotsStart[account]; !ok {
		stat.slotsStart[account] = time.Now()
	}
}

// FinishContract updates the generator stats for a specific just-finished contract.
func (stat *GenerateStats) FinishContract(account common.Hash, done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.slots += done
	delete(stat.slotsHead, account)
	delete(stat.slotsStart, account)
}

// Report prints the cumulative progress statistic smartly.
func (stat *GenerateStats) Report() {
	stat.lock.RLock()
	defer stat.lock.RUnlock()

	ctx := []interface{}{
		"accounts", stat.accounts,
		"slots", stat.slots,
		"elapsed", common.PrettyDuration(time.Since(stat.start)),
	}
	if stat.accounts > 0 {
		if done := binary.BigEndian.Uint64(stat.head[:8]) / stat.accounts; done > 0 {
			var (
				left = (math.MaxUint64 - binary.BigEndian.Uint64(stat.head[:8])) / stat.accounts
				eta  = common.CalculateETA(done, left, time.Since(stat.start))
			)
			// If there are large contract crawls in progress, estimate their finish time
			for acc, head := range stat.slotsHead {
				start := stat.slotsStart[acc]
				if done := binary.BigEndian.Uint64(head[:8]); done > 0 {
					left := math.MaxUint64 - binary.BigEndian.Uint64(head[:8])

					// Override the ETA if larger than the largest until now
					if slotETA := common.CalculateETA(done, left, time.Since(start)); eta < slotETA {
						eta = slotETA
					}
				}
			}
			ctx = append(ctx, []interface{}{
				"eta", common.PrettyDuration(eta),
			}...)
		}
	}
	log.Info("Iterating state snapshot", ctx...)
}

// ReportDone prints the last log when the whole generation is finished.
func (stat *GenerateStats) ReportDone() {
	stat.lock.RLock()
	defer stat.lock.RUnlock()

	var ctx []interface{}
	ctx = append(ctx, []interface{}{"accounts", stat.accounts}...)
	if stat.slots != 0 {
		ctx = append(ctx, []interface{}{"slots", stat.slots}...)
	}
	ctx = append(ctx, []interface{}{"elapsed", common.PrettyDuration(time.Since(stat.start))}...)
	log.Info("Iterated snapshot", ctx...)
}

// RunReport periodically prints the progress information.
func RunReport(stats *GenerateStats, stop chan bool) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			stats.Report()
			timer.Reset(time.Second * 8)
		case success := <-stop:
			if success {
				stats.ReportDone()
			}
			return
		}
	}
}

// GenerateTrieRoot generates the trie hash based on the snapshot iterator.
// It can be used for generating account trie, storage trie or even the
// whole state which connects the accounts and the corresponding storages.
func GenerateTrieRoot(db ethdb.KeyValueWriter, scheme string, it Iterator, account common.Hash, generatorFn TrieGeneratorFn, leafCallback LeafCallbackFn, stats *GenerateStats, report bool) (common.Hash, error) {
	var (
		in      = make(chan TrieKV)         // chan to pass leaves
		out     = make(chan common.Hash, 1) // chan to collect result
		stoplog = make(chan bool, 1)        // 1-size buffer, works when logging is not enabled
		wg      sync.WaitGroup
	)
	// Spin up a go-routine for trie hash re-generation
	wg.Add(1)
	go func() {
		defer wg.Done()
		generatorFn(db, scheme, account, in, out)
	}()
	// Spin up a go-routine for progress logging
	if report && stats != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RunReport(stats, stoplog)
		}()
	}
	// Create a semaphore to assign tasks and collect results through. We'll pre-
	// fill it with nils, thus using the same channel for both limiting concurrent
	// processing and gathering results.
	threads := runtime.NumCPU()
	results := make(chan error, threads)
	for i := 0; i < threads; i++ {
		results <- nil // fill the semaphore
	}
	// stop is a helper function to shutdown the background threads
	// and return the re-generated trie hash.
	stop := func(fail error) (common.Hash, error) {
		close(in)
		result := <-out
		for i := 0; i < threads; i++ {
			if err := <-results; err != nil && fail == nil {
				fail = err
			}
		}
		stoplog <- fail == nil

		wg.Wait()
		return result, fail
	}
	var (
		logged    = time.Now()
		processed = uint64(0)
		leaf      TrieKV
	)
	// Start to feed leaves
	for it.Next() {
		if account == (common.Hash{}) {
			var (
				err      error
				fullData []byte
			)
			if leafCallback == nil {
				fullData, err = types.FullAccountRLP(it.(AccountIterator).Account())
				if err != nil {
					return stop(err)
				}
			} else {
				// Wait until the semaphore allows us to continue, aborting if
				// a sub-task failed
				if err := <-results; err != nil {
					results <- nil // stop will drain the results, add a noop back for this error we just consumed
					return stop(err)
				}
				// Fetch the next account and process it concurrently
				account, err := types.FullAccount(it.(AccountIterator).Account())
				if err != nil {
					return stop(err)
				}
				go func(hash common.Hash) {
					subroot, err := leafCallback(db, hash, common.BytesToHash(account.CodeHash), stats)
					if err != nil {
						results <- err
						return
					}
					if account.Root != subroot {
						results <- fmt.Errorf("invalid subroot(path %x), want %x, have %x", hash, account.Root, subroot)
						return
					}
					results <- nil
				}(it.Hash())
				fullData, err = rlp.EncodeToBytes(account)
				if err != nil {
					return stop(err)
				}
			}
			leaf = TrieKV{it.Hash(), fullData}
		} else {
			leaf = TrieKV{it.Hash(), common.CopyBytes(it.(StorageIterator).Slot())}
		}
		in <- leaf

		// Accumulate the generation statistic if it's required.
		processed++
		if time.Since(logged) > 3*time.Second && stats != nil {
			if account == (common.Hash{}) {
				stats.ProgressAccounts(it.Hash(), processed)
			} else {
				stats.ProgressContract(account, it.Hash(), processed)
			}
			logged, processed = time.Now(), 0
		}
	}
	// Commit the last part statistic.
	if processed > 0 && stats != nil {
		if account == (common.Hash{}) {
			stats.FinishAccounts(processed)
		} else {
			stats.FinishContract(account, processed)
		}
	}
	return stop(nil)
}

// StackTrieGenerate is the trie generation function that creates a StackTrie
// and persists nodes via rawdb.WriteTrieNode.
func StackTrieGenerate(db ethdb.KeyValueWriter, scheme string, owner common.Hash, in chan TrieKV, out chan common.Hash) {
	var onTrieNode trie.OnTrieNode
	if db != nil {
		onTrieNode = func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(db, owner, path, hash, blob, scheme)
		}
	}
	t := trie.NewStackTrie(onTrieNode)
	for leaf := range in {
		t.Update(leaf.Key[:], leaf.Value)
	}
	out <- t.Hash()
}
