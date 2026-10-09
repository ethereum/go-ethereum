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
	Process  ProcessStats  // Statistics of the block processing
	Validate ValidateStats // Statistics of the block validation
	Commit   CommitStats   // Statistics of the state commit

	CrossValidation time.Duration // Optional, time spent on the block cross validation
	TotalTime       time.Duration // The total time spent on block execution
	MgasPerSecond   float64       // The million gas processed per second

	// Cache hit rates
	StateReadCacheStats     state.ReaderStats
	StatePrefetchCacheStats state.ReaderStats
}

// ProcessStats contains the statistics recorded by the Processor. The times are
// spent on the critical path, while the state reads are summed over all the states
// used for execution, which run concurrently in the parallel execution.
type ProcessStats struct {
	Execution     time.Duration // Time spent on the system calls and transactions
	AccountReads  time.Duration // Time spent on the account reads
	StorageReads  time.Duration // Time spent on the storage reads
	CodeReads     time.Duration // Time spent on the contract code reads
	AccountLoaded int           // Number of accounts loaded
	StorageLoaded int           // Number of storage slots loaded
	CodeLoaded    int           // Number of contract code loaded
	CodeLoadBytes int           // Number of bytes read from contract code
}

// stateReads returns the total time spent on state reading.
func (s *ProcessStats) stateReads() time.Duration {
	return s.AccountReads + s.StorageReads + s.CodeReads
}

// addReads accumulates the state reads performed through the given state.
func (s *ProcessStats) addReads(db *state.StateDB) {
	s.AccountReads += db.AccountReads
	s.StorageReads += db.StorageReads
	s.CodeReads += db.CodeReads
	s.AccountLoaded += db.AccountLoaded
	s.StorageLoaded += db.StorageLoaded
	s.CodeLoaded += db.CodeLoaded
	s.CodeLoadBytes += db.CodeLoadBytes
}

// mergeReads accumulates the state reads of another set of statistics.
func (s *ProcessStats) mergeReads(o *ProcessStats) {
	s.AccountReads += o.AccountReads
	s.StorageReads += o.StorageReads
	s.CodeReads += o.CodeReads
	s.AccountLoaded += o.AccountLoaded
	s.StorageLoaded += o.StorageLoaded
	s.CodeLoaded += o.CodeLoaded
	s.CodeLoadBytes += o.CodeLoadBytes
}

// ValidateStats contains the statistics recorded by the Validator.
type ValidateStats struct {
	Validation time.Duration // Time spent on the block validation, excluding the state hashing

	// Trie hashing of the state, including the part done ahead of the validation
	AccountHashes  time.Duration // Time spent on the account trie hash
	AccountUpdates time.Duration // Time spent on the account trie update
	StorageUpdates time.Duration // Time spent on the storage trie update

	AccountUpdated  int // Number of accounts updated
	AccountDeleted  int // Number of accounts deleted
	StorageUpdated  int // Number of storage slots updated
	StorageDeleted  int // Number of storage slots deleted
	CodeUpdated     int // Number of contract code written (CREATE/CREATE2 + EIP-7702)
	CodeUpdateBytes int // Total bytes of code written
}

// stateHashes returns the total time spent on state hashing.
func (s *ValidateStats) stateHashes() time.Duration {
	return s.AccountHashes + s.AccountUpdates + s.StorageUpdates
}

// CommitStats contains the statistics of the state commit, gathered from the
// canonical state.
type CommitStats struct {
	AccountCommits time.Duration // Time spent on the account trie commit
	StorageCommits time.Duration // Time spent on the storage trie commit
	DatabaseCommit time.Duration // Time spent on database commit
}

// reportMetrics uploads execution statistics to the metrics system.
func (s *ExecuteStats) reportMetrics() {
	p, v, c := &s.Process, &s.Validate, &s.Commit
	if p.AccountLoaded != 0 {
		accountReadTimer.Update(p.AccountReads)
		accountReadSingleTimer.Update(p.AccountReads / time.Duration(p.AccountLoaded))
	}
	if p.StorageLoaded != 0 {
		storageReadTimer.Update(p.StorageReads)
		storageReadSingleTimer.Update(p.StorageReads / time.Duration(p.StorageLoaded))
	}
	if p.CodeLoaded != 0 {
		codeReadTimer.Update(p.CodeReads)
		codeReadSingleTimer.Update(p.CodeReads / time.Duration(p.CodeLoaded))
		codeReadBytesTimer.Update(time.Duration(p.CodeLoadBytes))
	}
	accountUpdateTimer.Update(v.AccountUpdates) // Account updates are complete(in validation)
	storageUpdateTimer.Update(v.StorageUpdates) // Storage updates are complete(in validation)
	accountHashTimer.Update(v.AccountHashes)    // Account hashes are complete(in validation)
	accountCommitTimer.Update(c.AccountCommits) // Account commits are complete, we can mark them
	storageCommitTimer.Update(c.StorageCommits) // Storage commits are complete, we can mark them

	blockExecutionTimer.Update(p.Execution)                 // The time spent on EVM processing
	blockValidationTimer.Update(v.Validation)               // The time spent on block validation
	blockCrossValidationTimer.Update(s.CrossValidation)     // The time spent on stateless cross validation
	triedbCommitTimer.Update(c.DatabaseCommit)              // Trie database commits are complete, we can mark them
	blockInsertTimer.Update(s.TotalTime)                    // The total time spent on block execution
	chainMgaspsMeter.Update(time.Duration(s.MgasPerSecond)) // TODO(rjl493456442) generalize the ResettingTimer

	// Cache hit rates
	accountCacheHitPrefetchMeter.Mark(s.StatePrefetchCacheStats.StateStats.AccountCacheHit)
	accountCacheMissPrefetchMeter.Mark(s.StatePrefetchCacheStats.StateStats.AccountCacheMiss)
	storageCacheHitPrefetchMeter.Mark(s.StatePrefetchCacheStats.StateStats.StorageCacheHit)
	storageCacheMissPrefetchMeter.Mark(s.StatePrefetchCacheStats.StateStats.StorageCacheMiss)
	codeCacheHitPrefetchMeter.Mark(s.StatePrefetchCacheStats.CodeStats.CacheHit)
	codeCacheMissPrefetchMeter.Mark(s.StatePrefetchCacheStats.CodeStats.CacheMiss)

	accountCacheHitMeter.Mark(s.StateReadCacheStats.StateStats.AccountCacheHit)
	accountCacheMissMeter.Mark(s.StateReadCacheStats.StateStats.AccountCacheMiss)
	storageCacheHitMeter.Mark(s.StateReadCacheStats.StateStats.StorageCacheHit)
	storageCacheMissMeter.Mark(s.StateReadCacheStats.StateStats.StorageCacheMiss)
	codeCacheHitMeter.Mark(s.StateReadCacheStats.CodeStats.CacheHit)
	codeCacheMissMeter.Mark(s.StateReadCacheStats.CodeStats.CacheMiss)
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
			ExecutionMs: durationToMs(s.Process.Execution),
			StateReadMs: durationToMs(s.Process.stateReads()),
			StateHashMs: durationToMs(s.Validate.stateHashes()),
			CommitMs:    durationToMs(max(s.Commit.AccountCommits, s.Commit.StorageCommits) + s.Commit.DatabaseCommit),
			TotalMs:     durationToMs(s.TotalTime),
		},
		Throughput: slowBlockThru{
			MgasPerSec: s.MgasPerSecond,
		},
		StateReads: slowBlockReads{
			Accounts:     s.Process.AccountLoaded,
			StorageSlots: s.Process.StorageLoaded,
			Code:         s.Process.CodeLoaded,
			CodeBytes:    s.Process.CodeLoadBytes,
		},
		StateWrites: slowBlockWrites{
			Accounts:            s.Validate.AccountUpdated,
			AccountsDeleted:     s.Validate.AccountDeleted,
			StorageSlots:        s.Validate.StorageUpdated,
			StorageSlotsDeleted: s.Validate.StorageDeleted,
			Code:                s.Validate.CodeUpdated,
			CodeBytes:           s.Validate.CodeUpdateBytes,
		},
		Cache: slowBlockCache{
			Account: slowBlockCacheEntry{
				Hits:    s.StateReadCacheStats.StateStats.AccountCacheHit,
				Misses:  s.StateReadCacheStats.StateStats.AccountCacheMiss,
				HitRate: s.StateReadCacheStats.StateStats.AccountCacheHitRate(),
			},
			Storage: slowBlockCacheEntry{
				Hits:    s.StateReadCacheStats.StateStats.StorageCacheHit,
				Misses:  s.StateReadCacheStats.StateStats.StorageCacheMiss,
				HitRate: s.StateReadCacheStats.StateStats.StorageCacheHitRate(),
			},
			Code: slowBlockCodeCacheEntry{
				Hits:      s.StateReadCacheStats.CodeStats.CacheHit,
				Misses:    s.StateReadCacheStats.CodeStats.CacheMiss,
				HitRate:   s.StateReadCacheStats.CodeStats.HitRate(),
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
