package downloader

import (
	"errors"
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

// peer represents an active peer
type peer struct {
	state int // Peer state (working, idle)
	rep   int // TODO peer reputation

	mu         sync.RWMutex
	id         string
	recentHash common.Hash

	ignored *set.Set

	getHashes hashFetcherFn
	getBlocks blockFetcherFn
}

// create a new peer
func newPeer(id string, hash common.Hash, getHashes hashFetcherFn, getBlocks blockFetcherFn) *peer {
	return &peer{
		id:         id,
		recentHash: hash,
		getHashes:  getHashes,
		getBlocks:  getBlocks,
		state:      idleState,
		ignored:    set.New(),
	}
}

// fetch a chunk using the peer
func (p *peer) fetch(request *fetchRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == workingState {
		return errors.New("peer already fetching chunk")
	}

	// set working state
	p.state = workingState

	// Convert the hash set to a fetchable slice
	hashes := make([]common.Hash, 0, len(request.Hashes))
	for hash, _ := range request.Hashes {
		hashes = append(hashes, hash)
	}
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
	p.ignored.Clear()
}
