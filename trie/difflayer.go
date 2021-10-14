// Copyright 2021 The go-ethereum Authors
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

package trie

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// diffLayer represents a collection of modifications made to the in-memory tries
// after running a block on top.
//
// The goal of a diff layer is to act as a journal, tracking recent modifications
// made to the state, that have not yet graduated into a semi-immutable state.
type diffLayer struct {
	// Immutables
	root  common.Hash            // Root hash to which this snapshot diff belongs to
	rid   uint64                 // Corresponding reverse diff id
	nodes map[string]*cachedNode // Keyed trie nodes for retrieval, indexed by storage key

	parent snapshot     // Parent snapshot modified by this one, never nil, **can be changed**
	memory uint64       // Approximate guess as to how much memory we use
	stale  uint32       // Signals that the layer became stale (state progressed)
	lock   sync.RWMutex // Lock used to protect parent and stale fields.
}

// newDiffLayer creates a new diff on top of an existing snapshot, whether that's a low
// level persistent database or a hierarchical diff already.
func newDiffLayer(parent snapshot, root common.Hash, rid uint64, nodes map[string]*cachedNode) *diffLayer {
	dl := &diffLayer{
		root:   root,
		rid:    rid,
		nodes:  nodes,
		parent: parent,
	}
	for key, node := range nodes {
		dl.memory += uint64(len(key) + int(node.size) + cachedNodeSize)
		triedbDirtyWriteMeter.Mark(int64(node.size))
	}
	triedbDiffLayerSizeMeter.Mark(int64(dl.memory))
	triedbDiffLayerNodesMeter.Mark(int64(len(nodes)))
	log.Debug("Created new diff layer", "nodes", len(nodes), "size", common.StorageSize(dl.memory))
	return dl
}

// Root returns the root hash of corresponding state.
func (dl *diffLayer) Root() common.Hash {
	return dl.root
}

// ID returns the id of associated reverse diff.
func (dl *diffLayer) ID() uint64 {
	return dl.rid
}

// Parent returns the subsequent layer of a diff layer.
func (dl *diffLayer) Parent() snapshot {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.parent
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diffLayer) Stale() bool {
	return atomic.LoadUint32(&dl.stale) != 0
}

// Node retrieves the trie node associated with a particular key.
func (dl *diffLayer) Node(storage []byte, hash common.Hash) (node, error) {
	return dl.node(storage, hash, 0)
}

// node is the inner version of Node which counts the accessed layer depth.
func (dl *diffLayer) node(storage []byte, hash common.Hash, depth int) (node, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If the trie node is known locally, return it
	n, ok := dl.nodes[string(storage)]
	if ok {
		// If the trie node is not hash matched, or marked as removed,
		// panic here. It shouldn't happen at all.
		if n.hash != hash || n.node == nil {
			owner, path := DecodeStorageKey(storage)
			panic(fmt.Sprintf("missing node %x(%x %v)", hash, owner, path))
		}
		triedbDirtyHitMeter.Mark(1)
		triedbDirtyNodeHitDepthHist.Update(int64(depth))
		triedbDirtyReadMeter.Mark(int64(n.size))
		return n.obj(hash), nil
	}
	// Trie node unknown to this diff, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.node(storage, hash, depth+1)
	}
	return dl.parent.Node(storage, hash)
}

// NodeBlob retrieves the trie node blob associated with a particular key.
func (dl *diffLayer) NodeBlob(storage []byte, hash common.Hash) ([]byte, error) {
	return dl.nodeBlob(storage, hash, 0)
}

// nodeBlob is the inner version of NodeBlob which counts the accessed layer depth.
func (dl *diffLayer) nodeBlob(storage []byte, hash common.Hash, depth int) ([]byte, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If the trie node is known locally, return it
	n, ok := dl.nodes[string(storage)]
	if ok {
		// If the trie node is not hash matched, or marked as removed,
		// panic here. It shouldn't happen at all.
		if n.hash != hash || n.node == nil {
			owner, path := DecodeStorageKey(storage)
			panic(fmt.Sprintf("missing node %x(%x %v)", hash, owner, path))
		}
		triedbDirtyHitMeter.Mark(1)
		triedbDirtyNodeHitDepthHist.Update(int64(depth))
		triedbDirtyReadMeter.Mark(int64(n.size))
		return n.rlp(), nil
	}
	// Trie node unknown to this diff, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.nodeBlob(storage, hash, depth+1)
	}
	return dl.parent.NodeBlob(storage, hash)
}

// Update creates a new layer on top of the existing snapshot diff tree with
// the specified data items.
func (dl *diffLayer) Update(blockRoot common.Hash, id uint64, nodes map[string]*cachedNode) *diffLayer {
	return newDiffLayer(dl, blockRoot, id, nodes)
}

// persist persists the diff layer and all its parent diff layers to disk.
// The order should be strictly from bottom to top.
// Note this function can destruct the ancestor layers(mark them as stale)
// of the given diff layer, please ensure prevent state access operation
// to this layer through any **descendant layer**.
func (dl *diffLayer) persist(config *Config) snapshot {
	parent, ok := dl.Parent().(*diffLayer)
	if ok {
		// Hold the lock to prevent any read operation until the new
		// parent is linked correctly.
		dl.lock.Lock()
		dl.parent = parent.persist(config)
		dl.lock.Unlock()
	}
	return diffToDisk(dl, config)
}

// diffToDisk merges a bottom-most diff into the persistent disk layer underneath
// it. The method will panic if called onto a non-bottom-most diff layer. The disk
// layer persistence should be operated in an atomic way. All updates should be
// discarded if the whole transition if not finished.
func diffToDisk(bottom *diffLayer, config *Config) *diskLayer {
	var (
		totalSize int64
		base      = bottom.Parent().(*diskLayer)
		start     = time.Now()
		nodes     = len(bottom.nodes)
		batch     = base.diskdb.NewBatch()
	)
	// Construct and store the reverse diff firstly. If crash happens
	// after storing the reverse diff but without flushing the corresponding
	// states, the stored reverse diff will be truncated in the next restart.
	if err := storeReverseDiff(bottom, params.FullImmutabilityThreshold); err != nil {
		log.Error("Failed to store reverse diff", "err", err)
	}
	// Mark the base layer(disk layer) as stale since we are pushing
	// new nodes into the disk. A new disk layer needed to be created
	// and be linked to all existent bottom diff layers later.
	base.MarkStale()

	defer func(start time.Time) {
		triedbCommitTimeTimer.Update(time.Since(start))
	}(time.Now())

	for storage, n := range bottom.nodes {
		var blob []byte
		if n.node == nil {
			rawdb.DeleteTrieNode(batch, []byte(storage))
		} else {
			blob = n.rlp()
			rawdb.WriteTrieNode(batch, []byte(storage), blob)
			if config != nil && config.WriteLegacy {
				rawdb.WriteArchiveTrieNode(batch, n.hash, blob)
			}
			if base.cache != nil {
				base.cache.Set(EncodeInternalKey([]byte(storage), n.hash), blob)
			}
		}
		totalSize += int64(len(blob) + len(storage))
	}
	rawdb.WriteReverseDiffHead(batch, bottom.rid)

	triedbCommitSizeMeter.Mark(totalSize)
	triedbCommitNodesMeter.Mark(int64(len(bottom.nodes)))

	// Flush all the updates in the single db operation. Ensure the
	// disk layer transition is atomic.
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write bottom dirty trie nodes", "err", err)
	}
	log.Debug("Persisted uncommitted nodes", "nodes", nodes, "size", common.StorageSize(totalSize), "elapsed", common.PrettyDuration(time.Since(start)))
	return newDiskLayer(bottom.root, bottom.rid, base.cache, base.diskdb)
}
