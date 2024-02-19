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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
)

const recentCanonicalLength = 256

type canonicalChain struct {
	lock           sync.Mutex
	head, finality *types.Header
	recent         map[uint64]common.Hash                // nil until initialized
	recentTail     uint64                                // if recent != nil then recent hashes are available from recentTail to head
	finalized      *lru.Cache[uint64, common.Hash]       // finalized but not recent hashes
	requests       map[uint64]objectRequest[common.Hash] // requested; neither in recent or finalized
	changeCounter  uint64
}

func newCanonicalChain() *canonicalChain {
	return &canonicalChain{
		finalized: lru.NewCache[uint64, common.Hash](10000),
		requests:  make(map[uint64]objectRequest[common.Hash]),
	}
}

func (c *canonicalChain) ChangeCounter() uint64 {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.changeCounter
}

func (c *canonicalChain) requestHash(number uint64) chan common.Hash {
	c.lock.Lock()
	defer c.lock.Unlock()

	if hash, ok := c.recent[number]; ok {
		ch := make(chan common.Hash, 1)
		ch <- hash
		return ch
	}
	if hash, ok := c.finalized.Get(number); ok {
		ch := make(chan common.Hash, 1)
		ch <- hash
		return ch
	}
	c.changeCounter++
	r := c.requests[number]
	if r == nil {
		r = newObjectRequest[common.Hash]()
		c.requests[number] = r
	}
	return r.addRequest()
}

func (c *canonicalChain) cancelRequest(number uint64, ch chan common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()

	r := c.requests[number]
	if r == nil {
		return
	}
	r.cancelRequest(ch)
	if r.isEmpty() {
		delete(c.requests, number)
	}
}

func (c *canonicalChain) setHead(head *types.Header) {
	c.lock.Lock()
	defer c.lock.Unlock()

	headNum, headHash := head.Number.Uint64(), head.Hash()
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

	if r, ok := c.requests[headNum]; ok {
		r.deliver(c.recent[headNum])
		delete(c.requests, headNum)
	}
}

func (c *canonicalChain) setFinality(finality *types.Header) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.finality = finality
	finalNum := finality.Number.Uint64()
	if finalNum < c.recentTail {
		c.finalized.Add(finalNum, finality.Hash())
	}

	if r, ok := c.requests[finalNum]; ok {
		r.deliver(finality.Hash())
		delete(c.requests, finalNum)
	}
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
		if r, ok := c.requests[c.recentTail]; ok {
			r.deliver(c.recent[c.recentTail])
			delete(c.requests, c.recentTail)
		}
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

func (c *canonicalChain) getRequestRange() (targetNum, tailNum uint64, tailHash common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()

	targetNum, tailNum, tailHash = c.recentTail, c.recentTail, c.recent[c.recentTail]
	for num := range c.requests {
		if num < targetNum {
			targetNum = num
		}
	}
	return
}

type blocksAndHeaders struct {
	lock           sync.Mutex
	headerRequests map[common.Hash]objectRequest[*types.Header]
	blockRequests  map[common.Hash]objectRequest[*types.Block]
	headerCache    *lru.Cache[common.Hash, *types.Header]
	blockCache     *lru.Cache[common.Hash, *types.Block]
	requestOrder   []common.Hash
	changeCounter  uint64
}

func newBlocksAndHeaders() *blocksAndHeaders {
	return &blocksAndHeaders{
		headerRequests: make(map[common.Hash]objectRequest[*types.Header]),
		blockRequests:  make(map[common.Hash]objectRequest[*types.Block]),
		headerCache:    lru.NewCache[common.Hash, *types.Header](1000),
		blockCache:     lru.NewCache[common.Hash, *types.Block](10),
	}
}

func (b *blocksAndHeaders) ChangeCounter() uint64 {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.changeCounter
}

func (b *blocksAndHeaders) requestHeader(hash common.Hash) chan *types.Header {
	b.lock.Lock()
	defer b.lock.Unlock()

	if header, ok := b.headerCache.Get(hash); ok {
		ch := make(chan *types.Header, 1)
		ch <- header
		return ch
	}
	b.changeCounter++
	if _, ok := b.headerRequests[hash]; !ok {
		if _, ok := b.blockRequests[hash]; !ok {
			b.requestOrder = append(b.requestOrder, hash)
		}
	}
	return b.headerRequests[hash].addRequest()
}

func (b *blocksAndHeaders) requestBlock(hash common.Hash) chan *types.Block {
	b.lock.Lock()
	defer b.lock.Unlock()

	if block, ok := b.blockCache.Get(hash); ok {
		ch := make(chan *types.Block, 1)
		ch <- block
		return ch
	}
	b.changeCounter++
	if _, ok := b.headerRequests[hash]; !ok {
		if _, ok := b.blockRequests[hash]; !ok {
			b.requestOrder = append(b.requestOrder, hash)
		}
	}
	return b.blockRequests[hash].addRequest()
}

func (b *blocksAndHeaders) cancelRequestHeader(hash common.Hash, ch chan *types.Header) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.headerRequests[hash].cancelRequest(ch)
	if b.headerRequests[hash].isEmpty() {
		delete(b.headerRequests, hash)
	}
}

func (b *blocksAndHeaders) cancelRequestBlock(hash common.Hash, ch chan *types.Block) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.blockRequests[hash].cancelRequest(ch)
	if b.blockRequests[hash].isEmpty() {
		delete(b.blockRequests, hash)
	}
}

func (b *blocksAndHeaders) getRequestLists() (headerReqs, blockReqs []common.Hash) {
	b.lock.Lock()
	defer b.lock.Unlock()

	headerReqs = make([]common.Hash, 0, len(b.requestOrder))
	blockReqs = make([]common.Hash, 0, len(b.requestOrder))
	var rl int
	for rp, hash := range b.requestOrder {
		if _, ok := b.blockRequests[hash]; ok {
			blockReqs = append(blockReqs, hash)
		} else if _, ok := b.headerRequests[hash]; ok {
			headerReqs = append(headerReqs, hash)
		} else {
			continue
		}
		b.requestOrder[rl] = b.requestOrder[rp]
		rp++
	}
	return
}

func (b *blocksAndHeaders) deliverHeader(header *types.Header) {
	b.lock.Lock()
	defer b.lock.Unlock()

	hash := header.Hash()
	b.headerRequests[hash].deliver(header)
	delete(b.headerRequests, hash)
	b.headerCache.Add(hash, header)
}

func (b *blocksAndHeaders) deliverBlock(block *types.Block) {
	b.lock.Lock()
	defer b.lock.Unlock()

	header := block.Header()
	hash := header.Hash()
	b.headerRequests[hash].deliver(header)
	b.blockRequests[hash].deliver(block)
	delete(b.headerRequests, hash)
	delete(b.blockRequests, hash)
	b.headerCache.Add(hash, header)
	b.blockCache.Add(hash, block)
}
