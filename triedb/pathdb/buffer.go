// Copyright 2022 The go-ethereum Authors
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

package pathdb

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// bufferSet groups the modified trie nodes and flat states aggregated from
// one or more contiguous state transitions.
type bufferSet struct {
	level  int          // compaction level: 0 for a single transition, a merge yields the max level plus one
	layers uint64       // the number of state transitions aggregated inside
	nodes  *nodeSet     // aggregated trie node set
	states *stateSet    // aggregated state set
	bloom  *bloomFilter // filter over the keys of the set, nil to probe the set always

	// global reports whether the keys of the set are inserted into the buffer-wide
	// filter, which happens in background in the order of commit, thus the sets
	// covered by the buffer-wide filter are always the oldest ones.
	global atomic.Bool
}

// newBufferSet constructs the set with the provided nodes and states, along
// with the filter over their bloom keys.
func newBufferSet(nodes *nodeSet, states *stateSet, layers uint64, bloom *bloomFilter) *bufferSet {
	// Don't panic for the lazy callers, initialize the nil set instead
	if nodes == nil {
		nodes = newNodeSet(nil)
	}
	if states == nil {
		states = newStates(nil, nil)
	}
	level := 0
	if layers > 1 {
		level = 1
	}
	return &bufferSet{
		level:  level,
		layers: layers,
		nodes:  nodes,
		states: states,
		bloom:  bloom,
	}
}

// size returns the approximate memory size of the held content.
func (s *bufferSet) size() uint64 {
	return s.nodes.size + s.states.size
}

// mergeBufferSets aggregates the provided sets (ordered from oldest to newest)
// into a single set. The resulting set assumes ownership of freshly allocated
// internal maps, leaving the source sets unmodified.
//
// The provided sets must be all covered by the buffer-wide filter, ensured by
// the callers (see compactRun and flatten).
func mergeBufferSets(sets []*bufferSet) *bufferSet {
	var (
		level  int
		layers uint64
		nodes  = make([]*nodeSet, 0, len(sets))
		states = make([]*stateSet, 0, len(sets))
	)
	for _, set := range sets {
		layers += set.layers
		level = max(level, set.level)
		nodes = append(nodes, set.nodes)
		states = append(states, set.states)
	}
	merged := &bufferSet{
		level:  level + 1,
		layers: layers,
		nodes:  mergeNodeSets(nodes),
		states: mergeStateSets(states),
	}
	merged.bloom = newSetBloom(appendBloomKeys(nil, merged.nodes, merged.states))
	merged.global.Store(true)
	return merged
}

// bufferView is an immutable snapshot of the buffer content. Any mutation on
// the buffer results in a new view being published atomically, therefore, the
// view can be accessed concurrently without any lock protection.
type bufferView struct {
	layers uint64       // the total number of state transitions aggregated inside
	size   uint64       // approximate memory size of the aggregated sets
	sets   []*bufferSet // aggregated sets, sorted from oldest to newest

	// bloom is the filter guarding the aggregated sets, short-circuiting the
	// linear scan over them for the lookups of the non-contained state entries
	// (the vast majority). The filter is shared between the consecutive views
	// and strictly additive. The keys of a set are inserted in background after
	// its publication, thus the filter only guards the sets marked as global,
	// while the others are probed by their own filters.
	bloom *bloomFilter

	// lists caches the sorted state key lists across the aggregated sets, shared
	// by the views holding the same content (e.g. before and after compaction).
	lists *bufferLists
}

// bufferLists is the cache of the sorted state key lists of a buffer view.
type bufferLists struct {
	accountOnce sync.Once
	accounts    []common.Hash

	lock     sync.RWMutex
	storages map[common.Hash][]common.Hash
}

// markBufferHit records a lookup resolved by the set at the given distance
// from the newest one.
func markBufferHit(depth int) {
	bufferHitMeter.Mark(1)
	bufferHitDepthMeter.Mark(int64(depth))
}

// find resolves the entry with the given bloom key by probing the sets from the
// newest one, with get retrieving the entry from a set. The oldest sets covered
// by the buffer-wide filter are ruled out by it as a whole, while the newest ones
// not covered yet are probed one by one.
func find[T any](v *bufferView, key uint64, get func(*bufferSet) (T, bool)) (T, bool) {
	var (
		zero    T
		covered = len(v.sets)
	)
	for covered > 0 && !v.sets[covered-1].global.Load() {
		covered--
	}
	if covered == len(v.sets) && !v.bloom.contains(key) {
		bloomShortcircuitMeter.Mark(1)
		return zero, false
	}
	for i := len(v.sets) - 1; i >= 0; i-- {
		if i == covered-1 && covered < len(v.sets) && !v.bloom.contains(key) {
			bloomShortcircuitMeter.Mark(1)
			return zero, false
		}
		if !v.sets[i].bloom.contains(key) {
			continue
		}
		if val, ok := get(v.sets[i]); ok {
			markBufferHit(len(v.sets) - 1 - i)
			return val, true
		}
		setBloomFalsePositiveMeter.Mark(1)
	}
	bloomFalsePositiveMeter.Mark(1)
	return zero, false
}

// node retrieves the trie node with node path and its trie identifier.
func (v *bufferView) node(owner common.Hash, path []byte) (*trienode.Node, bool) {
	return find(v, bloomNodeKey(owner, path), func(set *bufferSet) (*trienode.Node, bool) {
		return set.nodes.node(owner, path)
	})
}

// account retrieves the account blob with account address hash.
func (v *bufferView) account(hash common.Hash) ([]byte, bool) {
	return find(v, bloomAccountKey(hash), func(set *bufferSet) ([]byte, bool) {
		return set.states.account(hash)
	})
}

// storage retrieves the storage slot with account address hash and slot key hash.
func (v *bufferView) storage(addrHash common.Hash, storageHash common.Hash) ([]byte, bool) {
	return find(v, bloomStorageKey(addrHash, storageHash), func(set *bufferSet) ([]byte, bool) {
		return set.states.storage(addrHash, storageHash)
	})
}

// compactThreshold returns the number of contiguous sets with the same level
// required to trigger a background compaction.
func compactThreshold(level int) int {
	switch level {
	case 0:
		return 8
	case 1:
		return 8
	default:
		return 4
	}
}

const (
	// minCompactSize is the lower bound of the compaction cap, preventing the
	// tiny buffers from being split into excessively small sets.
	minCompactSize = 2 * 1024 * 1024

	// maxCompactSize is the upper bound of the compaction cap, bounding the
	// latency of a single merge regardless of the buffer size. A larger buffer
	// holds more sets instead, each of which only costs a bloom probe on the
	// lookups.
	maxCompactSize = 64 * 1024 * 1024
)

// compactCap returns the maximum size of a set produced by the compaction for
// a buffer with the given memory allowance. The per-set bloom filters keep the
// lookups cheap across many sets, so the compaction only folds the small sets
// together and never builds a large set, whose merge would contend with the
// block processing.
func compactCap(limit uint64) uint64 {
	return min(max(limit/4, minCompactSize), maxCompactSize)
}

// sealed reports whether the set is too large to take part in the compaction:
// two sealed sets would merge into one above the cap.
func (s *bufferSet) sealed(maxSize uint64) bool {
	return s.size() >= maxSize/2
}

// compactRun locates the first contiguous run of unsealed sets covered by the
// buffer-wide filter with the same level whose length reaches the compaction
// threshold, returning the boundaries of the longest prefix of it that merges
// within the cap. (-1, -1) is returned if no such run exists.
func compactRun(sets []*bufferSet, maxSize uint64) (int, int) {
	for i := 0; i < len(sets); {
		if sets[i].sealed(maxSize) || !sets[i].global.Load() {
			i++
			continue
		}
		j := i
		for j < len(sets) && sets[j].level == sets[i].level && !sets[j].sealed(maxSize) && sets[j].global.Load() {
			j++
		}
		if j-i >= compactThreshold(sets[i].level) {
			// Each unsealed set is below half the cap, so at least two of them
			// fit into a merge within the cap.
			var (
				k    = i
				size uint64
			)
			for k < j && size+sets[k].size() <= maxSize {
				size += sets[k].size()
				k++
			}
			return i, k
		}
		i = j
	}
	return -1, -1
}

// buffer is a collection of modified states along with the modified trie nodes.
// They are cached here to aggregate the disk write. The content of the buffer
// must be checked before diving into disk (since it basically is not yet written
// data).
//
// Internally the buffer content is organized as a list of immutable sets, one
// per aggregated state transition, published atomically as a view. Committing
// a state transition is therefore a cheap append; the accumulated small sets
// are folded together by a background compaction thread to keep the read
// amplification bounded. Readers always resolve the state through the atomic
// view without any lock, thus they are never blocked by the compaction.
type buffer struct {
	limit  uint64                     // the maximum memory allowance in bytes
	view   atomic.Pointer[bufferView] // the current content snapshot
	viewMu sync.Mutex                 // serializes the view publications

	compactMu   sync.Mutex // protects the compaction states below
	compactCond *sync.Cond // signaled when the compaction thread terminates
	compacting  bool       // whether the background compaction thread is active
	paused      int        // the number of pending compaction pause requests
	global      *taskQueue // inserting the keys of the committed sets into the buffer-wide filter

	// done is the notifier whether the content in buffer has been flushed or not.
	// This channel is nil if the buffer is not frozen.
	done chan struct{}

	// flushErr memorizes the error if any exception occurs during flushing
	flushErr error
}

// newBuffer initializes the buffer with the provided states and trie nodes.
func newBuffer(limit int, nodes *nodeSet, states *stateSet, layers uint64) *buffer {
	b := &buffer{
		limit:  uint64(limit),
		global: newTaskQueue(),
	}
	b.compactCond = sync.NewCond(&b.compactMu)

	var (
		size  uint64
		sets  []*bufferSet
		bloom = newBufferBloom(uint64(limit))
	)
	if nodes != nil || states != nil || layers != 0 {
		keys := appendBloomKeys(nil, nodes, states)
		set := newBufferSet(nodes, states, layers, newSetBloom(keys))
		for _, key := range keys {
			bloom.insert(key)
		}
		set.global.Store(true)
		if set.size() > 0 || layers > 0 {
			sets = []*bufferSet{set}
			size = set.size()
		}
	}
	b.view.Store(&bufferView{
		layers: layers,
		size:   size,
		sets:   sets,
		bloom:  bloom,
		lists:  new(bufferLists),
	})
	return b
}

// account retrieves the account blob with account address hash.
func (b *buffer) account(hash common.Hash) ([]byte, bool) {
	return b.view.Load().account(hash)
}

// storage retrieves the storage slot with account address hash and slot key hash.
func (b *buffer) storage(addrHash common.Hash, storageHash common.Hash) ([]byte, bool) {
	return b.view.Load().storage(addrHash, storageHash)
}

// node retrieves the trie node with node path and its trie identifier.
func (b *buffer) node(owner common.Hash, path []byte) (*trienode.Node, bool) {
	return b.view.Load().node(owner, path)
}

// mustAccount retrieves the account blob with account address hash, returning
// an error if the account is not found in the buffer.
func (b *buffer) mustAccount(hash common.Hash) ([]byte, error) {
	if blob, ok := b.account(hash); ok {
		return blob, nil
	}
	return nil, fmt.Errorf("account is not found, %x", hash)
}

// mustStorage retrieves the storage slot with account address hash and slot
// key hash, returning an error if the slot is not found in the buffer.
func (b *buffer) mustStorage(addrHash common.Hash, storageHash common.Hash) ([]byte, error) {
	if blob, ok := b.storage(addrHash, storageHash); ok {
		return blob, nil
	}
	return nil, fmt.Errorf("storage slot is not found, %x %x", addrHash, storageHash)
}

// accountList returns a sorted list of all accounts cached in the buffer,
// including the deleted ones.
//
// Note, the returned slice is not a copy, so do not modify it.
func (b *buffer) accountList() []common.Hash {
	view := b.view.Load()
	switch len(view.sets) {
	case 0:
		return nil
	case 1:
		return view.sets[0].states.accountList()
	}
	view.lists.accountOnce.Do(func() {
		var combined []common.Hash
		for _, set := range view.sets {
			combined = append(combined, set.states.accountList()...)
		}
		slices.SortFunc(combined, common.Hash.Cmp)
		view.lists.accounts = slices.Compact(combined)
	})
	return view.lists.accounts
}

// storageList returns a sorted list of all storage slot hashes cached in the
// buffer for the given account. Nil is returned if the account is unknown to
// the buffer.
//
// Note, the returned slice is not a copy, so do not modify it.
func (b *buffer) storageList(accountHash common.Hash) []common.Hash {
	view := b.view.Load()
	view.lists.lock.RLock()
	list, ok := view.lists.storages[accountHash]
	view.lists.lock.RUnlock()
	if ok {
		return list
	}
	var (
		found    int
		last     []common.Hash
		combined []common.Hash
	)
	for _, set := range view.sets {
		list := set.states.storageList(accountHash)
		if list == nil {
			continue
		}
		found += 1
		last = list
		combined = append(combined, list...)
	}
	if found == 0 {
		return nil
	}
	if found == 1 {
		return last
	}
	slices.SortFunc(combined, common.Hash.Cmp)
	list = slices.Compact(combined)

	view.lists.lock.Lock()
	if view.lists.storages == nil {
		view.lists.storages = make(map[common.Hash][]common.Hash)
	}
	view.lists.storages[accountHash] = list
	view.lists.lock.Unlock()
	return list
}

// commit adds the provided states and trie nodes into the buffer as a new set.
// It's a cheap append without merging the data into the existing sets; folding
// the accumulated sets together is performed by the background compaction
// thread.
//
// The ownership of the provided sets still belongs to the caller (typically
// the bottom-most diff layer), thus they are regarded as immutable and shall
// never be mutated by the buffer.
func (b *buffer) commit(nodes *nodeSet, states *stateSet) *buffer {
	keys := deriveBloomKeys(nodes, states)
	return b.commitWithBloom(nodes, states, newSetBloom(keys...), keys)
}

// commitWithBloom is the variant of commit taking the bloom keys of the sets
// along with the filter over them, e.g. the ones of a diff layer.
func (b *buffer) commitWithBloom(nodes *nodeSet, states *stateSet, bloom *bloomFilter, keys [][]uint64) *buffer {
	start := time.Now()

	b.viewMu.Lock()
	view := b.view.Load()
	set := newBufferSet(nodes, states, 1, bloom)

	// Construct the set list for the new view. Note the list held by the
	// current view must not be modified in place (append could mutate the
	// shared backing array which might still be referenced by the readers
	// of previous views), clone it instead.
	sets := make([]*bufferSet, 0, len(view.sets)+1)
	sets = append(sets, view.sets...)
	sets = append(sets, set)

	next := &bufferView{
		layers: view.layers + 1,
		size:   view.size + set.size(),
		sets:   sets,
		bloom:  view.bloom,
		lists:  new(bufferLists),
	}
	b.view.Store(next)
	b.viewMu.Unlock()

	// Insert the keys of the new set into the buffer-wide filter in background,
	// the set is probed by its own filter until then.
	b.global.schedule(func() {
		for _, part := range keys {
			for _, key := range part {
				view.bloom.insertAtomic(key)
			}
		}
		set.global.Store(true)

		// The set becomes compactable once covered by the buffer-wide filter
		b.tryCompact()
	})
	bufferAppendTimer.UpdateSince(start)
	bufferAppendBytesMeter.Mark(int64(set.size()))
	bufferSetsGauge.Update(int64(len(next.sets)))
	bufferSizeGauge.Update(int64(next.size))
	bufferLayersGauge.Update(int64(next.layers))

	b.tryCompact()
	return b
}

// revertTo is the reverse operation of commit. It also merges the provided states
// and trie nodes into the buffer. The key difference is that the provided state
// set should reverse the changes made by the most recent state transition.
func (b *buffer) revertTo(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node, accounts map[common.Hash][]byte, storages map[common.Hash]map[common.Hash][]byte) error {
	// Block until the background compaction is terminated, ensuring the
	// exclusive access of the aggregated sets.
	b.pauseCompaction()
	defer b.resumeCompaction()

	b.viewMu.Lock()
	defer b.viewMu.Unlock()

	view := b.view.Load()

	// Short circuit if no embedded state transition to revert
	if view.layers == 0 {
		return errStateUnrecoverable
	}
	// Reset the entire buffer if only a single transition left
	if view.layers == 1 {
		b.view.Store(&bufferView{
			bloom: newBufferBloom(b.limit),
			lists: new(bufferLists),
		})
		return nil
	}
	var (
		last = view.sets[len(view.sets)-1]
		sets []*bufferSet
		size uint64
	)
	if last.layers == 1 {
		// The most recent transition is aggregated in its own set, simply
		// drop the set to reverse the transition.
		sets = view.sets[:len(view.sets)-1]
	} else {
		// The most recent transition was already folded together with other
		// transitions, revert the changes upon the aggregated set directly.
		// It's safe as the exclusive access to the buffer is guaranteed by
		// the caller (the disk layer) in the revert scenario.
		last.nodes.revertTo(db, nodes)
		last.states.revertTo(accounts, storages)
		last.layers -= 1
		sets = view.sets
	}
	for _, set := range sets {
		size += set.size()
	}
	// Note the bloom filter is reused as is: the keys of the reverted
	// transition are left inside as stale entries, which only slightly
	// increases the false positive rate (harmless for correctness) and
	// avoids an expensive rebuild for the rare revert operation.
	b.view.Store(&bufferView{
		layers: view.layers - 1,
		size:   size,
		sets:   sets,
		bloom:  view.bloom,
		lists:  new(bufferLists),
	})
	return nil
}

// empty returns an indicator if buffer is empty.
func (b *buffer) empty() bool {
	return b.layers() == 0
}

// layers returns the number of state transitions aggregated in the buffer.
func (b *buffer) layers() uint64 {
	return b.view.Load().layers
}

// full returns an indicator if the size of accumulated content exceeds the
// configured threshold.
func (b *buffer) full() bool {
	return b.size() > b.limit
}

// size returns the approximate memory size of the held content. Note the
// returned size might be slightly overestimated, as the overlapped entries
// between the aggregated sets are counted repeatedly before being folded
// together by the compaction.
func (b *buffer) size() uint64 {
	return b.view.Load().size
}

// pauseCompaction suspends the background compaction and blocks until the
// active compaction thread is terminated. The caller must invoke the paired
// resumeCompaction afterwards.
func (b *buffer) pauseCompaction() {
	b.compactMu.Lock()
	b.paused += 1
	for b.compacting {
		b.compactCond.Wait()
	}
	b.compactMu.Unlock()
}

// resumeCompaction lifts the compaction suspension and reschedules the
// compaction if it's necessary.
func (b *buffer) resumeCompaction() {
	b.compactMu.Lock()
	b.paused -= 1
	b.compactMu.Unlock()

	b.tryCompact()
}

// tryCompact schedules a background compaction if enough sets have been
// accumulated in the buffer.
func (b *buffer) tryCompact() {
	b.compactMu.Lock()
	defer b.compactMu.Unlock()

	if b.compacting || b.paused > 0 {
		return
	}
	if i, _ := compactRun(b.view.Load().sets, compactCap(b.limit)); i < 0 {
		return
	}
	b.compacting = true
	go b.compact()
}

// compact iteratively folds the accumulated sets together to keep the read
// amplification of the buffer bounded, terminating if no more compactable
// run exists or the compaction is suspended.
func (b *buffer) compact() {
	for {
		b.compactMu.Lock()
		if b.paused > 0 {
			b.compacting = false
			b.compactCond.Broadcast()
			b.compactMu.Unlock()
			return
		}
		b.compactMu.Unlock()

		view := b.view.Load()
		i, j := compactRun(view.sets, compactCap(b.limit))
		if i < 0 {
			b.compactMu.Lock()
			// Recheck the compaction condition with the lock held, a new set
			// might be committed after the view was loaded, whose scheduling
			// attempt was rejected due to the active compaction flag.
			if b.paused == 0 {
				if k, _ := compactRun(b.view.Load().sets, compactCap(b.limit)); k >= 0 {
					b.compactMu.Unlock()
					continue
				}
			}
			b.compacting = false
			b.compactCond.Broadcast()
			b.compactMu.Unlock()
			return
		}
		var (
			start  = time.Now()
			merged = mergeBufferSets(view.sets[i:j])
		)
		// Publish the folded set by replacing the constituents in the current
		// view. Note the view might have been extended with new sets in the
		// meantime, while the position of the merged sets remains unchanged.
		b.viewMu.Lock()
		cur := b.view.Load()
		sets := make([]*bufferSet, 0, len(cur.sets)-(j-i)+1)
		sets = append(sets, cur.sets[:i]...)
		sets = append(sets, merged)
		sets = append(sets, cur.sets[j:]...)

		var size uint64
		for _, set := range sets {
			size += set.size()
		}
		// The bloom filter and the key lists are untouched by the compaction.
		b.view.Store(&bufferView{
			layers: cur.layers,
			size:   size,
			sets:   sets,
			bloom:  cur.bloom,
			lists:  cur.lists,
		})
		b.viewMu.Unlock()

		var input uint64
		for _, set := range view.sets[i:j] {
			input += set.size()
		}
		compactTimeTimer.UpdateSince(start)
		compactInputMeter.Mark(int64(input))
		compactOutputMeter.Mark(int64(merged.size()))
		log.Debug("Compacted buffer sets", "count", j-i, "level", merged.level, "layers", merged.layers, "size", common.StorageSize(merged.size()), "elapsed", common.PrettyDuration(time.Since(start)))
	}
}

// flatten collapses all the aggregated sets into a single one and returns the
// combined trie nodes and flat states. The flattened set is also published as
// the current view, reducing the lookup cost for the subsequent state access.
func (b *buffer) flatten() (*nodeSet, *stateSet) {
	// Block until the background compaction is terminated, ensuring the
	// stability of the set list, and the sets are covered by the buffer-wide
	// filter, which then covers the flattened set.
	b.pauseCompaction()
	defer b.resumeCompaction()
	b.global.wait()

	b.viewMu.Lock()
	defer b.viewMu.Unlock()

	view := b.view.Load()
	switch len(view.sets) {
	case 0:
		return newNodeSet(nil), newStates(nil, nil)
	case 1:
		return view.sets[0].nodes, view.sets[0].states
	}
	merged := mergeBufferSets(view.sets)
	b.view.Store(&bufferView{
		layers: view.layers,
		size:   merged.size(),
		sets:   []*bufferSet{merged},
		bloom:  view.bloom,
		lists:  view.lists,
	})
	return merged.nodes, merged.states
}

// flush persists the in-memory dirty trie node into the disk if the configured
// memory threshold is reached. Note, all data must be written atomically.
func (b *buffer) flush(root common.Hash, db ethdb.KeyValueStore, syncHistory func() error, progress []byte, nodesCache, statesCache *fastcache.Cache, id uint64, postFlush func()) {
	if b.done != nil {
		panic("duplicated flush operation")
	}
	b.done = make(chan struct{}) // allocate the channel for notification

	// Schedule the background thread to construct the batch, which usually
	// take a few seconds.
	go func() {
		defer func() {
			if postFlush != nil {
				postFlush()
			}
			close(b.done)
		}()

		// Ensure the target state id is aligned with the internal counter.
		head := rawdb.ReadPersistentStateID(db)
		if head+b.layers() != id {
			b.flushErr = fmt.Errorf("buffer layers (%d) cannot be applied on top of persisted state id (%d) to reach requested state id (%d)", b.layers(), head, id)
			return
		}

		// Collapse the aggregated sets into a single one for flushing. The
		// buffer content is still fully available for state access during
		// the flattening.
		var (
			accounted    = b.size()
			flattenStart = time.Now()
		)
		nodes, states := b.flatten()
		commitFlattenTimer.UpdateSince(flattenStart)
		commitLayersMeter.Mark(int64(b.layers()))
		commitAccountedMeter.Mark(int64(accounted))
		commitFlattenedMeter.Mark(int64(nodes.size + states.size))

		// Terminate the state snapshot generation if it's active
		var (
			start = time.Now()
			batch = db.NewBatchWithSize((nodes.dbsize() + states.dbsize()) * 11 / 10) // extra 10% for potential pebble internal stuff
		)
		// Wait for the scheduled history writes and explicitly sync the history
		// freezers to ensure all written data is persisted to disk before
		// updating the key-value store.
		//
		// This step is crucial to guarantee that the corresponding state history remains
		// available for state rollback.
		if err := syncHistory(); err != nil {
			b.flushErr = err
			return
		}
		nodeCount := nodes.write(batch, nodesCache)
		accounts, slots := states.write(batch, progress, statesCache)
		rawdb.WritePersistentStateID(batch, id)
		rawdb.WriteSnapshotRoot(batch, root)

		// Flush all mutations in a single batch
		size := batch.ValueSize()
		if err := batch.Write(); err != nil {
			b.flushErr = err
			return
		}
		batch.Close()

		commitBytesMeter.Mark(int64(size))
		commitNodesMeter.Mark(int64(nodeCount))
		commitAccountsMeter.Mark(int64(accounts))
		commitStoragesMeter.Mark(int64(slots))
		commitTimeTimer.UpdateSince(start)

		// Release the persisted content, served by the clean caches and the disk
		// from now on. The items written above are held by the buffer until the
		// batch is committed, so no stale value can be cached by the readers.
		b.viewMu.Lock()
		b.view.Store(&bufferView{
			bloom: newBloomFilter(0), // empty filter, short-circuiting all the reads
			lists: new(bufferLists),
		})
		b.viewMu.Unlock()

		log.Debug("Persisted buffer content", "nodes", nodeCount, "accounts", accounts, "slots", slots, "bytes", common.StorageSize(size), "elapsed", common.PrettyDuration(time.Since(start)))
	}()
}

// waitFlush blocks until the buffer has been fully flushed and returns any
// stored errors that occurred during the process.
func (b *buffer) waitFlush() error {
	if b.done == nil {
		return errors.New("the buffer is not frozen")
	}
	<-b.done
	return b.flushErr
}
