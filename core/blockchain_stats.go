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

package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// ExecuteStats includes all the statistics of a block execution in details.
type ExecuteStats struct {
	// State read times
	AccountReads   time.Duration // Time spent on the account reads
	StorageReads   time.Duration // Time spent on the storage reads
	AccountHashes  time.Duration // Time spent on the account trie hash
	AccountUpdates time.Duration // Time spent on the account trie update
	AccountCommits time.Duration // Time spent on the account trie commit
	StorageUpdates time.Duration // Time spent on the storage trie update
	StorageCommits time.Duration // Time spent on the storage trie commit
	CodeReads      time.Duration // Time spent on the contract code read

	AccountLoaded  int // Number of accounts loaded
	AccountUpdated int // Number of accounts updated
	AccountDeleted int // Number of accounts deleted
	StorageLoaded  int // Number of storage slots loaded
	StorageUpdated int // Number of storage slots updated
	StorageDeleted int // Number of storage slots deleted
	CodeLoaded     int // Number of contract code loaded

	Execution       time.Duration // Time spent on the EVM execution
	Validation      time.Duration // Time spent on the block validation
	CrossValidation time.Duration // Optional, time spent on the block cross validation
	SnapshotCommit  time.Duration // Time spent on snapshot commit
	TrieDBCommit    time.Duration // Time spent on database commit
	BlockWrite      time.Duration // Time spent on block write
	TotalTime       time.Duration // The total time spent on block execution
	MgasPerSecond   float64       // The million gas processed per second

	// Cache hit rates
	StateReadCacheStats     state.ReaderStats
	StatePrefetchCacheStats state.ReaderStats
}

// reportMetrics uploads execution statistics to the metrics system.
func (s *ExecuteStats) reportMetrics() {
	if s.AccountLoaded != 0 {
		accountReadTimer.Update(s.AccountReads)
		accountReadSingleTimer.Update(s.AccountReads / time.Duration(s.AccountLoaded))
	}
	if s.StorageLoaded != 0 {
		storageReadTimer.Update(s.StorageReads)
		storageReadSingleTimer.Update(s.StorageReads / time.Duration(s.StorageLoaded))
	}
	if s.CodeLoaded != 0 {
		codeReadTimer.Update(s.CodeReads)
		codeReadSingleTimer.Update(s.CodeReads / time.Duration(s.CodeLoaded))
	}
	accountUpdateTimer.Update(s.AccountUpdates) // Account updates are complete(in validation)
	storageUpdateTimer.Update(s.StorageUpdates) // Storage updates are complete(in validation)
	accountHashTimer.Update(s.AccountHashes)    // Account hashes are complete(in validation)
	accountCommitTimer.Update(s.AccountCommits) // Account commits are complete, we can mark them
	storageCommitTimer.Update(s.StorageCommits) // Storage commits are complete, we can mark them

	blockExecutionTimer.Update(s.Execution)                 // The time spent on EVM processing
	blockValidationTimer.Update(s.Validation)               // The time spent on block validation
	blockCrossValidationTimer.Update(s.CrossValidation)     // The time spent on stateless cross validation
	snapshotCommitTimer.Update(s.SnapshotCommit)            // Snapshot commits are complete, we can mark them
	triedbCommitTimer.Update(s.TrieDBCommit)                // Trie database commits are complete, we can mark them
	blockWriteTimer.Update(s.BlockWrite)                    // The time spent on block write
	blockInsertTimer.Update(s.TotalTime)                    // The total time spent on block execution
	chainMgaspsMeter.Update(time.Duration(s.MgasPerSecond)) // TODO(rjl493456442) generalize the ResettingTimer

	// Cache hit rates
	accountCacheHitPrefetchMeter.Mark(s.StatePrefetchCacheStats.AccountCacheHit)
	accountCacheMissPrefetchMeter.Mark(s.StatePrefetchCacheStats.AccountCacheMiss)
	storageCacheHitPrefetchMeter.Mark(s.StatePrefetchCacheStats.StorageCacheHit)
	storageCacheMissPrefetchMeter.Mark(s.StatePrefetchCacheStats.StorageCacheMiss)

	accountCacheHitMeter.Mark(s.StateReadCacheStats.AccountCacheHit)
	accountCacheMissMeter.Mark(s.StateReadCacheStats.AccountCacheMiss)
	storageCacheHitMeter.Mark(s.StateReadCacheStats.StorageCacheHit)
	storageCacheMissMeter.Mark(s.StateReadCacheStats.StorageCacheMiss)
}

// logSlow prints the detailed execution statistics if the block is regarded as slow.
func (s *ExecuteStats) logSlow(block *types.Block, slowBlockThreshold time.Duration) {
	if slowBlockThreshold == 0 {
		return
	}
	if s.TotalTime < slowBlockThreshold {
		return
	}
	msg := fmt.Sprintf(`
########## SLOW BLOCK #########
Block: %v (%#x) txs: %d, mgasps: %.2f, elapsed: %v

EVM execution: %v
Validation: %v
State read: %v
    Account read: %v(%d)
    Storage read: %v(%d)
    Code read: %v(%d)

State hash: %v
    Account hash: %v
    Storage hash: %v
    Trie commit: %v

DB write: %v
    State write: %v
    Block write: %v

%s
##############################
`, block.Number(), block.Hash(), len(block.Transactions()), s.MgasPerSecond, common.PrettyDuration(s.TotalTime),
		common.PrettyDuration(s.Execution),
		common.PrettyDuration(s.Validation+s.CrossValidation),

		// State read
		common.PrettyDuration(s.AccountReads+s.StorageReads+s.CodeReads),
		common.PrettyDuration(s.AccountReads), s.AccountLoaded,
		common.PrettyDuration(s.StorageReads), s.StorageLoaded,
		common.PrettyDuration(s.CodeReads), s.CodeLoaded,

		// State hash
		common.PrettyDuration(s.AccountHashes+s.AccountUpdates+s.StorageUpdates+max(s.AccountCommits, s.StorageCommits)),
		common.PrettyDuration(s.AccountHashes+s.AccountUpdates),
		common.PrettyDuration(s.StorageUpdates),
		common.PrettyDuration(max(s.AccountCommits, s.StorageCommits)),

		// Database commit
		common.PrettyDuration(s.TrieDBCommit+s.SnapshotCommit+s.BlockWrite),
		common.PrettyDuration(s.TrieDBCommit+s.SnapshotCommit),
		common.PrettyDuration(s.BlockWrite),

		// cache statistics
		s.StateReadCacheStats)
	for _, line := range strings.Split(msg, "\n") {
		if line == "" {
			continue
		}
		log.Info(line)
	}
}
