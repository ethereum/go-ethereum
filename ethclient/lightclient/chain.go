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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const recentCanonicalLength = 256

type canonicalChain struct {
	lock             sync.Mutex
	headTracker      *light.HeadTracker
	blocksAndHeaders *blocksAndHeaders
	newHeadCb        func(*types.Header)

	head, finality *types.Header
	recent         map[uint64]common.Hash           // nil until initialized
	recentTail     uint64                           // if recent != nil then recent hashes are available from recentTail to head
	finalized      *lru.Cache[uint64, common.Hash]  // finalized but not recent hashes
	requests       *requestMap[uint64, common.Hash] // requested; neither recent nor finalized
}

func newCanonicalChain(headTracker *light.HeadTracker, blocksAndHeaders *blocksAndHeaders, newHeadCb func(*types.Header)) *canonicalChain {
	return &canonicalChain{
		headTracker:      headTracker,
		blocksAndHeaders: blocksAndHeaders,
		newHeadCb:        newHeadCb,
		finalized:        lru.NewCache[uint64, common.Hash](10000),
		requests:         newRequestMap[uint64, common.Hash](),
	}
}

// Process implements request.Module in order to get notified about new heads.
func (c *canonicalChain) Process(requester request.Requester, events []request.Event) {
	if finality, ok := c.headTracker.ValidatedFinality(); ok {
		finalized := finality.Finalized.ExecHeader()
		c.setFinality(finalized)
		c.blocksAndHeaders.addHeader(finalized)
	}
	if optimistic, ok := c.headTracker.ValidatedOptimistic(); ok {
		head := optimistic.Attested.ExecHeader()
		c.blocksAndHeaders.addHeader(head)
		if c.setHead(head) {
			c.newHeadCb(head)
		}
	}
}

func (c *canonicalChain) getHash(ctx context.Context, number uint64) (common.Hash, error) {
	c.lock.Lock()
	if hash, ok := c.recent[number]; ok {
		c.lock.Unlock()
		return hash, nil
	}
	if hash, ok := c.finalized.Get(number); ok {
		c.lock.Unlock()
		return hash, nil
	}
	ch, _ := c.requests.add(number)
	c.lock.Unlock()
	return c.requests.waitForValue(ctx, number, ch)
}

func (c *canonicalChain) setHead(head *types.Header) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	headNum, headHash := head.Number.Uint64(), head.Hash()
	if c.head != nil && c.head.Hash() == headHash {
		return false
	}
	if c.recent == nil || c.head == nil || c.head.Number.Uint64()+1 != headNum || headHash != head.ParentHash {
		c.recent = make(map[uint64]common.Hash)
		if headNum > 0 {
			c.recent[headNum-1] = head.ParentHash
			c.recentTail = headNum - 1
		} else {
			c.recentTail = 0
		}
	}
	c.head = head
	c.recent[headNum] = headHash
	for headNum >= c.recentTail+recentCanonicalLength {
		if c.finality != nil && c.recentTail <= c.finality.Number.Uint64() {
			c.finalized.Add(c.recentTail, c.recent[c.recentTail])
		}
		delete(c.recent, c.recentTail)
		c.recentTail++
	}
	c.requests.deliver(headNum, headHash, nil)
	return true
}

func (c *canonicalChain) setFinality(finality *types.Header) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.finality = finality
	finalNum := finality.Number.Uint64()
	if finalNum < c.recentTail {
		c.finalized.Add(finalNum, finality.Hash())
	}
	c.requests.deliver(finalNum, finality.Hash(), nil)
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
		c.requests.deliver(c.recentTail, tail.ParentHash, nil)
	}
	return true
}

func (c *canonicalChain) getHead() *types.Header {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.head
}

func (c *canonicalChain) getFinality() *types.Header {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.finality
}

type blocksAndHeaders struct {
	lock           sync.Mutex
	client         *rpc.Client
	headerCache    *lru.Cache[common.Hash, *types.Header]
	headerRequests *requestMap[common.Hash, *types.Header]
	blockCache     *lru.Cache[common.Hash, *types.Block]
	blockRequests  *requestMap[common.Hash, *types.Block]
}

func newBlocksAndHeaders(client *rpc.Client) *blocksAndHeaders {
	return &blocksAndHeaders{
		client:         client,
		headerCache:    lru.NewCache[common.Hash, *types.Header](1000),
		headerRequests: newRequestMap[common.Hash, *types.Header](),
		blockCache:     lru.NewCache[common.Hash, *types.Block](10),
		blockRequests:  newRequestMap[common.Hash, *types.Block](),
	}
}

func (b *blocksAndHeaders) getHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	b.lock.Lock()
	if header, ok := b.headerCache.Get(hash); ok {
		b.lock.Unlock()
		return header, nil
	}
	if block, ok := b.blockCache.Get(hash); ok {
		b.lock.Unlock()
		return block.Header(), nil
	}
	if b.blockRequests.has(hash) && !b.headerRequests.has(hash) {
		ch, request := b.blockRequests.add(hash)
		if !request {
			b.lock.Unlock()
			block, err := b.blockRequests.waitForValue(ctx, hash, ch)
			if err == nil {
				return block.Header(), nil
			}
			return nil, err
		}
		b.blockRequests.remove(hash, ch)
	}
	ch, request := b.headerRequests.add(hash)
	if request {
		go func() {
			var header *types.Header
			err := b.client.CallContext(b.headerRequests.requestContext(hash), &header, "eth_getBlockByHash", hash, false)
			b.headerRequests.deliver(hash, header, err)
		}()
	}
	b.lock.Unlock()
	return b.headerRequests.waitForValue(ctx, hash, ch)
}

func (b *blocksAndHeaders) getBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	b.lock.Lock()
	if block, ok := b.blockCache.Get(hash); ok {
		b.lock.Unlock()
		return block, nil
	}
	ch, request := b.blockRequests.add(hash)
	if request {
		go func() {
			var (
				raw   json.RawMessage
				block *types.Block
			)
			err := b.client.CallContext(b.headerRequests.requestContext(hash), &raw, "eth_getBlockByHash", hash, true)
			if err == nil {
				block, err = decodeBlock(raw)
			}
			b.blockRequests.deliver(hash, block, err)
		}()
	}
	b.lock.Unlock()
	return b.blockRequests.waitForValue(ctx, hash, ch)
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
	return types.NewBlockWithHeader(head).WithBody(txs, nil).WithWithdrawals(body.Withdrawals), nil
}

func (b *blocksAndHeaders) addHeader(header *types.Header) {
	b.lock.Lock()
	defer b.lock.Unlock()

	hash := header.Hash()
	b.headerRequests.deliver(hash, header, nil)
	b.headerCache.Add(hash, header)
}

func (b *blocksAndHeaders) addBlock(block *types.Block) {
	b.lock.Lock()
	defer b.lock.Unlock()

	header := block.Header()
	hash := header.Hash()
	b.headerRequests.deliver(hash, header, nil)
	b.headerCache.Add(hash, header)
	b.blockRequests.deliver(hash, block, nil)
	b.blockCache.Add(hash, block)
}
