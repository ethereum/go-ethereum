// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

// Package downloader contains the manual full chain synchronisation.
package downloader

import (
	"bytes"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/fatih/set.v0"
)

const (
	eth60 = 60 // Constant to check for old protocol support
	eth61 = 61 // Constant to check for new protocol support
)

var (
	MinHashFetch  = 512 // Minimum amount of hashes to not consider a peer stalling
	MaxHashFetch  = 512 // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch = 128 // Amount of blocks to be fetched per retrieval request

	hashTTL         = 5 * time.Second  // Time it takes for a hash request to time out
	blockSoftTTL    = 3 * time.Second  // Request completion threshold for increasing or decreasing a peer's bandwidth
	blockHardTTL    = 3 * blockSoftTTL // Maximum time allowance before a block request is considered expired
	crossCheckCycle = time.Second      // Period after which to check for expired cross checks

	maxQueuedHashes = 256 * 1024 // Maximum number of hashes to queue for import (DOS protection)
	maxBannedHashes = 4096       // Number of bannable hashes before phasing old ones out
	maxBlockProcess = 256        // Number of blocks to import at once into the chain
)

var (
	errBusy             = errors.New("busy")
	errUnknownPeer      = errors.New("peer is unknown or unhealthy")
	errBadPeer          = errors.New("action from bad peer ignored")
	errStallingPeer     = errors.New("peer is stalling")
	errBannedHead       = errors.New("peer head hash already banned")
	errNoPeers          = errors.New("no peers to keep download active")
	errPendingQueue     = errors.New("pending items in queue")
	errTimeout          = errors.New("timeout")
	errEmptyHashSet     = errors.New("empty hash set by peer")
	errPeersUnavailable = errors.New("no peers available or all peers tried for block download process")
	errAlreadyInPool    = errors.New("hash already in pool")
	errInvalidChain     = errors.New("retrieved hash chain is invalid")
	errCrossCheckFailed = errors.New("block cross-check failed")
	errCancelHashFetch  = errors.New("hash fetching canceled (requested)")
	errCancelBlockFetch = errors.New("block downloading canceled (requested)")
	errNoSyncActive     = errors.New("no sync active")
)

// hashCheckFn is a callback type for verifying a hash's presence in the local chain.
type hashCheckFn func(common.Hash) bool

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
type blockRetrievalFn func(common.Hash) *types.Block

// headRetrievalFn is a callback type for retrieving the head block from the local chain.
type headRetrievalFn func() *types.Block

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
type chainInsertFn func(types.Blocks) (int, error)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

type blockPack struct {
	peerId string
	blocks []*types.Block
}

type hashPack struct {
	peerId string
	hashes []common.Hash
}

type crossCheck struct {
	expire time.Time
	parent common.Hash
}

type Downloader struct {
	mux *event.TypeMux

	queue  *queue                      // Scheduler for selecting the hashes to download
	peers  *peerSet                    // Set of active peers from which download can proceed
	checks map[common.Hash]*crossCheck // Pending cross checks to verify a hash chain
	banned *set.Set                    // Set of hashes we've received and banned

	interrupt int32 // Atomic boolean to signal termination

	// Statistics
	importStart time.Time // Instance when the last blocks were taken from the cache
	importQueue []*Block  // Previously taken blocks to check import progress
	importDone  int       // Number of taken blocks already imported from the last batch
	importLock  sync.Mutex

	// Callbacks
	hasBlock    hashCheckFn      // Checks if a block is present in the chain
	getBlock    blockRetrievalFn // Retrieves a block from the chain
	headBlock   headRetrievalFn  // Retrieves the head block from the chain
	insertChain chainInsertFn    // Injects a batch of blocks into the chain
	dropPeer    peerDropFn       // Drops a peer for misbehaving

	// Status
	synchroniseMock func(id string, hash common.Hash) error // Replacement for synchronise during testing
	synchronising   int32
	processing      int32
	notified        int32

	// Channels
	newPeerCh chan *peer
	hashCh    chan hashPack  // Channel receiving inbound hashes
	blockCh   chan blockPack // Channel receiving inbound blocks
	processCh chan bool      // Channel to signal the block fetcher of new or finished work

	cancelCh   chan struct{} // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex  // Lock to protect the cancel channel in delivers
}

// Block is an origin-tagged blockchain block.
type Block struct {
	RawBlock   *types.Block
	OriginPeer string
}

// New creates a new downloader to fetch hashes and blocks from remote peers.
func New(mux *event.TypeMux, hasBlock hashCheckFn, getBlock blockRetrievalFn, headBlock headRetrievalFn, insertChain chainInsertFn, dropPeer peerDropFn) *Downloader {
	// Create the base downloader
	downloader := &Downloader{
		mux:         mux,
		queue:       newQueue(),
		peers:       newPeerSet(),
		hasBlock:    hasBlock,
		getBlock:    getBlock,
		headBlock:   headBlock,
		insertChain: insertChain,
		dropPeer:    dropPeer,
		newPeerCh:   make(chan *peer, 1),
		hashCh:      make(chan hashPack, 1),
		blockCh:     make(chan blockPack, 1),
		processCh:   make(chan bool, 1),
	}
	// Inject all the known bad hashes
	downloader.banned = set.New()
	for hash, _ := range core.BadHashes {
		downloader.banned.Add(hash)
	}
	return downloader
}

// Stats retrieves the current status of the downloader.
func (d *Downloader) Stats() (pending int, cached int, importing int, estimate time.Duration) {
	// Fetch the download status
	pending, cached = d.queue.Size()

	// Figure out the import progress
	d.importLock.Lock()
	defer d.importLock.Unlock()

	for len(d.importQueue) > 0 && d.hasBlock(d.importQueue[0].RawBlock.Hash()) {
		d.importQueue = d.importQueue[1:]
		d.importDone++
	}
	importing = len(d.importQueue)

	// Make an estimate on the total sync
	estimate = 0
	if d.importDone > 0 {
		estimate = time.Since(d.importStart) / time.Duration(d.importDone) * time.Duration(pending+cached+importing)
	}
	return
}

// Synchronising returns whether the downloader is currently retrieving blocks.
func (d *Downloader) Synchronising() bool {
	return atomic.LoadInt32(&d.synchronising) > 0
}

// RegisterPeer injects a new download peer into the set of block source to be
// used for fetching hashes and blocks from.
func (d *Downloader) RegisterPeer(id string, version int, head common.Hash, getRelHashes relativeHashFetcherFn, getAbsHashes absoluteHashFetcherFn, getBlocks blockFetcherFn) error {
	// If the peer wants to send a banned hash, reject
	if d.banned.Has(head) {
		glog.V(logger.Debug).Infoln("Register rejected, head hash banned:", id)
		return errBannedHead
	}
	// Otherwise try to construct and register the peer
	glog.V(logger.Detail).Infoln("Registering peer", id)
	if err := d.peers.Register(newPeer(id, version, head, getRelHashes, getAbsHashes, getBlocks)); err != nil {
		glog.V(logger.Error).Infoln("Register failed:", err)
		return err
	}
	return nil
}

// UnregisterPeer remove a peer from the known list, preventing any action from
// the specified peer.
func (d *Downloader) UnregisterPeer(id string) error {
	glog.V(logger.Detail).Infoln("Unregistering peer", id)
	if err := d.peers.Unregister(id); err != nil {
		glog.V(logger.Error).Infoln("Unregister failed:", err)
		return err
	}
	return nil
}

// Synchronise tries to sync up our local block chain with a remote peer, both
// adding various sanity checks as well as wrapping it with various log entries.
func (d *Downloader) Synchronise(id string, head common.Hash) {
	glog.V(logger.Detail).Infof("Attempting synchronisation: %v, 0x%x", id, head)

	switch err := d.synchronise(id, head); err {
	case nil:
		glog.V(logger.Detail).Infof("Synchronisation completed")

	case errBusy:
		glog.V(logger.Detail).Infof("Synchronisation already in progress")

	case errTimeout, errBadPeer, errStallingPeer, errBannedHead, errEmptyHashSet, errPeersUnavailable, errInvalidChain, errCrossCheckFailed:
		glog.V(logger.Debug).Infof("Removing peer %v: %v", id, err)
		d.dropPeer(id)

	case errPendingQueue:
		glog.V(logger.Debug).Infoln("Synchronisation aborted:", err)

	default:
		glog.V(logger.Warn).Infof("Synchronisation failed: %v", err)
	}
}

// synchronise will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronize if it's TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) synchronise(id string, hash common.Hash) error {
	// Mock out the synchonisation if testing
	if d.synchroniseMock != nil {
		return d.synchroniseMock(id, hash)
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.synchronising, 0, 1) {
		return errBusy
	}
	defer atomic.StoreInt32(&d.synchronising, 0)

	// If the head hash is banned, terminate immediately
	if d.banned.Has(hash) {
		return errBannedHead
	}
	// Post a user notification of the sync (only once per session)
	if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
		glog.V(logger.Info).Infoln("Block synchronisation started")
	}
	// Abort if the queue still contains some leftover data
	if _, cached := d.queue.Size(); cached > 0 && d.queue.GetHeadBlock() != nil {
		return errPendingQueue
	}
	// Reset the queue and peer set to clean any internal leftover state
	d.queue.Reset()
	d.peers.Reset()
	d.checks = make(map[common.Hash]*crossCheck)

	// Create cancel channel for aborting mid-flight
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelLock.Unlock()

	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(p, hash)
}

// Has checks if the downloader knows about a particular hash, meaning that its
// either already downloaded of pending retrieval.
func (d *Downloader) Has(hash common.Hash) bool {
	return d.queue.Has(hash)
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hash.
func (d *Downloader) syncWithPeer(p *peer, hash common.Hash) (err error) {
	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if err != nil {
			d.cancel()
			d.mux.Post(FailedEvent{err})
		} else {
			d.mux.Post(DoneEvent{})
		}
	}()

	glog.V(logger.Debug).Infof("Synchronizing with the network using: %s, eth/%d", p.id, p.version)
	switch p.version {
	case eth60:
		// Old eth/60 version, use reverse hash retrieval algorithm
		if err = d.fetchHashes60(p, hash); err != nil {
			return err
		}
		if err = d.fetchBlocks60(); err != nil {
			return err
		}
	case eth61:
		// New eth/61, use forward, concurrent hash and block retrieval algorithm
		number, err := d.findAncestor(p)
		if err != nil {
			return err
		}
		errc := make(chan error, 2)
		go func() { errc <- d.fetchHashes(p, number+1) }()
		go func() { errc <- d.fetchBlocks(number + 1) }()

		// If any fetcher fails, cancel the other
		if err := <-errc; err != nil {
			d.cancel()
			<-errc
			return err
		}
		return <-errc

	default:
		// Something very wrong, stop right here
		glog.V(logger.Error).Infof("Unsupported eth protocol: %d", p.version)
		return errBadPeer
	}
	glog.V(logger.Debug).Infoln("Synchronization completed")

	return nil
}

// cancel cancels all of the operations and resets the queue. It returns true
// if the cancel operation was completed.
func (d *Downloader) cancel() {
	// Close the current cancel channel
	d.cancelLock.Lock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
			// Channel was already closed
		default:
			close(d.cancelCh)
		}
	}
	d.cancelLock.Unlock()

	// Reset the queue
	d.queue.Reset()
}

// Terminate interrupts the downloader, canceling all pending operations.
func (d *Downloader) Terminate() {
	atomic.StoreInt32(&d.interrupt, 1)
	d.cancel()
}

// fetchHashes60 starts retrieving hashes backwards from a specific peer and hash,
// up until it finds a common ancestor. If the source peer times out, alternative
// ones are tried for continuation.
func (d *Downloader) fetchHashes60(p *peer, h common.Hash) error {
	var (
		start  = time.Now()
		active = p             // active peer will help determine the current active peer
		head   = common.Hash{} // common and last hash

		timeout     = time.NewTimer(0)                // timer to dump a non-responsive active peer
		attempted   = make(map[string]bool)           // attempted peers will help with retries
		crossTicker = time.NewTicker(crossCheckCycle) // ticker to periodically check expired cross checks
	)
	defer crossTicker.Stop()
	defer timeout.Stop()

	glog.V(logger.Debug).Infof("Downloading hashes (%x) from %s", h[:4], p.id)
	<-timeout.C // timeout channel should be initially empty.

	getHashes := func(from common.Hash) {
		go active.getRelHashes(from)
		timeout.Reset(hashTTL)
	}

	// Add the hash to the queue, and start hash retrieval.
	d.queue.Insert([]common.Hash{h}, false)
	getHashes(h)

	attempted[p.id] = true
	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return errCancelHashFetch

		case hashPack := <-d.hashCh:
			// Make sure the active peer is giving us the hashes
			if hashPack.peerId != active.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", hashPack.peerId)
				break
			}
			timeout.Stop()

			// Make sure the peer actually gave something valid
			if len(hashPack.hashes) == 0 {
				glog.V(logger.Debug).Infof("Peer (%s) responded with empty hash set", active.id)
				return errEmptyHashSet
			}
			for index, hash := range hashPack.hashes {
				if d.banned.Has(hash) {
					glog.V(logger.Debug).Infof("Peer (%s) sent a known invalid chain", active.id)

					d.queue.Insert(hashPack.hashes[:index+1], false)
					if err := d.banBlocks(active.id, hash); err != nil {
						glog.V(logger.Debug).Infof("Failed to ban batch of blocks: %v", err)
					}
					return errInvalidChain
				}
			}
			// Determine if we're done fetching hashes (queue up all pending), and continue if not done
			done, index := false, 0
			for index, head = range hashPack.hashes {
				if d.hasBlock(head) || d.queue.GetBlock(head) != nil {
					glog.V(logger.Debug).Infof("Found common hash %x", head[:4])
					hashPack.hashes = hashPack.hashes[:index]
					done = true
					break
				}
			}
			// Insert all the new hashes, but only continue if got something useful
			inserts := d.queue.Insert(hashPack.hashes, false)
			if len(inserts) == 0 && !done {
				glog.V(logger.Debug).Infof("Peer (%s) responded with stale hashes", active.id)
				return errBadPeer
			}
			if !done {
				// Check that the peer is not stalling the sync
				if len(inserts) < MinHashFetch {
					return errStallingPeer
				}
				// Try and fetch a random block to verify the hash batch
				// Skip the last hash as the cross check races with the next hash fetch
				cross := rand.Intn(len(inserts) - 1)
				origin, parent := inserts[cross], inserts[cross+1]
				glog.V(logger.Detail).Infof("Cross checking (%s) with %x/%x", active.id, origin, parent)

				d.checks[origin] = &crossCheck{
					expire: time.Now().Add(blockSoftTTL),
					parent: parent,
				}
				go active.getBlocks([]common.Hash{origin})

				// Also fetch a fresh batch of hashes
				getHashes(head)
				continue
			}
			// We're done, prepare the download cache and proceed pulling the blocks
			offset := uint64(0)
			if block := d.getBlock(head); block != nil {
				offset = block.NumberU64() + 1
			}
			d.queue.Prepare(offset)
			finished = true

		case blockPack := <-d.blockCh:
			// Cross check the block with the random verifications
			if blockPack.peerId != active.id || len(blockPack.blocks) != 1 {
				continue
			}
			block := blockPack.blocks[0]
			if check, ok := d.checks[block.Hash()]; ok {
				if block.ParentHash() != check.parent {
					return errCrossCheckFailed
				}
				delete(d.checks, block.Hash())
			}

		case <-crossTicker.C:
			// Iterate over all the cross checks and fail the hash chain if they're not verified
			for hash, check := range d.checks {
				if time.Now().After(check.expire) {
					glog.V(logger.Debug).Infof("Cross check timeout for %x", hash)
					return errCrossCheckFailed
				}
			}

		case <-timeout.C:
			glog.V(logger.Debug).Infof("Peer (%s) didn't respond in time for hash request", p.id)

			var p *peer // p will be set if a peer can be found
			// Attempt to find a new peer by checking inclusion of peers best hash in our
			// already fetched hash list. This can't guarantee 100% correctness but does
			// a fair job. This is always either correct or false incorrect.
			for _, peer := range d.peers.AllPeers() {
				if d.queue.Has(peer.head) && !attempted[peer.id] {
					p = peer
					break
				}
			}
			// if all peers have been tried, abort the process entirely or if the hash is
			// the zero hash.
			if p == nil || (head == common.Hash{}) {
				return errTimeout
			}
			// set p to the active peer. this will invalidate any hashes that may be returned
			// by our previous (delayed) peer.
			active = p
			getHashes(head)
			glog.V(logger.Debug).Infof("Hash fetching switched to new peer(%s)", p.id)
		}
	}
	glog.V(logger.Debug).Infof("Downloaded hashes (%d) in %v", d.queue.Pending(), time.Since(start))

	return nil
}

// fetchBlocks60 iteratively downloads the entire schedules block-chain, taking
// any available peers, reserving a chunk of blocks for each, wait for delivery
// and periodically checking for timeouts.
func (d *Downloader) fetchBlocks60() error {
	glog.V(logger.Debug).Infoln("Downloading", d.queue.Pending(), "block(s)")
	start := time.Now()

	// Start a ticker to continue throttled downloads and check for bad peers
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

out:
	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case <-d.hashCh:
			// Out of bounds hashes received, ignore them

		case blockPack := <-d.blockCh:
			// Short circuit if it's a stale cross check
			if len(blockPack.blocks) == 1 {
				block := blockPack.blocks[0]
				if _, ok := d.checks[block.Hash()]; ok {
					delete(d.checks, block.Hash())
					break
				}
			}
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if peer := d.peers.Peer(blockPack.peerId); peer != nil {
				// Deliver the received chunk of blocks, and demote in case of errors
				err := d.queue.Deliver(blockPack.peerId, blockPack.blocks)
				switch err {
				case nil:
					// If no blocks were delivered, demote the peer (need the delivery above)
					if len(blockPack.blocks) == 0 {
						peer.Demote()
						peer.SetIdle()
						glog.V(logger.Detail).Infof("%s: no blocks delivered", peer)
						break
					}
					// All was successful, promote the peer and potentially start processing
					peer.Promote()
					peer.SetIdle()
					glog.V(logger.Detail).Infof("%s: delivered %d blocks", peer, len(blockPack.blocks))
					go d.process()

				case errInvalidChain:
					// The hash chain is invalid (blocks are not ordered properly), abort
					return err

				case errNoFetchesPending:
					// Peer probably timed out with its delivery but came through
					// in the end, demote, but allow to to pull from this peer.
					peer.Demote()
					peer.SetIdle()
					glog.V(logger.Detail).Infof("%s: out of bound delivery", peer)

				case errStaleDelivery:
					// Delivered something completely else than requested, usually
					// caused by a timeout and delivery during a new sync cycle.
					// Don't set it to idle as the original request should still be
					// in flight.
					peer.Demote()
					glog.V(logger.Detail).Infof("%s: stale delivery", peer)

				default:
					// Peer did something semi-useful, demote but keep it around
					peer.Demote()
					peer.SetIdle()
					glog.V(logger.Detail).Infof("%s: delivery partially failed: %v", peer, err)
					go d.process()
				}
			}

		case <-ticker.C:
			// Short circuit if we lost all our peers
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			// Check for block request timeouts and demote the responsible peers
			badPeers := d.queue.Expire(blockHardTTL)
			for _, pid := range badPeers {
				if peer := d.peers.Peer(pid); peer != nil {
					peer.Demote()
					glog.V(logger.Detail).Infof("%s: block delivery timeout", peer)
				}
			}
			// If there are unrequested hashes left start fetching from the available peers
			if d.queue.Pending() > 0 {
				// Throttle the download if block cache is full and waiting processing
				if d.queue.Throttle() {
					break
				}
				// Send a download request to all idle peers, until throttled
				idlePeers := d.peers.IdlePeers()
				for _, peer := range idlePeers {
					// Short circuit if throttling activated since above
					if d.queue.Throttle() {
						break
					}
					// Get a possible chunk. If nil is returned no chunk
					// could be returned due to no hashes available.
					request := d.queue.Reserve(peer, peer.Capacity())
					if request == nil {
						continue
					}
					if glog.V(logger.Detail) {
						glog.Infof("%s: requesting %d blocks", peer, len(request.Hashes))
					}
					// Fetch the chunk and check for error. If the peer was somehow
					// already fetching a chunk due to a bug, it will be returned to
					// the queue
					if err := peer.Fetch(request); err != nil {
						glog.V(logger.Error).Infof("Peer %s received double work", peer.id)
						d.queue.Cancel(request)
					}
				}
				// Make sure that we have peers available for fetching. If all peers have been tried
				// and all failed throw an error
				if d.queue.InFlight() == 0 {
					return errPeersUnavailable
				}

			} else if d.queue.InFlight() == 0 {
				// When there are no more queue and no more in flight, We can
				// safely assume we're done. Another part of the process will  check
				// for parent errors and will re-request anything that's missing
				break out
			}
		}
	}
	glog.V(logger.Detail).Infoln("Downloaded block(s) in", time.Since(start))
	return nil
}

// findAncestor tries to locate the common ancestor block of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N blocks should already get us a match.
// In the rare scenario when we ended up on a long soft fork (i.e. none of the
// head blocks match), we do a binary search to find the common ancestor.
func (d *Downloader) findAncestor(p *peer) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: looking for common ancestor", p)

	// Request out head blocks to short circuit ancestor location
	head := d.headBlock().NumberU64()
	from := int64(head) - int64(MaxHashFetch)
	if from < 0 {
		from = 0
	}
	go p.getAbsHashes(uint64(from), MaxHashFetch)

	// Wait for the remote response to the head fetch
	number, hash := uint64(0), common.Hash{}
	timeout := time.After(hashTTL)

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCancelHashFetch

		case hashPack := <-d.hashCh:
			// Discard anything not from the origin peer
			if hashPack.peerId != p.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", hashPack.peerId)
				break
			}
			// Make sure the peer actually gave something valid
			hashes := hashPack.hashes
			if len(hashes) == 0 {
				glog.V(logger.Debug).Infof("%v: empty head hash set", p)
				return 0, errEmptyHashSet
			}
			// Check if a common ancestor was found
			finished = true
			for i := len(hashes) - 1; i >= 0; i-- {
				if d.hasBlock(hashes[i]) {
					number, hash = uint64(from)+uint64(i), hashes[i]
					break
				}
			}

		case <-d.blockCh:
			// Out of bounds blocks received, ignore them

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head hash timeout", p)
			return 0, errTimeout
		}
	}
	// If the head fetch already found an ancestor, return
	if !common.EmptyHash(hash) {
		glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x]", p, number, hash[:4])
		return number, nil
	}
	// Ancestor not found, we need to binary search over our chain
	start, end := uint64(0), head
	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		timeout := time.After(hashTTL)
		go p.getAbsHashes(uint64(check), 1)

		// Wait until a reply arrives to this request
		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCancelHashFetch

			case hashPack := <-d.hashCh:
				// Discard anything not from the origin peer
				if hashPack.peerId != p.id {
					glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", hashPack.peerId)
					break
				}
				// Make sure the peer actually gave something valid
				hashes := hashPack.hashes
				if len(hashes) != 1 {
					glog.V(logger.Debug).Infof("%v: invalid search hash set (%d)", p, len(hashes))
					return 0, errBadPeer
				}
				arrived = true

				// Modify the search interval based on the response
				block := d.getBlock(hashes[0])
				if block == nil {
					end = check
					break
				}
				if block.NumberU64() != check {
					glog.V(logger.Debug).Infof("%v: non requested hash #%d [%x], instead of #%d", p, block.NumberU64(), block.Hash().Bytes()[:4], check)
					return 0, errBadPeer
				}
				start = check

			case <-d.blockCh:
				// Out of bounds blocks received, ignore them

			case <-timeout:
				glog.V(logger.Debug).Infof("%v: search hash timeout", p)
				return 0, errTimeout
			}
		}
	}
	return start, nil
}

// fetchHashes keeps retrieving hashes from the requested number, until no more
// are returned, potentially throttling on the way.
func (d *Downloader) fetchHashes(p *peer, from uint64) error {
	glog.V(logger.Debug).Infof("%v: downloading hashes from #%d", p, from)

	// Create a timeout timer, and the associated hash fetcher
	timeout := time.NewTimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	getHashes := func(from uint64) {
		glog.V(logger.Detail).Infof("%v: fetching %d hashes from #%d", p, MaxHashFetch, from)

		go p.getAbsHashes(from, MaxHashFetch)
		timeout.Reset(hashTTL)
	}
	// Start pulling hashes, until all are exhausted
	getHashes(from)
	for {
		select {
		case <-d.cancelCh:
			return errCancelHashFetch

		case hashPack := <-d.hashCh:
			// Make sure the active peer is giving us the hashes
			if hashPack.peerId != p.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", hashPack.peerId)
				break
			}
			timeout.Stop()

			// If no more hashes are inbound, notify the block fetcher and return
			if len(hashPack.hashes) == 0 {
				glog.V(logger.Debug).Infof("%v: no available hashes", p)

				select {
				case d.processCh <- false:
				case <-d.cancelCh:
				}
				return nil
			}
			// Otherwise insert all the new hashes, aborting in case of junk
			glog.V(logger.Detail).Infof("%v: inserting %d hashes from #%d", p, len(hashPack.hashes), from)

			inserts := d.queue.Insert(hashPack.hashes, true)
			if len(inserts) != len(hashPack.hashes) {
				glog.V(logger.Debug).Infof("%v: stale hashes", p)
				return errBadPeer
			}
			// Notify the block fetcher of new hashes, but stop if queue is full
			cont := d.queue.Pending() < maxQueuedHashes
			select {
			case d.processCh <- cont:
			default:
			}
			if !cont {
				return nil
			}
			// Queue not yet full, fetch the next batch
			from += uint64(len(hashPack.hashes))
			getHashes(from)

		case <-timeout.C:
			glog.V(logger.Debug).Infof("%v: hash request timed out", p)
			return errTimeout
		}
	}
}

// fetchBlocks iteratively downloads the scheduled hashes, taking any available
// peers, reserving a chunk of blocks for each, waiting for delivery and also
// periodically checking for timeouts.
func (d *Downloader) fetchBlocks(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading blocks from #%d", from)
	defer glog.V(logger.Debug).Infof("Block download terminated")

	// Create a timeout timer for scheduling expiration tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Prepare the queue and fetch blocks until the hash fetcher's done
	d.queue.Prepare(from)
	finished := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case blockPack := <-d.blockCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if peer := d.peers.Peer(blockPack.peerId); peer != nil {
				// Deliver the received chunk of blocks, and demote in case of errors
				err := d.queue.Deliver(blockPack.peerId, blockPack.blocks)
				switch err {
				case nil:
					// If no blocks were delivered, demote the peer (need the delivery above)
					if len(blockPack.blocks) == 0 {
						peer.Demote()
						peer.SetIdle()
						glog.V(logger.Detail).Infof("%s: no blocks delivered", peer)
						break
					}
					// All was successful, promote the peer and potentially start processing
					peer.Promote()
					peer.SetIdle()
					glog.V(logger.Detail).Infof("%s: delivered %d blocks", peer, len(blockPack.blocks))
					go d.process()

				case errInvalidChain:
					// The hash chain is invalid (blocks are not ordered properly), abort
					return err

				case errNoFetchesPending:
					// Peer probably timed out with its delivery but came through
					// in the end, demote, but allow to to pull from this peer.
					peer.Demote()
					peer.SetIdle()
					glog.V(logger.Detail).Infof("%s: out of bound delivery", peer)

				case errStaleDelivery:
					// Delivered something completely else than requested, usually
					// caused by a timeout and delivery during a new sync cycle.
					// Don't set it to idle as the original request should still be
					// in flight.
					peer.Demote()
					glog.V(logger.Detail).Infof("%s: stale delivery", peer)

				default:
					// Peer did something semi-useful, demote but keep it around
					peer.Demote()
					peer.SetIdle()
					glog.V(logger.Detail).Infof("%s: delivery partially failed: %v", peer, err)
					go d.process()
				}
			}
			// Blocks arrived, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-d.processCh:
			// The hash fetcher sent a continuation flag, check if it's done
			if !cont {
				finished = true
			}
			// Hashes arrive, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-ticker.C:
			// Sanity check update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-update:
			// Short circuit if we lost all our peers
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			// Check for block request timeouts and demote the responsible peers
			for _, pid := range d.queue.Expire(blockHardTTL) {
				if peer := d.peers.Peer(pid); peer != nil {
					peer.Demote()
					glog.V(logger.Detail).Infof("%s: block delivery timeout", peer)
				}
			}
			// If there's noting more to fetch, wait or terminate
			if d.queue.Pending() == 0 {
				if d.queue.InFlight() == 0 && finished {
					glog.V(logger.Debug).Infof("Block fetching completed")
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			for _, peer := range d.peers.IdlePeers() {
				// Short circuit if throttling activated
				if d.queue.Throttle() {
					break
				}
				// Reserve a chunk of hashes for a peer. A nil can mean either that
				// no more hashes are available, or that the peer is known not to
				// have them.
				request := d.queue.Reserve(peer, peer.Capacity())
				if request == nil {
					continue
				}
				if glog.V(logger.Detail) {
					glog.Infof("%s: requesting %d blocks", peer, len(request.Hashes))
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if err := peer.Fetch(request); err != nil {
					glog.V(logger.Error).Infof("%v: fetch failed, rescheduling", peer)
					d.queue.Cancel(request)
				}
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !d.queue.Throttle() && d.queue.InFlight() == 0 {
				return errPeersUnavailable
			}
		}
	}
}

// banBlocks retrieves a batch of blocks from a peer feeding us invalid hashes,
// and bans the head of the retrieved batch.
//
// This method only fetches one single batch as the goal is not ban an entire
// (potentially long) invalid chain - wasting a lot of time in the meanwhile -,
// but rather to gradually build up a blacklist if the peer keeps reconnecting.
func (d *Downloader) banBlocks(peerId string, head common.Hash) error {
	glog.V(logger.Debug).Infof("Banning a batch out of %d blocks from %s", d.queue.Pending(), peerId)

	// Ask the peer being banned for a batch of blocks from the banning point
	peer := d.peers.Peer(peerId)
	if peer == nil {
		return nil
	}
	request := d.queue.Reserve(peer, MaxBlockFetch)
	if request == nil {
		return nil
	}
	if err := peer.Fetch(request); err != nil {
		return err
	}
	// Wait a bit for the reply to arrive, and ban if done so
	timeout := time.After(blockHardTTL)
	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case <-timeout:
			return errTimeout

		case <-d.hashCh:
			// Out of bounds hashes received, ignore them

		case blockPack := <-d.blockCh:
			blocks := blockPack.blocks

			// Short circuit if it's a stale cross check
			if len(blocks) == 1 {
				block := blocks[0]
				if _, ok := d.checks[block.Hash()]; ok {
					delete(d.checks, block.Hash())
					break
				}
			}
			// Short circuit if it's not from the peer being banned
			if blockPack.peerId != peerId {
				break
			}
			// Short circuit if no blocks were returned
			if len(blocks) == 0 {
				return errors.New("no blocks returned to ban")
			}
			// Reconstruct the original chain order and ensure we're banning the correct blocks
			types.BlockBy(types.Number).Sort(blocks)
			if bytes.Compare(blocks[0].Hash().Bytes(), head.Bytes()) != 0 {
				return errors.New("head block not the banned one")
			}
			index := 0
			for _, block := range blocks[1:] {
				if bytes.Compare(block.ParentHash().Bytes(), blocks[index].Hash().Bytes()) != 0 {
					break
				}
				index++
			}
			// Ban the head hash and phase out any excess
			d.banned.Add(blocks[index].Hash())
			for d.banned.Size() > maxBannedHashes {
				var evacuate common.Hash

				d.banned.Each(func(item interface{}) bool {
					// Skip any hard coded bans
					if core.BadHashes[item.(common.Hash)] {
						return true
					}
					evacuate = item.(common.Hash)
					return false
				})
				d.banned.Remove(evacuate)
			}
			glog.V(logger.Debug).Infof("Banned %d blocks from: %s", index+1, peerId)
			return nil
		}
	}
}

// process takes blocks from the queue and tries to import them into the chain.
//
// The algorithmic flow is as follows:
//  - The `processing` flag is swapped to 1 to ensure singleton access
//  - The current `cancel` channel is retrieved to detect sync abortions
//  - Blocks are iteratively taken from the cache and inserted into the chain
//  - When the cache becomes empty, insertion stops
//  - The `processing` flag is swapped back to 0
//  - A post-exit check is made whether new blocks became available
//     - This step is important: it handles a potential race condition between
//       checking for no more work, and releasing the processing "mutex". In
//       between these state changes, a block may have arrived, but a processing
//       attempt denied, so we need to re-enter to ensure the block isn't left
//       to idle in the cache.
func (d *Downloader) process() {
	// Make sure only one goroutine is ever allowed to process blocks at once
	if !atomic.CompareAndSwapInt32(&d.processing, 0, 1) {
		return
	}
	// If the processor just exited, but there are freshly pending items, try to
	// reenter. This is needed because the goroutine spinned up for processing
	// the fresh blocks might have been rejected entry to to this present thread
	// not yet releasing the `processing` state.
	defer func() {
		if atomic.LoadInt32(&d.interrupt) == 0 && d.queue.GetHeadBlock() != nil {
			d.process()
		}
	}()
	// Release the lock upon exit (note, before checking for reentry!), and set
	// the import statistics to zero.
	defer func() {
		d.importLock.Lock()
		d.importQueue = nil
		d.importDone = 0
		d.importLock.Unlock()

		atomic.StoreInt32(&d.processing, 0)
	}()
	// Repeat the processing as long as there are blocks to import
	for {
		// Fetch the next batch of blocks
		blocks := d.queue.TakeBlocks()
		if len(blocks) == 0 {
			return
		}
		// Reset the import statistics
		d.importLock.Lock()
		d.importStart = time.Now()
		d.importQueue = blocks
		d.importDone = 0
		d.importLock.Unlock()

		// Actually import the blocks
		glog.V(logger.Debug).Infof("Inserting chain with %d blocks (#%v - #%v)\n", len(blocks), blocks[0].RawBlock.Number(), blocks[len(blocks)-1].RawBlock.Number())
		for len(blocks) != 0 {
			// Check for any termination requests
			if atomic.LoadInt32(&d.interrupt) == 1 {
				return
			}
			// Retrieve the first batch of blocks to insert
			max := int(math.Min(float64(len(blocks)), float64(maxBlockProcess)))
			raw := make(types.Blocks, 0, max)
			for _, block := range blocks[:max] {
				raw = append(raw, block.RawBlock)
			}
			// Try to inset the blocks, drop the originating peer if there's an error
			index, err := d.insertChain(raw)
			if err != nil {
				glog.V(logger.Debug).Infof("Block #%d import failed: %v", raw[index].NumberU64(), err)
				d.dropPeer(blocks[index].OriginPeer)
				d.cancel()
				return
			}
			blocks = blocks[max:]
		}
	}
}

// DeliverBlocks injects a new batch of blocks received from a remote node.
// This is usually invoked through the BlocksMsg by the protocol handler.
func (d *Downloader) DeliverBlocks(id string, blocks []*types.Block) error {
	// Make sure the downloader is active
	if atomic.LoadInt32(&d.synchronising) == 0 {
		return errNoSyncActive
	}
	// Deliver or abort if the sync is canceled while queuing
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case d.blockCh <- blockPack{id, blocks}:
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}

// DeliverHashes injects a new batch of hashes received from a remote node into
// the download schedule. This is usually invoked through the BlockHashesMsg by
// the protocol handler.
func (d *Downloader) DeliverHashes(id string, hashes []common.Hash) error {
	// Make sure the downloader is active
	if atomic.LoadInt32(&d.synchronising) == 0 {
		return errNoSyncActive
	}
	// Deliver or abort if the sync is canceled while queuing
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case d.hashCh <- hashPack{id, hashes}:
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}
