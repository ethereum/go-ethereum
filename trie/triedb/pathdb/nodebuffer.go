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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

var _ trienodebuffer = &asyncnodebuffer{}

// asyncnodebuffer implement trienodebuffer interface, and async flush nodebuffer
// to disk. It includes two nodebuffer, the mutable and immutable nodebuffer. The
// mutable nodebuffer that up to the size limit switches the immutable, and the new
// mutable nodebuffer can continue to be committed nodes. Retrieves node will access
// mutable nodebuffer firstly, then immutable nodebuffer.
type asyncnodebuffer struct {
	mu         sync.RWMutex // Lock used to protect current and background switch
	current    *nodebuffer  // mutable nodebuffer is used to write and read nodes
	background *nodebuffer  // immutable nodebuffer is readonly and async flush to disk
}

// newAsyncNodeBuffer initializes the async node buffer with the provided nodes.
func newAsyncNodeBuffer(limit int, nodes map[common.Hash]map[string]*trienode.Node, layers uint64) *asyncnodebuffer {
	return &asyncnodebuffer{
		current:    newNodeBuffer(limit, nodes, layers),
		background: newNodeBuffer(limit, nil, 0),
	}
}

// node retrieves the trie node with given node info, retrieves the current, then background.
func (a *asyncnodebuffer) node(owner common.Hash, path []byte, hash common.Hash) (*trienode.Node, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	node, err := a.current.node(owner, path, hash)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return a.background.node(owner, path, hash)
	}
	return node, nil
}

// commit merges the dirty nodes into the current nodebuffer. This operation
// won't take the ownership of the nodes map which belongs to the bottom-most
// diff layer. It will just hold the node references from the given map which
// are safe to copy.
func (a *asyncnodebuffer) commit(nodes map[common.Hash]map[string]*trienode.Node) trienodebuffer {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.current.commit(nodes); err != nil {
		log.Warn("Failed to commit trie nodes", "error", err)
	}
	return a
}

// revert is the reverse operation of commit. It also merges the provided nodes
// into the nodebuffer, the difference is that the provided node set should
// revert the changes made by the last state transition.
func (a *asyncnodebuffer) revert(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	newBuf, err := a.current.merge(a.background)
	if err != nil {
		log.Warn("[BUG] failed to merge node cache under revert async node buffer", "error", err)
		return err
	}
	a.current = newBuf
	a.background.reset()
	return a.current.revert(db, nodes)
}

// setSize sets the nodebuffer size limit.
func (a *asyncnodebuffer) setSize(size int, db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	newBuf, err := a.current.merge(a.background)
	if err != nil {
		log.Warn("[BUG] failed to merge node cache under revert async node buffer", "error", err)
		return err
	}
	a.current = newBuf
	a.background.reset()
	a.current.setSize(size, db, clean, id)
	a.background.size = uint64(size)
	return nil
}

// reset cleans up the disk cache.
func (a *asyncnodebuffer) reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.current.reset()
	a.background.reset()
}

// empty returns an indicator if nodebuffer contains any state transition inside.
func (a *asyncnodebuffer) empty() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.current.empty() && a.background.empty()
}

// flush persists the immutable dirty trie node into the disk. If the configured
// memory threshold is reached, switch the mutable nodebuffer to immutable, if the
// previous immutable nodebuffer flushing to disk immediately return. Note, all
// data belongs the same nodebuffer must be written atomically.
func (a *asyncnodebuffer) flush(db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64, force bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if force {
		for {
			if a.background.immutable.Load() {
				time.Sleep(time.Duration(DefaultBackgroundFlushInterval) * time.Second)
				log.Info("waiting background memory table flush to disk for force flush node buffer")
				continue
			}
			a.current.immutable.Store(true)
			return a.current.flush(db, clean, id, true)
		}
	}

	if a.current.size < a.current.limit {
		return nil
	}

	// background flush doing
	if a.background.immutable.Load() {
		return nil
	}
	// immutable the current nodebuffer, ready for switching
	a.current.immutable.Store(true)
	a.current, a.background = a.background, a.current

	go func(persistId uint64) {
		for {
			err := a.background.flush(db, clean, persistId, true)
			if err == nil {
				log.Debug("succeed to flush background nodecahce to disk", "state_id", persistId)
				return
			}
			log.Error("failed to flush background nodecahce to disk", "state_id", persistId, "error", err)
		}
	}(id)
	return nil
}

func (a *asyncnodebuffer) getAllNodes() map[common.Hash]map[string]*trienode.Node {
	a.mu.Lock()
	defer a.mu.Unlock()

	cached, err := a.current.merge(a.background)
	if err != nil {
		log.Crit("[BUG] failed to merge nodecache under revert asyncnodebuffer", "error", err)
	}
	return cached.nodes
}

func (a *asyncnodebuffer) getLayers() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.current.layers + a.background.layers
}

func (a *asyncnodebuffer) getSize() (uint64, uint64) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.current.size, a.background.size
}

// nodebuffer is a collection of modified trie nodes to aggregate the disk
// write. The content of the nodebuffer must be checked before diving into
// disk (since it basically is not-yet-written data).
type nodebuffer struct {
	layers uint64                                    // The number of diff layers aggregated inside
	size   uint64                                    // The size of aggregated writes
	limit  uint64                                    // The maximum memory allowance in bytes
	nodes  map[common.Hash]map[string]*trienode.Node // The dirty node set, mapped by owner and path
	// If this is set to true, then this nodebuffer is immutable and any write-operations to it will exit with error.
	// If this is set to true, then some other thread is performing a flush in the background, and thus nonblock the write/read-operations.
	immutable atomic.Bool // The flag equal true, readonly wait to flush nodes to disk background
}

// newNodeBuffer initializes the node buffer with the provided nodes.
func newNodeBuffer(limit int, nodes map[common.Hash]map[string]*trienode.Node, layers uint64) *nodebuffer {
	if nodes == nil {
		nodes = make(map[common.Hash]map[string]*trienode.Node)
	}
	var size uint64
	for _, subset := range nodes {
		for path, n := range subset {
			size += uint64(len(n.Blob) + len(path))
		}
	}
	nb := &nodebuffer{
		layers: layers,
		nodes:  nodes,
		size:   size,
		limit:  uint64(limit),
	}
	nb.immutable.Store(false)
	return nb
}

// node retrieves the trie node with given node info.
func (b *nodebuffer) node(owner common.Hash, path []byte, hash common.Hash) (*trienode.Node, error) {
	subset, ok := b.nodes[owner]
	if !ok {
		return nil, nil
	}
	n, ok := subset[string(path)]
	if !ok {
		return nil, nil
	}
	if n.Hash != hash {
		dirtyFalseMeter.Mark(1)
		log.Error("Unexpected trie node in node buffer", "owner", owner, "path", path, "expect", hash, "got", n.Hash)
		return nil, newUnexpectedNodeError("dirty", hash, n.Hash, owner, path, n.Blob)
	}
	return n, nil
}

// commit merges the dirty nodes into the nodebuffer. This operation won't take
// the ownership of the nodes map which belongs to the bottom-most diff layer.
// It will just hold the node references from the given map which are safe to
// copy.
func (b *nodebuffer) commit(nodes map[common.Hash]map[string]*trienode.Node) error {
	if b.immutable.Load() {
		return errWriteImmutable
	}

	var (
		delta         int64
		overwrite     int64
		overwriteSize int64
	)
	for owner, subset := range nodes {
		current, exist := b.nodes[owner]
		if !exist {
			// Allocate a new map for the subset instead of claiming it directly
			// from the passed map to avoid potential concurrent map read/write.
			// The nodes belong to original diff layer are still accessible even
			// after merging, thus the ownership of nodes map should still belong
			// to original layer and any mutation on it should be prevented.
			current = make(map[string]*trienode.Node)
			for path, n := range subset {
				current[path] = n
				delta += int64(len(n.Blob) + len(path))
			}
			b.nodes[owner] = current
			continue
		}
		for path, n := range subset {
			if orig, exist := current[path]; !exist {
				delta += int64(len(n.Blob) + len(path))
			} else {
				delta += int64(len(n.Blob) - len(orig.Blob))
				overwrite++
				overwriteSize += int64(len(orig.Blob) + len(path))
			}
			current[path] = n
		}
		b.nodes[owner] = current
	}
	b.updateSize(delta)
	b.layers++
	gcNodesMeter.Mark(overwrite)
	gcBytesMeter.Mark(overwriteSize)
	return nil
}

// revert is the reverse operation of commit. It also merges the provided nodes
// into the nodebuffer, the difference is that the provided node set should
// revert the changes made by the last state transition.
func (b *nodebuffer) revert(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node) error {
	if b.immutable.Load() {
		return errRevertImmutable
	}

	// Short circuit if no embedded state transition to revert.
	if b.layers == 0 {
		return errStateUnrecoverable
	}
	b.layers--

	// Reset the entire buffer if only a single transition left.
	if b.layers == 0 {
		b.reset()
		return nil
	}
	var delta int64
	for owner, subset := range nodes {
		current, ok := b.nodes[owner]
		if !ok {
			panic(fmt.Sprintf("non-existent subset (%x)", owner))
		}
		for path, n := range subset {
			orig, ok := current[path]
			if !ok {
				// There is a special case in MPT that one child is removed from
				// a fullNode which only has two children, and then a new child
				// with different position is immediately inserted into the fullNode.
				// In this case, the clean child of the fullNode will also be
				// marked as dirty because of node collapse and expansion.
				//
				// In case of database rollback, don't panic if this "clean"
				// node occurs which is not present in buffer.
				var nhash common.Hash
				if owner == (common.Hash{}) {
					_, nhash = rawdb.ReadAccountTrieNode(db, []byte(path))
				} else {
					_, nhash = rawdb.ReadStorageTrieNode(db, owner, []byte(path))
				}
				// Ignore the clean node in the case described above.
				if nhash == n.Hash {
					continue
				}
				panic(fmt.Sprintf("non-existent node (%x %v) blob: %v", owner, path, crypto.Keccak256Hash(n.Blob).Hex()))
			}
			current[path] = n
			delta += int64(len(n.Blob)) - int64(len(orig.Blob))
		}
	}
	b.updateSize(delta)
	return nil
}

// updateSize updates the total cache size by the given delta.
func (b *nodebuffer) updateSize(delta int64) {
	size := int64(b.size) + delta
	if size >= 0 {
		b.size = uint64(size)
		return
	}
	s := b.size
	b.size = 0
	log.Error("Invalid pathdb buffer size", "prev", common.StorageSize(s), "delta", common.StorageSize(delta))
}

// reset cleans up the disk cache.
func (b *nodebuffer) reset() {
	b.immutable.Store(false)
	b.layers = 0
	b.size = 0
	b.nodes = make(map[common.Hash]map[string]*trienode.Node)
}

// empty returns an indicator if nodebuffer contains any state transition inside.
func (b *nodebuffer) empty() bool {
	return b.layers == 0
}

// setSize sets the buffer size to the provided number, and invokes a flush
// operation if the current memory usage exceeds the new limit.
func (b *nodebuffer) setSize(size int, db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64) error {
	if b.immutable.Load() {
		return errWriteImmutable
	}

	b.limit = uint64(size)
	return b.flush(db, clean, id, false)
}

// merge returns a new nodebuffer instances that include `b` and `nb` nodes.
func (b *nodebuffer) merge(nb *nodebuffer) (*nodebuffer, error) {
	if b == nil && nb == nil {
		return nil, nil
	}
	if b == nil || b.empty() {
		res := copyNodeBuffer(nb)
		res.immutable.Store(false)
		return nb, nil
	}
	if nb == nil || nb.empty() {
		res := copyNodeBuffer(b)
		res.immutable.Store(false)
		return b, nil
	}
	if b.immutable.Load() == nb.immutable.Load() {
		return nil, errIncompatibleMerge
	}

	var (
		immutable *nodebuffer
		mutable   *nodebuffer
	)
	if b.immutable.Load() {
		immutable = b
		mutable = nb
	} else {
		immutable = nb
		mutable = b
	}

	nodes := make(map[common.Hash]map[string]*trienode.Node)
	for acc, subTree := range immutable.nodes {
		if _, ok := nodes[acc]; !ok {
			nodes[acc] = make(map[string]*trienode.Node)
		}
		for path, node := range subTree {
			nodes[acc][path] = node
		}
	}

	for acc, subTree := range mutable.nodes {
		if _, ok := nodes[acc]; !ok {
			nodes[acc] = make(map[string]*trienode.Node)
		}
		for path, node := range subTree {
			nodes[acc][path] = node
		}
	}
	return newNodeBuffer(int(mutable.limit), nodes, immutable.layers+mutable.layers), nil
}

// flush persists the in-memory dirty trie node into the disk if the configured
// memory threshold is reached. Note, all data must be written atomically.
func (b *nodebuffer) flush(db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64, force bool) error {
	if !b.immutable.Load() {
		return errFlushMutable
	}

	if b.size <= b.limit && !force {
		return nil
	}
	// Ensure the target state id is aligned with the internal counter.
	head := rawdb.ReadPersistentStateID(db)
	if head+b.layers != id {
		return fmt.Errorf("buffer layers (%d) cannot be applied on top of persisted state id (%d) to reach requested state id (%d)", b.layers, head, id)
	}
	var (
		start = time.Now()
		batch = db.NewBatchWithSize(int(b.size))
	)
	nodes := writeNodes(batch, b.nodes, clean)
	rawdb.WritePersistentStateID(batch, id)

	// Flush all mutations in a single batch
	size := batch.ValueSize()
	if err := batch.Write(); err != nil {
		return err
	}
	commitBytesMeter.Mark(int64(size))
	commitNodesMeter.Mark(int64(nodes))
	commitTimeTimer.UpdateSince(start)
	log.Debug("Persisted pathdb nodes", "nodes", len(b.nodes), "bytes", common.StorageSize(size), "elapsed", common.PrettyDuration(time.Since(start)))
	b.reset()
	return nil
}

// writeNodes writes the trie nodes into the provided database batch.
// Note this function will also inject all the newly written nodes
// into clean cache.
func writeNodes(batch ethdb.Batch, nodes map[common.Hash]map[string]*trienode.Node, clean *fastcache.Cache) (total int) {
	for owner, subset := range nodes {
		for path, n := range subset {
			if n.IsDeleted() {
				if owner == (common.Hash{}) {
					rawdb.DeleteAccountTrieNode(batch, []byte(path))
				} else {
					rawdb.DeleteStorageTrieNode(batch, owner, []byte(path))
				}
				if clean != nil {
					clean.Del(cacheKey(owner, []byte(path)))
				}
			} else {
				if owner == (common.Hash{}) {
					rawdb.WriteAccountTrieNode(batch, []byte(path), n.Blob)
				} else {
					rawdb.WriteStorageTrieNode(batch, owner, []byte(path), n.Blob)
				}
				if clean != nil {
					clean.Set(cacheKey(owner, []byte(path)), n.Blob)
				}
			}
		}
		total += len(subset)
	}
	return total
}

// cacheKey constructs the unique key of clean cache.
func cacheKey(owner common.Hash, path []byte) []byte {
	if owner == (common.Hash{}) {
		return path
	}
	return append(owner.Bytes(), path...)
}

// copyNodeBuffer returns a new instance nodebuffer that copy the data of 'n'.
func copyNodeBuffer(n *nodebuffer) *nodebuffer {
	if n == nil {
		return nil
	}
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	for acc, subTree := range n.nodes {
		if _, ok := nodes[acc]; !ok {
			nodes[acc] = make(map[string]*trienode.Node)
		}
		for path, node := range subTree {
			nodes[acc][path] = node
		}
	}
	nb := newNodeBuffer(int(n.limit), nodes, n.layers)
	nb.immutable.Store(n.immutable.Load())
	return nb
}
