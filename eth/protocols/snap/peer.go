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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/tracker"
	"github.com/ethereum/go-ethereum/rlp"
)

// Peer is a collection of relevant information we have about a `snap` peer.
type Peer struct {
	id string // Unique ID for the peer, cached

	*p2p.Peer                   // The embedded P2P package peer
	rw        p2p.MsgReadWriter // Input/output streams for snap
	version   uint              // Protocol version negotiated
	tracker   *tracker.Tracker

	logger log.Logger // Contextual logger with the peer id injected
}

// NewPeer creates a wrapper for a network connection and negotiated  protocol
// version.
func NewPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	cap := p2p.Cap{Name: ProtocolName, Version: version}
	id := p.ID().String()
	return &Peer{
		id:      id,
		Peer:    p,
		rw:      rw,
		version: version,
		tracker: tracker.New(cap, id, 1*time.Minute),
		logger:  log.New("peer", id[:8]),
	}
}

// NewFakePeer creates a fake snap peer without a backing p2p peer, for testing purposes.
func NewFakePeer(version uint, id string, rw p2p.MsgReadWriter) *Peer {
	cap := p2p.Cap{Name: ProtocolName, Version: version}
	return &Peer{
		id:      id,
		rw:      rw,
		version: version,
		tracker: tracker.New(cap, id, 1*time.Minute),
		logger:  log.New("peer", id[:8]),
	}
}

// ID retrieves the peer's unique identifier.
func (p *Peer) ID() string {
	return p.id
}

// Version retrieves the peer's negotiated `snap` protocol version.
func (p *Peer) Version() uint {
	return p.version
}

// Log overrides the P2P logger with the higher level one containing only the id.
func (p *Peer) Log() log.Logger {
	return p.logger
}

// Close releases resources associated with the peer.
func (p *Peer) Close() {
	p.tracker.Stop()
}

// RequestAccountRange fetches a batch of accounts rooted in a specific account
// trie, starting with the origin.
func (p *Peer) RequestAccountRange(id uint64, root common.Hash, origin, limit common.Hash, bytes int) error {
	p.logger.Trace("Fetching range of accounts", "reqid", id, "root", root, "origin", origin, "limit", limit, "bytes", common.StorageSize(bytes))

	err := p.tracker.Track(tracker.Request{
		ReqCode:  GetAccountRangeMsg,
		RespCode: AccountRangeMsg,
		ID:       id,
		Size:     2 * bytes,
	})
	if err != nil {
		return err
	}
	return p2p.Send(p.rw, GetAccountRangeMsg, &GetAccountRangePacket{
		ID:     id,
		Root:   root,
		Origin: origin,
		Limit:  limit,
		Bytes:  uint64(bytes),
	})
}

// RequestStorageRanges fetches a batch of storage slots belonging to one or more
// accounts. If slots from only one account is requested, an origin marker may also
// be used to retrieve from there.
func (p *Peer) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes int) error {
	if len(accounts) == 1 && origin != nil {
		p.logger.Trace("Fetching range of large storage slots", "reqid", id, "root", root, "account", accounts[0], "origin", common.BytesToHash(origin), "limit", common.BytesToHash(limit), "bytes", common.StorageSize(bytes))
	} else {
		p.logger.Trace("Fetching ranges of small storage slots", "reqid", id, "root", root, "accounts", len(accounts), "first", accounts[0], "bytes", common.StorageSize(bytes))
	}

	p.tracker.Track(tracker.Request{
		ReqCode:  GetStorageRangesMsg,
		RespCode: StorageRangesMsg,
		ID:       id,
		Size:     2 * bytes,
	})
	return p2p.Send(p.rw, GetStorageRangesMsg, &GetStorageRangesPacket{
		ID:       id,
		Root:     root,
		Accounts: accounts,
		Origin:   origin,
		Limit:    limit,
		Bytes:    uint64(bytes),
	})
}

// RequestByteCodes fetches a batch of bytecodes by hash.
func (p *Peer) RequestByteCodes(id uint64, hashes []common.Hash, bytes int) error {
	p.logger.Trace("Fetching set of byte codes", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))

	err := p.tracker.Track(tracker.Request{
		ReqCode:  GetByteCodesMsg,
		RespCode: ByteCodesMsg,
		ID:       id,
		Size:     len(hashes), // ByteCodes is limited by the length of the hash list.
	})
	if err != nil {
		return err
	}
	return p2p.Send(p.rw, GetByteCodesMsg, &GetByteCodesPacket{
		ID:     id,
		Hashes: hashes,
		Bytes:  uint64(bytes),
	})
}

// RequestTrieNodes fetches a batch of account or storage trie nodes rooted in
// a specific state trie. The `count` is the total count of paths being requested.
func (p *Peer) RequestTrieNodes(id uint64, root common.Hash, count int, paths []TrieNodePathSet, bytes int) error {
	p.logger.Trace("Fetching set of trie nodes", "reqid", id, "root", root, "pathsets", len(paths), "bytes", common.StorageSize(bytes))

	err := p.tracker.Track(tracker.Request{
		ReqCode:  GetTrieNodesMsg,
		RespCode: TrieNodesMsg,
		ID:       id,
		Size:     count, // TrieNodes is limited by number of items.
	})
	if err != nil {
		return err
	}
	encPaths, _ := rlp.EncodeToRawList(paths)
	return p2p.Send(p.rw, GetTrieNodesMsg, &GetTrieNodesPacket{
		ID:    id,
		Root:  root,
		Paths: encPaths,
		Bytes: uint64(bytes),
	})
}
