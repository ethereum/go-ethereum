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

// Ristretto is a fast, fixed size, in-memory cache with a dual focus on
// throughput and hit ratio performance. You can easily add Ristretto to an
// existing system and keep the most valuable data where you need it.
package ristretto

import (
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/dgraph-io/ristretto/z"
)

// Cache is a thread-safe implementation of a hashmap with a TinyLFU admission
// policy and a Sampled LFU eviction policy. You can use the same Cache instance
// from as many goroutines as you want.
type Cache struct {
	// store is the central concurrent hashmap where key-value items are stored
	store store
	// policy determines what gets let in to the cache and what gets kicked out
	policy policy
	// getBuf is a custom ring buffer implementation that gets pushed to when
	// keys are read
	getBuf *ringBuffer
	// setBuf is a buffer allowing us to batch/drop Sets during times of high
	// contention
	setBuf chan *item
	// stats contains a running log of important statistics like hits, misses,
	// and dropped items
	stats *metrics
	// onEvict is called for item evictions
	onEvict func(uint64, interface{}, int64)
	// KeyToHash function is used to customize the key hashing algorithm.
	// Each key will be hashed using the provided function. If keyToHash value
	// is not set, the default keyToHash function is used.
	keyToHash func(interface{}) uint64
}

// Config is passed to NewCache for creating new Cache instances.
type Config struct {
	// NumCounters determines the number of counters (keys) to keep that hold
	// access frequency information. It's generally a good idea to have more
	// counters than the max cache capacity, as this will improve eviction
	// accuracy and subsequent hit ratios.
	//
	// For example, if you expect your cache to hold 1,000,000 items when full,
	// NumCounters should be 10,000,000 (10x). Each counter takes up 4 bits, so
	// keeping 10,000,000 counters would require 5MB of memory.
	NumCounters int64
	// MaxCost can be considered as the cache capacity, in whatever units you
	// choose to use.
	//
	// For example, if you want the cache to have a max capacity of 100MB, you
	// would set MaxCost to 100,000,000 and pass an item's number of bytes as
	// the `cost` parameter for calls to Set. If new items are accepted, the
	// eviction process will take care of making room for the new item and not
	// overflowing the MaxCost value.
	MaxCost int64
	// BufferItems determines the size of Get buffers.
	//
	// Unless you have a rare use case, using `64` as the BufferItems value
	// results in good performance.
	BufferItems int64
	// Metrics determines whether cache statistics are kept during the cache's
	// lifetime. There *is* some overhead to keeping statistics, so you should
	// only set this flag to true when testing or throughput performance isn't a
	// major factor.
	Metrics bool
	// OnEvict is called for every eviction and passes the hashed key, value,
	// and cost to the function.
	OnEvict func(key uint64, value interface{}, cost int64)
	// KeyToHash function is used to customize the key hashing algorithm.
	// Each key will be hashed using the provided function. If keyToHash value
	// is not set, the default keyToHash function is used.
	KeyToHash func(key interface{}) uint64
}

// item is passed to setBuf so items can eventually be added to the cache
type item struct {
	key  uint64
	val  interface{}
	cost int64
}

// NewCache returns a new Cache instance and any configuration errors, if any.
func NewCache(config *Config) (*Cache, error) {
	switch {
	case config.NumCounters == 0:
		return nil, errors.New("NumCounters can't be zero.")
	case config.MaxCost == 0:
		return nil, errors.New("MaxCost can't be zero.")
	case config.BufferItems == 0:
		return nil, errors.New("BufferItems can't be zero.")
	}
	policy := newPolicy(config.NumCounters, config.MaxCost)
	cache := &Cache{
		store:  newStore(),
		policy: policy,
		getBuf: newRingBuffer(ringLossy, &ringConfig{
			Consumer: policy,
			Capacity: config.BufferItems,
		}),
		// TODO: size configuration for this? like BufferItems but for setBuf?
		setBuf:    make(chan *item, 32*1024),
		onEvict:   config.OnEvict,
		keyToHash: config.KeyToHash,
	}
	if config.Metrics {
		cache.collectMetrics()
	}
	// We can possibly make this configurable. But having 2 goroutines
	// processing this seems sufficient for now.
	//
	// TODO: Allow a way to stop these goroutines.
	for i := 0; i < 2; i++ {
		go cache.processItems()
	}
	return cache, nil
}

// Get returns the value (if any) and a boolean representing whether the
// value was found or not. The value can be nil and the boolean can be true at
// the same time.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	if c == nil {
		return nil, false
	}
	hash := c.keyHash(key)
	c.getBuf.Push(hash)
	val, ok := c.store.Get(hash)
	if ok {
		c.stats.Add(hit, hash, 1)
	} else {
		c.stats.Add(miss, hash, 1)
	}
	return val, ok
}

// keyHash generates the hash for a given key using the cutom keyToHash function, if provided.
// Otherwise it generates the hash using the z.KeyToHash funcion.
func (c *Cache) keyHash(key interface{}) uint64 {
	if c.keyToHash != nil {
		return c.keyToHash(key)
	}
	return z.KeyToHash(key)
}

// Set attempts to add the key-value item to the cache. If it returns false,
// then the Set was dropped and the key-value item isn't added to the cache. If
// it returns true, there's still a chance it could be dropped by the policy if
// its determined that the key-value item isn't worth keeping, but otherwise the
// item will be added and other items will be evicted in order to make room.
func (c *Cache) Set(key interface{}, val interface{}, cost int64) bool {
	if c == nil {
		return false
	}
	hash := c.keyHash(key)
	// TODO: Add a c.store.UpdateIfPresent here. This would catch any value updates and avoid having
	// to push the key in setBuf.

	// attempt to add the (possibly) new item to the setBuf where it will later
	// be processed by the policy and evaluated
	select {
	case c.setBuf <- &item{key: hash, val: val, cost: cost}:
		return true
	default:
		// drop the set and avoid blocking
		c.stats.Add(dropSets, hash, 1)
		return false
	}
}

// TODO: Add a public Update function, which would update a key only if present.

// Del deletes the key-value item from the cache if it exists.
func (c *Cache) Del(key interface{}) {
	if c == nil {
		return
	}
	hash := c.keyHash(key)
	c.policy.Del(hash)
	c.store.Del(hash)
}

// Close stops all goroutines and closes all channels.
func (c *Cache) Close() {}

// processItems is ran by goroutines processing the Set buffer.
func (c *Cache) processItems() {
	for item := range c.setBuf {
		victims, added := c.policy.Add(item.key, item.cost)
		if added {
			// item was accepted by the policy, so add to the hashmap
			c.store.Set(item.key, item.val)
		}
		// delete victims that are no longer worthy of being in the cache
		for _, victim := range victims {
			// eviction callback
			if c.onEvict != nil {
				victim.val, _ = c.store.Get(victim.key)
				c.onEvict(victim.key, victim.val, victim.cost)
			}
			// delete from hashmap
			c.store.Del(victim.key)
		}
	}
}

func (c *Cache) collectMetrics() {
	c.stats = newMetrics()
	c.policy.CollectMetrics(c.stats)
}

// Metrics returns statistics about cache performance.
func (c *Cache) Metrics() *metrics {
	return c.stats
}

type metricType int

const (
	// The following 2 keep track of hits and misses.
	hit = iota
	miss

	// The following 3 keep track of number of keys added, updated and evicted.
	keyAdd
	keyUpdate
	keyEvict

	// The following 2 keep track of cost of keys added and evicted.
	costAdd
	costEvict

	// The following keep track of how many sets were dropped or rejected later.
	dropSets
	rejectSets

	// The following 2 keep track of how many gets were kept and dropped on the floor.
	dropGets
	keepGets

	// This should be the final enum. Other enums should be set before this.
	doNotUse
)

func stringFor(t metricType) string {
	switch t {
	case hit:
		return "hit"
	case miss:
		return "miss"
	case keyAdd:
		return "keys-added"
	case keyUpdate:
		return "keys-updated"
	case keyEvict:
		return "keys-evicted"
	case costAdd:
		return "cost-added"
	case costEvict:
		return "cost-evicted"
	case dropSets:
		return "sets-dropped"
	case rejectSets:
		return "sets-rejected" // by policy.
	case dropGets:
		return "gets-dropped"
	case keepGets:
		return "gets-kept"
	default:
		return "unidentified"
	}
}

// metrics is the struct for hit ratio statistics. Note that there is some
// cost to maintaining the counters, so it's best to wrap Policies via the
// Recorder type when hit ratio analysis is needed.
type metrics struct {
	all [doNotUse][]*uint64
}

func newMetrics() *metrics {
	s := &metrics{}
	for i := 0; i < doNotUse; i++ {
		s.all[i] = make([]*uint64, 256)
		slice := s.all[i]
		for j := range slice {
			slice[j] = new(uint64)
		}
	}
	return s
}

func (p *metrics) Add(t metricType, hash, delta uint64) {
	if p == nil {
		return
	}
	valp := p.all[t]
	// Avoid false sharing by padding at least 64 bytes of space between two
	// atomic counters which would be incremented.
	idx := (hash % 25) * 10
	atomic.AddUint64(valp[idx], delta)
}

func (p *metrics) Get(t metricType) uint64 {
	if p == nil {
		return 0
	}
	valp := p.all[t]
	var total uint64
	for i := range valp {
		total += atomic.LoadUint64(valp[i])
	}
	return total
}

func (p *metrics) Ratio() float64 {
	if p == nil {
		return 0.0
	}
	hits, misses := p.Get(hit), p.Get(miss)
	if hits == 0 && misses == 0 {
		return 0.0
	}
	return float64(hits) / float64(hits+misses)
}

func (p *metrics) String() string {
	if p == nil {
		return ""
	}
	var buf bytes.Buffer
	for i := 0; i < doNotUse; i++ {
		t := metricType(i)
		fmt.Fprintf(&buf, "%s: %d ", stringFor(t), p.Get(t))
	}
	fmt.Fprintf(&buf, "gets-total: %d ", p.Get(hit)+p.Get(miss))
	fmt.Fprintf(&buf, "hit-ratio: %.2f", p.Ratio())
	return buf.String()
}
