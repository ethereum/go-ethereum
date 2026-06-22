// Copyright 2024 The go-ethereum Authors
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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// lookupTaskQueueCap is the buffer size of the channel feeding the background
// lookup builder. Steady state holds at most a couple of pending tasks (one
// add and one remove per block); the generous buffer simply absorbs bursts so
// the producer (block import) is virtually never blocked.
const lookupTaskQueueCap = 1024

// storageKey returns a key for uniquely identifying the storage slot.
func storageKey(accountHash common.Hash, slotHash common.Hash) [64]byte {
	var key [64]byte
	copy(key[:32], accountHash[:])
	copy(key[32:], slotHash[:])
	return key
}

// storageKeySlice returns a key for uniquely identifying the storage slot in
// the slice format.
func storageKeySlice(accountHash common.Hash, slotHash common.Hash) []byte {
	key := storageKey(accountHash, slotHash)
	return key[:]
}

// layerList records the set of diff-layer roots that modified a single state
// entry, ordered from oldest to newest.
//
// The overwhelmingly common case is a key touched by exactly one in-memory
// diff layer; that single root is stored inline in head with rest left nil,
// so no backing slice is allocated and the value lives entirely inside the
// map bucket. Only the rare hot keys (e.g. fee recipient, popular contracts)
// modified across multiple layers spill into rest.
//
// Note head may legitimately be the zero hash (e.g. the empty bintrie root),
// so callers must rely on the map's presence flag rather than comparing head
// against common.Hash{} to detect an absent entry.
type layerList struct {
	head common.Hash   // the oldest layer; the only one in the common case
	rest []common.Hash // additional layers (newest at the tail), nil when len==1
}

// lookupTask is a unit of work consumed by the background lookup builder. It
// either integrates a diff layer into the index (diff set, remove=false),
// unlinks it (diff set, remove=true), or acts as a barrier that the builder
// acknowledges by closing done once all prior tasks are processed.
type lookupTask struct {
	diff   *diffLayer
	remove bool
	done   chan struct{}
}

// lookup is an internal structure used to efficiently determine the layer in
// which a state entry resides.
//
// The index is maintained asynchronously: callers enqueue diff layers via
// addLayer/removeLayer, and a single background goroutine integrates them into
// the maps below. This keeps the (relatively expensive) index maintenance off
// the block-import critical path. Reads consult the index only for layers that
// have already been incorporated (tracked in `indexed`); a read targeting a
// layer the builder hasn't reached yet transparently falls back to a direct
// parent-chain traversal, which is always correct.
type lookup struct {
	// lock guards the maps and the indexed set against concurrent access by
	// the background builder (writer) and the read path (readers).
	lock sync.RWMutex

	// accounts represents the mutation history for specific accounts.
	// The key is the account address hash, and the value is the list
	// of **diff layer** IDs indicating where the account was modified,
	// with the order from oldest to newest.
	accounts map[common.Hash]layerList

	// storages represents the mutation history for specific storage
	// slot. The key is the account address hash and the storage key
	// hash, the value is the list of **diff layer** IDs indicating
	// where the slot was modified, with the order from oldest to newest.
	storages map[[64]byte]layerList

	// indexed holds the roots of the diff layers already integrated into the
	// maps above. Because layers are always enqueued (and thus processed) in
	// child-after-parent order, the presence of a layer here implies all of
	// its ancestors are present too.
	indexed map[common.Hash]struct{}

	// descendant is the callback indicating whether the layer with
	// given root is a descendant of the one specified by `ancestor`.
	descendant func(state common.Hash, ancestor common.Hash) bool

	// Background builder plumbing.
	tasks     chan *lookupTask
	quit      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// newLookup initializes the lookup structure and launches the background
// builder. The layers already present below the head are integrated
// synchronously, so the index is fully populated by the time this returns.
func newLookup(head layer, descendant func(state common.Hash, ancestor common.Hash) bool) *lookup {
	var (
		current = head
		layers  []layer
	)
	for current != nil {
		layers = append(layers, current)
		current = current.parentLayer()
	}
	l := &lookup{
		accounts:   make(map[common.Hash]layerList),
		storages:   make(map[[64]byte]layerList),
		indexed:    make(map[common.Hash]struct{}),
		descendant: descendant,
		tasks:      make(chan *lookupTask, lookupTaskQueueCap),
		quit:       make(chan struct{}),
	}
	// Apply the diff layers from bottom to top. The builder isn't running yet,
	// so this can mutate the maps directly without contending for the lock.
	for i := len(layers) - 1; i >= 0; i-- {
		if diff, ok := layers[i].(*diffLayer); ok {
			l.applyAdd(diff)
		}
	}
	l.wg.Add(1)
	go l.loop()
	return l
}

// loop is the background builder, integrating diff layers into the index in
// the order they are enqueued until the lookup is closed.
func (l *lookup) loop() {
	defer l.wg.Done()

	for {
		select {
		case <-l.quit:
			return
		case task := <-l.tasks:
			if task.diff != nil {
				if task.remove {
					l.applyRemove(task.diff)
				} else {
					l.applyAdd(task.diff)
				}
			}
			// Acknowledge the barrier, if any, after all preceding tasks (this
			// one included) have been applied.
			if task.done != nil {
				close(task.done)
			}
		}
	}
}

// waitBuild blocks until the background builder has processed every task
// enqueued before this call. Because the builder is strictly FIFO, observing
// the barrier guarantees all prior add/remove tasks have been applied.
func (l *lookup) waitBuild() {
	done := make(chan struct{})
	select {
	case l.tasks <- &lookupTask{done: done}:
	case <-l.quit:
		return
	}
	select {
	case <-done:
	case <-l.quit:
	}
}

// close terminates the background builder and waits for it to exit. Any tasks
// still queued are discarded; this is only called when the whole index is
// about to be replaced or the database is shutting down.
func (l *lookup) close() {
	l.closeOnce.Do(func() {
		close(l.quit)
	})
	l.wg.Wait()
}

// isIndexed reports whether reads for the given state can be served from the
// index. The disk layer (base) is always considered indexed since it carries
// no diff data of its own.
//
// This method assumes the read lock has been held.
func (l *lookup) isIndexed(state common.Hash, base common.Hash) bool {
	if state == base {
		return true
	}
	_, ok := l.indexed[state]
	return ok
}

// tip scans the layer list from newest to oldest and returns the first entry
// that either matches the supplied stateID or is a descendant of it. It returns
// (common.Hash{}, false) when no such entry exists.
func (list layerList) tip(stateID common.Hash, descendant func(state common.Hash, ancestor common.Hash) bool) (common.Hash, bool) {
	for i := len(list.rest) - 1; i >= 0; i-- {
		if list.rest[i] == stateID || descendant(stateID, list.rest[i]) {
			return list.rest[i], true
		}
	}
	if list.head == stateID || descendant(stateID, list.head) {
		return list.head, true
	}
	return common.Hash{}, false
}

// add appends the given layer root as the newest entry of the list.
func (list layerList) add(state common.Hash) layerList {
	list.rest = append(list.rest, state)
	return list
}

// remove unlinks the given layer root from the list. It returns the updated
// list, a flag indicating whether the list became empty, and a flag indicating
// whether the element was found and removed.
func (list layerList) remove(state common.Hash) (layerList, bool, bool) {
	// The newest layers are flattened into the disk layer from the bottom up,
	// so removals almost always target the oldest entry held in head.
	if list.head == state {
		if len(list.rest) == 0 {
			return list, true, true
		}
		list.head, list.rest = list.rest[0], list.rest[1:]
		// Release the backing array if it has grown excessively, otherwise the
		// re-slicing above would pin the whole array and leak memory.
		if cap(list.rest) > 1024 {
			list.rest = append(make([]common.Hash, 0, len(list.rest)), list.rest...)
		}
		return list, false, true
	}
	for i := 0; i < len(list.rest); i++ {
		if list.rest[i] == state {
			list.rest = append(list.rest[:i], list.rest[i+1:]...)
			return list, false, true
		}
	}
	return list, false, false
}

// accountTip traverses the layer list associated with the given account in
// reverse order to locate the first entry that either matches the specified
// stateID or is a descendant of it.
//
// If found, the account data corresponding to the supplied stateID resides
// in the layer identified by the returned hash (ok=true). Otherwise,
// (common.Hash{}, false) is returned to signal that the supplied stateID is
// stale.
//
// Note the returned hash may itself be common.Hash{} when the disk layer's
// root is zero — as is the case for a fresh verkle/bintrie database whose
// empty trie hashes to EmptyVerkleHash. Callers must therefore consult the
// boolean rather than comparing the returned hash against common.Hash{}
// directly.
//
// The third return value reports whether the requested stateID has been
// incorporated into the index. When false, the (hash, ok) results are
// meaningless and the caller must fall back to a direct traversal.
func (l *lookup) accountTip(accountHash common.Hash, stateID common.Hash, base common.Hash) (common.Hash, bool, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	// Bail out if the builder hasn't reached this layer yet; the maps may be
	// missing recent mutations and would resolve to a stale layer.
	if !l.isIndexed(stateID, base) {
		return common.Hash{}, false, false
	}
	// Traverse the mutation history from latest to oldest one. Several
	// scenarios are possible:
	//
	// Chain:
	//     D->C1->C2->C3->C4 (HEAD)
	//      ->C1'->C2'->C3'
	// State:
	//     x: [C1, C1', C3', C3]
	//     y: []
	//
	// - (x, C4) => C3
	// - (x, C3) => C3
	// - (x, C2) => C1
	// - (x, C3') => C3'
	// - (x, C2') => C1'
	// - (y, C4) => D
	// - (y, C3') => D
	// - (y, C0) => null
	if list, ok := l.accounts[accountHash]; ok {
		if tip, found := list.tip(stateID, l.descendant); found {
			return tip, true, true
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base, true, true
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}, false, true
}

// storageTip traverses the layer list associated with the given account and
// slot hash in reverse order to locate the first entry that either matches
// the specified stateID or is a descendant of it.
//
// See accountTip for the returned-hash / ok convention — the same
// bintrie-zero-root caveat applies here.
func (l *lookup) storageTip(accountHash common.Hash, slotHash common.Hash, stateID common.Hash, base common.Hash) (common.Hash, bool, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	// Bail out if the builder hasn't reached this layer yet; the maps may be
	// missing recent mutations and would resolve to a stale layer.
	if !l.isIndexed(stateID, base) {
		return common.Hash{}, false, false
	}
	// Traverse the mutation history from latest to oldest, returning the most
	// recent layer that contains the requested data.
	if list, ok := l.storages[storageKey(accountHash, slotHash)]; ok {
		if tip, found := list.tip(stateID, l.descendant); found {
			return tip, true, true
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base, true, true
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}, false, true
}

// addLayer enqueues the diff layer to be integrated into the lookup index by
// the background builder. It returns immediately, keeping the index maintenance
// off the caller's critical path.
//
// Layers must be enqueued strictly in child-after-parent order so that the
// builder processes them bottom-to-top.
func (l *lookup) addLayer(diff *diffLayer) {
	select {
	case l.tasks <- &lookupTask{diff: diff}:
	case <-l.quit:
	}
}

// removeLayer enqueues the diff layer to be unlinked from the lookup index by
// the background builder. The error return is retained for call-site
// compatibility and is always nil; failures are reported asynchronously.
func (l *lookup) removeLayer(diff *diffLayer) error {
	select {
	case l.tasks <- &lookupTask{diff: diff, remove: true}:
	case <-l.quit:
	}
	return nil
}

// applyAdd traverses the state data retained in the specified diff layer and
// integrates it into the lookup set. It runs on the background builder.
//
// This function assumes that all layers older than the provided one have already
// been processed, ensuring that layers are processed strictly in a bottom-to-top
// order.
func (l *lookup) applyAdd(diff *diffLayer) {
	defer func(now time.Time) {
		lookupAddLayerTimer.UpdateSince(now)
	}(time.Now())

	l.lock.Lock()
	defer l.lock.Unlock()

	var (
		wg    sync.WaitGroup
		state = diff.rootHash()
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for accountHash := range diff.states.accountData {
			// The common case is a key touched by a single layer, where the
			// zero-value layerList already holds the root inline in head; only
			// hot keys modified across layers allocate a backing slice.
			list, exists := l.accounts[accountHash]
			if !exists {
				l.accounts[accountHash] = layerList{head: state}
				continue
			}
			l.accounts[accountHash] = list.add(state)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for accountHash, slots := range diff.states.storageData {
			for slotHash := range slots {
				key := storageKey(accountHash, slotHash)
				list, exists := l.storages[key]
				if !exists {
					l.storages[key] = layerList{head: state}
					continue
				}
				l.storages[key] = list.add(state)
			}
		}
	}()
	wg.Wait()
	l.indexed[state] = struct{}{}
}

// applyRemove traverses the state data retained in the specified diff layer and
// unlinks them from the lookup set. It runs on the background builder.
func (l *lookup) applyRemove(diff *diffLayer) {
	defer func(now time.Time) {
		lookupRemoveLayerTimer.UpdateSince(now)
	}(time.Now())

	l.lock.Lock()
	defer l.lock.Unlock()

	state := diff.rootHash()
	for accountHash := range diff.states.accountData {
		list, empty, found := l.accounts[accountHash].remove(state)
		if !found {
			log.Error("Account lookup is not found", "account", accountHash, "state", state)
			continue
		}
		if empty {
			delete(l.accounts, accountHash)
		} else {
			l.accounts[accountHash] = list
		}
	}
	for accountHash, slots := range diff.states.storageData {
		for slotHash := range slots {
			key := storageKey(accountHash, slotHash)
			list, empty, found := l.storages[key].remove(state)
			if !found {
				log.Error("Storage lookup is not found", "account", accountHash, "slot", slotHash, "state", state)
				continue
			}
			if empty {
				delete(l.storages, key)
			} else {
				l.storages[key] = list
			}
		}
	}
	delete(l.indexed, state)
}
