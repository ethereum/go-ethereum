// Copyright 2026 The go-ethereum Authors
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

package blobpool

import (
	"container/heap"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/billy"
	"github.com/holiman/uint256"
)

// cacheEntry is a blob transaction stored in the cache.
type cacheEntry struct {
	ptx   *blobTxForPool
	tip   *uint256.Int // priority key
	txID  uint64       // store id
	index int          // position in priorityQueue
}

// priorityQueue is a min heap of cacheEntries, ordered by tip.
type priorityQueue []*cacheEntry

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].tip.Lt(pq[j].tip)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x any) {
	e := x.(*cacheEntry)
	e.index = len(*pq)
	*pq = append(*pq, e)
}
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	e.index = -1
	*pq = old[:n-1]
	return e
}

// cache has blob transactions ordered by execution tip.
type cache struct {
	mu       sync.Mutex
	entries  map[uint64]*cacheEntry // store id -> cache entry
	pq       priorityQueue
	capacity int
	store    billy.Database
}

func newCache(store billy.Database, capacity int) *cache {
	return &cache{
		entries:  make(map[uint64]*cacheEntry, capacity),
		pq:       make(priorityQueue, 0, capacity),
		capacity: capacity,
		store:    store,
	}
}

// get returns the cached decoded transaction for the given storage id, if any.
func (c *cache) get(id uint64) *blobTxForPool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[id]
	if !ok {
		return nil
	}
	return e.ptx
}

type addedTx struct {
	id  uint64
	ptx *blobTxForPool
}

// update applies a batch of additions and deletions in blobpool.
// Each addition is inserted only if it is more expansive than the
// lowest-tip tx or there is enough capacity.
func (c *cache) update(added []*addedTx, deleted []uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, id := range deleted {
		e, ok := c.entries[id]
		if !ok {
			continue
		}
		heap.Remove(&c.pq, e.index)
		delete(c.entries, id)
	}
	for _, tx := range added {
		if _, exists := c.entries[tx.id]; exists {
			continue
		}
		tip := uint256.MustFromBig(tx.ptx.Tx.GasTipCap())
		if len(c.entries) >= c.capacity {
			if !tip.Gt(c.pq[0].tip) {
				continue
			}
			evicted := heap.Pop(&c.pq).(*cacheEntry)
			delete(c.entries, evicted.txID)
		}
		e := &cacheEntry{
			ptx:  tx.ptx,
			tip:  tip,
			txID: tx.id,
		}
		c.entries[tx.id] = e
		heap.Push(&c.pq, e)
	}
	cacheEntriesGauge.Update(int64(len(c.entries)))
}

// reset rebuilds the cache with the given wanted ids by computing the
// diff against the current entries.
func (c *cache) reset(wanted []uint64) {
	wantedSet := make(map[uint64]struct{}, len(wanted))
	for _, id := range wanted {
		wantedSet[id] = struct{}{}
	}

	c.mu.Lock()
	var toDelete []uint64
	for id := range c.entries {
		if _, ok := wantedSet[id]; !ok {
			toDelete = append(toDelete, id)
		}
	}
	var toLoad []uint64
	for _, id := range wanted {
		if _, ok := c.entries[id]; !ok {
			toLoad = append(toLoad, id)
		}
	}
	c.mu.Unlock()

	var toAdd []*addedTx
	for _, id := range toLoad {
		data, err := c.store.Get(id)
		if err != nil {
			log.Trace("Cache load skipped missing blob tx", "id", id, "err", err)
			continue
		}
		var ptx blobTxForPool
		if err := rlp.DecodeBytes(data, &ptx); err != nil {
			log.Error("Cache load failed to decode blob tx", "id", id, "err", err)
			continue
		}
		toAdd = append(toAdd, &addedTx{id: id, ptx: &ptx})
	}

	c.update(toAdd, toDelete)
}
