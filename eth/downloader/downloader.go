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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

var (
	MaxBlockFetch   = 128 // Number of blocks to be fetched per retrieval request
	MaxHeaderFetch  = 192 // Number of block headers to be fetched per retrieval request
	MaxReceiptFetch = 256 // Number of transaction receipts to allow fetching per request

	maxQueuedHeaders           = 32 * 1024                        // [eth/62] Maximum number of headers to queue for import (DOS protection)
	maxHeadersProcess          = 2048                             // Number of header download results to import at once into the chain
	maxResultsProcess          = 2048                             // Number of content download results to import at once into the chain
	fullMaxForkAncestry uint64 = params.FullImmutabilityThreshold // Maximum chain reorganisation (locally redeclared so tests can reduce it)

	reorgProtHeaderDelay = 2 // Number of headers to delay delivering to cover mini reorgs

	fsHeaderSafetyNet = 2048            // Number of headers to discard in case a chain violation is detected
	fsHeaderContCheck = 3 * time.Second // Time interval to check for header continuations during state download
	fsMinFullBlocks   = 64              // Number of blocks to retrieve fully even in snap sync
)

var (
	errBusy    = errors.New("busy")
	errBadPeer = errors.New("action from bad peer ignored")

	errTimeout                 = errors.New("timeout")
	errInvalidChain            = errors.New("retrieved hash chain is invalid")
	errInvalidBody             = errors.New("retrieved block body is invalid")
	errInvalidReceipt          = errors.New("retrieved receipt is invalid")
	errCancelStateFetch        = errors.New("state data download canceled (requested)")
	errCancelContentProcessing = errors.New("content processing canceled (requested)")
	errCanceled                = errors.New("syncing canceled (requested)")
	errNoPivotHeader           = errors.New("pivot header is not found")
)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// badBlockFn is a callback for the async beacon sync to notify the caller that
// the origin header requested to sync to, produced a chain with a bad block.
type badBlockFn func(invalid *types.Header, origin *types.Header)

// headerTask is a set of downloaded headers to queue along with their precomputed
// hashes to avoid constant rehashing.
type headerTask struct {
	headers []*types.Header
	hashes  []common.Hash
}

type Downloader struct {
	mode atomic.Uint32  // Synchronisation mode defining the strategy used (per sync cycle), use d.getMode() to get the SyncMode
	mux  *event.TypeMux // Event multiplexer to announce sync operation events

	queue *queue   // Scheduler for selecting the hashes to download
	peers *peerSet // Set of active peers from which download can proceed

	stateDB ethdb.Database // Database to state sync into (and deduplicate via)

	// Statistics
	syncStatsChainOrigin uint64       // Origin block number where syncing started at
	syncStatsChainHeight uint64       // Highest block number known when syncing started
	syncStatsLock        sync.RWMutex // Lock protecting the sync stats fields

	blockchain BlockChain

	// Callbacks
	dropPeer peerDropFn // Drops a peer for misbehaving
	badBlock badBlockFn // Reports a block as rejected by the chain

	// Status
	synchronising atomic.Bool
	notified      atomic.Bool
	committed     atomic.Bool
	ancientLimit  uint64 // The maximum block number which can be regarded as ancient data.

	// Channels
	headerProcCh chan *headerTask // Channel to feed the header processor new tasks

	// Skeleton sync
	skeleton *skeleton // Header skeleton to backfill the chain with (eth2 mode)

	// State sync
	pivotHeader *types.Header // Pivot block header to dynamically push the syncing state root
	pivotLock   sync.RWMutex  // Lock protecting pivot header reads from updates

	SnapSyncer     *snap.Syncer // TODO(karalabe): make private! hack for now
	stateSyncStart chan *stateSync

	// Cancellation and termination
	cancelCh   chan struct{}  // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex   // Lock to protect the cancel channel and peer in delivers
	cancelWg   sync.WaitGroup // Make sure all fetcher goroutines have exited.

	quitCh   chan struct{} // Quit channel to signal termination
	quitLock sync.Mutex    // Lock to prevent double closes

	// Testing hooks
	bodyFetchHook    func([]*types.Header) // Method to call upon starting a block body fetch
	receiptFetchHook func([]*types.Header) // Method to call upon starting a receipt fetch
	chainInsertHook  func([]*fetchResult)  // Method to call upon inserting a chain of blocks (possibly in multiple invocations)

	// Progress reporting metrics
	syncStartBlock uint64    // Head snap block when Geth was started
	syncStartTime  time.Time // Time instance when chain sync started
	syncLogTime    time.Time // Time instance when status was last reported
}

// BlockChain encapsulates functions required to sync a (full or snap) blockchain.
type BlockChain interface {
	// HasHeader verifies a header's presence in the local chain.
	HasHeader(common.Hash, uint64) bool

	// GetHeaderByHash retrieves a header from the local chain.
	GetHeaderByHash(common.Hash) *types.Header

	// CurrentHeader retrieves the head header from the local chain.
	CurrentHeader() *types.Header

	// GetTd returns the total difficulty of a local block.
	GetTd(common.Hash, uint64) *big.Int

	// InsertHeaderChain inserts a batch of headers into the local chain.
	InsertHeaderChain([]*types.Header) (int, error)

	// SetHead rewinds the local chain to a new head.
	SetHead(uint64) error

	// HasBlock verifies a block's presence in the local chain.
	HasBlock(common.Hash, uint64) bool

	// HasFastBlock verifies a snap block's presence in the local chain.
	HasFastBlock(common.Hash, uint64) bool

	// GetBlockByHash retrieves a block from the local chain.
	GetBlockByHash(common.Hash) *types.Block

	// CurrentBlock retrieves the head block from the local chain.
	CurrentBlock() *types.Header

	// CurrentSnapBlock retrieves the head snap block from the local chain.
	CurrentSnapBlock() *types.Header

	// SnapSyncCommitHead directly commits the head block to a certain entity.
	SnapSyncCommitHead(common.Hash) error

	// InsertChain inserts a batch of blocks into the local chain.
	InsertChain(types.Blocks) (int, error)

	// InsertReceiptChain inserts a batch of receipts into the local chain.
	InsertReceiptChain(types.Blocks, []types.Receipts, uint64) (int, error)

	// Snapshots returns the blockchain snapshot tree to paused it during sync.
	Snapshots() *snapshot.Tree

	// TrieDB retrieves the low level trie database used for interacting
	// with trie nodes.
	TrieDB() *triedb.Database
}

// New creates a new downloader to fetch hashes and blocks from remote peers.
func New(stateDb ethdb.Database, mux *event.TypeMux, chain BlockChain, dropPeer peerDropFn, success func()) *Downloader {
	dl := &Downloader{
		stateDB:        stateDb,
		mux:            mux,
		queue:          newQueue(blockCacheMaxItems, blockCacheInitialItems),
		peers:          newPeerSet(),
		blockchain:     chain,
		dropPeer:       dropPeer,
		headerProcCh:   make(chan *headerTask, 1),
		quitCh:         make(chan struct{}),
		SnapSyncer:     snap.NewSyncer(stateDb, chain.TrieDB().Scheme()),
		stateSyncStart: make(chan *stateSync),
		syncStartBlock: chain.CurrentSnapBlock().Number.Uint64(),
	}
	// Create the post-merge skeleton syncer and start the process
	dl.skeleton = newSkeleton(stateDb, dl.peers, dropPeer, newBeaconBackfiller(dl, success))

	go dl.stateFetcher()
	return dl
}

// Progress retrieves the synchronisation boundaries, specifically the origin
// block where synchronisation started at (may have failed/suspended); the block
// or header sync is currently at; and the latest known block which the sync targets.
//
// In addition, during the state download phase of snap synchronisation the number
// of processed and the total number of known states are also returned. Otherwise
// these are zero.
func (d *Downloader) Progress() ethereum.SyncProgress {
	// Lock the current stats and return the progress
	d.syncStatsLock.RLock()
	defer d.syncStatsLock.RUnlock()

	current := uint64(0)
	mode := d.getMode()
	switch mode {
	case FullSync:
		current = d.blockchain.CurrentBlock().Number.Uint64()
	case SnapSync:
		current = d.blockchain.CurrentSnapBlock().Number.Uint64()
	default:
		log.Error("Unknown downloader mode", "mode", mode)
	}
	progress, pending := d.SnapSyncer.Progress()

	return ethereum.SyncProgress{
		StartingBlock:       d.syncStatsChainOrigin,
		CurrentBlock:        current,
		HighestBlock:        d.syncStatsChainHeight,
		SyncedAccounts:      progress.AccountSynced,
		SyncedAccountBytes:  uint64(progress.AccountBytes),
		SyncedBytecodes:     progress.BytecodeSynced,
		SyncedBytecodeBytes: uint64(progress.BytecodeBytes),
		SyncedStorage:       progress.StorageSynced,
		SyncedStorageBytes:  uint64(progress.StorageBytes),
		HealedTrienodes:     progress.TrienodeHealSynced,
		HealedTrienodeBytes: uint64(progress.TrienodeHealBytes),
		HealedBytecodes:     progress.BytecodeHealSynced,
		HealedBytecodeBytes: uint64(progress.BytecodeHealBytes),
		HealingTrienodes:    pending.TrienodeHeal,
		HealingBytecode:     pending.BytecodeHeal,
	}
}

// RegisterPeer injects a new download peer into the set of block source to be
// used for fetching hashes and blocks from.
func (d *Downloader) RegisterPeer(id string, version uint, peer Peer) error {
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	logger.Trace("Registering sync peer")
	if err := d.peers.Register(newPeerConnection(id, version, peer, logger)); err != nil {
		logger.Error("Failed to register sync peer", "err", err)
		return err
	}
	return nil
}

// UnregisterPeer remove a peer from the known list, preventing any action from
// the specified peer. An effort is also made to return any pending fetches into
// the queue.
func (d *Downloader) UnregisterPeer(id string) error {
	// Unregister the peer from the active peer set and revoke any fetch tasks
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	logger.Trace("Unregistering sync peer")
	if err := d.peers.Unregister(id); err != nil {
		logger.Error("Failed to unregister sync peer", "err", err)
		return err
	}
	d.queue.Revoke(id)

	return nil
}

// synchronise will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronize if its TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) synchronise(mode SyncMode, beaconPing chan struct{}) error {
	// The beacon header syncer is async. It will start this synchronization and
	// will continue doing other tasks. However, if synchronization needs to be
	// cancelled, the syncer needs to know if we reached the startup point (and
	// inited the cancel channel) or not yet. Make sure that we'll signal even in
	// case of a failure.
	if beaconPing != nil {
		defer func() {
			select {
			case <-beaconPing: // already notified
			default:
				close(beaconPing) // weird exit condition, notify that it's safe to cancel (the nothing)
			}
		}()
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !d.synchronising.CompareAndSwap(false, true) {
		return errBusy
	}
	defer d.synchronising.Store(false)

	// Post a user notification of the sync (only once per session)
	if d.notified.CompareAndSwap(false, true) {
		log.Info("Block synchronisation started")
	}
	if mode == SnapSync {
		// Snap sync will directly modify the persistent state, making the entire
		// trie database unusable until the state is fully synced. To prevent any
		// subsequent state reads, explicitly disable the trie database and state
		// syncer is responsible to address and correct any state missing.
		if d.blockchain.TrieDB().Scheme() == rawdb.PathScheme {
			if err := d.blockchain.TrieDB().Disable(); err != nil {
				return err
			}
		}
		// Snap sync uses the snapshot namespace to store potentially flaky data until
		// sync completely heals and finishes. Pause snapshot maintenance in the mean-
		// time to prevent access.
		if snapshots := d.blockchain.Snapshots(); snapshots != nil { // Only nil in tests
			snapshots.Disable()
		}
	}
	// Reset the queue, peer set and wake channels to clean any internal leftover state
	d.queue.Reset(blockCacheMaxItems, blockCacheInitialItems)
	d.peers.Reset()

	for _, ch := range []chan bool{d.queue.blockWakeCh, d.queue.receiptWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	for empty := false; !empty; {
		select {
		case <-d.headerProcCh:
		default:
			empty = true
		}
	}
	// Create cancel channel for aborting mid-flight and mark the master peer
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelLock.Unlock()

	defer d.Cancel() // No matter what, we can't leave the cancel channel open

	// Atomically set the requested sync mode
	d.mode.Store(uint32(mode))

	if beaconPing != nil {
		close(beaconPing)
	}
	return d.syncToHead()
}

func (d *Downloader) getMode() SyncMode {
	return SyncMode(d.mode.Load())
}

// syncToHead starts a block synchronization based on the hash chain from
// the specified head hash.
func (d *Downloader) syncToHead() (err error) {
	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if err != nil {
			d.mux.Post(FailedEvent{err})
		} else {
			latest := d.blockchain.CurrentHeader()
			d.mux.Post(DoneEvent{latest})
		}
	}()
	mode := d.getMode()

	log.Debug("Backfilling with the network", "mode", mode)
	defer func(start time.Time) {
		log.Debug("Synchronisation terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	// Look up the sync boundaries: the common ancestor and the target block
	var latest, pivot, final *types.Header
	latest, _, final, err = d.skeleton.Bounds()
	if err != nil {
		return err
	}
	if latest.Number.Uint64() > uint64(fsMinFullBlocks) {
		number := latest.Number.Uint64() - uint64(fsMinFullBlocks)

		// Retrieve the pivot header from the skeleton chain segment but
		// fallback to local chain if it's not found in skeleton space.
		if pivot = d.skeleton.Header(number); pivot == nil {
			_, oldest, _, _ := d.skeleton.Bounds() // error is already checked
			if number < oldest.Number.Uint64() {
				count := int(oldest.Number.Uint64() - number) // it's capped by fsMinFullBlocks
				headers := d.readHeaderRange(oldest, count)
				if len(headers) == count {
					pivot = headers[len(headers)-1]
					log.Warn("Retrieved pivot header from local", "number", pivot.Number, "hash", pivot.Hash(), "latest", latest.Number, "oldest", oldest.Number)
				}
			}
		}
		// Print an error log and return directly in case the pivot header
		// is still not found. It means the skeleton chain is not linked
		// correctly with local chain.
		if pivot == nil {
			log.Error("Pivot header is not found", "number", number)
			return errNoPivotHeader
		}
	}
	// If no pivot block was returned, the head is below the min full block
	// threshold (i.e. new chain). In that case we won't really snap sync
	// anyway, but still need a valid pivot block to avoid some code hitting
	// nil panics on access.
	if mode == SnapSync && pivot == nil {
		pivot = d.blockchain.CurrentBlock()
	}
	height := latest.Number.Uint64()

	// In beacon mode, use the skeleton chain for the ancestor lookup
	origin, err := d.findBeaconAncestor()
	if err != nil {
		return err
	}
	d.syncStatsLock.Lock()
	if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
		d.syncStatsChainOrigin = origin
	}
	d.syncStatsChainHeight = height
	d.syncStatsLock.Unlock()

	// Ensure our origin point is below any snap sync pivot point
	if mode == SnapSync {
		if height <= uint64(fsMinFullBlocks) {
			origin = 0
		} else {
			pivotNumber := pivot.Number.Uint64()
			if pivotNumber <= origin {
				origin = pivotNumber - 1
			}
			// Write out the pivot into the database so a rollback beyond it will
			// reenable snap sync
			rawdb.WriteLastPivotNumber(d.stateDB, pivotNumber)
		}
	}
	d.committed.Store(true)
	if mode == SnapSync && pivot.Number.Uint64() != 0 {
		d.committed.Store(false)
	}
	if mode == SnapSync {
		// Set the ancient data limitation. If we are running snap sync, all block
		// data older than ancientLimit will be written to the ancient store. More
		// recent data will be written to the active database and will wait for the
		// freezer to migrate.
		//
		// If the network is post-merge, use either the last announced finalized
		// block as the ancient limit, or if we haven't yet received one, the head-
		// a max fork ancestry limit. One quirky case if we've already passed the
		// finalized block, in which case the skeleton.Bounds will return nil and
		// we'll revert to head - 90K. That's fine, we're finishing sync anyway.
		//
		// For non-merged networks, if there is a checkpoint available, then calculate
		// the ancientLimit through that. Otherwise calculate the ancient limit through
		// the advertised height of the remote peer. This most is mostly a fallback for
		// legacy networks, but should eventually be dropped. TODO(karalabe).
		//
		// Beacon sync, use the latest finalized block as the ancient limit
		// or a reasonable height if no finalized block is yet announced.
		if final != nil {
			d.ancientLimit = final.Number.Uint64()
		} else if height > fullMaxForkAncestry+1 {
			d.ancientLimit = height - fullMaxForkAncestry - 1
		} else {
			d.ancientLimit = 0
		}
		frozen, _ := d.stateDB.Ancients() // Ignore the error here since light client can also hit here.

		// If a part of blockchain data has already been written into active store,
		// disable the ancient style insertion explicitly.
		if origin >= frozen && frozen != 0 {
			d.ancientLimit = 0
			log.Info("Disabling direct-ancient mode", "origin", origin, "ancient", frozen-1)
		} else if d.ancientLimit > 0 {
			log.Debug("Enabling direct-ancient mode", "ancient", d.ancientLimit)
		}
		// Rewind the ancient store and blockchain if reorg happens.
		if origin+1 < frozen {
			if err := d.blockchain.SetHead(origin); err != nil {
				return err
			}
			log.Info("Truncated excess ancient chain segment", "oldhead", frozen-1, "newhead", origin)
		}
	}
	// Initiate the sync using a concurrent header and content retrieval algorithm
	d.queue.Prepare(origin+1, mode)

	// In beacon mode, headers are served by the skeleton syncer
	fetchers := []func() error{
		func() error { return d.fetchHeaders(origin + 1) },  // Headers are always retrieved
		func() error { return d.fetchBodies(origin + 1) },   // Bodies are retrieved during normal and snap sync
		func() error { return d.fetchReceipts(origin + 1) }, // Receipts are retrieved during snap sync
		func() error { return d.processHeaders(origin + 1) },
	}
	if mode == SnapSync {
		d.pivotLock.Lock()
		d.pivotHeader = pivot
		d.pivotLock.Unlock()

		fetchers = append(fetchers, func() error { return d.processSnapSyncContent() })
	} else if mode == FullSync {
		fetchers = append(fetchers, func() error { return d.processFullSyncContent() })
	}
	return d.spawnSync(fetchers)
}

// spawnSync runs d.process and all given fetcher functions to completion in
// separate goroutines, returning the first error that appears.
func (d *Downloader) spawnSync(fetchers []func() error) error {
	errc := make(chan error, len(fetchers))
	d.cancelWg.Add(len(fetchers))
	for _, fn := range fetchers {
		fn := fn
		go func() { defer d.cancelWg.Done(); errc <- fn() }()
	}
	// Wait for the first error, then terminate the others.
	var err error
	for i := 0; i < len(fetchers); i++ {
		if i == len(fetchers)-1 {
			// Close the queue when all fetchers have exited.
			// This will cause the block processor to end when
			// it has processed the queue.
			d.queue.Close()
		}
		if got := <-errc; got != nil {
			err = got
			if got != errCanceled {
				break // receive a meaningful error, bubble it up
			}
		}
	}
	d.queue.Close()
	d.Cancel()
	return err
}

// cancel aborts all of the operations and resets the queue. However, cancel does
// not wait for the running download goroutines to finish. This method should be
// used when cancelling the downloads from inside the downloader.
func (d *Downloader) cancel() {
	// Close the current cancel channel
	d.cancelLock.Lock()
	defer d.cancelLock.Unlock()

	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
			// Channel was already closed
		default:
			close(d.cancelCh)
		}
	}
}

// Cancel aborts all of the operations and waits for all download goroutines to
// finish before returning.
func (d *Downloader) Cancel() {
	d.cancel()
	d.cancelWg.Wait()
}

// Terminate interrupts the downloader, canceling all pending operations.
// The downloader cannot be reused after calling Terminate.
func (d *Downloader) Terminate() {
	// Close the termination channel (make sure double close is allowed)
	d.quitLock.Lock()
	select {
	case <-d.quitCh:
	default:
		close(d.quitCh)

		// Terminate the internal beacon syncer
		d.skeleton.Terminate()
	}
	d.quitLock.Unlock()

	// Cancel any pending download requests
	d.Cancel()
}

// fetchBodies iteratively downloads the scheduled block bodies, taking any
// available peers, reserving a chunk of blocks for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchBodies(from uint64) error {
	log.Debug("Downloading block bodies", "origin", from)
	err := d.concurrentFetch((*bodyQueue)(d))

	log.Debug("Block body download terminated", "err", err)
	return err
}

// fetchReceipts iteratively downloads the scheduled block receipts, taking any
// available peers, reserving a chunk of receipts for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchReceipts(from uint64) error {
	log.Debug("Downloading receipts", "origin", from)
	err := d.concurrentFetch((*receiptQueue)(d))

	log.Debug("Receipt download terminated", "err", err)
	return err
}

// processHeaders takes batches of retrieved headers from an input channel and
// keeps processing and scheduling them into the header chain and downloader's
// queue until the stream ends or a failure occurs.
func (d *Downloader) processHeaders(origin uint64) error {
	var (
		mode  = d.getMode()
		timer = time.NewTimer(time.Second)
	)
	defer timer.Stop()

	for {
		select {
		case <-d.cancelCh:
			return errCanceled

		case task := <-d.headerProcCh:
			// Terminate header processing if we synced up
			if task == nil || len(task.headers) == 0 {
				// Notify everyone that headers are fully processed
				for _, ch := range []chan bool{d.queue.blockWakeCh, d.queue.receiptWakeCh} {
					select {
					case ch <- false:
					case <-d.cancelCh:
					}
				}
				return nil
			}
			// Otherwise split the chunk of headers into batches and process them
			headers, hashes := task.headers, task.hashes

			for len(headers) > 0 {
				// Terminate if something failed in between processing chunks
				select {
				case <-d.cancelCh:
					return errCanceled
				default:
				}
				// Select the next chunk of headers to import
				limit := maxHeadersProcess
				if limit > len(headers) {
					limit = len(headers)
				}
				chunkHeaders := headers[:limit]
				chunkHashes := hashes[:limit]

				// In case of header only syncing, validate the chunk immediately
				if mode == SnapSync {
					// Although the received headers might be all valid, a legacy
					// PoW/PoA sync must not accept post-merge headers. Make sure
					// that any transition is rejected at this point.
					if len(chunkHeaders) > 0 {
						if n, err := d.blockchain.InsertHeaderChain(chunkHeaders); err != nil {
							log.Warn("Invalid header encountered", "number", chunkHeaders[n].Number, "hash", chunkHashes[n], "parent", chunkHeaders[n].ParentHash, "err", err)
							return fmt.Errorf("%w: %v", errInvalidChain, err)
						}
					}
				}
				// If we've reached the allowed number of pending headers, stall a bit
				for d.queue.PendingBodies() >= maxQueuedHeaders || d.queue.PendingReceipts() >= maxQueuedHeaders {
					timer.Reset(time.Second)
					select {
					case <-d.cancelCh:
						return errCanceled
					case <-timer.C:
					}
				}
				// Otherwise insert the headers for content retrieval
				inserts := d.queue.Schedule(chunkHeaders, chunkHashes, origin)
				if len(inserts) != len(chunkHeaders) {
					return fmt.Errorf("%w: stale headers", errBadPeer)
				}

				headers = headers[limit:]
				hashes = hashes[limit:]
				origin += uint64(limit)
			}
			// Update the highest block number we know if a higher one is found.
			d.syncStatsLock.Lock()
			if d.syncStatsChainHeight < origin {
				d.syncStatsChainHeight = origin - 1
			}
			d.syncStatsLock.Unlock()

			// Signal the content downloaders of the availability of new tasks
			for _, ch := range []chan bool{d.queue.blockWakeCh, d.queue.receiptWakeCh} {
				select {
				case ch <- true:
				default:
				}
			}
		}
	}
}

// processFullSyncContent takes fetch results from the queue and imports them into the chain.
func (d *Downloader) processFullSyncContent() error {
	for {
		results := d.queue.Results(true)
		if len(results) == 0 {
			return nil
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		if err := d.importBlockResults(results); err != nil {
			return err
		}
	}
}

func (d *Downloader) importBlockResults(results []*fetchResult) error {
	// Check for any early termination requests
	if len(results) == 0 {
		return nil
	}
	select {
	case <-d.quitCh:
		return errCancelContentProcessing
	default:
	}
	// Retrieve a batch of results to import
	first, last := results[0].Header, results[len(results)-1].Header
	log.Debug("Inserting downloaded chain", "items", len(results),
		"firstnum", first.Number, "firsthash", first.Hash(),
		"lastnum", last.Number, "lasthash", last.Hash(),
	)
	blocks := make([]*types.Block, len(results))
	for i, result := range results {
		blocks[i] = types.NewBlockWithHeader(result.Header).WithBody(result.body())
	}
	// Downloaded blocks are always regarded as trusted after the
	// transition. Because the downloaded chain is guided by the
	// consensus-layer.
	if index, err := d.blockchain.InsertChain(blocks); err != nil {
		if index < len(results) {
			log.Debug("Downloaded item processing failed", "number", results[index].Header.Number, "hash", results[index].Header.Hash(), "err", err)

			// In post-merge, notify the engine API of encountered bad chains
			if d.badBlock != nil {
				head, _, _, err := d.skeleton.Bounds()
				if err != nil {
					log.Error("Failed to retrieve beacon bounds for bad block reporting", "err", err)
				} else {
					d.badBlock(blocks[index].Header(), head)
				}
			}
		} else {
			// The InsertChain method in blockchain.go will sometimes return an out-of-bounds index,
			// when it needs to preprocess blocks to import a sidechain.
			// The importer will put together a new list of blocks to import, which is a superset
			// of the blocks delivered from the downloader, and the indexing will be off.
			log.Debug("Downloaded item processing failed on sidechain import", "index", index, "err", err)
		}
		return fmt.Errorf("%w: %v", errInvalidChain, err)
	}
	return nil
}

// processSnapSyncContent takes fetch results from the queue and writes them to the
// database. It also controls the synchronisation of state nodes of the pivot block.
func (d *Downloader) processSnapSyncContent() error {
	// Start syncing state of the reported head block. This should get us most of
	// the state of the pivot block.
	d.pivotLock.RLock()
	sync := d.syncState(d.pivotHeader.Root)
	d.pivotLock.RUnlock()

	defer func() {
		// The `sync` object is replaced every time the pivot moves. We need to
		// defer close the very last active one, hence the lazy evaluation vs.
		// calling defer sync.Cancel() !!!
		sync.Cancel()
	}()

	closeOnErr := func(s *stateSync) {
		if err := s.Wait(); err != nil && err != errCancelStateFetch && err != errCanceled && err != snap.ErrCancelled {
			d.queue.Close() // wake up Results
		}
	}
	go closeOnErr(sync)

	// To cater for moving pivot points, track the pivot block and subsequently
	// accumulated download results separately.
	//
	// These will be nil up to the point where we reach the pivot, and will only
	// be set temporarily if the synced blocks are piling up, but the pivot is
	// still busy downloading. In that case, we need to occasionally check for
	// pivot moves, so need to unblock the loop. These fields will accumulate
	// the results in the meantime.
	//
	// Note, there's no issue with memory piling up since after 64 blocks the
	// pivot will forcefully move so these accumulators will be dropped.
	var (
		oldPivot *fetchResult   // Locked in pivot block, might change eventually
		oldTail  []*fetchResult // Downloaded content after the pivot
		timer    = time.NewTimer(time.Second)
	)
	defer timer.Stop()

	for {
		// Wait for the next batch of downloaded data to be available. If we have
		// not yet reached the pivot point, wait blockingly as there's no need to
		// spin-loop check for pivot moves. If we reached the pivot but have not
		// yet processed it, check for results async, so we might notice pivot
		// moves while state syncing. If the pivot was passed fully, block again
		// as there's no more reason to check for pivot moves at all.
		results := d.queue.Results(oldPivot == nil)
		if len(results) == 0 {
			// If pivot sync is done, stop
			if d.committed.Load() {
				d.reportSnapSyncProgress(true)
				return sync.Cancel()
			}
			// If sync failed, stop
			select {
			case <-d.cancelCh:
				sync.Cancel()
				return errCanceled
			default:
			}
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		d.reportSnapSyncProgress(false)

		// If we haven't downloaded the pivot block yet, check pivot staleness
		// notifications from the header downloader
		d.pivotLock.RLock()
		pivot := d.pivotHeader
		d.pivotLock.RUnlock()

		if oldPivot == nil { // no results piling up, we can move the pivot
			if !d.committed.Load() { // not yet passed the pivot, we can move the pivot
				if pivot.Root != sync.root { // pivot position changed, we can move the pivot
					sync.Cancel()
					sync = d.syncState(pivot.Root)

					go closeOnErr(sync)
				}
			}
		} else { // results already piled up, consume before handling pivot move
			results = append(append([]*fetchResult{oldPivot}, oldTail...), results...)
		}
		// Split around the pivot block and process the two sides via snap/full sync
		if !d.committed.Load() {
			latest := results[len(results)-1].Header
			// If the height is above the pivot block by 2 sets, it means the pivot
			// become stale in the network, and it was garbage collected, move to a
			// new pivot.
			//
			// Note, we have `reorgProtHeaderDelay` number of blocks withheld, Those
			// need to be taken into account, otherwise we're detecting the pivot move
			// late and will drop peers due to unavailable state!!!
			if height := latest.Number.Uint64(); height >= pivot.Number.Uint64()+2*uint64(fsMinFullBlocks)-uint64(reorgProtHeaderDelay) {
				log.Warn("Pivot became stale, moving", "old", pivot.Number.Uint64(), "new", height-uint64(fsMinFullBlocks)+uint64(reorgProtHeaderDelay))
				pivot = results[len(results)-1-fsMinFullBlocks+reorgProtHeaderDelay].Header // must exist as lower old pivot is uncommitted

				d.pivotLock.Lock()
				d.pivotHeader = pivot
				d.pivotLock.Unlock()

				// Write out the pivot into the database so a rollback beyond it will
				// reenable snap sync
				rawdb.WriteLastPivotNumber(d.stateDB, pivot.Number.Uint64())
			}
		}
		P, beforeP, afterP := splitAroundPivot(pivot.Number.Uint64(), results)
		if err := d.commitSnapSyncData(beforeP, sync); err != nil {
			return err
		}
		if P != nil {
			// If new pivot block found, cancel old state retrieval and restart
			if oldPivot != P {
				sync.Cancel()
				sync = d.syncState(P.Header.Root)

				go closeOnErr(sync)
				oldPivot = P
			}
			// Wait for completion, occasionally checking for pivot staleness
			timer.Reset(time.Second)
			select {
			case <-sync.done:
				if sync.err != nil {
					return sync.err
				}
				if err := d.commitPivotBlock(P); err != nil {
					return err
				}
				oldPivot = nil

			case <-timer.C:
				oldTail = afterP
				continue
			}
		}
		// Fast sync done, pivot commit done, full import
		if err := d.importBlockResults(afterP); err != nil {
			return err
		}
	}
}

func splitAroundPivot(pivot uint64, results []*fetchResult) (p *fetchResult, before, after []*fetchResult) {
	if len(results) == 0 {
		return nil, nil, nil
	}
	if lastNum := results[len(results)-1].Header.Number.Uint64(); lastNum < pivot {
		// the pivot is somewhere in the future
		return nil, results, nil
	}
	// This can also be optimized, but only happens very seldom
	for _, result := range results {
		num := result.Header.Number.Uint64()
		switch {
		case num < pivot:
			before = append(before, result)
		case num == pivot:
			p = result
		default:
			after = append(after, result)
		}
	}
	return p, before, after
}

func (d *Downloader) commitSnapSyncData(results []*fetchResult, stateSync *stateSync) error {
	// Check for any early termination requests
	if len(results) == 0 {
		return nil
	}
	select {
	case <-d.quitCh:
		return errCancelContentProcessing
	case <-stateSync.done:
		if err := stateSync.Wait(); err != nil {
			return err
		}
	default:
	}
	// Retrieve the batch of results to import
	first, last := results[0].Header, results[len(results)-1].Header
	log.Debug("Inserting snap-sync blocks", "items", len(results),
		"firstnum", first.Number, "firsthash", first.Hash(),
		"lastnumn", last.Number, "lasthash", last.Hash(),
	)
	blocks := make([]*types.Block, len(results))
	receipts := make([]types.Receipts, len(results))
	for i, result := range results {
		blocks[i] = types.NewBlockWithHeader(result.Header).WithBody(result.body())
		receipts[i] = result.Receipts
	}
	if index, err := d.blockchain.InsertReceiptChain(blocks, receipts, d.ancientLimit); err != nil {
		log.Debug("Downloaded item processing failed", "number", results[index].Header.Number, "hash", results[index].Header.Hash(), "err", err)
		return fmt.Errorf("%w: %v", errInvalidChain, err)
	}
	return nil
}

func (d *Downloader) commitPivotBlock(result *fetchResult) error {
	block := types.NewBlockWithHeader(result.Header).WithBody(result.body())
	log.Debug("Committing snap sync pivot as new head", "number", block.Number(), "hash", block.Hash())

	// Commit the pivot block as the new head, will require full sync from here on
	if _, err := d.blockchain.InsertReceiptChain([]*types.Block{block}, []types.Receipts{result.Receipts}, d.ancientLimit); err != nil {
		return err
	}
	if err := d.blockchain.SnapSyncCommitHead(block.Hash()); err != nil {
		return err
	}
	d.committed.Store(true)
	return nil
}

// DeliverSnapPacket is invoked from a peer's message handler when it transmits a
// data packet for the local node to consume.
func (d *Downloader) DeliverSnapPacket(peer *snap.Peer, packet snap.Packet) error {
	switch packet := packet.(type) {
	case *snap.AccountRangePacket:
		hashes, accounts, err := packet.Unpack()
		if err != nil {
			return err
		}
		return d.SnapSyncer.OnAccounts(peer, packet.ID, hashes, accounts, packet.Proof)

	case *snap.StorageRangesPacket:
		hashset, slotset := packet.Unpack()
		return d.SnapSyncer.OnStorage(peer, packet.ID, hashset, slotset, packet.Proof)

	case *snap.ByteCodesPacket:
		return d.SnapSyncer.OnByteCodes(peer, packet.ID, packet.Codes)

	case *snap.TrieNodesPacket:
		return d.SnapSyncer.OnTrieNodes(peer, packet.ID, packet.Nodes)

	default:
		return fmt.Errorf("unexpected snap packet type: %T", packet)
	}
}

// readHeaderRange returns a list of headers, using the given last header as the base,
// and going backwards towards genesis. This method assumes that the caller already has
// placed a reasonable cap on count.
func (d *Downloader) readHeaderRange(last *types.Header, count int) []*types.Header {
	var (
		current = last
		headers []*types.Header
	)
	for {
		parent := d.blockchain.GetHeaderByHash(current.ParentHash)
		if parent == nil {
			break // The chain is not continuous, or the chain is exhausted
		}
		headers = append(headers, parent)
		if len(headers) >= count {
			break
		}
		current = parent
	}
	return headers
}

// reportSnapSyncProgress calculates various status reports and provides it to the user.
func (d *Downloader) reportSnapSyncProgress(force bool) {
	// Initialize the sync start time if it's the first time we're reporting
	if d.syncStartTime.IsZero() {
		d.syncStartTime = time.Now().Add(-time.Millisecond) // -1ms offset to avoid division by zero
	}
	// Don't report all the events, just occasionally
	if !force && time.Since(d.syncLogTime) < 8*time.Second {
		return
	}
	// Don't report anything until we have a meaningful progress
	var (
		headerBytes, _  = d.stateDB.AncientSize(rawdb.ChainFreezerHeaderTable)
		bodyBytes, _    = d.stateDB.AncientSize(rawdb.ChainFreezerBodiesTable)
		receiptBytes, _ = d.stateDB.AncientSize(rawdb.ChainFreezerReceiptTable)
	)
	syncedBytes := common.StorageSize(headerBytes + bodyBytes + receiptBytes)
	if syncedBytes == 0 {
		return
	}
	var (
		header = d.blockchain.CurrentHeader()
		block  = d.blockchain.CurrentSnapBlock()
	)
	syncedBlocks := block.Number.Uint64() - d.syncStartBlock
	if syncedBlocks == 0 {
		return
	}
	// Retrieve the current chain head and calculate the ETA
	latest, _, _, err := d.skeleton.Bounds()
	if err != nil {
		// We're going to cheat for non-merged networks, but that's fine
		latest = d.pivotHeader
	}
	if latest == nil {
		// This should really never happen, but add some defensive code for now.
		// TODO(karalabe): Remove it eventually if we don't see it blow.
		log.Error("Nil latest block in sync progress report")
		return
	}
	var (
		left = latest.Number.Uint64() - block.Number.Uint64()
		eta  = time.Since(d.syncStartTime) / time.Duration(syncedBlocks) * time.Duration(left)

		progress = fmt.Sprintf("%.2f%%", float64(block.Number.Uint64())*100/float64(latest.Number.Uint64()))
		headers  = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(header.Number.Uint64()), common.StorageSize(headerBytes).TerminalString())
		bodies   = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(block.Number.Uint64()), common.StorageSize(bodyBytes).TerminalString())
		receipts = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(block.Number.Uint64()), common.StorageSize(receiptBytes).TerminalString())
	)
	log.Info("Syncing: chain download in progress", "synced", progress, "chain", syncedBytes, "headers", headers, "bodies", bodies, "receipts", receipts, "eta", common.PrettyDuration(eta))
	d.syncLogTime = time.Now()
}
