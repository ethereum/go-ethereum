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

package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
)

// ethPeerInfo represents a short summary of the `eth` sub-protocol metadata known
// about a connected peer.
type ethPeerInfo struct {
	Version uint `json:"version"` // Ethereum protocol version negotiated
	*peerBlockRange
}

type peerBlockRange struct {
	Earliest   uint64      `json:"earliestBlock"`
	Latest     uint64      `json:"latestBlock"`
	LatestHash common.Hash `json:"latestBlockHash"`
}

// ethPeer is a wrapper around eth.Peer to maintain a few extra metadata.
type ethPeer struct {
	*eth.Peer
	snapExt *snapPeer // Satellite `snap` connection
}

// info gathers and returns some `eth` protocol metadata known about a peer.
func (p *ethPeer) info() *ethPeerInfo {
	info := &ethPeerInfo{Version: p.Version()}
	if br := p.BlockRange(); br != nil {
		info.peerBlockRange = &peerBlockRange{
			Earliest:   br.EarliestBlock,
			Latest:     br.LatestBlock,
			LatestHash: br.LatestBlockHash,
		}
	}
	return info
}

// snapPeerInfo represents a short summary of the `snap` sub-protocol metadata known
// about a connected peer.
type snapPeerInfo struct {
	Version uint `json:"version"` // Snapshot protocol version negotiated
}

// snapPeer is a wrapper around snap.Peer to maintain a few extra metadata.
type snapPeer struct {
	*snap.Peer
}

// info gathers and returns some `snap` protocol metadata known about a peer.
func (p *snapPeer) info() *snapPeerInfo {
	return &snapPeerInfo{
		Version: p.Version(),
	}
}
