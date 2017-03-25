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

// Contains the block download scheduler to collect download tasks and schedule
// them in an ordered, and throttled way.

package downloader

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var (
	blockCacheLimit   = 8192 // Maximum number of blocks to cache before throttling the download
	maxInFlightStates = 8192 // Maximum number of state downloads to allow concurrently
)

var (
	errNoFetchesPending = errors.New("no fetches pending")
	errStaleDelivery    = errors.New("stale delivery")
)

// fetchRequest is a currently running data retrieval operation.
type fetchRequest struct {
	Peer    *peer               // Peer to which the request was sent
	From    uint64              // [eth/62] Requested chain element index (used for skeleton fills only)
	Hashes  map[common.Hash]int // [eth/61] Requested hashes with their insertion index (priority)
	Headers []*types.Header     // [eth/62] Requested headers, sorted by request order
	Time    time.Time           // Time when the request was made
}

// fetchResult is a struct collecting partial results from data fetchers until
// all outstanding pieces complete and the result as a whole can be processed.
type fetchResult struct {
	Pending int // Number of data fetches still pending

	Header       *types.Header
	Uncles       []*types.Header
	Transactions types.Transactions
	Receipts     types.Receipts
}

// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	mode          SyncMode // Synchronisation mode to decide on the block parts to schedule for fetching
	fastSyncPivot uint64   // Block number where the fast sync pivots into archive synchronisation mode

	headerHead common.Hash // [eth/62] Hash of the last queued header to verify order

	// Headers are "special", they download in batches, supported by a skeleton chain
	headerTaskPool  map[uint64]*types.Header       // [eth/62] Pending header retrieval tasks, mapping starting indexes to skeleton headers
	headerTaskQueue *prque.Prque                   // [eth/62] Priority queue of the skeleton indexes to fetch the filling headers for
	headerPeerMiss  map[string]map[uint64]struct{} // [eth/62] Set of per-peer header batches known to be unavailable
	headerPendPool  map[string]*fetchRequest       // [eth/62] Currently pending header retrieval operations
	headerResults   []*types.Header                // [eth/62] Result cache accumulating the completed headers
	headerProced    int                            // [eth/62] Number of headers already processed from the results
	headerOffset    uint64                         // [eth/62] Number of the first header in the result cache
	headerContCh    chan bool                      // [eth/62] Channel to notify when header download finishes

	// All data retrievals below are based on an already assembles header chain
	blockTaskPool  map[common.Hash]*types.Header // [eth/62] Pending block (body) retrieval tasks, mapping hashes to headers
	blockTaskQueue *prque.Prque                  // [eth/62] Priority queue of the headers to fetch the blocks (bodies) for
	blockPendPool  map[string]*fetchRequest      // [eth/62] Currently pending block (body) retrieval operations
	blockDonePool  map[common.Hash]struct{}      // [eth/62] Set of the completed block (body) fetches

	receiptTaskPool  map[common.Hash]*types.Header // [eth/63] Pending receipt retrieval tasks, mapping hashes to headers
	receiptTaskQueue *prque.Prque                  // [eth/63] Priority queue of the headers to fetch the receipts for
	receiptPendPool  map[string]*fetchRequest      // [eth/63] Currently pending receipt retrieval operations
	receiptDonePool  map[common.Hash]struct{}      // [eth/63] Set of the completed receipt fetches

	stateTaskIndex int                      // [eth/63] Counter indexing the added hashes to ensure prioritised retrieval order
	stateTaskPool  map[common.Hash]int      // [eth/63] Pending node data retrieval tasks, mapping to their priority
	stateTaskQueue *prque.Prque             // [eth/63] Priority queue of the hashes to fetch the node data for
	statePendPool  map[string]*fetchRequest // [eth/63] Currently pending node data retrieval operations

	stateDatabase  ethdb.Database   // [eth/63] Trie database to populate during state reassembly
	stateScheduler *state.StateSync // [eth/63] State trie synchronisation scheduler and integrator
	stateWriters   int              // [eth/63] Number of running state DB writer goroutines

	resultCache  []*fetchResult // Downloaded but not yet delivered fetch results
	resultOffset uint64         // Offset of the first cached fetch result in the block chain

	lock   *sync.Mutex
	active *sync.Cond
	closed bool
}

// newQueue creates a new download queue for scheduling block retrieval.
func newQueue(stateDb ethdb.Database) *queue {
	lock := new(sync.Mutex)
	return &queue{
		headerPendPool:   make(map[string]*fetchRequest),
		headerContCh:     make(chan bool),
		blockTaskPool:    make(map[common.Hash]*types.Header),
		blockTaskQueue:   prque.New(),
		blockPendPool:    make(map[string]*fetchRequest),
		blockDonePool:    make(map[common.Hash]struct{}),
		receiptTaskPool:  make(map[common.Hash]*types.Header),
		receiptTaskQueue: prque.New(),
		receiptPendPool:  make(map[string]*fetchRequest),
		receiptDonePool:  make(map[common.Hash]struct{}),
		stateTaskPool:    make(map[common.Hash]int),
		stateTaskQueue:   prque.New(),
		statePendPool:    make(map[string]*fetchRequest),
		stateDatabase:    stateDb,
		resultCache:      make([]*fetchResult, blockCacheLimit),
		active:           sync.NewCond(lock),
		lock:             lock,
	}
}

// Reset clears out the queue contents.
func (q *queue) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.closed = false
	q.mode = FullSync
	q.fastSyncPivot = 0

	q.headerHead = common.Hash{}

	q.headerPendPool = make(map[string]*fetchRequest)

	q.blockTaskPool = make(map[common.Hash]*types.Header)
	q.blockTaskQueue.Reset()
	q.blockPendPool = make(map[string]*fetchRequest)
	q.blockDonePool = make(map[common.Hash]struct{})

	q.receiptTaskPool = make(map[common.Hash]*types.Header)
	q.receiptTaskQueue.Reset()
	q.receiptPendPool = make(map[string]*fetchRequest)
	q.receiptDonePool = make(map[common.Hash]struct{})

	q.stateTaskIndex = 0
	q.stateTaskPool = make(map[common.Hash]int)
	q.stateTaskQueue.Reset()
	q.statePendPool = make(map[string]*fetchRequest)
	q.stateScheduler = nil

	q.resultCache = make([]*fetchResult, blockCacheLimit)
	q.resultOffset = 0
}

// Close marks the end of the sync, unblocking WaitResults.
// It may be called even if the queue is already closed.
func (q *queue) Close() {
	q.lock.Lock()
	q.closed = true
	q.lock.Unlock()
	q.active.Broadcast()
}

// PendingHeaders retrieves the number of header requests pending for retrieval.
func (q *queue) PendingHeaders() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.headerTaskQueue.Size()
}

// PendingBlocks retrieves the number of block (body) requests pending for retrieval.
func (q *queue) PendingBlocks() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.blockTaskQueue.Size()
}

// PendingReceipts retrieves the number of block receipts pending for retrieval.
func (q *queue) PendingReceipts() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.receiptTaskQueue.Size()
}

// PendingNodeData retrieves the number of node data entries pending for retrieval.
func (q *queue) PendingNodeData() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.pendingNodeDataLocked()
}

// pendingNodeDataLocked retrieves the number of node data entries pending for retrieval.
// The caller must hold q.lock.
func (q *queue) pendingNodeDataLocked() int {
	var n int
	if q.stateScheduler != nil {
		n = q.stateScheduler.Pending()
	}
	// Ensure that PendingNodeData doesn't return 0 until all state is written.
	if q.stateWriters > 0 {
		n++
	}
	return n
}

// InFlightHeaders retrieves whether there are header fetch requests currently
// in flight.
func (q *queue) InFlightHeaders() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.headerPendPool) > 0
}

// InFlightBlocks retrieves whether there are block fetch requests currently in
// flight.
func (q *queue) InFlightBlocks() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.blockPendPool) > 0
}

// InFlightReceipts retrieves whether there are receipt fetch requests currently
// in flight.
func (q *queue) InFlightReceipts() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.receiptPendPool) > 0
}

// InFlightNodeData retrieves whether there are node data entry fetch requests
// currently in flight.
func (q *queue) InFlightNodeData() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.statePendPool)+q.stateWriters > 0
}

// Idle returns if the queue is fully idle or has some data still inside. This
// method is used by the tester to detect termination events.
func (q *queue) Idle() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	queued := q.blockTaskQueue.Size() + q.receiptTaskQueue.Size() + q.stateTaskQueue.Size()
	pending := len(q.blockPendPool) + len(q.receiptPendPool) + len(q.statePendPool)
	cached := len(q.blockDonePool) + len(q.receiptDonePool)

	if q.stateScheduler != nil {
		queued += q.stateScheduler.Pending()
	}
	return (queued + pending + cached) == 0
}

// FastSyncPivot retrieves the currently used fast sync pivot point.
func (q *queue) FastSyncPivot() uint64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.fastSyncPivot
}

// ShouldThrottleBlocks checks if the download should be throttled (active block (body)
// fetches exceed block cache).
func (q *queue) ShouldThrottleBlocks() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Calculate the currently in-flight block (body) requests
	pending := 0
	for _, request := range q.blockPendPool {
		pending += len(request.Hashes) + len(request.Headers)
	}
	// Throttle if more blocks (bodies) are in-flight than free space in the cache
	return pending >= len(q.resultCache)-len(q.blockDonePool)
}

// ShouldThrottleReceipts checks if the download should be throttled (active receipt
// fetches exceed block cache).
func (q *queue) ShouldThrottleReceipts() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Calculate the currently in-flight receipt requests
	pending := 0
	for _, request := range q.receiptPendPool {
		pending += len(request.Headers)
	}
	// Throttle if more receipts are in-flight than free space in the cache
	return pending >= len(q.resultCache)-len(q.receiptDonePool)
}

// ScheduleSkeleton adds a batch of header retrieval tasks to the queue to fill
// up an already retrieved header skeleton.
func (q *queue) ScheduleSkeleton(from uint64, skeleton []*types.Header) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// No skeleton retrieval can be in progress, fail hard if so (huge implementation bug)
	if q.headerResults != nil {
		panic("skeleton assembly already in progress")
	}
	// Shedule all the header retrieval tasks for the skeleton assembly
	q.headerTaskPool = make(map[uint64]*types.Header)
	q.headerTaskQueue = prque.New()
	q.headerPeerMiss = make(map[string]map[uint64]struct{}) // Reset availability to correct invalid chains
	q.headerResults = make([]*types.Header, len(skeleton)*MaxHeaderFetch)
	q.headerProced = 0
	q.headerOffset = from
	q.headerContCh = make(chan bool, 1)

	for i, header := range skeleton {
		index := from + uint64(i*MaxHeaderFetch)

		q.headerTaskPool[index] = header
		q.headerTaskQueue.Push(index, -float32(index))
	}
}

// RetrieveHeaders retrieves the header chain assemble based on the scheduled
// skeleton.
func (q *queue) RetrieveHeaders() ([]*types.Header, int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	headers, proced := q.headerResults, q.headerProced
	q.headerResults, q.headerProced = nil, 0

	return headers, proced
}

// Schedule adds a set of headers for the download queue for scheduling, returning
// the new headers encountered.
func (q *queue) Schedule(headers []*types.Header, from uint64) []*types.Header {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Insert all the headers prioritised by the contained block number
	inserts := make([]*types.Header, 0, len(headers))
	for _, header := range headers {
		// Make sure chain order is honoured and preserved throughout
		hash := header.Hash()
		if header.Number == nil || header.Number.Uint64() != from {
			log.Warn("Header broke chain ordering", "number", header.Number, "hash", hash, "expected", from)
			break
		}
		if q.headerHead != (common.Hash{}) && q.headerHead != header.ParentHash {
			log.Warn("Header broke chain ancestry", "number", header.Number, "hash", hash)
			break
		}
		// Make sure no duplicate requests are executed
		if _, ok := q.blockTaskPool[hash]; ok {
			log.Warn("Header  already scheduled for block fetch", "number", header.Number, "hash", hash)
			continue
		}
		if _, ok := q.receiptTaskPool[hash]; ok {
			log.Warn("Header already scheduled for receipt fetch", "number", header.Number, "hash", hash)
			continue
		}
		// Queue the header for content retrieval
		q.blockTaskPool[hash] = header
		q.blockTaskQueue.Push(header, -float32(header.Number.Uint64()))

		if q.mode == FastSync && header.Number.Uint64() <= q.fastSyncPivot {
			// Fast phase of the fast sync, retrieve receipts too
			q.receiptTaskPool[hash] = header
			q.receiptTaskQueue.Push(header, -float32(header.Number.Uint64()))
		}
		if q.mode == FastSync && header.Number.Uint64() == q.fastSyncPivot {
			// Pivoting point of the fast sync, switch the state retrieval to this
			log.Debug("Switching state downloads to new block", "number", header.Number, "hash", hash)

			q.stateTaskIndex = 0
			q.stateTaskPool = make(map[common.Hash]int)
			q.stateTaskQueue.Reset()
			for _, req := range q.statePendPool {
				req.Hashes = make(map[common.Hash]int) // Make sure executing requests fail, but don't disappear
			}

			q.stateScheduler = state.NewStateSync(header.Root, q.stateDatabase)
		}
		inserts = append(inserts, header)
		q.headerHead = hash
		from++
	}
	return inserts
}

// WaitResults retrieves and permanently removes a batch of fetch
// results from the cache. the result slice will be empty if the queue
// has been closed.
func (q *queue) WaitResults() []*fetchResult {
	q.lock.Lock()
	defer q.lock.Unlock()

	nproc := q.countProcessableItems()
	for nproc == 0 && !q.closed {
		q.active.Wait()
		nproc = q.countProcessableItems()
	}
	results := make([]*fetchResult, nproc)
	copy(results, q.resultCache[:nproc])
	if len(results) > 0 {
		// Mark results as done before dropping them from the cache.
		for _, result := range results {
			hash := result.Header.Hash()
			delete(q.blockDonePool, hash)
			delete(q.receiptDonePool, hash)
		}
		// Delete the results from the cache and clear the tail.
		copy(q.resultCache, q.resultCache[nproc:])
		for i := len(q.resultCache) - nproc; i < len(q.resultCache); i++ {
			q.resultCache[i] = nil
		}
		// Advance the expected block number of the first cache entry.
		q.resultOffset += uint64(nproc)
	}
	return results
}

// countProcessableItems counts the processable items.
func (q *queue) countProcessableItems() int {
	for i, result := range q.resultCache {
		// Don't process incomplete or unavailable items.
		if result == nil || result.Pending > 0 {
			return i
		}
		// Special handling for the fast-sync pivot block:
		if q.mode == FastSync {
			bnum := result.Header.Number.Uint64()
			if bnum == q.fastSyncPivot {
				// If the state of the pivot block is not
				// available yet, we cannot proceed and return 0.
				//
				// Stop before processing the pivot block to ensure that
				// resultCache has space for fsHeaderForceVerify items. Not
				// doing this could leave us unable to download the required
				// amount of headers.
				if i > 0 || len(q.stateTaskPool) > 0 || q.pendingNodeDataLocked() > 0 {
					return i
				}
				for j := 0; j < fsHeaderForceVerify; j++ {
					if i+j+1 >= len(q.resultCache) || q.resultCache[i+j+1] == nil {
						return i
					}
				}
			}
			// If we're just the fast sync pivot, stop as well
			// because the following batch needs different insertion.
			// This simplifies handling the switchover in d.process.
			if bnum == q.fastSyncPivot+1 && i > 0 {
				return i
			}
		}
	}
	return len(q.resultCache)
}

// ReserveHeaders reserves a set of headers for the given peer, skipping any
// previously failed batches.
func (q *queue) ReserveHeaders(p *peer, count int) *fetchRequest {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the peer's already downloading something (sanity check to
	// not corrupt state)
	if _, ok := q.headerPendPool[p.id]; ok {
		return nil
	}
	// Retrieve a batch of hashes, skipping previously failed ones
	send, skip := uint64(0), []uint64{}
	for send == 0 && !q.headerTaskQueue.Empty() {
		from, _ := q.headerTaskQueue.Pop()
		if q.headerPeerMiss[p.id] != nil {
			if _, ok := q.headerPeerMiss[p.id][from.(uint64)]; ok {
				skip = append(skip, from.(uint64))
				continue
			}
		}
		send = from.(uint64)
	}
	// Merge all the skipped batches back
	for _, from := range skip {
		q.headerTaskQueue.Push(from, -float32(from))
	}
	// Assemble and return the block download request
	if send == 0 {
		return nil
	}
	request := &fetchRequest{
		Peer: p,
		From: send,
		Time: time.Now(),
	}
	q.headerPendPool[p.id] = request
	return request
}

// ReserveNodeData reserves a set of node data hashes for the given peer, skipping
// any previously failed download.
func (q *queue) ReserveNodeData(p *peer, count int) *fetchRequest {
	// Create a task generator to fetch status-fetch tasks if all schedules ones are done
	generator := func(max int) {
		if q.stateScheduler != nil {
			for _, hash := range q.stateScheduler.Missing(max) {
				q.stateTaskPool[hash] = q.stateTaskIndex
				q.stateTaskQueue.Push(hash, -float32(q.stateTaskIndex))
				q.stateTaskIndex++
			}
		}
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.reserveHashes(p, count, q.stateTaskQueue, generator, q.statePendPool, maxInFlightStates)
}

// reserveHashes reserves a set of hashes for the given peer, skipping previously
// failed ones.
//
// Note, this method expects the queue lock to be already held for writing. The
// reason the lock is not obtained in here is because the parameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) reserveHashes(p *peer, count int, taskQueue *prque.Prque, taskGen func(int), pendPool map[string]*fetchRequest, maxPending int) *fetchRequest {
	// Short circuit if the peer's already downloading something (sanity check to
	// not corrupt state)
	if _, ok := pendPool[p.id]; ok {
		return nil
	}
	// Calculate an upper limit on the hashes we might fetch (i.e. throttling)
	allowance := maxPending
	if allowance > 0 {
		for _, request := range pendPool {
			allowance -= len(request.Hashes)
		}
	}
	// If there's a task generator, ask it to fill our task queue
	if taskGen != nil && taskQueue.Size() < allowance {
		taskGen(allowance - taskQueue.Size())
	}
	if taskQueue.Empty() {
		return nil
	}
	// Retrieve a batch of hashes, skipping previously failed ones
	send := make(map[common.Hash]int)
	skip := make(map[common.Hash]int)

	for proc := 0; (allowance == 0 || proc < allowance) && len(send) < count && !taskQueue.Empty(); proc++ {
		hash, priority := taskQueue.Pop()
		if p.Lacks(hash.(common.Hash)) {
			skip[hash.(common.Hash)] = int(priority)
		} else {
			send[hash.(common.Hash)] = int(priority)
		}
	}
	// Merge all the skipped hashes back
	for hash, index := range skip {
		taskQueue.Push(hash, float32(index))
	}
	// Assemble and return the block download request
	if len(send) == 0 {
		return nil
	}
	request := &fetchRequest{
		Peer:   p,
		Hashes: send,
		Time:   time.Now(),
	}
	pendPool[p.id] = request

	return request
}

// ReserveBodies reserves a set of body fetches for the given peer, skipping any
// previously failed downloads. Beside the next batch of needed fetches, it also
// returns a flag whether empty blocks were queued requiring processing.
func (q *queue) ReserveBodies(p *peer, count int) (*fetchRequest, bool, error) {
	isNoop := func(header *types.Header) bool {
		return header.TxHash == types.EmptyRootHash && header.UncleHash == types.EmptyUncleHash
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.reserveHeaders(p, count, q.blockTaskPool, q.blockTaskQueue, q.blockPendPool, q.blockDonePool, isNoop)
}

// ReserveReceipts reserves a set of receipt fetches for the given peer, skipping
// any previously failed downloads. Beside the next batch of needed fetches, it
// also returns a flag whether empty receipts were queued requiring importing.
func (q *queue) ReserveReceipts(p *peer, count int) (*fetchRequest, bool, error) {
	isNoop := func(header *types.Header) bool {
		return header.ReceiptHash == types.EmptyRootHash
	}
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.reserveHeaders(p, count, q.receiptTaskPool, q.receiptTaskQueue, q.receiptPendPool, q.receiptDonePool, isNoop)
}

// reserveHeaders reserves a set of data download operations for a given peer,
// skipping any previously failed ones. This method is a generic version used
// by the individual special reservation functions.
//
// Note, this method expects the queue lock to be already held for writing. The
// reason the lock is not obtained in here is because the parameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) reserveHeaders(p *peer, count int, taskPool map[common.Hash]*types.Header, taskQueue *prque.Prque,
	pendPool map[string]*fetchRequest, donePool map[common.Hash]struct{}, isNoop func(*types.Header) bool) (*fetchRequest, bool, error) {
	// Short circuit if the pool has been depleted, or if the peer's already
	// downloading something (sanity check not to corrupt state)
	if taskQueue.Empty() {
		return nil, false, nil
	}
	if _, ok := pendPool[p.id]; ok {
		return nil, false, nil
	}
	// Calculate an upper limit on the items we might fetch (i.e. throttling)
	space := len(q.resultCache) - len(donePool)
	for _, request := range pendPool {
		space -= len(request.Headers)
	}
	// Retrieve a batch of tasks, skipping previously failed ones
	send := make([]*types.Header, 0, count)
	skip := make([]*types.Header, 0)

	progress := false
	for proc := 0; proc < space && len(send) < count && !taskQueue.Empty(); proc++ {
		header := taskQueue.PopItem().(*types.Header)

		// If we're the first to request this task, initialise the result container
		index := int(header.Number.Int64() - int64(q.resultOffset))
		if index >= len(q.resultCache) || index < 0 {
			common.Report("index allocation went beyond available resultCache space")
			return nil, false, errInvalidChain
		}
		if q.resultCache[index] == nil {
			components := 1
			if q.mode == FastSync && header.Number.Uint64() <= q.fastSyncPivot {
				components = 2
			}
			q.resultCache[index] = &fetchResult{
				Pending: components,
				Header:  header,
			}
		}
		// If this fetch task is a noop, skip this fetch operation
		if isNoop(header) {
			donePool[header.Hash()] = struct{}{}
			delete(taskPool, header.Hash())

			space, proc = space-1, proc-1
			q.resultCache[index].Pending--
			progress = true
			continue
		}
		// Otherwise unless the peer is known not to have the data, add to the retrieve list
		if p.Lacks(header.Hash()) {
			skip = append(skip, header)
		} else {
			send = append(send, header)
		}
	}
	// Merge all the skipped headers back
	for _, header := range skip {
		taskQueue.Push(header, -float32(header.Number.Uint64()))
	}
	if progress {
		// Wake WaitResults, resultCache was modified
		q.active.Signal()
	}
	// Assemble and return the block download request
	if len(send) == 0 {
		return nil, progress, nil
	}
	request := &fetchRequest{
		Peer:    p,
		Headers: send,
		Time:    time.Now(),
	}
	pendPool[p.id] = request

	return request, progress, nil
}

// CancelHeaders aborts a fetch request, returning all pending skeleton indexes to the queue.
func (q *queue) CancelHeaders(request *fetchRequest) {
	q.cancel(request, q.headerTaskQueue, q.headerPendPool)
}

// CancelBodies aborts a body fetch request, returning all pending headers to the
// task queue.
func (q *queue) CancelBodies(request *fetchRequest) {
	q.cancel(request, q.blockTaskQueue, q.blockPendPool)
}

// CancelReceipts aborts a body fetch request, returning all pending headers to
// the task queue.
func (q *queue) CancelReceipts(request *fetchRequest) {
	q.cancel(request, q.receiptTaskQueue, q.receiptPendPool)
}

// CancelNodeData aborts a node state data fetch request, returning all pending
// hashes to the task queue.
func (q *queue) CancelNodeData(request *fetchRequest) {
	q.cancel(request, q.stateTaskQueue, q.statePendPool)
}

// Cancel aborts a fetch request, returning all pending hashes to the task queue.
func (q *queue) cancel(request *fetchRequest, taskQueue *prque.Prque, pendPool map[string]*fetchRequest) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if request.From > 0 {
		taskQueue.Push(request.From, -float32(request.From))
	}
	for hash, index := range request.Hashes {
		taskQueue.Push(hash, float32(index))
	}
	for _, header := range request.Headers {
		taskQueue.Push(header, -float32(header.Number.Uint64()))
	}
	delete(pendPool, request.Peer.id)
}

// Revoke cancels all pending requests belonging to a given peer. This method is
// meant to be called during a peer drop to quickly reassign owned data fetches
// to remaining nodes.
func (q *queue) Revoke(peerId string) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if request, ok := q.blockPendPool[peerId]; ok {
		for _, header := range request.Headers {
			q.blockTaskQueue.Push(header, -float32(header.Number.Uint64()))
		}
		delete(q.blockPendPool, peerId)
	}
	if request, ok := q.receiptPendPool[peerId]; ok {
		for _, header := range request.Headers {
			q.receiptTaskQueue.Push(header, -float32(header.Number.Uint64()))
		}
		delete(q.receiptPendPool, peerId)
	}
	if request, ok := q.statePendPool[peerId]; ok {
		for hash, index := range request.Hashes {
			q.stateTaskQueue.Push(hash, float32(index))
		}
		delete(q.statePendPool, peerId)
	}
}

// ExpireHeaders checks for in flight requests that exceeded a timeout allowance,
// canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireHeaders(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.headerPendPool, q.headerTaskQueue, headerTimeoutMeter)
}

// ExpireBodies checks for in flight block body requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireBodies(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.blockPendPool, q.blockTaskQueue, bodyTimeoutMeter)
}

// ExpireReceipts checks for in flight receipt requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireReceipts(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.receiptPendPool, q.receiptTaskQueue, receiptTimeoutMeter)
}

// ExpireNodeData checks for in flight node data requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireNodeData(timeout time.Duration) map[string]int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.statePendPool, q.stateTaskQueue, stateTimeoutMeter)
}

// expire is the generic check that move expired tasks from a pending pool back
// into a task pool, returning all entities caught with expired tasks.
//
// Note, this method expects the queue lock to be already held. The
// reason the lock is not obtained in here is because the parameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) expire(timeout time.Duration, pendPool map[string]*fetchRequest, taskQueue *prque.Prque, timeoutMeter metrics.Meter) map[string]int {
	// Iterate over the expired requests and return each to the queue
	expiries := make(map[string]int)
	for id, request := range pendPool {
		if time.Since(request.Time) > timeout {
			// Update the metrics with the timeout
			timeoutMeter.Mark(1)

			// Return any non satisfied requests to the pool
			if request.From > 0 {
				taskQueue.Push(request.From, -float32(request.From))
			}
			for hash, index := range request.Hashes {
				taskQueue.Push(hash, float32(index))
			}
			for _, header := range request.Headers {
				taskQueue.Push(header, -float32(header.Number.Uint64()))
			}
			// Add the peer to the expiry report along the the number of failed requests
			expirations := len(request.Hashes)
			if expirations < len(request.Headers) {
				expirations = len(request.Headers)
			}
			expiries[id] = expirations
		}
	}
	// Remove the expired requests from the pending pool
	for id := range expiries {
		delete(pendPool, id)
	}
	return expiries
}

// DeliverHeaders injects a header retrieval response into the header results
// cache. This method either accepts all headers it received, or none of them
// if they do not map correctly to the skeleton.
//
// If the headers are accepted, the method makes an attempt to deliver the set
// of ready headers to the processor to keep the pipeline full. However it will
// not block to prevent stalling other pending deliveries.
func (q *queue) DeliverHeaders(id string, headers []*types.Header, headerProcCh chan []*types.Header) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the data was never requested
	request := q.headerPendPool[id]
	if request == nil {
		return 0, errNoFetchesPending
	}
	headerReqTimer.UpdateSince(request.Time)
	delete(q.headerPendPool, id)

	// Ensure headers can be mapped onto the skeleton chain
	target := q.headerTaskPool[request.From].Hash()

	accepted := len(headers) == MaxHeaderFetch
	if accepted {
		if headers[0].Number.Uint64() != request.From {
			log.Trace("First header broke chain ordering", "peer", id, "number", headers[0].Number, "hash", headers[0].Hash(), request.From)
			accepted = false
		} else if headers[len(headers)-1].Hash() != target {
			log.Trace("Last header broke skeleton structure ", "peer", id, "number", headers[len(headers)-1].Number, "hash", headers[len(headers)-1].Hash(), "expected", target)
			accepted = false
		}
	}
	if accepted {
		for i, header := range headers[1:] {
			hash := header.Hash()
			if want := request.From + 1 + uint64(i); header.Number.Uint64() != want {
				log.Warn("Header broke chain ordering", "peer", id, "number", header.Number, "hash", hash, "expected", want)
				accepted = false
				break
			}
			if headers[i].Hash() != header.ParentHash {
				log.Warn("Header broke chain ancestry", "peer", id, "number", header.Number, "hash", hash)
				accepted = false
				break
			}
		}
	}
	// If the batch of headers wasn't accepted, mark as unavailable
	if !accepted {
		log.Trace("Skeleton filling not accepted", "peer", id, "from", request.From)

		miss := q.headerPeerMiss[id]
		if miss == nil {
			q.headerPeerMiss[id] = make(map[uint64]struct{})
			miss = q.headerPeerMiss[id]
		}
		miss[request.From] = struct{}{}

		q.headerTaskQueue.Push(request.From, -float32(request.From))
		return 0, errors.New("delivery not accepted")
	}
	// Clean up a successful fetch and try to deliver any sub-results
	copy(q.headerResults[request.From-q.headerOffset:], headers)
	delete(q.headerTaskPool, request.From)

	ready := 0
	for q.headerProced+ready < len(q.headerResults) && q.headerResults[q.headerProced+ready] != nil {
		ready += MaxHeaderFetch
	}
	if ready > 0 {
		// Headers are ready for delivery, gather them and push forward (non blocking)
		process := make([]*types.Header, ready)
		copy(process, q.headerResults[q.headerProced:q.headerProced+ready])

		select {
		case headerProcCh <- process:
			log.Trace("Pre-scheduled new headers", "peer", id, "count", len(process), "from", process[0].Number)
			q.headerProced += len(process)
		default:
		}
	}
	// Check for termination and return
	if len(q.headerTaskPool) == 0 {
		q.headerContCh <- false
	}
	return len(headers), nil
}

// DeliverBodies injects a block body retrieval response into the results queue.
// The method returns the number of blocks bodies accepted from the delivery and
// also wakes any threads waiting for data delivery.
func (q *queue) DeliverBodies(id string, txLists [][]*types.Transaction, uncleLists [][]*types.Header) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	reconstruct := func(header *types.Header, index int, result *fetchResult) error {
		if types.DeriveSha(types.Transactions(txLists[index])) != header.TxHash || types.CalcUncleHash(uncleLists[index]) != header.UncleHash {
			return errInvalidBody
		}
		result.Transactions = txLists[index]
		result.Uncles = uncleLists[index]
		return nil
	}
	return q.deliver(id, q.blockTaskPool, q.blockTaskQueue, q.blockPendPool, q.blockDonePool, bodyReqTimer, len(txLists), reconstruct)
}

// DeliverReceipts injects a receipt retrieval response into the results queue.
// The method returns the number of transaction receipts accepted from the delivery
// and also wakes any threads waiting for data delivery.
func (q *queue) DeliverReceipts(id string, receiptList [][]*types.Receipt) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	reconstruct := func(header *types.Header, index int, result *fetchResult) error {
		if types.DeriveSha(types.Receipts(receiptList[index])) != header.ReceiptHash {
			return errInvalidReceipt
		}
		result.Receipts = receiptList[index]
		return nil
	}
	return q.deliver(id, q.receiptTaskPool, q.receiptTaskQueue, q.receiptPendPool, q.receiptDonePool, receiptReqTimer, len(receiptList), reconstruct)
}

// deliver injects a data retrieval response into the results queue.
//
// Note, this method expects the queue lock to be already held for writing. The
// reason the lock is not obtained in here is because the parameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) deliver(id string, taskPool map[common.Hash]*types.Header, taskQueue *prque.Prque,
	pendPool map[string]*fetchRequest, donePool map[common.Hash]struct{}, reqTimer metrics.Timer,
	results int, reconstruct func(header *types.Header, index int, result *fetchResult) error) (int, error) {

	// Short circuit if the data was never requested
	request := pendPool[id]
	if request == nil {
		return 0, errNoFetchesPending
	}
	reqTimer.UpdateSince(request.Time)
	delete(pendPool, id)

	// If no data items were retrieved, mark them as unavailable for the origin peer
	if results == 0 {
		for _, header := range request.Headers {
			request.Peer.MarkLacking(header.Hash())
		}
	}
	// Assemble each of the results with their headers and retrieved data parts
	var (
		accepted int
		failure  error
		useful   bool
	)
	for i, header := range request.Headers {
		// Short circuit assembly if no more fetch results are found
		if i >= results {
			break
		}
		// Reconstruct the next result if contents match up
		index := int(header.Number.Int64() - int64(q.resultOffset))
		if index >= len(q.resultCache) || index < 0 || q.resultCache[index] == nil {
			failure = errInvalidChain
			break
		}
		if err := reconstruct(header, i, q.resultCache[index]); err != nil {
			failure = err
			break
		}
		donePool[header.Hash()] = struct{}{}
		q.resultCache[index].Pending--
		useful = true
		accepted++

		// Clean up a successful fetch
		request.Headers[i] = nil
		delete(taskPool, header.Hash())
	}
	// Return all failed or missing fetches to the queue
	for _, header := range request.Headers {
		if header != nil {
			taskQueue.Push(header, -float32(header.Number.Uint64()))
		}
	}
	// Wake up WaitResults
	if accepted > 0 {
		q.active.Signal()
	}
	// If none of the data was good, it's a stale delivery
	switch {
	case failure == nil || failure == errInvalidChain:
		return accepted, failure
	case useful:
		return accepted, fmt.Errorf("partial failure: %v", failure)
	default:
		return accepted, errStaleDelivery
	}
}

// DeliverNodeData injects a node state data retrieval response into the queue.
// The method returns the number of node state accepted from the delivery.
func (q *queue) DeliverNodeData(id string, data [][]byte, callback func(int, bool, error)) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the data was never requested
	request := q.statePendPool[id]
	if request == nil {
		return 0, errNoFetchesPending
	}
	stateReqTimer.UpdateSince(request.Time)
	delete(q.statePendPool, id)

	// If no data was retrieved, mark their hashes as unavailable for the origin peer
	if len(data) == 0 {
		for hash := range request.Hashes {
			request.Peer.MarkLacking(hash)
		}
	}
	// Iterate over the downloaded data and verify each of them
	errs := make([]error, 0)
	process := []trie.SyncResult{}
	for _, blob := range data {
		// Skip any state trie entries that were not requested
		hash := common.BytesToHash(crypto.Keccak256(blob))
		if _, ok := request.Hashes[hash]; !ok {
			errs = append(errs, fmt.Errorf("non-requested state data %x", hash))
			continue
		}
		// Inject the next state trie item into the processing queue
		process = append(process, trie.SyncResult{Hash: hash, Data: blob})
		delete(request.Hashes, hash)
		delete(q.stateTaskPool, hash)
	}
	// Return all failed or missing fetches to the queue
	for hash, index := range request.Hashes {
		q.stateTaskQueue.Push(hash, float32(index))
	}
	if q.stateScheduler == nil {
		return 0, errNoFetchesPending
	}

	// Run valid nodes through the trie download scheduler. It writes completed nodes to a
	// batch, which is committed asynchronously. This may lead to over-fetches because the
	// scheduler treats everything as written after Process has returned, but it's
	// unlikely to be an issue in practice.
	batch := q.stateDatabase.NewBatch()
	progressed, nproc, procerr := q.stateScheduler.Process(process, batch)
	q.stateWriters += 1
	go func() {
		if procerr == nil {
			nproc = len(process)
			procerr = batch.Write()
		}
		// Return processing errors through the callback so the sync gets canceled. The
		// number of writers is decremented prior to the call so PendingNodeData will
		// return zero when the callback runs.
		q.lock.Lock()
		q.stateWriters -= 1
		q.lock.Unlock()
		callback(nproc, progressed, procerr)
		// Wake up WaitResults after the state has been written because it might be
		// waiting for completion of the pivot block's state download.
		q.active.Signal()
	}()

	// If none of the data items were good, it's a stale delivery
	switch {
	case len(errs) == 0:
		return len(process), nil
	case len(errs) == len(request.Hashes):
		return len(process), errStaleDelivery
	default:
		return len(process), fmt.Errorf("multiple failures: %v", errs)
	}
}

// Prepare configures the result cache to allow accepting and caching inbound
// fetch results.
func (q *queue) Prepare(offset uint64, mode SyncMode, pivot uint64, head *types.Header) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Prepare the queue for sync results
	if q.resultOffset < offset {
		q.resultOffset = offset
	}
	q.fastSyncPivot = pivot
	q.mode = mode

	// If long running fast sync, also start up a head stateretrieval immediately
	if mode == FastSync && pivot > 0 {
		q.stateScheduler = state.NewStateSync(head.Root, q.stateDatabase)
	}
}
