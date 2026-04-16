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
)

var errTerminated = errors.New("fetcher is already terminated")

type slotKey struct {
	addr common.Address
	slot common.Hash
}

type taskKind uint8

const (
	kindAccount taskKind = iota
	kindStorage
)

type prefetchTask struct {
	read bool
	kind taskKind

	accounts []common.Address // kindAccount: addresses to prefetch
	account  common.Address   // kindStorage: owner address
	slots    []common.Hash    // kindStorage: slot keys to prefetch
}

// prefetcher is a background goroutine that preloads trie nodes for a single
// trie. It deduplicates requests and stops when explicitly terminated.
type prefetcher struct {
	prefetchRead bool            // Whether the state read will trigger preloading
	trie         Trie            // Trie being populated with nodes
	tasks        []*prefetchTask // Items queued up for retrieval
	lock         sync.Mutex      // Lock protecting the task queue

	wake chan struct{} // Wake channel if a new task is scheduled
	stop chan struct{} // Channel to interrupt processing
	term chan struct{} // Channel to signal interruption

	seenReadAddr  map[common.Address]struct{} // Dedup: accounts loaded via reads
	seenWriteAddr map[common.Address]struct{} // Dedup: accounts loaded via writes
	seenReadSlot  map[slotKey]struct{}        // Dedup: slots loaded via reads
	seenWriteSlot map[slotKey]struct{}        // Dedup: slots loaded via writes
}

// newPrefetcher creates a background goroutine to prefetch state items from the
// given trie.
func newPrefetcher(tr Trie, prefetchRead bool) *prefetcher {
	p := &prefetcher{
		prefetchRead:  prefetchRead,
		trie:          tr,
		wake:          make(chan struct{}, 1),
		stop:          make(chan struct{}),
		term:          make(chan struct{}),
		seenReadAddr:  make(map[common.Address]struct{}),
		seenWriteAddr: make(map[common.Address]struct{}),
		seenReadSlot:  make(map[slotKey]struct{}),
		seenWriteSlot: make(map[slotKey]struct{}),
	}
	go p.loop()
	return p
}

// scheduleAccounts adds a batch of accounts to the prefetch queue.
func (p *prefetcher) scheduleAccounts(addrs []common.Address, read bool) error {
	select {
	case <-p.term:
		return errTerminated
	default:
	}
	if !p.prefetchRead && read {
		return nil
	}
	p.lock.Lock()
	p.tasks = append(p.tasks, &prefetchTask{
		read:     read,
		kind:     kindAccount,
		accounts: addrs,
	})
	p.lock.Unlock()

	select {
	case p.wake <- struct{}{}:
	default:
	}
	return nil
}

// scheduleSlots adds a batch of storage slots to the prefetch queue.
func (p *prefetcher) scheduleSlots(addr common.Address, slots []common.Hash, read bool) error {
	select {
	case <-p.term:
		return errTerminated
	default:
	}
	if !p.prefetchRead && read {
		return nil
	}
	p.lock.Lock()
	p.tasks = append(p.tasks, &prefetchTask{
		read:    read,
		kind:    kindStorage,
		account: addr,
		slots:   slots,
	})
	p.lock.Unlock()

	select {
	case p.wake <- struct{}{}:
	default:
	}
	return nil
}

// terminate requests the prefetcher to stop and optionally waits for it.
func (p *prefetcher) terminate() {
	select {
	case <-p.stop:
	default:
		close(p.stop)
	}
	<-p.term
}

// loop processes prefetch tasks until terminated.
func (p *prefetcher) loop() {
	defer close(p.term)

	for {
		select {
		case <-p.wake:
			p.lock.Lock()
			tasks := p.tasks
			p.tasks = nil
			p.lock.Unlock()

			var (
				addrs []common.Address
				slots = make(map[common.Address][][]byte)
			)
			for _, task := range tasks {
				if task.kind == kindAccount {
					for _, addr := range task.accounts {
						if p.dedupAddr(addr, task.read) {
							continue
						}
						addrs = append(addrs, addr)
					}
				} else {
					for _, slot := range task.slots {
						if p.dedupSlot(task.account, slot, task.read) {
							continue
						}
						slots[task.account] = append(slots[task.account], slot.Bytes())
					}
				}
			}
			if len(addrs) > 0 {
				if err := p.trie.PrefetchAccount(addrs); err != nil {
					log.Error("Failed to prefetch accounts", "err", err)
				}
			}
			for addr, keys := range slots {
				if err := p.trie.PrefetchStorage(addr, keys); err != nil {
					log.Error("Failed to prefetch storage", "err", err)
				}
			}

		case <-p.stop:
			p.lock.Lock()
			done := p.tasks == nil
			p.lock.Unlock()

			if done {
				return
			}
		}
	}
}

// dedupAddr returns true if addr was already seen for this read/write category.
func (p *prefetcher) dedupAddr(addr common.Address, read bool) bool {
	if read {
		if _, ok := p.seenReadAddr[addr]; ok {
			return true
		}
		if _, ok := p.seenWriteAddr[addr]; ok {
			return true
		}
		p.seenReadAddr[addr] = struct{}{}
	} else {
		if _, ok := p.seenReadAddr[addr]; ok {
			return true
		}
		if _, ok := p.seenWriteAddr[addr]; ok {
			return true
		}
		p.seenWriteAddr[addr] = struct{}{}
	}
	return false
}

// dedupSlot returns true if slot was already seen for this read/write category.
func (p *prefetcher) dedupSlot(addr common.Address, slot common.Hash, read bool) bool {
	key := slotKey{addr: addr, slot: slot}
	if read {
		if _, ok := p.seenReadSlot[key]; ok {
			return true
		}
		if _, ok := p.seenWriteSlot[key]; ok {
			return true
		}
		p.seenReadSlot[key] = struct{}{}
	} else {
		if _, ok := p.seenReadSlot[key]; ok {
			return true
		}
		if _, ok := p.seenWriteSlot[key]; ok {
			return true
		}
		p.seenWriteSlot[key] = struct{}{}
	}
	return false
}
