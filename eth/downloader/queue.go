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
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var (
	blockCacheLimit = 1024 // Maximum number of blocks to cache before throttling the download
)

var (
	errNoFetchesPending = errors.New("no fetches pending")
	errStateSyncPending = errors.New("state trie sync already scheduled")
	errStaleDelivery    = errors.New("stale delivery")
)

// fetchRequest is a currently running data retrieval operation.
type fetchRequest struct {
	Peer    *peer               // Peer to which the request was sent
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

	hashPool    map[common.Hash]int // [eth/61] Pending hashes, mapping to their insertion index (priority)
	hashQueue   *prque.Prque        // [eth/61] Priority queue of the block hashes to fetch
	hashCounter int                 // [eth/61] Counter indexing the added hashes to ensure retrieval order

	headerHead common.Hash // [eth/62] Hash of the last queued header to verify order

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

	stateDatabase   ethdb.Database   // [eth/63] Trie database to populate during state reassembly
	stateScheduler  *state.StateSync // [eth/63] State trie synchronisation scheduler and integrator
	stateProcessors int32            // [eth/63] Number of currently running state processors
	stateSchedLock  sync.RWMutex     // [eth/63] Lock serialising access to the state scheduler

	resultCache  []*fetchResult // Downloaded but not yet delivered fetch results
	resultOffset uint64         // Offset of the first cached fetch result in the block chain

	lock sync.RWMutex
}

// newQueue creates a new download queue for scheduling block retrieval.
func newQueue(stateDb ethdb.Database) *queue {
	return &queue{
		hashPool:         make(map[common.Hash]int),
		hashQueue:        prque.New(),
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
	}
}

// Reset clears out the queue contents.
func (q *queue) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.stateSchedLock.Lock()
	defer q.stateSchedLock.Unlock()

	q.mode = FullSync
	q.fastSyncPivot = 0

	q.hashPool = make(map[common.Hash]int)
	q.hashQueue.Reset()
	q.hashCounter = 0

	q.headerHead = common.Hash{}

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

// PendingBlocks retrieves the number of block (body) requests pending for retrieval.
func (q *queue) PendingBlocks() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return q.hashQueue.Size() + q.blockTaskQueue.Size()
}

// PendingReceipts retrieves the number of block receipts pending for retrieval.
func (q *queue) PendingReceipts() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return q.receiptTaskQueue.Size()
}

// PendingNodeData retrieves the number of node data entries pending for retrieval.
func (q *queue) PendingNodeData() int {
	q.stateSchedLock.RLock()
	defer q.stateSchedLock.RUnlock()

	if q.stateScheduler != nil {
		return q.stateScheduler.Pending()
	}
	return 0
}

// InFlightBlocks retrieves whether there are block fetch requests currently in
// flight.
func (q *queue) InFlightBlocks() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.blockPendPool) > 0
}

// InFlightReceipts retrieves whether there are receipt fetch requests currently
// in flight.
func (q *queue) InFlightReceipts() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.receiptPendPool) > 0
}

// InFlightNodeData retrieves whether there are node data entry fetch requests
// currently in flight.
func (q *queue) InFlightNodeData() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.statePendPool)+int(atomic.LoadInt32(&q.stateProcessors)) > 0
}

// Idle returns if the queue is fully idle or has some data still inside. This
// method is used by the tester to detect termination events.
func (q *queue) Idle() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	queued := q.hashQueue.Size() + q.blockTaskQueue.Size() + q.receiptTaskQueue.Size() + q.stateTaskQueue.Size()
	pending := len(q.blockPendPool) + len(q.receiptPendPool) + len(q.statePendPool)
	cached := len(q.blockDonePool) + len(q.receiptDonePool)

	q.stateSchedLock.RLock()
	if q.stateScheduler != nil {
		queued += q.stateScheduler.Pending()
	}
	q.stateSchedLock.RUnlock()

	return (queued + pending + cached) == 0
}

// FastSyncPivot retrieves the currently used fast sync pivot point.
func (q *queue) FastSyncPivot() uint64 {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return q.fastSyncPivot
}

// ShouldThrottleBlocks checks if the download should be throttled (active block (body)
// fetches exceed block cache).
func (q *queue) ShouldThrottleBlocks() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

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
	q.lock.RLock()
	defer q.lock.RUnlock()

	// Calculate the currently in-flight receipt requests
	pending := 0
	for _, request := range q.receiptPendPool {
		pending += len(request.Headers)
	}
	// Throttle if more receipts are in-flight than free space in the cache
	return pending >= len(q.resultCache)-len(q.receiptDonePool)
}

// Schedule61 adds a set of hashes for the download queue for scheduling, returning
// the new hashes encountered.
func (q *queue) Schedule61(hashes []common.Hash, fifo bool) []common.Hash {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Insert all the hashes prioritised in the arrival order
	inserts := make([]common.Hash, 0, len(hashes))
	for _, hash := range hashes {
		// Skip anything we already have
		if old, ok := q.hashPool[hash]; ok {
			glog.V(logger.Warn).Infof("Hash %x already scheduled at index %v", hash, old)
			continue
		}
		// Update the counters and insert the hash
		q.hashCounter = q.hashCounter + 1
		inserts = append(inserts, hash)

		q.hashPool[hash] = q.hashCounter
		if fifo {
			q.hashQueue.Push(hash, -float32(q.hashCounter)) // Lowest gets schedules first
		} else {
			q.hashQueue.Push(hash, float32(q.hashCounter)) // Highest gets schedules first
		}
	}
	return inserts
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
			glog.V(logger.Warn).Infof("Header #%v [%x] broke chain ordering, expected %d", header.Number, hash[:4], from)
			break
		}
		if q.headerHead != (common.Hash{}) && q.headerHead != header.ParentHash {
			glog.V(logger.Warn).Infof("Header #%v [%x] broke chain ancestry", header.Number, hash[:4])
			break
		}
		// Make sure no duplicate requests are executed
		if _, ok := q.blockTaskPool[hash]; ok {
			glog.V(logger.Warn).Infof("Header #%d [%x] already scheduled for block fetch", header.Number.Uint64(), hash[:4])
			continue
		}
		if _, ok := q.receiptTaskPool[hash]; ok {
			glog.V(logger.Warn).Infof("Header #%d [%x] already scheduled for receipt fetch", header.Number.Uint64(), hash[:4])
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
			// Pivoting point of the fast sync, retrieve the state tries
			q.stateSchedLock.Lock()
			q.stateScheduler = state.NewStateSync(header.Root, q.stateDatabase)
			q.stateSchedLock.Unlock()
		}
		inserts = append(inserts, header)
		q.headerHead = hash
		from++
	}
	return inserts
}

// GetHeadResult retrieves the first fetch result from the cache, or nil if it hasn't
// been downloaded yet (or simply non existent).
func (q *queue) GetHeadResult() *fetchResult {
	q.lock.RLock()
	defer q.lock.RUnlock()

	// If there are no results pending, return nil
	if len(q.resultCache) == 0 || q.resultCache[0] == nil {
		return nil
	}
	// If the next result is still incomplete, return nil
	if q.resultCache[0].Pending > 0 {
		return nil
	}
	// If the next result is the fast sync pivot...
	if q.mode == FastSync && q.resultCache[0].Header.Number.Uint64() == q.fastSyncPivot {
		// If the pivot state trie is still being pulled, return nil
		if len(q.stateTaskPool) > 0 {
			return nil
		}
		if q.PendingNodeData() > 0 {
			return nil
		}
		// If the state is done, but not enough post-pivot headers were verified, stall...
		for i := 0; i < fsHeaderForceVerify; i++ {
			if i+1 >= len(q.resultCache) || q.resultCache[i+1] == nil {
				return nil
			}
		}
	}
	return q.resultCache[0]
}

// TakeResults retrieves and permanently removes a batch of fetch results from
// the cache.
func (q *queue) TakeResults() []*fetchResult {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Accumulate all available results
	results := []*fetchResult{}
	for i, result := range q.resultCache {
		// Stop if no more results are ready
		if result == nil || result.Pending > 0 {
			break
		}
		// The fast sync pivot block may only be processed after state fetch completes
		if q.mode == FastSync && result.Header.Number.Uint64() == q.fastSyncPivot {
			if len(q.stateTaskPool) > 0 {
				break
			}
			if q.PendingNodeData() > 0 {
				break
			}
			// Even is state fetch is done, ensure post-pivot headers passed verifications
			safe := true
			for j := 0; j < fsHeaderForceVerify; j++ {
				if i+j+1 >= len(q.resultCache) || q.resultCache[i+j+1] == nil {
					safe = false
				}
			}
			if !safe {
				break
			}
		}
		// If we've just inserted the fast sync pivot, stop as the following batch needs different insertion
		if q.mode == FastSync && result.Header.Number.Uint64() == q.fastSyncPivot+1 && len(results) > 0 {
			break
		}
		results = append(results, result)

		hash := result.Header.Hash()
		delete(q.blockDonePool, hash)
		delete(q.receiptDonePool, hash)
	}
	// Delete the results from the slice and let them be garbage collected
	// without this slice trick the results would stay in memory until nil
	// would be assigned to them.
	copy(q.resultCache, q.resultCache[len(results):])
	for k, n := len(q.resultCache)-len(results), len(q.resultCache); k < n; k++ {
		q.resultCache[k] = nil
	}
	q.resultOffset += uint64(len(results))

	return results
}

// ReserveBlocks reserves a set of block hashes for the given peer, skipping any
// previously failed download.
func (q *queue) ReserveBlocks(p *peer, count int) *fetchRequest {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.reserveHashes(p, count, q.hashQueue, nil, q.blockPendPool, len(q.resultCache)-len(q.blockDonePool))
}

// ReserveNodeData reserves a set of node data hashes for the given peer, skipping
// any previously failed download.
func (q *queue) ReserveNodeData(p *peer, count int) *fetchRequest {
	// Create a task generator to fetch status-fetch tasks if all schedules ones are done
	generator := func(max int) {
		q.stateSchedLock.Lock()
		defer q.stateSchedLock.Unlock()

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

	return q.reserveHashes(p, count, q.stateTaskQueue, generator, q.statePendPool, count)
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
		if p.ignored.Has(hash) {
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
		if p.ignored.Has(header.Hash()) {
			skip = append(skip, header)
		} else {
			send = append(send, header)
		}
	}
	// Merge all the skipped headers back
	for _, header := range skip {
		taskQueue.Push(header, -float32(header.Number.Uint64()))
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

// CancelBlocks aborts a fetch request, returning all pending hashes to the queue.
func (q *queue) CancelBlocks(request *fetchRequest) {
	q.cancel(request, q.hashQueue, q.blockPendPool)
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
		for hash, index := range request.Hashes {
			q.hashQueue.Push(hash, float32(index))
		}
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

// ExpireBlocks checks for in flight requests that exceeded a timeout allowance,
// canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireBlocks(timeout time.Duration) []string {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.blockPendPool, q.hashQueue, blockTimeoutMeter)
}

// ExpireBodies checks for in flight block body requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireBodies(timeout time.Duration) []string {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.blockPendPool, q.blockTaskQueue, bodyTimeoutMeter)
}

// ExpireReceipts checks for in flight receipt requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireReceipts(timeout time.Duration) []string {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.receiptPendPool, q.receiptTaskQueue, receiptTimeoutMeter)
}

// ExpireNodeData checks for in flight node data requests that exceeded a timeout
// allowance, canceling them and returning the responsible peers for penalisation.
func (q *queue) ExpireNodeData(timeout time.Duration) []string {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.expire(timeout, q.statePendPool, q.stateTaskQueue, stateTimeoutMeter)
}

// expire is the generic check that move expired tasks from a pending pool back
// into a task pool, returning all entities caught with expired tasks.
//
// Note, this method expects the queue lock to be already held for writing. The
// reason the lock is not obtained in here is because the parameters already need
// to access the queue, so they already need a lock anyway.
func (q *queue) expire(timeout time.Duration, pendPool map[string]*fetchRequest, taskQueue *prque.Prque, timeoutMeter metrics.Meter) []string {
	// Iterate over the expired requests and return each to the queue
	peers := []string{}
	for id, request := range pendPool {
		if time.Since(request.Time) > timeout {
			// Update the metrics with the timeout
			timeoutMeter.Mark(1)

			// Return any non satisfied requests to the pool
			for hash, index := range request.Hashes {
				taskQueue.Push(hash, float32(index))
			}
			for _, header := range request.Headers {
				taskQueue.Push(header, -float32(header.Number.Uint64()))
			}
			peers = append(peers, id)
		}
	}
	// Remove the expired requests from the pending pool
	for _, id := range peers {
		delete(pendPool, id)
	}
	return peers
}

// DeliverBlocks injects a block retrieval response into the download queue.
func (q *queue) DeliverBlocks(id string, blocks []*types.Block) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the blocks were never requested
	request := q.blockPendPool[id]
	if request == nil {
		return errNoFetchesPending
	}
	blockReqTimer.UpdateSince(request.Time)
	delete(q.blockPendPool, id)

	// If no blocks were retrieved, mark them as unavailable for the origin peer
	if len(blocks) == 0 {
		for hash, _ := range request.Hashes {
			request.Peer.ignored.Add(hash)
		}
	}
	// Iterate over the downloaded blocks and add each of them
	errs := make([]error, 0)
	for _, block := range blocks {
		// Skip any blocks that were not requested
		hash := block.Hash()
		if _, ok := request.Hashes[hash]; !ok {
			errs = append(errs, fmt.Errorf("non-requested block %x", hash))
			continue
		}
		// Reconstruct the next result if contents match up
		index := int(block.Number().Int64() - int64(q.resultOffset))
		if index >= len(q.resultCache) || index < 0 {
			errs = []error{errInvalidChain}
			break
		}
		q.resultCache[index] = &fetchResult{
			Header:       block.Header(),
			Transactions: block.Transactions(),
			Uncles:       block.Uncles(),
		}
		q.blockDonePool[block.Hash()] = struct{}{}

		delete(request.Hashes, hash)
		delete(q.hashPool, hash)
	}
	// Return all failed or missing fetches to the queue
	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
	}
	// If none of the blocks were good, it's a stale delivery
	switch {
	case len(errs) == 0:
		return nil

	case len(errs) == 1 && (errs[0] == errInvalidChain || errs[0] == errInvalidBlock):
		return errs[0]

	case len(errs) == len(blocks):
		return errStaleDelivery

	default:
		return fmt.Errorf("multiple failures: %v", errs)
	}
}

// DeliverBodies injects a block body retrieval response into the results queue.
func (q *queue) DeliverBodies(id string, txLists [][]*types.Transaction, uncleLists [][]*types.Header) error {
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
func (q *queue) DeliverReceipts(id string, receiptList [][]*types.Receipt) error {
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
func (q *queue) deliver(id string, taskPool map[common.Hash]*types.Header, taskQueue *prque.Prque, pendPool map[string]*fetchRequest,
	donePool map[common.Hash]struct{}, reqTimer metrics.Timer, results int, reconstruct func(header *types.Header, index int, result *fetchResult) error) error {
	// Short circuit if the data was never requested
	request := pendPool[id]
	if request == nil {
		return errNoFetchesPending
	}
	reqTimer.UpdateSince(request.Time)
	delete(pendPool, id)

	// If no data items were retrieved, mark them as unavailable for the origin peer
	if results == 0 {
		for hash, _ := range request.Headers {
			request.Peer.ignored.Add(hash)
		}
	}
	// Assemble each of the results with their headers and retrieved data parts
	var (
		failure error
		useful  bool
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
	// If none of the data was good, it's a stale delivery
	switch {
	case failure == nil || failure == errInvalidChain:
		return failure

	case useful:
		return fmt.Errorf("partial failure: %v", failure)

	default:
		return errStaleDelivery
	}
}

// DeliverNodeData injects a node state data retrieval response into the queue.
func (q *queue) DeliverNodeData(id string, data [][]byte, callback func(error, int)) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the data was never requested
	request := q.statePendPool[id]
	if request == nil {
		return errNoFetchesPending
	}
	stateReqTimer.UpdateSince(request.Time)
	delete(q.statePendPool, id)

	// If no data was retrieved, mark their hashes as unavailable for the origin peer
	if len(data) == 0 {
		for hash, _ := range request.Hashes {
			request.Peer.ignored.Add(hash)
		}
	}
	// Iterate over the downloaded data and verify each of them
	errs := make([]error, 0)
	process := []trie.SyncResult{}
	for _, blob := range data {
		// Skip any blocks that were not requested
		hash := common.BytesToHash(crypto.Sha3(blob))
		if _, ok := request.Hashes[hash]; !ok {
			errs = append(errs, fmt.Errorf("non-requested state data %x", hash))
			continue
		}
		// Inject the next state trie item into the processing queue
		process = append(process, trie.SyncResult{hash, blob})

		delete(request.Hashes, hash)
		delete(q.stateTaskPool, hash)
	}
	// Start the asynchronous node state data injection
	atomic.AddInt32(&q.stateProcessors, 1)
	go func() {
		defer atomic.AddInt32(&q.stateProcessors, -1)
		q.deliverNodeData(process, callback)
	}()
	// Return all failed or missing fetches to the queue
	for hash, index := range request.Hashes {
		q.stateTaskQueue.Push(hash, float32(index))
	}
	// If none of the data items were good, it's a stale delivery
	switch {
	case len(errs) == 0:
		return nil

	case len(errs) == len(request.Hashes):
		return errStaleDelivery

	default:
		return fmt.Errorf("multiple failures: %v", errs)
	}
}

// deliverNodeData is the asynchronous node data processor that injects a batch
// of sync results into the state scheduler.
func (q *queue) deliverNodeData(results []trie.SyncResult, callback func(error, int)) {
	// Process results one by one to permit task fetches in between
	for i, result := range results {
		q.stateSchedLock.Lock()

		if q.stateScheduler == nil {
			// Syncing aborted since this async delivery started, bail out
			q.stateSchedLock.Unlock()
			callback(errNoFetchesPending, i)
			return
		}
		if _, err := q.stateScheduler.Process([]trie.SyncResult{result}); err != nil {
			// Processing a state result failed, bail out
			q.stateSchedLock.Unlock()
			callback(err, i)
			return
		}
		// Item processing succeeded, release the lock (temporarily)
		q.stateSchedLock.Unlock()
	}
	callback(nil, len(results))
}

// Prepare configures the result cache to allow accepting and caching inbound
// fetch results.
func (q *queue) Prepare(offset uint64, mode SyncMode, pivot uint64) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.resultOffset < offset {
		q.resultOffset = offset
	}
	q.fastSyncPivot = pivot
	q.mode = mode
}
