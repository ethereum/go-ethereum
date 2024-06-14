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

	"github.com/ethereum/go-ethereum"
	btypes "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// blocksAndHeadersFields defines Client fields related to blocks and headers.
type blocksAndHeadersFields struct {
	headerCache        *lru.Cache[common.Hash, *types.Header]
	headerRequests     *requestMap[common.Hash, *types.Header]
	payloadHeaderCache *lru.Cache[common.Hash, *btypes.ExecutionHeader]
	blockCache         *lru.Cache[common.Hash, *types.Block]
	blockRequests      *requestMap[common.Hash, *types.Block]
}

// initBlocksAndHeaders initializes the structures related to blocks and headers.
func (c *Client) initBlocksAndHeaders() {
	c.headerCache = lru.NewCache[common.Hash, *types.Header](1000)
	c.payloadHeaderCache = lru.NewCache[common.Hash, *btypes.ExecutionHeader](1000)
	c.blockCache = lru.NewCache[common.Hash, *types.Block](10)
	c.headerRequests = newRequestMap[common.Hash, *types.Header](c.requestHeader)
	c.blockRequests = newRequestMap[common.Hash, *types.Block](c.requestBlock)
}

// closeBlocksAndHeaders shuts down the structures related to blocks and headers.
func (c *Client) closeBlocksAndHeaders() {
	c.headerRequests.close()
	c.blockRequests.close()
}

// getHeader returns the header with the given block hash.
func (c *Client) getHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if header, ok := c.headerCache.Get(hash); ok {
		return header, nil
	}
	if block, ok := c.blockCache.Get(hash); ok {
		return block.Header(), nil
	}
	if c.blockRequests.has(hash) && !c.headerRequests.has(hash) {
		req := c.blockRequests.request(hash)
		block, err := req.waitForResult(ctx)
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
	header, err := req.waitForResult(ctx)
	if err == nil {
		c.headerCache.Add(hash, header)
	}
	req.release()
	return header, err
}

// getPayloadHeader returns the payload header with the given block hash.
// Note that types.Header and the CL execution payload header (btypes.ExecutionHeader)
// both contain the information relevant to the light client but they are not
// interchangeable. Payload headers are received from head/finality updates and
// are cached but they are not obtainable later on demand as they are not directly
// chained to each other by hash reference. If reverse syncing is required then
// EL headers should be used.
func (c *Client) getPayloadHeader(hash common.Hash) *btypes.ExecutionHeader {
	pheader, _ := c.payloadHeaderCache.Get(hash)
	return pheader
}

// addPayloadHeader caches the given payload header.
func (c *Client) addPayloadHeader(header *btypes.ExecutionHeader) {
	c.payloadHeaderCache.Add(header.BlockHash(), header)
}

// getBlock returns the block with the given hash. If the block has been retrieved
// from the server then the transaction inclusion positions are also cached.
func (c *Client) getBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if block, ok := c.blockCache.Get(hash); ok {
		return block, nil
	}
	req := c.blockRequests.request(hash)
	block, err := req.waitForResult(ctx)
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

// requestHeader requests the header with the specified hash from the RPC client.
// Either the header with the given hash or an error is returned.
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

// requestBlock requests the block with the specified hash from the RPC client.
// Either the block with the given hash or an error is returned.
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

// decodeBlock decodes a JSON encoded block.
//TODO de-duplicate
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
