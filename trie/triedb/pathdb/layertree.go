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
)

// layerTree is a group of state layers identified by the state root.
// This structure defines a few basic operations for manipulating
// state layers linked with each other in a tree structure. It's
// thread-safe to use. However, callers need to ensure the thread-safety
// of the layer layer operated by themselves.
type layerTree struct {
	lock   sync.RWMutex
	layers map[common.Hash]layer
}

// newLayerTree initializes the layerTree by the given head layer.
// All the ancestors will be iterated out and linked in the tree.
func newLayerTree(head layer) *layerTree {
	var layers = make(map[common.Hash]layer)
	for head != nil {
		layers[head.Root()] = head
		head = head.Parent()
	}
	return &layerTree{layers: layers}
}

// get retrieves a layer belonging to the given block root.
func (tree *layerTree) get(blockRoot common.Hash) layer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.layers[types.TrieRootHash(blockRoot)]
}

// forEach iterates the stored layer layers inside and applies the
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

// add inserts a new layer into the tree if it can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
func (tree *layerTree) add(root common.Hash, parentRoot common.Hash, sets *trienode.MergedNodeSet) error {
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
	nodes, err := fixset(sets, parent)
	if err != nil {
		return err
	}
	l := parent.Update(root, parent.ID()+1, nodes)

	tree.lock.Lock()
	tree.layers[l.root] = l
	tree.lock.Unlock()
	return nil
}

// cap traverses downwards the diff tree until the number of allowed diff layers
// are crossed. All diffs beyond the permitted number are flattened downwards.
// An optional reserve set can be provided to prevent the specified diff layers
// from being flattened. Note that this may prevent the diff layers from being
// written to disk and eventually leads to out-of-memory.
func (tree *layerTree) cap(root common.Hash, layers int) error {
	// Retrieve the head layer to cap from
	root = types.TrieRootHash(root)
	snap := tree.get(root)
	if snap == nil {
		return fmt.Errorf("triedb layer [%#x] missing", root)
	}
	diff, ok := snap.(*diffLayer)
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
		// Replace the entire layer tree with the flat base
		tree.layers = map[common.Hash]layer{base.Root(): base}
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
	for root, l := range tree.layers {
		if l.Stale() {
			remove(root)
		}
	}
	return nil
}

// bottom returns the bottom-most layer in this tree. The returned
// layer can be diskLayer or nil if something corrupted.
func (tree *layerTree) bottom() layer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	if len(tree.layers) == 0 {
		return nil // Shouldn't happen, empty tree
	}
	// pick a random one
	var current layer
	for _, layer := range tree.layers {
		current = layer
		break
	}
	for current.Parent() != nil {
		current = current.Parent()
	}
	return current
}

// fixset iterates the provided nodeset and tries to retrieve the original value
// of nodes from parent layer in case the original value is not recorded.
func fixset(sets *trienode.MergedNodeSet, parent layer) (map[common.Hash]map[string]*trienode.WithPrev, error) {
	var (
		hits  int
		miss  int
		nodes = make(map[common.Hash]map[string]*trienode.WithPrev)
	)
	for owner, set := range sets.Sets {
		nodes[owner] = set.Nodes
		for path, n := range nodes[owner] {
			if len(n.Prev) != 0 {
				hits += 1
				continue
			}
			miss += 1
			// If the original value is not recorded, try to retrieve
			// it from database directly. It can happen that the node
			// is newly created which is considered not existent in
			// database previously. But since we left the storage tries
			// of destructed account in the database, it's possible
			// that there are some dangling nodes have the exact same
			// node path which should be treated as the origin value.
			prev, err := parent.nodeByPath(owner, []byte(path))
			if err != nil {
				return nil, err
			}
			nodes[owner][path] = trienode.NewWithPrev(n.Hash, n.Blob, prev)
		}
	}
	hitAccessListMeter.Mark(int64(hits))
	hitDatabaseMeter.Mark(int64(miss))
	return nodes, nil
}
