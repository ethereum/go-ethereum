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
	"golang.org/x/sync/errgroup"
)

// storageKey returns a key for uniquely identifying the storage slot.
func storageKey(accountHash common.Hash, slotHash common.Hash) [64]byte {
	var key [64]byte
	copy(key[:32], accountHash[:])
	copy(key[32:], slotHash[:])
	return key
}

// storageKeySlice returns a key for uniquely identifying the storage slot in
// the slice format.
func storageKeySlice(accountHash common.Hash, slotHash common.Hash) []byte {
	key := storageKey(accountHash, slotHash)
	return key[:]
}

// layerList records the set of diff-layer roots that modified a single state
// entry, ordered from oldest to newest.
//
// The overwhelmingly common case is a key touched by exactly one in-memory
// diff layer; that single root is stored inline in head with rest left nil,
// so no backing slice is allocated and the value lives entirely inside the
// map bucket. Only the rare hot keys (e.g. fee recipient, popular contracts)
// modified across multiple layers spill into rest.
//
// Note head may legitimately be the zero hash (e.g. the empty bintrie root),
// so callers must rely on the map's presence flag rather than comparing head
// against common.Hash{} to detect an absent entry.
type layerList struct {
	head common.Hash   // the oldest layer; the only one in the common case
	rest []common.Hash // additional layers (newest at the tail), nil when len==1
}

// lookup is an internal structure used to efficiently determine the layer in
// which a state entry resides.
type lookup struct {
	// accounts represents the mutation history for specific accounts.
	// The key is the account address hash, and the value is the list
	// of **diff layer** IDs indicating where the account was modified,
	// with the order from oldest to newest.
	accounts map[common.Hash]layerList

	// storages represents the mutation history for specific storage
	// slot. The key is the account address hash and the storage key
	// hash, the value is the list of **diff layer** IDs indicating
	// where the slot was modified, with the order from oldest to newest.
	storages map[[64]byte]layerList

	// descendant is the callback indicating whether the layer with
	// given root is a descendant of the one specified by `ancestor`.
	descendant func(state common.Hash, ancestor common.Hash) bool
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
		accounts:   make(map[common.Hash]layerList),
		storages:   make(map[[64]byte]layerList),
		descendant: descendant,
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

// tip scans the layer list from newest to oldest and returns the first entry
// that either matches the supplied stateID or is a descendant of it. It returns
// (common.Hash{}, false) when no such entry exists.
func (list layerList) tip(stateID common.Hash, descendant func(state common.Hash, ancestor common.Hash) bool) (common.Hash, bool) {
	for i := len(list.rest) - 1; i >= 0; i-- {
		if list.rest[i] == stateID || descendant(stateID, list.rest[i]) {
			return list.rest[i], true
		}
	}
	if list.head == stateID || descendant(stateID, list.head) {
		return list.head, true
	}
	return common.Hash{}, false
}

// add appends the given layer root as the newest entry of the list.
func (list layerList) add(state common.Hash) layerList {
	list.rest = append(list.rest, state)
	return list
}

// remove unlinks the given layer root from the list. It returns the updated
// list, a flag indicating whether the list became empty, and a flag indicating
// whether the element was found and removed.
func (list layerList) remove(state common.Hash) (layerList, bool, bool) {
	// The newest layers are flattened into the disk layer from the bottom up,
	// so removals almost always target the oldest entry held in head.
	if list.head == state {
		if len(list.rest) == 0 {
			return list, true, true
		}
		list.head, list.rest = list.rest[0], list.rest[1:]
		// Release the backing array if it has grown excessively, otherwise the
		// re-slicing above would pin the whole array and leak memory.
		if cap(list.rest) > 1024 {
			list.rest = append(make([]common.Hash, 0, len(list.rest)), list.rest...)
		}
		return list, false, true
	}
	for i := 0; i < len(list.rest); i++ {
		if list.rest[i] == state {
			list.rest = append(list.rest[:i], list.rest[i+1:]...)
			return list, false, true
		}
	}
	return list, false, false
}

// accountTip traverses the layer list associated with the given account in
// reverse order to locate the first entry that either matches the specified
// stateID or is a descendant of it.
//
// If found, the account data corresponding to the supplied stateID resides
// in the layer identified by the returned hash (ok=true). Otherwise,
// (common.Hash{}, false) is returned to signal that the supplied stateID is
// stale.
//
// Note the returned hash may itself be common.Hash{} when the disk layer's
// root is zero — as is the case for a fresh verkle/bintrie database whose
// empty trie hashes to EmptyVerkleHash. Callers must therefore consult the
// boolean rather than comparing the returned hash against common.Hash{}
// directly.
func (l *lookup) accountTip(accountHash common.Hash, stateID common.Hash, base common.Hash) (common.Hash, bool) {
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
	// Traverse the mutation history (if any) from latest to oldest, returning
	// the most recent layer that contains the requested data.
	if list, ok := l.accounts[accountHash]; ok {
		if tip, found := list.tip(stateID, l.descendant); found {
			return tip, true
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base, true
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}, false
}

// storageTip traverses the layer list associated with the given account and
// slot hash in reverse order to locate the first entry that either matches
// the specified stateID or is a descendant of it.
//
// See accountTip for the returned-hash / ok convention — the same
// bintrie-zero-root caveat applies here.
func (l *lookup) storageTip(accountHash common.Hash, slotHash common.Hash, stateID common.Hash, base common.Hash) (common.Hash, bool) {
	// Traverse the mutation history (if any) from latest to oldest, returning
	// the most recent layer that contains the requested data.
	if list, ok := l.storages[storageKey(accountHash, slotHash)]; ok {
		if tip, found := list.tip(stateID, l.descendant); found {
			return tip, true
		}
	}
	// No layer matching the stateID or its descendants was found. Use the
	// current disk layer as a fallback.
	if base == stateID || l.descendant(stateID, base) {
		return base, true
	}
	// The layer associated with 'stateID' is not the descendant of the current
	// disk layer, it's already stale, return nothing.
	return common.Hash{}, false
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
	}(time.Now())

	var (
		wg    sync.WaitGroup
		state = diff.rootHash()
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for accountHash := range diff.states.accountData {
			// The common case is a key touched by a single layer, where the
			// zero-value layerList already holds the root inline in head; only
			// hot keys modified across layers allocate a backing slice.
			list, exists := l.accounts[accountHash]
			if !exists {
				l.accounts[accountHash] = layerList{head: state}
				continue
			}
			l.accounts[accountHash] = list.add(state)
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
					l.storages[key] = layerList{head: state}
					continue
				}
				l.storages[key] = list.add(state)
			}
		}
	}()
	wg.Wait()
}

// removeLayer traverses the state data retained in the specified diff layer and
// unlink them from the lookup set.
func (l *lookup) removeLayer(diff *diffLayer) error {
	defer func(now time.Time) {
		lookupRemoveLayerTimer.UpdateSince(now)
	}(time.Now())

	var (
		eg    errgroup.Group
		state = diff.rootHash()
	)
	eg.Go(func() error {
		for accountHash := range diff.states.accountData {
			list, empty, found := l.accounts[accountHash].remove(state)
			if !found {
				return fmt.Errorf("account lookup is not found, %x, state: %x", accountHash, state)
			}
			if empty {
				delete(l.accounts, accountHash)
			} else {
				l.accounts[accountHash] = list
			}
		}
		return nil
	})

	eg.Go(func() error {
		for accountHash, slots := range diff.states.storageData {
			for slotHash := range slots {
				key := storageKey(accountHash, slotHash)
				list, empty, found := l.storages[key].remove(state)
				if !found {
					return fmt.Errorf("storage lookup is not found, %x %x, state: %x", accountHash, slotHash, state)
				}
				if empty {
					delete(l.storages, key)
				} else {
					l.storages[key] = list
				}
			}
		}
		return nil
	})
	return eg.Wait()
}
