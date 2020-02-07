// Copyright 2020 The go-ethereum Authors
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

package core

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	triePrefetchFetchMeter = metrics.NewRegisteredMeter("trie/prefetch/fetch", nil)
	triePrefetchSkipMeter  = metrics.NewRegisteredMeter("trie/prefetch/skip", nil)
	triePrefetchDropMeter  = metrics.NewRegisteredMeter("trie/prefetch/drop", nil)
)

// triePrefetcher is an active prefetcher, which receives accounts or storage
// items on two channels, and does trie-loading of the items.
// The goal is to get as much useful content into the caches as possible
type triePrefetcher struct {
	cmdCh   chan (command)
	abortCh chan (struct{})
	db      state.Database
	stale   uint64
}

func newTriePrefetcher(db state.Database) *triePrefetcher {
	return &triePrefetcher{
		cmdCh:   make(chan command, 200),
		abortCh: make(chan struct{}),
		db:      db,
	}
}

type command struct {
	root    *common.Hash
	address *common.Address
	slots   []common.Hash
}

func (p *triePrefetcher) loop() {
	var (
		tr          state.Trie
		err         error
		currentRoot common.Hash
		// Some tracking of performance
		skipped int64
		fetched int64
	)
	for {
		select {
		case cmd := <-p.cmdCh:
			// New roots are sent synchoronously
			if cmd.root != nil && cmd.slots == nil {
				// Update metrics at new block events
				triePrefetchFetchMeter.Mark(fetched)
				fetched = 0
				triePrefetchSkipMeter.Mark(skipped)
				skipped = 0
				// New root and number
				currentRoot = *cmd.root
				tr, err = p.db.OpenTrie(currentRoot)
				if err != nil {
					log.Warn("trie prefetcher failed opening trie", "root", currentRoot, "err", err)
				}
				// Open for business again
				atomic.StoreUint64(&p.stale, 0)
				continue
			}
			// Don't get stuck precaching on old blocks
			if atomic.LoadUint64(&p.stale) == 1 {
				if nSlots := len(cmd.slots); nSlots > 0 {
					skipped += int64(nSlots)
				} else {
					skipped++
				}
				// Keep reading until we're in step with the chain
				continue
			}
			// It's either storage slots or an account
			if cmd.slots != nil {
				storageTrie, err := p.db.OpenTrie(*cmd.root)
				if err != nil {
					log.Warn("trie prefetcher failed opening storage trie", "root", *cmd.root, "err", err)
					skipped += int64(len(cmd.slots))
					continue
				}
				for i, key := range cmd.slots {
					storageTrie.TryGet(key[:])
					fetched++
					// Abort if we fall behind
					if atomic.LoadUint64(&p.stale) == 1 {
						skipped += int64(len(cmd.slots[i:]))
						break
					}
				}
			} else { // an account
				if tr == nil {
					skipped++
					continue
				}
				// We're in sync with the chain, do preloading
				if cmd.address != nil {
					fetched++
					addr := *cmd.address
					tr.TryGet(addr[:])
				}
			}
		case <-p.abortCh:
			return
		}
	}
}

// Close stops the prefetcher
func (p *triePrefetcher) Close() {
	p.abortCh <- struct{}{}
}

// Reset prevent the prefetcher from entering a state where it is
// behind the actual block processing.
// It causes any existing (stale) work to be ignored, and the prefetcher will skip ahead
// to current tasks
func (p *triePrefetcher) Reset(number uint64, root common.Hash) {
	// Set staleness
	atomic.StoreUint64(&p.stale, 1)
	// Do a synced send, so we're sure it punches through any old (now stale) commands
	cmd := command{
		root: &root,
	}
	p.cmdCh <- cmd
}

func (p *triePrefetcher) Pause() {
	// Set staleness
	atomic.StoreUint64(&p.stale, 1)
}

// PrefetchAddress adds an address for prefetching
func (p *triePrefetcher) PrefetchAddress(addr common.Address) {
	cmd := command{
		address: &addr,
	}
	// We do an async send here, to not cause the caller to block
	select {
	case p.cmdCh <- cmd:
	default:
		triePrefetchDropMeter.Mark(1)
	}
}

// PrefetchStorage adds a storage root and a set of keys for prefetching
func (p *triePrefetcher) PrefetchStorage(root common.Hash, slots []common.Hash) {
	cmd := command{
		root:  &root,
		slots: slots,
	}
	// We do an async send here, to not cause the caller to block
	select {
	case p.cmdCh <- cmd:
	default:
		triePrefetchDropMeter.Mark(int64(len(slots)))
	}

}
