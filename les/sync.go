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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/les/downloader"
	"github.com/ethereum/go-ethereum/log"
)

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
	// Notify testing framework if syncing has completed (for testing purpose).
	defer func() {
		if h.syncEnd != nil {
			h.syncEnd(h.backend.blockchain.CurrentHeader())
		}
	}()
	start := time.Now()
	if h.syncStart != nil {
		h.syncStart(h.backend.blockchain.CurrentHeader())
	}
	// Fetch the remaining block headers based on the current chain header.
	if err := h.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync); err != nil {
		log.Debug("Synchronise failed", "reason", err)
		return
	}
	log.Debug("Synchronise finished", "elapsed", common.PrettyDuration(time.Since(start)))
}
