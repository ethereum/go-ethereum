// Copyright 2026 The go-ethereum Authors
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

// The EIP27928 reader utilizes a hierarchical architecture to optimize state
// access during block execution:
//
// - Base layer: The reader is initialized with the pre-transition state root,
//   providing the access of the state.
//
// - Prefetching Layer: This base reader is wrapped by newPrefetchStateReader.
//   Using an Access List hint, it asynchronously fetches required state data
//   in the background, minimizing I/O blocking during transaction processing.
//
// - Execution Layer: To support parallel transaction execution within the EIP
//   7928 context, readers are wrapped in ReaderWithBlockLevelAccessList.
//   This layer provides a "unified view" by merging the pre-transition state
//   with mutated states from preceding transactions in the block.
//
// - Tracking Layer: Finally, the readerTracker wraps the execution reader to
//   capture all state accesses made during a specific transaction. These individual
//   access are subsequently merged to construct a comprehensive access list
//   for the entire block.
//
// The architecture can be illustrated by the diagram below:

//       [ Block Level Access List ]  <────────────────┐
//                  ▲                                  │ (Merge)
//                  │                                  │
//          ┌───────┴───────┐                  ┌───────┴───────┐
//          │ readerTracker │                  │ readerTracker │  (Access Tracking)
//          └───────┬───────┘                  └───────┬───────┘
//                  │                                  │
//   ┌──────────────┴──────────────┐    ┌──────────────┴──────────────┐
//   │ ReaderWithBlockLevelAL      │    │ ReaderWithBlockLevelAL      │  (Unified View)
//   │ (Pre-state + Mutations)     │    │ (Pre-state + Mutations)     │
//   └──────────────┬──────────────┘    └──────────────┬──────────────┘
//                  │                                  │
//                  └────────────────┬─────────────────┘
//                                   │
//                    ┌──────────────┴──────────────┐
//                    │    newPrefetchStateReader   │ (Async I/O)
//                    │  (Access List Hint driven)  │
//                    └──────────────┬──────────────┘
//                                   │
//                    ┌──────────────┴──────────────┐
//                    │        Base Reader          │ (State Root)
//                    │ (State & Contract Code)     │
//                    └─────────────────────────────┘

import (
	"maps"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
)

type fetchTask struct {
	addr  common.Address
	slots []common.Hash
}

func (t *fetchTask) weight() int { return 1 + len(t.slots) }

type prefetchStateReader struct {
	StateReader
	tasks     []*fetchTask
	nThreads  int
	done      chan struct{}
	term      chan struct{}
	closeOnce sync.Once
}

func newPrefetchStateReader(reader StateReader, accessList map[common.Address][]common.Hash, nThreads int) *prefetchStateReader {
	tasks := make([]*fetchTask, 0, len(accessList))
	for addr, slots := range accessList {
		tasks = append(tasks, &fetchTask{
			addr:  addr,
			slots: slots,
		})
	}
	return newPrefetchStateReaderInternal(reader, tasks, nThreads)
}

func newPrefetchStateReaderInternal(reader StateReader, tasks []*fetchTask, nThreads int) *prefetchStateReader {
	r := &prefetchStateReader{
		StateReader: reader,
		tasks:       tasks,
		nThreads:    nThreads,
		done:        make(chan struct{}),
		term:        make(chan struct{}),
	}
	go r.prefetch()
	return r
}

func (r *prefetchStateReader) Close() {
	r.closeOnce.Do(func() {
		close(r.term)
		<-r.done
	})
}

func (r *prefetchStateReader) Wait() error {
	select {
	case <-r.term:
		return nil
	case <-r.done:
		return nil
	}
}

func (r *prefetchStateReader) prefetch() {
	defer close(r.done)

	if len(r.tasks) == 0 {
		return
	}
	var total int
	for _, t := range r.tasks {
		total += t.weight()
	}
	var (
		wg   sync.WaitGroup
		unit = (total + r.nThreads - 1) / r.nThreads // round-up the per worker unit
	)
	for i := 0; i < r.nThreads; i++ {
		start := i * unit
		if start >= total {
			break
		}
		limit := (i + 1) * unit
		if i == r.nThreads-1 {
			limit = total
		}
		// Schedule the worker for prefetching, the items on the range [start, limit)
		// is exclusively assigned for this worker.
		wg.Add(1)
		go func(workerID, startW, endW int) {
			r.process(startW, endW)
			wg.Done()
		}(i, start, limit)
	}
	wg.Wait()
}

func (r *prefetchStateReader) process(start, limit int) {
	var total = 0
	for _, t := range r.tasks {
		tw := t.weight()
		if total+tw > start {
			s := 0
			if start > total {
				s = start - total
			}
			l := tw
			if limit < total+tw {
				l = limit - total
			}
			for j := s; j < l; j++ {
				select {
				case <-r.term:
					return
				default:
					if j == 0 {
						r.StateReader.Account(t.addr)
					} else {
						r.StateReader.Storage(t.addr, t.slots[j-1])
					}
				}
			}
		}
		total += tw
		if total >= limit {
			return
		}
	}
}

// ReaderWithBlockLevelAccessList provides state access that reflects the
// pre-transition state combined with the mutations made by transactions
// prior to TxIndex.
type ReaderWithBlockLevelAccessList struct {
	Reader
	AccessList *bal.ConstructionBlockAccessList
	TxIndex    int
}

func NewReaderWithBlockLevelAccessList(base Reader, accessList *bal.ConstructionBlockAccessList, txIndex int) *ReaderWithBlockLevelAccessList {
	return &ReaderWithBlockLevelAccessList{
		Reader:     base,
		AccessList: accessList,
		TxIndex:    txIndex,
	}
}

// Account implements Reader, returning the account with the specific address.
func (r *ReaderWithBlockLevelAccessList) Account(addr common.Address) (*types.StateAccount, error) {
	panic("implement me")
}

// Storage implements Reader, returning the storage slot with the specific
// address and slot key.
func (r *ReaderWithBlockLevelAccessList) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	panic("implement me")
}

// Has implements Reader, returning the flag indicating whether the contract
// code with specified address and hash exists or not.
func (r *ReaderWithBlockLevelAccessList) Has(addr common.Address, codeHash common.Hash) bool {
	panic("implement me")
}

// Code implements Reader, returning the contract code with specified address
// and hash.
func (r *ReaderWithBlockLevelAccessList) Code(addr common.Address, codeHash common.Hash) ([]byte, error) {
	panic("implement me")
}

// CodeSize implements Reader, returning the contract code size with specified
// address and hash.
func (r *ReaderWithBlockLevelAccessList) CodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	panic("implement me")
}

// StorageAccessList represents a set of storage slots accessed within an account.
type StorageAccessList map[common.Hash]struct{}

// StateAccessList maps account addresses to their respective accessed storage slots.
type StateAccessList map[common.Address]StorageAccessList

// Merge merges the entries from the other StateAccessList into the receiver.
func (s StateAccessList) Merge(other StateAccessList) {
	for addr, otherSlots := range other {
		slots, exists := s[addr]
		if !exists {
			s[addr] = otherSlots
			continue
		}
		maps.Copy(slots, otherSlots)
	}
}

// StateReaderTracker defines the capability to retrieve the access footprint
// recorded during state reading operations.
type StateReaderTracker interface {
	GetStateAccessList() StateAccessList
}

type readerTracker struct {
	Reader
	access StateAccessList
	lock   sync.RWMutex
}

func newReaderTracker(reader Reader) *readerTracker {
	return &readerTracker{
		Reader: reader,
		access: make(StateAccessList),
	}
}

// Account implements StateReader, tracking the accessed address locally.
func (r *readerTracker) Account(addr common.Address) (*types.StateAccount, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, exists := r.access[addr]
	if !exists {
		r.access[addr] = make(StorageAccessList)
	}
	return r.Reader.Account(addr)
}

// Storage implements StateReader, tracking the accessed slot identifier locally.
func (r *readerTracker) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	list, exists := r.access[addr]
	if !exists {
		list = make(StorageAccessList)
		r.access[addr] = list
	}
	list[slot] = struct{}{}

	return r.Reader.Storage(addr, slot)
}

// GetStateAccessList implements StateReaderTracker, returning the access footprint.
func (r *readerTracker) GetStateAccessList() StateAccessList {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.access
}
