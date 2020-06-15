// Copyright 2020 The go-ethereum Authors
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

package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// snapHandler implements the snap.Backend interface to handle the various network
// packets that are sent as replies or broadcasts.
type snapHandler handler

func (h *snapHandler) Chain() *core.BlockChain { return h.chain }

// RunPeer is invoked when a peer joins on the `snap` protocol.
func (h *snapHandler) RunPeer(peer *snap.Peer, hand snap.Handler) error {
	return (*handler)(h).runSnapPeer(peer, hand)
}

// PeerInfo retrieves all known `snap` information about a peer.
func (h *snapHandler) PeerInfo(id enode.ID) interface{} {
	if p := h.peers.snapPeer(id.String()); p != nil {
		return p.info()
	}
	return nil
}

// OnAccounts is invoked from a peer's message handler when it transmits a range
// of accounts for the local node to process.
func (h *snapHandler) OnAccounts(peer *snap.Peer, id uint64, keys []common.Hash, accounts [][]byte, proof [][]byte) error {
	return h.downloader.DeliverSnapshotAccounts(peer, id, keys, accounts, proof)
}

// OnStorage is invoked from a peer's message handler when it transmits ranges
// of storage slots for the local node to process.
func (h *snapHandler) OnStorage(peer *snap.Peer, id uint64, keys [][]common.Hash, slots [][][]byte, proof [][]byte) error {
	return h.downloader.DeliverSnapshotStorage(peer, id, keys, slots, proof)
}

// OnByteCodes is invoked from a peer's message handler when it transmits a batch
// of byte codes for the local node to process.
func (h *snapHandler) OnByteCodes(peer *snap.Peer, id uint64, codes [][]byte) error {
	return h.downloader.DeliverSnapshotByteCodes(peer, id, codes)
}

// OnTrieNodes is invoked from a peer's message handler when it transmits a batch
// of trie nodes for the local node to process.
func (h *snapHandler) OnTrieNodes(peer *snap.Peer, id uint64, nodes [][]byte) error {
	return h.downloader.DeliverSnapshotTrieNodes(peer, id, nodes)
}
