// Copyright 2015 The go-ethereum Authors
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

// Package downloader contains the manual full chain synchronisation.
package downloader

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/rcrowley/go-metrics"
)

var (
	MaxHashFetch    = 512 // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch   = 128 // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch  = 192 // Amount of block headers to be fetched per retrieval request
	MaxBodyFetch    = 128 // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch = 256 // Amount of transaction receipts to allow fetching per request
	MaxStateFetch   = 384 // Amount of node state values to allow fetching per request

	hashTTL        = 5 * time.Second    // [eth/61] Time it takes for a hash request to time out
	blockSoftTTL   = 3 * time.Second    // [eth/61] Request completion threshold for increasing or decreasing a peer's bandwidth
	blockHardTTL   = 3 * blockSoftTTL   // [eth/61] Maximum time allowance before a block request is considered expired
	headerTTL      = 5 * time.Second    // [eth/62] Time it takes for a header request to time out
	bodySoftTTL    = 3 * time.Second    // [eth/62] Request completion threshold for increasing or decreasing a peer's bandwidth
	bodyHardTTL    = 3 * bodySoftTTL    // [eth/62] Maximum time allowance before a block body request is considered expired
	receiptSoftTTL = 3 * time.Second    // [eth/63] Request completion threshold for increasing or decreasing a peer's bandwidth
	receiptHardTTL = 3 * receiptSoftTTL // [eth/63] Maximum time allowance before a receipt request is considered expired
	stateSoftTTL   = 2 * time.Second    // [eth/63] Request completion threshold for increasing or decreasing a peer's bandwidth
	stateHardTTL   = 3 * stateSoftTTL   // [eth/63] Maximum time allowance before a node data request is considered expired

	maxQueuedHashes   = 256 * 1024 // [eth/61] Maximum number of hashes to queue for import (DOS protection)
	maxQueuedHeaders  = 256 * 1024 // [eth/62] Maximum number of headers to queue for import (DOS protection)
	maxQueuedStates   = 256 * 1024 // [eth/63] Maximum number of state requests to queue (DOS protection)
	maxResultsProcess = 256        // Number of download results to import at once into the chain

	fsHeaderCheckFrequency = 100  // Verification frequency of the downloaded headers during fast sync
	fsHeaderSafetyNet      = 2048 // Number of headers to discard in case a chain violation is detected
	fsHeaderForceVerify    = 24   // Number of headers to verify before and after the pivot to accept it
	fsPivotInterval        = 512  // Number of headers out of which to randomize the pivot point
	fsMinFullBlocks        = 1024 // Number of blocks to retrieve fully even in fast sync
)

var (
	errBusy               = errors.New("busy")
	errUnknownPeer        = errors.New("peer is unknown or unhealthy")
	errBadPeer            = errors.New("action from bad peer ignored")
	errStallingPeer       = errors.New("peer is stalling")
	errNoPeers            = errors.New("no peers to keep download active")
	errPendingQueue       = errors.New("pending items in queue")
	errTimeout            = errors.New("timeout")
	errEmptyHashSet       = errors.New("empty hash set by peer")
	errEmptyHeaderSet     = errors.New("empty header set by peer")
	errPeersUnavailable   = errors.New("no peers available or all tried for download")
	errAlreadyInPool      = errors.New("hash already in pool")
	errInvalidChain       = errors.New("retrieved hash chain is invalid")
	errInvalidBlock       = errors.New("retrieved block is invalid")
	errInvalidBody        = errors.New("retrieved block body is invalid")
	errInvalidReceipt     = errors.New("retrieved receipt is invalid")
	errCancelHashFetch    = errors.New("hash download canceled (requested)")
	errCancelBlockFetch   = errors.New("block download canceled (requested)")
	errCancelHeaderFetch  = errors.New("block header download canceled (requested)")
	errCancelBodyFetch    = errors.New("block body download canceled (requested)")
	errCancelReceiptFetch = errors.New("receipt download canceled (requested)")
	errCancelStateFetch   = errors.New("state data download canceled (requested)")
	errNoSyncActive       = errors.New("no sync active")
)

type Downloader struct {
	mode   SyncMode       // Synchronisation mode defining the strategy used (per sync cycle)
	noFast bool           // Flag to disable fast syncing in case of a security error
	mux    *event.TypeMux // Event multiplexer to announce sync operation events

	queue *queue   // Scheduler for selecting the hashes to download
	peers *peerSet // Set of active peers from which download can proceed

	interrupt int32 // Atomic boolean to signal termination

	// Statistics
	syncStatsChainOrigin uint64       // Origin block number where syncing started at
	syncStatsChainHeight uint64       // Highest block number known when syncing started
	syncStatsStateTotal  uint64       // Total number of node state entries known so far
	syncStatsStateDone   uint64       // Number of state trie entries already pulled
	syncStatsLock        sync.RWMutex // Lock protecting the sync stats fields

	// Callbacks
	hasHeader       headerCheckFn            // Checks if a header is present in the chain
	hasBlock        blockCheckFn             // Checks if a block is present in the chain
	getHeader       headerRetrievalFn        // Retrieves a header from the chain
	getBlock        blockRetrievalFn         // Retrieves a block from the chain
	headHeader      headHeaderRetrievalFn    // Retrieves the head header from the chain
	headBlock       headBlockRetrievalFn     // Retrieves the head block from the chain
	headFastBlock   headFastBlockRetrievalFn // Retrieves the head fast-sync block from the chain
	commitHeadBlock headBlockCommitterFn     // Commits a manually assembled block as the chain head
	getTd           tdRetrievalFn            // Retrieves the TD of a block from the chain
	insertHeaders   headerChainInsertFn      // Injects a batch of headers into the chain
	insertBlocks    blockChainInsertFn       // Injects a batch of blocks into the chain
	insertReceipts  receiptChainInsertFn     // Injects a batch of blocks and their receipts into the chain
	rollback        chainRollbackFn          // Removes a batch of recently added chain links
	dropPeer        peerDropFn               // Drops a peer for misbehaving

	// Status
	synchroniseMock func(id string, hash common.Hash) error // Replacement for synchronise during testing
	synchronising   int32
	processing      int32
	notified        int32

	// Channels
	newPeerCh     chan *peer
	hashCh        chan dataPack // [eth/61] Channel receiving inbound hashes
	blockCh       chan dataPack // [eth/61] Channel receiving inbound blocks
	headerCh      chan dataPack // [eth/62] Channel receiving inbound block headers
	bodyCh        chan dataPack // [eth/62] Channel receiving inbound block bodies
	receiptCh     chan dataPack // [eth/63] Channel receiving inbound receipts
	stateCh       chan dataPack // [eth/63] Channel receiving inbound node state data
	blockWakeCh   chan bool     // [eth/61] Channel to signal the block fetcher of new tasks
	bodyWakeCh    chan bool     // [eth/62] Channel to signal the block body fetcher of new tasks
	receiptWakeCh chan bool     // [eth/63] Channel to signal the receipt fetcher of new tasks
	stateWakeCh   chan bool     // [eth/63] Channel to signal the state fetcher of new tasks

	cancelCh   chan struct{} // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex  // Lock to protect the cancel channel in delivers

	// Testing hooks
	syncInitHook     func(uint64, uint64)  // Method to call upon initiating a new sync run
	bodyFetchHook    func([]*types.Header) // Method to call upon starting a block body fetch
	receiptFetchHook func([]*types.Header) // Method to call upon starting a receipt fetch
	chainInsertHook  func([]*fetchResult)  // Method to call upon inserting a chain of blocks (possibly in multiple invocations)
}

// New creates a new downloader to fetch hashes and blocks from remote peers.
func New(stateDb ethdb.Database, mux *event.TypeMux, hasHeader headerCheckFn, hasBlock blockCheckFn, getHeader headerRetrievalFn,
	getBlock blockRetrievalFn, headHeader headHeaderRetrievalFn, headBlock headBlockRetrievalFn, headFastBlock headFastBlockRetrievalFn,
	commitHeadBlock headBlockCommitterFn, getTd tdRetrievalFn, insertHeaders headerChainInsertFn, insertBlocks blockChainInsertFn,
	insertReceipts receiptChainInsertFn, rollback chainRollbackFn, dropPeer peerDropFn) *Downloader {

	return &Downloader{
		mode:            FullSync,
		mux:             mux,
		queue:           newQueue(stateDb),
		peers:           newPeerSet(),
		hasHeader:       hasHeader,
		hasBlock:        hasBlock,
		getHeader:       getHeader,
		getBlock:        getBlock,
		headHeader:      headHeader,
		headBlock:       headBlock,
		headFastBlock:   headFastBlock,
		commitHeadBlock: commitHeadBlock,
		getTd:           getTd,
		insertHeaders:   insertHeaders,
		insertBlocks:    insertBlocks,
		insertReceipts:  insertReceipts,
		rollback:        rollback,
		dropPeer:        dropPeer,
		newPeerCh:       make(chan *peer, 1),
		hashCh:          make(chan dataPack, 1),
		blockCh:         make(chan dataPack, 1),
		headerCh:        make(chan dataPack, 1),
		bodyCh:          make(chan dataPack, 1),
		receiptCh:       make(chan dataPack, 1),
		stateCh:         make(chan dataPack, 1),
		blockWakeCh:     make(chan bool, 1),
		bodyWakeCh:      make(chan bool, 1),
		receiptWakeCh:   make(chan bool, 1),
		stateWakeCh:     make(chan bool, 1),
	}
}

// Progress retrieves the synchronisation boundaries, specifically the origin
// block where synchronisation started at (may have failed/suspended); the block
// or header sync is currently at; and the latest known block which the sync targets.
func (d *Downloader) Progress() (uint64, uint64, uint64) {
	d.syncStatsLock.RLock()
	defer d.syncStatsLock.RUnlock()

	current := uint64(0)
	switch d.mode {
	case FullSync:
		current = d.headBlock().NumberU64()
	case FastSync:
		current = d.headFastBlock().NumberU64()
	case LightSync:
		current = d.headHeader().Number.Uint64()
	}
	return d.syncStatsChainOrigin, current, d.syncStatsChainHeight
}

// Synchronising returns whether the downloader is currently retrieving blocks.
func (d *Downloader) Synchronising() bool {
	return atomic.LoadInt32(&d.synchronising) > 0 || atomic.LoadInt32(&d.processing) > 0
}

// RegisterPeer injects a new download peer into the set of block source to be
// used for fetching hashes and blocks from.
func (d *Downloader) RegisterPeer(id string, version int, head common.Hash,
	getRelHashes relativeHashFetcherFn, getAbsHashes absoluteHashFetcherFn, getBlocks blockFetcherFn, // eth/61 callbacks, remove when upgrading
	getRelHeaders relativeHeaderFetcherFn, getAbsHeaders absoluteHeaderFetcherFn, getBlockBodies blockBodyFetcherFn,
	getReceipts receiptFetcherFn, getNodeData stateFetcherFn) error {

	glog.V(logger.Detail).Infoln("Registering peer", id)
	if err := d.peers.Register(newPeer(id, version, head, getRelHashes, getAbsHashes, getBlocks, getRelHeaders, getAbsHeaders, getBlockBodies, getReceipts, getNodeData)); err != nil {
		glog.V(logger.Error).Infoln("Register failed:", err)
		return err
	}
	return nil
}

// UnregisterPeer remove a peer from the known list, preventing any action from
// the specified peer. An effort is also made to return any pending fetches into
// the queue.
func (d *Downloader) UnregisterPeer(id string) error {
	glog.V(logger.Detail).Infoln("Unregistering peer", id)
	if err := d.peers.Unregister(id); err != nil {
		glog.V(logger.Error).Infoln("Unregister failed:", err)
		return err
	}
	d.queue.Revoke(id)
	return nil
}

// Synchronise tries to sync up our local block chain with a remote peer, both
// adding various sanity checks as well as wrapping it with various log entries.
func (d *Downloader) Synchronise(id string, head common.Hash, td *big.Int, mode SyncMode) error {
	glog.V(logger.Detail).Infof("Attempting synchronisation: %v, head [%x…], TD %v", id, head[:4], td)

	err := d.synchronise(id, head, td, mode)
	switch err {
	case nil:
		glog.V(logger.Detail).Infof("Synchronisation completed")

	case errBusy:
		glog.V(logger.Detail).Infof("Synchronisation already in progress")

	case errTimeout, errBadPeer, errStallingPeer, errEmptyHashSet, errEmptyHeaderSet, errPeersUnavailable, errInvalidChain:
		glog.V(logger.Debug).Infof("Removing peer %v: %v", id, err)
		d.dropPeer(id)

	case errPendingQueue:
		glog.V(logger.Debug).Infoln("Synchronisation aborted:", err)

	default:
		glog.V(logger.Warn).Infof("Synchronisation failed: %v", err)
	}
	return err
}

// synchronise will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronize if it's TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) synchronise(id string, hash common.Hash, td *big.Int, mode SyncMode) error {
	// Mock out the synchonisation if testing
	if d.synchroniseMock != nil {
		return d.synchroniseMock(id, hash)
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.synchronising, 0, 1) {
		return errBusy
	}
	defer atomic.StoreInt32(&d.synchronising, 0)

	// Post a user notification of the sync (only once per session)
	if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
		glog.V(logger.Info).Infoln("Block synchronisation started")
	}
	// Abort if the queue still contains some leftover data
	if d.queue.GetHeadResult() != nil {
		return errPendingQueue
	}
	// Reset the queue, peer set and wake channels to clean any internal leftover state
	d.queue.Reset()
	d.peers.Reset()

	for _, ch := range []chan bool{d.blockWakeCh, d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	// Reset and ephemeral sync statistics
	d.syncStatsLock.Lock()
	d.syncStatsStateTotal = 0
	d.syncStatsStateDone = 0
	d.syncStatsLock.Unlock()

	// Create cancel channel for aborting mid-flight
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelLock.Unlock()

	// Set the requested sync mode, unless it's forbidden
	d.mode = mode
	if d.mode == FastSync && d.noFast {
		d.mode = FullSync
	}
	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(p, hash, td)
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hash.
func (d *Downloader) syncWithPeer(p *peer, hash common.Hash, td *big.Int) (err error) {
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

	glog.V(logger.Debug).Infof("Synchronising with the network using: %s [eth/%d]", p.id, p.version)
	defer func(start time.Time) {
		glog.V(logger.Debug).Infof("Synchronisation terminated after %v", time.Since(start))
	}(time.Now())

	switch {
	case p.version == 61:
		// Look up the sync boundaries: the common ancestor and the target block
		latest, err := d.fetchHeight61(p)
		if err != nil {
			return err
		}
		origin, err := d.findAncestor61(p)
		if err != nil {
			return err
		}
		d.syncStatsLock.Lock()
		if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
			d.syncStatsChainOrigin = origin
		}
		d.syncStatsChainHeight = latest
		d.syncStatsLock.Unlock()

		// Initiate the sync using a  concurrent hash and block retrieval algorithm
		if d.syncInitHook != nil {
			d.syncInitHook(origin, latest)
		}
		d.queue.Prepare(origin+1, d.mode, 0)

		errc := make(chan error, 2)
		go func() { errc <- d.fetchHashes61(p, td, origin+1) }()
		go func() { errc <- d.fetchBlocks61(origin + 1) }()

		// If any fetcher fails, cancel the other
		if err := <-errc; err != nil {
			d.cancel()
			<-errc
			return err
		}
		return <-errc

	case p.version >= 62:
		// Look up the sync boundaries: the common ancestor and the target block
		latest, err := d.fetchHeight(p)
		if err != nil {
			return err
		}
		origin, err := d.findAncestor(p)
		if err != nil {
			return err
		}
		d.syncStatsLock.Lock()
		if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
			d.syncStatsChainOrigin = origin
		}
		d.syncStatsChainHeight = latest
		d.syncStatsLock.Unlock()

		// Initiate the sync using a concurrent header and content retrieval algorithm
		pivot := uint64(0)
		switch d.mode {
		case LightSync:
			pivot = latest

		case FastSync:
			// Calculate the new fast/slow sync pivot point
			pivotOffset, err := rand.Int(rand.Reader, big.NewInt(int64(fsPivotInterval)))
			if err != nil {
				panic(fmt.Sprintf("Failed to access crypto random source: %v", err))
			}
			if latest > uint64(fsMinFullBlocks)+pivotOffset.Uint64() {
				pivot = latest - uint64(fsMinFullBlocks) - pivotOffset.Uint64()
			}
			// If the point is below the origin, move origin back to ensure state download
			if pivot < origin {
				if pivot > 0 {
					origin = pivot - 1
				} else {
					origin = 0
				}
			}
			glog.V(logger.Debug).Infof("Fast syncing until pivot block #%d", pivot)
		}
		d.queue.Prepare(origin+1, d.mode, pivot)

		if d.syncInitHook != nil {
			d.syncInitHook(origin, latest)
		}
		errc := make(chan error, 4)
		go func() { errc <- d.fetchHeaders(p, td, origin+1) }() // Headers are always retrieved
		go func() { errc <- d.fetchBodies(origin + 1) }()       // Bodies are retrieved during normal and fast sync
		go func() { errc <- d.fetchReceipts(origin + 1) }()     // Receipts are retrieved during fast sync
		go func() { errc <- d.fetchNodeData() }()               // Node state data is retrieved during fast sync

		// If any fetcher fails, cancel the others
		var fail error
		for i := 0; i < cap(errc); i++ {
			if err := <-errc; err != nil {
				if fail == nil {
					fail = err
					d.cancel()
				}
			}
		}
		return fail

	default:
		// Something very wrong, stop right here
		glog.V(logger.Error).Infof("Unsupported eth protocol: %d", p.version)
		return errBadPeer
	}
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

// fetchHeight61 retrieves the head block of the remote peer to aid in estimating
// the total time a pending synchronisation would take.
func (d *Downloader) fetchHeight61(p *peer) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: retrieving remote chain height", p)

	// Request the advertised remote head block and wait for the response
	go p.getBlocks([]common.Hash{p.head})

	timeout := time.After(blockSoftTTL)
	for {
		select {
		case <-d.cancelCh:
			return 0, errCancelBlockFetch

		case <-d.headerCh:
			// Out of bounds eth/62 block headers received, ignore them

		case <-d.bodyCh:
			// Out of bounds eth/62 block bodies received, ignore them

		case <-d.hashCh:
			// Out of bounds hashes received, ignore them

		case packet := <-d.blockCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received blocks from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			blocks := packet.(*blockPack).blocks
			if len(blocks) != 1 {
				glog.V(logger.Debug).Infof("%v: invalid number of head blocks: %d != 1", p, len(blocks))
				return 0, errBadPeer
			}
			return blocks[0].NumberU64(), nil

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head block timeout", p)
			return 0, errTimeout
		}
	}
}

// findAncestor61 tries to locate the common ancestor block of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N blocks should already get us a match.
// In the rare scenario when we ended up on a long reorganization (i.e. none of
// the head blocks match), we do a binary search to find the common ancestor.
func (d *Downloader) findAncestor61(p *peer) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: looking for common ancestor", p)

	// Request out head blocks to short circuit ancestor location
	head := d.headBlock().NumberU64()
	from := int64(head) - int64(MaxHashFetch) + 1
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

		case packet := <-d.hashCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			hashes := packet.(*hashPack).hashes
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

		case <-d.headerCh:
			// Out of bounds eth/62 block headers received, ignore them

		case <-d.bodyCh:
			// Out of bounds eth/62 block bodies received, ignore them

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head hash timeout", p)
			return 0, errTimeout
		}
	}
	// If the head fetch already found an ancestor, return
	if !common.EmptyHash(hash) {
		glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x…]", p, number, hash[:4])
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

			case packet := <-d.hashCh:
				// Discard anything not from the origin peer
				if packet.PeerId() != p.id {
					glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", packet.PeerId())
					break
				}
				// Make sure the peer actually gave something valid
				hashes := packet.(*hashPack).hashes
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
					glog.V(logger.Debug).Infof("%v: non requested hash #%d [%x…], instead of #%d", p, block.NumberU64(), block.Hash().Bytes()[:4], check)
					return 0, errBadPeer
				}
				start = check

			case <-d.blockCh:
				// Out of bounds blocks received, ignore them

			case <-d.headerCh:
				// Out of bounds eth/62 block headers received, ignore them

			case <-d.bodyCh:
				// Out of bounds eth/62 block bodies received, ignore them

			case <-timeout:
				glog.V(logger.Debug).Infof("%v: search hash timeout", p)
				return 0, errTimeout
			}
		}
	}
	return start, nil
}

// fetchHashes61 keeps retrieving hashes from the requested number, until no more
// are returned, potentially throttling on the way.
func (d *Downloader) fetchHashes61(p *peer, td *big.Int, from uint64) error {
	glog.V(logger.Debug).Infof("%v: downloading hashes from #%d", p, from)

	// Create a timeout timer, and the associated hash fetcher
	request := time.Now()       // time of the last fetch request
	timeout := time.NewTimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	getHashes := func(from uint64) {
		glog.V(logger.Detail).Infof("%v: fetching %d hashes from #%d", p, MaxHashFetch, from)

		go p.getAbsHashes(from, MaxHashFetch)
		request = time.Now()
		timeout.Reset(hashTTL)
	}
	// Start pulling hashes, until all are exhausted
	getHashes(from)
	gotHashes := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelHashFetch

		case <-d.headerCh:
			// Out of bounds eth/62 block headers received, ignore them

		case <-d.bodyCh:
			// Out of bounds eth/62 block bodies received, ignore them

		case packet := <-d.hashCh:
			// Make sure the active peer is giving us the hashes
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)", packet.PeerId())
				break
			}
			hashReqTimer.UpdateSince(request)
			timeout.Stop()

			// If no more hashes are inbound, notify the block fetcher and return
			if packet.Items() == 0 {
				glog.V(logger.Debug).Infof("%v: no available hashes", p)

				select {
				case d.blockWakeCh <- false:
				case <-d.cancelCh:
				}
				// If no hashes were retrieved at all, the peer violated it's TD promise that it had a
				// better chain compared to ours. The only exception is if it's promised blocks were
				// already imported by other means (e.g. fecher):
				//
				// R <remote peer>, L <local node>: Both at block 10
				// R: Mine block 11, and propagate it to L
				// L: Queue block 11 for import
				// L: Notice that R's head and TD increased compared to ours, start sync
				// L: Import of block 11 finishes
				// L: Sync begins, and finds common ancestor at 11
				// L: Request new hashes up from 11 (R's TD was higher, it must have something)
				// R: Nothing to give
				if !gotHashes && td.Cmp(d.getTd(d.headBlock().Hash())) > 0 {
					return errStallingPeer
				}
				return nil
			}
			gotHashes = true
			hashes := packet.(*hashPack).hashes

			// Otherwise insert all the new hashes, aborting in case of junk
			glog.V(logger.Detail).Infof("%v: scheduling %d hashes from #%d", p, len(hashes), from)

			inserts := d.queue.Schedule61(hashes, true)
			if len(inserts) != len(hashes) {
				glog.V(logger.Debug).Infof("%v: stale hashes", p)
				return errBadPeer
			}
			// Notify the block fetcher of new hashes, but stop if queue is full
			if d.queue.PendingBlocks() < maxQueuedHashes {
				// We still have hashes to fetch, send continuation wake signal (potential)
				select {
				case d.blockWakeCh <- true:
				default:
				}
			} else {
				// Hash limit reached, send a termination wake signal (enforced)
				select {
				case d.blockWakeCh <- false:
				case <-d.cancelCh:
				}
				return nil
			}
			// Queue not yet full, fetch the next batch
			from += uint64(len(hashes))
			getHashes(from)

		case <-timeout.C:
			glog.V(logger.Debug).Infof("%v: hash request timed out", p)
			hashTimeoutMeter.Mark(1)
			return errTimeout
		}
	}
}

// fetchBlocks61 iteratively downloads the scheduled hashes, taking any available
// peers, reserving a chunk of blocks for each, waiting for delivery and also
// periodically checking for timeouts.
func (d *Downloader) fetchBlocks61(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading blocks from #%d", from)
	defer glog.V(logger.Debug).Infof("Block download terminated")

	// Create a timeout timer for scheduling expiration tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Fetch blocks until the hash fetcher's done
	finished := false
	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case <-d.headerCh:
			// Out of bounds eth/62 block headers received, ignore them

		case <-d.bodyCh:
			// Out of bounds eth/62 block bodies received, ignore them

		case packet := <-d.blockCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if peer := d.peers.Peer(packet.PeerId()); peer != nil {
				// Deliver the received chunk of blocks, and demote in case of errors
				blocks := packet.(*blockPack).blocks
				err := d.queue.DeliverBlocks(peer.id, blocks)
				switch err {
				case nil:
					// If no blocks were delivered, demote the peer (need the delivery above)
					if len(blocks) == 0 {
						peer.Demote()
						peer.SetBlocksIdle()
						glog.V(logger.Detail).Infof("%s: no blocks delivered", peer)
						break
					}
					// All was successful, promote the peer and potentially start processing
					peer.Promote()
					peer.SetBlocksIdle()
					glog.V(logger.Detail).Infof("%s: delivered %d blocks", peer, len(blocks))
					go d.process()

				case errInvalidChain:
					// The hash chain is invalid (blocks are not ordered properly), abort
					return err

				case errNoFetchesPending:
					// Peer probably timed out with its delivery but came through
					// in the end, demote, but allow to to pull from this peer.
					peer.Demote()
					peer.SetBlocksIdle()
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
					peer.SetBlocksIdle()
					glog.V(logger.Detail).Infof("%s: delivery partially failed: %v", peer, err)
					go d.process()
				}
			}
			// Blocks arrived, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-d.blockWakeCh:
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
			for _, pid := range d.queue.ExpireBlocks(blockHardTTL) {
				if peer := d.peers.Peer(pid); peer != nil {
					peer.Demote()
					glog.V(logger.Detail).Infof("%s: block delivery timeout", peer)
				}
			}
			// If there's nothing more to fetch, wait or terminate
			if d.queue.PendingBlocks() == 0 {
				if !d.queue.InFlightBlocks() && finished {
					glog.V(logger.Debug).Infof("Block fetching completed")
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			throttled := false
			idles, total := d.peers.BlockIdlePeers()

			for _, peer := range idles {
				// Short circuit if throttling activated
				if d.queue.ShouldThrottleBlocks() {
					throttled = true
					break
				}
				// Reserve a chunk of hashes for a peer. A nil can mean either that
				// no more hashes are available, or that the peer is known not to
				// have them.
				request := d.queue.ReserveBlocks(peer, peer.BlockCapacity())
				if request == nil {
					continue
				}
				if glog.V(logger.Detail) {
					glog.Infof("%s: requesting %d blocks", peer, len(request.Hashes))
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if err := peer.Fetch61(request); err != nil {
					// Although we could try and make an attempt to fix this, this error really
					// means that we've double allocated a fetch task to a peer. If that is the
					// case, the internal state of the downloader and the queue is very wrong so
					// better hard crash and note the error instead of silently accumulating into
					// a much bigger issue.
					panic(fmt.Sprintf("%v: fetch assignment failed, hard panic", peer))
					d.queue.CancelBlocks(request) // noop for now
				}
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !throttled && !d.queue.InFlightBlocks() && len(idles) == total {
				return errPeersUnavailable
			}
		}
	}
}

// fetchHeight retrieves the head header of the remote peer to aid in estimating
// the total time a pending synchronisation would take.
func (d *Downloader) fetchHeight(p *peer) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: retrieving remote chain height", p)

	// Request the advertised remote head block and wait for the response
	go p.getRelHeaders(p.head, 1, 0, false)

	timeout := time.After(headerTTL)
	for {
		select {
		case <-d.cancelCh:
			return 0, errCancelBlockFetch

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received headers from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			headers := packet.(*headerPack).headers
			if len(headers) != 1 {
				glog.V(logger.Debug).Infof("%v: invalid number of head headers: %d != 1", p, len(headers))
				return 0, errBadPeer
			}
			return headers[0].Number.Uint64(), nil

		case <-d.bodyCh:
			// Out of bounds block bodies received, ignore them

		case <-d.hashCh:
			// Out of bounds eth/61 hashes received, ignore them

		case <-d.blockCh:
			// Out of bounds eth/61 blocks received, ignore them

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head header timeout", p)
			return 0, errTimeout
		}
	}
}

// findAncestor tries to locate the common ancestor link of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N links should already get us a match.
// In the rare scenario when we ended up on a long reorganization (i.e. none of
// the head links match), we do a binary search to find the common ancestor.
func (d *Downloader) findAncestor(p *peer) (uint64, error) {
	glog.V(logger.Debug).Infof("%v: looking for common ancestor", p)

	// Request our head headers to short circuit ancestor location
	head := d.headHeader().Number.Uint64()
	if d.mode == FullSync {
		head = d.headBlock().NumberU64()
	} else if d.mode == FastSync {
		head = d.headFastBlock().NumberU64()
	}
	from := int64(head) - int64(MaxHeaderFetch) + 1
	if from < 0 {
		from = 0
	}
	go p.getAbsHeaders(uint64(from), MaxHeaderFetch, 0, false)

	// Wait for the remote response to the head fetch
	number, hash := uint64(0), common.Hash{}
	timeout := time.After(hashTTL)

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCancelHashFetch

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received headers from incorrect peer(%s)", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			headers := packet.(*headerPack).headers
			if len(headers) == 0 {
				glog.V(logger.Debug).Infof("%v: empty head header set", p)
				return 0, errEmptyHeaderSet
			}
			// Check if a common ancestor was found
			finished = true
			for i := len(headers) - 1; i >= 0; i-- {
				if (d.mode != LightSync && d.hasBlock(headers[i].Hash())) || (d.mode == LightSync && d.hasHeader(headers[i].Hash())) {
					number, hash = headers[i].Number.Uint64(), headers[i].Hash()
					break
				}
			}

		case <-d.bodyCh:
			// Out of bounds block bodies received, ignore them

		case <-d.hashCh:
			// Out of bounds eth/61 hashes received, ignore them

		case <-d.blockCh:
			// Out of bounds eth/61 blocks received, ignore them

		case <-timeout:
			glog.V(logger.Debug).Infof("%v: head header timeout", p)
			return 0, errTimeout
		}
	}
	// If the head fetch already found an ancestor, return
	if !common.EmptyHash(hash) {
		glog.V(logger.Debug).Infof("%v: common ancestor: #%d [%x…]", p, number, hash[:4])
		return number, nil
	}
	// Ancestor not found, we need to binary search over our chain
	start, end := uint64(0), head
	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		timeout := time.After(hashTTL)
		go p.getAbsHeaders(uint64(check), 1, 0, false)

		// Wait until a reply arrives to this request
		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCancelHashFetch

			case packer := <-d.headerCh:
				// Discard anything not from the origin peer
				if packer.PeerId() != p.id {
					glog.V(logger.Debug).Infof("Received headers from incorrect peer(%s)", packer.PeerId())
					break
				}
				// Make sure the peer actually gave something valid
				headers := packer.(*headerPack).headers
				if len(headers) != 1 {
					glog.V(logger.Debug).Infof("%v: invalid search header set (%d)", p, len(headers))
					return 0, errBadPeer
				}
				arrived = true

				// Modify the search interval based on the response
				if (d.mode == FullSync && !d.hasBlock(headers[0].Hash())) || (d.mode != FullSync && !d.hasHeader(headers[0].Hash())) {
					end = check
					break
				}
				header := d.getHeader(headers[0].Hash()) // Independent of sync mode, header surely exists
				if header.Number.Uint64() != check {
					glog.V(logger.Debug).Infof("%v: non requested header #%d [%x…], instead of #%d", p, header.Number, header.Hash().Bytes()[:4], check)
					return 0, errBadPeer
				}
				start = check

			case <-d.bodyCh:
				// Out of bounds block bodies received, ignore them

			case <-d.hashCh:
				// Out of bounds eth/61 hashes received, ignore them

			case <-d.blockCh:
				// Out of bounds eth/61 blocks received, ignore them

			case <-timeout:
				glog.V(logger.Debug).Infof("%v: search header timeout", p)
				return 0, errTimeout
			}
		}
	}
	return start, nil
}

// fetchHeaders keeps retrieving headers from the requested number, until no more
// are returned, potentially throttling on the way.
//
// The queue parameter can be used to switch between queuing headers for block
// body download too, or directly import as pure header chains.
func (d *Downloader) fetchHeaders(p *peer, td *big.Int, from uint64) error {
	glog.V(logger.Debug).Infof("%v: downloading headers from #%d", p, from)
	defer glog.V(logger.Debug).Infof("%v: header download terminated", p)

	// Calculate the pivoting point for switching from fast to slow sync
	pivot := d.queue.FastSyncPivot()

	// Keep a count of uncertain headers to roll back
	rollback := []*types.Header{}
	defer func() {
		if len(rollback) > 0 {
			// Flatten the headers and roll them back
			hashes := make([]common.Hash, len(rollback))
			for i, header := range rollback {
				hashes[i] = header.Hash()
			}
			lh, lfb, lb := d.headHeader().Number, d.headFastBlock().Number(), d.headBlock().Number()
			d.rollback(hashes)
			glog.V(logger.Warn).Infof("Rolled back %d headers (LH: %d->%d, FB: %d->%d, LB: %d->%d)",
				len(hashes), lh, d.headHeader().Number, lfb, d.headFastBlock().Number(), lb, d.headBlock().Number())

			// If we're already past the pivot point, this could be an attack, disable fast sync
			if rollback[len(rollback)-1].Number.Uint64() > pivot {
				d.noFast = true
			}
		}
	}()

	// Create a timeout timer, and the associated hash fetcher
	request := time.Now()       // time of the last fetch request
	timeout := time.NewTimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	getHeaders := func(from uint64) {
		glog.V(logger.Detail).Infof("%v: fetching %d headers from #%d", p, MaxHeaderFetch, from)

		go p.getAbsHeaders(from, MaxHeaderFetch, 0, false)
		request = time.Now()
		timeout.Reset(headerTTL)
	}
	// Start pulling headers, until all are exhausted
	getHeaders(from)
	gotHeaders := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelHeaderFetch

		case <-d.hashCh:
			// Out of bounds eth/61 hashes received, ignore them

		case <-d.blockCh:
			// Out of bounds eth/61 blocks received, ignore them

		case packet := <-d.headerCh:
			// Make sure the active peer is giving us the headers
			if packet.PeerId() != p.id {
				glog.V(logger.Debug).Infof("Received headers from incorrect peer (%s)", packet.PeerId())
				break
			}
			headerReqTimer.UpdateSince(request)
			timeout.Stop()

			// If no more headers are inbound, notify the content fetchers and return
			if packet.Items() == 0 {
				glog.V(logger.Debug).Infof("%v: no available headers", p)

				for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
					select {
					case ch <- false:
					case <-d.cancelCh:
					}
				}
				// If no headers were retrieved at all, the peer violated it's TD promise that it had a
				// better chain compared to ours. The only exception is if it's promised blocks were
				// already imported by other means (e.g. fecher):
				//
				// R <remote peer>, L <local node>: Both at block 10
				// R: Mine block 11, and propagate it to L
				// L: Queue block 11 for import
				// L: Notice that R's head and TD increased compared to ours, start sync
				// L: Import of block 11 finishes
				// L: Sync begins, and finds common ancestor at 11
				// L: Request new headers up from 11 (R's TD was higher, it must have something)
				// R: Nothing to give
				if !gotHeaders && td.Cmp(d.getTd(d.headBlock().Hash())) > 0 {
					return errStallingPeer
				}
				// If fast or light syncing, ensure promised headers are indeed delivered. This is
				// needed to detect scenarios where an attacker feeds a bad pivot and then bails out
				// of delivering the post-pivot blocks that would flag the invalid content.
				//
				// This check cannot be executed "as is" for full imports, since blocks may still be
				// queued for processing when the header download completes. However, as long as the
				// peer gave us something useful, we're already happy/progressed (above check).
				if d.mode == FastSync || d.mode == LightSync {
					if td.Cmp(d.getTd(d.headHeader().Hash())) > 0 {
						return errStallingPeer
					}
				}
				rollback = nil
				return nil
			}
			gotHeaders = true
			headers := packet.(*headerPack).headers

			// Otherwise insert all the new headers, aborting in case of junk
			glog.V(logger.Detail).Infof("%v: schedule %d headers from #%d", p, len(headers), from)

			if d.mode == FastSync || d.mode == LightSync {
				// Collect the yet unknown headers to mark them as uncertain
				unknown := make([]*types.Header, 0, len(headers))
				for _, header := range headers {
					if !d.hasHeader(header.Hash()) {
						unknown = append(unknown, header)
					}
				}
				// If we're importing pure headers, verify based on their recentness
				frequency := fsHeaderCheckFrequency
				if headers[len(headers)-1].Number.Uint64()+uint64(fsHeaderForceVerify) > pivot {
					frequency = 1
				}
				if n, err := d.insertHeaders(headers, frequency); err != nil {
					glog.V(logger.Debug).Infof("%v: invalid header #%d [%x…]: %v", p, headers[n].Number, headers[n].Hash().Bytes()[:4], err)
					return errInvalidChain
				}
				// All verifications passed, store newly found uncertain headers
				rollback = append(rollback, unknown...)
				if len(rollback) > fsHeaderSafetyNet {
					rollback = append(rollback[:0], rollback[len(rollback)-fsHeaderSafetyNet:]...)
				}
			}
			if d.mode == FullSync || d.mode == FastSync {
				inserts := d.queue.Schedule(headers, from)
				if len(inserts) != len(headers) {
					glog.V(logger.Debug).Infof("%v: stale headers", p)
					return errBadPeer
				}
			}
			// Notify the content fetchers of new headers, but stop if queue is full
			cont := d.queue.PendingBlocks() < maxQueuedHeaders || d.queue.PendingReceipts() < maxQueuedHeaders
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
				if cont {
					// We still have headers to fetch, send continuation wake signal (potential)
					select {
					case ch <- true:
					default:
					}
				} else {
					// Header limit reached, send a termination wake signal (enforced)
					select {
					case ch <- false:
					case <-d.cancelCh:
					}
					return nil
				}
			}
			// Queue not yet full, fetch the next batch
			from += uint64(len(headers))
			getHeaders(from)

		case <-timeout.C:
			// Header retrieval timed out, consider the peer bad and drop
			glog.V(logger.Debug).Infof("%v: header request timed out", p)
			headerTimeoutMeter.Mark(1)
			d.dropPeer(p.id)

			// Finish the sync gracefully instead of dumping the gathered data though
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh, d.stateWakeCh} {
				select {
				case ch <- false:
				case <-d.cancelCh:
				}
			}
			return nil
		}
	}
}

// fetchBodies iteratively downloads the scheduled block bodies, taking any
// available peers, reserving a chunk of blocks for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchBodies(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading block bodies from #%d", from)

	var (
		deliver = func(packet dataPack) error {
			pack := packet.(*bodyPack)
			return d.queue.DeliverBodies(pack.peerId, pack.transactions, pack.uncles)
		}
		expire   = func() []string { return d.queue.ExpireBodies(bodyHardTTL) }
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchBodies(req) }
		capacity = func(p *peer) int { return p.BlockCapacity() }
		setIdle  = func(p *peer) { p.SetBodiesIdle() }
	)
	err := d.fetchParts(errCancelBodyFetch, d.bodyCh, deliver, d.bodyWakeCh, expire,
		d.queue.PendingBlocks, d.queue.InFlightBlocks, d.queue.ShouldThrottleBlocks, d.queue.ReserveBodies,
		d.bodyFetchHook, fetch, d.queue.CancelBodies, capacity, d.peers.BodyIdlePeers, setIdle, "Body")

	glog.V(logger.Debug).Infof("Block body download terminated: %v", err)
	return err
}

// fetchReceipts iteratively downloads the scheduled block receipts, taking any
// available peers, reserving a chunk of receipts for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchReceipts(from uint64) error {
	glog.V(logger.Debug).Infof("Downloading receipts from #%d", from)

	var (
		deliver = func(packet dataPack) error {
			pack := packet.(*receiptPack)
			return d.queue.DeliverReceipts(pack.peerId, pack.receipts)
		}
		expire   = func() []string { return d.queue.ExpireReceipts(receiptHardTTL) }
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchReceipts(req) }
		capacity = func(p *peer) int { return p.ReceiptCapacity() }
		setIdle  = func(p *peer) { p.SetReceiptsIdle() }
	)
	err := d.fetchParts(errCancelReceiptFetch, d.receiptCh, deliver, d.receiptWakeCh, expire,
		d.queue.PendingReceipts, d.queue.InFlightReceipts, d.queue.ShouldThrottleReceipts, d.queue.ReserveReceipts,
		d.receiptFetchHook, fetch, d.queue.CancelReceipts, capacity, d.peers.ReceiptIdlePeers, setIdle, "Receipt")

	glog.V(logger.Debug).Infof("Receipt download terminated: %v", err)
	return err
}

// fetchNodeData iteratively downloads the scheduled state trie nodes, taking any
// available peers, reserving a chunk of nodes for each, waiting for delivery and
// also periodically checking for timeouts.
func (d *Downloader) fetchNodeData() error {
	glog.V(logger.Debug).Infof("Downloading node state data")

	var (
		deliver = func(packet dataPack) error {
			start := time.Now()
			return d.queue.DeliverNodeData(packet.PeerId(), packet.(*statePack).states, func(err error, delivered int) {
				if err != nil {
					// If the node data processing failed, the root hash is very wrong, abort
					glog.V(logger.Error).Infof("peer %d: state processing failed: %v", packet.PeerId(), err)
					d.cancel()
					return
				}
				// Processing succeeded, notify state fetcher and processor of continuation
				if d.queue.PendingNodeData() == 0 {
					go d.process()
				} else {
					select {
					case d.stateWakeCh <- true:
					default:
					}
				}
				// Log a message to the user and return
				d.syncStatsLock.Lock()
				defer d.syncStatsLock.Unlock()

				d.syncStatsStateDone += uint64(delivered)
				glog.V(logger.Info).Infof("imported %d state entries in %v: processed %d in total", delivered, time.Since(start), d.syncStatsStateDone)
			})
		}
		expire   = func() []string { return d.queue.ExpireNodeData(stateHardTTL) }
		throttle = func() bool { return false }
		reserve  = func(p *peer, count int) (*fetchRequest, bool, error) {
			return d.queue.ReserveNodeData(p, count), false, nil
		}
		fetch    = func(p *peer, req *fetchRequest) error { return p.FetchNodeData(req) }
		capacity = func(p *peer) int { return p.NodeDataCapacity() }
		setIdle  = func(p *peer) { p.SetNodeDataIdle() }
	)
	err := d.fetchParts(errCancelStateFetch, d.stateCh, deliver, d.stateWakeCh, expire,
		d.queue.PendingNodeData, d.queue.InFlightNodeData, throttle, reserve, nil, fetch,
		d.queue.CancelNodeData, capacity, d.peers.NodeDataIdlePeers, setIdle, "State")

	glog.V(logger.Debug).Infof("Node state data download terminated: %v", err)
	return err
}

// fetchParts iteratively downloads scheduled block parts, taking any available
// peers, reserving a chunk of fetch requests for each, waiting for delivery and
// also periodically checking for timeouts.
func (d *Downloader) fetchParts(errCancel error, deliveryCh chan dataPack, deliver func(packet dataPack) error, wakeCh chan bool,
	expire func() []string, pending func() int, inFlight func() bool, throttle func() bool, reserve func(*peer, int) (*fetchRequest, bool, error),
	fetchHook func([]*types.Header), fetch func(*peer, *fetchRequest) error, cancel func(*fetchRequest), capacity func(*peer) int,
	idle func() ([]*peer, int), setIdle func(*peer), kind string) error {

	// Create a ticker to detect expired retrieval tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Prepare the queue and fetch block parts until the block header fetcher's done
	finished := false
	for {
		select {
		case <-d.cancelCh:
			return errCancel

		case <-d.hashCh:
			// Out of bounds eth/61 hashes received, ignore them

		case <-d.blockCh:
			// Out of bounds eth/61 blocks received, ignore them

		case packet := <-deliveryCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if peer := d.peers.Peer(packet.PeerId()); peer != nil {
				// Deliver the received chunk of data, and demote in case of errors
				switch err := deliver(packet); err {
				case nil:
					// If no blocks were delivered, demote the peer (need the delivery above to clean internal queue!)
					if packet.Items() == 0 {
						peer.Demote()
						setIdle(peer)
						glog.V(logger.Detail).Infof("%s: no %s delivered", peer, strings.ToLower(kind))
						break
					}
					// All was successful, promote the peer and potentially start processing
					peer.Promote()
					setIdle(peer)
					glog.V(logger.Detail).Infof("%s: delivered %s %s(s)", peer, packet.Stats(), strings.ToLower(kind))
					go d.process()

				case errInvalidChain:
					// The hash chain is invalid (blocks are not ordered properly), abort
					return err

				case errNoFetchesPending:
					// Peer probably timed out with its delivery but came through
					// in the end, demote, but allow to to pull from this peer.
					peer.Demote()
					setIdle(peer)
					glog.V(logger.Detail).Infof("%s: out of bound %s delivery", peer, strings.ToLower(kind))

				case errStaleDelivery:
					// Delivered something completely else than requested, usually
					// caused by a timeout and delivery during a new sync cycle.
					// Don't set it to idle as the original request should still be
					// in flight.
					peer.Demote()
					glog.V(logger.Detail).Infof("%s: %s stale delivery", peer, strings.ToLower(kind))

				default:
					// Peer did something semi-useful, demote but keep it around
					peer.Demote()
					setIdle(peer)
					glog.V(logger.Detail).Infof("%s: %s delivery partially failed: %v", peer, strings.ToLower(kind), err)
					go d.process()
				}
			}
			// Blocks assembled, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-wakeCh:
			// The header fetcher sent a continuation flag, check if it's done
			if !cont {
				finished = true
			}
			// Headers arrive, try to update the progress
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
			// Check for fetch request timeouts and demote the responsible peers
			for _, pid := range expire() {
				if peer := d.peers.Peer(pid); peer != nil {
					peer.Demote()
					setIdle(peer)
					glog.V(logger.Detail).Infof("%s: %s delivery timeout", peer, strings.ToLower(kind))
				}
			}
			// If there's nothing more to fetch, wait or terminate
			if pending() == 0 {
				if !inFlight() && finished {
					glog.V(logger.Debug).Infof("%s fetching completed", kind)
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			progressed, throttled, running := false, false, inFlight()
			idles, total := idle()

			for _, peer := range idles {
				// Short circuit if throttling activated
				if throttle() {
					throttled = true
					break
				}
				// Reserve a chunk of fetches for a peer. A nil can mean either that
				// no more headers are available, or that the peer is known not to
				// have them.
				request, progress, err := reserve(peer, capacity(peer))
				if err != nil {
					return err
				}
				if progress {
					progressed = true
					go d.process()
				}
				if request == nil {
					continue
				}
				if glog.V(logger.Detail) {
					if len(request.Headers) > 0 {
						glog.Infof("%s: requesting %d %s(s), first at #%d", peer, len(request.Headers), strings.ToLower(kind), request.Headers[0].Number)
					} else {
						glog.Infof("%s: requesting %d %s(s)", peer, len(request.Hashes), strings.ToLower(kind))
					}
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if fetchHook != nil {
					fetchHook(request.Headers)
				}
				if err := fetch(peer, request); err != nil {
					// Although we could try and make an attempt to fix this, this error really
					// means that we've double allocated a fetch task to a peer. If that is the
					// case, the internal state of the downloader and the queue is very wrong so
					// better hard crash and note the error instead of silently accumulating into
					// a much bigger issue.
					panic(fmt.Sprintf("%v: %s fetch assignment failed, hard panic", peer, strings.ToLower(kind)))
					cancel(request) // noop for now
				}
				running = true
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !progressed && !throttled && !running && len(idles) == total && pending() > 0 {
				return errPeersUnavailable
			}
		}
	}
}

// process takes fetch results from the queue and tries to import them into the
// chain. The type of import operation will depend on the result contents:
//  -
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
	// the fresh results might have been rejected entry to to this present thread
	// not yet releasing the `processing` state.
	defer func() {
		if atomic.LoadInt32(&d.interrupt) == 0 && d.queue.GetHeadResult() != nil {
			d.process()
		}
	}()
	// Release the lock upon exit (note, before checking for reentry!)
	// the import statistics to zero.
	defer atomic.StoreInt32(&d.processing, 0)

	// Repeat the processing as long as there are results to process
	for {
		// Fetch the next batch of results
		pivot := d.queue.FastSyncPivot() // Fetch pivot before results to prevent reset race
		results := d.queue.TakeResults()
		if len(results) == 0 {
			return
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		// Actually import the blocks
		if glog.V(logger.Debug) {
			first, last := results[0].Header, results[len(results)-1].Header
			glog.Infof("Inserting chain with %d items (#%d [%x…] - #%d [%x…])", len(results), first.Number, first.Hash().Bytes()[:4], last.Number, last.Hash().Bytes()[:4])
		}
		for len(results) != 0 {
			// Check for any termination requests
			if atomic.LoadInt32(&d.interrupt) == 1 {
				return
			}
			// Retrieve the a batch of results to import
			var (
				blocks   = make([]*types.Block, 0, maxResultsProcess)
				receipts = make([]types.Receipts, 0, maxResultsProcess)
			)
			items := int(math.Min(float64(len(results)), float64(maxResultsProcess)))
			for _, result := range results[:items] {
				switch {
				case d.mode == FullSync:
					blocks = append(blocks, types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles))
				case d.mode == FastSync:
					blocks = append(blocks, types.NewBlockWithHeader(result.Header).WithBody(result.Transactions, result.Uncles))
					if result.Header.Number.Uint64() <= pivot {
						receipts = append(receipts, result.Receipts)
					}
				}
			}
			// Try to process the results, aborting if there's an error
			var (
				err   error
				index int
			)
			switch {
			case len(receipts) > 0:
				index, err = d.insertReceipts(blocks, receipts)
				if err == nil && blocks[len(blocks)-1].NumberU64() == pivot {
					glog.V(logger.Debug).Infof("Committing block #%d [%x…] as the new head", blocks[len(blocks)-1].Number(), blocks[len(blocks)-1].Hash().Bytes()[:4])
					index, err = len(blocks)-1, d.commitHeadBlock(blocks[len(blocks)-1].Hash())
				}
			default:
				index, err = d.insertBlocks(blocks)
			}
			if err != nil {
				glog.V(logger.Debug).Infof("Result #%d [%x…] processing failed: %v", results[index].Header.Number, results[index].Header.Hash().Bytes()[:4], err)
				d.cancel()
				return
			}
			// Shift the results to the next batch
			results = results[items:]
		}
	}
}

// DeliverHashes injects a new batch of hashes received from a remote node into
// the download schedule. This is usually invoked through the BlockHashesMsg by
// the protocol handler.
func (d *Downloader) DeliverHashes(id string, hashes []common.Hash) (err error) {
	return d.deliver(id, d.hashCh, &hashPack{id, hashes}, hashInMeter, hashDropMeter)
}

// DeliverBlocks injects a new batch of blocks received from a remote node.
// This is usually invoked through the BlocksMsg by the protocol handler.
func (d *Downloader) DeliverBlocks(id string, blocks []*types.Block) (err error) {
	return d.deliver(id, d.blockCh, &blockPack{id, blocks}, blockInMeter, blockDropMeter)
}

// DeliverHeaders injects a new batch of blck headers received from a remote
// node into the download schedule.
func (d *Downloader) DeliverHeaders(id string, headers []*types.Header) (err error) {
	return d.deliver(id, d.headerCh, &headerPack{id, headers}, headerInMeter, headerDropMeter)
}

// DeliverBodies injects a new batch of block bodies received from a remote node.
func (d *Downloader) DeliverBodies(id string, transactions [][]*types.Transaction, uncles [][]*types.Header) (err error) {
	return d.deliver(id, d.bodyCh, &bodyPack{id, transactions, uncles}, bodyInMeter, bodyDropMeter)
}

// DeliverReceipts injects a new batch of receipts received from a remote node.
func (d *Downloader) DeliverReceipts(id string, receipts [][]*types.Receipt) (err error) {
	return d.deliver(id, d.receiptCh, &receiptPack{id, receipts}, receiptInMeter, receiptDropMeter)
}

// DeliverNodeData injects a new batch of node state data received from a remote node.
func (d *Downloader) DeliverNodeData(id string, data [][]byte) (err error) {
	return d.deliver(id, d.stateCh, &statePack{id, data}, stateInMeter, stateDropMeter)
}

// deliver injects a new batch of data received from a remote node.
func (d *Downloader) deliver(id string, destCh chan dataPack, packet dataPack, inMeter, dropMeter metrics.Meter) (err error) {
	// Update the delivery metrics for both good and failed deliveries
	inMeter.Mark(int64(packet.Items()))
	defer func() {
		if err != nil {
			dropMeter.Mark(int64(packet.Items()))
		}
	}()
	// Make sure the downloader is active
	if atomic.LoadInt32(&d.synchronising) == 0 {
		return errNoSyncActive
	}
	// Deliver or abort if the sync is canceled while queuing
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case destCh <- packet:
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}
