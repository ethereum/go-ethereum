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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var (
	blockCacheLimit = 8 * MaxBlockFetch // Maximum number of blocks to cache before throttling the download
)

var (
	errNoFetchesPending = errors.New("no fetches pending")
	errStaleDelivery    = errors.New("stale delivery")
)

// fetchRequest is a currently running block retrieval operation.
type fetchRequest struct {
	Peer    *peer               // Peer to which the request was sent
	Hashes  map[common.Hash]int // [eth/61] Requested hashes with their insertion index (priority)
	Headers []*types.Header     // [eth/62] Requested headers, sorted by request order
	Time    time.Time           // Time when the request was made
}

// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	hashPool    map[common.Hash]int // [eth/61] Pending hashes, mapping to their insertion index (priority)
	hashQueue   *prque.Prque        // [eth/61] Priority queue of the block hashes to fetch
	hashCounter int                 // [eth/61] Counter indexing the added hashes to ensure retrieval order

	headerPool  map[common.Hash]*types.Header // [eth/62] Pending headers, mapping from their hashes
	headerQueue *prque.Prque                  // [eth/62] Priority queue of the headers to fetch the bodies for

	pendPool map[string]*fetchRequest // Currently pending block retrieval operations

	blockPool   map[common.Hash]uint64 // Hash-set of the downloaded data blocks, mapping to cache indexes
	blockCache  []*Block               // Downloaded but not yet delivered blocks
	blockOffset uint64                 // Offset of the first cached block in the block-chain

	lock sync.RWMutex
}

// newQueue creates a new download queue for scheduling block retrieval.
func newQueue() *queue {
	return &queue{
		hashPool:    make(map[common.Hash]int),
		hashQueue:   prque.New(),
		headerPool:  make(map[common.Hash]*types.Header),
		headerQueue: prque.New(),
		pendPool:    make(map[string]*fetchRequest),
		blockPool:   make(map[common.Hash]uint64),
		blockCache:  make([]*Block, blockCacheLimit),
	}
}

// Reset clears out the queue contents.
func (q *queue) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.hashPool = make(map[common.Hash]int)
	q.hashQueue.Reset()
	q.hashCounter = 0

	q.headerPool = make(map[common.Hash]*types.Header)
	q.headerQueue.Reset()

	q.pendPool = make(map[string]*fetchRequest)

	q.blockPool = make(map[common.Hash]uint64)
	q.blockOffset = 0
	q.blockCache = make([]*Block, blockCacheLimit)
}

// Size retrieves the number of blocks in the queue, returning separately for
// pending and already downloaded.
func (q *queue) Size() (int, int) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.hashPool) + len(q.headerPool), len(q.blockPool)
}

// Pending retrieves the number of blocks pending for retrieval.
func (q *queue) Pending() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return q.hashQueue.Size() + q.headerQueue.Size()
}

// InFlight retrieves the number of fetch requests currently in flight.
func (q *queue) InFlight() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.pendPool)
}

// Throttle checks if the download should be throttled (active block fetches
// exceed block cache).
func (q *queue) Throttle() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	// Calculate the currently in-flight block requests
	pending := 0
	for _, request := range q.pendPool {
		pending += len(request.Hashes) + len(request.Headers)
	}
	// Throttle if more blocks are in-flight than free space in the cache
	return pending >= len(q.blockCache)-len(q.blockPool)
}

// Has checks if a hash is within the download queue or not.
func (q *queue) Has(hash common.Hash) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	if _, ok := q.hashPool[hash]; ok {
		return true
	}
	if _, ok := q.headerPool[hash]; ok {
		return true
	}
	if _, ok := q.blockPool[hash]; ok {
		return true
	}
	return false
}

// Insert61 adds a set of hashes for the download queue for scheduling, returning
// the new hashes encountered.
func (q *queue) Insert61(hashes []common.Hash, fifo bool) []common.Hash {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Insert all the hashes prioritized in the arrival order
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

// Insert adds a set of headers for the download queue for scheduling, returning
// the new headers encountered.
func (q *queue) Insert(headers []*types.Header) []*types.Header {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Insert all the headers prioritized by the contained block number
	inserts := make([]*types.Header, 0, len(headers))
	for _, header := range headers {
		// Make sure no duplicate requests are executed
		hash := header.Hash()
		if _, ok := q.headerPool[hash]; ok {
			glog.V(logger.Warn).Infof("Header %x already scheduled", hash)
			continue
		}
		// Queue the header for body retrieval
		inserts = append(inserts, header)
		q.headerPool[hash] = header
		q.headerQueue.Push(header, -float32(header.Number.Uint64()))
	}
	return inserts
}

// GetHeadBlock retrieves the first block from the cache, or nil if it hasn't
// been downloaded yet (or simply non existent).
func (q *queue) GetHeadBlock() *Block {
	q.lock.RLock()
	defer q.lock.RUnlock()

	if len(q.blockCache) == 0 {
		return nil
	}
	return q.blockCache[0]
}

// GetBlock retrieves a downloaded block, or nil if non-existent.
func (q *queue) GetBlock(hash common.Hash) *Block {
	q.lock.RLock()
	defer q.lock.RUnlock()

	// Short circuit if the block hasn't been downloaded yet
	index, ok := q.blockPool[hash]
	if !ok {
		return nil
	}
	// Return the block if it's still available in the cache
	if q.blockOffset <= index && index < q.blockOffset+uint64(len(q.blockCache)) {
		return q.blockCache[index-q.blockOffset]
	}
	return nil
}

// TakeBlocks retrieves and permanently removes a batch of blocks from the cache.
func (q *queue) TakeBlocks() []*Block {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Accumulate all available blocks
	blocks := []*Block{}
	for _, block := range q.blockCache {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		delete(q.blockPool, block.RawBlock.Hash())
	}
	// Delete the blocks from the slice and let them be garbage collected
	// without this slice trick the blocks would stay in memory until nil
	// would be assigned to q.blocks
	copy(q.blockCache, q.blockCache[len(blocks):])
	for k, n := len(q.blockCache)-len(blocks), len(q.blockCache); k < n; k++ {
		q.blockCache[k] = nil
	}
	q.blockOffset += uint64(len(blocks))

	return blocks
}

// Reserve61 reserves a set of hashes for the given peer, skipping any previously
// failed download.
func (q *queue) Reserve61(p *peer, count int) *fetchRequest {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the pool has been depleted, or if the peer's already
	// downloading something (sanity check not to corrupt state)
	if q.hashQueue.Empty() {
		return nil
	}
	if _, ok := q.pendPool[p.id]; ok {
		return nil
	}
	// Calculate an upper limit on the hashes we might fetch (i.e. throttling)
	space := len(q.blockCache) - len(q.blockPool)
	for _, request := range q.pendPool {
		space -= len(request.Hashes)
	}
	// Retrieve a batch of hashes, skipping previously failed ones
	send := make(map[common.Hash]int)
	skip := make(map[common.Hash]int)

	for proc := 0; proc < space && len(send) < count && !q.hashQueue.Empty(); proc++ {
		hash, priority := q.hashQueue.Pop()
		if p.ignored.Has(hash) {
			skip[hash.(common.Hash)] = int(priority)
		} else {
			send[hash.(common.Hash)] = int(priority)
		}
	}
	// Merge all the skipped hashes back
	for hash, index := range skip {
		q.hashQueue.Push(hash, float32(index))
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
	q.pendPool[p.id] = request

	return request
}

// Reserve reserves a set of headers for the given peer, skipping any previously
// failed download. Beside the next batch of needed fetches, it also returns a
// flag whether empty blocks were queued requiring processing.
func (q *queue) Reserve(p *peer, count int) (*fetchRequest, bool, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the pool has been depleted, or if the peer's already
	// downloading something (sanity check not to corrupt state)
	if q.headerQueue.Empty() {
		return nil, false, nil
	}
	if _, ok := q.pendPool[p.id]; ok {
		return nil, false, nil
	}
	// Calculate an upper limit on the bodies we might fetch (i.e. throttling)
	space := len(q.blockCache) - len(q.blockPool)
	for _, request := range q.pendPool {
		space -= len(request.Headers)
	}
	// Retrieve a batch of headers, skipping previously failed ones
	send := make([]*types.Header, 0, count)
	skip := make([]*types.Header, 0)

	process := false
	for proc := 0; proc < space && len(send) < count && !q.headerQueue.Empty(); proc++ {
		header := q.headerQueue.PopItem().(*types.Header)

		// If the header defines an empty block, deliver straight
		if header.TxHash == types.DeriveSha(types.Transactions{}) && header.UncleHash == types.CalcUncleHash([]*types.Header{}) {
			if err := q.enqueue("", types.NewBlockWithHeader(header)); err != nil {
				return nil, false, errInvalidChain
			}
			delete(q.headerPool, header.Hash())
			process, space, proc = true, space-1, proc-1
			continue
		}
		// If it's a content block, add to the body fetch request
		if p.ignored.Has(header.Hash()) {
			skip = append(skip, header)
		} else {
			send = append(send, header)
		}
	}
	// Merge all the skipped headers back
	for _, header := range skip {
		q.headerQueue.Push(header, -float32(header.Number.Uint64()))
	}
	// Assemble and return the block download request
	if len(send) == 0 {
		return nil, process, nil
	}
	request := &fetchRequest{
		Peer:    p,
		Headers: send,
		Time:    time.Now(),
	}
	q.pendPool[p.id] = request

	return request, process, nil
}

// Cancel aborts a fetch request, returning all pending hashes to the queue.
func (q *queue) Cancel(request *fetchRequest) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
	}
	for _, header := range request.Headers {
		q.headerQueue.Push(header, -float32(header.Number.Uint64()))
	}
	delete(q.pendPool, request.Peer.id)
}

// Expire checks for in flight requests that exceeded a timeout allowance,
// canceling them and returning the responsible peers for penalization.
func (q *queue) Expire(timeout time.Duration) []string {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Iterate over the expired requests and return each to the queue
	peers := []string{}
	for id, request := range q.pendPool {
		if time.Since(request.Time) > timeout {
			// Update the metrics with the timeout
			if len(request.Hashes) > 0 {
				blockTimeoutMeter.Mark(1)
			} else {
				bodyTimeoutMeter.Mark(1)
			}
			// Return any non satisfied requests to the pool
			for hash, index := range request.Hashes {
				q.hashQueue.Push(hash, float32(index))
			}
			for _, header := range request.Headers {
				q.headerQueue.Push(header, -float32(header.Number.Uint64()))
			}
			peers = append(peers, id)
		}
	}
	// Remove the expired requests from the pending pool
	for _, id := range peers {
		delete(q.pendPool, id)
	}
	return peers
}

// Deliver61 injects a block retrieval response into the download queue.
func (q *queue) Deliver61(id string, blocks []*types.Block) (err error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the blocks were never requested
	request := q.pendPool[id]
	if request == nil {
		return errNoFetchesPending
	}
	blockReqTimer.UpdateSince(request.Time)
	delete(q.pendPool, id)

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
		// Queue the block up for processing
		if err := q.enqueue(id, block); err != nil {
			return err
		}
		delete(request.Hashes, hash)
		delete(q.hashPool, hash)
	}
	// Return all failed or missing fetches to the queue
	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
	}
	// If none of the blocks were good, it's a stale delivery
	if len(errs) != 0 {
		if len(errs) == len(blocks) {
			return errStaleDelivery
		}
		return fmt.Errorf("multiple failures: %v", errs)
	}
	return nil
}

// Deliver injects a block body retrieval response into the download queue.
func (q *queue) Deliver(id string, txLists [][]*types.Transaction, uncleLists [][]*types.Header) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the block bodies were never requested
	request := q.pendPool[id]
	if request == nil {
		return errNoFetchesPending
	}
	bodyReqTimer.UpdateSince(request.Time)
	delete(q.pendPool, id)

	// If no block bodies were retrieved, mark them as unavailable for the origin peer
	if len(txLists) == 0 || len(uncleLists) == 0 {
		for hash, _ := range request.Headers {
			request.Peer.ignored.Add(hash)
		}
	}
	// Assemble each of the block bodies with their headers and queue for processing
	errs := make([]error, 0)
	for i, header := range request.Headers {
		// Short circuit block assembly if no more bodies are found
		if i >= len(txLists) || i >= len(uncleLists) {
			break
		}
		// Reconstruct the next block if contents match up
		if types.DeriveSha(types.Transactions(txLists[i])) != header.TxHash || types.CalcUncleHash(uncleLists[i]) != header.UncleHash {
			errs = []error{errInvalidBody}
			break
		}
		block := types.NewBlockWithHeader(header).WithBody(txLists[i], uncleLists[i])

		// Queue the block up for processing
		if err := q.enqueue(id, block); err != nil {
			errs = []error{err}
			break
		}
		request.Headers[i] = nil
		delete(q.headerPool, header.Hash())
	}
	// Return all failed or missing fetches to the queue
	for _, header := range request.Headers {
		if header != nil {
			q.headerQueue.Push(header, -float32(header.Number.Uint64()))
		}
	}
	// If none of the blocks were good, it's a stale delivery
	switch {
	case len(errs) == 0:
		return nil

	case len(errs) == 1 && errs[0] == errInvalidBody:
		return errInvalidBody

	case len(errs) == 1 && errs[0] == errInvalidChain:
		return errInvalidChain

	case len(errs) == len(request.Headers):
		return errStaleDelivery

	default:
		return fmt.Errorf("multiple failures: %v", errs)
	}
}

// enqueue inserts a new block into the final delivery queue, waiting for pickup
// by the processor.
func (q *queue) enqueue(origin string, block *types.Block) error {
	// If a requested block falls out of the range, the hash chain is invalid
	index := int(int64(block.NumberU64()) - int64(q.blockOffset))
	if index >= len(q.blockCache) || index < 0 {
		return errInvalidChain
	}
	// Otherwise merge the block and mark the hash done
	q.blockCache[index] = &Block{
		RawBlock:   block,
		OriginPeer: origin,
	}
	q.blockPool[block.Header().Hash()] = block.NumberU64()
	return nil
}

// Prepare configures the block cache offset to allow accepting inbound blocks.
func (q *queue) Prepare(offset uint64) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.blockOffset < offset {
		q.blockOffset = offset
	}
}
