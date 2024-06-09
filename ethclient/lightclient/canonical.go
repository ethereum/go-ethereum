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
	"sync"
	"time"

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
