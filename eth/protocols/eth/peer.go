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
	"math/big"
	"math/rand"
	"sync"

	mapset "github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// maxKnownTxs is the maximum transactions hashes to keep in the known list
	// before starting to randomly evict them.
	maxKnownTxs = 32768

	// maxKnownBlocks is the maximum block hashes to keep in the known list
	// before starting to randomly evict them.
	maxKnownBlocks = 1024

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
)

// max is a helper function which returns the larger of the two given integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Peer is a collection of relevant information we have about a `eth` peer.
type Peer struct {
	id string // Unique ID for the peer, cached

	*p2p.Peer                   // The embedded P2P package peer
	rw        p2p.MsgReadWriter // Input/output streams for snap
	version   uint              // Protocol version negotiated

	head common.Hash // Latest advertised head block hash
	td   *big.Int    // Latest advertised head block total difficulty

	knownBlocks     mapset.Set             // Set of block hashes known to be known by this peer
	queuedBlocks    chan *blockPropagation // Queue of blocks to broadcast to the peer
	queuedBlockAnns chan *types.Block      // Queue of blocks to announce to the peer

	txpool      TxPool             // Transaction pool used by the broadcasters for liveness checks
	knownTxs    mapset.Set         // Set of transaction hashes known to be known by this peer
	txBroadcast chan []common.Hash // Channel used to queue transaction propagation requests
	txAnnounce  chan []common.Hash // Channel used to queue transaction announcement requests

	term chan struct{} // Termination channel to stop the broadcasters
	lock sync.RWMutex  // Mutex protecting the internal fields
}

// NewPeer create a wrapper for a network connection and negotiated  protocol
// version.
func NewPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, txpool TxPool) *Peer {
	peer := &Peer{
		id:              p.ID().String(),
		Peer:            p,
		rw:              rw,
		version:         version,
		knownTxs:        mapset.NewSet(),
		knownBlocks:     mapset.NewSet(),
		queuedBlocks:    make(chan *blockPropagation, maxQueuedBlocks),
		queuedBlockAnns: make(chan *types.Block, maxQueuedBlockAnns),
		txBroadcast:     make(chan []common.Hash),
		txAnnounce:      make(chan []common.Hash),
		txpool:          txpool,
		term:            make(chan struct{}),
	}
	// Start up all the broadcasters
	go peer.broadcastBlocks()
	go peer.broadcastTransactions()
	if version >= ETH65 {
		go peer.announceTransactions()
	}
	return peer
}

// Close signals the broadcast goroutine to terminate. Only ever call this if
// you created the peer yourself via NewPeer. Otherwise let whoever created it
// clean it up!
func (p *Peer) Close() {
	close(p.term)
}

// ID retrieves the peer's unique identifier.
func (p *Peer) ID() string {
	return p.id
}

// Version retrieves the peer's negoatiated `eth` protocol version.
func (p *Peer) Version() uint {
	return p.version
}

// Head retrieves the current head hash and total difficulty of the peer.
func (p *Peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *Peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// KnownBlock returns whether peer is known to already have a block.
func (p *Peer) KnownBlock(hash common.Hash) bool {
	return p.knownBlocks.Contains(hash)
}

// KnownTransaction returns whether peer is known to already have a transaction.
func (p *Peer) KnownTransaction(hash common.Hash) bool {
	return p.knownTxs.Contains(hash)
}

// markBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *Peer) markBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// markTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *Peer) markTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Cardinality() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
//
// This method is a helper used by the async transaction sender. Don't call it
// directly as the queueing (memory) and transmission (bandwidth) costs should
// not be managed directly.
//
// The reasons this is public is to allow packages using this protocol to write
// tests that directly send messages without having to do the asyn queueing.
func (p *Peer) SendTransactions(txs types.Transactions) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(txs)) {
		p.knownTxs.Pop()
	}
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	return p2p.Send(p.rw, TransactionsMsg, txs)
}

// AsyncSendTransactions queues a list of transactions (by hash) to eventually
// propagate to a remote peer. The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *Peer) AsyncSendTransactions(hashes []common.Hash) {
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

// sendPooledTransactionHashes sends transaction hashes to the peer and includes
// them in its transaction hash set for future reference.
//
// This method is a helper used by the async transaction announcer. Don't call it
// directly as the queueing (memory) and transmission (bandwidth) costs should
// not be managed directly.
func (p *Peer) sendPooledTransactionHashes(hashes []common.Hash) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
		p.knownTxs.Pop()
	}
	for _, hash := range hashes {
		p.knownTxs.Add(hash)
	}
	return p2p.Send(p.rw, NewPooledTransactionHashesMsg, NewPooledTransactionHashesPacket(hashes))
}

// AsyncSendPooledTransactionHashes queues a list of transactions hashes to eventually
// announce to a remote peer.  The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *Peer) AsyncSendPooledTransactionHashes(hashes []common.Hash) {
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
func (p *Peer) SendPooledTransactionsRLP(hashes []common.Hash, txs []rlp.RawValue) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
		p.knownTxs.Pop()
	}
	for _, hash := range hashes {
		p.knownTxs.Add(hash)
	}
	return p2p.Send(p.rw, PooledTransactionsMsg, txs) // Not packed into PooledTransactionsPacket to avoid RLP decoding
}

// ReplyPooledTransactionsRLP is the eth/66 version of SendPooledTransactionsRLP.
func (p *Peer) ReplyPooledTransactionsRLP(id uint64, hashes []common.Hash, txs []rlp.RawValue) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for p.knownTxs.Cardinality() > max(0, maxKnownTxs-len(hashes)) {
		p.knownTxs.Pop()
	}
	for _, hash := range hashes {
		p.knownTxs.Add(hash)
	}
	// Not packed into PooledTransactionsPacket to avoid RLP decoding
	return p2p.Send(p.rw, PooledTransactionsMsg, PooledTransactionsRLPPacket66{
		RequestId:                   id,
		PooledTransactionsRLPPacket: txs,
	})
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (p *Peer) SendNewBlockHashes(hashes []common.Hash, numbers []uint64) error {
	// Mark all the block hashes as known, but ensure we don't overflow our limits
	for p.knownBlocks.Cardinality() > max(0, maxKnownBlocks-len(hashes)) {
		p.knownBlocks.Pop()
	}
	for _, hash := range hashes {
		p.knownBlocks.Add(hash)
	}
	request := make(NewBlockHashesPacket, len(hashes))
	for i := 0; i < len(hashes); i++ {
		request[i].Hash = hashes[i]
		request[i].Number = numbers[i]
	}
	return p2p.Send(p.rw, NewBlockHashesMsg, request)
}

// AsyncSendNewBlockHash queues the availability of a block for propagation to a
// remote peer. If the peer's broadcast queue is full, the event is silently
// dropped.
func (p *Peer) AsyncSendNewBlockHash(block *types.Block) {
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

// SendNewBlock propagates an entire block to a remote peer.
func (p *Peer) SendNewBlock(block *types.Block, td *big.Int) error {
	// Mark all the block hash as known, but ensure we don't overflow our limits
	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(block.Hash())
	return p2p.Send(p.rw, NewBlockMsg, &NewBlockPacket{
		Block: block,
		TD:    td,
	})
}

// AsyncSendNewBlock queues an entire block for propagation to a remote peer. If
// the peer's broadcast queue is full, the event is silently dropped.
func (p *Peer) AsyncSendNewBlock(block *types.Block, td *big.Int) {
	select {
	case p.queuedBlocks <- &blockPropagation{block: block, td: td}:
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
func (p *Peer) SendBlockHeaders(headers []*types.Header) error {
	return p2p.Send(p.rw, BlockHeadersMsg, BlockHeadersPacket(headers))
}

// ReplyBlockHeaders is the eth/66 version of SendBlockHeaders.
func (p *Peer) ReplyBlockHeaders(id uint64, headers []*types.Header) error {
	return p2p.Send(p.rw, BlockHeadersMsg, BlockHeadersPacket66{
		RequestId:          id,
		BlockHeadersPacket: headers,
	})
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (p *Peer) SendBlockBodiesRLP(bodies []rlp.RawValue) error {
	return p2p.Send(p.rw, BlockBodiesMsg, bodies) // Not packed into BlockBodiesPacket to avoid RLP decoding
}

// ReplyBlockBodiesRLP is the eth/66 version of SendBlockBodiesRLP.
func (p *Peer) ReplyBlockBodiesRLP(id uint64, bodies []rlp.RawValue) error {
	// Not packed into BlockBodiesPacket to avoid RLP decoding
	return p2p.Send(p.rw, BlockBodiesMsg, BlockBodiesRLPPacket66{
		RequestId:            id,
		BlockBodiesRLPPacket: bodies,
	})
}

// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *Peer) SendNodeData(data [][]byte) error {
	return p2p.Send(p.rw, NodeDataMsg, NodeDataPacket(data))
}

// ReplyNodeData is the eth/66 response to GetNodeData.
func (p *Peer) ReplyNodeData(id uint64, data [][]byte) error {
	return p2p.Send(p.rw, NodeDataMsg, NodeDataPacket66{
		RequestId:      id,
		NodeDataPacket: data,
	})
}

// SendReceiptsRLP sends a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (p *Peer) SendReceiptsRLP(receipts []rlp.RawValue) error {
	return p2p.Send(p.rw, ReceiptsMsg, receipts) // Not packed into ReceiptsPacket to avoid RLP decoding
}

// ReplyReceiptsRLP is the eth/66 response to GetReceipts.
func (p *Peer) ReplyReceiptsRLP(id uint64, receipts []rlp.RawValue) error {
	return p2p.Send(p.rw, ReceiptsMsg, ReceiptsRLPPacket66{
		RequestId:         id,
		ReceiptsRLPPacket: receipts,
	})
}

// RequestOneHeader is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *Peer) RequestOneHeader(hash common.Hash) error {
	p.Log().Debug("Fetching single header", "hash", hash)
	query := GetBlockHeadersPacket{
		Origin:  HashOrNumber{Hash: hash},
		Amount:  uint64(1),
		Skip:    uint64(0),
		Reverse: false,
	}
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetBlockHeadersMsg, BlockHeadersMsg, id)
		return p2p.Send(p.rw, GetBlockHeadersMsg, &GetBlockHeadersPacket66{
			RequestId:             id,
			GetBlockHeadersPacket: &query,
		})
	}
	return p2p.Send(p.rw, GetBlockHeadersMsg, &query)
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *Peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	query := GetBlockHeadersPacket{
		Origin:  HashOrNumber{Hash: origin},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetBlockHeadersMsg, BlockHeadersMsg, id)
		return p2p.Send(p.rw, GetBlockHeadersMsg, &GetBlockHeadersPacket66{
			RequestId:             id,
			GetBlockHeadersPacket: &query,
		})
	}
	return p2p.Send(p.rw, GetBlockHeadersMsg, &query)
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *Peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	query := GetBlockHeadersPacket{
		Origin:  HashOrNumber{Number: origin},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetBlockHeadersMsg, BlockHeadersMsg, id)
		return p2p.Send(p.rw, GetBlockHeadersMsg, &GetBlockHeadersPacket66{
			RequestId:             id,
			GetBlockHeadersPacket: &query,
		})
	}
	return p2p.Send(p.rw, GetBlockHeadersMsg, &query)
}

// ExpectRequestHeadersByNumber is a testing method to mirror the recipient side
// of the RequestHeadersByNumber operation.
func (p *Peer) ExpectRequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	req := &GetBlockHeadersPacket{
		Origin:  HashOrNumber{Number: origin},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}
	return p2p.ExpectMsg(p.rw, GetBlockHeadersMsg, req)
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *Peer) RequestBodies(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetBlockBodiesMsg, BlockBodiesMsg, id)
		return p2p.Send(p.rw, GetBlockBodiesMsg, &GetBlockBodiesPacket66{
			RequestId:            id,
			GetBlockBodiesPacket: hashes,
		})
	}
	return p2p.Send(p.rw, GetBlockBodiesMsg, GetBlockBodiesPacket(hashes))
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *Peer) RequestNodeData(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of state data", "count", len(hashes))
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetNodeDataMsg, NodeDataMsg, id)
		return p2p.Send(p.rw, GetNodeDataMsg, &GetNodeDataPacket66{
			RequestId:         id,
			GetNodeDataPacket: hashes,
		})
	}
	return p2p.Send(p.rw, GetNodeDataMsg, GetNodeDataPacket(hashes))
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *Peer) RequestReceipts(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetReceiptsMsg, ReceiptsMsg, id)
		return p2p.Send(p.rw, GetReceiptsMsg, &GetReceiptsPacket66{
			RequestId:         id,
			GetReceiptsPacket: hashes,
		})
	}
	return p2p.Send(p.rw, GetReceiptsMsg, GetReceiptsPacket(hashes))
}

// RequestTxs fetches a batch of transactions from a remote node.
func (p *Peer) RequestTxs(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of transactions", "count", len(hashes))
	if p.Version() >= ETH66 {
		id := rand.Uint64()

		requestTracker.Track(p.id, p.version, GetPooledTransactionsMsg, PooledTransactionsMsg, id)
		return p2p.Send(p.rw, GetPooledTransactionsMsg, &GetPooledTransactionsPacket66{
			RequestId:                   id,
			GetPooledTransactionsPacket: hashes,
		})
	}
	return p2p.Send(p.rw, GetPooledTransactionsMsg, GetPooledTransactionsPacket(hashes))
}
