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
	"github.com/ethereum/go-ethereum/trie"
)

//lint:file-ignore U1000 this file intentionally keeps unused helpers for future use

var (
	// triePrefetchMetricsPrefix is the prefix under which to publish the metrics.
	triePrefetchMetricsPrefix = "trie/prefetch/"

	// errTerminated is returned if a fetcher is attempted to be operated after it
	// has already terminated.
	errTerminated = errors.New("fetcher is already terminated")
)

type trieOpener func(id trie.ID, addr common.Address) (Trie, error) // Define the handler to open the trie for pulling

// triePrefetcher is an active prefetcher, which receives accounts or storage
// items and does trie-loading of them. The goal is to get as much useful content
// into the caches as possible.
//
// Note, the prefetcher's API is not thread safe.
type triePrefetcher struct {
	root     common.Hash             // Root hash of the account trie for metrics
	fetchers map[trie.ID]*subfetcher // Subfetchers for each trie
	term     chan struct{}           // Channel to signal interruption
	noreads  bool                    // Whether to ignore state-read-only prefetch requests
	opener   trieOpener              // Handler to open the trie for pulling

	deliveryMissMeter *metrics.Meter

	accountLoadReadMeter  *metrics.Meter
	accountLoadWriteMeter *metrics.Meter
	accountDupReadMeter   *metrics.Meter
	accountDupWriteMeter  *metrics.Meter
	accountDupCrossMeter  *metrics.Meter
	accountWasteMeter     *metrics.Meter

	storageLoadReadMeter  *metrics.Meter
	storageLoadWriteMeter *metrics.Meter
	storageDupReadMeter   *metrics.Meter
	storageDupWriteMeter  *metrics.Meter
	storageDupCrossMeter  *metrics.Meter
	storageWasteMeter     *metrics.Meter
}

func newTriePrefetcher(opener trieOpener, root common.Hash, namespace string, noreads bool) *triePrefetcher {
	prefix := triePrefetchMetricsPrefix + namespace
	return &triePrefetcher{
		root:     root,
		fetchers: make(map[trie.ID]*subfetcher), // Active prefetchers use the fetchers map
		term:     make(chan struct{}),
		noreads:  noreads,
		opener:   opener,

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
// nolint:unused
func (p *triePrefetcher) report() {
	if !metrics.Enabled() {
		return
	}
	for _, fetcher := range p.fetchers {
		fetcher.wait() // ensure the fetcher's idle before poking in its internals

		if fetcher.id.Owner == (common.Hash{}) {
			p.accountLoadReadMeter.Mark(int64(len(fetcher.seenReadAddr)))
			p.accountLoadWriteMeter.Mark(int64(len(fetcher.seenWriteAddr)))

			p.accountDupReadMeter.Mark(int64(fetcher.dupsRead))
			p.accountDupWriteMeter.Mark(int64(fetcher.dupsWrite))
			p.accountDupCrossMeter.Mark(int64(fetcher.dupsCross))

			for _, key := range fetcher.usedAddr {
				delete(fetcher.seenReadAddr, key)
				delete(fetcher.seenWriteAddr, key)
			}
			p.accountWasteMeter.Mark(int64(len(fetcher.seenReadAddr) + len(fetcher.seenWriteAddr)))
		} else {
			p.storageLoadReadMeter.Mark(int64(len(fetcher.seenReadSlot)))
			p.storageLoadWriteMeter.Mark(int64(len(fetcher.seenWriteSlot)))

			p.storageDupReadMeter.Mark(int64(fetcher.dupsRead))
			p.storageDupWriteMeter.Mark(int64(fetcher.dupsWrite))
			p.storageDupCrossMeter.Mark(int64(fetcher.dupsCross))

			for _, key := range fetcher.usedSlot {
				delete(fetcher.seenReadSlot, key)
				delete(fetcher.seenWriteSlot, key)
			}
			p.storageWasteMeter.Mark(int64(len(fetcher.seenReadSlot) + len(fetcher.seenWriteSlot)))
		}
	}
}

// prefetchAccounts schedules a batch of accounts to prefetch.
func (p *triePrefetcher) prefetchAccounts(id trie.ID, addrs []common.Address, read bool) error {
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
	fetcher := p.fetchers[id]
	if fetcher == nil {
		fetcher = newSubfetcher(p.opener, id, common.Address{})
		p.fetchers[id] = fetcher
	}
	return fetcher.scheduleAccounts(addrs, read)
}

// prefetchStorage schedules a batch of storage slots to prefetch.
func (p *triePrefetcher) prefetchStorage(id trie.ID, addr common.Address, slots []common.Hash, read bool) error {
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
	fetcher := p.fetchers[id]
	if fetcher == nil {
		fetcher = newSubfetcher(p.opener, id, addr)
		p.fetchers[id] = fetcher
	}
	return fetcher.scheduleSlots(addr, slots, read)
}

// trie returns the trie matching the root hash, blocking until the fetcher of
// the given trie terminates. If no fetcher exists for the request, nil will be
// returned.
func (p *triePrefetcher) trie(id trie.ID) Trie {
	// Bail if no trie was prefetched for this root
	fetcher := p.fetchers[id]
	if fetcher == nil {
		log.Error("Prefetcher missed to load trie", "owner", id.Owner, "root", id.Root)
		p.deliveryMissMeter.Mark(1)
		return nil
	}
	// Subfetcher exists, retrieve its trie
	return fetcher.peek()
}

// used marks a batch of state items used to allow creating statistics as to
// how useful or wasteful the fetcher is.
// nolint:unused
func (p *triePrefetcher) used(id trie.ID, usedAddr []common.Address, usedSlot []common.Hash) {
	if fetcher := p.fetchers[id]; fetcher != nil {
		fetcher.wait() // ensure the fetcher's idle before poking in its internals

		fetcher.usedAddr = append(fetcher.usedAddr, usedAddr...)
		fetcher.usedSlot = append(fetcher.usedSlot, usedSlot...)
	}
}

// subfetcher is a trie fetcher goroutine responsible for pulling entries for a
// single trie. It is spawned when a new root is encountered and lives until the
// main prefetcher is paused and either all requested items are processed or if
// the trie being worked on is retrieved from the prefetcher.
type subfetcher struct {
	id     trie.ID        // The identifier of the trie being populated
	addr   common.Address // Address of the account that the trie belongs to
	trie   Trie           // Trie being populated with nodes
	opener trieOpener     // Handler to open the trie for pulling

	tasks []*subfetcherTask // Items queued up for retrieval
	lock  sync.Mutex        // Lock protecting the task queue

	wake chan struct{} // Wake channel if a new task is scheduled
	stop chan struct{} // Channel to interrupt processing
	term chan struct{} // Channel to signal interruption

	seenReadAddr  map[common.Address]struct{} // Tracks the accounts already loaded via read operations
	seenWriteAddr map[common.Address]struct{} // Tracks the accounts already loaded via write operations
	seenReadSlot  map[common.Hash]struct{}    // Tracks the storage already loaded via read operations
	seenWriteSlot map[common.Hash]struct{}    // Tracks the storage already loaded via write operations

	dupsRead  int // Number of duplicate preload tasks via reads only
	dupsWrite int // Number of duplicate preload tasks via writes only
	dupsCross int // Number of duplicate preload tasks via read-write-crosses

	usedAddr []common.Address // Tracks the accounts used in the end
	usedSlot []common.Hash    // Tracks the storage used in the end
}

type subfetcherTaskKind uint8

const (
	kindAccount subfetcherTaskKind = iota
	kindStorage
)

type subfetcherTask struct {
	read bool
	kind subfetcherTaskKind

	// The list of accounts being pulling in kindAccount type
	accounts []common.Address

	// The list of storage keys being pulling in kindStorage type
	account common.Address
	slots   []common.Hash
}

// newSubfetcher creates a goroutine to prefetch state items belonging to a
// particular root hash.
func newSubfetcher(opener trieOpener, id trie.ID, addr common.Address) *subfetcher {
	sf := &subfetcher{
		id:            id,
		addr:          addr,
		opener:        opener,
		wake:          make(chan struct{}, 1),
		stop:          make(chan struct{}),
		term:          make(chan struct{}),
		seenReadAddr:  make(map[common.Address]struct{}),
		seenWriteAddr: make(map[common.Address]struct{}),
		seenReadSlot:  make(map[common.Hash]struct{}),
		seenWriteSlot: make(map[common.Hash]struct{}),
	}
	go sf.loop()
	return sf
}

// scheduleAccounts adds a batch of accounts to the queue to prefetch.
func (sf *subfetcher) scheduleAccounts(addrs []common.Address, read bool) error {
	// Ensure the subfetcher is still alive
	select {
	case <-sf.term:
		return errTerminated
	default:
	}
	// Append the tasks to the current queue
	sf.lock.Lock()
	sf.tasks = append(sf.tasks, &subfetcherTask{
		read:     read,
		kind:     kindAccount,
		accounts: addrs,
	})
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

// scheduleSlots adds a batch of storage slots to the queue to prefetch.
func (sf *subfetcher) scheduleSlots(addr common.Address, slots []common.Hash, read bool) error {
	// Ensure the subfetcher is still alive
	select {
	case <-sf.term:
		return errTerminated
	default:
	}
	// Append the tasks to the current queue
	sf.lock.Lock()
	sf.tasks = append(sf.tasks, &subfetcherTask{
		read:    read,
		kind:    kindStorage,
		account: addr,
		slots:   slots,
	})
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
	tr, err := sf.opener(sf.id, sf.addr)
	if err != nil {
		log.Warn("Trie prefetcher failed opening trie", "id", sf.id, "err", err)
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

			var (
				// Account tasks
				addresses []common.Address

				// Slot tasks
				slots = make(map[common.Address][][]byte)
			)
			for _, task := range tasks {
				if task.kind == kindAccount {
					for _, addr := range task.accounts {
						if task.read {
							if _, ok := sf.seenReadAddr[addr]; ok {
								sf.dupsRead++
								continue
							}
							if _, ok := sf.seenWriteAddr[addr]; ok {
								sf.dupsCross++
								continue
							}
							sf.seenReadAddr[addr] = struct{}{}
						} else {
							if _, ok := sf.seenReadAddr[addr]; ok {
								sf.dupsCross++
								continue
							}
							if _, ok := sf.seenWriteAddr[addr]; ok {
								sf.dupsWrite++
								continue
							}
							sf.seenWriteAddr[addr] = struct{}{}
						}
						addresses = append(addresses, addr)
					}
				} else {
					var keys [][]byte
					for _, slot := range task.slots {
						if task.read {
							if _, ok := sf.seenReadSlot[slot]; ok {
								sf.dupsRead++
								continue
							}
							if _, ok := sf.seenWriteSlot[slot]; ok {
								sf.dupsCross++
								continue
							}
							sf.seenReadSlot[slot] = struct{}{}
						} else {
							if _, ok := sf.seenReadSlot[slot]; ok {
								sf.dupsCross++
								continue
							}
							if _, ok := sf.seenWriteSlot[slot]; ok {
								sf.dupsWrite++
								continue
							}
							sf.seenWriteSlot[slot] = struct{}{}
						}
						keys = append(keys, slot.Bytes())
					}
					slots[task.account] = append(slots[task.account], keys...)
				}
			}
			if len(addresses) != 0 {
				if err := sf.trie.PrefetchAccount(addresses); err != nil {
					log.Error("Failed to prefetch accounts", "err", err)
				}
			}
			for addr, keys := range slots {
				if err := sf.trie.PrefetchStorage(addr, keys); err != nil {
					log.Error("Failed to prefetch storage", "err", err)
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
