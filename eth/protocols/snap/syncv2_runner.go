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

package snap

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// The snap/2 jobs are the persistence half of a response: the runloop keeps
// the bookkeeping (task cursors, pending counters, completion flags) and hands
// the flat-state writes over to the worker pool. Unlike snap/1 there is no
// inline trie generation, the tries are rebuilt from the flat state once the
// download completes, so every job is a plain batch of flat writes with no
// ordering constraints between jobs: the keys of distinct jobs never overlap
// (each account range and storage chunk is delivered once per cycle), and a
// re-delivered overlap on resume rewrites identical values.
//
// The only invariant the workers must uphold is that the journal never runs
// ahead of the disk, which saveSyncStatus enforces with a runner barrier.

// accountJobV2 is the execution part of an account-task forward: the flat
// account writes of the consecutive complete prefix.
type accountJobV2 struct {
	hashes   []common.Hash
	accounts []*types.StateAccount
}

// executeAccountJob runs one account-forward job on a worker thread.
func (s *syncerV2) executeAccountJob(job *accountJobV2) {
	start := time.Now()

	batch := ethdb.HookedBatch{
		Batch: s.db.NewBatch(),
		OnPut: func(key []byte, value []byte) {
			s.accountBytes.Add(int64(len(key) + len(value)))
		},
	}
	for i, hash := range job.hashes {
		rawdb.WriteAccountSnapshot(batch, hash, types.SlimAccountRLP(*job.accounts[i]))
	}
	commitStart := time.Now()
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist accounts", "err", err)
	}
	commits := s.profile.observeCommit(profAccount, commitStart)
	s.profile.exec[profAccount].observe(time.Since(start) - commits)
}

// storageJobV2 is the execution part of a storage delivery: the flat storage
// writes of every account in the response, complete or chunked alike.
type storageJobV2 struct {
	accounts []common.Hash
	hashes   [][]common.Hash
	slots    [][][]byte
}

// executeStorageJob runs one storage job on a worker thread.
func (s *syncerV2) executeStorageJob(job *storageJobV2) {
	start := time.Now()

	batch := ethdb.HookedBatch{
		Batch: s.db.NewBatch(),
		OnPut: func(key []byte, value []byte) {
			s.storageBytes.Add(int64(len(key) + len(value)))
		},
	}
	// Persist the received storage segments. The flat state may be outdated
	// during the sync, but it will be fixed by the BAL catch-up.
	for i, account := range job.accounts {
		for j := 0; j < len(job.hashes[i]); j++ {
			rawdb.WriteStorageSnapshot(batch, account, job.hashes[i][j], job.slots[i][j])
		}
	}
	commitStart := time.Now()
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist storage slots", "err", err)
	}
	commits := s.profile.observeCommit(profStorage, commitStart)
	s.profile.exec[profStorage].observe(time.Since(start) - commits)
}

// bytecodeJobV2 is the execution part of a bytecode delivery: the code writes
// are content addressed and carry no ordering constraints.
type bytecodeJobV2 struct {
	hashes []common.Hash
	codes  [][]byte
}

// executeBytecodeJob runs one bytecode-persistence job on a worker thread.
func (s *syncerV2) executeBytecodeJob(job *bytecodeJobV2) {
	var (
		start = time.Now()
		batch = s.db.NewBatch()
	)
	for i, hash := range job.hashes {
		rawdb.WriteCode(batch, hash, job.codes[i])
	}
	s.bytecodeBytes.Add(int64(batch.ValueSize()))

	commitStart := time.Now()
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist bytecodes", "err", err)
	}
	commits := s.profile.observeCommit(profBytecode, commitStart)
	s.profile.exec[profBytecode].observe(time.Since(start) - commits)
}
