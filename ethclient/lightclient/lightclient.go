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
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	scheduler        *request.Scheduler
	canonicalChain   *canonicalChain
	blocksAndHeaders *blocksAndHeaders
	headSubLock      ssync.Mutex
	headSubs         map[*headSub]struct{}
}

func NewClient(config light.ClientConfig, db ethdb.Database, rpcClient *rpc.Client) *Client {
	// create data structures
	var (
		committeeChain = light.NewCommitteeChain(db, config)
		headTracker    = light.NewHeadTracker(committeeChain, config.Threshold)
	)
	// set up scheduler and sync modules
	//chainHeadFeed := new(event.Feed)
	scheduler := request.NewScheduler()
	blocksAndHeaders := newBlocksAndHeaders(rpcClient)
	client := &Client{
		scheduler:        scheduler,
		blocksAndHeaders: blocksAndHeaders,
		headSubs:         make(map[*headSub]struct{}),
	}
	canonicalChain := newCanonicalChain(headTracker, blocksAndHeaders, client.broadcastNewHead)
	client.canonicalChain = canonicalChain

	checkpointInit := sync.NewCheckpointInit(committeeChain, config.Checkpoint)
	forwardSync := sync.NewForwardUpdateSync(committeeChain)
	headSync := sync.NewHeadSync(headTracker, committeeChain)
	scheduler.RegisterTarget(headTracker)
	scheduler.RegisterTarget(committeeChain)
	scheduler.RegisterModule(checkpointInit, "checkpointInit")
	scheduler.RegisterModule(forwardSync, "forwardSync")
	scheduler.RegisterModule(headSync, "headSync")
	scheduler.RegisterModule(client.canonicalChain, "canonicalChain")
	return client
}

func (c *Client) Start() {
	c.scheduler.Start()
}

func (c *Client) Stop() {
	c.scheduler.Stop()
}

func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return c.blocksAndHeaders.getBlock(ctx, hash)
}

func (c *Client) numberToHash(ctx context.Context, number *big.Int) (common.Hash, error) {
	if !number.IsInt64() {
		return common.Hash{}, errors.New("Invalid block number")
	}
	num := number.Int64()
	if num < 0 {
		switch rpc.BlockNumber(num) {
		case rpc.SafeBlockNumber, rpc.FinalizedBlockNumber:
			if header := c.canonicalChain.getFinality(); header != nil {
				return header.Hash(), nil
			}
			return common.Hash{}, errors.New("Finalized block unknown")
		case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
			if header := c.canonicalChain.getHead(); header != nil {
				return header.Hash(), nil
			}
			return common.Hash{}, errors.New("Head block unknown")
		default:
			return common.Hash{}, errors.New("Invalid block number")
		}
	}
	return c.canonicalChain.getHash(ctx, uint64(num))
}

func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	hash, err := c.numberToHash(ctx, number)
	if err != nil {
		return nil, err
	}
	return c.BlockByHash(ctx, hash)
}

func (c *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return c.blocksAndHeaders.getHeader(ctx, hash)
}

func (c *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	hash, err := c.numberToHash(ctx, number)
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

func (c *Client) broadcastNewHead(head *types.Header) {
	c.headSubLock.Lock()
	for sub := range c.headSubs {
		sub.headCh <- head
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
