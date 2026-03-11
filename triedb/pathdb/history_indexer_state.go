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

package pathdb

import (
	"bytes"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// state represents the syncing status of the node.
type state int

const (
	// stateSynced indicates that the local chain head is sufficiently close to the
	// network chain head, and the majority of the data has been fully synchronized.
	stateSynced state = iota

	// stateSyncing indicates that the sync process is still in progress. Local node
	// is actively catching up with the network chain head.
	stateSyncing

	// stateStalled indicates that sync progress has stopped for a while
	// with no progress. This may be caused by network instability (e.g., no peers),
	// manual operation such as syncing the local chain to a specific block.
	stateStalled
)

const (
	// syncStateTimeWindow defines the maximum allowed lag behind the network
	// chain head.
	//
	// If the local chain head falls within this threshold, the node is considered
	// close to the tip and will be marked as stateSynced.
	syncStateTimeWindow = 6 * time.Hour

	// syncStalledTimeout defines the maximum duration during which no sync
	// progress is observed. If this timeout is exceeded, the node's status
	// will be considered stalled.
	syncStalledTimeout = 5 * time.Minute
)

type initerState struct {
	state     state
	stateLock sync.Mutex
	disk      ethdb.Database
	term      chan struct{}
}

func newIniterState(disk ethdb.Database) *initerState {
	s := &initerState{
		disk: disk,
		term: make(chan struct{}),
	}
	go s.update()
	return s
}

func (s *initerState) get() state {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	return s.state
}

func (s *initerState) is(state state) bool {
	return s.get() == state
}

func (s *initerState) set(state state) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	s.state = state
}

func (s *initerState) update() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	headBlock := s.readLastBlock()
	if headBlock != nil && time.Since(time.Unix(int64(headBlock.Time), 0)) < syncStateTimeWindow {
		s.set(stateSynced)
		log.Info("Marked indexing initer as synced")
	} else {
		s.set(stateSyncing)
	}

	var (
		hhash        = rawdb.ReadHeadHeaderHash(s.disk)
		fhash        = rawdb.ReadHeadFastBlockHash(s.disk)
		bhash        = rawdb.ReadHeadBlockHash(s.disk)
		skeleton     = rawdb.ReadSkeletonSyncStatus(s.disk)
		lastProgress = time.Now()
	)
	for {
		select {
		case <-ticker.C:
			state := s.get()
			if state == stateSynced || state == stateStalled {
				continue
			}
			headBlock := s.readLastBlock()
			if headBlock == nil {
				continue
			}
			// State machine: stateSyncing => stateSynced
			if time.Since(time.Unix(int64(headBlock.Time), 0)) < syncStateTimeWindow {
				s.set(stateSynced)
				log.Info("Marked indexing initer as synced")
				continue
			}
			// State machine: stateSyncing => stateStalled
			newhhash := rawdb.ReadHeadHeaderHash(s.disk)
			newfhash := rawdb.ReadHeadFastBlockHash(s.disk)
			newbhash := rawdb.ReadHeadBlockHash(s.disk)
			newskeleton := rawdb.ReadSkeletonSyncStatus(s.disk)
			hasProgress := newhhash.Cmp(hhash) != 0 || newfhash.Cmp(fhash) != 0 || newbhash.Cmp(bhash) != 0 || !bytes.Equal(newskeleton, skeleton)

			if !hasProgress && time.Since(lastProgress) > syncStalledTimeout {
				s.set(stateStalled)
				log.Info("Marked indexing initer as stalled")
				continue
			}
			if hasProgress {
				hhash = newhhash
				fhash = newfhash
				bhash = newbhash
				skeleton = newskeleton
				lastProgress = time.Now()
			}

		case <-s.term:
			return
		}
	}
}

func (s *initerState) close() {
	select {
	case <-s.term:
	default:
		close(s.term)
	}
	return
}

// readLastBlock returns the local chain head.
func (s *initerState) readLastBlock() *types.Header {
	hash := rawdb.ReadHeadBlockHash(s.disk)
	if hash == (common.Hash{}) {
		return nil
	}
	number, exists := rawdb.ReadHeaderNumber(s.disk, hash)
	if !exists {
		return nil
	}
	return rawdb.ReadHeader(s.disk, hash, number)
}
