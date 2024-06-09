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
	"github.com/ethereum/go-ethereum/beacon/light/request"
	btypes "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const recentCanonicalLength = 256

type canonicalChainFields struct {
	chainLock      sync.Mutex
	head, finality *btypes.ExecutionHeader
	recent         map[uint64]common.Hash // nil while head == nil
	recentTail     uint64                 // if recent != nil then recent hashes are available from recentTail to head
	tailFetchCh    chan struct{}
	finalized      *lru.Cache[uint64, common.Hash]  // finalized but not recent hashes
	requests       *requestMap[uint64, common.Hash] // requested; neither recent nor cached finalized
}

func (c *Client) initCanonicalChain() {
	c.finalized = lru.NewCache[uint64, common.Hash](10000)
	c.requests = newRequestMap[uint64, common.Hash](nil)
	c.tailFetchCh = make(chan struct{})
	go c.tailFetcher()
}

func (c *Client) closeCanonicalChain() {
	c.requests.close()
}

// Process implements request.Module in order to get notified about new heads.
func (c *Client) Process(requester request.Requester, events []request.Event) {
	if finality, ok := c.headTracker.ValidatedFinality(); ok {
		finalized := finality.Finalized.PayloadHeader
		c.setFinality(finalized)
		c.addPayloadHeader(finalized)
	}
	if optimistic, ok := c.headTracker.ValidatedOptimistic(); ok {
		head := optimistic.Attested.PayloadHeader
		c.addPayloadHeader(head)
		if c.setHead(head) {
			go c.processNewHead(head.BlockNumber(), head.BlockHash())
		}
	}
}

func (c *Client) tailFetcher() { //TODO stop
	for {
		c.chainLock.Lock()
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
		c.chainLock.Unlock()
		if needTail < tailNum { //TODO check recentCanonicalLength
			log.Debug("Fetching tail headers", "have", tailNum, "need", needTail)
			ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
			//TODO parallel fetch by number
			if header, err := c.getHeader(ctx, tailHash); err == nil {
				c.addRecentTail(header)
			}
		} else {
			<-c.tailFetchCh
		}
	}
}

func (c *Client) getHash(ctx context.Context, number uint64) (common.Hash, error) {
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

func (c *Client) getCachedHash(number uint64) (common.Hash, bool) {
	c.chainLock.Lock()
	hash, ok := c.recent[number]
	c.chainLock.Unlock()
	if ok {
		return hash, true
	}
	return c.finalized.Get(number)
}

func (c *Client) setHead(head *btypes.ExecutionHeader) bool {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

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

func (c *Client) setFinality(finality *btypes.ExecutionHeader) {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	c.finality = finality
	finalNum := finality.BlockNumber()
	if finalNum < c.recentTail {
		c.finalized.Add(finalNum, finality.BlockHash())
	}
	c.requests.tryDeliver(finalNum, finality.BlockHash())
}

func (c *Client) addRecentTail(tail *types.Header) bool {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

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

func (c *Client) getHead() *btypes.ExecutionHeader {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	return c.head
}

func (c *Client) getFinality() *btypes.ExecutionHeader {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	return c.finality
}

func (c *Client) resolveBlockNumber(number *big.Int) (uint64, *btypes.ExecutionHeader, error) {
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

func (c *Client) blockNumberToHash(ctx context.Context, number *big.Int) (common.Hash, error) {
	num, pheader, err := c.resolveBlockNumber(number)
	if err != nil {
		return common.Hash{}, err
	}
	if pheader != nil {
		return pheader.BlockHash(), nil
	}
	return c.getHash(ctx, num)
}

func (c *Client) blockNumberOrHashToHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (common.Hash, error) {
	if blockNrOrHash.BlockNumber != nil {
		return c.blockNumberToHash(ctx, big.NewInt(int64(*blockNrOrHash.BlockNumber)))
	}
	hash := *blockNrOrHash.BlockHash
	if blockNrOrHash.RequireCanonical {
		header, err := c.getHeader(ctx, hash)
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

type blocksAndHeadersFields struct {
	headerCache        *lru.Cache[common.Hash, *types.Header]
	headerRequests     *requestMap[common.Hash, *types.Header]
	payloadHeaderCache *lru.Cache[common.Hash, *btypes.ExecutionHeader]
	blockCache         *lru.Cache[common.Hash, *types.Block]
	blockRequests      *requestMap[common.Hash, *types.Block]
}

func (c *Client) initBlocksAndHeaders() {
	c.headerCache = lru.NewCache[common.Hash, *types.Header](1000)
	c.payloadHeaderCache = lru.NewCache[common.Hash, *btypes.ExecutionHeader](1000)
	c.blockCache = lru.NewCache[common.Hash, *types.Block](10)
	c.headerRequests = newRequestMap[common.Hash, *types.Header](c.requestHeader)
	c.blockRequests = newRequestMap[common.Hash, *types.Block](c.requestBlock)
}

func (c *Client) closeBlocksAndHeaders() {
	c.headerRequests.close()
	c.blockRequests.close()
}

func (c *Client) requestHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	var header *types.Header
	log.Debug("Starting RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", false)
	err := c.client.CallContext(ctx, &header, "eth_getBlockByHash", hash, false)
	if err == nil && header == nil {
		err = ethereum.NotFound
	}
	if err == nil && header.Hash() != hash {
		header, err = nil, errors.New("header hash does not match")
	}
	log.Debug("Finished RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", false, "error", err)
	return header, err
}

func (c *Client) requestBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	var (
		raw   json.RawMessage
		block *types.Block
	)
	log.Debug("Starting RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", true)
	err := c.client.CallContext(ctx, &raw, "eth_getBlockByHash", hash, true)
	log.Debug("Finished RPC request", "type", "eth_getBlockByHash", "hash", hash, "full", true, "error", err)
	if err != nil {
		return nil, err
	}
	block, err = decodeBlock(raw) // returns ethereum.NotFound if block not found
	if err != nil {
		return nil, err
	}
	if block.Hash() != hash {
		return nil, errors.New("block hash does not match")
	}
	return block, nil
}

func (c *Client) getHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if header, ok := c.headerCache.Get(hash); ok {
		return header, nil
	}
	if block, ok := c.blockCache.Get(hash); ok {
		return block.Header(), nil
	}
	if c.blockRequests.has(hash) && !c.headerRequests.has(hash) {
		req := c.blockRequests.request(hash)
		block, err := req.getResult(ctx)
		if err == nil {
			header := block.Header()
			c.headerCache.Add(hash, header)
			c.blockCache.Add(hash, block)
			req.release()
			return header, nil
		} else {
			req.release()
			return nil, err
		}
	}
	req := c.headerRequests.request(hash)
	header, err := req.getResult(ctx)
	if err == nil {
		c.headerCache.Add(hash, header)
	}
	req.release()
	return header, err
}

func (c *Client) getPayloadHeader(hash common.Hash) *btypes.ExecutionHeader {
	pheader, _ := c.payloadHeaderCache.Get(hash)
	return pheader
}

func (c *Client) getBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if block, ok := c.blockCache.Get(hash); ok {
		return block, nil
	}
	req := c.blockRequests.request(hash)
	block, err := req.getResult(ctx)
	if err == nil {
		header := block.Header()
		c.headerCache.Add(hash, header)
		c.headerRequests.tryDeliver(hash, header)
		c.blockCache.Add(hash, block)
		c.cacheBlockTxPositions(block)
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

func (c *Client) addPayloadHeader(header *btypes.ExecutionHeader) {
	c.payloadHeaderCache.Add(header.BlockHash(), header)
}
