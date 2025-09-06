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

package state

import (
	"container/heap"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
	"golang.org/x/sync/errgroup"
)

const (
	statEvictThreshold = 128 // the depth of statistic to be preserved
)

// Database key scheme for states.
var (
	accountKeySize            = int64(len(rawdb.SnapshotAccountPrefix) + common.HashLength)
	storageKeySize            = int64(len(rawdb.SnapshotStoragePrefix) + common.HashLength*2)
	accountTrienodePrefixSize = int64(len(rawdb.TrieNodeAccountPrefix))
	storageTrienodePrefixSize = int64(len(rawdb.TrieNodeStoragePrefix) + common.HashLength)
	codeKeySize               = int64(len(rawdb.CodePrefix) + common.HashLength)
)

// SizeStats represents either the current state size statistics or the size
// differences resulting from a state transition.
type SizeStats struct {
	StateRoot   common.Hash // State root hash at the time of measurement
	BlockNumber uint64      // Associated block number at the time of measurement

	Accounts             int64 // Total number of accounts in the state
	AccountBytes         int64 // Total storage size used by all account data (in bytes)
	Storages             int64 // Total number of storage slots across all accounts
	StorageBytes         int64 // Total storage size used by all storage slot data (in bytes)
	AccountTrienodes     int64 // Total number of account trie nodes in the state
	AccountTrienodeBytes int64 // Total storage size occupied by account trie nodes (in bytes)
	StorageTrienodes     int64 // Total number of storage trie nodes in the state
	StorageTrienodeBytes int64 // Total storage size occupied by storage trie nodes (in bytes)
	ContractCodes        int64 // Total number of contract codes in the state
	ContractCodeBytes    int64 // Total size of all contract code (in bytes)
}

func (s SizeStats) String() string {
	return fmt.Sprintf("Accounts: %d(%s), Storages: %d(%s), AccountTrienodes: %d(%s), StorageTrienodes: %d(%s), Codes: %d(%s)",
		s.Accounts, common.StorageSize(s.AccountBytes),
		s.Storages, common.StorageSize(s.StorageBytes),
		s.AccountTrienodes, common.StorageSize(s.AccountTrienodeBytes),
		s.StorageTrienodes, common.StorageSize(s.StorageTrienodeBytes),
		s.ContractCodes, common.StorageSize(s.ContractCodeBytes),
	)
}

// add applies the given state diffs and produces a new version of the statistics.
func (s SizeStats) add(diff SizeStats) SizeStats {
	s.StateRoot = diff.StateRoot
	s.BlockNumber = diff.BlockNumber

	s.Accounts += diff.Accounts
	s.AccountBytes += diff.AccountBytes
	s.Storages += diff.Storages
	s.StorageBytes += diff.StorageBytes
	s.AccountTrienodes += diff.AccountTrienodes
	s.AccountTrienodeBytes += diff.AccountTrienodeBytes
	s.StorageTrienodes += diff.StorageTrienodes
	s.StorageTrienodeBytes += diff.StorageTrienodeBytes
	s.ContractCodes += diff.ContractCodes
	s.ContractCodeBytes += diff.ContractCodeBytes
	return s
}

// calSizeStats measures the state size changes of the provided state update.
func calSizeStats(update *stateUpdate) (SizeStats, error) {
	stats := SizeStats{
		BlockNumber: update.blockNumber,
		StateRoot:   update.root,
	}

	// Measure the account changes
	for addr, oldValue := range update.accountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newValue, exists := update.accounts[addrHash]
		if !exists {
			return SizeStats{}, fmt.Errorf("account %x not found", addr)
		}
		oldLen, newLen := len(oldValue), len(newValue)

		switch {
		case oldLen > 0 && newLen == 0:
			// Account deletion
			stats.Accounts -= 1
			stats.AccountBytes -= accountKeySize + int64(oldLen)
		case oldLen == 0 && newLen > 0:
			// Account creation
			stats.Accounts += 1
			stats.AccountBytes += accountKeySize + int64(newLen)
		default:
			// Account update
			stats.AccountBytes += int64(newLen - oldLen)
		}
	}

	// Measure storage changes
	for addr, slots := range update.storagesOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		subset, exists := update.storages[addrHash]
		if !exists {
			return SizeStats{}, fmt.Errorf("storage %x not found", addr)
		}
		for key, oldValue := range slots {
			var (
				exists   bool
				newValue []byte
			)
			if update.rawStorageKey {
				newValue, exists = subset[crypto.Keccak256Hash(key.Bytes())]
			} else {
				newValue, exists = subset[key]
			}
			if !exists {
				return SizeStats{}, fmt.Errorf("storage slot %x-%x not found", addr, key)
			}
			oldLen, newLen := len(oldValue), len(newValue)

			switch {
			case oldLen > 0 && newLen == 0:
				// Storage deletion
				stats.Storages -= 1
				stats.StorageBytes -= storageKeySize + int64(oldLen)
			case oldLen == 0 && newLen > 0:
				// Storage creation
				stats.Storages += 1
				stats.StorageBytes += storageKeySize + int64(newLen)
			default:
				// Storage update
				stats.StorageBytes += int64(newLen - oldLen)
			}
		}
	}

	// Measure trienode changes
	for owner, subset := range update.nodes.Sets {
		var (
			keyPrefix int64
			isAccount = owner == (common.Hash{})
		)
		if isAccount {
			keyPrefix = accountTrienodePrefixSize
		} else {
			keyPrefix = storageTrienodePrefixSize
		}

		// Iterate over Origins since every modified node has an origin entry
		for path, oldNode := range subset.Origins {
			newNode, exists := subset.Nodes[path]
			if !exists {
				return SizeStats{}, fmt.Errorf("node %x-%v not found", owner, path)
			}
			keySize := keyPrefix + int64(len(path))

			switch {
			case len(oldNode) > 0 && len(newNode.Blob) == 0:
				// Node deletion
				if isAccount {
					stats.AccountTrienodes -= 1
					stats.AccountTrienodeBytes -= keySize + int64(len(oldNode))
				} else {
					stats.StorageTrienodes -= 1
					stats.StorageTrienodeBytes -= keySize + int64(len(oldNode))
				}
			case len(oldNode) == 0 && len(newNode.Blob) > 0:
				// Node creation
				if isAccount {
					stats.AccountTrienodes += 1
					stats.AccountTrienodeBytes += keySize + int64(len(newNode.Blob))
				} else {
					stats.StorageTrienodes += 1
					stats.StorageTrienodeBytes += keySize + int64(len(newNode.Blob))
				}
			default:
				// Node update
				if isAccount {
					stats.AccountTrienodeBytes += int64(len(newNode.Blob) - len(oldNode))
				} else {
					stats.StorageTrienodeBytes += int64(len(newNode.Blob) - len(oldNode))
				}
			}
		}
	}

	// Measure code changes. Note that the reported contract code size may be slightly
	// inaccurate due to database deduplication (code is stored by its hash). However,
	// this deviation is negligible and acceptable for measurement purposes.
	for _, code := range update.codes {
		stats.ContractCodes += 1
		stats.ContractCodeBytes += codeKeySize + int64(len(code.blob))
	}
	return stats, nil
}

type stateSizeQuery struct {
	root   *common.Hash    // nil means latest
	err    error           // non-nil if the state size is not yet initialized
	result chan *SizeStats // nil means the state is unknown
}

// SizeTracker handles the state size initialization and tracks of state size metrics.
type SizeTracker struct {
	db       ethdb.KeyValueStore
	triedb   *triedb.Database
	abort    chan struct{}
	aborted  chan struct{}
	updateCh chan *stateUpdate
	queryCh  chan *stateSizeQuery
}

// NewSizeTracker creates a new state size tracker and starts it automatically
func NewSizeTracker(db ethdb.KeyValueStore, triedb *triedb.Database) (*SizeTracker, error) {
	if triedb.Scheme() != rawdb.PathScheme {
		return nil, errors.New("state size tracker is not compatible with hash mode")
	}
	t := &SizeTracker{
		db:       db,
		triedb:   triedb,
		abort:    make(chan struct{}),
		aborted:  make(chan struct{}),
		updateCh: make(chan *stateUpdate),
		queryCh:  make(chan *stateSizeQuery),
	}
	go t.run()
	return t, nil
}

func (t *SizeTracker) Stop() {
	close(t.abort)
	<-t.aborted
}

// sizeStatsHeap is a heap.Interface implementation over statesize statistics for
// retrieving the oldest statistics for eviction.
type sizeStatsHeap []SizeStats

func (h sizeStatsHeap) Len() int           { return len(h) }
func (h sizeStatsHeap) Less(i, j int) bool { return h[i].BlockNumber < h[j].BlockNumber }
func (h sizeStatsHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *sizeStatsHeap) Push(x any) {
	*h = append(*h, x.(SizeStats))
}

func (h *sizeStatsHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// run performs the state size initialization and handles updates
func (t *SizeTracker) run() {
	defer close(t.aborted)

	var last common.Hash
	stats, err := t.init() // launch background thread for state size init
	if err != nil {
		return
	}
	h := sizeStatsHeap(slices.Collect(maps.Values(stats)))
	heap.Init(&h)

	for {
		select {
		case u := <-t.updateCh:
			base, found := stats[u.originRoot]
			if !found {
				log.Debug("Ignored the state size without parent", "parent", u.originRoot, "root", u.root, "number", u.blockNumber)
				continue
			}
			diff, err := calSizeStats(u)
			if err != nil {
				continue
			}
			stat := base.add(diff)
			stats[u.root] = stat
			last = u.root

			heap.Push(&h, stats[u.root])
			for u.blockNumber-h[0].BlockNumber > statEvictThreshold {
				delete(stats, h[0].StateRoot)
				heap.Pop(&h)
			}
			log.Debug("Update state size", "number", stat.BlockNumber, "root", stat.StateRoot, "stat", stat)

		case r := <-t.queryCh:
			var root common.Hash
			if r.root != nil {
				root = *r.root
			} else {
				root = last
			}
			if s, ok := stats[root]; ok {
				r.result <- &s
			} else {
				r.result <- nil
			}

		case <-t.abort:
			return
		}
	}
}

type buildResult struct {
	stat        SizeStats
	root        common.Hash
	blockNumber uint64
	elapsed     time.Duration
	err         error
}

func (t *SizeTracker) init() (map[common.Hash]SizeStats, error) {
	// Wait for snapshot completion and then init
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

wait:
	for {
		select {
		case <-ticker.C:
			if t.triedb.SnapshotCompleted() {
				break wait
			}
		case <-t.updateCh:
			continue
		case r := <-t.queryCh:
			r.err = errors.New("state size is not initialized yet")
			r.result <- nil
		case <-t.abort:
			return nil, errors.New("size tracker closed")
		}
	}

	var (
		updates  = make(map[common.Hash]*stateUpdate)
		children = make(map[common.Hash][]common.Hash)
		done     chan buildResult
	)

	for {
		select {
		case u := <-t.updateCh:
			updates[u.root] = u
			children[u.originRoot] = append(children[u.originRoot], u.root)
			log.Debug("Received state update", "root", u.root, "blockNumber", u.blockNumber)

		case r := <-t.queryCh:
			r.err = errors.New("state size is not initialized yet")
			r.result <- nil

		case <-ticker.C:
			// Only check timer if build hasn't started yet
			if done != nil {
				continue
			}
			root := rawdb.ReadSnapshotRoot(t.db)
			if root == (common.Hash{}) {
				continue
			}
			entry, exists := updates[root]
			if !exists {
				continue
			}
			done = make(chan buildResult)
			go t.build(entry.root, entry.blockNumber, done)
			log.Info("Measuring persistent state size", "root", root.Hex(), "number", entry.blockNumber)

		case result := <-done:
			if result.err != nil {
				return nil, result.err
			}
			var (
				stats = make(map[common.Hash]SizeStats)
				apply func(root common.Hash, stat SizeStats) error
			)
			apply = func(root common.Hash, base SizeStats) error {
				for _, child := range children[root] {
					entry, ok := updates[child]
					if !ok {
						return fmt.Errorf("the state update is not found, %x", child)
					}
					diff, err := calSizeStats(entry)
					if err != nil {
						return err
					}
					stats[child] = base.add(diff)
					if err := apply(child, stats[child]); err != nil {
						return err
					}
				}
				return nil
			}
			if err := apply(result.root, result.stat); err != nil {
				return nil, err
			}

			// Set initial latest stats
			stats[result.root] = result.stat
			log.Info("Measured persistent state size", "root", result.root, "number", result.blockNumber, "stat", result.stat, "elapsed", common.PrettyDuration(result.elapsed))
			return stats, nil

		case <-t.abort:
			return nil, errors.New("size tracker closed")
		}
	}
}

func (t *SizeTracker) build(root common.Hash, blockNumber uint64, done chan buildResult) {
	// Metrics will be directly updated by each goroutine
	var (
		accounts, accountBytes int64
		storages, storageBytes int64
		codes, codeBytes       int64

		accountTrienodes, accountTrienodeBytes int64
		storageTrienodes, storageTrienodeBytes int64

		group errgroup.Group
		start = time.Now()
	)

	// Start all table iterations concurrently with direct metric updates
	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.SnapshotAccountPrefix, "account")
		if err != nil {
			return err
		}
		accounts, accountBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.SnapshotStoragePrefix, "storage")
		if err != nil {
			return err
		}
		storages, storageBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.TrieNodeAccountPrefix, "accountnode")
		if err != nil {
			return err
		}
		accountTrienodes, accountTrienodeBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.TrieNodeStoragePrefix, "storagenode")
		if err != nil {
			return err
		}
		storageTrienodes, storageTrienodeBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTable(t.abort, rawdb.CodePrefix, "contractcode")
		if err != nil {
			return err
		}
		codes, codeBytes = count, bytes
		return nil
	})

	// Wait for all goroutines to complete
	if err := group.Wait(); err != nil {
		done <- buildResult{err: err}
	} else {
		stat := SizeStats{
			StateRoot:            root,
			BlockNumber:          blockNumber,
			Accounts:             accounts,
			AccountBytes:         accountBytes,
			Storages:             storages,
			StorageBytes:         storageBytes,
			AccountTrienodes:     accountTrienodes,
			AccountTrienodeBytes: accountTrienodeBytes,
			StorageTrienodes:     storageTrienodes,
			StorageTrienodeBytes: storageTrienodeBytes,
			ContractCodes:        codes,
			ContractCodeBytes:    codeBytes,
		}
		done <- buildResult{
			root:        root,
			blockNumber: blockNumber,
			stat:        stat,
			elapsed:     time.Since(start),
		}
	}
}

// iterateTable performs iteration over a specific table and returns the results.
func (t *SizeTracker) iterateTable(closed chan struct{}, prefix []byte, name string) (int64, int64, error) {
	var (
		start        = time.Now()
		logged       = time.Now()
		count, bytes int64
	)

	iter := t.db.NewIterator(prefix, nil)
	defer iter.Release()

	log.Debug("Iterating state", "category", name)
	for iter.Next() {
		count++
		bytes += int64(len(iter.Key()) + len(iter.Value()))

		if time.Since(logged) > time.Second*8 {
			logged = time.Now()

			select {
			case <-closed:
				log.Debug("State iteration cancelled", "category", name)
				return 0, 0, errors.New("size tracker closed")
			default:
				log.Debug("Iterating state", "category", name, "count", count, "size", common.StorageSize(bytes))
			}
		}
	}
	// Check for iterator errors
	if err := iter.Error(); err != nil {
		log.Error("Iterator error", "category", name, "err", err)
		return 0, 0, err
	}
	log.Debug("Finished state iteration", "category", name, "count", count, "size", common.StorageSize(bytes), "elapsed", common.PrettyDuration(time.Since(start)))
	return count, bytes, nil
}

// iterateTableParallel performs parallel iteration over a table by splitting into
// hex ranges. For storage tables, it splits on the first byte of the account hash
// (after the prefix).
func (t *SizeTracker) iterateTableParallel(closed chan struct{}, prefix []byte, name string) (int64, int64, error) {
	var (
		totalCount int64
		totalBytes int64

		start   = time.Now()
		workers = runtime.NumCPU()
		group   errgroup.Group
		mu      sync.Mutex
	)
	group.SetLimit(workers)
	log.Debug("Starting parallel state iteration", "category", name, "workers", workers)

	if len(prefix) > 0 {
		if blob, err := t.db.Get(prefix); err == nil && len(blob) > 0 {
			// If there's a direct hit on the prefix, include it in the stats
			totalCount = 1
			totalBytes = int64(len(prefix) + len(blob))
		}
	}
	for i := 0; i < 256; i++ {
		h := byte(i)
		group.Go(func() error {
			count, bytes, err := t.iterateTable(closed, slices.Concat(prefix, []byte{h}), fmt.Sprintf("%s-%02x", name, h))
			if err != nil {
				return err
			}
			mu.Lock()
			totalCount += count
			totalBytes += bytes
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return 0, 0, err
	}
	log.Debug("Finished parallel state iteration", "category", name, "count", totalCount, "size", common.StorageSize(totalBytes), "elapsed", common.PrettyDuration(time.Since(start)))
	return totalCount, totalBytes, nil
}

// Notify is an async method used to send the state update to the size tracker.
// It ignores empty updates (where no state changes occurred).
// If the channel is full, it drops the update to avoid blocking.
func (t *SizeTracker) Notify(update *stateUpdate) {
	if update == nil || update.empty() {
		return
	}
	select {
	case t.updateCh <- update:
	case <-t.abort:
		return
	}
}

// Query returns the state size specified by the root, or nil if not available.
// If the root is nil, query the size of latest chain head;
// If the root is non-nil, query the size of the specified state;
func (t *SizeTracker) Query(root *common.Hash) (*SizeStats, error) {
	r := &stateSizeQuery{
		root:   root,
		result: make(chan *SizeStats, 1),
	}
	select {
	case <-t.aborted:
		return nil, errors.New("state sizer has been closed")
	case t.queryCh <- r:
		return <-r.result, r.err
	}
}
