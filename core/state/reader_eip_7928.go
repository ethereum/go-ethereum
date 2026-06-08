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
// The architecture can be illustrated by the diagram below:

//       [ Block Level Access List ]  <────────────────┐
//                  ▲                                  │ (Merge)
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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

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
	start     time.Time
	metrics   PrefetchMetrics
}

type PrefetchMetrics struct {
	// the total amount of time it took to complete the scheduled workload
	Elapsed time.Duration
}

// PrefetcherMetricer is an object that can expose metrics related to the state
// prefetching.
type PrefetcherMetricer interface {
	Metrics() PrefetchMetrics
}

func newPrefetchStateReader(reader StateReader, accessList bal.StorageKeys, nThreads int) *prefetchStateReader {
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
		start:       time.Now(),
	}
	go r.prefetch()
	return r
}

func (r *prefetchStateReader) Metrics() PrefetchMetrics {
	return r.metrics
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
	defer func() {
		r.metrics = PrefetchMetrics{time.Since(r.start)}
		close(r.done)
	}()

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
//
// It is a cheap, per-transaction view over a shared, read-only
// bal.AccessListReader: constructing one is O(1) and every lookup is an
// allocation-free binary search.
type ReaderWithBlockLevelAccessList struct {
	Reader
	prepared *bal.AccessListReader
	TxIndex  int
}

// NewReaderWithPreparedAccessList wraps a base reader with a shared, already
// preprocessed access list. This is the cheap constructor used on the hot path:
// the prepared list is built once per block and borrowed by every per-tx reader.
func NewReaderWithPreparedAccessList(base Reader, prepared *bal.AccessListReader, txIndex int) *ReaderWithBlockLevelAccessList {
	return &ReaderWithBlockLevelAccessList{
		Reader:   base,
		prepared: prepared,
		TxIndex:  txIndex,
	}
}

// NewReaderWithBlockLevelAccessList wraps a base reader with a raw access list,
// preprocessing it on the spot. Prefer NewReaderWithPreparedAccessList when the
// prepared list can be built once and shared across multiple readers.
func NewReaderWithBlockLevelAccessList(base Reader, accessList bal.BlockAccessList, txIndex int) *ReaderWithBlockLevelAccessList {
	return NewReaderWithPreparedAccessList(base, bal.NewAccessListReader(accessList), txIndex)
}

// Account implements Reader, returning the account with the specific address.
func (r *ReaderWithBlockLevelAccessList) Account(addr common.Address) (acct *types.StateAccount, err error) {
	acct, err = r.Reader.Account(addr)
	if err != nil {
		return nil, err
	}

	balance := r.prepared.Balance(addr, r.TxIndex)
	code := r.prepared.Code(addr, r.TxIndex)
	nonce, hasNonce := r.prepared.Nonce(addr, r.TxIndex)
	if balance == nil && code == nil && !hasNonce {
		return acct, nil
	}

	if acct == nil {
		acct = types.NewEmptyStateAccount()
	} else {
		// the account returned by the underlying reader is a reference
		// copy it to avoid mutating the reader's instance
		acct = acct.Copy()
	}

	// balance and code alias the shared access list; this is safe because the
	// EVM never mutates them in place (it replaces the pointer/slice wholesale,
	// and the journal clones before stashing).
	if balance != nil {
		acct.Balance = balance
	}
	if code != nil {
		codeHash := crypto.Keccak256Hash(code)
		acct.CodeHash = codeHash[:]
	}
	if hasNonce {
		acct.Nonce = nonce
	}
	return
}

// Storage implements Reader, returning the storage slot with the specific
// address and slot key.
func (r *ReaderWithBlockLevelAccessList) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	if val, ok := r.prepared.StorageAt(addr, slot, r.TxIndex); ok {
		return val, nil
	}
	return r.Reader.Storage(addr, slot)
}

// Has implements Reader, returning the flag indicating whether the contract
// code with specified address and hash exists or not.
func (r *ReaderWithBlockLevelAccessList) Has(addr common.Address, codeHash common.Hash) bool {
	if code := r.prepared.Code(addr, r.TxIndex); code != nil {
		return crypto.Keccak256Hash(code) == codeHash
	}
	return r.Reader.Has(addr, codeHash)
}

// Code implements Reader, returning the contract code with specified address
// and hash.
func (r *ReaderWithBlockLevelAccessList) Code(addr common.Address, codeHash common.Hash) []byte {
	if code := r.prepared.Code(addr, r.TxIndex); code != nil && crypto.Keccak256Hash(code) == codeHash {
		return code
	}
	return r.Reader.Code(addr, codeHash)
}

// CodeSize implements Reader, returning the contract code size with specified
// address and hash.
func (r *ReaderWithBlockLevelAccessList) CodeSize(addr common.Address, codeHash common.Hash) int {
	if code := r.prepared.Code(addr, r.TxIndex); code != nil && crypto.Keccak256Hash(code) == codeHash {
		return len(code)
	}
	return r.Reader.CodeSize(addr, codeHash)
}
