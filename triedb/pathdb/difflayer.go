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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

type RefTrieNode struct {
	refCount uint32
	node     *trienode.Node
}

type HashNodeCache struct {
	lock  sync.RWMutex
	cache map[common.Hash]*RefTrieNode
}

func (h *HashNodeCache) length() int {
	if h == nil {
		return 0
	}
	h.lock.RLock()
	defer h.lock.RUnlock()
	return len(h.cache)
}

func (h *HashNodeCache) set(hash common.Hash, node *trienode.Node) {
	if h == nil {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	if n, ok := h.cache[hash]; ok {
		n.refCount++
	} else {
		h.cache[hash] = &RefTrieNode{1, node}
	}
}

func (h *HashNodeCache) Get(hash common.Hash) *trienode.Node {
	if h == nil {
		return nil
	}
	h.lock.RLock()
	defer h.lock.RUnlock()
	if n, ok := h.cache[hash]; ok {
		return n.node
	}
	return nil
}

func (h *HashNodeCache) del(hash common.Hash) {
	if h == nil {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	n, ok := h.cache[hash]
	if !ok {
		return
	}
	if n.refCount > 0 {
		n.refCount--
	}
	if n.refCount == 0 {
		delete(h.cache, hash)
	}
}

func (h *HashNodeCache) Add(ly layer) {
	if h == nil {
		return
	}
	dl, ok := ly.(*diffLayer)
	if !ok {
		return
	}
	beforeAdd := h.length()
	for _, subset := range dl.nodes {
		for _, node := range subset {
			h.set(node.Hash, node)
		}
	}
	diffHashCacheLengthGauge.Update(int64(h.length()))
	log.Debug("Add difflayer to hash map", "root", ly.rootHash(), "block_number", dl.block, "map_len", h.length(), "add_delta", h.length()-beforeAdd)
}

func (h *HashNodeCache) Remove(ly layer) {
	if h == nil {
		return
	}
	dl, ok := ly.(*diffLayer)
	if !ok {
		return
	}
	go func() {
		beforeDel := h.length()
		for _, subset := range dl.nodes {
			for _, node := range subset {
				h.del(node.Hash)
			}
		}
		diffHashCacheLengthGauge.Update(int64(h.length()))
		log.Debug("Remove difflayer from hash map", "root", ly.rootHash(), "block_number", dl.block, "map_len", h.length(), "del_delta", beforeDel-h.length())
	}()
}

// diffLayer represents a collection of modifications made to the in-memory tries
// along with associated state changes after running a block on top.
//
// The goal of a diff layer is to act as a journal, tracking recent modifications
// made to the state, that have not yet graduated into a semi-immutable state.
type diffLayer struct {
	// Immutables
	root   common.Hash                               // Root hash to which this layer diff belongs to
	id     uint64                                    // Corresponding state id
	block  uint64                                    // Associated block number
	nodes  map[common.Hash]map[string]*trienode.Node // Cached trie nodes indexed by owner and path
	states *triestate.Set                            // Associated state change set for building history
	memory uint64                                    // Approximate guess as to how much memory we use
	cache  *HashNodeCache                            // trienode cache by hash key. cache is immutable, but cache's item can be add/del.

	// mutables
	origin *diskLayer   // The current difflayer corresponds to the underlying disklayer and is updated during cap.
	parent layer        // Parent layer modified by this one, never nil, **can be changed**
	lock   sync.RWMutex // Lock used to protect parent
}

// newDiffLayer creates a new diff layer on top of an existing layer.
func newDiffLayer(parent layer, root common.Hash, id uint64, block uint64, nodes map[common.Hash]map[string]*trienode.Node, states *triestate.Set) *diffLayer {
	var (
		size  int64
		count int
	)
	dl := &diffLayer{
		root:   root,
		id:     id,
		block:  block,
		nodes:  nodes,
		states: states,
		parent: parent,
	}
	switch l := parent.(type) {
	case *diskLayer:
		dl.origin = l
		dl.cache = &HashNodeCache{
			cache: make(map[common.Hash]*RefTrieNode),
		}
	case *diffLayer:
		dl.origin = l.originDiskLayer()
		dl.cache = l.cache
	default:
		panic("unknown parent type")
	}

	for _, subset := range nodes {
		for path, n := range subset {
			dl.memory += uint64(n.Size() + len(path))
			size += int64(len(n.Blob) + len(path))
		}
		count += len(subset)
	}
	if states != nil {
		dl.memory += uint64(states.Size())
	}
	dirtyWriteMeter.Mark(size)
	diffLayerNodesMeter.Mark(int64(count))
	diffLayerBytesMeter.Mark(int64(dl.memory))
	log.Debug("Created new diff layer", "id", id, "block", block, "nodes", count, "size", common.StorageSize(dl.memory))
	return dl
}

func (dl *diffLayer) originDiskLayer() *diskLayer {
	dl.lock.RLock()
	defer dl.lock.RUnlock()
	return dl.origin
}

// rootHash implements the layer interface, returning the root hash of
// corresponding state.
func (dl *diffLayer) rootHash() common.Hash {
	return dl.root
}

// stateID implements the layer interface, returning the state id of the layer.
func (dl *diffLayer) stateID() uint64 {
	return dl.id
}

// parentLayer implements the layer interface, returning the subsequent
// layer of the diff layer.
func (dl *diffLayer) parentLayer() layer {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.parent
}

// node implements the layer interface, retrieving the trie node blob with the
// provided node information. No error will be returned if the node is not found.
func (dl *diffLayer) node(owner common.Hash, path []byte, hash common.Hash, depth int) ([]byte, common.Hash, *nodeLoc, error) {
	if hash != (common.Hash{}) {
		if n := dl.cache.Get(hash); n != nil {
			// The query from the hash map is fastpath,
			// avoiding recursive query of 128 difflayers.
			diffHashCacheHitMeter.Mark(1)
			diffHashCacheReadMeter.Mark(int64(len(n.Blob)))
			return n.Blob, n.Hash, &nodeLoc{loc: locDiffLayer, depth: depth}, nil
		}
	}

	diffHashCacheMissMeter.Mark(1)
	persistLayer := dl.originDiskLayer()
	if persistLayer != nil {
		blob, bhash, nloc, err := persistLayer.node(owner, path, hash, depth+1)
		if err != nil {
			// This is a bad case with a very low probability.
			// r/w the difflayer cache and r/w the disklayer are not in the same lock,
			// so in extreme cases, both reading the difflayer cache and reading the disklayer may fail, eg, disklayer is stale.
			// In this case, fallback to the original 128-layer recursive difflayer query path.
			diffHashCacheSlowPathMeter.Mark(1)
			log.Debug("Retry difflayer due to query origin failed", "owner", owner, "path", path, "hash", hash.String(), "error", err)
			return dl.intervalNode(owner, path, hash, 0)
		} else { // This is the fastpath.
			return blob, bhash, nloc, nil
		}
	}
	diffHashCacheSlowPathMeter.Mark(1)
	log.Debug("Retry difflayer due to origin is nil", "owner", owner, "path", path, "hash", hash.String())
	return dl.intervalNode(owner, path, hash, 0)
}

func (dl *diffLayer) intervalNode(owner common.Hash, path []byte, hash common.Hash, depth int) ([]byte, common.Hash, *nodeLoc, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the trie node is known locally, return it
	subset, ok := dl.nodes[owner]
	if ok {
		n, ok := subset[string(path)]
		if ok {
			dirtyHitMeter.Mark(1)
			dirtyNodeHitDepthHist.Update(int64(depth))
			dirtyReadMeter.Mark(int64(len(n.Blob)))
			return n.Blob, n.Hash, &nodeLoc{loc: locDiffLayer, depth: depth}, nil
		}
	}
	// Trie node unknown to this layer, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.intervalNode(owner, path, hash, depth+1)
	}
	// Failed to resolve through diff layers, fallback to disk layer
	return dl.parent.node(owner, path, hash, depth+1)
}

// update implements the layer interface, creating a new layer on top of the
// existing layer tree with the specified data items.
func (dl *diffLayer) update(root common.Hash, id uint64, block uint64, nodes map[common.Hash]map[string]*trienode.Node, states *triestate.Set) *diffLayer {
	return newDiffLayer(dl, root, id, block, nodes, states)
}

// persist flushes the diff layer and all its parent layers to disk layer.
func (dl *diffLayer) persist(force bool) (layer, error) {
	if parent, ok := dl.parentLayer().(*diffLayer); ok {
		// Hold the lock to prevent any read operation until the new
		// parent is linked correctly.
		dl.lock.Lock()

		// The merging of diff layers starts at the bottom-most layer,
		// therefore we recurse down here, flattening on the way up
		// (diffToDisk).
		result, err := parent.persist(force)
		if err != nil {
			dl.lock.Unlock()
			return nil, err
		}
		dl.parent = result
		dl.lock.Unlock()
	}
	return diffToDisk(dl, force)
}

// diffToDisk merges a bottom-most diff into the persistent disk layer underneath
// it. The method will panic if called onto a non-bottom-most diff layer.
func diffToDisk(layer *diffLayer, force bool) (layer, error) {
	disk, ok := layer.parentLayer().(*diskLayer)
	if !ok {
		panic(fmt.Sprintf("unknown layer type: %T", layer.parentLayer()))
	}
	return disk.commit(layer, force)
}
