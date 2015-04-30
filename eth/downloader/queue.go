package downloader

import (
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/fatih/set.v0"
)

// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	hashPool    *set.Set
	fetchPool   *set.Set
	blockHashes *set.Set

	mu       sync.Mutex
	fetching map[string]*chunk

	blockOffset int
	blocks      []*types.Block
}

func newqueue() *queue {
	return &queue{
		hashPool:    set.New(),
		fetchPool:   set.New(),
		blockHashes: set.New(),
		fetching:    make(map[string]*chunk),
	}
}

func (c *queue) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.resetNoTS()
}
func (c *queue) resetNoTS() {
	c.blockOffset = 0
	c.hashPool.Clear()
	c.fetchPool.Clear()
	c.blockHashes.Clear()
	c.blocks = nil
	c.fetching = make(map[string]*chunk)
}

func (c *queue) size() int {
	return c.hashPool.Size() + c.blockHashes.Size() + c.fetchPool.Size()
}

// reserve a `max` set of hashes for `p` peer.
func (c *queue) get(p *peer, max int) *chunk {
	c.mu.Lock()
	defer c.mu.Unlock()

	// return nothing if the pool has been depleted
	if c.hashPool.Size() == 0 {
		return nil
	}

	limit := int(math.Min(float64(max), float64(c.hashPool.Size())))
	// Create a new set of hashes
	hashes, i := set.New(), 0
	c.hashPool.Each(func(v interface{}) bool {
		// break on limit
		if i == limit {
			return false
		}
		// skip any hashes that have previously been requested from the peer
		if p.ignored.Has(v) {
			return true
		}

		hashes.Add(v)
		i++

		return true
	})
	// if no hashes can be requested return a nil chunk
	if hashes.Size() == 0 {
		return nil
	}

	// remove the fetchable hashes from hash pool
	c.hashPool.Separate(hashes)
	c.fetchPool.Merge(hashes)

	// Create a new chunk for the seperated hashes. The time is being used
	// to reset the chunk (timeout)
	chunk := &chunk{p, hashes, time.Now()}
	// register as 'fetching' state
	c.fetching[p.id] = chunk

	// create new chunk for peer
	return chunk
}

func (c *queue) has(hash common.Hash) bool {
	return c.hashPool.Has(hash) || c.fetchPool.Has(hash) || c.blockHashes.Has(hash)
}

func (c *queue) addBlock(id string, block *types.Block) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// when adding a block make sure it doesn't already exist
	if !c.blockHashes.Has(block.Hash()) {
		c.hashPool.Remove(block.Hash())
		c.blocks = append(c.blocks, block)
	}
}

func (c *queue) getBlock(hash common.Hash) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.blockHashes.Has(hash) {
		return nil
	}

	for _, block := range c.blocks {
		if block.Hash() == hash {
			return block
		}
	}
	return nil
}

// deliver delivers a chunk to the queue that was requested of the peer
func (c *queue) deliver(id string, blocks []*types.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	chunk := c.fetching[id]
	// If the chunk was never requested simply ignore it
	if chunk != nil {
		delete(c.fetching, id)
		// check the length of the returned blocks. If the length of blocks is 0
		// we'll assume the peer doesn't know about the chain.
		if len(blocks) == 0 {
			// So we can ignore the blocks we didn't know about
			chunk.peer.ignored.Merge(chunk.hashes)
		}

		// seperate the blocks and the hashes
		blockHashes := chunk.fetchedHashes(blocks)
		// merge block hashes
		c.blockHashes.Merge(blockHashes)
		// Add the blocks
		for _, block := range blocks {
			// See (1) for future limitation
			n := int(block.NumberU64()) - c.blockOffset
			if n > len(c.blocks) || n < 0 {
				return errBlockNumberOverflow
			}
			c.blocks[n] = block
		}
		// Add back whatever couldn't be delivered
		c.hashPool.Merge(chunk.hashes)
		c.fetchPool.Separate(chunk.hashes)
	}

	return nil
}

func (c *queue) alloc(offset, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.blockOffset < offset {
		c.blockOffset = offset
	}

	// (1) XXX at some point we could limit allocation to memory and use the disk
	// to store future blocks.
	if len(c.blocks) < size {
		c.blocks = append(c.blocks, make([]*types.Block, size)...)
	}
}

// puts puts sets of hashes on to the queue for fetching
func (c *queue) put(hashes *set.Set) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hashPool.Merge(hashes)
}

type chunk struct {
	peer   *peer
	hashes *set.Set
	itime  time.Time
}

func (ch *chunk) fetchedHashes(blocks []*types.Block) *set.Set {
	fhashes := set.New()
	for _, block := range blocks {
		fhashes.Add(block.Hash())
	}
	ch.hashes.Separate(fhashes)

	return fhashes
}
