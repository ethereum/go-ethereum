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

package pathdb

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

// layerTree is a group of state layers identified by the state root.
// This structure defines a few basic operations for manipulating
// state layers linked with each other in a tree structure. It's
// thread-safe to use. However, callers need to ensure the thread-safety
// of the referenced layer by themselves.
type layerTree struct {
	lock        sync.RWMutex
	base        *diskLayer
	layers      map[common.Hash]layer
	descendants map[common.Hash]map[common.Hash]struct{}
	lookup      *lookup
}

// newLayerTree constructs the layerTree with the given head layer.
func newLayerTree(head layer) *layerTree {
	tree := new(layerTree)
	tree.reset(head)
	return tree
}

// reset initializes the layerTree by the given head layer.
func (tree *layerTree) reset(head layer) {
	tree.lock.Lock()
	defer tree.lock.Unlock()

	var (
		current     = head
		layers      = make(map[common.Hash]layer)
		descendants = make(map[common.Hash]map[common.Hash]struct{})
	)
	for {
		hash := current.rootHash()
		layers[hash] = current

		// Traverse the ancestors (diff only) of the current layer and link them
		for h := range diffAncestors(current) {
			subset := descendants[h]
			if subset == nil {
				subset = make(map[common.Hash]struct{})
				descendants[h] = subset
			}
			subset[hash] = struct{}{}
		}
		parent := current.parentLayer()
		if parent == nil {
			break
		}
		current = parent
	}
	tree.base = current.(*diskLayer) // panic if it's a diff layer
	tree.layers = layers
	tree.descendants = descendants
	tree.lookup = newLookup(head, tree.isDescendant)
}

// diffAncestors returns all the ancestors of the specific layer (disk layer
// is not included).
func diffAncestors(layer layer) map[common.Hash]struct{} {
	set := make(map[common.Hash]struct{})
	for {
		parent := layer.parentLayer()
		if parent == nil {
			break
		}
		if _, ok := parent.(*diskLayer); ok {
			break
		}
		set[parent.rootHash()] = struct{}{}
		layer = parent
	}
	return set
}

// get retrieves a layer belonging to the given state root.
func (tree *layerTree) get(root common.Hash) layer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.layers[types.TrieRootHash(root)]
}

// isDescendant returns whether the specified layer with given root is a
// descendant of a specific ancestor.
func (tree *layerTree) isDescendant(root common.Hash, ancestor common.Hash) bool {
	subset := tree.descendants[ancestor]
	if subset == nil {
		return false
	}
	_, ok := subset[root]
	return ok
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
func (tree *layerTree) add(root common.Hash, parentRoot common.Hash, block uint64, nodes *trienode.MergedNodeSet, states *triestate.Set) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// happen for clique networks and proof-of-stake networks where empty blocks
	// don't modify the state (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a layer.
	root, parentRoot = types.TrieRootHash(root), types.TrieRootHash(parentRoot)
	if root == parentRoot {
		return errors.New("layer cycle")
	}
	parent := tree.get(parentRoot)
	if parent == nil {
		return fmt.Errorf("triedb parent [%#x] layer missing", parentRoot)
	}
	l := parent.update(root, parent.stateID()+1, block, nodes.Flatten(), states)

	tree.lock.Lock()
	defer tree.lock.Unlock()

	tree.layers[l.rootHash()] = l

	// Link the new layer into the descendents set
	for h := range diffAncestors(l) {
		subset := tree.descendants[h]
		if subset == nil {
			subset = make(map[common.Hash]struct{})
			tree.descendants[h] = subset
		}
		subset[l.rootHash()] = struct{}{}
	}
	tree.lookup.addLayer(l)
	return nil
}

// cap traverses downwards the diff tree until the number of allowed diff layers
// are crossed. All diffs beyond the permitted number are flattened downwards.
func (tree *layerTree) cap(root common.Hash, layers int) error {
	// Retrieve the head layer to cap from
	root = types.TrieRootHash(root)
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

		// Mark all diff layers are stale, note the original disk layer
		// has already been marked as stale previously.
		for _, l := range tree.layers {
			if dl, ok := l.(*diffLayer); ok {
				dl.markStale()
			}
		}
		// Reset the layer tree with the single new disk layer
		tree.layers = map[common.Hash]layer{base.rootHash(): base}
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
		// Before change: base <- C1 <- C2 <- C3 (diff)
		// After change:  base(stale) <- C1 (C2 is replaced by newBase)
		//                newBase <- C3 (diff)
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
	// clearDiff removes the indexes of the specific layer if it's a
	// diff layer (disk layer has no index).
	clearDiff := func(layer layer) {
		diff, ok := layer.(*diffLayer)
		if !ok {
			return
		}
		diff.markStale()
		delete(tree.descendants, diff.rootHash())
		tree.lookup.removeLayer(diff)
	}
	var remove func(root common.Hash)
	remove = func(root common.Hash) {
		clearDiff(tree.layers[root])

		// Unlink the layer from the layer tree and cascade to its children
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

// lookupNode returns the layer that is confirmed to contain the node being
// searched for.
func (tree *layerTree) lookupNode(accountHash common.Hash, path []byte, state common.Hash) layer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	tip := tree.lookup.nodeTip(accountHash, path, state)
	if tip == (common.Hash{}) {
		return tree.base
	}
	return tree.layers[tip]
}
