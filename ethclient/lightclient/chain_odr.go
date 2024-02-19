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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type chainOdr struct {
	headTracker           *light.HeadTracker
	canonicalChain        *canonicalChain
	blocksAndHeaders      *blocksAndHeaders
	headerLock, blockLock map[common.Hash]request.ServerAndID
}

func newChainOdr(headTracker *light.HeadTracker, canonicalChain *canonicalChain, blocksAndHeaders *blocksAndHeaders) *chainOdr {
	return &chainOdr{
		headTracker:      headTracker,
		canonicalChain:   canonicalChain,
		blocksAndHeaders: blocksAndHeaders,
		headerLock:       make(map[common.Hash]request.ServerAndID),
		blockLock:        make(map[common.Hash]request.ServerAndID), // also locks header requests
	}
}

func (c *chainOdr) Process(requester request.Requester, events []request.Event) {
	if optimistic, ok := c.headTracker.ValidatedOptimistic(); ok {
		head := optimistic.Attested.ExecHeader()
		c.canonicalChain.setHead(head)
		c.blocksAndHeaders.deliverHeader(head)
	}
	if finality, ok := c.headTracker.ValidatedFinality(); ok {
		finalized := finality.Finalized.ExecHeader()
		c.canonicalChain.setFinality(finalized)
		c.blocksAndHeaders.deliverHeader(finalized)
	}

	for _, event := range events {
		if !event.IsRequestEvent() {
			continue
		}
		sid, req, resp := event.RequestInfo()
		switch data := req.(type) {
		case ReqHeader:
			reqHash := common.Hash(data)
			if c.headerLock[reqHash] == sid {
				delete(c.headerLock, reqHash)
			}
			if resp != nil {
				if header := resp.(*types.Header); header.Hash() == reqHash {
					c.blocksAndHeaders.deliverHeader(header)
				} else {
					requester.Fail(event.Server, "invalid header")
				}
			}
		case ReqBlock:
			reqHash := common.Hash(data)
			if c.headerLock[reqHash] == sid {
				delete(c.blockLock, reqHash)
			}
			if resp != nil {
				if block := resp.(*types.Block); block.Hash() == reqHash {
					c.blocksAndHeaders.deliverBlock(block)
				} else {
					requester.Fail(event.Server, "invalid block")
				}
			}
		default:
			panic("Request event for unknown request type received")
		}
	}

	headerReqs, blockReqs := c.blocksAndHeaders.getRequestLists()
	var servers []request.Server

	for _, hash := range headerReqs {
		if _, ok := c.headerLock[hash]; ok {
			continue
		}
		if _, ok := c.blockLock[hash]; ok {
			continue
		}
		if len(servers) == 0 {
			servers = requester.CanSendTo()
			if len(servers) == 0 {
				return
			}
		}
		server := servers[0]
		servers = servers[1:]
		id := requester.Send(server, ReqHeader(hash))
		c.headerLock[hash] = request.ServerAndID{Server: server, ID: id}
	}

	for _, hash := range blockReqs {
		if _, ok := c.blockLock[hash]; ok {
			continue
		}
		if len(servers) == 0 {
			servers = requester.CanSendTo()
			if len(servers) == 0 {
				return
			}
		}
		server := servers[0]
		servers = servers[1:]
		id := requester.Send(server, ReqBlock(hash))
		c.headerLock[hash] = request.ServerAndID{Server: server, ID: id}
	}

	//TODO reverse sync canonical headers if requested and not cached from head updates
}

func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	ch := c.blocksAndHeaders.requestBlock(hash)
	select {
	case block := <-ch:
		return block, nil
	case <-ctx.Done():
		c.blocksAndHeaders.cancelRequestBlock(hash, ch)
		return nil, ctx.Err()
	}
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
	ch := c.canonicalChain.requestHash(uint64(num))
	select {
	case hash := <-ch:
		return hash, nil
	case <-ctx.Done():
		c.canonicalChain.cancelRequest(uint64(num), ch)
		return common.Hash{}, ctx.Err()
	}
}

func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	hash, err := c.numberToHash(ctx, number)
	if err != nil {
		return nil, err
	}
	return c.BlockByHash(ctx, hash)
}

func (c *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	ch := c.blocksAndHeaders.requestHeader(hash)
	select {
	case header := <-ch:
		return header, nil
	case <-ctx.Done():
		c.blocksAndHeaders.cancelRequestHeader(hash, ch)
		return nil, ctx.Err()
	}
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
	return nil, nil //TODO
}
