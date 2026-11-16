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
)

// layerTree is a group of state layers identified by the state root.
// This structure defines a few basic operations for manipulating
// state layers linked with each other in a tree structure. It's
// thread-safe to use. However, callers need to ensure the thread-safety
// of the referenced layer by themselves.
type layerTree struct {
	base   *diskLayer
	layers map[common.Hash]layer
	index  *layerIndex // Index locating the diff layers holding an entry
	lock   sync.RWMutex
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

	var (
		current = head
		diffs   []*diffLayer
	)
	tree.layers = make(map[common.Hash]layer)
	for {
		tree.layers[current.rootHash()] = current
		if dl, ok := current.(*diffLayer); ok {
			diffs = append(diffs, dl)
		}
		parent := current.parentLayer()
		if parent == nil {
			break
		}
		current = parent
	}
	tree.base = current.(*diskLayer) // panic if it's not a disk layer

	// Reject the layers beyond the span of the index, never happens in practice
	// as the diff layers are capped far below
	if len(diffs) > 0 && diffs[0].id > tree.base.id+indexSlots {
		panic(fmt.Sprintf("too many diff layers, id: %d, base: %d", diffs[0].id, tree.base.id))
	}
	// Index the diff layers in the order of creation
	tree.index = newLayerIndex(tree.base.id)
	for i := len(diffs) - 1; i >= 0; i-- {
		tree.index.schedule(indexTask{layer: diffs[i]})
	}
}

// get retrieves a layer belonging to the given state root.
func (tree *layerTree) get(root common.Hash) layer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.layers[root]
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
func (tree *layerTree) add(root common.Hash, parentRoot common.Hash, block uint64, nodes *nodeSetWithOrigin, states *StateSetWithOrigin) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// happen for clique networks and proof-of-stake networks where empty blocks
	// don't modify the state (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a layer.
	if root == parentRoot {
		return errors.New("layer cycle")
	}
	// If a layer with this root already exists, skip the insertion. Fork blocks
	// can produce the same state root as the canonical block (same parent, same
	// coinbase, zero txs); overwriting tree.layers[root] would corrupt the parent
	// chain for any child layers already built on top of the existing one.
	if tree.get(root) != nil {
		return nil
	}
	parent := tree.get(parentRoot)
	if parent == nil {
		return fmt.Errorf("triedb parent [%#x] layer missing", parentRoot)
	}
	l := parent.update(root, parent.stateID()+1, block, nodes, states)

	tree.lock.Lock()
	defer tree.lock.Unlock()

	// Reject the layer beyond the span of the index, never happens in practice
	// as the diff layers are capped far below
	if l.stateID() > tree.base.id+indexSlots {
		return fmt.Errorf("too many diff layers, id: %d, base: %d", l.stateID(), tree.base.id)
	}
	// Link the given layer into the layer set and schedule the indexing
	tree.layers[l.rootHash()] = l
	tree.index.schedule(indexTask{layer: l})
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
		tree.index = newLayerIndex(base.id)
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
		err     error
		newBase *diskLayer
	)
	switch parent := diff.parentLayer().(type) {
	case *diskLayer:
		return nil

	case *diffLayer:
		// Hold the lock to prevent any read operations until the new
		// parent is linked correctly.
		diff.lock.Lock()

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
	var remove func(root common.Hash)
	remove = func(root common.Hash) {
		// Unlink the layer from the layer tree and cascade to its children
		delete(tree.layers, root)
		for _, child := range children[root] {
			remove(child)
		}
		delete(children, root)
	}
	remove(tree.base.rootHash()) // remove the old/stale disk layer
	tree.base = newBase          // update the base layer with newly constructed one

	// Clear the index for the persisted layers
	tree.index.schedule(indexTask{base: newBase.id})
	return nil
}

// bottom returns the bottom-most disk layer in this tree.
func (tree *layerTree) bottom() *diskLayer {
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	return tree.base
}

// lookup returns the layer holding the entry with the given bloom key in the
// specified state, along with its distance from the state, or the disk layer if
// no diff layer holds it. has reports whether a diff layer holds the entry, it's
// consulted only for the layers whose filter fails to rule the key out.
func (tree *layerTree) lookup(state common.Hash, key uint64, has func(*diffLayer) bool) (layer, int, error) {
	// Hold the read lock to prevent the unexpected layer changes, the parent
	// links are stable with the lock held
	tree.lock.RLock()
	defer tree.lock.RUnlock()

	l := tree.layers[state]
	if l == nil {
		return nil, 0, fmt.Errorf("[%#x] %w", state, errSnapshotStale)
	}
	var found layer = tree.base
	if dl := tree.lookupDiff(l, key, has); dl != nil {
		found = dl
	}
	var depth int
	if top := l.stateID(); top > found.stateID() {
		depth = int(top - found.stateID())
	}
	return found, depth, nil
}

// lookupDiff returns the newest diff layer at or below the given layer holding
// the entry with the given bloom key, nil if none of the diff layers above the
// disk layer holds it. The read lock must be held.
func (tree *layerTree) lookupDiff(l layer, key uint64, has func(*diffLayer) bool) *diffLayer {
	var (
		base  = tree.base.id
		index = tree.index
		found *diffLayer

		unindexed, candidates int64 // the number of the layers probed
		falses                int64 // the number of the candidates not holding the entry
	)
	// Probe the layers not indexed yet one by one, they're always on the top as
	// the layers are indexed in the order of creation.
	dl, ok := l.(*diffLayer)
	for ok && dl.id > base && dl.indexed.Load() != index {
		unindexed++
		if dl.bloom.Load().contains(key) && has(dl) {
			found = dl
			break
		}
		dl, ok = dl.parent.(*diffLayer)
	}
	// Resolve the candidates located by the index from the nearest one, the
	// layers are descended to on the way.
	if found == nil && ok && dl.id > base {
		var (
			top   = dl.id
			cands = index.candidates(key, top)
		)
		for {
			dist, ok := nextCandidate(&cands)
			if !ok || dist >= top-base {
				break
			}
			if dl = descendLayer(dl, top-dist, base); dl == nil {
				break
			}
			candidates++
			if dl.bloom.Load().contains(key) && has(dl) {
				found = dl
				break
			}
			falses++
		}
	}
	if unindexed > 0 {
		layerLookupUnindexedMeter.Mark(unindexed)
	}
	if candidates > 0 {
		layerLookupCandidateMeter.Mark(candidates)
	}
	if falses > 0 {
		layerLookupFalseMeter.Mark(falses)
	}
	return found
}

// descendLayer returns the ancestor of the given diff layer with the specified
// state id, nil if it's at or below the disk layer.
func descendLayer(dl *diffLayer, id uint64, base uint64) *diffLayer {
	for dl.id > id {
		parent, ok := dl.parent.(*diffLayer)
		if !ok || parent.id <= base {
			return nil
		}
		dl = parent
	}
	return dl
}

// lookupAccount returns the layer that is guaranteed to contain the account data
// corresponding to the specified state root being queried.
func (tree *layerTree) lookupAccount(accountHash common.Hash, state common.Hash) (layer, error) {
	l, _, err := tree.lookup(state, bloomAccountKey(accountHash), func(dl *diffLayer) bool {
		_, ok := dl.states.account(accountHash)
		return ok
	})
	return l, err
}

// lookupStorage returns the layer that is guaranteed to contain the storage slot
// data corresponding to the specified state root being queried.
func (tree *layerTree) lookupStorage(accountHash common.Hash, slotHash common.Hash, state common.Hash) (layer, error) {
	l, _, err := tree.lookup(state, bloomStorageKey(accountHash, slotHash), func(dl *diffLayer) bool {
		_, ok := dl.states.storage(accountHash, slotHash)
		return ok
	})
	return l, err
}

// lookupNode returns the layer that is guaranteed to contain the trie node
// corresponding to the specified state root being queried, along with its
// distance from the state.
func (tree *layerTree) lookupNode(owner common.Hash, path []byte, state common.Hash) (layer, int, error) {
	return tree.lookup(state, bloomNodeKey(owner, path), func(dl *diffLayer) bool {
		_, ok := dl.nodes.node(owner, path)
		return ok
	})
}
