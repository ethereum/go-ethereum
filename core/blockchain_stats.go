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
	"encoding/json"
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

	AccountLoaded   int // Number of accounts loaded
	AccountUpdated  int // Number of accounts updated
	AccountDeleted  int // Number of accounts deleted
	StorageLoaded   int // Number of storage slots loaded
	StorageUpdated  int // Number of storage slots updated
	StorageDeleted  int // Number of storage slots deleted
	CodeLoaded      int // Number of contract code loaded
	CodeLoadBytes   int // Number of bytes read from contract code
	CodeUpdated     int // Number of contract code written (CREATE/CREATE2 + EIP-7702)
	CodeUpdateBytes int // Total bytes of code written

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
		codeReadBytesTimer.Update(time.Duration(s.CodeLoadBytes))
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

// slowBlockLog represents the JSON structure for slow block logging.
// This format is designed for cross-client compatibility with other
// Ethereum execution clients (reth, Besu, Nethermind).
type slowBlockLog struct {
	Level       string          `json:"level"`
	Msg         string          `json:"msg"`
	Block       slowBlockInfo   `json:"block"`
	Timing      slowBlockTime   `json:"timing"`
	Throughput  slowBlockThru   `json:"throughput"`
	StateReads  slowBlockReads  `json:"state_reads"`
	StateWrites slowBlockWrites `json:"state_writes"`
	Cache       slowBlockCache  `json:"cache"`
}

type slowBlockInfo struct {
	Number  uint64      `json:"number"`
	Hash    common.Hash `json:"hash"`
	GasUsed uint64      `json:"gas_used"`
	TxCount int         `json:"tx_count"`
}

type slowBlockTime struct {
	ExecutionMs float64 `json:"execution_ms"`
	StateReadMs float64 `json:"state_read_ms"`
	StateHashMs float64 `json:"state_hash_ms"`
	CommitMs    float64 `json:"commit_ms"`
	TotalMs     float64 `json:"total_ms"`
}

type slowBlockThru struct {
	MgasPerSec float64 `json:"mgas_per_sec"`
}

type slowBlockReads struct {
	Accounts     int `json:"accounts"`
	StorageSlots int `json:"storage_slots"`
	Code         int `json:"code"`
	CodeBytes    int `json:"code_bytes"`
}

type slowBlockWrites struct {
	Accounts            int `json:"accounts"`
	AccountsDeleted     int `json:"accounts_deleted"`
	StorageSlots        int `json:"storage_slots"`
	StorageSlotsDeleted int `json:"storage_slots_deleted"`
	Code                int `json:"code"`
	CodeBytes           int `json:"code_bytes"`
}

// slowBlockCache represents cache hit/miss statistics for cross-client analysis.
type slowBlockCache struct {
	Account slowBlockCacheEntry     `json:"account"`
	Storage slowBlockCacheEntry     `json:"storage"`
	Code    slowBlockCodeCacheEntry `json:"code"`
}

// slowBlockCacheEntry represents cache statistics for account/storage caches.
type slowBlockCacheEntry struct {
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hit_rate"`
}

// slowBlockCodeCacheEntry represents cache statistics for code cache with byte-level granularity.
type slowBlockCodeCacheEntry struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	HitBytes  int64   `json:"hit_bytes"`
	MissBytes int64   `json:"miss_bytes"`
}

// calculateHitRate computes the cache hit rate as a percentage (0-100).
func calculateHitRate(hits, misses int64) float64 {
	if total := hits + misses; total > 0 {
		return float64(hits) / float64(total) * 100.0
	}
	return 0.0
}

// durationToMs converts a time.Duration to milliseconds as a float64
// with sub-millisecond precision for accurate cross-client metrics.
func durationToMs(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

// logSlow prints the detailed execution statistics in JSON format if the block
// is regarded as slow. The JSON format is designed for cross-client compatibility
// with other Ethereum execution clients.
func (s *ExecuteStats) logSlow(block *types.Block, slowBlockThreshold time.Duration) {
	// Negative threshold means disabled (default when flag not set)
	if slowBlockThreshold < 0 {
		return
	}
	// Threshold of 0 logs all blocks; positive threshold filters
	if slowBlockThreshold > 0 && s.TotalTime < slowBlockThreshold {
		return
	}
	logEntry := slowBlockLog{
		Level: "warn",
		Msg:   "Slow block",
		Block: slowBlockInfo{
			Number:  block.NumberU64(),
			Hash:    block.Hash(),
			GasUsed: block.GasUsed(),
			TxCount: len(block.Transactions()),
		},
		Timing: slowBlockTime{
			ExecutionMs: durationToMs(s.Execution),
			StateReadMs: durationToMs(s.AccountReads + s.StorageReads + s.CodeReads),
			StateHashMs: durationToMs(s.AccountHashes + s.AccountUpdates + s.StorageUpdates),
			CommitMs:    durationToMs(max(s.AccountCommits, s.StorageCommits) + s.TrieDBCommit + s.SnapshotCommit + s.BlockWrite),
			TotalMs:     durationToMs(s.TotalTime),
		},
		Throughput: slowBlockThru{
			MgasPerSec: s.MgasPerSecond,
		},
		StateReads: slowBlockReads{
			Accounts:     s.AccountLoaded,
			StorageSlots: s.StorageLoaded,
			Code:         s.CodeLoaded,
			CodeBytes:    s.CodeLoadBytes,
		},
		StateWrites: slowBlockWrites{
			Accounts:            s.AccountUpdated,
			AccountsDeleted:     s.AccountDeleted,
			StorageSlots:        s.StorageUpdated,
			StorageSlotsDeleted: s.StorageDeleted,
			Code:                s.CodeUpdated,
			CodeBytes:           s.CodeUpdateBytes,
		},
		Cache: slowBlockCache{
			Account: slowBlockCacheEntry{
				Hits:    s.StateReadCacheStats.AccountCacheHit,
				Misses:  s.StateReadCacheStats.AccountCacheMiss,
				HitRate: calculateHitRate(s.StateReadCacheStats.AccountCacheHit, s.StateReadCacheStats.AccountCacheMiss),
			},
			Storage: slowBlockCacheEntry{
				Hits:    s.StateReadCacheStats.StorageCacheHit,
				Misses:  s.StateReadCacheStats.StorageCacheMiss,
				HitRate: calculateHitRate(s.StateReadCacheStats.StorageCacheHit, s.StateReadCacheStats.StorageCacheMiss),
			},
			Code: slowBlockCodeCacheEntry{
				Hits:      s.StateReadCacheStats.CodeStats.CacheHit,
				Misses:    s.StateReadCacheStats.CodeStats.CacheMiss,
				HitRate:   calculateHitRate(s.StateReadCacheStats.CodeStats.CacheHit, s.StateReadCacheStats.CodeStats.CacheMiss),
				HitBytes:  s.StateReadCacheStats.CodeStats.CacheHitBytes,
				MissBytes: s.StateReadCacheStats.CodeStats.CacheMissBytes,
			},
		},
	}
	jsonBytes, err := json.Marshal(logEntry)
	if err != nil {
		log.Error("Failed to marshal slow block log", "error", err)
		return
	}
	log.Warn(string(jsonBytes))
}
