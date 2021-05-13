// Copyright 2021 The go-ethereum Authors
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

package downloader

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// scratchHeaders is the number of headers to store in a scratch space to allow
// concurrent downloads. A header is about 0.5KB in size, so there is no worry
// about using too much memory. The only catch is that we can only validate gaps
// afer they're linked to the head, so the bigger the scratch space, the larger
// potential for invalid headers.
//
// The current scratch space of 32768 headers is expected to use 16MB RAM.
const scratchHeaders = 32768

// requestHeaders is the number of header to request from a remote peer in a single
// network packet. Although the skeleton downloader takes into consideration peer
// capacities when picking idlers, the packet size was decided to remain constant
// since headers are relatively small and it's easier to work with fixed batches
// vs. dynamic interval fillings.
const requestHeaders = 128

func init() {
	// Tuning parameters is nice, but the scratch space must be assignable in
	// full to peers. It's a useless cornercase to support a dangling half-group.
	if scratchHeaders%requestHeaders != 0 {
		panic("Please make scratchHeaders divisible by requestHeaders")
	}
}

// headEvent is a notification that the chain should reorg to a new head.
type headEvent struct {
	header *types.Header // New chain head to reorg to
	result chan error    // Channel to return if the head was accepted or denied
}

// subchain is a contiguous header chain segment that is backed by the database,
// but may not be linked to the live chain. The skeleton downloader may produce
// a new one of these every time it is restarted until the subchain grows large
// enough to connect with a previous subchain.
//
// The subchains use the exact same database namespace and are not disjoint from
// each other. As such, extending one to overlap the other entails reducing the
// second one first. This combined buffer model is used to avoid having to move
// data on disk when two subchains are joined together.
type subchain struct {
	Tail uint64 // Block number of the oldest header in the subchain
	Head uint64 // Block number of the newest header in the subchain
}

// skeletonProgress is a database entry to allow suspending and resuming a chain
// sync. As the skeleton header chain is downloaded backwards, restarts can and
// will produce temporarilly disjoint subchains. There is no way to restart a
// suspended skeleton sync without prior knowlege of all prior suspension points.
type skeletonProgress struct {
	Subchains []*subchain // Disjoint subchains downloaded until now
}

// headerRequest tracks a pending header request to ensure responses are to
// actual requests and to validate any security constraints.
//
// Concurrency note: header requests and responses are handled concurrently from
// the main runloop to allow Keccak256 hash verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. subchains).
// That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type headerRequest struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *headerResponse // Channel to deliver successful response on
	revert  chan *headerRequest  // Channel to deliver request failure on
	cancel  chan struct{}        // Channel to track sync cancellation
	timeout *time.Timer          // Timer to track delivery timeout
	stale   chan struct{}        // Channel to signal the request was dropped

	head uint64 // Head number of the requested batch of headers
}

// headerResponse is an already verified remote response to a header request.
type headerResponse struct {
	headers []*types.Header // Chain of headers
}

// skeleton represents a header chain synchronized after the Ethereum 2 merge,
// where blocks aren't validated any more via PoW in a forward fashion, rather
// are dictated and extended at the head via the beacon chain and backfilled on
// the original Ethereum 1 block sync protocol.
//
// Since the skeleton is grown backwards from head to genesis, it is handled as
// a separate entity, not mixed in with the logical sequential transition of the
// blocks. Once the skeleton is connected to an existing, validated chain, the
// headers will be moved into the main downloader for filling and execution.
//
// Opposed to the Ethereum 1 block synchronization which is trustless (and uses a
// master peer to minimize the attack surface), Ethereum 2 block synchronization
// starts from a trusted head. As such, there is no need for a master peer any
// more and headers can be requested fully concurrently (though some batches might
// be discarded if they don't link up correctly).
//
// Although a skeleton is part of a sync cycle, it is not recreated, rather stays
// alive throughout the lifetime of the downloader. This allows it to be extended
// concurrently with the sync cycle, since extensions arrive from an API surface,
// not from within (vs. Ethereum 1 sync).
//
// Since the skeleton tracks the entire header chain until it is cosumed by the
// forward block filling, it needs 0.5KB/block storage. At current mainnet sizes
// this is only possible with a disk backend. Since the skeleton is separate from
// the node's header chain, storing the headers ephemerally until sync finishes
// is wasted disk IO, but it's a price we're going to pay to keep things simple
// for now.
type skeleton struct {
	db    ethdb.Database // Database backing the skeleton
	chain LightChain     // Header chain accessor for cross links

	peers *peerSet                   // Set of peers we can sync from
	idles map[string]*peerConnection // Peers that aren't serving requests (dup func with peers, keeps things separate)

	progress *skeletonProgress // Sync progress tracker for resumption and metrics

	scratchSpace  []*types.Header // Scratch space to accumulate headers in (first = recent)
	scratchOwners []string        // Peer IDs owning chunks of the scratch space (pend or delivered)
	scratchHead   uint64          // Block number of the first item in the scratch space

	requests map[uint64]*headerRequest // Header requests currently running

	headEvents chan *headEvent // Notification channel for new heads
	abortSync  chan chan error // Termination channel to abort sync
}

// newSkeleton creates a new sync skeleton that tracks a potentially dangling
// header chain until it's linked into an existing set of blocks.
func newSkeleton(db ethdb.Database, peers *peerSet) *skeleton {
	return &skeleton{
		db:         db,
		peers:      peers,
		idles:      make(map[string]*peerConnection),
		requests:   make(map[uint64]*headerRequest),
		headEvents: make(chan *headEvent),
		abortSync:  make(chan chan error),
	}
}

// Sync starts or resumes a previous sync cycle to download and maintain a reverse
// header chain starting at the head and leading towards genesis to an available
// ancestor.
func (s *skeleton) Sync(head *types.Header) error {
	// Initialize the sync, trimming and previous leftovers until we're consistent
	// with the newly requested chain head
	s.initSync(head)

	// Create the scratch space to fill with concurrently downloaded headers
	s.scratchSpace = make([]*types.Header, scratchHeaders)
	defer func() { s.scratchSpace = nil }() // don't hold on to references after sync

	s.scratchOwners = make([]string, scratchHeaders/requestHeaders)
	defer func() { s.scratchOwners = nil }() // don't hold on to references after sync

	s.scratchHead = s.progress.Subchains[len(s.progress.Subchains)-1].Tail - 1 // tail must not be 0!

	// Create a set of unique channels for this sync cycle. We need these to be
	// ephemeral so a data race doesn't accidentally deliver something stale on
	// a persistent channel across syncs (yup, this happened)
	var (
		requestFails = make(chan *headerRequest)
		responses    = make(chan *headerResponse)
	)
	cancel := make(chan struct{})
	defer close(cancel)

	log.Debug("Starting reverse header sync cycle", "head", head.Number, "hash", head.Hash(), "cont", s.scratchHead)

	// Whether sync completed or not, disregard any future packets
	defer func() {
		log.Debug("Terminating reverse header sync cycle", "head", head.Number, "hash", head.Hash(), "cont", s.scratchHead)
		s.requests = make(map[uint64]*headerRequest)
	}()

	// Keep scheduling sync tasks
	peerJoin := make(chan *peerConnection, 16)
	peerJoinSub := s.peers.SubscribeNewPeers(peerJoin)
	defer peerJoinSub.Unsubscribe()

	peerDrop := make(chan *peerConnection, 16)
	peerDropSub := s.peers.SubscribePeerDrops(peerDrop)
	defer peerDropSub.Unsubscribe()

	for {
		// Something happened, try to assign new tasks to any idle peers
		s.assingTasks(responses, requestFails, cancel)

		// Wait for something to happen
		select {
		case peer := <-peerJoin:
			s.idles[peer.id] = peer
		case peer := <-peerDrop:
			s.revertRequests(peer.id)
			delete(s.idles, peer.id)
		case errc := <-s.abortSync:
			errc <- nil
			return nil

		case head := <-s.headEvents:
			// New head was announced, try to integrate it. If successful, nothing
			// needs to be done as the head simply extended the last range. For now
			// we don't seamlessly integrate reorgs to keep things simple. If the
			// network starts doing many mini reorgs, it might be worthwhile handling
			// a limited depth without an error (TODO(karalabe)).
			s.processNewHead(head)

		case req := <-requestFails:
			s.revertRequest(req)
		case res := <-responses:
			s.processResponse(res)
		}
	}
}

// initSync attempts to get the skeleton sync into a consistent state wrt any
// past state on disk and the newly requested head to sync to.
func (s *skeleton) initSync(head *types.Header) {
	// Extract the head number, we'll need it all over
	number := head.Number.Uint64()

	// Retrieve the previously saved sync progress
	if status := rawdb.ReadSkeletonSyncStatus(s.db); len(status) > 0 {
		s.progress = new(skeletonProgress)
		if err := json.Unmarshal(status, s.progress); err != nil {
			log.Error("Failed to decode skeleton sync status", "err", err)
		} else {
			// Previous sync was available, print some continuation logs
			for i, subchain := range s.progress.Subchains {
				log.Debug("Restarting skeleton subchain", "idx", i, "tail", subchain.Tail, "head", subchain.Head)
			}
			// Create a new subchain for the head (unless the last can be extended),
			// trimming anything it would overwrite
			headchain := &subchain{
				Tail: number,
				Head: number,
			}
			for len(s.progress.Subchains) > 0 {
				// If the last chain is above the new head, delete altogether
				idx := len(s.progress.Subchains) - 1

				lastchain := s.progress.Subchains[idx]
				if lastchain.Tail >= headchain.Tail {
					log.Debug("Dropping skeleton subchain", "idx", idx, "tail", lastchain.Tail, "head", lastchain.Head)
					s.progress.Subchains = s.progress.Subchains[:idx]
					continue
				}
				// Otherwise truncate the last chain if needed and abort trimming
				if lastchain.Head >= headchain.Tail {
					log.Debug("Trimming skeleton subchain", "idx", idx, "tail", lastchain.Tail, "oldhead", lastchain.Head, "newhead", headchain.Tail-1)
					lastchain.Head = headchain.Tail - 1
				}
				break
			}
			// If the last subchain can be extended, we're lucky. Otherwise create
			// a new subchain sync task.
			var extended bool
			if n := len(s.progress.Subchains); n > 0 {
				lastchain := s.progress.Subchains[n-1]
				if lastchain.Head == headchain.Tail-1 {
					lasthead := rawdb.ReadSkeletonHeader(s.db, lastchain.Head)
					if lasthead.Hash() == head.ParentHash {
						log.Debug("Extended skeleton subchain with new head", "idx", n-1, "tail", lastchain.Tail, "head", headchain.Tail)
						lastchain.Head = headchain.Tail
						extended = true
					}
				}
			}
			if !extended {
				log.Debug("Created new skeleton subchain", "tail", number, "head", number)
				s.progress.Subchains = append(s.progress.Subchains, headchain)
			}
			// Update the database with the new sync stats and insert the new
			// head header. We won't delete any trimmed skeleton headers since
			// those will be outside the index space of the many subchains and
			// the database space will be reclaimed eventually when processing
			// blocks above the current head (TODO(karalabe): don't forget).
			batch := s.db.NewBatch()

			rawdb.WriteSkeletonHeader(batch, head)
			s.saveSyncStatus(batch)

			if err := batch.Write(); err != nil {
				log.Crit("Failed to write skeleton sync status", "err", err)
			}
			return
		}
	}
	// Either we've failed to decode the previus state, or there was none. Start
	// a fresh sync with a single subchain represente db by the currently sent
	// chain head.
	s.progress = &skeletonProgress{
		Subchains: []*subchain{
			{
				Tail: number,
				Head: number,
			},
		},
	}
	s.saveSyncStatus(s.db)

	log.Debug("Created initial skeleton subchain", "tail", number, "head", number)
}

// saveSyncStatus marshals the remaining sync tasks into leveldb.
func (s *skeleton) saveSyncStatus(db ethdb.KeyValueWriter) {
	status, err := json.Marshal(s.progress)
	if err != nil {
		panic(err) // This can only fail during implementation
	}
	rawdb.WriteSkeletonSyncStatus(db, status)
}

// processNewHead does the internal shuffling for a new head marker and either
// accepts and integrates it into the skeleton or rejects it. Upon rejection,
// it's up to the caller to tear down the sync cycle and restart it.
func (s *skeleton) processNewHead(head *headEvent) {
	// If the header cannot be inserted without interruption, return an error for
	// the downloader to tear down the skeleton sync and restart it
	number := head.header.Number.Uint64()

	lastchain := s.progress.Subchains[len(s.progress.Subchains)-1]
	if lastchain.Tail >= number {
		head.result <- fmt.Errorf("skeleton reorged: tail: %d, newHead: %d", lastchain.Tail, number)
		return
	}
	if lastchain.Head+1 < number {
		head.result <- fmt.Errorf("skeleton gapped: head: %d, newHead: %d", lastchain.Head, number)
		return
	}
	if parent := rawdb.ReadSkeletonHeader(s.db, number-1); parent.Hash() != head.header.ParentHash {
		head.result <- fmt.Errorf("skeleton forked: ancestor: %d [%x], newAncestor: %d [%x]", parent.Number, parent.Hash(), number-1, head.header.ParentHash)
		return
	}
	// New header seems to be in the last subchain range. Unwind any extra headers
	// from the chain tip and insert the new head. We won't delete any trimmed
	// skeleton headers since those will be outside the index space of the many
	// subchains and the database space will be reclaimed eventually when processing
	// blocks above the current head (TODO(karalabe): don't forget).
	batch := s.db.NewBatch()

	rawdb.WriteSkeletonHeader(batch, head.header)
	lastchain.Head = number
	s.saveSyncStatus(batch)

	if err := batch.Write(); err != nil {
		log.Crit("Failed to write skeleton sync status", "err", err)
	}
	// Notify the new head event that it's been successfully processed
	head.result <- nil
}

// Abort tears down the current sync cycle. Note, if no sync cycle is running,
// this method will block.
func (s *skeleton) Abort() error {
	errc := make(chan error)
	s.abortSync <- errc
	return <-errc
}

// OnNewHead is a notification when we receive a new head marker from the beacon
// chain node. The method will either extend the current sync cycle seamlessly,
// or if the new header cannot be linked to the existing chain, it will tear down
// and restart sync.
func (s *skeleton) OnNewHead(header *types.Header) error {
	event := &headEvent{
		header: header,
		result: make(chan error),
	}
	return <-event.result
}

// assingTasks attempts to match idle peers to pending header retrievals.
func (s *skeleton) assingTasks(success chan *headerResponse, fail chan *headerRequest, cancel chan struct{}) {
	// Sort the peers by download capacity to use faster ones if many available
	idlers := &peerCapacitySort{
		peers: make([]*peerConnection, 0, len(s.idles)),
		caps:  make([]int, 0, len(s.idles)),
	}
	targetTTL := s.peers.rates.TargetTimeout()
	for _, peer := range s.idles {
		idlers.peers = append(idlers.peers, peer)
		idlers.caps = append(idlers.caps, s.peers.rates.Capacity(peer.id, eth.BlockHeadersMsg, targetTTL))
	}
	if len(idlers.peers) == 0 {
		return
	}
	sort.Sort(idlers)

	// Find header regions not yet downloading and fill them
	for task, owner := range s.scratchOwners {
		// If we're out of idle peers, stop assigning tasks
		if len(idlers.peers) == 0 {
			return
		}
		idle := idlers.peers[0]

		idlers.peers = idlers.peers[1:]
		idlers.caps = idlers.caps[1:]

		// Skip any tasks already filling
		if owner != "" {
			continue
		}
		// Matched a pending task to an idle peer, allocate a unique request id
		var reqid uint64
		for {
			reqid = uint64(rand.Int63())
			if reqid == 0 {
				continue
			}
			if _, ok := s.requests[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer
		req := &headerRequest{
			peer:    idle.id,
			id:      reqid,
			time:    time.Now(),
			deliver: success,
			revert:  fail,
			cancel:  cancel,
			stale:   make(chan struct{}),
			head:    s.scratchHead + uint64(task*requestHeaders),
		}
		req.timeout = time.AfterFunc(s.peers.rates.TargetTimeout(), func() {
			idle.log.Debug("Header request timed out", "reqid", reqid)
			s.peers.rates.Update(idle.id, eth.BlockHeadersMsg, 0, 0)
			s.scheduleRevertRequest(req)
		})
		s.requests[reqid] = req
		delete(s.idles, idle.id)

		// Generate the network query and send it to the peer
		go func() {
			if err := idle.peer.RequestHeadersByNumber(reqid, req.head, requestHeaders, 0, true); err != nil {
				idle.log.Debug("Failed to request headers", "err", err)
				s.scheduleRevertRequest(req)
			}
		}()

		// Inject the request into the task to block further assignments
		s.scratchOwners[task] = idle.id
	}
}

// revertRequests locates all the currently pending reuqests from a particular
// peer and reverts them, rescheduling for others to fulfill.
func (s *skeleton) revertRequests(peer string) {
	// Gather the requests first, revertals need the lock too
	var requests []*headerRequest
	for _, req := range s.requests {
		if req.peer == peer {
			requests = append(requests, req)
		}
	}
	// Revert all the requests matching the peer
	for _, req := range requests {
		s.revertRequest(req)
	}
}

// scheduleRevertRequest asks the event loop to clean up a request and return
// all failed retrieval tasks to the scheduler for reassignment.
func (s *skeleton) scheduleRevertRequest(req *headerRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertRequest cleans up a request and returns all failed retrieval tasks to
// the scheduler for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle peers.
// On peer threads, use scheduleRevertRequest.
func (s *skeleton) revertRequest(req *headerRequest) {
	log.Debug("Reverting header request", "peer", req.peer, "reqid", req.id)
	select {
	case <-req.stale:
		log.Trace("Header request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set
	delete(s.requests, req.id)

	// If there's a timeout timer still running, abort it and mark the task as
	// not-pending, ready for resheduling
	req.timeout.Stop()
	s.scratchOwners[(req.head-s.scratchHead)/requestHeaders] = ""
}

func (s *skeleton) processResponse(res *headerResponse) {

}
