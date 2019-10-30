// Copyright 2019 The go-ethereum Authors
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

// resultcache implements a structure for maintaining fetchResults, tracking their
// download-progress and delivering (finished) results

package downloader

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type resultStore struct {
	items        []*fetchResult // Downloaded but not yet delivered fetch results
	lock         *sync.RWMutex  // lock protect internals
	resultOffset uint64         // Offset of the first cached fetch result in the block chain

	// Internal index of first non-completed entry, updated atomically when needed.
	// If all items are complete, this will equal length(items), so
	// *important* : is not safe to use for indexing without checking against length
	indexIncomplete int32 // atomic access

	// throttleThreshold is the limit up to which we _want_ to fill the
	// results. If blocks are large, we want to limit the results to less
	// than the number of available slots, and maybe only fill 1024 out of
	// 8192 possible places. The queue will, at certain times, recalibrate
	// this index.
	throttleThreshold uint64
}

func newResultStore(size int) *resultStore {
	return &resultStore{
		resultOffset:      0,
		items:             make([]*fetchResult, size),
		lock:              new(sync.RWMutex),
		throttleThreshold: 3 * uint64(size) / 4, // 75%
	}
}

func (r *resultStore) SetThrottleThreshold(threshold uint64) {
	r.lock.Lock()
	defer r.lock.Unlock()
	limit := uint64(len(r.items)) * 3 / 4
	if threshold >= limit {
		threshold = limit
	}
	r.throttleThreshold = threshold
}

// AddFetch adds a header for body/receipt fetching. This is used when the queue
// wants to reserve headers for fetching.
// It returns the following:
// stale       -- if true, this item is already passed, and should not be requested again.
// throttled   -- if true, the resultcache is at capacity, and this particular header is not
//                prio right now
// fetchResult -- the result to store data into
// err         -- any error that occurred
func (r *resultStore) AddFetch(header *types.Header, fastSync bool) (stale, throttled bool, item *fetchResult, err error) {
	header.Hash()
	r.lock.RLock()
	var index int
	if item, index, stale, throttled, err = r.getFetchResult(header); err != nil {
		r.lock.RUnlock()
		return
	}
	if stale {
		r.lock.RUnlock()
		return
	}
	if throttled {
		// Index is above the current threshold of 'prioritized' blocks,
		log.Debug("resultcache throttle", "index", index, "threshold", r.throttleThreshold)
		r.lock.RUnlock()
		return
	}
	if item != nil {
		// All good, item already exists (perhaps a receipt fetch following
		// a body fetch)
		r.lock.RUnlock()
		return
	}
	r.lock.RUnlock()
	// Need to create a fetchresult, and as we've just release the Rlock,
	// we need to check again after obtaining the writelock
	r.lock.Lock()
	defer r.lock.Unlock()
	// Same checks as above, now with wlock
	if item, index, stale, throttled, err = r.getFetchResult(header); err != nil {
		return
	}
	if stale || throttled {
		return
	}
	if item == nil {
		item = newFetchResult(header, fastSync)
		r.items[index] = item
	}
	return
}

// GetFetchResult returns the fetchResult for the given header. If the 'stale' flag
// is true, that means the header has already been delivered 'upstream'.
func (r *resultStore) GetFetchResult(header *types.Header) (*fetchResult, bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	res, _, stale, _, err := r.getFetchResult(header)
	return res, stale, err
}

// getFetchResult returns the fetchResult corresponding to the given item, and the index where
// the result is stored.
func (r *resultStore) getFetchResult(header *types.Header) (item *fetchResult, index int, stale, throttle bool, err error) {

	index = int(header.Number.Int64() - int64(r.resultOffset))
	throttle = index >= int(r.throttleThreshold)
	stale = index < 0

	if index >= len(r.items) {
		err = fmt.Errorf("index allocation went beyond available resultStore space "+
			"(index [%d] = header [%d] - resultOffset [%d], len(resultStore) = %d",
			index, header.Number.Int64(), r.resultOffset, len(r.items))
		return
	}
	if stale {
		return
	}
	item = r.items[index]
	return
}

// hasCompletedItems returns true if there are processable items available
// this method is cheaper than countCompleted
func (r *resultStore) HasCompletedItems() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if len(r.items) == 0 {
		return false
	}
	if item := r.items[0]; item != nil && item.AllDone() {
		return true
	}
	return false
}

// countCompleted returns the number of items ready for delivery, stopping at
// the first non-complete item.
// It assumes (at least) rlock is held
func (r *resultStore) countCompleted() int {
	// We iterate from the already known complete point, and see
	// if any more has completed since last count
	index := atomic.LoadInt32(&r.indexIncomplete)
	for ; ; index++ {
		if index >= int32(len(r.items)) {
			break
		}
		result := r.items[index]
		if result == nil || !result.AllDone() {
			break
		}
	}
	atomic.StoreInt32(&r.indexIncomplete, index)
	return int(index)
}

// GetCompleted returns the next batch of completed fetchResults
func (r *resultStore) GetCompleted(limit int) []*fetchResult {
	r.lock.Lock()
	defer r.lock.Unlock()

	completed := r.countCompleted()
	if limit > completed {
		limit = completed
	}
	results := make([]*fetchResult, limit)
	copy(results, r.items[:limit])

	// Delete the results from the cache and clear the tail.
	copy(r.items, r.items[limit:])
	for i := len(r.items) - limit; i < len(r.items); i++ {
		r.items[i] = nil
	}
	// Advance the expected block number of the first cache entry.
	r.resultOffset += uint64(limit)
	// And subtract the number of items from our index
	atomic.AddInt32(&r.indexIncomplete, int32(-limit))
	return results
}

// Prepare initialises the offset with the given block number
func (r *resultStore) Prepare(offset uint64) {
	r.lock.Lock()
	if r.resultOffset < offset {
		r.resultOffset = offset
	}
	r.lock.Unlock()
}
