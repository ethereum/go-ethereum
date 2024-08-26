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

	// errTerminated is returned if a fetcher is attempted to be operated after it
	// has already terminated.
	errTerminated = errors.New("fetcher is already terminated")
)

// triePrefetcher is an active prefetcher, which receives accounts or storage
// items and does trie-loading of them. The goal is to get as much useful content
// into the caches as possible.
//
// Note, the prefetcher's API is not thread safe.
type triePrefetcher struct {
	verkle   bool                   // Flag whether the prefetcher is in verkle mode
	db       Database               // Database to fetch trie nodes through
	root     common.Hash            // Root hash of the account trie for metrics
	fetchers map[string]*subfetcher // Subfetchers for each trie
	term     chan struct{}          // Channel to signal interruption
	noreads  bool                   // Whether to ignore state-read-only prefetch requests

	deliveryMissMeter metrics.Meter

	accountLoadReadMeter  metrics.Meter
	accountLoadWriteMeter metrics.Meter
	accountDupReadMeter   metrics.Meter
	accountDupWriteMeter  metrics.Meter
	accountDupCrossMeter  metrics.Meter
	accountWasteMeter     metrics.Meter

	storageLoadReadMeter  metrics.Meter
	storageLoadWriteMeter metrics.Meter
	storageDupReadMeter   metrics.Meter
	storageDupWriteMeter  metrics.Meter
	storageDupCrossMeter  metrics.Meter
	storageWasteMeter     metrics.Meter
}

func newTriePrefetcher(db Database, root common.Hash, namespace string, noreads bool) *triePrefetcher {
	prefix := triePrefetchMetricsPrefix + namespace
	return &triePrefetcher{
		verkle:   db.TrieDB().IsVerkle(),
		db:       db,
		root:     root,
		fetchers: make(map[string]*subfetcher), // Active prefetchers use the fetchers map
		term:     make(chan struct{}),
		noreads:  noreads,

		deliveryMissMeter: metrics.GetOrRegisterMeter(prefix+"/deliverymiss", nil),

		accountLoadReadMeter:  metrics.GetOrRegisterMeter(prefix+"/account/load/read", nil),
		accountLoadWriteMeter: metrics.GetOrRegisterMeter(prefix+"/account/load/write", nil),
		accountDupReadMeter:   metrics.GetOrRegisterMeter(prefix+"/account/dup/read", nil),
		accountDupWriteMeter:  metrics.GetOrRegisterMeter(prefix+"/account/dup/write", nil),
		accountDupCrossMeter:  metrics.GetOrRegisterMeter(prefix+"/account/dup/cross", nil),
		accountWasteMeter:     metrics.GetOrRegisterMeter(prefix+"/account/waste", nil),

		storageLoadReadMeter:  metrics.GetOrRegisterMeter(prefix+"/storage/load/read", nil),
		storageLoadWriteMeter: metrics.GetOrRegisterMeter(prefix+"/storage/load/write", nil),
		storageDupReadMeter:   metrics.GetOrRegisterMeter(prefix+"/storage/dup/read", nil),
		storageDupWriteMeter:  metrics.GetOrRegisterMeter(prefix+"/storage/dup/write", nil),
		storageDupCrossMeter:  metrics.GetOrRegisterMeter(prefix+"/storage/dup/cross", nil),
		storageWasteMeter:     metrics.GetOrRegisterMeter(prefix+"/storage/waste", nil),
	}
}

// terminate iterates over all the subfetchers and issues a termination request
// to all of them. Depending on the async parameter, the method will either block
// until all subfetchers spin down, or return immediately.
func (p *triePrefetcher) terminate(async bool) {
	// Short circuit if the fetcher is already closed
	select {
	case <-p.term:
		return
	default:
	}
	// Terminate all sub-fetchers, sync or async, depending on the request
	for _, fetcher := range p.fetchers {
		fetcher.terminate(async)
	}
	close(p.term)
}

// report aggregates the pre-fetching and usage metrics and reports them.
func (p *triePrefetcher) report() {
	if !metrics.Enabled {
		return
	}
	for _, fetcher := range p.fetchers {
		fetcher.wait() // ensure the fetcher's idle before poking in its internals

		if fetcher.root == p.root {
			p.accountLoadReadMeter.Mark(int64(len(fetcher.seenRead)))
			p.accountLoadWriteMeter.Mark(int64(len(fetcher.seenWrite)))

			p.accountDupReadMeter.Mark(int64(fetcher.dupsRead))
			p.accountDupWriteMeter.Mark(int64(fetcher.dupsWrite))
			p.accountDupCrossMeter.Mark(int64(fetcher.dupsCross))

			for _, key := range fetcher.used {
				delete(fetcher.seenRead, string(key))
				delete(fetcher.seenWrite, string(key))
			}
			p.accountWasteMeter.Mark(int64(len(fetcher.seenRead) + len(fetcher.seenWrite)))
		} else {
			p.storageLoadReadMeter.Mark(int64(len(fetcher.seenRead)))
			p.storageLoadWriteMeter.Mark(int64(len(fetcher.seenWrite)))

			p.storageDupReadMeter.Mark(int64(fetcher.dupsRead))
			p.storageDupWriteMeter.Mark(int64(fetcher.dupsWrite))
			p.storageDupCrossMeter.Mark(int64(fetcher.dupsCross))

			for _, key := range fetcher.used {
				delete(fetcher.seenRead, string(key))
				delete(fetcher.seenWrite, string(key))
			}
			p.storageWasteMeter.Mark(int64(len(fetcher.seenRead) + len(fetcher.seenWrite)))
		}
	}
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
func (p *triePrefetcher) prefetch(owner common.Hash, root common.Hash, addr common.Address, keys [][]byte, read bool) error {
	// If the state item is only being read, but reads are disabled, return
	if read && p.noreads {
		return nil
	}
	// Ensure the subfetcher is still alive
	select {
	case <-p.term:
		return errTerminated
	default:
	}
	id := p.trieID(owner, root)
	fetcher := p.fetchers[id]
	if fetcher == nil {
		fetcher = newSubfetcher(p.db, p.root, owner, root, addr)
		p.fetchers[id] = fetcher
	}
	return fetcher.schedule(keys, read)
}

// trie returns the trie matching the root hash, blocking until the fetcher of
// the given trie terminates. If no fetcher exists for the request, nil will be
// returned.
func (p *triePrefetcher) trie(owner common.Hash, root common.Hash) Trie {
	// Bail if no trie was prefetched for this root
	fetcher := p.fetchers[p.trieID(owner, root)]
	if fetcher == nil {
		log.Error("Prefetcher missed to load trie", "owner", owner, "root", root)
		p.deliveryMissMeter.Mark(1)
		return nil
	}
	// Subfetcher exists, retrieve its trie
	return fetcher.peek()
}

// used marks a batch of state items used to allow creating statistics as to
// how useful or wasteful the fetcher is.
func (p *triePrefetcher) used(owner common.Hash, root common.Hash, used [][]byte) {
	if fetcher := p.fetchers[p.trieID(owner, root)]; fetcher != nil {
		fetcher.wait() // ensure the fetcher's idle before poking in its internals
		fetcher.used = append(fetcher.used, used...)
	}
}

// trieID returns an unique trie identifier consists the trie owner and root hash.
func (p *triePrefetcher) trieID(owner common.Hash, root common.Hash) string {
	// The trie in verkle is only identified by state root
	if p.verkle {
		return p.root.Hex()
	}
	// The trie in merkle is either identified by state root (account trie),
	// or identified by the owner and trie root (storage trie)
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

	tasks []*subfetcherTask // Items queued up for retrieval
	lock  sync.Mutex        // Lock protecting the task queue

	wake chan struct{} // Wake channel if a new task is scheduled
	stop chan struct{} // Channel to interrupt processing
	term chan struct{} // Channel to signal interruption

	seenRead  map[string]struct{} // Tracks the entries already loaded via read operations
	seenWrite map[string]struct{} // Tracks the entries already loaded via write operations

	dupsRead  int // Number of duplicate preload tasks via reads only
	dupsWrite int // Number of duplicate preload tasks via writes only
	dupsCross int // Number of duplicate preload tasks via read-write-crosses

	used [][]byte // Tracks the entries used in the end
}

// subfetcherTask is a trie path to prefetch, tagged with whether it originates
// from a read or a write request.
type subfetcherTask struct {
	read bool
	key  []byte
}

// newSubfetcher creates a goroutine to prefetch state items belonging to a
// particular root hash.
func newSubfetcher(db Database, state common.Hash, owner common.Hash, root common.Hash, addr common.Address) *subfetcher {
	sf := &subfetcher{
		db:        db,
		state:     state,
		owner:     owner,
		root:      root,
		addr:      addr,
		wake:      make(chan struct{}, 1),
		stop:      make(chan struct{}),
		term:      make(chan struct{}),
		seenRead:  make(map[string]struct{}),
		seenWrite: make(map[string]struct{}),
	}
	go sf.loop()
	return sf
}

// schedule adds a batch of trie keys to the queue to prefetch.
func (sf *subfetcher) schedule(keys [][]byte, read bool) error {
	// Ensure the subfetcher is still alive
	select {
	case <-sf.term:
		return errTerminated
	default:
	}
	// Append the tasks to the current queue
	sf.lock.Lock()
	for _, key := range keys {
		key := key // closure for the append below
		sf.tasks = append(sf.tasks, &subfetcherTask{read: read, key: key})
	}
	sf.lock.Unlock()

	// Notify the background thread to execute scheduled tasks
	select {
	case sf.wake <- struct{}{}:
		// Wake signal sent
	default:
		// Wake signal not sent as a previous one is already queued
	}
	return nil
}

// wait blocks until the subfetcher terminates. This method is used to block on
// an async termination before accessing internal fields from the fetcher.
func (sf *subfetcher) wait() {
	<-sf.term
}

// peek retrieves the fetcher's trie, populated with any pre-fetched data. The
// returned trie will be a shallow copy, so modifying it will break subsequent
// peeks for the original data. The method will block until all the scheduled
// data has been loaded and the fethcer terminated.
func (sf *subfetcher) peek() Trie {
	// Block until the fetcher terminates, then retrieve the trie
	sf.wait()
	return sf.trie
}

// terminate requests the subfetcher to stop accepting new tasks and spin down
// as soon as everything is loaded. Depending on the async parameter, the method
// will either block until all disk loads finish or return immediately.
func (sf *subfetcher) terminate(async bool) {
	select {
	case <-sf.stop:
	default:
		close(sf.stop)
	}
	if async {
		return
	}
	<-sf.term
}

// openTrie resolves the target trie from database for prefetching.
func (sf *subfetcher) openTrie() error {
	// Open the verkle tree if the sub-fetcher is in verkle mode. Note, there is
	// only a single fetcher for verkle.
	if sf.db.TrieDB().IsVerkle() {
		tr, err := sf.db.OpenTrie(sf.state)
		if err != nil {
			log.Warn("Trie prefetcher failed opening verkle trie", "root", sf.root, "err", err)
			return err
		}
		sf.trie = tr
		return nil
	}
	// Open the merkle tree if the sub-fetcher is in merkle mode
	if sf.owner == (common.Hash{}) {
		tr, err := sf.db.OpenTrie(sf.state)
		if err != nil {
			log.Warn("Trie prefetcher failed opening account trie", "root", sf.root, "err", err)
			return err
		}
		sf.trie = tr
		return nil
	}
	tr, err := sf.db.OpenStorageTrie(sf.state, sf.addr, sf.root, nil)
	if err != nil {
		log.Warn("Trie prefetcher failed opening storage trie", "root", sf.root, "err", err)
		return err
	}
	sf.trie = tr
	return nil
}

// loop loads newly-scheduled trie tasks as they are received and loads them, stopping
// when requested.
func (sf *subfetcher) loop() {
	// No matter how the loop stops, signal anyone waiting that it's terminated
	defer close(sf.term)

	if err := sf.openTrie(); err != nil {
		return
	}
	for {
		select {
		case <-sf.wake:
			// Execute all remaining tasks in a single run
			sf.lock.Lock()
			tasks := sf.tasks
			sf.tasks = nil
			sf.lock.Unlock()

			for _, task := range tasks {
				key := string(task.key)
				if task.read {
					if _, ok := sf.seenRead[key]; ok {
						sf.dupsRead++
						continue
					}
					if _, ok := sf.seenWrite[key]; ok {
						sf.dupsCross++
						continue
					}
				} else {
					if _, ok := sf.seenRead[key]; ok {
						sf.dupsCross++
						continue
					}
					if _, ok := sf.seenWrite[key]; ok {
						sf.dupsWrite++
						continue
					}
				}
				if len(task.key) == common.AddressLength {
					sf.trie.GetAccount(common.BytesToAddress(task.key))
				} else {
					sf.trie.GetStorage(sf.addr, task.key)
				}
				if task.read {
					sf.seenRead[key] = struct{}{}
				} else {
					sf.seenWrite[key] = struct{}{}
				}
			}

		case <-sf.stop:
			// Termination is requested, abort if no more tasks are pending. If
			// there are some, exhaust them first.
			sf.lock.Lock()
			done := sf.tasks == nil
			sf.lock.Unlock()

			if done {
				return
			}
			// Some tasks are pending, loop and pick them up (that wake branch
			// will be selected eventually, whilst stop remains closed to this
			// branch will also run afterwards).
		}
	}
}
