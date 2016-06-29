// Copyright 2015 The go-ethereum Authors
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

package les

import (
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/downloader"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing
)

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pm *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	//pm.fetcher.Start()
	//defer pm.fetcher.Stop()
	defer pm.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.Tick(forceSyncCycle)
	for {
		select {
		case <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			if pm.peers.Len() < minDesiredPeerCount {
				break
			}
			go pm.synchronise(pm.peers.BestPeer())

		case <-forceSync:
			// Force a sync even if not enough peers are present
			go pm.synchronise(pm.peers.BestPeer())

		case <-pm.noMorePeers:
			return
		}
	}
}

func (pm *ProtocolManager) needToSync(peerHead blockInfo) bool {
	head := pm.blockchain.CurrentHeader()
	currentTd := core.GetTd(pm.chainDb, head.Hash(), head.Number.Uint64())
	return currentTd != nil && peerHead.Td.Cmp(currentTd) > 0
}

// synchronise tries to sync up our local block chain with a remote peer.
func (pm *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}
	
	// Make sure the peer's TD is higher than our own.
	if !pm.needToSync(peer.headBlockInfo()) {
		return
	}

	pm.waitSyncLock()
	pm.syncWithLockAcquired(peer)
}

func (pm *ProtocolManager) waitSyncLock() {
	for {
		chn := pm.getSyncLock(true)
		if chn == nil {
			break
		}
		<-chn
	}
}

// getSyncLock either acquires the sync lock and returns nil or returns a channel
// which is closed when the lock is free again
func (pm *ProtocolManager) getSyncLock(acquire bool) chan struct{} {
	pm.syncMu.Lock()
	defer pm.syncMu.Unlock()

	if pm.syncing {
		if pm.syncDone == nil {
			pm.syncDone = make(chan struct{})
		}
		return pm.syncDone
	} else {
		pm.syncing = acquire
		return nil
	}	
}

func (pm *ProtocolManager) releaseSyncLock() {
	pm.syncMu.Lock()
	pm.syncing = false
	if pm.syncDone != nil {
		close(pm.syncDone)
		pm.syncDone = nil
	}
	pm.syncMu.Unlock()
}

func (pm *ProtocolManager) syncWithLockAcquired(peer *peer) {
	pm.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync)
	pm.releaseSyncLock()
}
