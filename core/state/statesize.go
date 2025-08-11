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
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
	"golang.org/x/sync/errgroup"
)

var (
	accountSnapKeySize       = int64(len(rawdb.SnapshotAccountPrefix) + common.HashLength)
	storageSnapKeySize       = int64(len(rawdb.SnapshotStoragePrefix) + common.HashLength + common.HashLength)
	accountTrieKeyPrefixSize = int64(len(rawdb.TrieNodeAccountPrefix))
	storageTrieKeyPrefixSize = int64(len(rawdb.TrieNodeStoragePrefix) + common.HashLength)
	codeKeySize              = int64(len(rawdb.CodePrefix) + common.HashLength)
)

// stateSizeMetrics represents the current state size statistics
type stateSizeMetrics struct {
	Root          common.Hash // Root hash of the state trie
	AccountCount  int64
	AccountBytes  int64
	StorageCount  int64
	StorageBytes  int64
	TrieNodeCount int64
	TrieNodeBytes int64
	ContractCount int64
	ContractBytes int64
}

// StateSizeGenerator handles the initialization and tracking of state size metrics
type StateSizeGenerator struct {
	db          ethdb.KeyValueStore
	triedb      *triedb.Database
	abort       chan struct{}
	done        chan struct{}
	updateChan  chan *stateUpdate // Async message channel for updates
	metrics     *stateSizeMetrics
	buffered    *stateSizeMetrics
	initialized atomic.Bool // Initialization state
}

// NewStateSizeGenerator creates a new state size generator and starts it automatically
func NewStateSizeGenerator(db ethdb.KeyValueStore, triedb *triedb.Database, root common.Hash) *StateSizeGenerator {
	g := &StateSizeGenerator{
		db:         db,
		triedb:     triedb,
		abort:      make(chan struct{}),
		done:       make(chan struct{}),
		updateChan: make(chan *stateUpdate, 1000), // Buffered channel for updates
		metrics:    &stateSizeMetrics{Root: root},
		buffered:   &stateSizeMetrics{Root: root},
	}

	// Start the generator automatically
	go g.generate()

	return g
}

// Stop terminates the background generation and persists the metrics.
func (g *StateSizeGenerator) Stop() {
	close(g.abort)

	<-g.done

	g.persistMetrics()
}

// generate performs the state size initialization and handles updates
func (g *StateSizeGenerator) generate() {
	defer close(g.done)

	initDone := g.initialize()

	for {
		select {
		case update := <-g.updateChan:
			g.handleUpdate(update, g.initialized.Load())

		case <-g.abort:
			log.Info("State size generation aborted")

			// Wait for initialization to complete with timeout
			if initDone != nil {
				select {
				case <-initDone:
					log.Debug("Initialization completed before abort")
				case <-time.After(5 * time.Second):
					log.Warn("Initialization did not finish in time during abort")
				}
			}
			return

		case <-initDone:
			// Initialization completed, merge buffered metrics
			if g.buffered != nil {
				log.Info("Merging buffered metrics into main metrics")
				g.metrics.Root = g.buffered.Root
				g.metrics.AccountCount += g.buffered.AccountCount
				g.metrics.AccountBytes += g.buffered.AccountBytes
				g.metrics.StorageCount += g.buffered.StorageCount
				g.metrics.StorageBytes += g.buffered.StorageBytes
				g.metrics.TrieNodeCount += g.buffered.TrieNodeCount
				g.metrics.TrieNodeBytes += g.buffered.TrieNodeBytes
				g.metrics.ContractCount += g.buffered.ContractCount
				g.metrics.ContractBytes += g.buffered.ContractBytes

				g.buffered = nil
			}
			initDone = nil // Clear the channel to prevent future selects
		}
	}
}

// initialize starts the initialization process if not already initialized
func (g *StateSizeGenerator) initialize() chan struct{} {
	// Check if we already have existing metrics
	if g.hasExistingMetrics() {
		log.Info("State size metrics already initialized")
		g.initialized.Store(true)
		return nil
	}

	initDone := make(chan struct{})

	// Wait for snapshot completion and then initialize
	go func() {
		defer close(initDone)

	LOOP:
		// Wait for snapshot generator to complete first
		for {
			root, done := g.triedb.SnapshotCompleted()
			if done {
				g.metrics.Root = root
				g.buffered.Root = root
				break LOOP
			}

			select {
			case <-g.abort:
				log.Info("State size initialization aborted during snapshot wait")
				return
			case <-time.After(10 * time.Second):
				// Continue checking for snapshot completion
			}
		}

		// Start actual initialization
		start := time.Now()
		log.Info("Starting state size initialization")
		if err := g.initializeMetrics(); err != nil {
			log.Error("Failed to initialize state size metrics", "err", err)
			return
		}

		g.initialized.Store(true)

		log.Info("Completed state size initialization", "elapsed", time.Since(start))
	}()

	return initDone
}

// handleUpdate processes a single update with proper root continuity checking
func (g *StateSizeGenerator) handleUpdate(update *stateUpdate, initialized bool) {
	diff := g.calculateUpdateDiff(update)

	var targetMetrics *stateSizeMetrics
	if initialized {
		targetMetrics = g.metrics
	} else {
		targetMetrics = g.buffered
	}

	// Check root continuity - the update should build on our current state
	if targetMetrics.Root != (common.Hash{}) && targetMetrics.Root != update.originRoot {
		log.Warn("State update root discontinuity detected", "current", targetMetrics.Root, "updateOrigin", update.originRoot, "updateNew", update.root)
	}

	// Update to the new state root
	targetMetrics.Root = update.root
	targetMetrics.AccountCount += diff.AccountCount
	targetMetrics.AccountBytes += diff.AccountBytes
	targetMetrics.StorageCount += diff.StorageCount
	targetMetrics.StorageBytes += diff.StorageBytes
	targetMetrics.TrieNodeCount += diff.TrieNodeCount
	targetMetrics.TrieNodeBytes += diff.TrieNodeBytes
	targetMetrics.ContractCount += diff.ContractCount
	targetMetrics.ContractBytes += diff.ContractBytes

	// Fire the metrics and persist only if initialization is done
	if initialized {
		g.updateMetrics()
		g.persistMetrics()
	}
}

// calculateUpdateDiff calculates the diff for a state update
func (g *StateSizeGenerator) calculateUpdateDiff(update *stateUpdate) stateSizeMetrics {
	var diff stateSizeMetrics

	// Calculate account changes
	for addr, oldValue := range update.accountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newValue, exists := update.accounts[addrHash]
		if !exists {
			log.Warn("State update missing account", "address", addr)
			continue
		}

		oldLen, newLen := len(oldValue), len(newValue)
		if oldLen > 0 && newLen == 0 {
			// Account deletion
			diff.AccountCount -= 1
			diff.AccountBytes -= accountSnapKeySize + int64(oldLen)
		} else if oldLen == 0 && newLen > 0 {
			// Account creation
			diff.AccountCount += 1
			diff.AccountBytes += accountSnapKeySize + int64(newLen)
		} else {
			// Account update
			diff.AccountBytes += int64(newLen - oldLen)
		}
	}

	// Calculate storage changes
	for addr, slots := range update.storagesOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		subset, exists := update.storages[addrHash]
		if !exists {
			log.Warn("State update missing storage", "address", addr)
			continue
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
				log.Warn("State update missing storage slot", "address", addr, "key", key)
				continue
			}

			oldLen, newLen := len(oldValue), len(newValue)
			if oldLen > 0 && newLen == 0 {
				// Storage deletion
				diff.StorageCount -= 1
				diff.StorageBytes -= storageSnapKeySize + int64(oldLen)
			} else if oldLen == 0 && newLen > 0 {
				// Storage creation
				diff.StorageCount += 1
				diff.StorageBytes += storageSnapKeySize + int64(newLen)
			} else {
				// Storage update
				diff.StorageBytes += int64(newLen - oldLen)
			}
		}
	}

	// Calculate trie node changes
	for owner, subset := range update.nodes.Sets {
		isAccountTrie := owner == (common.Hash{})
		var keyPrefixSize int64
		if isAccountTrie {
			keyPrefixSize = accountTrieKeyPrefixSize
		} else {
			keyPrefixSize = storageTrieKeyPrefixSize
		}

		// Iterate over Origins since every modified node has an origin entry
		for path, oldNode := range subset.Origins {
			newNode, hasNew := subset.Nodes[path]

			keySize := keyPrefixSize + int64(len(path))

			if len(oldNode) > 0 && (!hasNew || len(newNode.Blob) == 0) {
				// Node deletion
				diff.TrieNodeCount -= 1
				diff.TrieNodeBytes -= keySize + int64(len(oldNode))
			} else if len(oldNode) == 0 && hasNew && len(newNode.Blob) > 0 {
				// Node creation
				diff.TrieNodeCount += 1
				diff.TrieNodeBytes += keySize + int64(len(newNode.Blob))
			} else if len(oldNode) > 0 && hasNew && len(newNode.Blob) > 0 {
				// Node update
				diff.TrieNodeBytes += int64(len(newNode.Blob) - len(oldNode))
			}
		}
	}

	// Calculate code changes
	for _, code := range update.codes {
		diff.ContractCount += 1
		diff.ContractBytes += codeKeySize + int64(len(code.blob))
	}

	return diff
}

// Track is an async method used to send the state update to the generator.
// It ignores empty updates (where no state changes occurred).
// If the channel is full, it drops the update to avoid blocking.
func (g *StateSizeGenerator) Track(update *stateUpdate) {
	if update == nil || update.empty() {
		return
	}

	g.updateChan <- update
}

// hasExistingMetrics checks if state size metrics already exist in the database
// and if they are continuous with the current root
func (g *StateSizeGenerator) hasExistingMetrics() bool {
	data := rawdb.ReadStateSizeMetrics(g.db)
	if data == nil {
		return false
	}

	var existed stateSizeMetrics
	if err := json.Unmarshal(data, &existed); err != nil {
		log.Warn("Failed to decode existed state size metrics", "err", err)
		return false
	}

	// Check if the existing metrics root matches our current root
	if (g.metrics.Root != common.Hash{}) && existed.Root != g.metrics.Root {
		log.Info("Existing state size metrics found but root mismatch", "existed", existed.Root, "current", g.metrics.Root)
		return false
	}

	// Root matches - load the existing metrics
	log.Info("Loading existing state size metrics", "root", existed.Root)
	g.metrics = &existed
	return true
}

// initializeMetrics performs the actual metrics initialization using errgroup
func (g *StateSizeGenerator) initializeMetrics() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-g.abort:
			cancel() // Cancel context when abort is signaled
		case <-ctx.Done():
			// Context already cancelled
		}
	}()

	// Create errgroup with context
	group, ctx := errgroup.WithContext(ctx)

	// Metrics will be directly updated by each goroutine
	var (
		accountSnapCount, accountSnapBytes int64
		storageSnapCount, storageSnapBytes int64
		accountTrieCount, accountTrieBytes int64
		storageTrieCount, storageTrieBytes int64
		contractCount, contractBytes       int64
	)

	// Start all table iterations concurrently with direct metric updates
	group.Go(func() error {
		count, bytes, err := g.iterateTable(ctx, rawdb.SnapshotAccountPrefix, "accountSnap")
		if err != nil {
			return err
		}
		accountSnapCount, accountSnapBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := g.iterateTable(ctx, rawdb.SnapshotStoragePrefix, "storageSnap")
		if err != nil {
			return err
		}
		storageSnapCount, storageSnapBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := g.iterateTable(ctx, rawdb.TrieNodeAccountPrefix, "accountTrie")
		if err != nil {
			return err
		}
		accountTrieCount, accountTrieBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := g.iterateTable(ctx, rawdb.TrieNodeStoragePrefix, "storageTrie")
		if err != nil {
			return err
		}
		storageTrieCount, storageTrieBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := g.iterateTable(ctx, rawdb.CodePrefix, "contract")
		if err != nil {
			return err
		}
		contractCount, contractBytes = count, bytes
		return nil
	})

	// Wait for all goroutines to complete
	if err := group.Wait(); err != nil {
		return err
	}

	g.metrics.AccountCount = accountSnapCount
	g.metrics.AccountBytes = accountSnapBytes
	g.metrics.StorageCount = storageSnapCount
	g.metrics.StorageBytes = storageSnapBytes
	g.metrics.TrieNodeCount = accountTrieCount + storageTrieCount
	g.metrics.TrieNodeBytes = accountTrieBytes + storageTrieBytes
	g.metrics.ContractCount = contractCount
	g.metrics.ContractBytes = contractBytes

	g.updateMetrics()
	g.persistMetrics()

	return nil
}

// iterateTable performs iteration over a specific table and returns the results
func (g *StateSizeGenerator) iterateTable(ctx context.Context, prefix []byte, name string) (int64, int64, error) {
	log.Info("Iterating over state size", "table", name)
	start := time.Now()

	var count, bytes int64
	iter := g.db.NewIterator(prefix, nil)
	defer iter.Release()

	for iter.Next() {
		count++
		bytes += int64(len(iter.Key()) + len(iter.Value()))

		// Check for cancellation periodically for performance
		if count%10000 == 0 {
			select {
			case <-ctx.Done():
				log.Info("State size iteration cancelled", "table", name, "count", count)
				return 0, 0, ctx.Err()
			default:
			}
		}
	}

	// Check for iterator errors
	if err := iter.Error(); err != nil {
		log.Error("Iterator error during state size calculation", "table", name, "err", err)
		return 0, 0, err
	}

	log.Info("Finished iterating over state size", "table", name, "count", count, "bytes", bytes, "elapsed", common.PrettyDuration(time.Since(start)))

	return count, bytes, nil
}

func (g *StateSizeGenerator) updateMetrics() {
	accountCountGauge.Update(g.metrics.AccountCount)
	accountBytesGauge.Update(g.metrics.AccountBytes)
	storageCountGauge.Update(g.metrics.StorageCount)
	storageBytesGauge.Update(g.metrics.StorageBytes)
	trienodeCountGauge.Update(g.metrics.TrieNodeCount)
	trienodeBytesGauge.Update(g.metrics.TrieNodeBytes)
	contractCountGauge.Update(g.metrics.ContractCount)
	contractBytesGauge.Update(g.metrics.ContractBytes)
}

// persistMetrics saves the current metrics to the database
func (g *StateSizeGenerator) persistMetrics() {
	// RLP doesn't support int64, so we use JSON for simplicity
	data, err := json.Marshal(*g.metrics)
	if err != nil {
		log.Error("Failed to encode state size metrics", "err", err)
		return
	}

	batch := g.db.NewBatch()
	rawdb.WriteStateSizeMetrics(batch, data)
	if err := batch.Write(); err != nil {
		log.Error("Failed to persist state size metrics", "err", err)
	}
}
