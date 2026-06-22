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

// mergeYieldThreshold is the number of entries the background merger integrates
// while holding the buffer lock before it releases and re-acquires the lock,
// giving any waiting frontend reads a chance to run. It is deliberately small:
// the merge is allowed to be slow, but it must not stall reads. With a
// writer-priority RWMutex the merger effectively owns the lock for the whole
// merge, so this threshold bounds the worst-case read stall (roughly
// mergeYieldThreshold map operations) rather than the total merge duration.
const mergeYieldThreshold = 32

// pendingMerge represents a merge operation which is handed to the buffer but
// not yet completed.
type pendingMerge struct {
	nodes  *nodeSet
	states *stateSet
	size   uint64
}

// buffer is a collection of modified states along with the modified trie nodes.
// They are cached here to aggregate the disk write. The content of the buffer
// must be checked before diving into disk (since it basically is not yet written
// data).
type buffer struct {
	layers uint64 // The number of diff layers aggregated inside
	limit  uint64 // The maximum memory allowance in bytes

	nodes       *nodeSet                        // Aggregated trie node set
	states      *stateSet                       // Aggregated state set
	pending     atomic.Pointer[[]*pendingMerge] // The list of pending merges
	pendingSize uint64                          // The size of pending merges
	merging     bool                            // Flag whether the background merger is running
	lock        sync.RWMutex                    // Guard the five fields above
	wg          sync.WaitGroup

	// done is the notifier whether the content in buffer has been flushed or not.
	// This channel is nil if the buffer is not frozen.
	done chan struct{}

	// flushErr memorizes the error if any exception occurs during flushing
	flushErr error
}

// newBuffer initializes the buffer with the provided states and trie nodes.
func newBuffer(limit int, nodes *nodeSet, states *stateSet, layers uint64) *buffer {
	// Don't panic for lazy users if any provided set is nil
	if nodes == nil {
		nodes = newNodeSet(nil)
	}
	if states == nil {
		states = newStates(nil, nil, false)
	}
	return &buffer{
		layers: layers,
		limit:  uint64(limit),
		nodes:  nodes,
		states: states,
	}
}

// loadPending returns the current immutable pending-overlay snapshot.
func (b *buffer) loadPending() []*pendingMerge {
	if p := b.pending.Load(); p != nil {
		return *p
	}
	return nil
}

// storePending sets the pending snapshot to the given list.
func (b *buffer) storePending(list []*pendingMerge) {
	b.pending.Store(&list)
}

// account retrieves the account blob with account address hash.
func (b *buffer) account(hash common.Hash) ([]byte, bool) {
	pending := b.loadPending()
	for i := len(pending) - 1; i >= 0; i-- {
		if blob, found := pending[i].states.account(hash); found {
			return blob, true
		}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.states.account(hash)
}

// storage retrieves the storage slot with account address hash and slot key hash.
func (b *buffer) storage(addrHash common.Hash, storageHash common.Hash) ([]byte, bool) {
	pending := b.loadPending()
	for i := len(pending) - 1; i >= 0; i-- {
		if blob, found := pending[i].states.storage(addrHash, storageHash); found {
			return blob, true
		}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.states.storage(addrHash, storageHash)
}

// node retrieves the trie node with node path and its trie identifier.
func (b *buffer) node(owner common.Hash, path []byte) (*trienode.Node, bool) {
	pending := b.loadPending()
	for i := len(pending) - 1; i >= 0; i-- {
		if n, found := pending[i].nodes.node(owner, path); found {
			return n, true
		}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.nodes.node(owner, path)
}

// accountList returns the sorted list of all accounts held by the buffer.
func (b *buffer) accountList() []common.Hash {
	b.lock.RLock()
	defer b.lock.RUnlock()

	pending := b.loadPending()
	if len(pending) == 0 {
		return b.states.accountList()
	}
	var (
		list []common.Hash
		seen = make(map[common.Hash]struct{})
	)
	add := func(hashes []common.Hash) {
		for _, h := range hashes {
			if _, ok := seen[h]; !ok {
				seen[h] = struct{}{}
				list = append(list, h)
			}
		}
	}
	add(b.states.accountList())
	for _, pf := range pending {
		add(pf.states.accountList())
	}
	slices.SortFunc(list, common.Hash.Cmp)
	return list
}

// storageList returns the sorted list of all storage slot hashes held by the
// buffer for the given account.
func (b *buffer) storageList(account common.Hash) []common.Hash {
	b.lock.RLock()
	defer b.lock.RUnlock()

	pending := b.loadPending()
	if len(pending) == 0 {
		return b.states.storageList(account)
	}
	var (
		list []common.Hash
		seen = make(map[common.Hash]struct{})
	)
	add := func(hashes []common.Hash) {
		for _, h := range hashes {
			if _, ok := seen[h]; !ok {
				seen[h] = struct{}{}
				list = append(list, h)
			}
		}
	}
	add(b.states.storageList(account))
	for _, m := range pending {
		add(m.states.storageList(account))
	}
	slices.SortFunc(list, common.Hash.Cmp)
	return list
}

// commit hands the provided states and trie nodes to the buffer as an immutable
// pending overlay and wakes the background folder. It returns immediately,
// keeping the (expensive) merge off the caller's critical path.
func (b *buffer) commit(nodes *nodeSet, states *stateSet) *buffer {
	m := &pendingMerge{
		nodes:  nodes,
		states: states,
		size:   nodes.size + states.size,
	}
	b.layers++

	b.lock.Lock()
	b.pendingSize += m.size
	b.storePending(append(slices.Clone(b.loadPending()), m))

	// Spawn an ephemeral merger if one isn't already running. It exits on its
	// own once it has drained the pending queue, so there is no goroutine to
	// stop/clean up later.
	if !b.merging {
		b.merging = true
		b.wg.Add(1)
		go b.merge()
	}
	b.lock.Unlock()

	return b
}

// merge integrates pending overlays into the aggregated maps until the queue is
// empty, then exits.
func (b *buffer) merge() {
	defer b.wg.Done()

	for {
		b.lock.Lock()
		cur := b.loadPending()
		if len(cur) == 0 {
			b.merging = false
			b.lock.Unlock()
			return
		}
		m := cur[0]

		// Periodically release the lock to unblock the frontend reads
		n := 0
		yield := func() {
			if n++; n >= mergeYieldThreshold {
				n = 0
				b.lock.Unlock()
				b.lock.Lock()
			}
		}
		b.nodes.merge(m.nodes, yield)
		b.states.merge(m.states, yield)

		// Drop the just-merged overlay from the front of the (possibly grown)
		// snapshot. Re-load rather than reuse `cur`: commit may have appended new
		// overlays during a yield window, but the merged one is always the front.
		b.storePending(slices.Clone(b.loadPending()[1:]))
		b.pendingSize -= m.size
		b.lock.Unlock()
	}
}

// waitMerge blocks until the background merger (if any) has drained all pending
// overlays and exited.
func (b *buffer) waitMerge() {
	b.wg.Wait()
}

// revertTo is the reverse operation of commit. It also merges the provided states
// and trie nodes into the buffer. The key difference is that the provided state
// set should reverse the changes made by the most recent state transition.
func (b *buffer) revertTo(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node, accounts map[common.Hash][]byte, storages map[common.Hash]map[common.Hash][]byte) error {
	// Reverting operates on the fully merged maps; wait for the merger to drain
	// all outstanding overlays first (no commit can be in flight here).
	b.waitMerge()

	// Short circuit if no embedded state transition to revert
	if b.layers == 0 {
		return errStateUnrecoverable
	}
	b.layers--

	// Reset the entire buffer if only a single transition left
	if b.layers == 0 {
		b.nodes.reset()
		b.states.reset()
		return nil
	}
	b.nodes.revertTo(db, nodes)
	b.states.revertTo(accounts, storages)
	return nil
}

// empty returns an indicator if buffer is empty.
func (b *buffer) empty() bool {
	return b.layers == 0
}

// full returns an indicator if the size of accumulated content exceeds the
// configured threshold.
func (b *buffer) full() bool {
	return b.size() > b.limit
}

// size returns the approximate memory size of the held content, including the
// not-yet-merged overlays.
func (b *buffer) size() uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.states.size + b.nodes.size + b.pendingSize // pendingSize is mutable
}

// flush persists the in-memory dirty trie node into the disk if the configured
// memory threshold is reached. Note, all data must be written atomically.
func (b *buffer) flush(root common.Hash, db ethdb.KeyValueStore, freezers []ethdb.AncientWriter, progress []byte, nodesCache, statesCache *fastcache.Cache, id uint64, postFlush func()) {
	if b.done != nil {
		panic("duplicated flush operation")
	}
	// Wait for the merger to integrate any remaining overlays before reading the
	// maps. The buffer is frozen at this point, so no new overlays can arrive and
	// the maps are stable once waitMerge returns.
	b.waitMerge()

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
		if head+b.layers != id {
			b.flushErr = fmt.Errorf("buffer layers (%d) cannot be applied on top of persisted state id (%d) to reach requested state id (%d)", b.layers, head, id)
			return
		}

		// Terminate the state snapshot generation if it's active
		var (
			start = time.Now()
			batch = db.NewBatchWithSize((b.nodes.dbsize() + b.states.dbsize()) * 11 / 10) // extra 10% for potential pebble internal stuff
		)
		// Explicitly sync the state freezer to ensure all written data is persisted to disk
		// before updating the key-value store.
		//
		// This step is crucial to guarantee that the corresponding state history remains
		// available for state rollback.
		if err := syncHistory(freezers...); err != nil {
			b.flushErr = err
			return
		}
		nodes := b.nodes.write(batch, nodesCache)
		accounts, slots := b.states.write(batch, progress, statesCache)
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
		commitNodesMeter.Mark(int64(nodes))
		commitAccountsMeter.Mark(int64(accounts))
		commitStoragesMeter.Mark(int64(slots))
		commitTimeTimer.UpdateSince(start)

		log.Debug("Persisted buffer content", "nodes", nodes, "accounts", accounts, "slots", slots, "bytes", common.StorageSize(size), "elapsed", common.PrettyDuration(time.Since(start)))
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
