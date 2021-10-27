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
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	bloomfilter "github.com/holiman/bloomfilter/v2"
)

var (
	// aggregatorMemoryLimit is the maximum size of the bottom-most diff layer
	// that aggregates the writes from above until it's flushed into the disk
	// layer.
	//
	// Note, bumping this up might drastically increase the size of the bloom
	// filters that's stored in every diff layer. Don't do that without fully
	// understanding all the implications.
	aggregatorMemoryLimit = uint64(32 * 1024 * 1024)

	// aggregatorItemLimit is an approximate number of items that will end up
	// in the aggregator layer before it's flushed out to disk. A plain node
	// weighs around 400B.
	aggregatorItemLimit = aggregatorMemoryLimit / 400

	// bloomTargetError is the target false positive rate when the aggregator
	// layer is at its fullest. The actual value will probably move around up
	// and down from this number, it's mostly a ballpark figure.
	//
	// Note, dropping this down might drastically increase the size of the bloom
	// filters that's stored in every diff layer. Don't do that without fully
	// understanding all the implications.
	bloomTargetError = 0.02

	// bloomSize is the ideal bloom filter size given the maximum number of items
	// it's expected to hold and the target false positive error rate.
	bloomSize = math.Ceil(float64(aggregatorItemLimit) * math.Log(bloomTargetError) / math.Log(1/math.Pow(2, math.Log(2))))

	// bloomFuncs is the ideal number of bits a single entry should set in the
	// bloom filter to keep its size to a minimum (given it's size and maximum
	// entry count).
	bloomFuncs = math.Round((bloomSize / float64(aggregatorItemLimit)) * math.Log(2))
)

// diffLayer represents a collection of modifications made to the in-memory tries
// after running a block on top.
//
// The goal of a diff layer is to act as a journal, tracking recent modifications
// made to the state, that have not yet graduated into a semi-immutable state.
type diffLayer struct {
	origin *diskLayer // Base disk layer to directly use on bloom misses
	parent snapshot   // Parent snapshot modified by this one, never nil
	memory uint64     // Approximate guess as to how much memory we use

	root   common.Hash            // Root hash to which this snapshot diff belongs to
	stale  uint32                 // Signals that the layer became stale (state progressed)
	nodes  map[string]*cachedNode // Keyed trie nodes for retrieval (nil means deleted)
	diffed *bloomfilter.Filter    // Bloom filter tracking all the diffed items up to the disk layer
	lock   sync.RWMutex
}

// newDiffLayer creates a new diff on top of an existing snapshot, whether that's a low
// level persistent database or a hierarchical diff already.
func newDiffLayer(parent snapshot, root common.Hash, nodes map[string]*cachedNode) *diffLayer {
	dl := &diffLayer{
		parent: parent,
		root:   root,
		nodes:  nodes,
	}
	switch parent := parent.(type) {
	case *diskLayer:
		dl.rebloom(parent)
	case *diffLayer:
		dl.rebloom(parent.origin)
	default:
		panic("unknown parent type")
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

// rebloom discards the layer's current bloom and rebuilds it from scratch based
// on the parent's and the local diffs.
func (dl *diffLayer) rebloom(origin *diskLayer) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	defer func(start time.Time) {
		triedbBloomIndexTimer.Update(time.Since(start))
	}(time.Now())

	// Inject the new origin that triggered the rebloom
	dl.origin = origin

	// Retrieve the parent bloom or create a fresh empty one
	if parent, ok := dl.parent.(*diffLayer); ok {
		parent.lock.RLock()
		dl.diffed, _ = parent.diffed.Copy()
		parent.lock.RUnlock()
	} else {
		dl.diffed, _ = bloomfilter.New(uint64(bloomSize), uint64(bloomFuncs))
	}
	for key := range dl.nodes {
		dl.diffed.Add(stateBloomHasher(key))
	}
	// Calculate the current false positive rate and update the error rate meter.
	// This is a bit cheating because subsequent layers will overwrite it, but it
	// should be fine, we're only interested in ballpark figures.
	k := float64(dl.diffed.K())
	n := float64(dl.diffed.N())
	m := float64(dl.diffed.M())
	triedbBloomErrorGauge.Update(math.Pow(1.0-math.Exp((-k)*(n+0.5)/(m-1)), k))
}

// Root returns the root hash of corresponding state.
func (dl *diffLayer) Root() common.Hash {
	return dl.root
}

// Parent returns the subsequent layer of a diff layer.
func (dl *diffLayer) Parent() snapshot {
	return dl.parent
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diffLayer) Stale() bool {
	return atomic.LoadUint32(&dl.stale) != 0
}

// Node retrieves the trie node associated with a particular key.
// The given key must be the internal format node key.
func (dl *diffLayer) Node(key []byte) (node, error) {
	// Check the bloom filter first whether there's even a point in reaching into
	// all the maps in all the layers below
	dl.lock.RLock()
	hit := dl.diffed.Contains(stateBloomHasher(key))
	if !hit {
		hit = dl.diffed.Contains(stateBloomHasher(key))
	}
	var origin *diskLayer
	if !hit {
		origin = dl.origin // extract origin while holding the lock
	}
	dl.lock.RUnlock()

	// If the bloom filter misses, don't even bother with traversing the memory
	// diff layers, reach straight into the bottom persistent disk layer
	if origin != nil {
		triedbBloomMissMeter.Mark(1)
		return origin.Node(key)
	}
	return dl.node(key, 0)
}

// node is the inner version of Node which counts the accessed layer depth.
func (dl *diffLayer) node(key []byte, depth int) (node, error) {
	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If the trie node is known locally, return it
	if n, ok := dl.nodes[string(key)]; ok {
		triedbDirtyHitMeter.Mark(1)
		triedbDirtyNodeHitDepthHist.Update(int64(depth))
		triedbBloomTrueHitMeter.Mark(1)

		// The trie node is marked as deleted, don't bother parent anymore.
		if n == nil {
			return nil, nil
		}
		triedbDirtyReadMeter.Mark(int64(n.size))
		_, hash := DecodeInternalKey(key)
		return n.obj(hash), nil
	}
	// Trie node unknown to this diff, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.node(key, depth+1)
	}
	// Failed to resolve through diff layers, mark a bloom error and use the disk
	triedbBloomFalseHitMeter.Mark(1)
	return dl.parent.Node(key)
}

// NodeBlob retrieves the trie node blob associated with a particular key.
// The given key must be the internal format node key.
func (dl *diffLayer) NodeBlob(key []byte) ([]byte, error) {
	// Check the bloom filter first whether there's even a point in reaching into
	// all the maps in all the layers below
	dl.lock.RLock()
	hit := dl.diffed.Contains(stateBloomHasher(key))
	if !hit {
		hit = dl.diffed.Contains(stateBloomHasher(key))
	}
	var origin *diskLayer
	if !hit {
		origin = dl.origin // extract origin while holding the lock
	}
	dl.lock.RUnlock()

	// If the bloom filter misses, don't even bother with traversing the memory
	// diff layers, reach straight into the bottom persistent disk layer
	if origin != nil {
		triedbBloomMissMeter.Mark(1)
		return origin.NodeBlob(key)
	}
	return dl.nodeBlob(key, 0)
}

// nodeBlob is the inner version of NodeBlob which counts the accessed layer depth.
func (dl *diffLayer) nodeBlob(key []byte, depth int) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If the trie node is known locally, return it
	if n, ok := dl.nodes[string(key)]; ok {
		triedbDirtyHitMeter.Mark(1)
		triedbDirtyNodeHitDepthHist.Update(int64(depth))
		triedbBloomTrueHitMeter.Mark(1)

		// The trie node is marked as deleted, don't bother parent anymore.
		if n == nil {
			return nil, nil
		}
		triedbDirtyReadMeter.Mark(int64(n.size))
		return n.rlp(), nil
	}
	// Trie node unknown to this diff, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.nodeBlob(key, depth+1)
	}
	// Failed to resolve through diff layers, mark a bloom error and use the disk
	triedbBloomFalseHitMeter.Mark(1)
	return dl.parent.NodeBlob(key)
}

// Update creates a new layer on top of the existing snapshot diff tree with
// the specified data items.
func (dl *diffLayer) Update(blockRoot common.Hash, nodes map[string]*cachedNode) *diffLayer {
	return newDiffLayer(dl, blockRoot, nodes)
}

// flatten pushes all data from this point downwards, flattening everything into
// a single diff at the bottom. Since usually the lowermost diff is the largest,
// the flattening builds up from there in reverse.
func (dl *diffLayer) flatten() snapshot {
	// If the parent is not diff, we're the first in line, return unmodified
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		return dl
	}
	// Parent is a diff, flatten it first (note, apart from weird corned cases,
	// flatten will realistically only ever merge 1 layer, so there's no need to
	// be smarter about grouping flattens together).
	parent = parent.flatten().(*diffLayer)

	parent.lock.Lock()
	defer parent.lock.Unlock()

	// Before actually writing all our data to the parent, first ensure that the
	// parent hasn't been 'corrupted' by someone else already flattening into it
	if atomic.SwapUint32(&parent.stale, 1) != 0 {
		panic("parent diff layer is stale") // we've flattened into the same parent from two children, boo
	}
	// Merge nodes of two layers together, overwrite the nodes with same path.
	storages := make(map[string]string)
	for key := range parent.nodes {
		storage, _ := DecodeInternalKey([]byte(key))
		storages[string(storage)] = key
	}
	for key, data := range dl.nodes {
		storage, _ := DecodeInternalKey([]byte(key))
		if internal, ok := storages[string(storage)]; ok {
			delete(parent.nodes, internal)
		}
		parent.nodes[key] = data
	}
	// Return the combo parent
	return &diffLayer{
		origin: parent.origin,
		parent: parent.parent,
		memory: parent.memory + dl.memory,
		root:   dl.root,
		nodes:  parent.nodes,
		diffed: dl.diffed,
	}
}
