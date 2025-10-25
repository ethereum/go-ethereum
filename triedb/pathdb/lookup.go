// Copyright 2024 The go-ethereum Authors
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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"golang.org/x/sync/errgroup"
)

// trienodeShardCount is the number of shards used for trie nodes.
const trienodeShardCount = 16

// storageKey returns a key for uniquely identifying the storage slot.
func storageKey(accountHash common.Hash, slotHash common.Hash) [64]byte {
	var key [64]byte
	copy(key[:32], accountHash[:])
	copy(key[32:], slotHash[:])
	return key
}

// trienodeKey uses a fixed-size byte array instead of string to avoid string allocations.
type trienodeKey [96]byte // 32 bytes for hash + up to 64 bytes for path

// makeTrienodeKey returns a key for uniquely identifying the trie node.
func makeTrienodeKey(accountHash common.Hash, path string) trienodeKey {
	var key trienodeKey
	copy(key[:32], accountHash[:])
	copy(key[32:], path)
	return key
}

// shardTask used to batch task by shard to minimize lock contention
type shardTask struct {
	accountHash common.Hash
	path        string
}

// lookup is an internal structure used to efficiently determine the layer in
// which a state entry resides.
type lookup struct {
	// accounts represents the mutation history for specific accounts.
	// The key is the account address hash, and the value is a slice
	// of **diff layer** IDs indicating where the account was modified,
	// with the order from oldest to newest.
	accounts map[common.Hash][]common.Hash

	// storages represents the mutation history for specific storage
	// slot. The key is the account address hash and the storage key
	// hash, the value is a slice of **diff layer** IDs indicating
	// where the slot was modified, with the order from oldest to newest.
	storages map[[64]byte][]common.Hash

	// accountNodes represents the mutation history for specific account
	// trie nodes, distributed across 16 shards for efficiency.
	// The key is the trie path of the node, and the value is a slice
	// of **diff layer** IDs indicating where the account was modified,
	// with the order from oldest to newest.
	accountNodes [trienodeShardCount]map[string][]common.Hash

	// storageNodes represents the mutation history for specific storage
	// slot trie nodes, distributed across 16 shards for efficiency.
	// The key is the account address hash and the trie path of the node,
	// the value is a slice of **diff layer** IDs indicating where the
	// slot was modified, with the order from oldest to newest.
	storageNodes [trienodeShardCount]map[trienodeKey][]common.Hash

	// descendant is the callback indicating whether the layer with
	// given root is a descendant of the one specified by `ancestor`.
	descendant func(state common.Hash, ancestor common.Hash) bool
}

// getNodeShardIndex returns the shard index for a given path
func getNodeShardIndex(path string) int {
	if len(path) == 0 {
		return 0
	}
	// use the first char of the path to determine the shard index
	return int(path[0]) % trienodeShardCount
}

// newLookup initializes the lookup structure.
func newLookup(head layer, descendant func(state common.Hash, ancestor common.Hash) bool) *lookup {
	var (
		current = head
		layers  []layer
	)
	for current != nil {
		layers = append(layers, current)
		current = current.parentLayer()
	}
	l := &lookup{
		accounts:   make(map[common.Hash][]common.Hash),
		storages:   make(map[[64]byte][]common.Hash),
		descendant: descendant,
	}
	// Initialize all 16 storage node shards
	for i := 0; i < trienodeShardCount; i++ {
		l.storageNodes[i] = make(map[trienodeKey][]common.Hash)
		l.accountNodes[i] = make(map[string][]common.Hash)
	}

	// Apply the diff layers from bottom to top
	for i := len(layers) - 1; i >= 0; i-- {
		switch diff := layers[i].(type) {
		case *diskLayer:
			continue
		case *diffLayer:
			l.addLayer(diff)
		}
	}
	return l
}

// accountTip traverses the layer list associated with the given account in
// reverse order to locate the first entry that either matches the specified
// stateID or is a descendant of it.
//
// If found, the account data corresponding to the supplied stateID resides
// in that layer. Otherwise, two scenarios are possible:
//
// (a) the account remains unmodified from the current disk layer up to the state
// layer specified by the stateID: fallback to the disk layer for data retrieval,
// (b) or the layer specified by the stateID is stale: reject the data retrieval.
func (l *lookup) accountTip(accountHash common.Hash, stateID common.Hash, base common.Hash) common.Hash {
	// Traverse the mutation history from latest to oldest one. Several
	// scenarios are possible:
	//
	// Chain:
	//     D->C1->C2->C3->C4 (HEAD)
	//      ->C1'->C2'->C3'
	// State:
	//     x: [C1, C1', C3', C3]
	//     y: []
	//
	// - (x, C4) => C3
	// - (x, C3) => C3
	// - (x, C2) => C1
	// - (x, C3') => C3'
	// - (x, C2') => C1'
	// - (y, C4) => D
	// - (y, C3') => D
	// - (y, C0) => null
	list := l.accounts[accountHash]
	for i := len(list) - 1; i >= 0; i-- {
		// If the current state matches the stateID, or the requested state is a
		// descendant of it, return the current state as the most recent one
		// containing the modified data. Otherwise, the current state may be ahead
		// of the requested one or belong to a different branch.
		if list[i] == stateID || l.descendant(stateID, list[i]) {
			return list[i]
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}
}

// storageTip traverses the layer list associated with the given account and
// slot hash in reverse order to locate the first entry that either matches
// the specified stateID or is a descendant of it.
//
// If found, the storage data corresponding to the supplied stateID resides
// in that layer. Otherwise, two scenarios are possible:
//
// (a) the storage slot remains unmodified from the current disk layer up to
// the state layer specified by the stateID: fallback to the disk layer for
// data retrieval, (b) or the layer specified by the stateID is stale: reject
// the data retrieval.
func (l *lookup) storageTip(accountHash common.Hash, slotHash common.Hash, stateID common.Hash, base common.Hash) common.Hash {
	list := l.storages[storageKey(accountHash, slotHash)]
	for i := len(list) - 1; i >= 0; i-- {
		// If the current state matches the stateID, or the requested state is a
		// descendant of it, return the current state as the most recent one
		// containing the modified data. Otherwise, the current state may be ahead
		// of the requested one or belong to a different branch.
		if list[i] == stateID || l.descendant(stateID, list[i]) {
			return list[i]
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}
}

// nodeTip traverses the layer list associated with the given account and path
// in reverse order to locate the first entry that either matches
// the specified stateID or is a descendant of it.
//
// If found, the trie node data corresponding to the supplied stateID resides
// in that layer. Otherwise, two scenarios are possible:
//
// (a) the trie node remains unmodified from the current disk layer up to
// the state layer specified by the stateID: fallback to the disk layer for
// data retrieval, (b) or the layer specified by the stateID is stale: reject
// the data retrieval.
func (l *lookup) nodeTip(accountHash common.Hash, path string, stateID common.Hash, base common.Hash) common.Hash {
	var list []common.Hash
	if accountHash == (common.Hash{}) {
		shardIndex := getNodeShardIndex(path)
		list = l.accountNodes[shardIndex][path]
	} else {
		shardIndex := getNodeShardIndex(path) // Use only path for sharding
		list = l.storageNodes[shardIndex][makeTrienodeKey(accountHash, path)]
	}
	for i := len(list) - 1; i >= 0; i-- {
		// If the current state matches the stateID, or the requested state is a
		// descendant of it, return the current state as the most recent one
		// containing the modified data. Otherwise, the current state may be ahead
		// of the requested one or belong to a different branch.
		if list[i] == stateID || l.descendant(stateID, list[i]) {
			return list[i]
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}
}

// addLayer traverses the state data retained in the specified diff layer and
// integrates it into the lookup set.
//
// This function assumes that all layers older than the provided one have already
// been processed, ensuring that layers are processed strictly in a bottom-to-top
// order.
func (l *lookup) addLayer(diff *diffLayer) {
	defer func(now time.Time) {
		lookupAddLayerTimer.UpdateSince(now)
		log.Debug("PathDB lookup add layer", "id", diff.id, "block", diff.block, "elapsed", time.Since(now))
	}(time.Now())

	var (
		wg    sync.WaitGroup
		state = diff.rootHash()
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for accountHash := range diff.states.accountData {
			list, exists := l.accounts[accountHash]
			if !exists {
				list = make([]common.Hash, 0, 16) // TODO(rjl493456442) use sync pool
			}
			list = append(list, state)
			l.accounts[accountHash] = list
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for accountHash, slots := range diff.states.storageData {
			for slotHash := range slots {
				key := storageKey(accountHash, slotHash)
				list, exists := l.storages[key]
				if !exists {
					list = make([]common.Hash, 0, 16) // TODO(rjl493456442) use sync pool
				}
				list = append(list, state)
				l.storages[key] = list
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.addAccountNodes(state, diff.nodes.accountNodes)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.addStorageNodes(state, diff.nodes.storageNodes)
	}()

	states := len(diff.states.accountData)
	for _, slots := range diff.states.storageData {
		states += len(slots)
	}
	lookupStateMeter.Mark(int64(states))

	trienodes := len(diff.nodes.accountNodes)
	for _, nodes := range diff.nodes.storageNodes {
		trienodes += len(nodes)
	}
	lookupTrienodeMeter.Mark(int64(trienodes))

	wg.Wait()
}

func (l *lookup) addStorageNodes(state common.Hash, nodes map[common.Hash]map[string]*trienode.Node) {
	defer func(start time.Time) {
		lookupAddTrienodeLayerTimer.UpdateSince(start)
	}(time.Now())

	var (
		wg    sync.WaitGroup
		tasks = make([][]shardTask, trienodeShardCount)
	)
	for accountHash, slots := range nodes {
		for path := range slots {
			shardIndex := getNodeShardIndex(path)
			tasks[shardIndex] = append(tasks[shardIndex], shardTask{
				accountHash: accountHash,
				path:        path,
			})
		}
	}
	for shardIdx := 0; shardIdx < trienodeShardCount; shardIdx++ {
		taskList := tasks[shardIdx]
		if len(taskList) == 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			shard := l.storageNodes[shardIdx]
			for _, task := range taskList {
				key := makeTrienodeKey(task.accountHash, task.path)
				shard[key] = append(shard[key], state)
			}
		}()
	}
	wg.Wait()
}

func (l *lookup) addAccountNodes(state common.Hash, nodes map[string]*trienode.Node) {
	defer func(start time.Time) {
		lookupAddTrienodeLayerTimer.UpdateSince(start)
	}(time.Now())

	var (
		wg    sync.WaitGroup
		tasks = make([][]string, trienodeShardCount)
	)
	for path := range nodes {
		shardIndex := getNodeShardIndex(path)
		tasks[shardIndex] = append(tasks[shardIndex], path)
	}
	for shardIdx := 0; shardIdx < trienodeShardCount; shardIdx++ {
		taskList := tasks[shardIdx]
		if len(taskList) == 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			shard := l.accountNodes[shardIdx]
			for _, path := range taskList {
				shard[path] = append(shard[path], state)
			}
		}()
	}
	wg.Wait()
}

// removeFromList removes the specified element from the provided list.
// It returns a flag indicating whether the element was found and removed.
func removeFromList(list []common.Hash, element common.Hash) (bool, []common.Hash) {
	// Traverse the list from oldest to newest to quickly locate the element.
	for i := 0; i < len(list); i++ {
		if list[i] == element {
			if i != 0 {
				list = append(list[:i], list[i+1:]...)
			} else {
				// Remove the first element by shifting the slice forward.
				// Pros: zero-copy.
				// Cons: may retain large backing array, causing memory leaks.
				// Mitigation: release the array if capacity exceeds threshold.
				list = list[1:]
				if cap(list) > 1024 {
					list = append(make([]common.Hash, 0, len(list)), list...)
				}
			}
			return true, list
		}
	}
	return false, nil
}

// removeLayer traverses the state data retained in the specified diff layer and
// unlink them from the lookup set.
func (l *lookup) removeLayer(diff *diffLayer) error {
	defer func(now time.Time) {
		lookupRemoveLayerTimer.UpdateSince(now)
		log.Debug("PathDB lookup remove layer", "id", diff.id, "block", diff.block, "elapsed", time.Since(now))
	}(time.Now())

	var (
		eg    errgroup.Group
		state = diff.rootHash()
	)
	eg.Go(func() error {
		for accountHash := range diff.states.accountData {
			found, list := removeFromList(l.accounts[accountHash], state)
			if !found {
				return fmt.Errorf("account lookup is not found, %x, state: %x", accountHash, state)
			}
			if len(list) != 0 {
				l.accounts[accountHash] = list
			} else {
				delete(l.accounts, accountHash)
			}
		}
		return nil
	})

	eg.Go(func() error {
		for accountHash, slots := range diff.states.storageData {
			for slotHash := range slots {
				key := storageKey(accountHash, slotHash)
				found, list := removeFromList(l.storages[key], state)
				if !found {
					return fmt.Errorf("storage lookup is not found, %x %x, state: %x", accountHash, slotHash, state)
				}
				if len(list) != 0 {
					l.storages[key] = list
				} else {
					delete(l.storages, key)
				}
			}
		}
		return nil
	})

	eg.Go(func() error {
		return l.removeAccountNodes(state, diff.nodes.accountNodes)
	})

	eg.Go(func() error {
		return l.removeStorageNodes(state, diff.nodes.storageNodes)
	})
	return eg.Wait()
}

func (l *lookup) removeStorageNodes(state common.Hash, nodes map[common.Hash]map[string]*trienode.Node) error {
	defer func(start time.Time) {
		lookupRemoveTrienodeLayerTimer.UpdateSince(start)
	}(time.Now())

	var (
		eg    errgroup.Group
		tasks = make([][]shardTask, trienodeShardCount)
	)
	for accountHash, slots := range nodes {
		for path := range slots {
			shardIndex := getNodeShardIndex(path)
			tasks[shardIndex] = append(tasks[shardIndex], shardTask{
				accountHash: accountHash,
				path:        path,
			})
		}
	}
	for shardIdx := 0; shardIdx < trienodeShardCount; shardIdx++ {
		taskList := tasks[shardIdx]
		if len(taskList) == 0 {
			continue
		}
		eg.Go(func() error {
			shard := l.storageNodes[shardIdx]
			for _, task := range taskList {
				key := makeTrienodeKey(task.accountHash, task.path)
				found, list := removeFromList(shard[key], state)
				if !found {
					return fmt.Errorf("storage lookup is not found, key: %x, state: %x", key, state)
				}
				if len(list) != 0 {
					shard[key] = list
				} else {
					delete(shard, key)
				}
			}
			return nil
		})
	}
	return eg.Wait()
}

func (l *lookup) removeAccountNodes(state common.Hash, nodes map[string]*trienode.Node) error {
	defer func(start time.Time) {
		lookupRemoveTrienodeLayerTimer.UpdateSince(start)
	}(time.Now())

	var (
		eg    errgroup.Group
		tasks = make([][]string, trienodeShardCount)
	)
	for path := range nodes {
		shardIndex := getNodeShardIndex(path)
		tasks[shardIndex] = append(tasks[shardIndex], path)
	}
	for shardIdx := 0; shardIdx < trienodeShardCount; shardIdx++ {
		taskList := tasks[shardIdx]
		if len(taskList) == 0 {
			continue
		}
		eg.Go(func() error {
			shard := l.accountNodes[shardIdx]
			for _, path := range taskList {
				found, list := removeFromList(shard[path], state)
				if !found {
					return fmt.Errorf("account lookup is not found, %x, state: %x", path, state)
				}
				if len(list) != 0 {
					shard[path] = list
				} else {
					delete(shard, path)
				}
			}
			return nil
		})
	}
	return eg.Wait()
}
