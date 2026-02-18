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

package parallel

import (
	"sync"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/stateconf"
)

// A Handler is responsible for processing [types.Transactions] in an
// embarrassingly parallel fashion. It is the responsibility of the Handler to
// determine whether this is possible, typically only so if one of the following
// is true with respect to a precompile associated with the Handler:
//
//  1. The destination address is that of the precompile; or
//  2. At least one [types.AccessTuple] references the precompile's address.
//
// Scenario (2) allows precompile access to be determined through inspection of
// the [types.Transaction] alone, without the need for execution.
//
// A [Processor] will orchestrate calling of Handler methods as follows:
//
//	|                                      - Prefetch(i) - Process(i)
//	|                                    /                        /
//	| BeforeBlock() - ShouldProcess(0..n)                         - PostProcess() - AfterBlock()
//	|                                    \                        \
//	|                                      - Prefetch(j) - Process(j)
//
// IntRA-Handler guarantees:
//
//  1. BeforeBlock() precedes all ShouldProcess() calls.
//  2. ShouldProcess() calls are sequential, in the same order as transactions in the block.
//  3. Prefetch() precedes the respective Process() call. Not called if ShouldProcess() returns false.
//  4. PostProcess() precedes AfterBlock().
//
// Note that PostProcess() MAY be called at any time after BeforeBlock(), and
// implementations MUST synchronise with Process() by using the [Results]. There
// are no intER-Handler guarantees except that AfterBlock() methods are called
// sequentially, in the same order as they were registered with [AddHandler].
//
// All [libevm.StateReader] instances are opened to the state at the beginning
// of the block. The [StateDB] is the same one used to execute the block, before
// being committed, and MAY be written to.
type Handler[CommonData, Data, Result, Aggregated any] interface {
	// BeforeBlock is called before all calls to ShouldProcess() on this
	// Handler.
	BeforeBlock(libevm.StateReader, *types.Header) CommonData
	// ShouldProcess reports whether the Handler SHOULD receive the transaction
	// for processing and, if so, how much gas to charge. Processing is
	// performed i.f.f. the returned boolean is true and there is sufficient gas
	// limit to cover intrinsic gas for all Handlers that returned true. If
	// there is insufficient gas for processing then the transaction will result
	// in [vm.ErrOutOfGas] as long as the [Processor] is registered with
	// [vm.RegisterHooks] as a [vm.Preprocessor].
	//
	// Implementations MUST NOT perform any meaningful computation
	// but MAY perform inter-transaction checks such as, for example,
	// deduplication of work.
	ShouldProcess(IndexedTx, CommonData) (do bool, gas uint64)
	// Prefetch is called before the respective call to Process() on this
	// Handler. It MUST NOT perform any meaningful computation beyond what is
	// necessary to determine the necessary state to propagate to Process().
	Prefetch(libevm.StateReader, IndexedTx, CommonData) Data
	// Process is responsible for performing all meaningful, per-transaction
	// computation. It receives the common data returned by the single call to
	// BeforeBlock() as well as the data from the respective call to Prefetch().
	// The returned result is propagated to PostProcess() and any calls to the
	// function returned by [AddHandler].
	//
	// NOTE: if the result is exposed to the EVM via a precompile then said
	// precompile will block until Process() returns. While this guarantees the
	// availability of pre-processed results, it is also the hot path for EVM
	// transactions.
	Process(libevm.StateReader, IndexedTx, CommonData, Data) Result
	// PostProcess is called concurrently with all calls to Process(). It allows
	// for online aggregation of results into a format ready for writing to
	// state.
	//
	// NOTE: although PostProcess() MAY perform computation, it will block the
	// calling of AfterBlock() and hence also the execution of the next block.
	PostProcess(CommonData, Results[Result]) Aggregated
	// AfterBlock is called after PostProcess() returns and all regular EVM
	// transaction processing is complete. It MUST NOT perform any meaningful
	// computation beyond what is necessary to (a) parse receipts, and (b)
	// persist aggregated results.
	AfterBlock(StateDB, Aggregated, *types.Block, types.Receipts)
}

// An IndexedTx couples a [types.Transaction] with its index in a block.
type IndexedTx struct {
	Index int
	*types.Transaction
}

// Results provides mechanisms for blocking on the output of [Handler.Process].
type Results[R any] struct {
	WaitForAll            func()
	TxOrder, ProcessOrder <-chan TxResult[R]
}

// A TxResult couples an [IndexedTx] with its respective result from
// [Handler.Process].
type TxResult[R any] struct {
	Tx     IndexedTx
	Result R
}

// StateDB is the subset of [state.StateDB] methods that MAY be called by
// [Handler.AfterBlock].
type StateDB interface {
	libevm.StateReader
	SetState(_ common.Address, key, val common.Hash, _ ...stateconf.StateDBStateOption)
}

var _ handler = (*wrapper[any, any, any, any])(nil)

// A wrapper exposes the generic functionality of a [Handler] in a non-generic
// manner, allowing [Processor] to be free of type parameters.
type wrapper[CD, D, R, A any] struct {
	Handler[CD, D, R, A]

	totalTxsInBlock   int
	txsBeingProcessed sync.WaitGroup

	common eventual[CD]
	data   []eventual[D]

	results                []eventual[result[R]]
	whenProcessed, txOrder chan TxResult[R]

	aggregated eventual[A]
}

// AddHandler registers the [Handler] with the [Processor] and returns a
// function to fetch the [TxResult] for the i'th transaction passed to
// [Processor.StartBlock].
//
// The returned function until the respective transaction has had its result
// processed, and then returns the value returned by the [Handler]. The returned
// boolean will be false if no processing occurred, either because the [Handler]
// indicated as such or because the transaction supplied insufficient gas.
//
// Multiple calls to Result with the same argument are allowed. Callers MUST NOT
// charge the gas price for preprocessing as this is handled by
// [Processor.PreprocessingGasCharge] if registered as a [vm.Preprocessor].
//
// Within the scope of a given block, the same value will be returned by each
// call with the same argument, such that if R is a pointer then modifications
// will persist between calls. However, the caller does NOT have mutually
// exclusive access to the [TxResult] so SHOULD NOT modify it, especially since
// the result MAY also be accessed by [Handler.PostProcess], with no ordering
// guarantees.
func AddHandler[CD, D, R, A any](p *Processor, h Handler[CD, D, R, A]) func(txIndex int) (TxResult[R], bool) {
	w := &wrapper[CD, D, R, A]{
		Handler:    h,
		common:     eventually[CD](),
		aggregated: eventually[A](),
	}
	p.handlers = append(p.handlers, w)
	return w.result
}

func (w *wrapper[CD, D, R, A]) beforeBlock(sdb libevm.StateReader, b *types.Block) {
	w.totalTxsInBlock = len(b.Transactions())
	// We can reuse the channels already in the data and results slices because
	// they're emptied by [wrapper.process] and [wrapper.finishBlock]
	// respectively.
	for i := len(w.results); i < w.totalTxsInBlock; i++ {
		w.data = append(w.data, eventually[D]())
		w.results = append(w.results, eventually[result[R]]())
	}

	go func() {
		// goroutine guaranteed to have completed by the time a respective
		// getter unblocks (i.e. in any call to [wrapper.prefetch]).
		w.common.put(w.BeforeBlock(sdb, types.CopyHeader(b.Header())))
	}()
}

func (w *wrapper[CD, D, R, A]) shouldProcess(tx IndexedTx) (do bool, gas uint64) {
	return w.Handler.ShouldProcess(tx, w.common.peek())
}

func (w *wrapper[CD, D, R, A]) beforeWork(jobs int) {
	w.txsBeingProcessed.Add(jobs)
	w.whenProcessed = make(chan TxResult[R], jobs)
	w.txOrder = make(chan TxResult[R], jobs)
	go func() {
		w.txsBeingProcessed.Wait()
		close(w.whenProcessed)
	}()
}

func (w *wrapper[CD, D, R, A]) prefetch(sdb libevm.StateReader, job *prefetch) {
	w.data[job.tx.Index].put(w.Prefetch(sdb, job.tx, w.common.peek()))
}

func (w *wrapper[CD, D, R, A]) process(sdb libevm.StateReader, job *process) {
	defer w.txsBeingProcessed.Done()

	idx := job.tx.Index
	val := w.Process(sdb, job.tx, w.common.peek(), w.data[idx].take())
	r := result[R]{
		tx:  job.tx,
		val: &val,
	}
	w.results[idx].put(r)
	w.whenProcessed <- TxResult[R]{
		Tx:     job.tx,
		Result: val,
	}
}

func (w *wrapper[CD, D, R, A]) nullResult(job *job) {
	w.results[job.tx.Index].put(result[R]{
		tx:  job.tx,
		val: nil,
	})
}

func (w *wrapper[CD, D, R, A]) result(i int) (TxResult[R], bool) {
	r := w.results[i].peek()

	txr := TxResult[R]{
		Tx: r.tx,
	}
	if r.val == nil {
		return txr, false
	}
	txr.Result = *r.val
	return txr, true
}

func (w *wrapper[CD, D, R, A]) postProcess() {
	go func() {
		defer close(w.txOrder)
		for i := range w.totalTxsInBlock {
			r, ok := w.result(i)
			if !ok {
				continue
			}
			w.txOrder <- r
		}
	}()

	res := Results[R]{
		WaitForAll:   w.txsBeingProcessed.Wait,
		TxOrder:      w.txOrder,
		ProcessOrder: w.whenProcessed,
	}
	w.aggregated.put(w.PostProcess(w.common.peek(), res))
}

func (w *wrapper[CD, D, R, A]) finishBlock(sdb vm.StateDB, b *types.Block, rs types.Receipts) {
	w.AfterBlock(sdb, w.aggregated.take(), b, rs)

	// [wrapper.postProcess] is guaranteed to have finished because it sets
	// [wrapper.aggregated], from which we have just read. However
	// [Handler.PostProcess] is under no obligation to block on anything, and
	// the goroutine filling [wrapper.txOrder] might still be reading results.
	// We therefore guarantee its completion before "getting and keeping" all of
	// [wrapper.results] otherwise said goroutine can leak.
	for range w.txOrder {
		// Nobody needs these anymore, but we need to know that the channel has
		// been closed.
	}
	// Although we know this will unblock effectively immediately, it's safer to
	// verify the intuition than to rely on complex reasoning.
	w.txsBeingProcessed.Wait()

	w.common.take()
	for _, v := range w.results[:w.totalTxsInBlock] {
		// Every result channel is guaranteed to have some value in its buffer
		// because [Processor.BeforeBlock] either sends a nil *R or it
		// dispatches a job, which will send a non-nil *R.
		v.take()
	}
}
