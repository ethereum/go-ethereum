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
	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	config           config.LightClientConfig
	scheduler        *request.Scheduler
	canonicalChain   *canonicalChain
	blocksAndHeaders *blocksAndHeaders
	state            *lightState
	headSubLock      ssync.Mutex
	headSubs         map[*headSub]struct{}
	cancelHeadFetch  func()
	headFetchCounter int
}

func NewClient(config config.LightClientConfig, db ethdb.KeyValueStore, rpcClient *rpc.Client) *Client {
	// create data structures
	var (
		committeeChain = light.NewCommitteeChain(db, config.ChainConfig, config.SignerThreshold, config.EnforceTime)
		headTracker    = light.NewHeadTracker(committeeChain, config.SignerThreshold)
	)
	// set up scheduler and sync modules
	//chainHeadFeed := new(event.Feed)
	scheduler := request.NewScheduler()
	blocksAndHeaders := newBlocksAndHeaders(rpcClient)
	client := &Client{
		config:           config,
		scheduler:        scheduler,
		blocksAndHeaders: blocksAndHeaders,
		headSubs:         make(map[*headSub]struct{}),
	}
	canonicalChain := newCanonicalChain(headTracker, blocksAndHeaders, client.newHead)
	client.canonicalChain = canonicalChain
	client.state = newLightState(rpcClient, canonicalChain, blocksAndHeaders)

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
	for _, url := range c.config.ApiUrls {
		beaconApi := api.NewBeaconLightApi(url, c.config.CustomHeader)
		c.scheduler.RegisterServer(request.NewServer(api.NewApiServer(beaconApi), &mclock.System{}))
	}
}

func (c *Client) Stop() {
	c.scheduler.Stop()
}

func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return c.blocksAndHeaders.getBlock(ctx, hash)
}

func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	hash, err := c.canonicalChain.blockNumberToHash(ctx, number)
	if err != nil {
		return nil, err
	}
	return c.BlockByHash(ctx, hash)
}

func (c *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return c.blocksAndHeaders.getHeader(ctx, hash)
}

func (c *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	hash, err := c.canonicalChain.blockNumberToHash(ctx, number)
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

func (c *Client) newHead(hash common.Hash) {
	go func() {
		log.Trace("New execution payload header received", "hash", hash)
		ctx, cancel := context.WithCancel(context.Background())
		c.headSubLock.Lock()
		if c.cancelHeadFetch != nil {
			c.cancelHeadFetch()
		}
		c.cancelHeadFetch = cancel
		c.headFetchCounter++
		hfc := c.headFetchCounter
		c.headSubLock.Unlock()

		head, err := c.blocksAndHeaders.getHeader(ctx, hash)
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
	}()
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

func (c *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	proof, _, err := c.state.getProof(ctx, blockNumber, account, nil, false)
	if err != nil {
		return nil, err
	}
	return proof.Balance, nil
}

func (c *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	proof, _, err := c.state.getProof(ctx, blockNumber, account, []string{key.Hex()}, false) //TODO hashed key?
	if err != nil {
		return nil, err
	}
	return stValueBytes(proof.StorageProof[0].Value)
}

func (c *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	_, code, err := c.state.getProof(ctx, blockNumber, account, nil, true)
	return code, err
}

func (c *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	proof, _, err := c.state.getProof(ctx, blockNumber, account, nil, false)
	if err != nil {
		return 0, err
	}
	return proof.Nonce, nil
}
