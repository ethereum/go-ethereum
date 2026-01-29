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
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	mapset "github.com/deckarep/golang-set/v2"
)

const (
	maxKnownTxsXDC        = 32768  // Maximum transactions hashes to keep in the known list
	maxKnownOrderTxs      = 32768  // Maximum order transactions hashes
	maxKnownLendingTxs    = 32768  // Maximum lending transactions hashes
	maxKnownBlocksXDC     = 1024   // Maximum block hashes to keep in the known list
	maxKnownVote          = 131072 // Maximum vote hashes
	maxKnownTimeout       = 131072 // Maximum timeout hashes
	maxKnownSyncInfo      = 131072 // Maximum sync info hashes
)

// XDCPeerInfo represents XDPoS-specific peer metadata
type XDCPeerInfo struct {
	Version    int      `json:"version"`
	Difficulty *big.Int `json:"difficulty"`
	Head       string   `json:"head"`
	Epoch      uint64   `json:"epoch,omitempty"`
	IsMaster   bool     `json:"isMaster,omitempty"`
}

// xdcPeer extends the base peer with XDPoS-specific functionality
type xdcPeer struct {
	id      string
	version int
	head    common.Hash
	td      *big.Int
	lock    sync.RWMutex

	// Known hashes for deduplication
	knownTxs        mapset.Set[common.Hash]
	knownBlocks     mapset.Set[common.Hash]
	knownOrderTxs   mapset.Set[common.Hash]
	knownLendingTxs mapset.Set[common.Hash]
	knownVotes      mapset.Set[common.Hash]
	knownTimeouts   mapset.Set[common.Hash]
	knownSyncInfos  mapset.Set[common.Hash]

	// Masternode tracking
	isMasternode bool
	epoch        uint64
}

// newXDCPeer creates a new XDPoS peer
func newXDCPeer(version int, id string) *xdcPeer {
	return &xdcPeer{
		id:              id,
		version:         version,
		td:              big.NewInt(0),
		knownTxs:        mapset.NewSet[common.Hash](),
		knownBlocks:     mapset.NewSet[common.Hash](),
		knownOrderTxs:   mapset.NewSet[common.Hash](),
		knownLendingTxs: mapset.NewSet[common.Hash](),
		knownVotes:      mapset.NewSet[common.Hash](),
		knownTimeouts:   mapset.NewSet[common.Hash](),
		knownSyncInfos:  mapset.NewSet[common.Hash](),
	}
}

// Info returns XDPoS-specific peer info
func (p *xdcPeer) Info() *XDCPeerInfo {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return &XDCPeerInfo{
		Version:    p.version,
		Difficulty: new(big.Int).Set(p.td),
		Head:       p.head.Hex(),
		Epoch:      p.epoch,
		IsMaster:   p.isMasternode,
	}
}

// Head retrieves the current head hash and total difficulty
func (p *xdcPeer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty
func (p *xdcPeer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// MarkBlock marks a block as known
func (p *xdcPeer) MarkBlock(hash common.Hash) {
	for p.knownBlocks.Cardinality() >= maxKnownBlocksXDC {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction marks a transaction as known
func (p *xdcPeer) MarkTransaction(hash common.Hash) {
	for p.knownTxs.Cardinality() >= maxKnownTxsXDC {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// MarkOrderTransaction marks an order transaction as known
func (p *xdcPeer) MarkOrderTransaction(hash common.Hash) {
	for p.knownOrderTxs.Cardinality() >= maxKnownOrderTxs {
		p.knownOrderTxs.Pop()
	}
	p.knownOrderTxs.Add(hash)
}

// MarkLendingTransaction marks a lending transaction as known
func (p *xdcPeer) MarkLendingTransaction(hash common.Hash) {
	for p.knownLendingTxs.Cardinality() >= maxKnownLendingTxs {
		p.knownLendingTxs.Pop()
	}
	p.knownLendingTxs.Add(hash)
}

// MarkVote marks a vote as known
func (p *xdcPeer) MarkVote(hash common.Hash) {
	for p.knownVotes.Cardinality() >= maxKnownVote {
		p.knownVotes.Pop()
	}
	p.knownVotes.Add(hash)
}

// MarkTimeout marks a timeout as known
func (p *xdcPeer) MarkTimeout(hash common.Hash) {
	for p.knownTimeouts.Cardinality() >= maxKnownTimeout {
		p.knownTimeouts.Pop()
	}
	p.knownTimeouts.Add(hash)
}

// MarkSyncInfo marks a sync info as known
func (p *xdcPeer) MarkSyncInfo(hash common.Hash) {
	for p.knownSyncInfos.Cardinality() >= maxKnownSyncInfo {
		p.knownSyncInfos.Pop()
	}
	p.knownSyncInfos.Add(hash)
}

// HasBlock checks if a block is known
func (p *xdcPeer) HasBlock(hash common.Hash) bool {
	return p.knownBlocks.Contains(hash)
}

// HasTransaction checks if a transaction is known
func (p *xdcPeer) HasTransaction(hash common.Hash) bool {
	return p.knownTxs.Contains(hash)
}

// HasOrderTransaction checks if an order transaction is known
func (p *xdcPeer) HasOrderTransaction(hash common.Hash) bool {
	return p.knownOrderTxs.Contains(hash)
}

// HasLendingTransaction checks if a lending transaction is known
func (p *xdcPeer) HasLendingTransaction(hash common.Hash) bool {
	return p.knownLendingTxs.Contains(hash)
}

// HasVote checks if a vote is known
func (p *xdcPeer) HasVote(hash common.Hash) bool {
	return p.knownVotes.Contains(hash)
}

// HasTimeout checks if a timeout is known
func (p *xdcPeer) HasTimeout(hash common.Hash) bool {
	return p.knownTimeouts.Contains(hash)
}

// HasSyncInfo checks if a sync info is known
func (p *xdcPeer) HasSyncInfo(hash common.Hash) bool {
	return p.knownSyncInfos.Contains(hash)
}

// SetMasternode sets whether this peer is a masternode
func (p *xdcPeer) SetMasternode(isMaster bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.isMasternode = isMaster
}

// IsMasternode returns whether this peer is a masternode
func (p *xdcPeer) IsMasternode() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.isMasternode
}

// SetEpoch sets the current epoch
func (p *xdcPeer) SetEpoch(epoch uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.epoch = epoch
}

// xdcPeerSet manages a set of XDPoS peers
type xdcPeerSet struct {
	peers  map[string]*xdcPeer
	lock   sync.RWMutex
	closed bool
}

// newXDCPeerSet creates a new XDPoS peer set
func newXDCPeerSet() *xdcPeerSet {
	return &xdcPeerSet{
		peers: make(map[string]*xdcPeer),
	}
}

// Register adds a peer to the set
func (ps *xdcPeerSet) Register(p *xdcPeer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	return nil
}

// Unregister removes a peer from the set
func (ps *xdcPeerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves a peer by ID
func (ps *xdcPeerSet) Peer(id string) *xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns the number of peers
func (ps *xdcPeerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock returns peers without a known block
func (ps *xdcPeerSet) PeersWithoutBlock(hash common.Hash) []*xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*xdcPeer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.HasBlock(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx returns peers without a known transaction
func (ps *xdcPeerSet) PeersWithoutTx(hash common.Hash) []*xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*xdcPeer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.HasTransaction(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutVote returns peers without a known vote
func (ps *xdcPeerSet) PeersWithoutVote(hash common.Hash) []*xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*xdcPeer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.HasVote(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTimeout returns peers without a known timeout
func (ps *xdcPeerSet) PeersWithoutTimeout(hash common.Hash) []*xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*xdcPeer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.HasTimeout(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutSyncInfo returns peers without a known sync info
func (ps *xdcPeerSet) PeersWithoutSyncInfo(hash common.Hash) []*xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*xdcPeer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.HasSyncInfo(hash) {
			list = append(list, p)
		}
	}
	return list
}

// MasternodePeers returns all masternode peers
func (ps *xdcPeerSet) MasternodePeers() []*xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*xdcPeer, 0)
	for _, p := range ps.peers {
		if p.IsMasternode() {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer returns the peer with highest total difficulty
func (ps *xdcPeerSet) BestPeer() *xdcPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *xdcPeer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// Close shuts down the peer set
func (ps *xdcPeerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.closed = true
}

// SendVote sends a vote message to a peer (placeholder for p2p integration)
func SendVote(rw p2p.MsgReadWriter, vote *types.Vote) error {
	return p2p.Send(rw, VoteMsgCode, vote)
}

// SendTimeout sends a timeout message to a peer
func SendTimeout(rw p2p.MsgReadWriter, timeout *types.Timeout) error {
	return p2p.Send(rw, TimeoutMsgCode, timeout)
}

// SendSyncInfo sends a sync info message to a peer
func SendSyncInfo(rw p2p.MsgReadWriter, syncInfo *types.SyncInfo) error {
	return p2p.Send(rw, SyncInfoMsgCode, syncInfo)
}

// SendOrderTransactions sends order transactions to a peer
func SendOrderTransactions(rw p2p.MsgReadWriter, txs types.OrderTransactions) error {
	return p2p.Send(rw, OrderTxMsgCode, txs)
}

// SendLendingTransactions sends lending transactions to a peer
func SendLendingTransactions(rw p2p.MsgReadWriter, txs types.LendingTransactions) error {
	return p2p.Send(rw, LendingTxMsgCode, txs)
}
