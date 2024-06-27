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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/forkid"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	mapset "github.com/deckarep/golang-set"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxKnownTxs        = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownOrderTxs   = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownLendingTxs = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks     = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	maxKnownVote       = 1024  // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownTimeout    = 1024  // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownSyncInfo   = 1024  // Maximum transactions hashes to keep in the known list (prevent DOS)
	// maxQueuedTxs is the maximum number of transactions to queue up before dropping
	// older broadcasts.
	maxQueuedTxs = 4096
	// maxQueuedTxAnns is the maximum number of transaction announcements to queue up
	// before dropping older announcements.
	maxQueuedTxAnns = 4096
	// maxQueuedBlocks is the maximum number of block propagations to queue up before
	// dropping broadcasts. There's not much point in queueing stale blocks, so a few
	// that might cover uncles should be enough.
	maxQueuedBlocks = 4
	// maxQueuedBlockAnns is the maximum number of block announcements to queue up before
	// dropping broadcasts. Similarly to block propagations, there's no point to queue
	// above some healthy uncle limit, so use that.
	maxQueuedBlockAnns = 4

	handshakeTimeout = 5 * time.Second
)

// max is a helper function which returns the larger of the two given integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// propEvent is a block propagation, waiting for its turn in the broadcast queue.
type propEvent struct {
	block *types.Block
	td    *big.Int
}

// PeerInfo represents a short summary of the Ethereum sub-protocol metadata known
// about a connected peer.
type PeerInfo struct {
	Version    int      `json:"version"`    // Ethereum protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	id string

	*p2p.Peer
	rw     p2p.MsgReadWriter
	pairRw p2p.MsgReadWriter

	version  int         // Protocol version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	knownBlocks     mapset.Set // Set of block hashes known to be known by this peer
	knownTxs        mapset.Set // Set of transaction hashes known to be known by this peer
	knownOrderTxs   mapset.Set // Set of order transaction hashes known to be known by this peer
	knownLendingTxs mapset.Set // Set of lending transaction hashes known to be known by this peer
	knownVote       mapset.Set // Set of BFT Vote known to be known by this peer
	knownTimeout    mapset.Set // Set of BFT timeout known to be known by this peer
	knownSyncInfo   mapset.Set // Set of BFT Sync Info known to be known by this peer

	queuedBlocks    chan *propEvent   // Queue of blocks to broadcast to the peer
	queuedBlockAnns chan *types.Block // Queue of blocks to announce to the peer

	txBroadcast chan []common.Hash                   // Channel used to queue transaction propagation requests
	txAnnounce  chan []common.Hash                   // Channel used to queue transaction announcement requests
	getPooledTx func(common.Hash) *types.Transaction // Callback used to retrieve transaction from txpool

	term chan struct{} // Termination channel to stop the broadcaster
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter, getPooledTx func(hash common.Hash) *types.Transaction) *peer {
	return &peer{
		Peer:            p,
		rw:              rw,
		version:         version,
		id:              fmt.Sprintf("%x", p.ID().Bytes()[:8]),
		knownTxs:        mapset.NewSet(),
		knownBlocks:     mapset.NewSet(),
		knownOrderTxs:   mapset.NewSet(),
		knownLendingTxs: mapset.NewSet(),
		knownVote:       mapset.NewSet(),
		knownTimeout:    mapset.NewSet(),
		knownSyncInfo:   mapset.NewSet(),
		queuedBlocks:    make(chan *propEvent, maxQueuedBlocks),
		queuedBlockAnns: make(chan *types.Block, maxQueuedBlockAnns),
		txBroadcast:     make(chan []common.Hash),
		txAnnounce:      make(chan []common.Hash),
		getPooledTx:     getPooledTx,
		term:            make(chan struct{}),
	}
}

// broadcastBlocks is a write loop that multiplexes blocks and block accouncements
// to the remote peer. The goal is to have an async writer that does not lock up
// node internals and at the same time rate limits queued data.
func (p *peer) broadcastBlocks() {
	for {
		select {
		case prop := <-p.queuedBlocks:
			if err := p.SendNewBlock(prop.block, prop.td); err != nil {
				return
			}
			p.Log().Trace("Propagated block", "number", prop.block.Number(), "hash", prop.block.Hash(), "td", prop.td)

		case block := <-p.queuedBlockAnns:
			if err := p.SendNewBlockHashes([]common.Hash{block.Hash()}, []uint64{block.NumberU64()}); err != nil {
				return
			}
			p.Log().Trace("Announced block", "number", block.Number(), "hash", block.Hash())

		case <-p.term:
			return
		}
	}
}

// broadcastTransactions is a write loop that schedules transaction broadcasts
// to the remote peer. The goal is to have an async writer that does not lock up
// node internals and at the same time rate limits queued data.
func (p *peer) broadcastTransactions() {
	var (
		queue []common.Hash      // Queue of hashes to broadcast as full transactions
		done  chan struct{}      // Non-nil if background broadcaster is running
		fail  = make(chan error) // Channel used to receive network error
	)
	for {
		// If there's no in-flight broadcast running, check if a new one is needed
		if done == nil && len(queue) > 0 {
			// Pile transaction until we reach our allowed network limit
			var (
				hashes []common.Hash
				txs    []*types.Transaction
				size   common.StorageSize
			)
			for i := 0; i < len(queue) && size < txsyncPackSize; i++ {
				if tx := p.getPooledTx(queue[i]); tx != nil {
					txs = append(txs, tx)
					size += tx.Size()
				}
				hashes = append(hashes, queue[i])
			}
			queue = queue[:copy(queue, queue[len(hashes):])]

			// If there's anything available to transfer, fire up an async writer
			if len(txs) > 0 {
				done = make(chan struct{})
				go func() {
					if err := p.sendTransactions(txs); err != nil {
						fail <- err
						return
					}
					close(done)
					p.Log().Trace("Sent transactions", "count", len(txs))
				}()
			}
		}
		// Transfer goroutine may or may not have been started, listen for events
		select {
		case hashes := <-p.txBroadcast:
			// New batch of transactions to be broadcast, queue them (with cap)
			queue = append(queue, hashes...)
			if len(queue) > maxQueuedTxs {
				// Fancy copy and resize to ensure buffer doesn't grow indefinitely
				queue = queue[:copy(queue, queue[len(queue)-maxQueuedTxs:])]
			}

		case <-done:
			done = nil

		case <-fail:
			return

		case <-p.term:
			return
		}
	}
}

// announceTransactions is a write loop that schedules transaction broadcasts
// to the remote peer. The goal is to have an async writer that does not lock up
// node internals and at the same time rate limits queued data.
func (p *peer) announceTransactions() {
	var (
		queue []common.Hash      // Queue of hashes to announce as transaction stubs
		done  chan struct{}      // Non-nil if background announcer is running
		fail  = make(chan error) // Channel used to receive network error
	)
	for {
		// If there's no in-flight announce running, check if a new one is needed
		if done == nil && len(queue) > 0 {
			// Pile transaction hashes until we reach our allowed network limit
			var (
				hashes  []common.Hash
				pending []common.Hash
				size    common.StorageSize
			)
			for i := 0; i < len(queue) && size < txsyncPackSize; i++ {
				if p.getPooledTx(queue[i]) != nil {
					pending = append(pending, queue[i])
					size += common.HashLength
				}
				hashes = append(hashes, queue[i])
			}
			queue = queue[:copy(queue, queue[len(hashes):])]

			// If there's anything available to transfer, fire up an async writer
			if len(pending) > 0 {
				done = make(chan struct{})
				go func() {
					if err := p.sendPooledTransactionHashes(pending); err != nil {
						fail <- err
						return
					}
					close(done)
					p.Log().Trace("Sent transaction announcements", "count", len(pending))
				}()
			}
		}
		// Transfer goroutine may or may not have been started, listen for events
		select {
		case hashes := <-p.txAnnounce:
			// New batch of transactions to be broadcast, queue them (with cap)
			queue = append(queue, hashes...)
			if len(queue) > maxQueuedTxAnns {
				// Fancy copy and resize to ensure buffer doesn't grow indefinitely
				queue = queue[:copy(queue, queue[len(queue)-maxQueuedTxs:])]
			}

		case <-done:
			done = nil

		case <-fail:
			return

		case <-p.term:
			return
		}
	}
}

// close signals the broadcast goroutine to terminate.
func (p *peer) close() {
	close(p.term)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash, td := p.Head()

	return &PeerInfo{
		Version:    p.version,
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// Head retrieves a copy of the current head hash and total difficulty of the
// peer.
func (p *peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) MarkBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Cardinality() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// OrderMarkTransaction marks a order transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkOrderTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownOrderTxs.Cardinality() >= maxKnownOrderTxs {
		p.knownOrderTxs.Pop()
	}
	p.knownOrderTxs.Add(hash)
}

// MarkLendingTransaction marks a lending transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkLendingTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownLendingTxs.Cardinality() >= maxKnownLendingTxs {
		p.knownLendingTxs.Pop()
	}
	p.knownLendingTxs.Add(hash)
}

// MarkVote marks a vote as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkVote(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownVote.Cardinality() >= maxKnownVote {
		p.knownVote.Pop()
	}
	p.knownVote.Add(hash)
}

// MarkTimeout marks a timeout as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTimeout(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTimeout.Cardinality() >= maxKnownTimeout {
		p.knownTimeout.Pop()
	}
	p.knownTimeout.Add(hash)
}

// MarkSyncInfo marks a syncInfo as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkSyncInfo(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownSyncInfo.Cardinality() >= maxKnownSyncInfo {
		p.knownSyncInfo.Pop()
	}
	p.knownSyncInfo.Add(hash)
}

// SendTransactions64 sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
//
// This method is legacy support for initial transaction exchange in eth/64 and
// prior. For eth/65 and higher use SendPooledTransactionHashes.
func (p *peer) SendTransactions64(txs types.Transactions) error {
	return p.sendTransactions(txs)
}

// // SendTransactions sends transactions to the peer and includes the hashes
// // in its transaction hash set for future reference.
// func (p *peer) SendTransactions(txs types.Transactions) error {
// 	for p.knownTxs.Cardinality() >= maxKnownTxs {
// 	p.knownTxs.Pop()
// }
// for _, tx := range txs {
// 	p.knownTxs.Add(tx.Hash())
// return p2p.Send(p.rw, TxMsg, txs)
// }

// sendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
//
// This method is a helper used by the async transaction sender. Don't call it
// directly as the queueing (memory) and transmission (bandwidth) costs should
// not be managed directly.
func (p *peer) sendTransactions(txs types.Transactions) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(txs)) {
		p.knownTxs.Pop()
	}
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	return p2p.Send(p.rw, TransactionMsg, txs)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendOrderTransactions(txs types.OrderTransactions) error {
	for p.knownOrderTxs.Cardinality() >= maxKnownOrderTxs {
		p.knownOrderTxs.Pop()
	}

	for _, tx := range txs {
		p.knownOrderTxs.Add(tx.Hash())
	}
	return p2p.Send(p.rw, OrderTxMsg, txs)
}

// AsyncSendTransactions queues a list of transactions (by hash) to eventually
// propagate to a remote peer. The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *peer) AsyncSendTransactions(hashes []common.Hash) {
	select {
	case p.txBroadcast <- hashes:
		// Mark all the transactions as known, but ensure we don't overflow our limits
		for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
			p.knownTxs.Pop()
		}
		for _, hash := range hashes {
			p.knownTxs.Add(hash)
		}
	case <-p.term:
		p.Log().Debug("Dropping transaction propagation", "count", len(hashes))
	}
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendLendingTransactions(txs types.LendingTransactions) error {
	for p.knownLendingTxs.Cardinality() >= maxKnownLendingTxs {
		p.knownLendingTxs.Pop()
	}

	for _, tx := range txs {
		p.knownLendingTxs.Add(tx.Hash())
	}
	return p2p.Send(p.rw, LendingTxMsg, txs)
}

// sendPooledTransactionHashes sends transaction hashes to the peer and includes
// them in its transaction hash set for future reference.
//
// This method is a helper used by the async transaction announcer. Don't call it
// directly as the queueing (memory) and transmission (bandwidth) costs should
// not be managed directly.
func (p *peer) sendPooledTransactionHashes(hashes []common.Hash) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
		p.knownTxs.Pop()
	}
	for _, hash := range hashes {
		p.knownTxs.Add(hash)
	}
	return p2p.Send(p.rw, NewPooledTransactionHashesMsg, hashes)
}

// AsyncSendPooledTransactionHashes queues a list of transactions hashes to eventually
// announce to a remote peer.  The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *peer) AsyncSendPooledTransactionHashes(hashes []common.Hash) {
	select {
	case p.txAnnounce <- hashes:
		// Mark all the transactions as known, but ensure we don't overflow our limits
		for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
			p.knownTxs.Pop()
		}
		for _, hash := range hashes {
			p.knownTxs.Add(hash)
		}
	case <-p.term:
		p.Log().Debug("Dropping transaction announcement", "count", len(hashes))
	}
}

// SendPooledTransactionsRLP sends requested transactions to the peer and adds the
// hashes in its transaction hash set for future reference.
//
// Note, the method assumes the hashes are correct and correspond to the list of
// transactions being sent.
func (p *peer) SendPooledTransactionsRLP(hashes []common.Hash, txs []rlp.RawValue) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
		p.knownTxs.Pop()
	}
	for _, hash := range hashes {
		p.knownTxs.Add(hash)
	}
	return p2p.Send(p.rw, PooledTransactionsMsg, txs)
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendNewBlockHashes(hashes []common.Hash, numbers []uint64) error {
	// Mark all the block hashes as known, but ensure we don't overflow our limits
	for p.knownBlocks.Cardinality() > max(0, maxKnownBlocks-len(hashes)) {
		p.knownBlocks.Pop()
	}
	for _, hash := range hashes {
		p.knownBlocks.Add(hash)
	}
	request := make(newBlockHashesData, len(hashes))
	for i := 0; i < len(hashes); i++ {
		request[i].Hash = hashes[i]
		request[i].Number = numbers[i]
	}
	return p2p.Send(p.rw, NewBlockHashesMsg, request)
}

// // SendNewBlock propagates an entire block to a remote peer.
// func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
// 	// Mark all the block hash as known, but ensure we don't overflow our limits
// 	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
// 		p.knownBlocks.Pop()
// 	}
// 	p.knownBlocks.Add(block.Hash())
// 	return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, td})
// }

// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}

	p.knownBlocks.Add(block.Hash())
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, NewBlockMsg, []interface{}{block, td})
	} else {
		return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, td})
	}
}

// AsyncSendNewBlockHash queues the availability of a block for propagation to a
// remote peer. If the peer's broadcast queue is full, the event is silently
// dropped.
func (p *peer) AsyncSendNewBlockHash(block *types.Block) {
	select {
	case p.queuedBlockAnns <- block:
		// Mark all the block hash as known, but ensure we don't overflow our limits
		for p.knownBlocks.Cardinality() >= maxKnownBlocks {
			p.knownBlocks.Pop()
		}
		p.knownBlocks.Add(block.Hash())
	default:
		p.Log().Debug("Dropping block announcement", "number", block.NumberU64(), "hash", block.Hash())
	}
}

// AsyncSendNewBlock queues an entire block for propagation to a remote peer. If
// the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendNewBlock(block *types.Block, td *big.Int) {
	select {
	case p.queuedBlocks <- &propEvent{block: block, td: td}:
		// Mark all the block hash as known, but ensure we don't overflow our limits
		for p.knownBlocks.Cardinality() >= maxKnownBlocks {
			p.knownBlocks.Pop()
		}
		p.knownBlocks.Add(block.Hash())
	default:
		p.Log().Debug("Dropping block propagation", "number", block.NumberU64(), "hash", block.Hash())
	}
}

// SendBlockHeaders sends a batch of block headers to the remote peer.
func (p *peer) SendBlockHeaders(headers []*types.Header) error {
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, BlockHeadersMsg, headers)
	} else {
		return p2p.Send(p.rw, BlockHeadersMsg, headers)
	}
}

// SendBlockBodies sends a batch of block contents to the remote peer.
func (p *peer) SendBlockBodies(bodies []*blockBody) error {
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, BlockBodiesMsg, blockBodiesData(bodies))
	} else {
		return p2p.Send(p.rw, BlockBodiesMsg, blockBodiesData(bodies))
	}
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (p *peer) SendBlockBodiesRLP(bodies []rlp.RawValue) error {
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, BlockBodiesMsg, bodies)
	} else {
		return p2p.Send(p.rw, BlockBodiesMsg, bodies)
	}
}

// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data [][]byte) error {
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, NodeDataMsg, data)
	} else {
		return p2p.Send(p.rw, NodeDataMsg, data)
	}
}

// SendReceiptsRLP sends a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (p *peer) SendReceiptsRLP(receipts []rlp.RawValue) error {
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, ReceiptsMsg, receipts)
	} else {
		return p2p.Send(p.rw, ReceiptsMsg, receipts)
	}
}

func (p *peer) SendVote(vote *types.Vote) error {
	for p.knownVote.Cardinality() >= maxKnownVote {
		p.knownVote.Pop()
	}

	p.knownVote.Add(vote.Hash())
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, VoteMsg, vote)
	} else {
		return p2p.Send(p.rw, VoteMsg, vote)
	}
}

/*
func (p *peer) AsyncSendVote() {

}
*/
func (p *peer) SendTimeout(timeout *types.Timeout) error {
	for p.knownTimeout.Cardinality() >= maxKnownTimeout {
		p.knownTimeout.Pop()
	}

	p.knownTimeout.Add(timeout.Hash())
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, TimeoutMsg, timeout)
	} else {
		return p2p.Send(p.rw, TimeoutMsg, timeout)
	}
}

/*
func (p *peer) AsyncSendTimeout() {

}
*/
func (p *peer) SendSyncInfo(syncInfo *types.SyncInfo) error {
	for p.knownSyncInfo.Cardinality() >= maxKnownSyncInfo {
		p.knownSyncInfo.Pop()
	}

	p.knownSyncInfo.Add(syncInfo.Hash())
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, SyncInfoMsg, syncInfo)
	} else {
		return p2p.Send(p.rw, SyncInfoMsg, syncInfo)
	}
}

/*
func (p *peer) AsyncSendSyncInfo() {

}
*/

// RequestOneHeader is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *peer) RequestOneHeader(hash common.Hash) error {
	p.Log().Debug("Fetching single header", "hash", hash)
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: uint64(1), Skip: uint64(0), Reverse: false})
	} else {
		return p2p.Send(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: uint64(1), Skip: uint64(0), Reverse: false})
	}
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
	} else {
		return p2p.Send(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
	}
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
	} else {
		return p2p.Send(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
	}
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestBodies(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, GetBlockBodiesMsg, hashes)
	} else {
		return p2p.Send(p.rw, GetBlockBodiesMsg, hashes)
	}
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of state data", "count", len(hashes))
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, GetNodeDataMsg, hashes)
	} else {
		return p2p.Send(p.rw, GetNodeDataMsg, hashes)
	}
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	if p.pairRw != nil {
		return p2p.Send(p.pairRw, GetReceiptsMsg, hashes)
	} else {
		return p2p.Send(p.rw, GetReceiptsMsg, hashes)
	}
}

// RequestTxs fetches a batch of transactions from a remote node.
func (p *peer) RequestTxs(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of transactions", "count", len(hashes))
	return p2p.Send(p.rw, GetPooledTransactionsMsg, hashes)
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(network uint64, td *big.Int, head common.Hash, genesis common.Hash, forkID forkid.ID, forkFilter forkid.Filter) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)

	var (
		status63 statusData63 // safe to read after two values have been received from errc
		status   statusData   // safe to read after two values have been received from errc
	)
	go func() {
		switch {
		case supportsEth65(p.version):
			errc <- p2p.Send(p.rw, StatusMsg, &statusData{
				ProtocolVersion: uint32(p.version),
				NetworkID:       network,
				TD:              td,
				Head:            head,
				Genesis:         genesis,
				ForkID:          forkID,
			})
		case supportsEth64(p.version):
			errc <- p2p.Send(p.rw, StatusMsg, &statusData{
				ProtocolVersion: uint32(p.version),
				NetworkID:       network,
				TD:              td,
				Head:            head,
				Genesis:         genesis,
				ForkID:          forkID,
			})
		case supportsEth63(p.version):
			errc <- p2p.Send(p.rw, StatusMsg, &statusData63{
				ProtocolVersion: uint32(p.version),
				NetworkId:       network,
				TD:              td,
				CurrentBlock:    head,
				GenesisBlock:    genesis,
			})
		default:
			panic(fmt.Sprintf("unsupported eth protocol version: %d", p.version))
		}
	}()
	go func() {
		switch {
		case supportsEth64(p.version):
			errc <- p.readStatus(network, &status, genesis, forkFilter)
		case supportsEth63(p.version):
			errc <- p.readStatusLegacy(network, &status63, genesis)
		default:
			panic(fmt.Sprintf("unsupported eth protocol version: %d", p.version))
		}
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	switch {
	case supportsEth64(p.version):
		p.td, p.head = status.TD, status.Head
	case supportsEth63(p.version):
		p.td, p.head = status63.TD, status63.CurrentBlock
	default:
		panic(fmt.Sprintf("unsupported eth protocol version: %d", p.version))
	}
	return nil
}

func (p *peer) readStatusLegacy(network uint64, status *statusData63, genesis common.Hash) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > protocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, protocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(ErrGenesisMismatch, "%x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		return errResp(ErrNetworkIDMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}

func (p *peer) readStatus(network uint64, status *statusData, genesis common.Hash, forkFilter forkid.Filter) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > protocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, protocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.NetworkID != network {
		return errResp(ErrNetworkIDMismatch, "%d (!= %d)", status.NetworkID, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	if status.Genesis != genesis {
		return errResp(ErrGenesisMismatch, "%x (!= %x)", status.Genesis, genesis)
	}
	if err := forkFilter(status.ForkID); err != nil {
		return errResp(ErrForkIDRejected, "%v", err)
	}
	return nil
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("eth/%2d", p.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if existPeer, ok := ps.peers[p.id]; ok {
		if existPeer.pairRw != nil {
			return errAlreadyRegistered
		}
		existPeer.PairPeer = p.Peer
		existPeer.pairRw = p.rw
		p.PairPeer = existPeer.Peer
		return p2p.ErrAddPairPeer
	}
	ps.peers[p.id] = p

	go p.broadcastBlocks()
	go p.broadcastTransactions()
	go p.announceTransactions()

	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownBlocks.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownTxs.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutVote retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutVote(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownVote.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTimeout retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutTimeout(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownTimeout.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutSyncInfo retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutSyncInfo(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownSyncInfo.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) OrderPeersWithoutTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownOrderTxs.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// LendingPeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) LendingPeersWithoutTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownLendingTxs.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}
