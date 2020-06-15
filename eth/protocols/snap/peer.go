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

package snap

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

// Peer is a collection of relevant information we have about a `snap` peer.
type Peer struct {
	id string // Unique ID for the peer, cached

	*p2p.Peer                   // The embedded P2P package peer
	rw        p2p.MsgReadWriter // Input/output streams for snap
	version   uint              // Protocol version negotiated

	logger log.Logger // Contextual logger with the peer id injected
}

// newPeer create a wrapper for a network connection and negotiated  protocol
// version.
func newPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	id := p.ID().String()
	return &Peer{
		id:      id,
		Peer:    p,
		rw:      rw,
		version: version,
		logger:  log.New("peer", id[:8]),
	}
}

// ID retrieves the peer's unique identifier.
func (p *Peer) ID() string {
	return p.id
}

// Version retrieves the peer's negoatiated `snap` protocol version.
func (p *Peer) Version() uint {
	return p.version
}

// RequestAccountRange fetches a batch of accounts rooted in a specific account
// trie, starting with the origin.
func (p *Peer) RequestAccountRange(id uint64, root common.Hash, origin common.Hash, bytes uint64) error {
	p.logger.Trace("Fetching range of accounts", "reqid", id, "root", root, "origin", origin, "bytes", common.StorageSize(bytes))
	return p2p.Send(p.rw, getAccountRangeMsg, &getAccountRangeData{
		ID:     id,
		Root:   root,
		Origin: origin,
		Bytes:  bytes,
	})
}

// RequestStorageRange fetches a batch of accounts rooted in a specific account
// trie, starting with the origin.
func (p *Peer) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin []byte, bytes uint64) error {
	p.logger.Trace("Fetching ranges of storage slots", "reqid", id, "root", root, "accounts", len(accounts), "origin", origin, "bytes", common.StorageSize(bytes))
	return p2p.Send(p.rw, getStorageRangesMsg, &getStorageRangesData{
		ID:       id,
		Root:     root,
		Accounts: accounts,
		Origin:   origin,
		Bytes:    bytes,
	})
}

// RequestByteCodes fetches a batch of accounts rooted in a specific account
// trie, starting with the origin.
func (p *Peer) RequestByteCodes(id uint64, hashes []common.Hash, bytes uint64) error {
	p.logger.Trace("Fetching set of byte codes", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	return p2p.Send(p.rw, getByteCodesMsg, &getByteCodesData{
		ID:     id,
		Hashes: hashes,
		Bytes:  bytes,
	})
}
