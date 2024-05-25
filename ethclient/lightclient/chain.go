// Copyright 2024 The go-ethereum Authors
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

package lightclient

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	btypes "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const recentCanonicalLength = 256

type canonicalChain struct {
	lock             sync.Mutex
	headTracker      *light.HeadTracker
	blocksAndHeaders *blocksAndHeaders
	newHeadCb        func(common.Hash)

	head, finality *btypes.ExecutionHeader
	recent         map[uint64]common.Hash // nil until initialized
	recentTail     uint64                 // if recent != nil then recent hashes are available from recentTail to head
	tailFetchCh    chan struct{}
	finalized      *lru.Cache[uint64, common.Hash]  // finalized but not recent hashes
	requests       *requestMap[uint64, common.Hash] // requested; neither recent nor cached finalized
}

func newCanonicalChain(headTracker *light.HeadTracker, blocksAndHeaders *blocksAndHeaders, newHeadCb func(common.Hash)) *canonicalChain {
	c := &canonicalChain{
		headTracker:      headTracker,
		blocksAndHeaders: blocksAndHeaders,
		newHeadCb:        newHeadCb,
		finalized:        lru.NewCache[uint64, common.Hash](10000),
		requests:         newRequestMap[uint64, common.Hash](nil),
		tailFetchCh:      make(chan struct{}),
	}
	go c.tailFetcher()
	return c
}

// Process implements request.Module in order to get notified about new heads.
func (c *canonicalChain) Process(requester request.Requester, events []request.Event) {
	if finality, ok := c.headTracker.ValidatedFinality(); ok {
		finalized := finality.Finalized.PayloadHeader
		c.setFinality(finalized)
		c.blocksAndHeaders.addPayloadHeader(finalized)
	}
	if optimistic, ok := c.headTracker.ValidatedOptimistic(); ok {
		head := optimistic.Attested.PayloadHeader
		c.blocksAndHeaders.addPayloadHeader(head)
		if c.setHead(head) {
			c.newHeadCb(head.BlockHash()) // should not block
		}
	}
}

func (c *canonicalChain) tailFetcher() { //TODO stop
	for {
		c.lock.Lock()
		var (
			tailNum  uint64
			tailHash common.Hash
		)
		if c.recent != nil {
			tailNum, tailHash = c.recentTail, c.recent[c.recentTail]
		}
		needTail := tailNum
		for _, reqNum := range c.requests.allKeys() {
			if reqNum < needTail {
				needTail = reqNum
			}
		}
		c.lock.Unlock()
		if needTail < tailNum { //TODO check recentCanonicalLength
			log.Debug("Fetching tail headers", "have", tailNum, "need", needTail)
			ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
			//TODO parallel fetch by number
			if header, err := c.blocksAndHeaders.getHeader(ctx, tailHash); err == nil {
				c.addRecentTail(header)
			}
		} else {
			<-c.tailFetchCh
		}
	}
}

func (c *canonicalChain) getHash(ctx context.Context, number uint64) (common.Hash, error) {
	if hash, ok := c.getCachedHash(number); ok {
		return hash, nil
	}
	req := c.requests.request(number)
	select {
	case c.tailFetchCh <- struct{}{}:
	default:
	}
	defer req.release()
	return req.getResult(ctx)
}

func (c *canonicalChain) getCachedHash(number uint64) (common.Hash, bool) {
	c.lock.Lock()
	hash, ok := c.recent[number]
	c.lock.Unlock()
	if ok {
		return hash, true
	}
	return c.finalized.Get(number)
}

func (c *canonicalChain) setHead(head *btypes.ExecutionHeader) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	headNum, headHash := head.BlockNumber(), head.BlockHash()
	if c.head != nil && c.head.BlockHash() == headHash {
		return false
	}
	if c.recent == nil || c.head == nil || c.head.BlockNumber()+1 != headNum || c.head.BlockHash() != head.ParentHash() {
		// initialize recent canonical hash map when first head is added or when
		// it is not a descendant of the previous head
		c.recent = make(map[uint64]common.Hash)
		if headNum > 0 {
			c.recent[headNum-1] = head.ParentHash()
			c.recentTail = headNum - 1
		} else {
			c.recentTail = 0
		}
	}
	c.head = head
	c.recent[headNum] = headHash
	for headNum >= c.recentTail+recentCanonicalLength {
		if c.finality != nil && c.recentTail <= c.finality.BlockNumber() {
			c.finalized.Add(c.recentTail, c.recent[c.recentTail])
		}
		delete(c.recent, c.recentTail)
		c.recentTail++
	}
	c.requests.tryDeliver(headNum, headHash)
	log.Debug("SetHead", "recentTail", c.recentTail, "head", headNum)
	return true
}

func (c *canonicalChain) setFinality(finality *btypes.ExecutionHeader) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.finality = finality
	finalNum := finality.BlockNumber()
	if finalNum < c.recentTail {
		c.finalized.Add(finalNum, finality.BlockHash())
	}
	c.requests.tryDeliver(finalNum, finality.BlockHash())
}

func (c *canonicalChain) addRecentTail(tail *types.Header) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.recent == nil || tail.Number.Uint64() != c.recentTail || c.recent[c.recentTail] != tail.Hash() {
		return false
	}
	if c.recentTail > 0 {
		c.recentTail--
		c.recent[c.recentTail] = tail.ParentHash
		c.requests.tryDeliver(c.recentTail, tail.ParentHash)
	}
	return true
}

func (c *canonicalChain) getHead() *btypes.ExecutionHeader {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.head
}

func (c *canonicalChain) getFinality() *btypes.ExecutionHeader {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.finality
}

func (c *canonicalChain) resolveBlockNumber(number *big.Int) (uint64, *btypes.ExecutionHeader, error) {
	if !number.IsInt64() {
		return 0, nil, errors.New("Invalid block number")
	}
	num := number.Int64()
	if num < 0 {
		switch rpc.BlockNumber(num) {
		case rpc.SafeBlockNumber, rpc.FinalizedBlockNumber:
			if header := c.getFinality(); header != nil {
				return header.BlockNumber(), header, nil
			}
			return 0, nil, errors.New("Finalized block unknown")
		case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
			if header := c.getHead(); header != nil {
				return header.BlockNumber(), header, nil
			}
			return 0, nil, errors.New("Head block unknown")
		default:
			return 0, nil, errors.New("Invalid block number")
		}
	}
	return uint64(num), nil, nil
}

func (c *canonicalChain) blockNumberToHash(ctx context.Context, number *big.Int) (common.Hash, error) {
	num, pheader, err := c.resolveBlockNumber(number)
	if err != nil {
		return common.Hash{}, err
	}
	if pheader != nil {
		return pheader.BlockHash(), nil
	}
	return c.getHash(ctx, num)
}

func (c *canonicalChain) blockNumberOrHashToHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (common.Hash, error) {
	if blockNrOrHash.BlockNumber != nil {
		return c.blockNumberToHash(ctx, big.NewInt(int64(*blockNrOrHash.BlockNumber)))
	}
	hash := *blockNrOrHash.BlockHash
	if blockNrOrHash.RequireCanonical {
		header, err := c.blocksAndHeaders.getHeader(ctx, hash)
		if err != nil {
			return common.Hash{}, err
		}
		chash, err := c.getHash(ctx, header.Number.Uint64())
		if err != nil {
			return common.Hash{}, err
		}
		if chash != hash {
			return common.Hash{}, errors.New("hash is not currently canonical")
		}
	}
	return hash, nil
}

type blocksAndHeaders struct {
	client             *rpc.Client
	headerCache        *lru.Cache[common.Hash, *types.Header]
	headerRequests     *requestMap[common.Hash, *types.Header]
	payloadHeaderCache *lru.Cache[common.Hash, *btypes.ExecutionHeader]
	blockCache         *lru.Cache[common.Hash, *types.Block]
	blockRequests      *requestMap[common.Hash, *types.Block]
}

func newBlocksAndHeaders(client *rpc.Client) *blocksAndHeaders {
	b := &blocksAndHeaders{
		client:             client,
		headerCache:        lru.NewCache[common.Hash, *types.Header](1000),
		payloadHeaderCache: lru.NewCache[common.Hash, *btypes.ExecutionHeader](1000),
		blockCache:         lru.NewCache[common.Hash, *types.Block](10),
	}
	b.headerRequests = newRequestMap[common.Hash, *types.Header](b.requestHeader)
	b.blockRequests = newRequestMap[common.Hash, *types.Block](b.requestBlock)
	return b
}

func (b *blocksAndHeaders) requestHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	var header *types.Header
	log.Debug("Starting RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", false)
	err := b.client.CallContext(ctx, &header, "eth_getBlockByHash", hash, false)
	if err == nil && header.Hash() != hash {
		header, err = nil, errors.New("header hash does not match")
	}
	log.Debug("Finished RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", false, "error", err)
	return header, err
}

func (b *blocksAndHeaders) requestBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	var (
		raw   json.RawMessage
		block *types.Block
	)
	log.Debug("Starting RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", true)
	err := b.client.CallContext(ctx, &raw, "eth_getBlockByHash", hash, true)
	log.Debug("Finished RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", true, "error", err)
	if err == nil {
		block, err = decodeBlock(raw)
		if block.Hash() != hash {
			block, err = nil, errors.New("block hash does not match")
		}
	}
	return block, err
}

func (b *blocksAndHeaders) getHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if header, ok := b.headerCache.Get(hash); ok {
		return header, nil
	}
	if block, ok := b.blockCache.Get(hash); ok {
		return block.Header(), nil
	}
	if b.blockRequests.has(hash) && !b.headerRequests.has(hash) {
		req := b.blockRequests.request(hash)
		block, err := req.getResult(ctx)
		if err == nil {
			header := block.Header()
			b.headerCache.Add(hash, header)
			b.blockCache.Add(hash, block)
			req.release()
			return header, nil
		} else {
			req.release()
			return nil, err
		}
	}
	req := b.headerRequests.request(hash)
	header, err := req.getResult(ctx)
	if err == nil {
		b.headerCache.Add(hash, header)
	}
	req.release()
	return header, err
}

func (b *blocksAndHeaders) getPayloadHeader(hash common.Hash) *btypes.ExecutionHeader {
	pheader, _ := b.payloadHeaderCache.Get(hash)
	return pheader
}

func (b *blocksAndHeaders) getBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if block, ok := b.blockCache.Get(hash); ok {
		return block, nil
	}
	req := b.blockRequests.request(hash)
	block, err := req.getResult(ctx)
	if err == nil {
		header := block.Header()
		b.headerCache.Add(hash, header)
		b.headerRequests.tryDeliver(hash, header)
		b.blockCache.Add(hash, block)
	}
	req.release()
	return block, err
}

//TODO de-duplicate json block decoding
type rpcBlock struct {
	Hash         common.Hash         `json:"hash"`
	Transactions []rpcTransaction    `json:"transactions"`
	UncleHashes  []common.Hash       `json:"uncles"`
	Withdrawals  []*types.Withdrawal `json:"withdrawals,omitempty"`
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

// senderFromServer is a types.Signer that remembers the sender address returned by the RPC
// server. It is stored in the transaction's sender address cache to avoid an additional
// request in TransactionSender.
type senderFromServer struct {
	addr      common.Address
	blockhash common.Hash
}

func setSenderFromServer(tx *types.Transaction, addr common.Address, block common.Hash) {
	// Use types.Sender for side-effect to store our signer into the cache.
	types.Sender(&senderFromServer{addr, block}, tx)
}

var errNotCached = errors.New("sender not cached")

func (s *senderFromServer) Equal(other types.Signer) bool {
	os, ok := other.(*senderFromServer)
	return ok && os.blockhash == s.blockhash
}

func (s *senderFromServer) Sender(tx *types.Transaction) (common.Address, error) {
	if s.addr == (common.Address{}) {
		return common.Address{}, errNotCached
	}
	return s.addr, nil
}

func (s *senderFromServer) ChainID() *big.Int {
	panic("can't sign with senderFromServer")
}
func (s *senderFromServer) Hash(tx *types.Transaction) common.Hash {
	panic("can't sign with senderFromServer")
}
func (s *senderFromServer) SignatureValues(tx *types.Transaction, sig []byte) (R, S, V *big.Int, err error) {
	panic("can't sign with senderFromServer")
}

func decodeBlock(raw json.RawMessage) (*types.Block, error) {
	// Decode header and transactions.
	var head *types.Header
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	// When the block is not found, the API returns JSON null.
	if head == nil {
		return nil, ethereum.NotFound
	}

	var body rpcBlock
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if head.UncleHash == types.EmptyUncleHash && len(body.UncleHashes) > 0 {
		return nil, errors.New("server returned non-empty uncle list but block header indicates no uncles")
	}
	if head.UncleHash != types.EmptyUncleHash && len(body.UncleHashes) == 0 {
		return nil, errors.New("server returned empty uncle list but block header indicates uncles")
	}
	if head.TxHash == types.EmptyTxsHash && len(body.Transactions) > 0 {
		return nil, errors.New("server returned non-empty transaction list but block header indicates no transactions")
	}
	if head.TxHash != types.EmptyTxsHash && len(body.Transactions) == 0 {
		return nil, errors.New("server returned empty transaction list but block header indicates transactions")
	}
	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		if tx.From != nil {
			setSenderFromServer(tx.tx, *tx.From, body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(types.Body{Transactions: txs, Withdrawals: body.Withdrawals}), nil
}

func (b *blocksAndHeaders) addPayloadHeader(header *btypes.ExecutionHeader) {
	b.payloadHeaderCache.Add(header.BlockHash(), header)
}
