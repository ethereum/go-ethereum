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
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// maxKnownTxs is the maximum transactions hashes to keep in the known list
	// before starting to randomly evict them.
	maxKnownTxs = 32768

	// maxQueuedTxs is the maximum number of transactions to queue up before dropping
	// older broadcasts.
	maxQueuedTxs = 4096

	// maxQueuedTxAnns is the maximum number of transaction announcements to queue up
	// before dropping older announcements.
	maxQueuedTxAnns = 4096
)

// receiptRequest tracks the state of an in-flight receipt retrieval operation.
type receiptRequest struct {
	request     []common.Hash    // block hashes corresponding to the requested receipts
	list        []*ReceiptList69 // list of partially collected receipts
	lastLogSize uint64           // log size of last receipt list
}

// Peer is a collection of relevant information we have about a `eth` peer.
type Peer struct {
	id string // Unique ID for the peer, cached

	*p2p.Peer                   // The embedded P2P package peer
	rw        p2p.MsgReadWriter // Input/output streams for snap
	version   uint              // Protocol version negotiated
	lastRange atomic.Pointer[BlockRangeUpdatePacket]

	txpool      TxPool             // Transaction pool used by the broadcasters for liveness checks
	knownTxs    *knownCache        // Set of transaction hashes known to be known by this peer
	txBroadcast chan []common.Hash // Channel used to queue transaction propagation requests
	txAnnounce  chan []common.Hash // Channel used to queue transaction announcement requests

	reqDispatch chan *request  // Dispatch channel to send requests and track then until fulfillment
	reqCancel   chan *cancel   // Dispatch channel to cancel pending requests and untrack them
	resDispatch chan *response // Dispatch channel to fulfil pending requests and untrack them

	receiptBuffer     map[uint64]*receiptRequest // Previously requested receipts to buffer partial receipts
	receiptBufferLock sync.RWMutex               // Lock for protecting the receiptBuffer

	term chan struct{} // Termination channel to stop the broadcasters
}

// NewPeer creates a wrapper for a network connection and negotiated  protocol
// version.
func NewPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, txpool TxPool) *Peer {
	peer := &Peer{
		id:            p.ID().String(),
		Peer:          p,
		rw:            rw,
		version:       version,
		knownTxs:      newKnownCache(maxKnownTxs),
		txBroadcast:   make(chan []common.Hash),
		txAnnounce:    make(chan []common.Hash),
		reqDispatch:   make(chan *request),
		reqCancel:     make(chan *cancel),
		resDispatch:   make(chan *response),
		txpool:        txpool,
		receiptBuffer: make(map[uint64]*receiptRequest),
		term:          make(chan struct{}),
	}
	// Start up all the broadcasters
	go peer.broadcastTransactions()
	go peer.announceTransactions()
	go peer.dispatcher()

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

// Version retrieves the peer's negotiated `eth` protocol version.
func (p *Peer) Version() uint {
	return p.version
}

// BlockRange returns the latest announced block range.
// This will be nil for peers below protocol version eth/69.
func (p *Peer) BlockRange() *BlockRangeUpdatePacket {
	return p.lastRange.Load()
}

// KnownTransaction returns whether peer is known to already have a transaction.
func (p *Peer) KnownTransaction(hash common.Hash) bool {
	return p.knownTxs.Contains(hash)
}

// markTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *Peer) markTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
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
// tests that directly send messages without having to do the async queueing.
func (p *Peer) SendTransactions(txs types.Transactions) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
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
		p.knownTxs.Add(hashes...)
	case <-p.term:
		p.Log().Debug("Dropping transaction propagation", "count", len(hashes))
	}
}

// sendPooledTransactionHashes sends transaction hashes (tagged with their type
// and size) to the peer and includes them in its transaction hash set for future
// reference.
//
// This method is a helper used by the async transaction announcer. Don't call it
// directly as the queueing (memory) and transmission (bandwidth) costs should
// not be managed directly.
func (p *Peer) sendPooledTransactionHashes(hashes []common.Hash, types []byte, sizes []uint32) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	p.knownTxs.Add(hashes...)
	return p2p.Send(p.rw, NewPooledTransactionHashesMsg, NewPooledTransactionHashesPacket{Types: types, Sizes: sizes, Hashes: hashes})
}

// AsyncSendPooledTransactionHashes queues a list of transactions hashes to eventually
// announce to a remote peer.  The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *Peer) AsyncSendPooledTransactionHashes(hashes []common.Hash) {
	select {
	case p.txAnnounce <- hashes:
		// Mark all the transactions as known, but ensure we don't overflow our limits
		p.knownTxs.Add(hashes...)
	case <-p.term:
		p.Log().Debug("Dropping transaction announcement", "count", len(hashes))
	}
}

// ReplyPooledTransactionsRLP is the response to RequestTxs.
func (p *Peer) ReplyPooledTransactionsRLP(id uint64, hashes []common.Hash, txs []rlp.RawValue) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	p.knownTxs.Add(hashes...)

	// Not packed into PooledTransactionsResponse to avoid RLP decoding
	return p2p.Send(p.rw, PooledTransactionsMsg, &PooledTransactionsRLPPacket{
		RequestId:                     id,
		PooledTransactionsRLPResponse: txs,
	})
}

// ReplyBlockHeadersRLP is the response to GetBlockHeaders.
func (p *Peer) ReplyBlockHeadersRLP(id uint64, headers []rlp.RawValue) error {
	return p2p.Send(p.rw, BlockHeadersMsg, &BlockHeadersRLPPacket{
		RequestId:               id,
		BlockHeadersRLPResponse: headers,
	})
}

// ReplyBlockBodiesRLP is the response to GetBlockBodies.
func (p *Peer) ReplyBlockBodiesRLP(id uint64, bodies []rlp.RawValue) error {
	// Not packed into BlockBodiesResponse to avoid RLP decoding
	return p2p.Send(p.rw, BlockBodiesMsg, &BlockBodiesRLPPacket{
		RequestId:              id,
		BlockBodiesRLPResponse: bodies,
	})
}

// ReplyReceiptsRLP69 is the response to GetReceipts.
func (p *Peer) ReplyReceiptsRLP69(id uint64, receipts []rlp.RawValue) error {
	return p2p.Send(p.rw, ReceiptsMsg, &ReceiptsRLPPacket69{
		RequestId:           id,
		ReceiptsRLPResponse: receipts,
	})
}

// ReplyReceiptsRLP70 is the response to GetReceipts.
func (p *Peer) ReplyReceiptsRLP70(id uint64, receipts []rlp.RawValue, lastBlockIncomplete bool) error {
	return p2p.Send(p.rw, ReceiptsMsg, &ReceiptsRLPPacket70{
		RequestId:           id,
		ReceiptsRLPResponse: receipts,
		LastBlockIncomplete: lastBlockIncomplete,
	})
}

// RequestOneHeader is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *Peer) RequestOneHeader(hash common.Hash, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching single header", "hash", hash)
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockHeadersMsg,
		want: BlockHeadersMsg,
		data: &GetBlockHeadersPacket{
			RequestId: id,
			GetBlockHeadersRequest: &GetBlockHeadersRequest{
				Origin:  HashOrNumber{Hash: hash},
				Amount:  uint64(1),
				Skip:    uint64(0),
				Reverse: false,
			},
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *Peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockHeadersMsg,
		want: BlockHeadersMsg,
		data: &GetBlockHeadersPacket{
			RequestId: id,
			GetBlockHeadersRequest: &GetBlockHeadersRequest{
				Origin:  HashOrNumber{Hash: origin},
				Amount:  uint64(amount),
				Skip:    uint64(skip),
				Reverse: reverse,
			},
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *Peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockHeadersMsg,
		want: BlockHeadersMsg,
		data: &GetBlockHeadersPacket{
			RequestId: id,
			GetBlockHeadersRequest: &GetBlockHeadersRequest{
				Origin:  HashOrNumber{Number: origin},
				Amount:  uint64(amount),
				Skip:    uint64(skip),
				Reverse: reverse,
			},
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *Peer) RequestBodies(hashes []common.Hash, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockBodiesMsg,
		want: BlockBodiesMsg,
		data: &GetBlockBodiesPacket{
			RequestId:             id,
			GetBlockBodiesRequest: hashes,
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *Peer) RequestReceipts(hashes []common.Hash, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	id := rand.Uint64()

	var req *Request
	if p.version > ETH69 {
		req = &Request{
			id:   id,
			sink: sink,
			code: GetReceiptsMsg,
			want: ReceiptsMsg,
			data: &GetReceiptsPacket70{
				RequestId:              id,
				FirstBlockReceiptIndex: 0,
				GetReceiptsRequest:     hashes,
			},
		}
		p.receiptBufferLock.Lock()
		p.receiptBuffer[id] = &receiptRequest{
			request: hashes,
		}
		p.receiptBufferLock.Unlock()
	} else {
		req = &Request{
			id:   id,
			sink: sink,
			code: GetReceiptsMsg,
			want: ReceiptsMsg,
			data: &GetReceiptsPacket69{
				RequestId:          id,
				GetReceiptsRequest: hashes,
			},
		}
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// HandlePartialReceipts re-request partial receipts
func (p *Peer) requestPartialReceipts(id uint64) error {
	p.receiptBufferLock.RLock()
	defer p.receiptBufferLock.RUnlock()

	if _, ok := p.receiptBuffer[id]; !ok {
		return fmt.Errorf("no partial receipt retreival in progress with id %d", id)
	}
	lastBlock := len(p.receiptBuffer[id].list) - 1
	lastReceipt := len(p.receiptBuffer[id].list[lastBlock].items)

	req := &Request{
		id:   id,
		sink: nil,
		code: GetReceiptsMsg,
		want: ReceiptsMsg,
		data: &GetReceiptsPacket70{
			RequestId:              id,
			FirstBlockReceiptIndex: uint64(lastReceipt),
			GetReceiptsRequest:     p.receiptBuffer[id].request[lastBlock:],
		},
	}
	return p.dispatchRequest(req)
}

// bufferReceiptsPacket validates a receipt packet and buffer the incomplete packet.
// If the request is completed, it appends previously collected receipts.
func (p *Peer) bufferReceiptsPacket(packet *ReceiptsPacket70, backend Backend) error {
	p.receiptBufferLock.Lock()
	defer p.receiptBufferLock.Unlock()

	requestId := packet.RequestId
	buffer := p.receiptBuffer[requestId]

	// Do not assign buffer to the response not requested
	if buffer == nil {
		return fmt.Errorf("no partial receipt retreival in progress with id %d", requestId)
	}
	// If the response is empty, the peer likely does not have the requested receipts.
	// Forward the empty response to the internal handler regardless. However, note
	// that an empty response marked as incomplete is considered invalid.
	if len(packet.List) == 0 {
		delete(p.receiptBuffer, requestId)

		if packet.LastBlockIncomplete {
			return errors.New("invalid empty receipt response with incomplete flag")
		}
		return nil
	}
	// Buffer the last block when the response is incomplete.
	if packet.LastBlockIncomplete {
		lastBlock := len(packet.List) - 1
		if len(buffer.list) > 0 {
			lastBlock += len(buffer.list) - 1
		}
		header := backend.Chain().GetHeaderByHash(buffer.request[lastBlock])
		if header == nil {
			return fmt.Errorf("unknown block #%d for receipt retrieval", lastBlock)
		}
		logSize, err := p.validateLastBlockReceipt(packet.List, requestId, header)
		if err != nil {
			return err
		}
		// Update the buffered data and trim the packet to exclude the incomplete block.
		if len(buffer.list) > 0 {
			// If the buffer is already allocated, it means that the previous response
			// was incomplete Append the first block receipts.
			buffer.list[len(buffer.list)-1].Append(packet.List[0])
			buffer.list = append(buffer.list, packet.List[1:]...)
			buffer.lastLogSize = logSize
		} else {
			buffer.list = packet.List
			buffer.lastLogSize = logSize
		}
		return nil
	}
	// If the request is complete, append the previously collected receipts to the
	// packet and discard the buffered receipts. Also clear the cached buffer used
	// for storing partial responses.
	delete(p.receiptBuffer, requestId)

	// Short circuit if there is nothing cached previously
	if len(buffer.list) == 0 {
		return nil
	}
	// Aggregate the cached result into the packet
	buffer.list[len(buffer.list)-1].Append(packet.List[0])
	packet.List = packet.List[1:]
	packet.List = append(buffer.list, packet.List...)
	return nil
}

// validateLastBlockReceipt validates receipts and return log size of last block receipt.
// This function is called only when the `lastBlockincomplete == true`.
// Note that the last receipt response (which completes receiptLists of a pending block) is not verified here.
// Those response doesn't need hueristics below since they can be verified by its trie root.
func (p *Peer) validateLastBlockReceipt(receiptLists []*ReceiptList69, id uint64, header *types.Header) (uint64, error) {
	lastReceipts := receiptLists[len(receiptLists)-1]

	// If the receipt is in the middle of retrieval, use the buffered data.
	// e.g. [[receipt1], [receipt1, receipt2], incomplete = true]
	//      [[receipt3, receipt4], incomplete = true] <<--
	//      [[receipt5], [receipt1], incomplete = false]
	// This case happens only if len(receiptLists) == 1 && incomplete == true && buffered before.
	var previousTxs int
	var previousLog uint64
	var log uint64
	if buffer, ok := p.receiptBuffer[id]; ok && len(buffer.list) > 0 && len(receiptLists) == 1 {
		previousTxs = len(buffer.list[len(buffer.list)-1].items)
		previousLog = buffer.lastLogSize
	}

	// Verify that the total number of transactions delivered is under the limit.
	if uint64(previousTxs+len(lastReceipts.items)) > header.GasUsed/21_000 {
		// should be dropped, don't clear the buffer
		return 0, fmt.Errorf("total number of tx exceeded limit")
	}
	for _, rc := range lastReceipts.items {
		log += uint64(len(rc.Logs))
	}
	// Verify that the overall downloaded receipt size does not exceed the block gas limit.
	if previousLog+log > header.GasUsed/params.LogDataGas {
		return 0, fmt.Errorf("total download receipt size exceeded the limit")
	}
	return previousLog + log, nil
}

// RequestTxs fetches a batch of transactions from a remote node.
func (p *Peer) RequestTxs(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of transactions", "count", len(hashes))
	id := rand.Uint64()

	requestTracker.Track(p.id, p.version, GetPooledTransactionsMsg, PooledTransactionsMsg, id)
	return p2p.Send(p.rw, GetPooledTransactionsMsg, &GetPooledTransactionsPacket{
		RequestId:                    id,
		GetPooledTransactionsRequest: hashes,
	})
}

// SendBlockRangeUpdate sends a notification about our available block range to the peer.
func (p *Peer) SendBlockRangeUpdate(msg BlockRangeUpdatePacket) error {
	if p.version < ETH69 {
		return nil
	}
	return p2p.Send(p.rw, BlockRangeUpdateMsg, &msg)
}

// knownCache is a cache for known hashes.
type knownCache struct {
	hashes mapset.Set[common.Hash]
	max    int
}

// newKnownCache creates a new knownCache with a max capacity.
func newKnownCache(max int) *knownCache {
	return &knownCache{
		max:    max,
		hashes: mapset.NewSet[common.Hash](),
	}
}

// Add adds a list of elements to the set.
func (k *knownCache) Add(hashes ...common.Hash) {
	for k.hashes.Cardinality() > max(0, k.max-len(hashes)) {
		k.hashes.Pop()
	}
	for _, hash := range hashes {
		k.hashes.Add(hash)
	}
}

// Contains returns whether the given item is in the set.
func (k *knownCache) Contains(hash common.Hash) bool {
	return k.hashes.Contains(hash)
}

// Cardinality returns the number of elements in the set.
func (k *knownCache) Cardinality() int {
	return k.hashes.Cardinality()
}
