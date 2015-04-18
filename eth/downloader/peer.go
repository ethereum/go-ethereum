package downloader

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/fatih/set.v0"
)

const (
	workingState = 2
	idleState    = 4
)

type hashFetcherFn func(common.Hash) error
type blockFetcherFn func([]common.Hash) error

// XXX make threadsafe!!!!
type peers map[string]*peer

func (p peers) reset() {
	for _, peer := range p {
		peer.reset()
	}
}

func (p peers) get(state int) []*peer {
	var peers []*peer
	for _, peer := range p {
		peer.mu.RLock()
		if peer.state == state {
			peers = append(peers, peer)
		}
		peer.mu.RUnlock()
	}

	return peers
}

func (p peers) setState(id string, state int) {
	if peer, exist := p[id]; exist {
		peer.mu.Lock()
		defer peer.mu.Unlock()
		peer.state = state
	}
}

func (p peers) getPeer(id string) *peer {
	return p[id]
}

func (p peers) bestPeer() *peer {
	var peer *peer
	for _, cp := range p {
		if peer == nil || cp.td.Cmp(peer.td) > 0 {
			peer = cp
		}
	}
	return peer
}

// peer represents an active peer
type peer struct {
	state int // Peer state (working, idle)
	rep   int // TODO peer reputation

	mu         sync.RWMutex
	id         string
	td         *big.Int
	recentHash common.Hash

	requested *set.Set

	getHashes hashFetcherFn
	getBlocks blockFetcherFn
}

// create a new peer
func newPeer(id string, td *big.Int, hash common.Hash, getHashes hashFetcherFn, getBlocks blockFetcherFn) *peer {
	return &peer{
		id:         id,
		td:         td,
		recentHash: hash,
		getHashes:  getHashes,
		getBlocks:  getBlocks,
		state:      idleState,
		requested:  set.New(),
	}
}

// fetch a chunk using the peer
func (p *peer) fetch(chunk *chunk) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == workingState {
		return errors.New("peer already fetching chunk")
	}

	p.requested.Merge(chunk.hashes)

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

	return nil
}

// promote increases the peer's reputation
func (p *peer) promote() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.rep++
}

// demote decreases the peer's reputation or leaves it at 0
func (p *peer) demote() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.rep > 1 {
		p.rep -= 2
	} else {
		p.rep = 0
	}
}

func (p *peer) reset() {
	p.state = idleState
}
