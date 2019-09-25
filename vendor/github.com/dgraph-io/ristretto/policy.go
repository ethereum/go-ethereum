/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ristretto

import (
	"container/list"
	"math"
	"sync"

	"github.com/dgraph-io/ristretto/z"
)

const (
	// lfuSample is the number of items to sample when looking at eviction
	// candidates. 5 seems to be the most optimal number [citation needed].
	lfuSample = 5
)

// policy is the interface encapsulating eviction/admission behavior.
type policy interface {
	ringConsumer
	// Add attempts to Add the key-cost pair to the Policy. It returns a slice
	// of evicted keys and a bool denoting whether or not the key-cost pair
	// was added. If it returns true, the key should be stored in cache.
	Add(uint64, int64) ([]*item, bool)
	// Has returns true if the key exists in the Policy.
	Has(uint64) bool
	// Del deletes the key from the Policy.
	Del(uint64)
	// Cap returns the available capacity.
	Cap() int64
	// Optionally, set stats object to track how policy is performing.
	CollectMetrics(stats *metrics)
}

func newPolicy(numCounters, maxCost int64) policy {
	p := &defaultPolicy{
		admit:   newTinyLFU(numCounters),
		evict:   newSampledLFU(maxCost),
		itemsCh: make(chan []uint64, 3),
	}
	// TODO: Add a way to stop the goroutine.
	go p.processItems()
	return p
}

// defaultPolicy is the default defaultPolicy, which is currently TinyLFU
// admission with sampledLFU eviction.
type defaultPolicy struct {
	sync.Mutex
	admit   *tinyLFU
	evict   *sampledLFU
	itemsCh chan []uint64
	stats   *metrics
}

func (p *defaultPolicy) CollectMetrics(stats *metrics) {
	p.stats = stats
	p.evict.stats = stats
}

type policyPair struct {
	key  uint64
	cost int64
}

func (p *defaultPolicy) processItems() {
	for items := range p.itemsCh {
		p.Lock()
		p.admit.Push(items)
		p.Unlock()
	}
}

func (p *defaultPolicy) Push(keys []uint64) bool {
	if len(keys) == 0 {
		return true
	}
	select {
	case p.itemsCh <- keys:
		p.stats.Add(keepGets, keys[0], uint64(len(keys)))
		return true
	default:
		p.stats.Add(dropGets, keys[0], uint64(len(keys)))
		return false
	}
}

func (p *defaultPolicy) Add(key uint64, cost int64) ([]*item, bool) {
	p.Lock()
	defer p.Unlock()
	// can't add an item bigger than entire cache
	if cost > p.evict.maxCost {
		return nil, false
	}
	// we don't need to go any further if the item is already in the cache
	if has := p.evict.updateIfHas(key, cost); has {
		return nil, true
	}
	// if we got this far, this key doesn't exist in the cache
	//
	// calculate the remaining room in the cache (usually bytes)
	room := p.evict.roomLeft(cost)
	if room >= 0 {
		// there's enough room in the cache to store the new item without
		// overflowing, so we can do that now and stop here
		p.evict.add(key, cost)
		return nil, true
	}
	// incHits is the hit count for the incoming item
	incHits := p.admit.Estimate(key)
	// sample is the eviction candidate pool to be filled via random sampling
	//
	// TODO: perhaps we should use a min heap here. Right now our time
	// complexity is N for finding the min. Min heap should bring it down to
	// O(lg N).
	sample := make([]*policyPair, 0, lfuSample)
	// as items are evicted they will be appended to victims
	victims := make([]*item, 0)
	// Delete victims until there's enough space or a minKey is found that has
	// more hits than incoming item.
	for ; room < 0; room = p.evict.roomLeft(cost) {
		// fill up empty slots in sample
		sample = p.evict.fillSample(sample)
		// find minimally used item in sample
		minKey, minHits, minId, minCost := uint64(0), int64(math.MaxInt64), 0, int64(0)
		for i, pair := range sample {
			// look up hit count for sample key
			if hits := p.admit.Estimate(pair.key); hits < minHits {
				minKey, minHits, minId, minCost = pair.key, hits, i, pair.cost
			}
		}
		// If the incoming item isn't worth keeping in the policy, reject.
		if incHits < minHits {
			p.stats.Add(rejectSets, key, 1)
			return victims, false
		}
		// delete the victim from metadata
		p.evict.del(minKey)
		// delete the victim from sample
		sample[minId] = sample[len(sample)-1]
		sample = sample[:len(sample)-1]
		// store victim in evicted victims slice
		victims = append(victims, &item{minKey, nil, minCost})
	}
	p.evict.add(key, cost)
	return victims, true
}

func (p *defaultPolicy) Has(key uint64) bool {
	p.Lock()
	defer p.Unlock()
	_, exists := p.evict.keyCosts[key]
	return exists
}

func (p *defaultPolicy) Del(key uint64) {
	p.Lock()
	defer p.Unlock()
	p.evict.del(key)
}

func (p *defaultPolicy) Cap() int64 {
	p.Lock()
	defer p.Unlock()
	return int64(p.evict.maxCost - p.evict.used)
}

// sampledLFU is an eviction helper storing key-cost pairs.
type sampledLFU struct {
	keyCosts map[uint64]int64
	maxCost  int64
	used     int64
	stats    *metrics
}

func newSampledLFU(maxCost int64) *sampledLFU {
	return &sampledLFU{
		keyCosts: make(map[uint64]int64),
		maxCost:  maxCost,
	}
}

func (p *sampledLFU) roomLeft(cost int64) int64 {
	return p.maxCost - (p.used + cost)
}

func (p *sampledLFU) fillSample(in []*policyPair) []*policyPair {
	if len(in) >= lfuSample {
		return in
	}
	for key, cost := range p.keyCosts {
		in = append(in, &policyPair{key, cost})
		if len(in) >= lfuSample {
			return in
		}
	}
	return in
}

func (p *sampledLFU) del(key uint64) {
	cost, ok := p.keyCosts[key]
	if !ok {
		return
	}

	p.stats.Add(keyEvict, key, 1)
	p.stats.Add(costEvict, key, uint64(cost))

	p.used -= cost
	delete(p.keyCosts, key)
}

func (p *sampledLFU) add(key uint64, cost int64) {
	p.stats.Add(keyAdd, key, 1)
	p.stats.Add(costAdd, key, uint64(cost))

	p.keyCosts[key] = cost
	p.used += cost
}

// TODO: Move this to the store itself. So, it can be used by public Set.
func (p *sampledLFU) updateIfHas(key uint64, cost int64) (updated bool) {
	if prev, exists := p.keyCosts[key]; exists {
		// Update the cost of the existing key. For simplicity, don't worry about evicting anything
		// if the updated cost causes the size to grow beyond maxCost.
		p.stats.Add(keyUpdate, key, 1)
		p.used += cost - prev
		p.keyCosts[key] = cost
		return true
	}
	return false
}

// tinyLFU is an admission helper that keeps track of access frequency using
// tiny (4-bit) counters in the form of a count-min sketch.
// tinyLFU is NOT thread safe.
type tinyLFU struct {
	freq    *cmSketch
	door    *z.Bloom
	incrs   int64
	resetAt int64
}

func newTinyLFU(numCounters int64) *tinyLFU {
	return &tinyLFU{
		freq:    newCmSketch(numCounters),
		door:    z.NewBloomFilter(float64(numCounters), 0.01),
		resetAt: numCounters,
	}
}

func (p *tinyLFU) Push(keys []uint64) {
	for _, key := range keys {
		p.Increment(key)
	}
}

func (p *tinyLFU) Estimate(key uint64) int64 {
	hits := p.freq.Estimate(key)
	if p.door.Has(key) {
		hits += 1
	}
	return hits
}

func (p *tinyLFU) Increment(key uint64) {
	// flip doorkeeper bit if not already
	if added := p.door.AddIfNotHas(key); !added {
		// increment count-min counter if doorkeeper bit is already set.
		p.freq.Increment(key)
	}
	p.incrs++
	if p.incrs >= p.resetAt {
		p.reset()
	}
}

func (p *tinyLFU) reset() {
	// Zero out incrs.
	p.incrs = 0
	// clears doorkeeper bits
	p.door.Clear()
	// halves count-min counters
	p.freq.Reset()
}

// lruPolicy is different than the default policy in that it uses exact LRU
// eviction rather than Sampled LFU eviction, which may be useful for certain
// workloads (ARC-OLTP for example; LRU heavy workloads).
//
// TODO: - cost based eviction (multiple evictions for one new item, etc.)
//       - sampled LRU
type lruPolicy struct {
	sync.Mutex
	admit   *tinyLFU
	ptrs    map[uint64]*lruItem
	vals    *list.List
	maxCost int64
	room    int64
}

type lruItem struct {
	ptr  *list.Element
	key  uint64
	cost int64
}

func newLRUPolicy(numCounters, maxCost int64) policy {
	return &lruPolicy{
		admit:   newTinyLFU(numCounters),
		ptrs:    make(map[uint64]*lruItem, maxCost),
		vals:    list.New(),
		room:    maxCost,
		maxCost: maxCost,
	}
}

func (p *lruPolicy) Push(keys []uint64) bool {
	if len(keys) == 0 {
		return true
	}
	p.Lock()
	defer p.Unlock()
	for _, key := range keys {
		// increment tinylfu counter
		p.admit.Increment(key)
		// move list item to front
		if val, ok := p.ptrs[key]; ok {
			// move accessed val to MRU position
			p.vals.MoveToFront(val.ptr)
		}
	}
	return true
}

func (p *lruPolicy) Add(key uint64, cost int64) ([]*item, bool) {
	p.Lock()
	defer p.Unlock()
	if cost > p.maxCost {
		return nil, false
	}
	if val, has := p.ptrs[key]; has {
		p.vals.MoveToFront(val.ptr)
		return nil, true
	}
	victims := make([]*item, 0)
	incHits := p.admit.Estimate(key)
	if p.room >= 0 {
		goto add
	}
	for p.room < 0 {
		lru := p.vals.Back()
		victim := lru.Value.(*lruItem)
		if incHits < p.admit.Estimate(victim.key) {
			return victims, false
		}
		// delete victim from metadata
		p.vals.Remove(victim.ptr)
		delete(p.ptrs, victim.key)
		victims = append(victims, &item{victim.key, nil, victim.cost})
		// adjust room
		p.room += victim.cost
	}
add:
	item := &lruItem{key: key, cost: cost}
	item.ptr = p.vals.PushFront(item)
	p.ptrs[key] = item
	p.room -= cost
	return victims, true
}

func (p *lruPolicy) Has(key uint64) bool {
	p.Lock()
	defer p.Unlock()
	_, has := p.ptrs[key]
	return has
}

func (p *lruPolicy) Del(key uint64) {
	p.Lock()
	defer p.Unlock()
	if val, ok := p.ptrs[key]; ok {
		p.vals.Remove(val.ptr)
		delete(p.ptrs, key)
	}
}

func (p *lruPolicy) Cap() int64 {
	p.Lock()
	defer p.Unlock()
	return int64(p.vals.Len())
}

// TODO
func (p *lruPolicy) CollectMetrics(stats *metrics) {
}
