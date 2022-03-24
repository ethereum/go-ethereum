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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// layerTree is a group of state layers identified by the state root.
// This structure defines a few basic operations for manipulating
// state layers linked with each other in a tree structure. It's
// thread-safe to use.
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

// Snapshot retrieves a snapshot belonging to the given block root.
func (tree *layerTree) get(blockRoot common.Hash) Snapshot {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.layers[convertEmpty(blockRoot)]
}

// forEach iterates the stored snapshot layers inside and applies the
// given callback on them. Interrupt the iteration if it's required.
func (tree *layerTree) forEach(onLayer func(common.Hash, snapshot) bool) {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	for root, layer := range tree.layers {
		cont := onLayer(root, layer)
		if !cont {
			return
		}
	}
	return
}

// len returns the number of layers cached.
func (tree *layerTree) len() int {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return len(tree.layers)
}

// add inserts a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
func (tree *layerTree) add(root common.Hash, parentRoot common.Hash, nodes map[string]*nodeWithPreValue) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// only happen for Clique networks where empty blocks don't modify the state
	// (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a snapshot.
	root, parentRoot = convertEmpty(root), convertEmpty(parentRoot)
	if root == parentRoot {
		// TODO(rjl493456442) why have this special case?
		if root == emptyRoot {
			return nil
		}
		return errSnapshotCycle
	}
	parent := tree.get(parentRoot)
	if parent == nil {
		return fmt.Errorf("triedb parent [%#x] snapshot missing", parentRoot)
	}
	tree.lock.Lock()
	defer tree.lock.Unlock()

	// Generate a new snapshot on top of the parent
	snap := parent.(snapshot).Update(root, parent.(snapshot).ID()+1, nodes)
	tree.layers[snap.root] = snap
	return nil
}

// cap traverses downwards the diff tree until the number of allowed layers are
// crossed. All diffs beyond the permitted number are flattened downwards. If the
// layer limit is reached, memory cap is also enforced (but not before).
//
// The method returns the new disk layer if diffs were persisted into it.
//
// Note, the final diff layer count in general will be one more than the amount
// requested. This happens because the bottom-most diff layer is the accumulator
// which may or may not overflow and cascade to disk. Since this last layer's
// survival is only known *after* capping, we need to omit it from the count if
// we want to ensure that *at least* the requested number of diff layers remain.
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
	case *diskLayer, *diskLayerSnapshot:
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
	for root, snap := range tree.layers {
		if diff, ok := snap.(*diffLayer); ok {
			parent := diff.Parent().Root()
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

// convertEmpty converts the given hash to predefined emptyHash if it's empty.
func convertEmpty(hash common.Hash) common.Hash {
	if hash == (common.Hash{}) {
		return emptyRoot
	}
	return hash
}
