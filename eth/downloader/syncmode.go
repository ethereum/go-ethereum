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

package downloader

import (
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// syncModer is responsible for managing the downloader's sync mode. It takes the
// user's preference at startup and then determines the appropriate sync mode
// based on the current chain status.
type syncModer struct {
	mode  ethconfig.SyncMode
	chain BlockChain
	disk  ethdb.KeyValueReader
	lock  sync.Mutex
}

func newSyncModer(mode ethconfig.SyncMode, chain BlockChain, disk ethdb.KeyValueReader) *syncModer {
	if mode == ethconfig.FullSync {
		// The database seems empty as the current block is the genesis. Yet the snap
		// block is ahead, so snap sync was enabled for this node at a certain point.
		// The scenarios where this can happen is
		// * if the user manually (or via a bad block) rolled back a snap sync node
		//   below the sync point.
		// * the last snap sync is not finished while user specifies a full sync this
		//   time. But we don't have any recent state for full sync.
		// In these cases however it's safe to reenable snap sync.
		fullBlock, snapBlock := chain.CurrentBlock(), chain.CurrentSnapBlock()
		if fullBlock.Number.Uint64() == 0 && snapBlock.Number.Uint64() > 0 {
			mode = ethconfig.SnapSync
			log.Warn("Switching from full-sync to snap-sync", "reason", "snap-sync incomplete")
		} else if !chain.HasState(fullBlock.Root) {
			mode = ethconfig.SnapSync
			log.Warn("Switching from full-sync to snap-sync", "reason", "head state missing")
		} else {
			// Grant the full sync mode
			log.Info("Enabled full-sync", "head", fullBlock.Number, "hash", fullBlock.Hash())
		}
	} else {
		head := chain.CurrentBlock()
		if head.Number.Uint64() > 0 && chain.HasState(head.Root) {
			mode = ethconfig.FullSync
			log.Info("Switching from snap-sync to full-sync", "reason", "snap-sync complete")
		} else {
			// If snap sync was requested and our database is empty, grant it
			log.Info("Enabled snap-sync", "head", head.Number, "hash", head.Hash())
		}
	}
	return &syncModer{
		mode:  mode,
		chain: chain,
		disk:  disk,
	}
}

// get retrieves the current sync mode, either explicitly set, or derived
// from the chain status.
func (m *syncModer) get(report bool) ethconfig.SyncMode {
	m.lock.Lock()
	defer m.lock.Unlock()

	// If we're in snap sync mode, return that directly
	if m.mode == ethconfig.SnapSync {
		return ethconfig.SnapSync
	}
	logger := log.Debug
	if report {
		logger = log.Info
	}
	// We are probably in full sync, but we might have rewound to before the
	// snap sync pivot, check if we should re-enable snap sync.
	head := m.chain.CurrentBlock()
	if pivot := rawdb.ReadLastPivotNumber(m.disk); pivot != nil {
		if head.Number.Uint64() < *pivot {
			logger("Reenabled snap-sync as chain is lagging behind the pivot", "head", head.Number, "pivot", pivot)
			return ethconfig.SnapSync
		}
	}
	// We are in a full sync, but the associated head state is missing. To complete
	// the head state, forcefully rerun the snap sync. Note it doesn't mean the
	// persistent state is corrupted, just mismatch with the head block.
	if !m.chain.HasState(head.Root) {
		logger("Reenabled snap-sync as chain is stateless")
		return ethconfig.SnapSync
	}
	// Nope, we're really full syncing
	return ethconfig.FullSync
}

// disableSnap disables the snap sync mode, usually it's called after a successful snap sync.
func (m *syncModer) disableSnap() {
	m.lock.Lock()
	m.mode = ethconfig.FullSync
	m.lock.Unlock()
}
