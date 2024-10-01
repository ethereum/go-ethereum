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

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// diskLayer is a low level persistent layer built on top of a key-value store.
type diskLayer struct {
	root   common.Hash      // Immutable, root hash to which this layer was made for
	id     uint64           // Immutable, corresponding state id
	db     *Database        // Path-based trie database
	nodes  *fastcache.Cache // GC friendly memory cache of clean nodes
	states *fastcache.Cache // GC friendly memory cache of clean states
	buffer *buffer          // Dirty buffer to aggregate writes of nodes and states
	stale  bool             // Signals that the layer became stale (state progressed)
	lock   sync.RWMutex     // Lock used to protect stale flag and genMarker

	// The generator is set if the state snapshot was not fully completed
	generator *generator
}

// newDiskLayer creates a new disk layer based on the passing arguments.
func newDiskLayer(root common.Hash, id uint64, db *Database, nodes *fastcache.Cache, states *fastcache.Cache, buffer *buffer) *diskLayer {
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

// isStale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diskLayer) isStale() bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.stale
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
	// Try to retrieve the trie node from the not-yet-written
	// node buffer first. Note the buffer is lock free since
	// it's impossible to mutate the buffer before tagging the
	// layer as stale.
	n, found := dl.buffer.node(owner, path)
	if found {
		dirtyNodeHitMeter.Mark(1)
		dirtyNodeReadMeter.Mark(int64(len(n.Blob)))
		dirtyNodeHitDepthHist.Update(int64(depth))
		return n.Blob, n.Hash, &nodeLoc{loc: locDirtyCache, depth: depth}, nil
	}
	dirtyNodeMissMeter.Mark(1)

	// Try to retrieve the trie node from the clean memory cache
	h := newHasher()
	defer h.release()

	key := nodeCacheKey(owner, path)
	if dl.nodes != nil {
		if blob := dl.nodes.Get(nil, key); len(blob) > 0 {
			cleanNodeHitMeter.Mark(1)
			cleanNodeReadMeter.Mark(int64(len(blob)))
			return blob, h.hash(blob), &nodeLoc{loc: locCleanCache, depth: depth}, nil
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
	if dl.nodes != nil && len(blob) > 0 {
		dl.nodes.Set(key, blob)
		cleanNodeWriteMeter.Mark(int64(len(blob)))
	}
	return blob, h.hash(blob), &nodeLoc{loc: locDiskLayer, depth: depth}, nil
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
	// Try to retrieve the trie node from the not-yet-written
	// node buffer first. Note the buffer is lock free since
	// it's impossible to mutate the buffer before tagging the
	// layer as stale.
	blob, found := dl.buffer.account(hash)
	if found {
		dirtyStateHitMeter.Mark(1)
		dirtyStateReadMeter.Mark(int64(len(blob)))
		dirtyStateHitDepthHist.Update(int64(depth))

		if len(blob) == 0 {
			stateAccountMissMeter.Mark(1)
		} else {
			stateAccountHitMeter.Mark(1)
		}
		return blob, nil
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
				stateAccountMissMeter.Mark(1)
			} else {
				stateAccountHitMeter.Mark(1)
			}
			return blob, nil
		}
		cleanStateMissMeter.Mark(1)
	}
	// Try to retrieve the account from the disk.
	blob = rawdb.ReadAccountSnapshot(dl.db.diskdb, hash)
	if dl.states != nil {
		dl.states.Set(hash[:], blob)
		cleanStateWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) == 0 {
		stateAccountMissMeter.Mark(1)
		stateAccountDiskMissMeter.Mark(1)
	} else {
		stateAccountHitMeter.Mark(1)
		stateAccountDiskHitMeter.Mark(1)
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
	if blob, found := dl.buffer.storage(accountHash, storageHash); found {
		dirtyStateHitMeter.Mark(1)
		dirtyStateReadMeter.Mark(int64(len(blob)))
		dirtyStateHitDepthHist.Update(int64(depth))

		if len(blob) == 0 {
			stateStorageMissMeter.Mark(1)
		} else {
			stateStorageHitMeter.Mark(1)
		}
		return blob, nil
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
				stateStorageMissMeter.Mark(1)
			} else {
				stateStorageHitMeter.Mark(1)
			}
			return blob, nil
		}
		cleanStateMissMeter.Mark(1)
	}
	// Try to retrieve the account from the disk
	blob := rawdb.ReadStorageSnapshot(dl.db.diskdb, accountHash, storageHash)
	if dl.states != nil {
		dl.states.Set(key, blob)
		cleanStateWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) == 0 {
		stateStorageMissMeter.Mark(1)
		stateStorageDiskMissMeter.Mark(1)
	} else {
		stateStorageHitMeter.Mark(1)
		stateStorageDiskHitMeter.Mark(1)
	}
	return blob, nil
}

// update implements the layer interface, returning a new diff layer on top
// with the given state set.
func (dl *diskLayer) update(root common.Hash, id uint64, block uint64, nodes *nodeSet, states *StateSetWithOrigin) *diffLayer {
	return newDiffLayer(dl, root, id, block, nodes, states)
}

// commit merges the given bottom-most diff layer into the node buffer
// and returns a newly constructed disk layer. Note the current disk
// layer must be tagged as stale first to prevent re-access.
func (dl *diskLayer) commit(bottom *diffLayer, force bool) (*diskLayer, error) {
	// Construct and store the state history first. If crash happens after storing
	// the state history but without flushing the corresponding states(journal),
	// the stored state history will be truncated from head in the next restart.
	var (
		overflow bool
		oldest   uint64
	)
	if dl.db.freezer != nil {
		err := writeHistory(dl.db.freezer, bottom)
		if err != nil {
			return nil, err
		}
		// Determine if the persisted history object has exceeded the configured
		// limitation, set the overflow as true if so.
		tail, err := dl.db.freezer.Tail()
		if err != nil {
			return nil, err
		}
		limit := dl.db.config.StateHistory
		if limit != 0 && bottom.stateID()-tail > limit {
			overflow = true
			oldest = bottom.stateID() - limit + 1 // track the id of history **after truncation**
		}
	}
	// Mark the diskLayer as stale before applying any mutations on top.
	dl.markStale()

	// Store the root->id lookup afterwards. All stored lookups are identified
	// by the **unique** state root. It's impossible that in the same chain
	// blocks are not adjacent but have the same root.
	if dl.id == 0 {
		rawdb.WriteStateID(dl.db.diskdb, dl.root, 0)
	}
	rawdb.WriteStateID(dl.db.diskdb, bottom.rootHash(), bottom.stateID())

	// In a unique scenario where the ID of the oldest history object (after tail
	// truncation) surpasses the persisted state ID, we take the necessary action
	// of forcibly committing the cached dirty states to ensure that the persisted
	// state ID remains higher.
	if !force && rawdb.ReadPersistentStateID(dl.db.diskdb) < oldest {
		force = true
	}
	// Merge the trie nodes and flat states of the bottom-most diff layer into the
	// buffer as the combined layer.
	combined := dl.buffer.commit(bottom.nodes, bottom.states.stateSet)

	// Terminate the background state snapshot generation before mutating the
	// persistent state.
	if combined.full() || force {
		// Terminate the background state snapshot generator to prevent data race
		var progress []byte
		if dl.generator != nil {
			dl.generator.stop()
			progress = dl.generator.progressMarker()
			log.Info("Terminated state snapshot generation")
		}
		// Flush the content in combined buffer. Any state data after the progress
		// marker will be ignored, as the generator will pick it up later.
		if err := combined.flush(bottom.root, dl.db.diskdb, progress, dl.nodes, dl.states, bottom.stateID()); err != nil {
			return nil, err
		}
		// Relaunch the state snapshot generation if it's not done yet
		if progress != nil {
			dl.generator.run(bottom.root)
			log.Info("Resumed state snapshot generation", "root", bottom.root)
		}
	}
	ndl := newDiskLayer(bottom.root, bottom.stateID(), dl.db, dl.nodes, dl.states, combined)

	// Link the generator if snapshot is not yet completed
	if dl.generator != nil && !dl.generator.completed() {
		ndl.setGenerator(dl.generator)
	}
	// To remove outdated history objects from the end, we set the 'tail' parameter
	// to 'oldest-1' due to the offset between the freezer index and the history ID.
	if overflow {
		pruned, err := truncateFromTail(ndl.db.diskdb, ndl.db.freezer, oldest-1)
		if err != nil {
			return nil, err
		}
		log.Debug("Pruned state history", "items", pruned, "tailid", oldest)
	}
	return ndl, nil
}

// revert applies the given state history and return a reverted disk layer.
func (dl *diskLayer) revert(h *history) (*diskLayer, error) {
	if h.meta.root != dl.rootHash() {
		return nil, errUnexpectedHistory
	}
	if dl.id == 0 {
		return nil, fmt.Errorf("%w: zero state id", errStateUnrecoverable)
	}
	var (
		buff     = crypto.NewKeccakState()
		accounts = make(map[common.Hash][]byte)
		storages = make(map[common.Hash]map[common.Hash][]byte)
	)
	for addr, blob := range h.accounts {
		accounts[crypto.HashData(buff, addr.Bytes())] = blob
	}
	for addr, storage := range h.storages {
		storages[crypto.HashData(buff, addr.Bytes())] = storage
	}
	// Apply the reverse state changes upon the current state. This must
	// be done before holding the lock in order to access state in "this"
	// layer.
	nodes, err := apply(dl.db, h.meta.parent, h.meta.root, h.accounts, h.storages)
	if err != nil {
		return nil, err
	}
	// Mark the diskLayer as stale before applying any mutations on top.
	dl.markStale()

	// State change may be applied to node buffer, or the persistent
	// state, depends on if node buffer is empty or not. If the node
	// buffer is not empty, it means that the state transition that
	// needs to be reverted is not yet flushed and cached in node
	// buffer, otherwise, manipulate persistent state directly.
	if !dl.buffer.empty() {
		err := dl.buffer.revert(dl.db.diskdb, nodes, accounts, storages)
		if err != nil {
			return nil, err
		}
		ndl := newDiskLayer(h.meta.parent, dl.id-1, dl.db, dl.nodes, dl.states, dl.buffer)

		// Link the generator if it exists
		if dl.generator != nil {
			ndl.setGenerator(dl.generator)
		}
		return ndl, nil
	}
	// Terminate the generation before writing any data into database
	var progress []byte
	if dl.generator != nil {
		dl.generator.stop()
		progress = dl.generator.progressMarker()
	}
	batch := dl.db.diskdb.NewBatch()
	writeNodes(batch, nodes, dl.nodes)

	// Provide the original values of modified accounts and storages for revert.
	// Note the account deletions are included in accounts map (with value as nil),
	// rather than the destruction list (nil list).
	writeStates(dl.db.diskdb, batch, progress, nil, accounts, storages, dl.states)
	rawdb.WritePersistentStateID(batch, dl.id-1)
	rawdb.WriteSnapshotRoot(batch, h.meta.parent)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write states", "err", err)
	}
	// Link the generator and resume generation if the snapshot is not yet
	// fully completed.
	ndl := newDiskLayer(h.meta.parent, dl.id-1, dl.db, dl.nodes, dl.states, dl.buffer)
	if dl.generator != nil && !dl.generator.completed() {
		ndl.generator = dl.generator
		ndl.generator.run(h.meta.parent)
		log.Info("Resumed state snapshot generation", "root", h.meta.parent)
	}
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

// hasher is used to compute the sha256 hash of the provided data.
type hasher struct{ sha crypto.KeccakState }

var hasherPool = sync.Pool{
	New: func() interface{} { return &hasher{sha: crypto.NewKeccakState()} },
}

func newHasher() *hasher {
	return hasherPool.Get().(*hasher)
}

func (h *hasher) hash(data []byte) common.Hash {
	return crypto.HashData(h.sha, data)
}

func (h *hasher) release() {
	hasherPool.Put(h)
}
