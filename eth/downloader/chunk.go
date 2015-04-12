package downloader

import (
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/fatih/set.v0"
)

// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	hashPool *set.Set

	mu       sync.Mutex
	fetching map[string]*chunk
	blocks   []*types.Block
}

func newqueue() *queue {
	return &queue{
		hashPool: set.New(),
		fetching: make(map[string]*chunk),
	}
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
		if i == limit {
			return false
		}

		hashes.Add(v)
		i++

		return true
	})
	// remove the fetchable hashes from hash pool
	c.hashPool.Separate(hashes)
	// Create a new chunk for the seperated hashes. The time is being used
	// to reset the chunk (timeout)
	chunk := &chunk{hashes, time.Now()}
	// register as 'fetching' state
	c.fetching[p.id] = chunk

	// create new chunk for peer
	return chunk
}

func (c *queue) deliver(id string, blocks []*types.Block) {
	c.mu.Lock()
	defer c.mu.Unlock()

	chunk := c.fetching[id]
	// If the chunk was never requested simply ignore it
	if chunk != nil {
		delete(c.fetching, id)

		// seperate the blocks and the hashes
		chunk.seperate(blocks)
		// Add the blocks
		c.blocks = append(c.blocks, blocks...)

		// Add back whatever couldn't be delivered
		c.hashPool.Merge(chunk.hashes)
	}
}

func (c *queue) put(hashes *set.Set) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hashPool.Merge(hashes)
}

type chunk struct {
	hashes *set.Set
	itime  time.Time
}

func (ch *chunk) seperate(blocks []*types.Block) {
	for _, block := range blocks {
		ch.hashes.Remove(block.Hash())
	}
}
