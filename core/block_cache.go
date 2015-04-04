package core

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BlockCache implements a caching mechanism specifically for blocks and uses FILO to pop
type BlockCache struct {
	size int

	hashes []common.Hash
	blocks map[common.Hash]*types.Block

	mu sync.RWMutex
}

// Creates and returns a `BlockCache` with `size`. If `size` is smaller than 1 it will panic
func NewBlockCache(size int) *BlockCache {
	if size < 1 {
		panic("block cache size not allowed to be smaller than 1")
	}

	bc := &BlockCache{size: size}
	bc.Clear()
	return bc
}

func (bc *BlockCache) Clear() {
	bc.blocks = make(map[common.Hash]*types.Block)
	bc.hashes = nil

}

func (bc *BlockCache) Push(block *types.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.hashes) == bc.size {
		delete(bc.blocks, bc.hashes[0])

		// XXX There are a few other options on solving this
		// 1) use a poller / GC like mechanism to clean up untracked objects
		// 2) copy as below
		// re-use the slice and remove the reference to bc.hashes[0]
		// this will allow the element to be garbage collected.
		copy(bc.hashes, bc.hashes[1:])
	} else {
		bc.hashes = append(bc.hashes, common.Hash{})
	}

	hash := block.Hash()
	bc.blocks[hash] = block
	bc.hashes[len(bc.hashes)-1] = hash
}

func (bc *BlockCache) Delete(hash common.Hash) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if _, ok := bc.blocks[hash]; ok {
		delete(bc.blocks, hash)
		for i, h := range bc.hashes {
			if hash == h {
				bc.hashes = bc.hashes[:i+copy(bc.hashes[i:], bc.hashes[i+1:])]
				// or ? => bc.hashes = append(bc.hashes[:i], bc.hashes[i+1]...)

				break
			}
		}
	}
}

func (bc *BlockCache) Get(hash common.Hash) *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if block, haz := bc.blocks[hash]; haz {
		return block
	}

	return nil
}

func (bc *BlockCache) Has(hash common.Hash) bool {
	_, ok := bc.blocks[hash]
	return ok
}

func (bc *BlockCache) Each(cb func(int, *types.Block)) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	i := 0
	for _, block := range bc.blocks {
		cb(i, block)
		i++
	}
}
