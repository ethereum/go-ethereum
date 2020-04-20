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

// validateCheckpoint verifies the advertised checkpoint by peer is valid or not.
//
// Each network has several hard-coded checkpoint signer addresses. Only the
// checkpoint issued by the specified signer is considered valid.
//
// In addition to the checkpoint registered in the registrar contract, there are
// several legacy hardcoded checkpoints in our codebase. These checkpoints are
// also considered as valid.
func (h *clientHandler) validateCheckpoint(peer *serverPeer) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Fetch the block header corresponding to the checkpoint registration.
	cp := peer.checkpoint
	header, err := light.GetUntrustedHeaderByNumber(ctx, h.backend.odr, peer.checkpointNumber, peer.id)
	if err != nil {
		return err
	}
	// Fetch block logs associated with the block header.
	logs, err := light.GetUntrustedBlockLogs(ctx, h.backend.odr, header)
	if err != nil {
		return err
	}
	events := h.backend.oracle.Contract().LookupCheckpointEvents(logs, cp.SectionIndex, cp.Hash())
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
	valid, signers := h.backend.oracle.VerifySigners(index, hash, signatures)
	if !valid {
		return errInvalidCheckpoint
	}
	log.Warn("Verified advertised checkpoint", "peer", peer.id, "signers", len(signers))
	return nil
}

// synchronise tries to sync up our local chain with a remote peer.
func (h *clientHandler) synchronise(peer *serverPeer) {
	// Short circuit if the peer is nil.
	if peer == nil {
		return
	}
	// Make sure the peer's TD is higher than our own.
	latest := h.backend.blockchain.CurrentHeader()
	currentTd := rawdb.ReadTd(h.backend.chainDb, latest.Hash(), latest.Number.Uint64())
	if currentTd != nil && peer.Td().Cmp(currentTd) < 0 {
		return
	}
	// Recap the checkpoint.
	//
	// The light client may be connected to several different versions of the server.
	// (1) Old version server which can not provide stable checkpoint in the handshake packet.
	//     => Use hardcoded checkpoint or empty checkpoint
	// (2) New version server but simple checkpoint syncing is not enabled(e.g. mainnet, new testnet or private network)
	//     => Use hardcoded checkpoint or empty checkpoint
	// (3) New version server but the provided stable checkpoint is even lower than the hardcoded one.
	//     => Use hardcoded checkpoint
	// (4) New version server with valid and higher stable checkpoint
	//     => Use provided checkpoint
	var checkpoint = &peer.checkpoint
	var hardcoded bool
	if h.checkpoint != nil && h.checkpoint.SectionIndex >= peer.checkpoint.SectionIndex {
		checkpoint = h.checkpoint // Use the hardcoded one.
		hardcoded = true
	}
	// Determine whether we should run checkpoint syncing or normal light syncing.
	//
	// Here has four situations that we will disable the checkpoint syncing:
	//
	// 1. The checkpoint is empty
	// 2. The latest head block of the local chain is above the checkpoint.
	// 3. The checkpoint is hardcoded(recap with local hardcoded checkpoint)
	// 4. For some networks the checkpoint syncing is not activated.
	mode := checkpointSync
	switch {
	case checkpoint.Empty():
		mode = lightSync
		log.Debug("Disable checkpoint syncing", "reason", "empty checkpoint")
	case latest.Number.Uint64() >= (checkpoint.SectionIndex+1)*h.backend.iConfig.ChtSize-1:
		mode = lightSync
		log.Debug("Disable checkpoint syncing", "reason", "local chain beyond the checkpoint")
	case hardcoded:
		mode = legacyCheckpointSync
		log.Debug("Disable checkpoint syncing", "reason", "checkpoint is hardcoded")
	case h.backend.oracle == nil || !h.backend.oracle.IsRunning():
		if h.checkpoint == nil {
			mode = lightSync // Downgrade to light sync unfortunately.
		} else {
			checkpoint = h.checkpoint
			mode = legacyCheckpointSync
		}
		log.Debug("Disable checkpoint syncing", "reason", "checkpoint syncing is not activated")
	}
	// Notify testing framework if syncing has completed(for testing purpose).
	defer func() {
		if h.syncDone != nil {
			h.syncDone()
		}
	}()
	start := time.Now()
	if mode == checkpointSync || mode == legacyCheckpointSync {
		// Validate the advertised checkpoint
		if mode == checkpointSync {
			if err := h.validateCheckpoint(peer); err != nil {
				log.Debug("Failed to validate checkpoint", "reason", err)
				h.removePeer(peer.id)
				return
			}
			h.backend.blockchain.AddTrustedCheckpoint(checkpoint)
		}
		log.Debug("Checkpoint syncing start", "peer", peer.id, "checkpoint", checkpoint.SectionIndex)

		// Fetch the start point block header.
		//
		// For the ethash consensus engine, the start header is the block header
		// of the checkpoint.
		//
		// For the clique consensus engine, the start header is the block header
		// of the latest epoch covered by checkpoint.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if !checkpoint.Empty() && !h.backend.blockchain.SyncCheckpoint(ctx, checkpoint) {
			log.Debug("Sync checkpoint failed")
			h.removePeer(peer.id)
			return
		}
	}
	// Fetch the remaining block headers based on the current chain header.
	if err := h.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync); err != nil {
		log.Debug("Synchronise failed", "reason", err)
		return
	}
	log.Debug("Synchronise finished", "elapsed", common.PrettyDuration(time.Since(start)))
}
