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

// recentCanonicalLength specifies the maximum number of most recent number to hash
// associations stored in the recent map
const recentCanonicalLength = 256

// canonicalChainFields defines Client fields related to canonical chain tracking.
type canonicalChainFields struct {
	chainLock            sync.Mutex
	head, finality       *btypes.ExecutionHeader
	recent               map[uint64]common.Hash // nil while head == nil
	recentTail           uint64                 // if recent != nil then recent hashes are available from recentTail to head
	tailFetchCh, closeCh chan struct{}
	cache                *lru.Cache[uint64, common.Hash]  // older than recentTail
	requests             *requestMap[uint64, common.Hash] // requested; neither recent nor cached
}

// initCanonicalChain initializes the structures related to canonical chain tracking.
func (c *Client) initCanonicalChain() {
	c.cache = lru.NewCache[uint64, common.Hash](10000)
	c.requests = newRequestMap[uint64, common.Hash](nil)
	c.tailFetchCh = make(chan struct{})
	c.closeCh = make(chan struct{})
	go c.tailFetcher()
}

// closeCanonicalChain shuts down the structures related to canonical chain tracking.
func (c *Client) closeCanonicalChain() {
	c.requests.close()
	close(c.closeCh)
}

// setHead sets a new head for the canonical chain. It also updates the recent
// hash associations and takes care of state cache invalidation if necessary.
func (c *Client) setHead(head *btypes.ExecutionHeader) bool {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	headNum, headHash := head.BlockNumber(), head.BlockHash()
	if c.head != nil && c.head.BlockHash() == headHash {
		return false
	}
	if c.recent == nil || c.head == nil || c.head.BlockNumber()+1 != headNum || c.head.BlockHash() != head.ParentHash() {
		// new head is not a descendant of the previous one; everything that was
		// not finalized should be invalidated.
		if c.finality == nil || c.recentTail > c.finality.BlockNumber()+1 {
			// purge cache if the previous contents were not all finalized
			c.cache.Purge()
		}
		// state proofs are cached by number
		c.clearStateCache()
		// initialize recent canonical hash map
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
		c.cache.Add(c.recentTail, c.recent[c.recentTail])
		delete(c.recent, c.recentTail)
		c.recentTail++
	}
	c.requests.tryDeliver(headNum, headHash)
	log.Debug("SetHead", "recentTail", c.recentTail, "head", headNum)
	return true
}

// setFinality sets a new finality slot.
func (c *Client) setFinality(finality *btypes.ExecutionHeader) {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	c.finality = finality
	finalNum := finality.BlockNumber()
	if finalNum < c.recentTail {
		c.cache.Add(finalNum, finality.BlockHash())
	}
	c.requests.tryDeliver(finalNum, finality.BlockHash())
}

// getHead returns the current local canonical chain head.
func (c *Client) getHead() *btypes.ExecutionHeader {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	return c.head
}

// getFinality returns the current finality header
func (c *Client) getFinality() *btypes.ExecutionHeader {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	return c.finality
}

// addRecentTail tries to add the given header to the tail of the recent canonical
// section and returns true if successful.
func (c *Client) addRecentTail(tail *types.Header) bool {
	c.chainLock.Lock()
	defer c.chainLock.Unlock()

	if c.recent == nil || c.head == nil || c.recentTail+recentCanonicalLength <= c.head.BlockNumber() ||
		tail.Number.Uint64() != c.recentTail ||
		c.recent[c.recentTail] != tail.Hash() {
		return false
	}
	if c.recentTail > 0 {
		c.recentTail--
		c.recent[c.recentTail] = tail.ParentHash
		c.requests.tryDeliver(c.recentTail, tail.ParentHash)
	}
	return true
}

// tailFetcher reverse syncs canonical hashes until all requested items are present.
func (c *Client) tailFetcher() {
	for {
		c.chainLock.Lock()
		var (
			tailNum, needTail uint64
			tailHash          common.Hash
		)
		if c.recent != nil && c.head != nil {
			tailNum, tailHash = c.recentTail, c.recent[c.recentTail]
			needTail = tailNum
			for _, reqNum := range c.requests.allKeys() {
				if reqNum < needTail {
					needTail = reqNum
				}
			}
			if headNum := c.head.BlockNumber(); needTail+recentCanonicalLength <= headNum {
				needTail = headNum + 1 - recentCanonicalLength
			}
		}
		c.chainLock.Unlock()
		if needTail < tailNum {
			log.Debug("Fetching tail headers", "have", tailNum, "need", needTail)
			ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
			//TODO parallel fetch by number
			if header, err := c.getHeader(ctx, tailHash); err == nil {
				c.addRecentTail(header)
			}
		} else {
			select {
			case <-c.tailFetchCh:
			case <-c.closeCh:
				return
			}
		}
	}
}

// getHash returns the requested number -> hash association.
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
	return req.waitForResult(ctx)
}

// getCachedHash returns the requested number -> hash association if it is already
// in memory cache.
func (c *Client) getCachedHash(number uint64) (common.Hash, bool) {
	c.chainLock.Lock()
	hash, ok := c.recent[number]
	c.chainLock.Unlock()
	if ok {
		return hash, true
	}
	return c.cache.Get(number)
}

// resolveBlockNumber resolves an RPC block number into uint64 and also returns
// the payload header in case of special (negative) RPC numbers.
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

// blockNumberToHash returns the canonical hash belonging to the given RPC block
// number. Note that requesting older canonical hashes might trigger a reverse
// header sync process which might take a very long time depending on the age
// of the specified block.
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

// blockNumberOrHashToHash resolves rpc.BlockNumberOrHash into a block hash.
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
