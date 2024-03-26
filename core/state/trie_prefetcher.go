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
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	// triePrefetchMetricsPrefix is the prefix under which to publish the metrics.
	triePrefetchMetricsPrefix = "trie/prefetch/"

	// errTerminated is returned if any invocation is applied on a terminated fetcher.
	errTerminated = errors.New("fetcher is already terminated")
)

// triePrefetcher is an active prefetcher, which receives accounts or storage
// items and does trie-loading of them. The goal is to get as much useful content
// into the caches as possible.
//
// Note, the prefetcher's API is not thread safe.
type triePrefetcher struct {
	db       Database               // Database to fetch trie nodes through
	root     common.Hash            // Root hash of the account trie for metrics
	fetchers map[string]*subfetcher // Subfetchers for each trie
	closed   bool

	deliveryMissMeter metrics.Meter
	accountLoadMeter  metrics.Meter
	accountDupMeter   metrics.Meter
	accountWasteMeter metrics.Meter
	storageLoadMeter  metrics.Meter
	storageDupMeter   metrics.Meter
	storageWasteMeter metrics.Meter
}

func newTriePrefetcher(db Database, root common.Hash, namespace string) *triePrefetcher {
	prefix := triePrefetchMetricsPrefix + namespace
	return &triePrefetcher{
		db:       db,
		root:     root,
		fetchers: make(map[string]*subfetcher), // Active prefetchers use the fetchers map

		deliveryMissMeter: metrics.GetOrRegisterMeter(prefix+"/deliverymiss", nil),
		accountLoadMeter:  metrics.GetOrRegisterMeter(prefix+"/account/load", nil),
		accountDupMeter:   metrics.GetOrRegisterMeter(prefix+"/account/dup", nil),
		accountWasteMeter: metrics.GetOrRegisterMeter(prefix+"/account/waste", nil),
		storageLoadMeter:  metrics.GetOrRegisterMeter(prefix+"/storage/load", nil),
		storageDupMeter:   metrics.GetOrRegisterMeter(prefix+"/storage/dup", nil),
		storageWasteMeter: metrics.GetOrRegisterMeter(prefix+"/storage/waste", nil),
	}
}

// close iterates over all the subfetchers, waits on any that were left spinning
// and reports the stats to the metrics subsystem.
func (p *triePrefetcher) close() {
	// Short circuit if the fetcher is already closed.
	if p.closed {
		return
	}
	for _, fetcher := range p.fetchers {
		fetcher.close()

		if metrics.Enabled {
			if fetcher.root == p.root {
				p.accountLoadMeter.Mark(int64(len(fetcher.seen)))
				p.accountDupMeter.Mark(int64(fetcher.dups))
				for _, key := range fetcher.used {
					delete(fetcher.seen, string(key))
				}
				p.accountWasteMeter.Mark(int64(len(fetcher.seen)))
			} else {
				p.storageLoadMeter.Mark(int64(len(fetcher.seen)))
				p.storageDupMeter.Mark(int64(fetcher.dups))
				for _, key := range fetcher.used {
					delete(fetcher.seen, string(key))
				}
				p.storageWasteMeter.Mark(int64(len(fetcher.seen)))
			}
		}
	}
	p.closed = true
	p.fetchers = nil
}

// prefetch schedules a batch of trie items to prefetch. After the prefetcher is
// closed, all the following tasks scheduled will not be executed and an error
// will be returned.
//
// prefetch is called from two locations:
//
//  1. Finalize of the state-objects storage roots. This happens at the end
//     of every transaction, meaning that if several transactions touches
//     upon the same contract, the parameters invoking this method may be
//     repeated.
//  2. Finalize of the main account trie. This happens only once per block.
func (p *triePrefetcher) prefetch(owner common.Hash, root common.Hash, addr common.Address, keys [][]byte) error {
	if p.closed {
		return errTerminated
	}
	id := p.trieID(owner, root)
	fetcher := p.fetchers[id]
	if fetcher == nil {
		fetcher = newSubfetcher(p.db, p.root, owner, root, addr)
		p.fetchers[id] = fetcher
	}
	return fetcher.schedule(keys)
}

// trie returns the trie matching the root hash, or nil if either the fetcher
// is terminated or the trie is not available.
func (p *triePrefetcher) trie(owner common.Hash, root common.Hash) Trie {
	if p.closed {
		return nil
	}
	// Bail if no trie was prefetched for this root
	fetcher := p.fetchers[p.trieID(owner, root)]
	if fetcher == nil {
		p.deliveryMissMeter.Mark(1)
		return nil
	}
	return fetcher.peek()
}

// used marks a batch of state items used to allow creating statistics as to
// how useful or wasteful the fetcher is.
func (p *triePrefetcher) used(owner common.Hash, root common.Hash, used [][]byte) {
	if p.closed {
		return
	}
	if fetcher := p.fetchers[p.trieID(owner, root)]; fetcher != nil {
		fetcher.used = used
	}
}

// trieID returns an unique trie identifier consists the trie owner and root hash.
func (p *triePrefetcher) trieID(owner common.Hash, root common.Hash) string {
	trieID := make([]byte, common.HashLength*2)
	copy(trieID, owner.Bytes())
	copy(trieID[common.HashLength:], root.Bytes())
	return string(trieID)
}

// subfetcher is a trie fetcher goroutine responsible for pulling entries for a
// single trie. It is spawned when a new root is encountered and lives until the
// main prefetcher is paused and either all requested items are processed or if
// the trie being worked on is retrieved from the prefetcher.
type subfetcher struct {
	db    Database       // Database to load trie nodes through
	state common.Hash    // Root hash of the state to prefetch
	owner common.Hash    // Owner of the trie, usually account hash
	root  common.Hash    // Root hash of the trie to prefetch
	addr  common.Address // Address of the account that the trie belongs to
	trie  Trie           // Trie being populated with nodes

	tasks [][]byte   // Items queued up for retrieval
	lock  sync.Mutex // Lock protecting the task queue

	wake chan struct{}  // Wake channel if a new task is scheduled
	stop chan struct{}  // Channel to interrupt processing
	term chan struct{}  // Channel to signal interruption
	copy chan chan Trie // channel for retrieving copies of the subfetcher's trie

	seen map[string]struct{} // Tracks the entries already loaded
	dups int                 // Number of duplicate preload tasks
	used [][]byte            // Tracks the entries used in the end
}

// newSubfetcher creates a goroutine to prefetch state items belonging to a
// particular root hash.
func newSubfetcher(db Database, state common.Hash, owner common.Hash, root common.Hash, addr common.Address) *subfetcher {
	sf := &subfetcher{
		db:    db,
		state: state,
		owner: owner,
		root:  root,
		addr:  addr,
		wake:  make(chan struct{}),
		stop:  make(chan struct{}),
		term:  make(chan struct{}),
		copy:  make(chan chan Trie),
		seen:  make(map[string]struct{}),
	}
	go sf.loop()
	return sf
}

// schedule adds a batch of trie keys to the queue to prefetch.
func (sf *subfetcher) schedule(keys [][]byte) error {
	// Append the tasks to the current queue
	sf.lock.Lock()
	sf.tasks = append(sf.tasks, keys...)
	sf.lock.Unlock()

	// Notify the background thread to execute scheduled tasks
	select {
	case sf.wake <- struct{}{}:
		return nil
	case <-sf.term:
		return errTerminated
	}
}

// peek tries to retrieve a deep copy of the fetcher's trie. Nil is returned
// if the fetcher is already terminated, or the associated trie is failing
// for opening.
func (sf *subfetcher) peek() Trie {
	ch := make(chan Trie)
	select {
	case sf.copy <- ch:
		return <-ch
	case <-sf.term:
		return nil
	}
}

// close waits for the subfetcher to finish its tasks. It cannot be called multiple times
func (sf *subfetcher) close() {
	select {
	case <-sf.stop:
	default:
		close(sf.stop)
	}
	<-sf.term
}

// loop loads newly-scheduled trie tasks as they are received and loads them, stopping
// when requested.
func (sf *subfetcher) loop() {
	// No matter how the loop stops, signal anyone waiting that it's terminated
	defer close(sf.term)

	// Start by opening the trie and stop processing if it fails.
	if sf.owner == (common.Hash{}) {
		trie, err := sf.db.OpenTrie(sf.root)
		if err != nil {
			log.Warn("Trie prefetcher failed opening trie", "root", sf.root, "err", err)
			return
		}
		sf.trie = trie
	} else {
		trie, err := sf.db.OpenStorageTrie(sf.state, sf.addr, sf.root, nil)
		if err != nil {
			log.Warn("Trie prefetcher failed opening trie", "root", sf.root, "err", err)
			return
		}
		sf.trie = trie
	}
	// Trie opened successfully, keep prefetching items
	for {
		select {
		case <-sf.wake:
			// Execute all remaining tasks in single run
			sf.lock.Lock()
			tasks := sf.tasks
			sf.tasks = nil
			sf.lock.Unlock()

			for _, task := range tasks {
				if _, ok := sf.seen[string(task)]; ok {
					sf.dups++
					continue
				}
				if len(task) == common.AddressLength {
					sf.trie.GetAccount(common.BytesToAddress(task))
				} else {
					sf.trie.GetStorage(sf.addr, task)
				}
				sf.seen[string(task)] = struct{}{}
			}
		case ch := <-sf.copy:
			// Somebody wants a copy of the current trie, grant them.
			ch <- sf.db.CopyTrie(sf.trie)

		case <-sf.stop:
			// Termination is requested, abort
			return
		}
	}
}
