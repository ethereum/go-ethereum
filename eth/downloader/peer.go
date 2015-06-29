// Contains the active peer-set of the downloader, maintaining both failures
// as well as reputation metrics to prioritize the block retrievals.

package downloader

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/fatih/set.v0"
)

type hashFetcherFn func(common.Hash) error
type blockFetcherFn func([]common.Hash) error

var (
	errAlreadyFetching   = errors.New("already fetching blocks from peer")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

// peer represents an active peer from which hashes and blocks are retrieved.
type peer struct {
	id   string      // Unique identifier of the peer
	head common.Hash // Hash of the peers latest known block

	idle int32 // Current activity state of the peer (idle = 0, active = 1)
	rep  int32 // Simple peer reputation

	capacity int32     // Number of blocks allowed to fetch per request
	started  time.Time // Time instance when the last fetch was started

	ignored *set.Set // Set of hashes not to request (didn't have previously)

	getHashes hashFetcherFn  // Method to retrieve a batch of hashes (mockable for testing)
	getBlocks blockFetcherFn // Method to retrieve a batch of blocks (mockable for testing)

	version int // Eth protocol version number to switch strategies
}

// newPeer create a new downloader peer, with specific hash and block retrieval
// mechanisms.
func newPeer(id string, version int, head common.Hash, getHashes hashFetcherFn, getBlocks blockFetcherFn) *peer {
	return &peer{
		id:        id,
		head:      head,
		capacity:  1,
		getHashes: getHashes,
		getBlocks: getBlocks,
		ignored:   set.New(),
		version:   version,
	}
}

// Reset clears the internal state of a peer entity.
func (p *peer) Reset() {
	atomic.StoreInt32(&p.idle, 0)
	atomic.StoreInt32(&p.capacity, 1)
	p.ignored.Clear()
}

// Fetch sends a block retrieval request to the remote peer.
func (p *peer) Fetch(request *fetchRequest) error {
	// Short circuit if the peer is already fetching
	if !atomic.CompareAndSwapInt32(&p.idle, 0, 1) {
		return errAlreadyFetching
	}
	p.started = time.Now()

	// Convert the hash set to a retrievable slice
	hashes := make([]common.Hash, 0, len(request.Hashes))
	for hash, _ := range request.Hashes {
		hashes = append(hashes, hash)
	}
	go p.getBlocks(hashes)

	return nil
}

// SetIdle sets the peer to idle, allowing it to execute new retrieval requests.
// Its block retrieval allowance will also be updated either up- or downwards,
// depending on whether the previous fetch completed in time or not.
func (p *peer) SetIdle() {
	// Update the peer's download allowance based on previous performance
	scale := 2.0
	if time.Since(p.started) > blockSoftTTL {
		scale = 0.5
		if time.Since(p.started) > blockHardTTL {
			scale = 1 / float64(MaxBlockFetch) // reduces capacity to 1
		}
	}
	for {
		// Calculate the new download bandwidth allowance
		prev := atomic.LoadInt32(&p.capacity)
		next := int32(math.Max(1, math.Min(float64(MaxBlockFetch), float64(prev)*scale)))

		// Try to update the old value
		if atomic.CompareAndSwapInt32(&p.capacity, prev, next) {
			// If we're having problems at 1 capacity, try to find better peers
			if next == 1 {
				p.Demote()
			}
			break
		}
	}
	// Set the peer to idle to allow further block requests
	atomic.StoreInt32(&p.idle, 0)
}

// Capacity retrieves the peers block download allowance based on its previously
// discovered bandwidth capacity.
func (p *peer) Capacity() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// Promote increases the peer's reputation.
func (p *peer) Promote() {
	atomic.AddInt32(&p.rep, 1)
}

// Demote decreases the peer's reputation or leaves it at 0.
func (p *peer) Demote() {
	for {
		// Calculate the new reputation value
		prev := atomic.LoadInt32(&p.rep)
		next := prev / 2

		// Try to update the old value
		if atomic.CompareAndSwapInt32(&p.rep, prev, next) {
			return
		}
	}
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("reputation %3d, ", atomic.LoadInt32(&p.rep))+
			fmt.Sprintf("capacity %3d, ", atomic.LoadInt32(&p.capacity))+
			fmt.Sprintf("ignored %4d", p.ignored.Size()),
	)
}

// peerSet represents the collection of active peer participating in the block
// download procedure.
type peerSet struct {
	peers map[string]*peer
	lock  sync.RWMutex
}

// newPeerSet creates a new peer set top track the active download sources.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Reset iterates over the current peer set, and resets each of the known peers
// to prepare for a next batch of block retrieval.
func (ps *peerSet) Reset() {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, peer := range ps.peers {
		peer.Reset()
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// AllPeers retrieves a flat list of all the peers within the set.
func (ps *peerSet) AllPeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
	}
	return list
}

// IdlePeers retrieves a flat list of all the currently idle peers within the
// active peer set, ordered by their reputation.
func (ps *peerSet) IdlePeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if atomic.LoadInt32(&p.idle) == 0 {
			list = append(list, p)
		}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if atomic.LoadInt32(&list[i].rep) < atomic.LoadInt32(&list[j].rep) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	return list
}
