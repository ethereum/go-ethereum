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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package xps

import (
	"math/big"

	"github.com/xpaymentsorg/go-xpayments/xps/protocols/snap"
	"github.com/xpaymentsorg/go-xpayments/xps/protocols/xps"
	// "github.com/ethereum/go-ethereum/eth/protocols/eth"
	// "github.com/ethereum/go-ethereum/eth/protocols/snap"
)

// xpsPeerInfo represents a short summary of the `xps` sub-protocol metadata known
// about a connected peer.
type xpsPeerInfo struct {
	Version    uint     `json:"version"`    // xPayments protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // Hex hash of the peer's best owned block
}

// xpsPeer is a wrapper around xps.Peer to maintain a few extra metadata.
type xpsPeer struct {
	*xps.Peer
	snapExt  *snapPeer     // Satellite `snap` connection
	snapWait chan struct{} // Notification channel for snap connections
}

// info gathers and returns some `xps` protocol metadata known about a peer.
func (p *xpsPeer) info() *xpsPeerInfo {
	hash, td := p.Head()

	return &xpsPeerInfo{
		Version:    p.Version(),
		Difficulty: td,
		Head:       hash.Hex(),
	}
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
