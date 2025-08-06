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

// stateSizeGenerator handles the initialization and tracking of state size metrics
type stateSizeGenerator struct {
	db  ethdb.KeyValueStore
	sdb Database

	// Generator state
	running bool
	abort   chan struct{}
	done    chan struct{}

	// Metrics state
	metrics     *StateSizeMetrics
	metricsLock sync.RWMutex
}

// newStateSizeGenerator creates a new state size generator
func newStateSizeGenerator(db ethdb.KeyValueStore, sdb Database, root common.Hash) *stateSizeGenerator {
	return &stateSizeGenerator{
		db:      db,
		sdb:     sdb,
		abort:   make(chan struct{}),
		done:    make(chan struct{}),
		metrics: &StateSizeMetrics{Root: root},
	}
}

// run starts the state size initialization in the background
func (g *stateSizeGenerator) run() {
	if g.running {
		g.stop()
		log.Warn("Paused the leftover state size generation cycle")
	}
	g.running = true
	go g.generate()
}

// stop terminates the background generation
func (g *stateSizeGenerator) stop() {
	if !g.running {
		return
	}
	close(g.abort)
	g.running = false
}

// generate performs the state size initialization
func (g *stateSizeGenerator) generate() {
	defer close(g.done)
	start := time.Now()

	if g.hasExistingMetrics() {
		log.Info("State size metrics already initialized")
		return
	}

	// Wait for snapshot generator to complete
	if db := g.sdb.TrieDB(); db != nil {
		for !db.SnapshotCompleted() {
			time.Sleep(5 * time.Second)
		}
	}

	log.Info("Starting state size initialization")
	g.initializeMetrics()
	log.Info("Completed state size initialization", "elapsed", time.Since(start))
}

// hasExistingMetrics checks if state size metrics already exist in the database
func (g *stateSizeGenerator) hasExistingMetrics() bool {
	// Check for existing metrics by looking for a marker key
	marker := rawdb.ReadStateSizeMetrics(g.db)
	// TODO: check if the marker's root is the same as the current root
	return marker != nil
}

// initializeMetrics performs the actual metrics initialization
func (g *stateSizeGenerator) initializeMetrics() {
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
	g.metricsLock.Lock()
	g.metrics.AccountCount = accountCount
	g.metrics.AccountBytes = accountBytes
	g.metrics.StorageCount = storageCount
	g.metrics.StorageBytes = storageBytes
	g.metrics.TrieNodeCount = trieAccountNodeCount + trieStorageNodeCount
	g.metrics.TrieNodeBytes = trieAccountNodeBytes + trieStorageNodeBytes
	g.metrics.ContractCount = contractCount
	g.metrics.ContractBytes = contractBytes
	g.metricsLock.Unlock()

	// Update metrics in database
	g.persistMetrics()

	// Update global metrics
	stateSizeAccountsCountMeter.Mark(int64(accountCount))
	stateSizeAccountsBytesMeter.Mark(int64(accountBytes))
	stateSizeStorageCountMeter.Mark(int64(storageCount))
	stateSizeStorageBytesMeter.Mark(int64(storageBytes))
	stateSizeTrieNodesCountMeter.Mark(int64(trieAccountNodeCount + trieStorageNodeCount))
	stateSizeTrieNodesBytesMeter.Mark(int64(trieStorageNodeBytes + trieStorageNodeBytes))
	stateSizeContractsCountMeter.Mark(int64(contractCount))
	stateSizeContractsBytesMeter.Mark(int64(contractBytes))
}

// persistMetrics saves the current metrics to the database
func (g *stateSizeGenerator) persistMetrics() {
	g.metricsLock.RLock()
	metrics := *g.metrics
	g.metricsLock.RUnlock()

	data, err := rlp.EncodeToBytes(metrics)
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

// updateMetrics updates metrics based on state changes
func (g *stateSizeGenerator) updateMetrics(update *stateUpdate) {
	var (
		accountBytes, storageBytes, nodeBytes, codeBytes int
		accountCount, storageCount, nodeCount, codeCount int
	)

	for addr, oldValue := range update.accountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newValue, exists := update.accounts[addrHash]
		if !exists {
			log.Warn("State update missing account", "address", addr)
			continue
		}
		if len(newValue) == 0 {
			accountCount -= 1
			accountBytes -= common.HashLength
		}
		if len(oldValue) == 0 {
			accountCount += 1
			accountBytes += common.HashLength
		}
		accountBytes += len(newValue) - len(oldValue)
	}

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
				storageCount -= 1
				storageBytes -= common.HashLength
			}
			if len(oldValue) == 0 {
				storageCount += 1
				storageBytes += common.HashLength
			}
			storageBytes += len(newValue) - len(oldValue)
		}
	}
	for _, subset := range update.nodes.Sets {
		for path, n := range subset.Nodes {
			if len(n.Blob) == 0 {
				nodeCount -= 1
				nodeBytes -= len(path) + common.HashLength
			}
			prev, ok := subset.Origins[path]
			if ok {
				nodeCount += 1
				nodeBytes += len(path) + common.HashLength
			}
			nodeBytes += len(n.Blob) - len(prev)
		}
	}
	for _, code := range update.codes {
		codeCount += 1
		codeBytes += len(code.blob) + common.HashLength // no deduplication
	}

	// Update local metrics
	g.metricsLock.Lock()
	g.metrics.Root = update.root
	g.metrics.AccountCount += uint64(accountCount)
	g.metrics.AccountBytes += uint64(accountBytes)
	g.metrics.StorageCount += uint64(storageCount)
	g.metrics.StorageBytes += uint64(storageBytes)
	g.metrics.TrieNodeCount += uint64(nodeCount)
	g.metrics.TrieNodeBytes += uint64(nodeBytes)
	g.metrics.ContractCount += uint64(codeCount)
	g.metrics.ContractBytes += uint64(codeBytes)
	g.metricsLock.Unlock()
}
