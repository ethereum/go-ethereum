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
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/errgroup"
)

// lookup is an internal help structure to quickly identify
type lookup struct {
	nodes      map[common.Hash]map[string][]common.Hash
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
	l := new(lookup)
	l.nodes = make(map[common.Hash]map[string][]common.Hash)
	l.descendant = descendant

	// Apply the layers from bottom to top
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

// nodeTip returns the first state entry that either matches the specified head
// or is a descendant of it. If all the entries are not qualified, empty hash
// is returned.
func (l *lookup) nodeTip(owner common.Hash, path []byte, head common.Hash) common.Hash {
	subset, exists := l.nodes[owner]
	if !exists {
		return common.Hash{}
	}
	list := subset[string(path)]

	// Traverse the list in reverse order to find the first entry that either
	// matches the specified head or is a descendant of it.
	for i := len(list) - 1; i >= 0; i-- {
		if list[i] == head || l.descendant(head, list[i]) {
			return list[i]
		}
	}
	return common.Hash{}
}

// addLayer traverses all the dirty nodes within the given diff layer and links
// them into the lookup set.
func (l *lookup) addLayer(diff *diffLayer) {
	defer func(now time.Time) {
		lookupAddLayerTimer.UpdateSince(now)
	}(time.Now())

	var (
		state   = diff.rootHash()
		lock    sync.Mutex
		workers errgroup.Group
	)
	workers.SetLimit(runtime.NumCPU() / 2)

	for accountHash, nodes := range diff.nodes {
		accountHash, nodes := accountHash, nodes // closure

		workers.Go(func() error {
			lock.Lock()
			subset := l.nodes[accountHash]
			if subset == nil {
				subset = make(map[string][]common.Hash)
				l.nodes[accountHash] = subset
			}
			lock.Unlock()

			// Put the layer hash at the end of the list
			for path := range nodes {
				subset[path] = append(subset[path], state)
			}
			return nil
		})
	}
	workers.Wait()
}

// removeLayer traverses all the dirty nodes within the given diff layer and
// unlinks them from the lookup set.
func (l *lookup) removeLayer(diff *diffLayer) error {
	defer func(now time.Time) {
		lookupRemoveLayerTimer.UpdateSince(now)
	}(time.Now())

	var (
		state   = diff.rootHash()
		lock    sync.RWMutex
		workers errgroup.Group
	)
	workers.SetLimit(runtime.NumCPU() / 2)

	for accountHash, nodes := range diff.nodes {
		accountHash, nodes := accountHash, nodes // closure

		workers.Go(func() error {
			lock.RLock()
			subset := l.nodes[accountHash]
			if subset == nil {
				lock.RUnlock()
				return fmt.Errorf("unknown node owner %x", accountHash)
			}
			lock.RUnlock()

			// Traverse the list from oldest to newest to quickly locate the ID
			// of the stale layer.
			for path := range nodes {
				var found bool
				for j := 0; j < len(subset[path]); j++ {
					if subset[path][j] == state {
						if j == 0 {
							subset[path] = subset[path][1:] // TODO what if the underlying slice is held forever?
						} else {
							subset[path] = append(subset[path][:j], subset[path][j+1:]...)
						}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("failed to delete lookup %x %v", accountHash, []byte(path))
				}
				if len(subset[path]) == 0 {
					delete(subset, path)
				}
			}
			if len(subset) == 0 {
				lock.Lock()
				delete(l.nodes, accountHash)
				lock.Unlock()
			}
			return nil
		})
	}
	return workers.Wait()
}
