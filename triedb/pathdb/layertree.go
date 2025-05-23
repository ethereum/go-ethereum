// Copyright 2023 The go-ethereum Authors
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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// layerTree is a group of state layers identified by the state root.
// This structure defines a few basic operations for manipulating
// state layers linked with each other in a tree structure. It's
// thread-safe to use. However, callers need to ensure the thread-safety
// of the referenced layer by themselves.
type layerTree struct {
	base   *diskLayer
	layers map[common.Hash]layer

	// descendants is a two-dimensional map where the keys represent
	// an ancestor state root, and the values are the state roots of
	// all its descendants.
	//
	// For example: r -> [c1, c2, ..., cn], where c1 through cn are
	// the descendants of state r.
	//
	// This map includes all the existing diff layers and the disk layer.
	descendants map[common.Hash]map[common.Hash]struct{}
	lookup      *lookup
	lock        sync.RWMutex
}

// newLayerTree constructs the layerTree with the given head layer.
func newLayerTree(head layer) *layerTree {
	tree := new(layerTree)
	tree.init(head)
	return tree
}

// init initializes the layerTree by the given head layer.
func (tree *layerTree) init(head layer) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	current := head
	tree.layers = make(map[common.Hash]layer)
	tree.descendants = make(map[common.Hash]map[common.Hash]struct{})

	for {
		tree.layers[current.rootHash()] = current
		tree.fillAncestors(current)

		parent := current.parentLayer()
		if parent == nil {
			break
		}
		current = parent
	}
	tree.base = current.(*diskLayer) // panic if it's not a disk layer
	tree.lookup = newLookup(head, tree.isDescendant)
}

// get retrieves a layer belonging to the given state root.
func (tree *layerTree) get(root common.Hash) layer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.layers[root]
}

// isDescendant returns whether the specified layer with given root is a
// descendant of a specific ancestor.
//
// This function assumes the read lock has been held.
func (tree *layerTree) isDescendant(root common.Hash, ancestor common.Hash) bool {
	subset := tree.descendants[ancestor]
	if subset == nil {
		return false
	}
	_, ok := subset[root]
	return ok
}

// fillAncestors identifies the ancestors of the given layer and populates the
// descendants set. The ancestors include the diff layers below the supplied
// layer and also the disk layer.
//
// This function assumes the write lock has been held.
func (tree *layerTree) fillAncestors(layer layer) {
	hash := layer.rootHash()
	for {
		parent := layer.parentLayer()
		if parent == nil {
			break
		}
		layer = parent

		phash := parent.rootHash()
		subset := tree.descendants[phash]
		if subset == nil {
			subset = make(map[common.Hash]struct{})
			tree.descendants[phash] = subset
		}
		subset[hash] = struct{}{}
	}
}

// forEach iterates the stored layers inside and applies the
// given callback on them.
func (tree *layerTree) forEach(onLayer func(layer)) {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	for _, layer := range tree.layers {
		onLayer(layer)
	}
}

// len returns the number of layers cached.
func (tree *layerTree) len() int {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return len(tree.layers)
}

// add inserts a new layer into the tree if it can be linked to an existing old parent.
func (tree *layerTree) add(root common.Hash, parentRoot common.Hash, block uint64, nodes *trienode.MergedNodeSet, states *StateSetWithOrigin) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// happen for clique networks and proof-of-stake networks where empty blocks
	// don't modify the state (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a layer.
	if root == parentRoot {
		return errors.New("layer cycle")
	}
	parent := tree.get(parentRoot)
	if parent == nil {
		return fmt.Errorf("triedb parent [%#x] layer missing", parentRoot)
	}
	l := parent.update(root, parent.stateID()+1, block, newNodeSet(nodes.Flatten()), states)

	tree.lock.Lock()
	defer tree.lock.Unlock()

	// Link the given layer into the layer set
	tree.layers[l.rootHash()] = l

	// Link the given layer into its ancestors (up to the current disk layer)
	tree.fillAncestors(l)

	// Link the given layer into the state mutation history
	tree.lookup.addLayer(l)
	return nil
}

// cap traverses downwards the diff tree until the number of allowed diff layers
// are crossed. All diffs beyond the permitted number are flattened downwards.
func (tree *layerTree) cap(root common.Hash, layers int) error {
	// Retrieve the head layer to cap from
	l := tree.get(root)
	if l == nil {
		return fmt.Errorf("triedb layer [%#x] missing", root)
	}
	diff, ok := l.(*diffLayer)
	if !ok {
		return fmt.Errorf("triedb layer [%#x] is disk layer", root)
	}
	tree.lock.Lock()
	defer tree.lock.Unlock()

	// If full commit was requested, flatten the diffs and merge onto disk
	if layers == 0 {
		base, err := diff.persist(true)
		if err != nil {
			return err
		}
		tree.base = base

		// Reset the layer tree with the single new disk layer
		tree.layers = map[common.Hash]layer{
			base.rootHash(): base,
		}
		// Resets the descendants map, since there's only a single disk layer
		// with no descendants.
		tree.descendants = make(map[common.Hash]map[common.Hash]struct{})
		tree.lookup = newLookup(base, tree.isDescendant)
		return nil
	}
	// Dive until we run out of layers or reach the persistent database
	for i := 0; i < layers-1; i++ {
		// If we still have diff layers below, continue down
		if parent, ok := diff.parentLayer().(*diffLayer); ok {
			diff = parent
		} else {
			// Diff stack too shallow, return without modifications
			return nil
		}
	}
	// We're out of layers, flatten anything below, stopping if it's the disk or if
	// the memory limit is not yet exceeded.
	var (
		err      error
		replaced layer
		newBase  *diskLayer
	)
	switch parent := diff.parentLayer().(type) {
	case *diskLayer:
		return nil

	case *diffLayer:
		// Hold the lock to prevent any read operations until the new
		// parent is linked correctly.
		diff.lock.Lock()

		// Hold the reference of the original layer being replaced
		replaced = parent

		// Replace the original parent layer with new disk layer. The procedure
		// can be illustrated as below:
		//
		// Before change:
		//     Chain:
		//        C1->C2->C3->C4 (HEAD)
		//          ->C2'->C3'->C4'
		//
		// After change:
		//     Chain:
		//        (a) C3->C4 (HEAD)
		//		  (b) C1->C2
		//		        ->C2'->C3'->C4'
		// The original C3 is replaced by the new base (with root C3)
		// Dangling layers in (b) will be removed later
		newBase, err = parent.persist(false)
		if err != nil {
			diff.lock.Unlock()
			return err
		}
		tree.layers[newBase.rootHash()] = newBase

		// Link the new parent and release the lock
		diff.parent = newBase
		diff.lock.Unlock()

	default:
		panic(fmt.Sprintf("unknown data layer in triedb: %T", parent))
	}
	// Remove any layer that is stale or links into a stale layer
	children := make(map[common.Hash][]common.Hash)
	for root, layer := range tree.layers {
		if dl, ok := layer.(*diffLayer); ok {
			parent := dl.parentLayer().rootHash()
			children[parent] = append(children[parent], root)
		}
	}
	clearDiff := func(layer layer) {
		diff, ok := layer.(*diffLayer)
		if !ok {
			return
		}
		tree.lookup.removeLayer(diff)
	}
	var remove func(root common.Hash)
	remove = func(root common.Hash) {
		clearDiff(tree.layers[root])

		// Unlink the layer from the layer tree and cascade to its children
		delete(tree.descendants, root)
		delete(tree.layers, root)
		for _, child := range children[root] {
			remove(child)
		}
		delete(children, root)
	}
	remove(tree.base.rootHash()) // remove the old/stale disk layer
	clearDiff(replaced)          // remove the lookup data of the stale parent being replaced
	tree.base = newBase          // update the base layer with newly constructed one
	return nil
}

// bottom returns the bottom-most disk layer in this tree.
func (tree *layerTree) bottom() *diskLayer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.base
}

// lookupAccount returns the layer that is guaranteed to contain the account data
// corresponding to the specified state root being queried.
func (tree *layerTree) lookupAccount(accountHash common.Hash, state common.Hash) (layer, error) {
	// Hold the read lock to prevent the unexpected layer changes
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	tip := tree.lookup.accountTip(accountHash, state, tree.base.root)
	if tip == (common.Hash{}) {
		return nil, fmt.Errorf("[%#x] %w", state, errSnapshotStale)
	}
	l := tree.layers[tip]
	if l == nil {
		return nil, fmt.Errorf("triedb layer [%#x] missing", tip)
	}
	return l, nil
}

// lookupStorage returns the layer that is guaranteed to contain the storage slot
// data corresponding to the specified state root being queried.
func (tree *layerTree) lookupStorage(accountHash common.Hash, slotHash common.Hash, state common.Hash) (layer, error) {
	// Hold the read lock to prevent the unexpected layer changes
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	tip := tree.lookup.storageTip(accountHash, slotHash, state, tree.base.root)
	if tip == (common.Hash{}) {
		return nil, fmt.Errorf("[%#x] %w", state, errSnapshotStale)
	}
	l := tree.layers[tip]
	if l == nil {
		return nil, fmt.Errorf("triedb layer [%#x] missing", tip)
	}
	return l, nil
}
