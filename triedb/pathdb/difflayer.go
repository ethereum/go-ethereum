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
)

// diffLayer represents a collection of modifications made to the in-memory tries
// along with associated state changes after running a block on top.
//
// The purpose of a diff layer is to serve as a journal, recording recent state modifications
// that have not yet been committed to a more stable or semi-permanent state.
type diffLayer struct {
	// Immutables
	root   common.Hash         // Root hash to which this layer diff belongs to
	id     uint64              // Corresponding state id
	block  uint64              // Associated block number
	nodes  *nodeSet            // Cached trie nodes indexed by owner and path
	states *StateSetWithOrigin // Associated state changes along with origin value

	parent layer        // Parent layer modified by this one, never nil, **can be changed**
	lock   sync.RWMutex // Lock used to protect parent
}

// newDiffLayer creates a new diff layer on top of an existing layer.
func newDiffLayer(parent layer, root common.Hash, id uint64, block uint64, nodes *nodeSet, states *StateSetWithOrigin) *diffLayer {
	dl := &diffLayer{
		root:   root,
		id:     id,
		block:  block,
		parent: parent,
		nodes:  nodes,
		states: states,
	}
	dirtyNodeWriteMeter.Mark(int64(nodes.size))
	dirtyStateWriteMeter.Mark(int64(states.size))
	log.Debug("Created new diff layer", "id", id, "block", block, "nodesize", common.StorageSize(nodes.size), "statesize", common.StorageSize(states.size))
	return dl
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
func (dl *diffLayer) node(owner common.Hash, path []byte, depth int) ([]byte, common.Hash, *nodeLoc, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the trie node is known locally, return it
	n, ok := dl.nodes.node(owner, path)
	if ok {
		dirtyNodeHitMeter.Mark(1)
		dirtyNodeHitDepthHist.Update(int64(depth))
		dirtyNodeReadMeter.Mark(int64(len(n.Blob)))
		return n.Blob, n.Hash, &nodeLoc{loc: locDiffLayer, depth: depth}, nil
	}
	// Trie node unknown to this layer, resolve from parent
	return dl.parent.node(owner, path, depth+1)
}

// account directly retrieves the account RLP associated with a particular
// hash in the slim data format.
//
// Note the returned account is not a copy, please don't modify it.
func (dl *diffLayer) account(hash common.Hash, depth int) ([]byte, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if blob, found := dl.states.account(hash); found {
		dirtyStateHitMeter.Mark(1)
		dirtyStateHitDepthHist.Update(int64(depth))
		dirtyStateReadMeter.Mark(int64(len(blob)))

		if len(blob) == 0 {
			stateAccountInexMeter.Mark(1)
		} else {
			stateAccountExistMeter.Mark(1)
		}
		return blob, nil
	}
	// Account is unknown to this layer, resolve from parent
	return dl.parent.account(hash, depth+1)
}

// storage directly retrieves the storage data associated with a particular hash,
// within a particular account.
//
// Note the returned storage slot is not a copy, please don't modify it.
func (dl *diffLayer) storage(accountHash, storageHash common.Hash, depth int) ([]byte, error) {
	// Hold the lock, ensure the parent won't be changed during the
	// state accessing.
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if blob, found := dl.states.storage(accountHash, storageHash); found {
		dirtyStateHitMeter.Mark(1)
		dirtyStateHitDepthHist.Update(int64(depth))
		dirtyStateReadMeter.Mark(int64(len(blob)))

		if len(blob) == 0 {
			stateStorageInexMeter.Mark(1)
		} else {
			stateStorageExistMeter.Mark(1)
		}
		return blob, nil
	}
	// storage slot is unknown to this layer, resolve from parent
	return dl.parent.storage(accountHash, storageHash, depth+1)
}

// update implements the layer interface, creating a new layer on top of the
// existing layer tree with the specified data items.
func (dl *diffLayer) update(root common.Hash, id uint64, block uint64, nodes *nodeSet, states *StateSetWithOrigin) *diffLayer {
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

// size returns the approximate memory size occupied by this diff layer.
func (dl *diffLayer) size() uint64 {
	return dl.nodes.size + dl.states.size
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
