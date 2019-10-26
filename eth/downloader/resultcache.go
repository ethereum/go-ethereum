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
	"github.com/ethereum/go-ethereum/log"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type resultStore struct {
	items        []*fetchResult     // Downloaded but not yet delivered fetch results
	lock         *sync.RWMutex      // lock protect internals
	resultOffset uint64             // Offset of the first cached fetch result in the block chain
	resultSize   common.StorageSize // Approximate size of a block (exponential moving average)

	// Internal index of first non-completed entry, updated atomically when needed.
	// If all items are complete, this will equal length(items), so
	// *important* : is not safe to use for indexing without checking against length
	indexIncomplete int32
}

func newResultStore(size int) *resultStore {
	return &resultStore{
		resultOffset: 0,
		items:        make([]*fetchResult, size),
		resultSize:   0, // TODO: use a saner default, left at zero as it was legacy
		lock:         new(sync.RWMutex),
	}
}

// AddFetch adds a header for body/receipt fetching.
// returning
// stale -- if true, this item is already passed, and should not be requested again
// fetchResult -- if `nil`, that means no fetch was created, and that the
// if an error is returned, that most likely means backpressure prevents the results from expanding,
// and someone needs to take care of results
func (r *resultStore) AddFetch(header *types.Header, fastSync bool) (stale bool, item *fetchResult, err error) {
	hash := header.Hash()
	r.lock.RLock()
	item, _, stale, err = r.getFetchResult(header)
	if err != nil {
		r.lock.RUnlock()
		log.Info("resultcache addfetch err [1]", "error", err.Error())
		// can't create a fetchResult right away
		return stale, nil, err
	}
	if item != nil {
		r.lock.RUnlock()
		return false, item, nil
	}
	r.lock.RUnlock()
	// Need to create a fetchresult, and as we've just release the Rlock,
	// we need to check again after obtaining the writelock
	r.lock.Lock()
	defer r.lock.Unlock()
	var index int
	item, index, stale, err = r.getFetchResult(header)
	if err != nil {
		// can't create a fetchResult right away
		log.Info("resultcache addfetch err [2]", "error", err.Error())
		return stale, nil, err

	}
	if item == nil {
		item = &fetchResult{
			Hash:   hash,
			Header: header,
		}
		// Need to fetch body?
		if !header.EmptyBody() {
			// yes
			item.Pending |= 0x1
		}
		// Do we need to fetch receipts?
		if fastSync && !header.EmptyReceipts() {
			item.Pending |= 0x2
		}
		r.items[index] = item
	}
	return false, item, nil

}

func (r *resultStore) GetFetchResult(header *types.Header) (*fetchResult, bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	res, _, stale, err := r.getFetchResult(header)
	return res, stale, err
}

// getFetchResult returns the fetchResult corresponding to the given item, and the index where
// the result is stored.
// There are two ways it can error:
// 1. The header is too far off in the future, and we don't have room for it.
// 2. The header is stale, and the results for that header has already been delivered upstream.
func (r *resultStore) getFetchResult(header *types.Header) (item *fetchResult, index int, stale bool, err error) {
	index = int(header.Number.Int64() - int64(r.resultOffset))
	if index >= len(r.items) {
		err = fmt.Errorf("index allocation went beyond available resultStore space "+
			"(index [%d] = header [%d] - resultOffset [%d], len(resultStore) = %d",
			index, header.Number.Int64(), r.resultOffset, len(r.items))
		return item, index, stale, err
	}
	if index < 0 {
		stale = true
		err = fmt.Errorf("index allocation went beyond available resultStore space "+
			"(index [%d] = header [%d] - resultOffset [%d], len(resultStore) = %d",
			index, header.Number.Int64(), r.resultOffset, len(r.items))
		return item, index, stale, err
	}
	item = r.items[index]
	return item, index, stale, err
}

// numberSpan returns the header number start and end, for the headers
// currently "allocated" for download (both completed, in-flight and pending)
func (r *resultStore) NumberSpan() (uint64, uint64) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.resultOffset, r.resultOffset + uint64(len(r.items))

}

// hasCompletedItems returns true if there are processable items available
// this method is cheaper than countCompleted
func (r *resultStore) HasCompletedItems() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if len(r.items) == 0 {
		return false
	}
	if item := r.items[0]; item != nil && item.Pending == 0 {
		return true
	}
	return false
}

// CountCompleted returns the number of items completed
func (r *resultStore) CountCompleted() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.countCompleted()
}

// countCompleted returns the number of items completed
// assumes (at least) rlock is held
func (r *resultStore) countCompleted() int {
	// We iterate from the already known complete point, and see
	// if any more has completed since last count
	// debug
	/*
		var (
			nils         = 0
			fins         = 0
			bodyneeds    = 0
			receiptneeds = 0
		)
		var ctx []interface{}
		for _, item := range r.items {
			if item == nil {
				nils++
			} else {
				if item.Pending == 0 {
					fins++
				} else {
					if item.Pending&0x01 != 0 {
						bodyneeds++
					} else {
						receiptneeds++
					}
				}
			}
		}
		ctx = append(ctx, "items", len(r.items), "nils", nil, "fins", fins,
			"needB", bodyneeds, "needR", receiptneeds)
	*/
	/// end debug
	index := atomic.LoadInt32(&r.indexIncomplete)
	for ; ; index++ {
		if index >= int32(len(r.items)) {
			break
		}
		result := r.items[index]
		if result == nil || result.Pending > 0 {
			break
		}
	}
	/*
		if index < int32(len(r.items)) {
			//ctx = append(ctx, []interface{}{"index", index, "blocknum", uint64(index) + r.resultOffset}...)
			if r.items[index] != nil {
				log.Info("resultstore", ctx...)
			} else {
				ctx = append(ctx, []interface{}{"first missing", "nil"}...)
				log.Info("resultstore", ctx...)
			}
		} else {
			ctx = append(ctx, []interface{}{"first missing", "out of range"}...)
			log.Info("resultstore", ctx...)
		}
	*/
	atomic.StoreInt32(&r.indexIncomplete, index)
	return int(index)
}

// getCompleted returns the next batch of completed fetchresults
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
	// And subtract the number of items from our two indexes
	atomic.StoreInt32(&r.indexIncomplete, int32(completed-limit))
	return results
}

func (r *resultStore) Prepare(offset uint64) {
	r.lock.Lock()
	if r.resultOffset < offset {
		r.resultOffset = offset
	}
	r.lock.Unlock()
}
