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
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// diskLayer is a low level persistent layer built on top of a key-value store.
type diskLayer struct {
	root common.Hash // Immutable, root hash to which this layer was made for
	id   uint64      // Immutable, corresponding state id
	db   *Database   // Path-based trie database

	// These two caches must be maintained separately, because the key
	// for the root node of the storage trie (accountHash) is identical
	// to the key for the account data.
	nodes  *fastcache.Cache // GC friendly memory cache of clean nodes
	states *fastcache.Cache // GC friendly memory cache of clean states

	buffer *buffer // Live buffer to aggregate writes
	frozen *buffer // Frozen node buffer waiting for flushing

	stale bool         // Signals that the layer became stale (state progressed)
	lock  sync.RWMutex // Lock used to protect stale flag and genMarker

	// The generator is set if the state snapshot was not fully completed,
	// regardless of whether the background generation is running or not.
	// It should only be unset if the generation completes.
	generator *generator
}

// newDiskLayer creates a new disk layer based on the passing arguments.
func newDiskLayer(root common.Hash, id uint64, db *Database, nodes *fastcache.Cache, states *fastcache.Cache, buffer *buffer, frozen *buffer) *diskLayer {
	// Initialize the clean caches if the memory allowance is not zero
	// or reuse the provided caches if they are not nil (inherited from
	// the original disk layer).
	if nodes == nil && db.config.TrieCleanSize != 0 {
		nodes = fastcache.New(db.config.TrieCleanSize)
	}
	if states == nil && db.config.StateCleanSize != 0 {
		states = fastcache.New(db.config.StateCleanSize)
	}
	return &diskLayer{
		root:   root,
		id:     id,
		db:     db,
		nodes:  nodes,
		states: states,
		buffer: buffer,
		frozen: frozen,
	}
}

// rootHash implements the layer interface, returning root hash of corresponding state.
func (dl *diskLayer) rootHash() common.Hash {
	return dl.root
}

// stateID implements the layer interface, returning the state id of disk layer.
func (dl *diskLayer) stateID() uint64 {
	return dl.id
}

// parentLayer implements the layer interface, returning nil as there's no layer
// below the disk.
func (dl *diskLayer) parentLayer() layer {
	return nil
}

// setGenerator links the given generator to disk layer, representing the
// associated state snapshot is not fully completed yet and the generation
// is potentially running in the background.
func (dl *diskLayer) setGenerator(generator *generator) {
	dl.generator = generator
}

// markStale sets the stale flag as true.
func (dl *diskLayer) markStale() {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.stale {
		panic("triedb disk layer is stale") // we've committed into the same base from two children, boom
	}
	dl.stale = true
}

// node implements the layer interface, retrieving the trie node with the
// provided node info. No error will be returned if the node is not found.
func (dl *diskLayer) node(owner common.Hash, path []byte, depth int) ([]byte, common.Hash, *nodeLoc, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, common.Hash{}, nil, errSnapshotStale
	}
	// Try to retrieve the trie node from the not-yet-written node buffer first
	// (both the live one and the frozen one). Note the buffer is lock free since
	// it's impossible to mutate the buffer before tagging the layer as stale.
	for _, buffer := range []*buffer{dl.buffer, dl.frozen} {
		if buffer != nil {
			n, found := buffer.node(owner, path)
			if found {
				dirtyNodeHitMeter.Mark(1)
				dirtyNodeReadMeter.Mark(int64(len(n.Blob)))
				dirtyNodeHitDepthHist.Update(int64(depth))
				return n.Blob, n.Hash, &nodeLoc{loc: locDirtyCache, depth: depth}, nil
			}
		}
	}
	dirtyNodeMissMeter.Mark(1)

	// Try to retrieve the trie node from the clean memory cache
	key := nodeCacheKey(owner, path)
	if dl.nodes != nil {
		if blob := dl.nodes.Get(nil, key); len(blob) > 0 {
			cleanNodeHitMeter.Mark(1)
			cleanNodeReadMeter.Mark(int64(len(blob)))
			return blob, crypto.Keccak256Hash(blob), &nodeLoc{loc: locCleanCache, depth: depth}, nil
		}
		cleanNodeMissMeter.Mark(1)
	}
	// Try to retrieve the trie node from the disk.
	var blob []byte
	if owner == (common.Hash{}) {
		blob = rawdb.ReadAccountTrieNode(dl.db.diskdb, path)
	} else {
		blob = rawdb.ReadStorageTrieNode(dl.db.diskdb, owner, path)
	}
	// Store the resolved data in the clean cache. The background buffer flusher
	// may also write to the clean cache concurrently, but two writers cannot
	// write the same item with different content. If the item already exists,
	// it will be found in the frozen buffer, eliminating the need to check the
	// database.
	if dl.nodes != nil && len(blob) > 0 {
		dl.nodes.Set(key, blob)
		cleanNodeWriteMeter.Mark(int64(len(blob)))
	}
	return blob, crypto.Keccak256Hash(blob), &nodeLoc{loc: locDiskLayer, depth: depth}, nil
}

// account directly retrieves the account RLP associated with a particular
// hash in the slim data format.
//
// Note the returned account is not a copy, please don't modify it.
func (dl *diskLayer) account(hash common.Hash, depth int) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, errSnapshotStale
	}
	// Try to retrieve the trie node from the not-yet-written node buffer first
	// (both the live one and the frozen one). Note the buffer is lock free since
	// it's impossible to mutate the buffer before tagging the layer as stale.
	for _, buffer := range []*buffer{dl.buffer, dl.frozen} {
		if buffer != nil {
			blob, found := buffer.account(hash)
			if found {
				dirtyStateHitMeter.Mark(1)
				dirtyStateReadMeter.Mark(int64(len(blob)))
				dirtyStateHitDepthHist.Update(int64(depth))

				if len(blob) == 0 {
					stateAccountInexMeter.Mark(1)
				} else {
					stateAccountExistMeter.Mark(1)
				}
				return blob, nil
			}
		}
	}
	dirtyStateMissMeter.Mark(1)

	// If the layer is being generated, ensure the requested account has
	// already been covered by the generator.
	marker := dl.genMarker()
	if marker != nil && bytes.Compare(hash.Bytes(), marker) > 0 {
		return nil, errNotCoveredYet
	}
	// Try to retrieve the account from the memory cache
	if dl.states != nil {
		if blob, found := dl.states.HasGet(nil, hash[:]); found {
			cleanStateHitMeter.Mark(1)
			cleanStateReadMeter.Mark(int64(len(blob)))

			if len(blob) == 0 {
				stateAccountInexMeter.Mark(1)
			} else {
				stateAccountExistMeter.Mark(1)
			}
			return blob, nil
		}
		cleanStateMissMeter.Mark(1)
	}
	// Try to retrieve the account from the disk.
	blob := rawdb.ReadAccountSnapshot(dl.db.diskdb, hash)

	// Store the resolved data in the clean cache. The background buffer flusher
	// may also write to the clean cache concurrently, but two writers cannot
	// write the same item with different content. If the item already exists,
	// it will be found in the frozen buffer, eliminating the need to check the
	// database.
	if dl.states != nil {
		dl.states.Set(hash[:], blob)
		cleanStateWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) == 0 {
		stateAccountInexMeter.Mark(1)
		stateAccountInexDiskMeter.Mark(1)
	} else {
		stateAccountExistMeter.Mark(1)
		stateAccountExistDiskMeter.Mark(1)
	}
	return blob, nil
}

// storage directly retrieves the storage data associated with a particular hash,
// within a particular account.
//
// Note the returned account is not a copy, please don't modify it.
func (dl *diskLayer) storage(accountHash, storageHash common.Hash, depth int) ([]byte, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, errSnapshotStale
	}
	// Try to retrieve the trie node from the not-yet-written node buffer first
	// (both the live one and the frozen one). Note the buffer is lock free since
	// it's impossible to mutate the buffer before tagging the layer as stale.
	for _, buffer := range []*buffer{dl.buffer, dl.frozen} {
		if buffer != nil {
			if blob, found := buffer.storage(accountHash, storageHash); found {
				dirtyStateHitMeter.Mark(1)
				dirtyStateReadMeter.Mark(int64(len(blob)))
				dirtyStateHitDepthHist.Update(int64(depth))

				if len(blob) == 0 {
					stateStorageInexMeter.Mark(1)
				} else {
					stateStorageExistMeter.Mark(1)
				}
				return blob, nil
			}
		}
	}
	dirtyStateMissMeter.Mark(1)

	// If the layer is being generated, ensure the requested storage slot
	// has already been covered by the generator.
	key := append(accountHash[:], storageHash[:]...)
	marker := dl.genMarker()
	if marker != nil && bytes.Compare(key, marker) > 0 {
		return nil, errNotCoveredYet
	}
	// Try to retrieve the storage slot from the memory cache
	if dl.states != nil {
		if blob, found := dl.states.HasGet(nil, key); found {
			cleanStateHitMeter.Mark(1)
			cleanStateReadMeter.Mark(int64(len(blob)))

			if len(blob) == 0 {
				stateStorageInexMeter.Mark(1)
			} else {
				stateStorageExistMeter.Mark(1)
			}
			return blob, nil
		}
		cleanStateMissMeter.Mark(1)
	}
	// Try to retrieve the account from the disk
	blob := rawdb.ReadStorageSnapshot(dl.db.diskdb, accountHash, storageHash)

	// Store the resolved data in the clean cache. The background buffer flusher
	// may also write to the clean cache concurrently, but two writers cannot
	// write the same item with different content. If the item already exists,
	// it will be found in the frozen buffer, eliminating the need to check the
	// database.
	if dl.states != nil {
		dl.states.Set(key, blob)
		cleanStateWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) == 0 {
		stateStorageInexMeter.Mark(1)
		stateStorageInexDiskMeter.Mark(1)
	} else {
		stateStorageExistMeter.Mark(1)
		stateStorageExistDiskMeter.Mark(1)
	}
	return blob, nil
}

// update implements the layer interface, returning a new diff layer on top
// with the given state set.
func (dl *diskLayer) update(root common.Hash, id uint64, block uint64, nodes *nodeSetWithOrigin, states *StateSetWithOrigin) *diffLayer {
	return newDiffLayer(dl, root, id, block, nodes, states)
}

// writeStateHistory stores the state history and indexes if indexing is
// permitted.
//
// What's more, this function also returns a flag indicating whether the
// buffer flushing is required, ensuring the persistent state ID is always
// greater than or equal to the first history ID.
func (dl *diskLayer) writeStateHistory(diff *diffLayer) (bool, error) {
	// Short circuit if state history is not permitted
	if dl.db.stateFreezer == nil {
		return false, nil
	}
	// Bail out with an error if writing the state history fails.
	// This can happen, for example, if the device is full.
	err := writeStateHistory(dl.db.stateFreezer, diff)
	if err != nil {
		return false, err
	}
	// Notify the state history indexer for newly created history
	if dl.db.stateIndexer != nil {
		if err := dl.db.stateIndexer.extend(diff.stateID()); err != nil {
			return false, err
		}
	}
	// Determine if the persisted history object has exceeded the
	// configured limitation.
	limit := dl.db.config.StateHistory
	if limit == 0 {
		return false, nil
	}
	tail, err := dl.db.stateFreezer.Tail()
	if err != nil {
		return false, err
	} // firstID = tail+1

	// length = diff.stateID()-firstID+1 = diff.stateID()-tail
	if diff.stateID()-tail <= limit {
		return false, nil
	}
	newFirst := diff.stateID() - limit + 1 // the id of first history **after truncation**

	// In a rare case where the ID of the first history object (after tail
	// truncation) exceeds the persisted state ID, we must take corrective
	// steps:
	//
	// - Skip tail truncation temporarily, avoid the scenario that associated
	//   history of persistent state is removed
	//
	// - Force a commit of the cached dirty states into persistent state
	//
	// These measures ensure the persisted state ID always remains greater
	// than or equal to the first history ID.
	if persistentID := rawdb.ReadPersistentStateID(dl.db.diskdb); persistentID < newFirst {
		log.Debug("Skip tail truncation", "persistentID", persistentID, "tailID", tail+1, "headID", diff.stateID(), "limit", limit)
		return true, nil
	}
	pruned, err := truncateFromTail(dl.db.stateFreezer, typeStateHistory, newFirst-1)
	if err != nil {
		return false, err
	}
	log.Debug("Pruned state history", "items", pruned, "tailid", newFirst)
	return false, nil
}

// commit merges the given bottom-most diff layer into the node buffer
// and returns a newly constructed disk layer. Note the current disk
// layer must be tagged as stale first to prevent re-access.
func (dl *diskLayer) commit(bottom *diffLayer, force bool) (*diskLayer, error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	// Construct and store the state history first. If crash happens after storing
	// the state history but without flushing the corresponding states(journal),
	// the stored state history will be truncated from head in the next restart.
	flush, err := dl.writeStateHistory(bottom)
	if err != nil {
		return nil, err
	}
	// Mark the diskLayer as stale before applying any mutations on top.
	dl.stale = true

	// Store the root->id lookup afterwards. All stored lookups are identified
	// by the **unique** state root. It's impossible that in the same chain
	// blocks are not adjacent but have the same root.
	if dl.id == 0 {
		rawdb.WriteStateID(dl.db.diskdb, dl.root, 0)
	}
	rawdb.WriteStateID(dl.db.diskdb, bottom.rootHash(), bottom.stateID())

	// Merge the trie nodes and flat states of the bottom-most diff layer into the
	// buffer as the combined layer.
	combined := dl.buffer.commit(bottom.nodes.nodeSet, bottom.states.stateSet)

	// Terminate the background state snapshot generation before mutating the
	// persistent state.
	if combined.full() || force || flush {
		// Wait until the previous frozen buffer is fully flushed
		if dl.frozen != nil {
			if err := dl.frozen.waitFlush(); err != nil {
				return nil, err
			}
		}
		// Release the frozen buffer and the internally referenced maps will
		// be reclaimed by GC.
		dl.frozen = nil

		// Terminate the background state snapshot generator before flushing
		// to prevent data race.
		var (
			progress []byte
			gen      = dl.generator
		)
		if gen != nil {
			gen.stop()
			progress = gen.progressMarker()

			// If the snapshot has been fully generated, unset the generator
			if progress == nil {
				dl.setGenerator(nil)
			} else {
				log.Info("Paused snapshot generation")
			}
		}

		// Freeze the live buffer and schedule background flushing
		dl.frozen = combined
		dl.frozen.flush(bottom.root, dl.db.diskdb, dl.db.stateFreezer, progress, dl.nodes, dl.states, bottom.stateID(), func() {
			// Resume the background generation if it's not completed yet.
			// The generator is assumed to be available if the progress is
			// not nil.
			//
			// Notably, the generator will be shared and linked by all the
			// disk layer instances, regardless of the generation is terminated
			// or not.
			if progress != nil {
				gen.run(bottom.root)
			}
		})
		// Block until the frozen buffer is fully flushed out if the async flushing
		// is not allowed.
		if dl.db.config.NoAsyncFlush {
			if err := dl.frozen.waitFlush(); err != nil {
				return nil, err
			}
			dl.frozen = nil
		}
		combined = newBuffer(dl.db.config.WriteBufferSize, nil, nil, 0)
	}
	// Link the generator if snapshot is not yet completed
	ndl := newDiskLayer(bottom.root, bottom.stateID(), dl.db, dl.nodes, dl.states, combined, dl.frozen)
	if dl.generator != nil {
		ndl.setGenerator(dl.generator)
	}
	return ndl, nil
}

// revert applies the given state history and return a reverted disk layer.
func (dl *diskLayer) revert(h *stateHistory) (*diskLayer, error) {
	start := time.Now()
	if h.meta.root != dl.rootHash() {
		return nil, errUnexpectedHistory
	}
	if dl.id == 0 {
		return nil, fmt.Errorf("%w: zero state id", errStateUnrecoverable)
	}
	// Apply the reverse state changes upon the current state. This must
	// be done before holding the lock in order to access state in "this"
	// layer.
	nodes, err := apply(dl.db, h.meta.parent, h.meta.root, h.meta.version != stateHistoryV0, h.accounts, h.storages)
	if err != nil {
		return nil, err
	}
	// Derive the state modification set from the history, keyed by the hash
	// of the account address and the storage key.
	accounts, storages := h.stateSet()

	// Mark the diskLayer as stale before applying any mutations on top.
	dl.lock.Lock()
	defer dl.lock.Unlock()

	dl.stale = true

	// Unindex the corresponding state history
	if dl.db.stateIndexer != nil {
		if err := dl.db.stateIndexer.shorten(dl.id); err != nil {
			return nil, err
		}
	}
	// State change may be applied to node buffer, or the persistent
	// state, depends on if node buffer is empty or not. If the node
	// buffer is not empty, it means that the state transition that
	// needs to be reverted is not yet flushed and cached in node
	// buffer, otherwise, manipulate persistent state directly.
	if !dl.buffer.empty() {
		err := dl.buffer.revertTo(dl.db.diskdb, nodes, accounts, storages)
		if err != nil {
			return nil, err
		}
		ndl := newDiskLayer(h.meta.parent, dl.id-1, dl.db, dl.nodes, dl.states, dl.buffer, dl.frozen)

		// Link the generator if it exists
		if dl.generator != nil {
			ndl.setGenerator(dl.generator)
		}
		log.Debug("Reverted data in write buffer", "oldroot", h.meta.root, "newroot", h.meta.parent, "elapsed", common.PrettyDuration(time.Since(start)))
		return ndl, nil
	}
	// Block until the frozen buffer is fully flushed
	if dl.frozen != nil {
		if err := dl.frozen.waitFlush(); err != nil {
			return nil, err
		}
		// Unset the frozen buffer if it exists, otherwise these "reverted"
		// states will still be accessible after revert in frozen buffer.
		dl.frozen = nil
	}

	// Terminate the generator before writing any data to the database.
	// This must be done after flushing the frozen buffer, as the generator
	// may be restarted at the end of the flush process.
	var progress []byte
	if dl.generator != nil {
		dl.generator.stop()
		progress = dl.generator.progressMarker()
	}
	batch := dl.db.diskdb.NewBatch()
	writeNodes(batch, nodes, dl.nodes)

	// Provide the original values of modified accounts and storages for revert
	writeStates(batch, progress, accounts, storages, dl.states)
	rawdb.WritePersistentStateID(batch, dl.id-1)
	rawdb.WriteSnapshotRoot(batch, h.meta.parent)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write states", "err", err)
	}
	// Link the generator and resume generation if the snapshot is not yet
	// fully completed.
	ndl := newDiskLayer(h.meta.parent, dl.id-1, dl.db, dl.nodes, dl.states, dl.buffer, dl.frozen)
	if dl.generator != nil && !dl.generator.completed() {
		ndl.generator = dl.generator
		ndl.generator.run(h.meta.parent)
	}
	log.Debug("Reverted data in persistent state", "oldroot", h.meta.root, "newroot", h.meta.parent, "elapsed", common.PrettyDuration(time.Since(start)))
	return ndl, nil
}

// size returns the approximate size of cached nodes in the disk layer.
func (dl *diskLayer) size() common.StorageSize {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return 0
	}
	return common.StorageSize(dl.buffer.size())
}

// resetCache releases the memory held by clean cache to prevent memory leak.
func (dl *diskLayer) resetCache() {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// Stale disk layer loses the ownership of clean caches.
	if dl.stale {
		return
	}
	if dl.nodes != nil {
		dl.nodes.Reset()
	}
	if dl.states != nil {
		dl.states.Reset()
	}
}

// genMarker returns the current state snapshot generation progress marker. If
// the state snapshot has already been fully generated, nil is returned.
func (dl *diskLayer) genMarker() []byte {
	if dl.generator == nil {
		return nil
	}
	return dl.generator.progressMarker()
}

// genComplete returns a flag indicating whether the state snapshot has been
// fully generated.
func (dl *diskLayer) genComplete() bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.genMarker() == nil
}

// waitFlush blocks until the background buffer flush is completed.
func (dl *diskLayer) waitFlush() error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.frozen == nil {
		return nil
	}
	return dl.frozen.waitFlush()
}

// terminate releases the frozen buffer if it's not nil and terminates the
// background state generator.
func (dl *diskLayer) terminate() error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.frozen != nil {
		if err := dl.frozen.waitFlush(); err != nil {
			return err
		}
		dl.frozen = nil
	}
	if dl.generator != nil {
		dl.generator.stop()
	}
	return nil
}
