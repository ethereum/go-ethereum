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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb"
)

// State size metrics
var (
	// Baseline state size metrics
	stateSizeAccountsCountMeter  = metrics.NewRegisteredMeter("statedb/statesize/accounts/count", nil)
	stateSizeAccountsBytesMeter  = metrics.NewRegisteredMeter("statedb/statesize/accounts/bytes", nil)
	stateSizeStorageCountMeter   = metrics.NewRegisteredMeter("statedb/statesize/storage/count", nil)
	stateSizeStorageBytesMeter   = metrics.NewRegisteredMeter("statedb/statesize/storage/bytes", nil)
	stateSizeTrieNodesCountMeter = metrics.NewRegisteredMeter("statedb/statesize/trienodes/count", nil)
	stateSizeTrieNodesBytesMeter = metrics.NewRegisteredMeter("statedb/statesize/trienodes/bytes", nil)
	stateSizeContractsCountMeter = metrics.NewRegisteredMeter("statedb/statesize/contracts/count", nil)
	stateSizeContractsBytesMeter = metrics.NewRegisteredMeter("statedb/statesize/contracts/bytes", nil)
)

// StateSizeMetrics represents the current state size statistics
type StateSizeMetrics struct {
	Root          common.Hash // Root hash of the state trie
	AccountCount  uint64
	AccountBytes  uint64
	StorageCount  uint64
	StorageBytes  uint64
	TrieNodeCount uint64
	TrieNodeBytes uint64
	ContractCount uint64
	ContractBytes uint64
}

// StateSizeGenerator handles the initialization and tracking of state size metrics
type StateSizeGenerator struct {
	db     ethdb.KeyValueStore
	triedb *triedb.Database

	// Generator state
	running bool
	abort   chan struct{}
	done    chan struct{}

	// Async message channel for updates
	updateChan chan *stateUpdate

	// Metrics state
	metrics  *StateSizeMetrics
	buffered *StateSizeMetrics
}

// NewStateSizeGenerator creates a new state size generator and starts it automatically
func NewStateSizeGenerator(db ethdb.KeyValueStore, triedb *triedb.Database, root common.Hash) *StateSizeGenerator {
	g := &StateSizeGenerator{
		db:         db,
		triedb:     triedb,
		abort:      make(chan struct{}),
		done:       make(chan struct{}),
		updateChan: make(chan *stateUpdate, 1000), // Buffered channel for updates
		metrics:    &StateSizeMetrics{Root: root},
		buffered:   &StateSizeMetrics{Root: root},
	}

	// Start the generator automatically
	g.running = true
	go g.generate()

	return g
}

// stop terminates the background generation
func (g *StateSizeGenerator) Stop() {
	if !g.running {
		return
	}

	// Signal the goroutine to stop
	close(g.abort)

	// Wait for the goroutine to actually finish
	<-g.done

	// Now it's safe to persist metrics since the goroutine has stopped
	g.running = false
	g.persistMetrics()
}

// isRunning returns true if the generator is currently running
func (g *StateSizeGenerator) IsRunning() bool {
	return g.running
}

// waitForCompletion waits for the generator to complete (useful for testing or graceful shutdown)
func (g *StateSizeGenerator) WaitForCompletion() {
	if g.running {
		<-g.done
	}
}

// generate performs the state size initialization and handles updates
func (g *StateSizeGenerator) generate() {
	defer close(g.done)

	var inited bool
	var initDone chan struct{}

	if g.hasExistingMetrics() {
		log.Info("State size metrics already initialized")
		inited = true
	}

	// Wait for snapshot generator to complete
	snapDone := make(chan struct{})
	go func() {
		defer close(snapDone)

		for !g.triedb.SnapshotCompleted() {
			select {
			case <-g.abort:
				log.Info("State size generation aborted during snapshot")
				return
			default:
				time.Sleep(10 * time.Second)
			}
		}
	}()

	for {
		select {
		case update := <-g.updateChan:
			g.handleUpdate(update, inited)
		case <-g.abort:
			log.Info("State size generation aborted")
			// Wait for initialization goroutine to finish if it's running
			if initDone != nil {
				select {
				case <-initDone:
				case <-time.After(5 * time.Second):
					log.Warn("Initialization goroutine did not finish in time")
				}
			}
			return
		case <-snapDone:
			if !inited {
				initDone = make(chan struct{})
				go func() {
					defer close(initDone)
					start := time.Now()
					log.Info("Starting state size initialization")
					g.initializeMetrics()
					log.Info("Completed state size initialization", "elapsed", time.Since(start))
					inited = true
				}()
			}
		case <-initDone:
			// Initialization completed, merge buffered metrics if needed
			if g.buffered != nil && g.buffered.Root != (common.Hash{}) {
				log.Info("Merging buffered metrics into main metrics")
				g.metrics.AccountCount += g.buffered.AccountCount
				g.metrics.AccountBytes += g.buffered.AccountBytes
				g.metrics.StorageCount += g.buffered.StorageCount
				g.metrics.StorageBytes += g.buffered.StorageBytes
				g.metrics.TrieNodeCount += g.buffered.TrieNodeCount
				g.metrics.TrieNodeBytes += g.buffered.TrieNodeBytes
				g.metrics.ContractCount += g.buffered.ContractCount
				g.metrics.ContractBytes += g.buffered.ContractBytes
				// Reset buffered metrics
				g.buffered = &StateSizeMetrics{Root: g.metrics.Root}
			}
			initDone = nil // Clear the channel
		}
	}
}

// handleUpdate processes a single update with proper root continuity checking
func (g *StateSizeGenerator) handleUpdate(update *stateUpdate, inited bool) {
	// TODO: Check if the update root matches the current metrics root

	// Calculate the diff
	diff := g.calculateUpdateDiff(update)

	var m *StateSizeMetrics
	if inited {
		m = g.metrics
	} else {
		m = g.buffered
	}

	// TODO: When to merge the buffered metrics into the main metrics
	m.Root = update.root
	m.AccountCount += diff.AccountCount
	m.AccountBytes += diff.AccountBytes
	m.StorageCount += diff.StorageCount
	m.StorageBytes += diff.StorageBytes
	m.TrieNodeCount += diff.TrieNodeCount
	m.TrieNodeBytes += diff.TrieNodeBytes
	m.ContractCount += diff.ContractCount
	m.ContractBytes += diff.ContractBytes

	// Fire the metrics only if the initialization is done
	if inited {
		g.updateMetrics()
		g.persistMetrics()
	}
}

// calculateUpdateDiff calculates the diff for a state update
func (g *StateSizeGenerator) calculateUpdateDiff(update *stateUpdate) StateSizeMetrics {
	var diff StateSizeMetrics

	// Calculate account changes
	for addr, oldValue := range update.accountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newValue, exists := update.accounts[addrHash]
		if !exists {
			log.Warn("State update missing account", "address", addr)
			continue
		}
		if len(newValue) == 0 {
			diff.AccountCount -= 1
			diff.AccountBytes -= common.HashLength
		}
		if len(oldValue) == 0 {
			diff.AccountCount += 1
			diff.AccountBytes += common.HashLength
		}
		diff.AccountBytes += uint64(len(newValue) - len(oldValue))
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
			if len(newValue) == 0 {
				diff.StorageCount -= 1
				diff.StorageBytes -= common.HashLength
			}
			if len(oldValue) == 0 {
				diff.StorageCount += 1
				diff.StorageBytes += common.HashLength
			}
			diff.StorageBytes += uint64(len(newValue) - len(oldValue))
		}
	}

	// Calculate trie node changes
	for _, subset := range update.nodes.Sets {
		for path, n := range subset.Nodes {
			if len(n.Blob) == 0 {
				diff.TrieNodeCount -= 1
				diff.TrieNodeBytes -= uint64(len(path) + common.HashLength)
			}
			prev, ok := subset.Origins[path]
			if ok {
				diff.TrieNodeCount += 1
				diff.TrieNodeBytes += uint64(len(path) + common.HashLength)
			}
			diff.TrieNodeBytes += uint64(len(n.Blob) - len(prev))
		}
	}

	// Calculate code changes
	for _, code := range update.codes {
		diff.ContractCount += 1
		diff.ContractBytes += uint64(len(code.blob) + common.HashLength)
	}

	return diff
}

// Track is an async method used to send the state update to the generator.
// It ignores empty updates (where no state changes occurred).
func (g *StateSizeGenerator) Track(update *stateUpdate) {
	if update == nil || update.empty() {
		return
	}
	g.updateChan <- update
}

// hasExistingMetrics checks if state size metrics already exist in the database
func (g *StateSizeGenerator) hasExistingMetrics() bool {
	// Check for existing metrics by looking for a marker key
	marker := rawdb.ReadStateSizeMetrics(g.db)
	// TODO: check if the marker's root is the same as the current root
	return marker != nil
}

// initializeMetrics performs the actual metrics initialization
func (g *StateSizeGenerator) initializeMetrics() {
	var (
		wg                                         sync.WaitGroup
		accountCount, accountBytes                 uint64
		storageCount, storageBytes                 uint64
		trieAccountNodeCount, trieAccountNodeBytes uint64
		trieStorageNodeCount, trieStorageNodeBytes uint64
		contractCount, contractBytes               uint64
	)

	iterate := func(prefix []byte, name string, count, bytes uint64) {
		defer wg.Done()

		log.Info("Iterating over state size", "table", name)
		defer func(st time.Time) {
			log.Info("Finished iterating over state size", "table", name, "count", count, "bytes", bytes, "elapsed", common.PrettyDuration(time.Since(st)))
		}(time.Now())

		iter := g.db.NewIterator(prefix, nil)
		defer iter.Release()
		for iter.Next() {
			count++
			bytes += uint64(len(iter.Key()) + len(iter.Value()))

			// Check for abort
			select {
			case <-g.abort:
				return
			default:
			}
		}
	}

	tables := []struct {
		prefix []byte
		name   string
		count  *uint64
		bytes  *uint64
	}{
		{rawdb.SnapshotAccountPrefix, "account", &accountCount, &accountBytes},
		{rawdb.SnapshotStoragePrefix, "storage", &storageCount, &storageBytes},
		{rawdb.TrieNodeAccountPrefix, "trie account node", &trieAccountNodeCount, &trieAccountNodeBytes},
		{rawdb.TrieNodeStoragePrefix, "trie storage node", &trieStorageNodeCount, &trieStorageNodeBytes},
		{rawdb.CodePrefix, "contract code", &contractCount, &contractBytes},
	}
	wg.Add(len(tables))
	for _, table := range tables {
		go iterate(table.prefix, table.name, *table.count, *table.bytes)
	}

	wg.Wait()

	// Update metrics
	g.metrics.AccountCount = accountCount
	g.metrics.AccountBytes = accountBytes
	g.metrics.StorageCount = storageCount
	g.metrics.StorageBytes = storageBytes
	g.metrics.TrieNodeCount = trieAccountNodeCount + trieStorageNodeCount
	g.metrics.TrieNodeBytes = trieAccountNodeBytes + trieStorageNodeBytes
	g.metrics.ContractCount = contractCount
	g.metrics.ContractBytes = contractBytes

	g.updateMetrics()
	g.persistMetrics()
}

func (g *StateSizeGenerator) updateMetrics() {
	// Update global metrics
	stateSizeAccountsCountMeter.Mark(int64(g.metrics.AccountCount))
	stateSizeAccountsBytesMeter.Mark(int64(g.metrics.AccountBytes))
	stateSizeStorageCountMeter.Mark(int64(g.metrics.StorageCount))
	stateSizeStorageBytesMeter.Mark(int64(g.metrics.StorageBytes))
	stateSizeTrieNodesCountMeter.Mark(int64(g.metrics.TrieNodeCount))
	stateSizeTrieNodesBytesMeter.Mark(int64(g.metrics.TrieNodeBytes))
	stateSizeContractsCountMeter.Mark(int64(g.metrics.ContractCount))
	stateSizeContractsBytesMeter.Mark(int64(g.metrics.ContractBytes))
}

// persistMetrics saves the current metrics to the database
func (g *StateSizeGenerator) persistMetrics() {
	data, err := rlp.EncodeToBytes(*g.metrics)
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
