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
	"errors"
	"math/big"
	ssync "sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/config"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/api"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	canonicalChainFields
	blocksAndHeadersFields
	lightStateFields
	txAndReceiptsFields

	clConfig config.LightClientConfig
	elConfig *params.ChainConfig

	client      *rpc.Client
	scheduler   *request.Scheduler
	headTracker *light.HeadTracker

	headSubLock      ssync.Mutex
	headSubs         map[*headSub]struct{}
	cancelHeadFetch  func()
	headFetchCounter int
}

func NewClient(clConfig config.LightClientConfig, elConfig *params.ChainConfig, db ethdb.KeyValueStore, rpcClient *rpc.Client) *Client {
	// create data structures
	committeeChain := light.NewCommitteeChain(db, clConfig.ChainConfig, clConfig.SignerThreshold, clConfig.EnforceTime)
	client := &Client{
		client:      rpcClient,
		clConfig:    clConfig,
		elConfig:    elConfig,
		scheduler:   request.NewScheduler(),
		headTracker: light.NewHeadTracker(committeeChain, clConfig.SignerThreshold),
		headSubs:    make(map[*headSub]struct{}),
	}
	client.initBlocksAndHeaders()
	client.initCanonicalChain()
	client.initTxAndReceipts()
	client.initLightState()

	checkpointInit := sync.NewCheckpointInit(committeeChain, clConfig.Checkpoint)
	forwardSync := sync.NewForwardUpdateSync(committeeChain)
	headSync := sync.NewHeadSync(client.headTracker, committeeChain)
	client.scheduler.RegisterTarget(client.headTracker)
	client.scheduler.RegisterTarget(committeeChain)
	client.scheduler.RegisterModule(checkpointInit, "checkpointInit")
	client.scheduler.RegisterModule(forwardSync, "forwardSync")
	client.scheduler.RegisterModule(headSync, "headSync")
	client.scheduler.RegisterModule(client, "canonicalChain")
	return client
}

func (c *Client) Start() {
	c.scheduler.Start()
	for _, url := range c.clConfig.ApiUrls {
		beaconApi := api.NewBeaconLightApi(url, c.clConfig.CustomHeader)
		c.scheduler.RegisterServer(request.NewServer(api.NewApiServer(beaconApi), &mclock.System{}))
	}
}

func (c *Client) Stop() {
	c.scheduler.Stop()
}

// ChainReader interface

func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return c.getBlock(ctx, hash)
}

func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	hash, err := c.blockNumberToHash(ctx, number)
	if err != nil {
		return nil, err
	}
	return c.BlockByHash(ctx, hash)
}

func (c *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return c.getHeader(ctx, hash)
}

func (c *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	hash, err := c.blockNumberToHash(ctx, number)
	if err != nil {
		return nil, err
	}
	return c.HeaderByHash(ctx, hash)
}

func (c *Client) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	block, err := c.BlockByHash(ctx, blockHash)
	if err != nil {
		return 0, err
	}
	return uint(len(block.Transactions())), nil
}

func (c *Client) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	block, err := c.BlockByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	if index >= uint(len(txs)) {
		return nil, errors.New("Invalid transaction index")
	}
	return txs[index], nil
}

func (c *Client) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	sub := &headSub{
		client: c,
		headCh: ch,
		errCh:  make(chan error, 1),
	}
	c.headSubLock.Lock()
	c.headSubs[sub] = struct{}{}
	c.headSubLock.Unlock()
	return sub, nil
}

func (c *Client) processNewHead(number uint64, hash common.Hash) {
	log.Trace("New execution payload header received", "hash", hash)
	c.txAndReceiptsNewHead(number, hash)
	ctx, cancel := context.WithCancel(context.Background())
	c.headSubLock.Lock()
	if len(c.headSubs) == 0 {
		c.headSubLock.Unlock()
		return
	}
	if c.cancelHeadFetch != nil {
		c.cancelHeadFetch()
	}
	c.cancelHeadFetch = cancel
	c.headFetchCounter++
	hfc := c.headFetchCounter
	c.headSubLock.Unlock()

	head, err := c.getHeader(ctx, hash)
	c.headSubLock.Lock()
	if c.headFetchCounter == hfc {
		c.cancelHeadFetch = nil
	}
	if err == nil {
		for sub := range c.headSubs {
			sub.headCh <- head
		}
	}
	c.headSubLock.Unlock()
}

func (c *Client) unsubscribeNewHead(sub *headSub) {
	c.headSubLock.Lock()
	delete(c.headSubs, sub)
	c.headSubLock.Unlock()
}

type headSub struct {
	client *Client
	headCh chan<- *types.Header
	errCh  chan error
}

func (h *headSub) Unsubscribe() {
	h.client.unsubscribeNewHead(h)
	close(h.errCh)
}

func (h *headSub) Err() <-chan error {
	return h.errCh
}

// TransactionReader interface

func (c *Client) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return c.getTxByHash(ctx, txHash)
}

func (c *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return c.getReceiptByTxHash(ctx, txHash)
}

// ChainStateReader interface

func (c *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	proof, _, err := c.getProof(ctx, blockNumber, account, nil, false)
	if err != nil {
		return nil, err
	}
	return proof.Balance, nil
}

func (c *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	proof, _, err := c.getProof(ctx, blockNumber, account, []string{key.Hex()}, false)
	if err != nil {
		return nil, err
	}
	return stValueBytes(proof.StorageProof[0].Value)
}

func (c *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	_, code, err := c.getProof(ctx, blockNumber, account, nil, true)
	return code, err
}

func (c *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	if blockNumber.IsInt64() && rpc.BlockNumber(blockNumber.Int64()) == rpc.PendingBlockNumber {
		return c.PendingNonceAt(ctx, account)
	}
	proof, _, err := c.getProof(ctx, blockNumber, account, nil, false)
	if err != nil {
		return 0, err
	}
	return proof.Nonce, nil
}

func (c *Client) BlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	hash, err := c.blockNumberOrHashToHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	return c.getBlockReceipts(ctx, hash)
}

// TransactionSender interface

func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return c.sendTransaction(ctx, tx)
}

// PendingStateReader interface

func (c *Client) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	return c.BalanceAt(ctx, account, big.NewInt(int64(rpc.LatestBlockNumber)))
	//TODO subtract upper estimate for pending tx costs?
}

func (c *Client) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	return c.StorageAt(ctx, account, key, big.NewInt(int64(rpc.LatestBlockNumber)))
}

func (c *Client) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return c.CodeAt(ctx, account, big.NewInt(int64(rpc.LatestBlockNumber)))
}

func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	head := c.getHead()
	if head == nil {
		return 0, errors.New("chain head unknown")
	}
	headNonce, pendingTxs, err := c.nonceAndPendingTxs(ctx, head, account)
	if err != nil {
		return 0, err
	}
	if len(pendingTxs) == 0 {
		return headNonce, nil
	}
	return pendingTxs[len(pendingTxs)-1].Nonce(), nil
}

func (c *Client) PendingTransactionCount(ctx context.Context) (uint, error) {
	head := c.getHead()
	if head == nil {
		return 0, errors.New("chain head unknown")
	}
	allSenders := c.allSenders()
	countCh := make(chan uint, len(allSenders))
	errCh := make(chan error, len(allSenders))
	for _, sender := range allSenders {
		go func() {
			_, pendingTxs, err := c.nonceAndPendingTxs(ctx, head, sender)
			errCh <- err
			countCh <- uint(len(pendingTxs))
		}()
	}
	var count uint
	for range allSenders {
		if err := <-errCh; err != nil {
			return 0, err
		}
		count += <-countCh
	}
	return count, nil
}

// BlockNumberReader interface

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	if head := c.getHead(); head != nil {
		return head.BlockNumber(), nil
	}
	return 0, errors.New("chain head unknown")
}
