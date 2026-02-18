// Copyright 2025-2026 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

// Package parallel provides functionality for precompiled contracts with
// lifespans of an entire block.
package parallel

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/params"
)

// A handler is the non-generic equivalent of a [Handler], exposed by [wrapper].
type handler interface {
	beforeBlock(libevm.StateReader, *types.Block)
	shouldProcess(IndexedTx) (do bool, gas uint64)
	beforeWork(jobs int)
	prefetch(libevm.StateReader, *prefetch)
	nullResult(*job)
	process(libevm.StateReader, *process)
	postProcess()
	finishBlock(vm.StateDB, *types.Block, types.Receipts)
}

// A Processor orchestrates dispatch and collection of results from one or more
// [Handler] instances.
type Processor struct {
	handlers []handler

	workers    sync.WaitGroup
	stateShare stateDBSharer
	prefetch   chan *prefetch
	process    chan *process

	txGas map[common.Hash]uint64
}

type (
	// job is an alias to allow it to be used as an "underlying type" for
	// generic type parameters, while prefetch and process are explicitly *not*
	// aliases, to guarantee that they aren't considered equivalent.
	job = struct {
		handler handler
		tx      IndexedTx
	}
	prefetch job
	process  job
)

type result[T any] struct {
	tx  IndexedTx
	val *T
}

// New constructs a new [Processor] with the specified number of concurrent
// prefetching and processing workers. As prefetching is typically IO-bound, it
// is reasonable to have more prefetchers than processors. The number of
// processors SHOULD be determined from GOMAXPROCS. Pipelining in such a fashion
// stops prefetching for later transactions being blocked by earlier,
// long-running processing; see the respective methods on [Handler] for more
// context.
//
// [Processor.Close] MUST be called after the final call to
// [Processor.FinishBlock] to avoid leaking goroutines.
func New(prefetchers, processors int) *Processor {
	prefetchers = max(prefetchers, 1)
	processors = max(processors, 1)
	workers := prefetchers + processors

	p := &Processor{
		stateShare: stateDBSharer{
			workers:   workers,
			available: make(chan struct{}),
			sdb:       make(chan *state.StateDB, 1),
		},
		prefetch: make(chan *prefetch),
		process:  make(chan *process),
		txGas:    make(map[common.Hash]uint64),
	}

	p.workers.Add(workers)       // for shutdown via [Processor.Close]
	p.stateShare.wg.Add(workers) // for readiness of [Processor.worker] loops
	for range prefetchers {
		go worker(p, p.prefetch, func(sdb libevm.StateReader, job *prefetch) {
			job.handler.prefetch(sdb, job)
		})
	}
	for range processors {
		go worker(p, p.process, func(sdb libevm.StateReader, job *process) {
			job.handler.process(sdb, job)
		})
	}
	p.stateShare.wg.Wait()

	return p
}

// A stateDBSharer allows concurrent workers to make copies of a primary
// database. When the `available` channel is closed, all workers call
// [state.StateDB.Copy] then signal completion on the [sync.WaitGroup]. The
// channel is replaced for each round of distribution.
type stateDBSharer struct {
	available chan struct{}
	sdb       chan *state.StateDB
	workers   int
	wg        sync.WaitGroup
}

func (s *stateDBSharer) distribute(sdb *state.StateDB) {
	ch := s.available                 // already copied by [Processor.worker], which is waiting for it to close
	s.available = make(chan struct{}) // will be copied, ready for the next distribution

	s.sdb <- sdb
	s.wg.Add(s.workers)
	close(ch) // Take a moment to enjoy the symmetry :)
	s.wg.Wait()
	<-s.sdb
}

func worker[J ~job](p *Processor, work <-chan *J, do func(libevm.StateReader, *J)) {
	defer p.workers.Done()

	var sdb *state.StateDB
	share := &p.stateShare
	stateAvailable := share.available
	// Without this signal of readiness, a premature call to
	// [Processor.StartBlock] could replace `share.nextAvailable` before we've
	// copied it.
	share.wg.Done()

	for {
		select {
		case <-stateAvailable: // guaranteed at the beginning of each block
			// [state.StateDB.Copy] is a complex method that isn't explicitly
			// documented as being threadsafe.
			sdb = (<-share.sdb).Copy()
			share.sdb <- sdb // no need to return the original as each worker copies

			stateAvailable = share.available
			share.wg.Done()

		case w, ok := <-work:
			if !ok {
				return
			}
			do(sdb, w)
		}
	}
}

// Close shuts down the [Processor], after which it can no longer be used.
func (p *Processor) Close() {
	close(p.prefetch)
	close(p.process)
	p.workers.Wait()
}

// StartBlock dispatches transactions to every [Handler] but returns immediately
// after performing preliminary setup. It MUST be paired with a call to
// [Processor.FinishBlock], without overlap of blocks.
func (p *Processor) StartBlock(sdb *state.StateDB, rules params.Rules, b *types.Block) error {
	// The distribution mechanism copies the StateDB so we don't need to do it
	// here, but [wrapper.beforeBlock] doesn't make its own copy. Note that even
	// reading from a [state.StateDB] is not threadsafe.
	p.stateShare.distribute(sdb)
	for _, h := range p.handlers {
		h.beforeBlock(sdb.Copy(), b)
	}

	txs := b.Transactions()
	jobs := make([]*job, 0, len(p.handlers)*len(txs))
	workloads := make([]int, len(p.handlers))

	for txIdx, rawTx := range txs {
		tx := IndexedTx{
			Index:       txIdx,
			Transaction: rawTx,
		}

		do, err := p.shouldProcess(tx, rules) // MUST NOT be concurrent within a Handler
		if err != nil {
			return err
		}
		for i, h := range p.handlers {
			j := &job{
				tx:      tx,
				handler: h,
			}
			if !do[i] {
				h.nullResult(j)
				continue
			}
			workloads[i]++
			jobs = append(jobs, j)
		}
	}

	for i, w := range workloads {
		p.handlers[i].beforeWork(w)
	}
	// All of the following goroutines are dependent on the one(s) preceding
	// them, while [wrapper.finishBlock] is dependent on [wrapper.postProcess].
	// The return of [Processor.FinishBlock] is therefore a guarantee of the end
	// of the lifespans of all of these goroutines.
	go func() {
		for _, j := range jobs {
			p.prefetch <- (*prefetch)(j)
		}
	}()
	go func() {
		for _, j := range jobs {
			p.process <- (*process)(j)
		}
	}()
	for _, h := range p.handlers {
		go h.postProcess()
	}
	return nil
}

// FinishBlock propagates its arguments to every [Handler] and resets the
// [Processor] to a state ready for the next block. A return from FinishBlock
// guarantees that all dispatched work from the respective call to
// [Processor.StartBlock] has been completed.
func (p *Processor) FinishBlock(sdb vm.StateDB, b *types.Block, rs types.Receipts) {
	// [Handler.FinishBlock] is allowed to write to state, so these MUST NOT be
	// concurrent.
	for _, h := range p.handlers {
		h.finishBlock(sdb, b, rs)
	}
	for tx := range p.txGas {
		delete(p.txGas, tx)
	}
}

func (p *Processor) shouldProcess(tx IndexedTx, rules params.Rules) (process []bool, retErr error) {
	// An explicit 0 is necessary to avoid [Processor.PreprocessingGasCharge]
	// returning [ErrTxUnknown].
	p.txGas[tx.Hash()] = 0

	process = make([]bool, len(p.handlers))
	var totalCost uint64
	for i, h := range p.handlers {
		do, cost := h.shouldProcess(tx)
		if !do {
			continue
		}
		process[i] = true
		// It's safe to cap total cost at [math.MaxUint64] because intrinsic gas
		// is always non-zero and the tx would therefore OOG. Not that we could
		// reasonably expect such high gas consumption though ¯\_(ツ)_/¯
		totalCost += min(cost, math.MaxUint64-totalCost)
	}

	defer func() {
		if retErr == nil {
			p.txGas[tx.Hash()] = totalCost
		}
	}()

	spent, err := txIntrinsicGas(tx.Transaction, &rules)
	if err != nil {
		return nil, fmt.Errorf("calculating intrinsic gas of %#x: %v", tx.Hash(), err)
	}
	if spent > tx.Gas() {
		// If this happens then consensus has a bug because the tx shouldn't
		// have been included. We include the check, however, for completeness
		// as we would otherwise underflow below.
		return nil, core.ErrIntrinsicGas
	}
	if remain := tx.Gas() - spent; remain < totalCost {
		for i := range process {
			process[i] = false
		}
	}
	return process, nil
}

func txIntrinsicGas(tx *types.Transaction, rules *params.Rules) (uint64, error) {
	return intrinsicGas(tx.Data(), tx.AccessList(), tx.To(), rules)
}

func intrinsicGas(data []byte, access types.AccessList, txTo *common.Address, rules *params.Rules) (uint64, error) {
	create := txTo == nil
	return core.IntrinsicGas(
		data,
		access,
		create,
		rules.IsHomestead,
		rules.IsIstanbul, // EIP-2028
		rules.IsShanghai, // EIP-3860
	)
}

// ErrTxUnknown is returned by [Processor.PreprocessingGasCharge] if it is
// called with a transaction hash that wasn't in the last block passed to
// [Processor.StartBlock].
var ErrTxUnknown = errors.New("transaction unknown by parallel preprocessor")

// PreprocessingGasCharge implements the [vm.Preprocessor] interface and MUST be
// registered via [vm.RegisterHooks] to ensure proper gas accounting.
func (p *Processor) PreprocessingGasCharge(tx common.Hash) (uint64, error) {
	g, ok := p.txGas[tx]
	if !ok {
		return 0, fmt.Errorf("%w: %v", ErrTxUnknown, tx)
	}
	return g, nil
}

var _ vm.Preprocessor = (*Processor)(nil)
