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
	storages map[common.Hash]map[common.Hash][]common.Hash

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
		accounts:   make(map[common.Hash][]common.Hash),
		storages:   make(map[common.Hash]map[common.Hash][]common.Hash),
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
	list := l.accounts[accountHash]
	for i := len(list) - 1; i >= 0; i-- {
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
	subset, exists := l.storages[accountHash]
	if exists {
		list := subset[slotHash]
		for i := len(list) - 1; i >= 0; i-- {
			if list[i] == stateID || l.descendant(stateID, list[i]) {
				return list[i]
			}
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
			subset := l.storages[accountHash]
			if subset == nil {
				subset = make(map[common.Hash][]common.Hash)
				l.storages[accountHash] = subset
			}
			for slotHash := range slots {
				list, exists := subset[slotHash]
				if !exists {
					list = make([]common.Hash, 0, 16) // TODO(rjl493456442) use sync pool
				}
				list = append(list, state)
				subset[slotHash] = list
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
		wg    errgroup.Group
		state = diff.rootHash()
	)
	wg.Go(func() error {
		for accountHash := range diff.states.accountData {
			var (
				found bool
				list  = l.accounts[accountHash]
			)
			// Traverse the list from oldest to newest to quickly locate the ID
			// of the stale layer.
			for i := 0; i < len(list); i++ {
				if list[i] == state {
					if i == 0 {
						list = list[1:]
						if cap(list) > 1024 {
							list = append(make([]common.Hash, 0, len(list)), list...)
						}
					} else {
						list = append(list[:i], list[i+1:]...)
					}
					found = true
					break
				}
			}
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

	wg.Go(func() error {
		for accountHash, slots := range diff.states.storageData {
			subset := l.storages[accountHash]
			if subset == nil {
				return fmt.Errorf("storage lookup is not found, %x", accountHash)
			}
			for slotHash := range slots {
				var (
					found bool
					list  = subset[slotHash]
				)
				// Traverse the list from oldest to newest to quickly locate the ID
				// of the stale layer.
				for i := 0; i < len(list); i++ {
					if list[i] == state {
						if i == 0 {
							list = list[1:]
							if cap(list) > 1024 {
								list = append(make([]common.Hash, 0, len(list)), list...)
							}
						} else {
							list = append(list[:i], list[i+1:]...)
						}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("storage lookup is not found, %x %x, state: %x", accountHash, slotHash, state)
				}
				if len(list) != 0 {
					subset[slotHash] = list
				} else {
					delete(subset, slotHash)
				}
			}
			if len(subset) == 0 {
				delete(l.storages, accountHash)
			}
		}
		return nil
	})
	return wg.Wait()
}
