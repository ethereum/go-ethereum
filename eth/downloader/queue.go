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

const (
	blockCacheLimit = 8 * MaxBlockFetch // Maximum number of blocks to cache before throttling the download
)

// fetchRequest is a currently running block retrieval operation.
type fetchRequest struct {
	Peer   *peer               // Peer to which the request was sent
	Hashes map[common.Hash]int // Requested hashes with their insertion index (priority)
	Time   time.Time           // Time when the request was made
}

// queue represents hashes that are either need fetching or are being fetched
type queue struct {
	hashPool    map[common.Hash]int // Pending hashes, mapping to their insertion index (priority)
	hashQueue   *prque.Prque        // Priority queue of the block hashes to fetch
	hashCounter int                 // Counter indexing the added hashes to ensure retrieval order

	pendPool map[string]*fetchRequest // Currently pending block retrieval operations

	blockPool   map[common.Hash]int // Hash-set of the downloaded data blocks, mapping to cache indexes
	blockCache  []*Block            // Downloaded but not yet delivered blocks
	blockOffset int                 // Offset of the first cached block in the block-chain

	lock sync.RWMutex
}

// newQueue creates a new download queue for scheduling block retrieval.
func newQueue() *queue {
	return &queue{
		hashPool:  make(map[common.Hash]int),
		hashQueue: prque.New(),
		pendPool:  make(map[string]*fetchRequest),
		blockPool: make(map[common.Hash]int),
	}
}

// Reset clears out the queue contents.
func (q *queue) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.hashPool = make(map[common.Hash]int)
	q.hashQueue.Reset()
	q.hashCounter = 0

	q.pendPool = make(map[string]*fetchRequest)

	q.blockPool = make(map[common.Hash]int)
	q.blockOffset = 0
	q.blockCache = nil
}

// Size retrieves the number of hashes in the queue, returning separately for
// pending and already downloaded.
func (q *queue) Size() (int, int) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.hashPool), len(q.blockPool)
}

// Pending retrieves the number of hashes pending for retrieval.
func (q *queue) Pending() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return q.hashQueue.Size()
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
		pending += len(request.Hashes)
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
	if _, ok := q.blockPool[hash]; ok {
		return true
	}
	return false
}

// Insert adds a set of hashes for the download queue for scheduling, returning
// the new hashes encountered.
func (q *queue) Insert(hashes []common.Hash) []common.Hash {
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
		q.hashQueue.Push(hash, float32(q.hashCounter)) // Highest gets schedules first
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
	if q.blockOffset <= index && index < q.blockOffset+len(q.blockCache) {
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
	q.blockOffset += len(blocks)

	return blocks
}

// Reserve reserves a set of hashes for the given peer, skipping any previously
// failed download.
func (q *queue) Reserve(p *peer) *fetchRequest {
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

	capacity := p.Capacity()
	for len(send) < space && len(send) < capacity && !q.hashQueue.Empty() {
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

// Cancel aborts a fetch request, returning all pending hashes to the queue.
func (q *queue) Cancel(request *fetchRequest) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
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
			for hash, index := range request.Hashes {
				q.hashQueue.Push(hash, float32(index))
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

// Deliver injects a block retrieval response into the download queue.
func (q *queue) Deliver(id string, blocks []*types.Block) (err error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Short circuit if the blocks were never requested
	request := q.pendPool[id]
	if request == nil {
		return errors.New("no fetches pending")
	}
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
			errs = append(errs, fmt.Errorf("non-requested block %v", hash))
			continue
		}
		// If a requested block falls out of the range, the hash chain is invalid
		index := int(block.NumberU64()) - q.blockOffset
		if index >= len(q.blockCache) || index < 0 {
			return ErrInvalidChain
		}
		// Otherwise merge the block and mark the hash block
		q.blockCache[index] = &Block{
			RawBlock:   block,
			OriginPeer: id,
		}
		delete(request.Hashes, hash)
		delete(q.hashPool, hash)
		q.blockPool[hash] = int(block.NumberU64())
	}
	// Return all failed fetches to the queue
	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
	}
	if len(errs) != 0 {
		return fmt.Errorf("multiple failures: %v", errs)
	}
	return nil
}

// Alloc ensures that the block cache is the correct size, given a starting
// offset, and a memory cap.
func (q *queue) Alloc(offset int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.blockOffset < offset {
		q.blockOffset = offset
	}
	size := len(q.hashPool)
	if size > blockCacheLimit {
		size = blockCacheLimit
	}
	if len(q.blockCache) < size {
		q.blockCache = append(q.blockCache, make([]*Block, size-len(q.blockCache))...)
	}
}
