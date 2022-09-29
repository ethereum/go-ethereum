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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package trie

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// layerTree is a group of state layers identified by the state root.
// This structure defines a few basic operations for manipulating
// state layers linked with each other in a tree structure. It's
// thread-safe to use. However, callers need to ensure the thread-safety
// of the snapshot layer operated by themselves.
type layerTree struct {
	lock   sync.RWMutex
	layers map[common.Hash]snapshot
}

// newLayerTree initializes the layerTree by the given head snapshot.
// All the ancestors will be iterated out and linked in the tree.
func newLayerTree(head snapshot) *layerTree {
	var layers = make(map[common.Hash]snapshot)
	for head != nil {
		layers[head.Root()] = head
		head = head.Parent()
	}
	return &layerTree{layers: layers}
}

// get retrieves a snapshot belonging to the given block root.
func (tree *layerTree) get(blockRoot common.Hash) Reader {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.layers[convertEmpty(blockRoot)]
}

// forEach iterates the stored snapshot layers inside and applies the
// given callback on them.
func (tree *layerTree) forEach(onLayer func(snapshot)) {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	for _, layer := range tree.layers {
		onLayer(layer)
	}
}

// forEachAndReset is mostly identical with forEach but reset the tree after
// iteration.
func (tree *layerTree) forEachAndReset(onLayer func(snapshot)) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	for _, layer := range tree.layers {
		onLayer(layer)
	}
	tree.layers = make(map[common.Hash]snapshot)
}

// len returns the number of layers cached.
func (tree *layerTree) len() int {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return len(tree.layers)
}

// add inserts a new snapshot into the tree if it can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
func (tree *layerTree) add(root common.Hash, parentRoot common.Hash, set *MergedNodeSet) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// happen for clique networks and proof-of-stake networks where empty blocks
	// don't modify the state (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a snapshot.
	root, parentRoot = convertEmpty(root), convertEmpty(parentRoot)
	if root == parentRoot {
		return errors.New("snapshot cycle")
	}
	parent := tree.get(parentRoot)
	if parent == nil {
		return fmt.Errorf("triedb parent [%#x] snapshot missing", parentRoot)
	}
	nodes, err := simplify(set, parent.(snapshot))
	if err != nil {
		return err
	}
	tree.lock.Lock()
	defer tree.lock.Unlock()

	snap := parent.(snapshot).Update(root, parent.(snapshot).ID()+1, nodes)
	tree.layers[snap.root] = snap
	return nil
}

// cap traverses downwards the diff tree until the number of allowed diff layers
// are crossed. All diffs beyond the permitted number are flattened downwards.
// An optional reserve set can be provided to prevent the specified diff layers
// from being flattened. Note that this may prevent the diff layers from being
// written to disk and eventually leads to out-of-memory.
func (tree *layerTree) cap(root common.Hash, layers int) error {
	// Retrieve the head snapshot to cap from
	root = convertEmpty(root)
	snap := tree.get(root)
	if snap == nil {
		return fmt.Errorf("triedb snapshot [%#x] missing", root)
	}
	diff, ok := snap.(*diffLayer)
	if !ok {
		return fmt.Errorf("triedb snapshot [%#x] is disk layer", root)
	}
	tree.lock.Lock()
	defer tree.lock.Unlock()

	// If full commit was requested, flatten the diffs and merge onto disk
	if layers == 0 {
		base, err := diff.persist(true)
		if err != nil {
			return err
		}
		// Replace the entire snapshot tree with the flat base
		tree.layers = map[common.Hash]snapshot{base.Root(): base}
		return nil
	}
	// Dive until we run out of layers or reach the persistent database
	for i := 0; i < layers-1; i++ {
		// If we still have diff layers below, continue down
		if parent, ok := diff.Parent().(*diffLayer); ok {
			diff = parent
		} else {
			// Diff stack too shallow, return without modifications
			return nil
		}
	}
	// We're out of layers, flatten anything below, stopping if it's the disk or if
	// the memory limit is not yet exceeded.
	switch parent := diff.Parent().(type) {
	case *diskLayer:
		return nil

	case *diffLayer:
		// Hold the lock to prevent any read operations until the new
		// parent is linked correctly.
		diff.lock.Lock()

		base, err := parent.persist(false)
		if err != nil {
			diff.lock.Unlock()
			return err
		}
		tree.layers[base.Root()] = base
		diff.parent = base

		diff.lock.Unlock()

	default:
		panic(fmt.Sprintf("unknown data layer in triedb: %T", parent))
	}
	// Remove any layer that is stale or links into a stale layer
	children := make(map[common.Hash][]common.Hash)
	for root, layer := range tree.layers {
		if dl, ok := layer.(*diffLayer); ok {
			parent := dl.Parent().Root()
			children[parent] = append(children[parent], root)
		}
	}
	var remove func(root common.Hash)
	remove = func(root common.Hash) {
		delete(tree.layers, root)
		for _, child := range children[root] {
			remove(child)
		}
		delete(children, root)
	}
	for root, snap := range tree.layers {
		if snap.Stale() {
			remove(root)
		}
	}
	return nil
}

// bottom returns the bottom-most snapshot layer in this tree. The returned
// layer can be diskLayer, diskLayerSnapshot or nil if something corrupted.
func (tree *layerTree) bottom() snapshot {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	if len(tree.layers) == 0 {
		return nil // Shouldn't happen, empty tree
	}
	var current snapshot
	for _, layer := range tree.layers {
		current = layer
		break
	}
	for current.Parent() != nil {
		current = current.Parent()
	}
	return current
}

// convertEmpty converts the given hash to predefined emptyHash if it's empty.
func convertEmpty(hash common.Hash) common.Hash {
	if hash == (common.Hash{}) {
		return emptyRoot
	}
	return hash
}

// simplify converts the set to a two-dimensional map in which nodes are mapped
// by owner and path, also fill the previous value either from database or the
// optional tracked accessList.
func simplify(set *MergedNodeSet, layer snapshot) (map[common.Hash]map[string]*nodeWithPrev, error) {
	var (
		hits  int
		miss  int
		nodes = make(map[common.Hash]map[string]*nodeWithPrev)
	)
	for owner, subset := range set.sets {
		nodes[owner] = make(map[string]*nodeWithPrev)
		for path, n := range subset.nodes {
			prev, ok := subset.accessList[path]
			if ok {
				hits += 1
			} else {
				// If the original value is not found in the accessList,
				// fallback to database. In fact this only happens if the
				// trie is created from scratch which doesn't load any node
				// from database, but in this case, it's still possible
				// there are some leftover nodes in database which should
				// be the previous values.
				blob, err := layer.nodeByPath(owner, []byte(path))
				if err != nil {
					return nil, err
				}
				prev = blob // blob can be nil in case it's not existent
				miss += 1
			}
			nodes[owner][path] = &nodeWithPrev{
				memoryNode: n,
				prev:       prev,
			}
		}
	}
	triedbHitAccessListMeter.Mark(int64(hits))
	triedbHitDatabaseMeter.Mark(int64(miss))
	return nodes, nil
}
