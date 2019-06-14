// Copyright 2016 The go-ethereum Authors
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
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var errInvalidCheckpoint = errors.New("invalid advertised checkpoint")

const (
	// lightSync starts syncing from the current highest block.
	// If the chain is empty, syncing the entire header chain.
	lightSync = iota

	// legacyCheckpointSync starts syncing from a hardcoded checkpoint.
	legacyCheckpointSync

	// checkpointSync starts syncing from a checkpoint signed by trusted
	// signer or hardcoded checkpoint for compatibility.
	checkpointSync
)

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pm *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	//pm.fetcher.Start()
	//defer pm.fetcher.Stop()
	defer pm.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	//forceSync := time.Tick(forceSyncCycle)
	for {
		select {
		case <-pm.newPeerCh:
			/*			// Make sure we have peers to select from, then sync
						if pm.peers.Len() < minDesiredPeerCount {
							break
						}
						go pm.synchronise(pm.peers.BestPeer())
			*/
		/*case <-forceSync:
		// Force a sync even if not enough peers are present
		go pm.synchronise(pm.peers.BestPeer())
		*/
		case <-pm.noMorePeers:
			return
		}
	}
}

// validateCheckpoint verifies the advertised checkpoint by peer is valid or not.
//
// Each network has several hard-coded checkpoint signer addresses. Only the
// checkpoint issued by the specified signer is considered valid.
//
// In addition to the checkpoint registered in the registrar contract, there are
// several legacy hardcoded checkpoints in our codebase. These checkpoints are
// also considered as valid.
func (pm *ProtocolManager) validateCheckpoint(peer *peer) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Fetch the block header corresponding to the checkpoint registration.
	cp := peer.advertisedCheckpoint
	header, err := light.GetUntrustedHeaderByNumber(ctx, pm.odr, peer.registeredHeight, peer.id)
	if err != nil {
		return err
	}
	// Fetch block logs associated with the block header.
	logs, err := light.GetUntrustedBlockLogs(ctx, pm.odr, header)
	if err != nil {
		return err
	}
	events := pm.reg.contract.LookupCheckpointEvents(logs, cp.SectionIndex, cp.Hash())
	if len(events) == 0 {
		return errInvalidCheckpoint
	}
	var (
		index      = events[0].Index
		hash       = events[0].CheckpointHash
		signatures [][]byte
	)
	for _, event := range events {
		signatures = append(signatures, append(event.R[:], append(event.S[:], event.V)...))
	}
	valid, signers := pm.reg.verifySigners(index, hash, signatures)
	if !valid {
		return errInvalidCheckpoint
	}
	log.Warn("Verifed advertised checkpoint", "peer", peer.id, "signers", len(signers))
	return nil
}

// synchronise tries to sync up our local chain with a remote peer.
func (pm *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if the peer is nil.
	if peer == nil {
		return
	}
	// Make sure the peer's TD is higher than our own.
	latest := pm.blockchain.CurrentHeader()
	currentTd := rawdb.ReadTd(pm.chainDb, latest.Hash(), latest.Number.Uint64())
	if currentTd != nil && peer.headBlockInfo().Td.Cmp(currentTd) < 0 {
		return
	}
	// Determine whether we should run checkpoint syncing or normal light syncing.
	//
	// Here has four situations that we will disable the checkpoint syncing:
	//
	// 1. The checkpoint is empty
	// 2. The latest head block of the local chain is above the checkpoint.
	// 3. The checkpoint is hardcoded(recap with local hardcoded checkpoint)
	// 4. For some networks the checkpoint syncing is not activated.
	cp := &peer.advertisedCheckpoint
	mode := checkpointSync
	switch {
	case cp.Empty():
		mode = lightSync
		log.Debug("Disable checkpoint syncing", "reason", "empty checkpoint")
	case latest.Number.Uint64() >= (cp.SectionIndex+1)*pm.iConfig.ChtSize-1:
		mode = lightSync
		log.Debug("Disable checkpoint syncing", "reason", "local chain beyond the checkpoint")
	case peer.hardcodedCheckpoint:
		mode = legacyCheckpointSync
		log.Debug("Disable checkpoint syncing", "reason", "checkpoint is hardcoded")
	case pm.reg == nil || !pm.reg.isRunning():
		mode = legacyCheckpointSync
		log.Debug("Disable checkpoint syncing", "reason", "checkpoint syncing is not activated")
	}
	// Notify testing framework if syncing has completed(for testing purpose).
	defer func() {
		if pm.reg != nil && pm.reg.syncDoneHook != nil {
			pm.reg.syncDoneHook()
		}
	}()
	start := time.Now()
	if mode == checkpointSync || mode == legacyCheckpointSync {
		// Validate the advertised checkpoint
		if mode == legacyCheckpointSync {
			cp = params.TrustedCheckpoints[pm.blockchain.Genesis().Hash()]
		} else if mode == checkpointSync {
			if err := pm.validateCheckpoint(peer); err != nil {
				log.Debug("Failed to validate checkpoint", "reason", err)
				pm.removePeer(peer.id)
				return
			}
			pm.blockchain.(*light.LightChain).AddTrustedCheckpoint(cp)
		}
		log.Debug("Checkpoint syncing start", "peer", peer.id, "checkpoint", cp.SectionIndex)

		// Fetch the start point block header.
		//
		// For the ethash consensus engine, the start header is the block header
		// of the checkpoint.
		//
		// For the clique consensus engine, the start header is the block header
		// of the latest epoch covered by checkpoint.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if !cp.Empty() && !pm.blockchain.(*light.LightChain).SyncCheckpoint(ctx, cp) {
			log.Debug("Sync checkpoint failed")
			pm.removePeer(peer.id)
			return
		}
	}
	// Fetch the remaining block headers based on the current chain header.
	if err := pm.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync); err != nil {
		log.Debug("Synchronise failed", "reason", err)
		return
	}
	log.Debug("Synchronise finished", "elapsed", common.PrettyDuration(time.Since(start)))
}
