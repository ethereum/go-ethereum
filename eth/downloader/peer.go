package downloader

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const (
	workingState = 2
	idleState    = 4
)

// peer represents an active peer
type peer struct {
	state int

	mu         sync.RWMutex
	id         string
	td         *big.Int
	recentHash common.Hash

	getHashes hashFetcherFn
	getBlocks blockFetcherFn
}

// create a new peer
func newPeer(id string, td *big.Int, hash common.Hash, getHashes hashFetcherFn, getBlocks blockFetcherFn) *peer {
	return &peer{id: id, td: td, recentHash: hash, getHashes: getHashes, getBlocks: getBlocks, state: idleState}
}

// fetch a chunk using the peer
func (p *peer) fetch(chunk *chunk) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// set working state
	p.state = workingState
	// convert the set to a fetchable slice
	hashes, i := make([]common.Hash, chunk.hashes.Size()), 0
	chunk.hashes.Each(func(v interface{}) bool {
		hashes[i] = v.(common.Hash)
		i++
		return true
	})
	p.getBlocks(hashes)
}
