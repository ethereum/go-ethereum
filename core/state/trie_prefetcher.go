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

package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	// trieDeliveryMeter counts how many times the prefetcher was unable to supply
	// the statedb with a prefilled trie. This meter should be zero -- if it's not, that
	// needs to be investigated
	trieDeliveryMissMeter = metrics.NewRegisteredMeter("trie/prefetch/deliverymiss", nil)

	triePrefetchFetchMeter = metrics.NewRegisteredMeter("trie/prefetch/fetch", nil)
	triePrefetchSkipMeter  = metrics.NewRegisteredMeter("trie/prefetch/skip", nil)
	triePrefetchDropMeter  = metrics.NewRegisteredMeter("trie/prefetch/drop", nil)
)

// TriePrefetcher is an active prefetcher, which receives accounts or storage
// items on two channels, and does trie-loading of the items.
// The goal is to get as much useful content into the caches as possible
type TriePrefetcher struct {
	requestCh  chan (fetchRequest) // Chan to receive requests for data to fetch
	cmdCh      chan (*cmd)         // Chan to control activity, pause/new root
	quitCh     chan (struct{})
	deliveryCh chan (struct{})
	db         Database

	paused bool

	storageTries    map[common.Hash]Trie
	accountTrie     Trie
	accountTrieRoot common.Hash
}

func NewTriePrefetcher(db Database) *TriePrefetcher {
	return &TriePrefetcher{
		requestCh:  make(chan fetchRequest, 200),
		cmdCh:      make(chan *cmd),
		quitCh:     make(chan struct{}),
		deliveryCh: make(chan struct{}),
		db:         db,
	}
}

type cmd struct {
	root common.Hash
}

type fetchRequest struct {
	slots       []common.Hash
	storageRoot *common.Hash
	addresses   []common.Address
}

func (p *TriePrefetcher) Loop() {
	var (
		accountTrieRoot common.Hash
		accountTrie     Trie
		storageTries    map[common.Hash]Trie

		err error
		// Some tracking of performance
		skipped int64
		fetched int64

		paused = true
	)
	// The prefetcher loop has two distinct phases:
	// 1: Paused: when in this state, the accumulated tries are accessible to outside
	// callers.
	// 2: Active prefetching, awaiting slots and accounts to prefetch
	for {
		select {
		case <-p.quitCh:
			return
		case cmd := <-p.cmdCh:
			// Clear out any old requests
		drain:
			for {
				select {
				case req := <-p.requestCh:
					if req.slots != nil {
						skipped += int64(len(req.slots))
					} else {
						skipped += int64(len(req.addresses))
					}
				default:
					break drain
				}
			}
			if paused {
				// Clear old data
				p.storageTries = nil
				p.accountTrie = nil
				p.accountTrieRoot = common.Hash{}
				// Resume again
				storageTries = make(map[common.Hash]Trie)
				accountTrieRoot = cmd.root
				accountTrie, err = p.db.OpenTrie(accountTrieRoot)
				if err != nil {
					log.Error("Trie prefetcher failed opening trie", "root", accountTrieRoot, "err", err)
				}
				if accountTrieRoot == (common.Hash{}) {
					log.Error("Trie prefetcher unpaused with bad root")
				}
				paused = false
			} else {
				// Update metrics at new block events
				triePrefetchFetchMeter.Mark(fetched)
				triePrefetchSkipMeter.Mark(skipped)
				fetched, skipped = 0, 0
				// Make the tries accessible
				p.accountTrie = accountTrie
				p.storageTries = storageTries
				p.accountTrieRoot = accountTrieRoot
				if cmd.root != (common.Hash{}) {
					log.Error("Trie prefetcher paused with non-empty root")
				}
				paused = true
			}
			p.deliveryCh <- struct{}{}
		case req := <-p.requestCh:
			if paused {
				continue
			}
			if sRoot := req.storageRoot; sRoot != nil {
				// Storage slots to fetch
				var (
					storageTrie Trie
					err         error
				)
				if storageTrie = storageTries[*sRoot]; storageTrie == nil {
					if storageTrie, err = p.db.OpenTrie(*sRoot); err != nil {
						log.Warn("trie prefetcher failed opening storage trie", "root", *sRoot, "err", err)
						skipped += int64(len(req.slots))
						continue
					}
					storageTries[*sRoot] = storageTrie
				}
				for _, key := range req.slots {
					storageTrie.TryGet(key[:])
				}
				fetched += int64(len(req.slots))
			} else { // an account
				for _, addr := range req.addresses {
					accountTrie.TryGet(addr[:])
				}
				fetched += int64(len(req.addresses))
			}
		}
	}
}

// Close stops the prefetcher
func (p *TriePrefetcher) Close() {
	if p.quitCh != nil {
		close(p.quitCh)
		p.quitCh = nil
	}
}

// Resume causes the prefetcher to clear out old data, and get ready to
// fetch data concerning the new root
func (p *TriePrefetcher) Resume(root common.Hash) {
	p.paused = false
	p.cmdCh <- &cmd{
		root: root,
	}
	// Wait for it
	<-p.deliveryCh
}

// Pause causes the prefetcher to pause prefetching, and make tries
// accessible to callers via GetTrie
func (p *TriePrefetcher) Pause() {
	if p.paused {
		return
	}
	p.paused = true
	p.cmdCh <- &cmd{
		root: common.Hash{},
	}
	// Wait for it
	<-p.deliveryCh
}

// PrefetchAddresses adds an address for prefetching
func (p *TriePrefetcher) PrefetchAddresses(addresses []common.Address) {
	cmd := fetchRequest{
		addresses: addresses,
	}
	// We do an async send here, to not cause the caller to block
	//p.requestCh <- cmd
	select {
	case p.requestCh <- cmd:
	default:
		triePrefetchDropMeter.Mark(int64(len(addresses)))
	}
}

// PrefetchStorage adds a storage root and a set of keys for prefetching
func (p *TriePrefetcher) PrefetchStorage(root common.Hash, slots []common.Hash) {
	cmd := fetchRequest{
		storageRoot: &root,
		slots:       slots,
	}
	// We do an async send here, to not cause the caller to block
	//p.requestCh <- cmd
	select {
	case p.requestCh <- cmd:
	default:
		triePrefetchDropMeter.Mark(int64(len(slots)))
	}
}

// GetTrie returns the trie matching the root hash, or nil if the prefetcher
// doesn't have it.
func (p *TriePrefetcher) GetTrie(root common.Hash) Trie {
	if root == p.accountTrieRoot {
		return p.accountTrie
	}
	if storageTrie, ok := p.storageTries[root]; ok {
		// Two accounts may well have the same storage root, but we cannot allow
		// them both to make updates to the same trie instance. Therefore,
		// we need to either delete the trie now, or deliver a copy of the trie.
		delete(p.storageTries, root)
		return storageTrie
	}
	trieDeliveryMissMeter.Mark(1)
	return nil
}
