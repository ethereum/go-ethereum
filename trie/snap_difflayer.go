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

package trie

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// diffLayer represents a collection of modifications made to the in-memory tries
// after running a block on top.
//
// The goal of a diff layer is to act as a journal, tracking recent modifications
// made to the state, that have not yet graduated into a semi-immutable state.
type diffLayer struct {
	// Immutables
	root   common.Hash                              // Root hash to which this snapshot diff belongs to
	id     uint64                                   // Corresponding state id
	nodes  map[common.Hash]map[string]*nodeWithPrev // Cached trie nodes indexed by owner and path
	memory uint64                                   // Approximate guess as to how much memory we use

	parent snapshot     // Parent snapshot modified by this one, never nil, **can be changed**
	stale  bool         // Signals that the layer became stale (state progressed)
	lock   sync.RWMutex // Lock used to protect parent and stale fields.
}

// newDiffLayer creates a new diff on top of an existing snapshot, whether that's a low
// level persistent database or a hierarchical diff already.
func newDiffLayer(parent snapshot, root common.Hash, id uint64, nodes map[common.Hash]map[string]*nodeWithPrev) *diffLayer {
	dl := &diffLayer{
		root:   root,
		id:     id,
		nodes:  nodes,
		parent: parent,
	}
	var (
		size  int64
		count int
	)
	for _, subset := range nodes {
		for path, n := range subset {
			dl.memory += uint64(n.memorySize(len(path)))
			size += int64(uint16(len(path)) + n.size)
		}
		count += len(subset)
	}
	triedbDirtyWriteMeter.Mark(size)
	triedbDiffLayerSizeMeter.Mark(int64(dl.memory))
	triedbDiffLayerNodesMeter.Mark(int64(count))
	log.Debug("Created new diff layer", "id", id, "nodes", count, "size", common.StorageSize(dl.memory))
	return dl
}

// Root returns the root hash of corresponding state.
func (dl *diffLayer) Root() common.Hash {
	return dl.root
}

// ID returns the state id represented by layer.
func (dl *diffLayer) ID() uint64 {
	return dl.id
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
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.stale
}

// MarkStale sets the stale flag as true.
func (dl *diffLayer) MarkStale() {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.stale {
		panic("triedb diff layer is stale")
	}
	dl.stale = true
}

// node retrieves the node with provided storage key and node hash. The returned
// node is in a wrapper through which callers can obtain the RLP-format or canonical
// node representation easily. No error will be returned if node is not found.
func (dl *diffLayer) node(owner common.Hash, path []byte, hash common.Hash, depth int) (*memoryNode, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.stale {
		return nil, errSnapshotStale
	}
	// If the trie node is known locally, return it
	subset, ok := dl.nodes[owner]
	if ok {
		n, ok := subset[string(path)]
		if ok {
			// If the trie node is not hash matched, or marked as removed,
			// bubble up an error here. It shouldn't happen at all.
			if n.hash != hash {
				return nil, fmt.Errorf("difflayer %w %x!=%x(%x %v)", errUnexpectedNode, n.hash, hash, owner, path)
			}
			triedbDirtyHitMeter.Mark(1)
			triedbDirtyNodeHitDepthHist.Update(int64(depth))
			triedbDirtyReadMeter.Mark(int64(n.size))
			return n.unwrap(), nil
		}
	}
	// Trie node unknown to this layer, resolve from parent
	return dl.parent.node(owner, path, hash, depth+1)
}

// Node retrieves the trie node with the provided trie identifier, node path
// and the corresponding node hash. No error will be returned if the node is
// not found.
func (dl *diffLayer) Node(owner common.Hash, path []byte, hash common.Hash) (node, error) {
	n, err := dl.node(owner, path, hash, 0)
	if err != nil || n == nil {
		return nil, err
	}
	return n.obj(), nil
}

// NodeBlob retrieves the RLP-encoded trie node blob with the provided trie
// identifier, node path and the corresponding node hash. No error will be
// returned if the node is not found.
func (dl *diffLayer) NodeBlob(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	n, err := dl.node(owner, path, hash, 0)
	if err != nil || n == nil {
		return nil, err
	}
	return n.rlp(), nil
}

// nodeByPath retrieves the RLP-encoded trie node blob with the provided trie
// identifier and node path. No error will be returned if the node is not found.
func (dl *diffLayer) nodeByPath(owner common.Hash, path []byte) ([]byte, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.stale {
		return nil, errSnapshotStale
	}
	// If the trie node is known locally, return it
	subset, ok := dl.nodes[owner]
	if ok {
		n, ok := subset[string(path)]
		if ok {
			if n.isDeleted() {
				return nil, nil
			}
			return n.rlp(), nil
		}
	}
	// Trie node unknown to this layer, resolve from parent
	return dl.parent.nodeByPath(owner, path)
}

// Update creates a new layer on top of the existing snapshot diff tree with
// the specified data items.
func (dl *diffLayer) Update(blockRoot common.Hash, id uint64, nodes map[common.Hash]map[string]*nodeWithPrev) *diffLayer {
	return newDiffLayer(dl, blockRoot, id, nodes)
}

// persist stores the diff layer and all its parent diff layers to disk.
// The order should be strictly from bottom to top.
//
// Note this function can destruct the ancestor layers(mark them as stale)
// of the given diff layer, please ensure prevent state access operation
// to this layer through any **descendant layer**.
func (dl *diffLayer) persist(force bool) (snapshot, error) {
	parent, ok := dl.Parent().(*diffLayer)
	if ok {
		// Hold the lock to prevent any read operation until the new
		// parent is linked correctly.
		dl.lock.Lock()
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
func diffToDisk(bottom *diffLayer, force bool) (snapshot, error) {
	switch layer := bottom.Parent().(type) {
	case *diskLayer:
		return layer.commit(bottom, force)
	default:
		panic(fmt.Sprintf("unknown layer type: %T", layer))
	}
}
