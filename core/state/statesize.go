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
	"bytes"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
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
	abort   chan chan struct{}
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
		abort:   make(chan chan struct{}),
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
	ch := make(chan struct{})
	g.abort <- ch
	<-ch
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
		accountCount, accountBytes   uint64
		storageCount, storageBytes   uint64
		trieNodeCount, trieNodeBytes uint64
		contractCount, contractBytes uint64
	)

	// Process accounts
	log.Info("Initializing account metrics")
	accountIter := g.db.NewIterator(rawdb.SnapshotAccountPrefix, nil)
	defer accountIter.Release()

	for accountIter.Next() {
		key := accountIter.Key()
		value := accountIter.Value()

		// Count account
		accountCount++
		accountBytes += uint64(len(key) + len(value))

		// Check if account has code (contract)
		var account types.StateAccount
		if err := rlp.DecodeBytes(value, &account); err == nil {
			if !bytes.Equal(account.CodeHash, types.EmptyCodeHash[:]) {
				contractCount++
				// Code size will be counted separately
			}
		}

		// Process storage for this account
		storageIter := g.db.NewIterator(append(rawdb.SnapshotStoragePrefix, key[1:]...), nil)
		for storageIter.Next() {
			storageKey := storageIter.Key()
			storageValue := storageIter.Value()
			storageCount++
			storageBytes += uint64(len(storageKey) + len(storageValue))
		}
		storageIter.Release()

		// Check for abort
		select {
		case abort := <-g.abort:
			close(abort)
			return
		default:
		}
	}

	// Process trie nodes
	log.Info("Initializing trie node metrics")
	trieNodeIter := g.db.NewIterator(rawdb.TrieNodeAccountPrefix, nil)
	defer trieNodeIter.Release()

	for trieNodeIter.Next() {
		key := trieNodeIter.Key()
		value := trieNodeIter.Value()
		trieNodeCount++
		trieNodeBytes += uint64(len(key) + len(value))
	}

	// Process storage trie nodes
	storageTrieIter := g.db.NewIterator(rawdb.TrieNodeStoragePrefix, nil)
	defer storageTrieIter.Release()

	for storageTrieIter.Next() {
		key := storageTrieIter.Key()
		value := storageTrieIter.Value()
		trieNodeCount++
		trieNodeBytes += uint64(len(key) + len(value))
	}

	// Process contract code
	log.Info("Initializing contract code metrics")
	codeIter := g.db.NewIterator(rawdb.CodePrefix, nil)
	defer codeIter.Release()

	for codeIter.Next() {
		key := codeIter.Key()
		value := codeIter.Value()
		contractBytes += uint64(len(key) + len(value))
	}

	// Update metrics
	g.metricsLock.Lock()
	g.metrics.AccountCount = accountCount
	g.metrics.AccountBytes = accountBytes
	g.metrics.StorageCount = storageCount
	g.metrics.StorageBytes = storageBytes
	g.metrics.TrieNodeCount = trieNodeCount
	g.metrics.TrieNodeBytes = trieNodeBytes
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
	stateSizeTrieNodesCountMeter.Mark(int64(trieNodeCount))
	stateSizeTrieNodesBytesMeter.Mark(int64(trieNodeBytes))
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
	var diff StateSizeMetrics

	// Calculate account changes
	for _, data := range update.accounts {
		if len(data) > 0 {
			diff.AccountCount++
			diff.AccountBytes += uint64(common.HashLength + len(data))
		}
	}

	// Calculate storage changes
	for _, slots := range update.storages {
		for _, data := range slots {
			if len(data) > 0 {
				diff.StorageCount++
				diff.StorageBytes += uint64(2*common.HashLength + len(data))
			}
		}
	}

	// Calculate trie node changes
	for _, nodeSet := range update.nodes.Sets {
		for _, node := range nodeSet.Nodes {
			diff.TrieNodeCount++
			diff.TrieNodeBytes += uint64(len(node.Blob))
		}
	}

	// Update local metrics
	g.metricsLock.Lock()
	g.metrics.Root = update.root
	g.metrics.AccountCount += diff.AccountCount
	g.metrics.AccountBytes += diff.AccountBytes
	g.metrics.StorageCount += diff.StorageCount
	g.metrics.StorageBytes += diff.StorageBytes
	g.metrics.TrieNodeCount += diff.TrieNodeCount
	g.metrics.TrieNodeBytes += diff.TrieNodeBytes
	g.metricsLock.Unlock()
}
