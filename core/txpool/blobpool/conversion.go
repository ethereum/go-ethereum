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

package blobpool

import (
	"errors"
	"slices"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// maxPendingConversionTasks caps the number of pending conversion tasks. This
// prevents excessive memory usage; the worst-case scenario (2k transactions
// with 6 blobs each) would consume approximately 1.5GB of memory.
const maxPendingConversionTasks = 2048

type convertResult struct {
	ptx *BlobTxForPool
	err error
}

// txConvert represents a conversion task with an attached legacy blob transaction.
type txConvert struct {
	tx   *types.Transaction // Legacy blob transaction
	done chan convertResult // Channel for signaling back if the conversion succeeds
}

// conversionQueue is a dedicated queue for converting legacy blob transactions
// received from the network after the Osaka fork. Since conversion is expensive,
// it is performed in the background by a single thread, ensuring the main Geth
// process is not overloaded.
type conversionQueue struct {
	tasks           chan *txConvert
	startConversion chan func()
	quit            chan struct{}
	closed          chan struct{}

	queue    []func()
	taskDone chan struct{}
}

// newConversionQueue constructs the conversion queue.
func newConversionQueue() *conversionQueue {
	q := &conversionQueue{
		tasks:           make(chan *txConvert),
		startConversion: make(chan func()),
		quit:            make(chan struct{}),
		closed:          make(chan struct{}),
	}
	go q.loop()
	return q
}

// convert accepts a legacy blob transaction with version-0 blobs and queues it
// for conversion.
//
// This function may block for a long time until the transaction is processed.
func (q *conversionQueue) convert(tx *types.Transaction) (*BlobTxForPool, error) {
	done := make(chan convertResult, 1)
	select {
	case q.tasks <- &txConvert{tx: tx, done: done}:
		res := <-done
		return res.ptx, res.err
	case <-q.closed:
		return nil, errors.New("conversion queue closed")
	}
}

// launchConversion starts a conversion task in the background.
func (q *conversionQueue) launchConversion(fn func()) error {
	select {
	case q.startConversion <- fn:
		return nil
	case <-q.closed:
		return errors.New("conversion queue closed")
	}
}

// close terminates the conversion queue.
func (q *conversionQueue) close() {
	select {
	case <-q.closed:
		return
	default:
		close(q.quit)
		<-q.closed
	}
}

// run converts a batch of legacy blob txs to the new cell proof format.
func (q *conversionQueue) run(tasks []*txConvert, done chan struct{}, interrupt *atomic.Int32) {
	defer close(done)

	for _, t := range tasks {
		if interrupt != nil && interrupt.Load() != 0 {
			t.done <- convertResult{err: errors.New("conversion is interrupted")}
			continue
		}
		// Run the conversion, the original sidecar will be mutated in place
		start := time.Now()
		ptx, err := newBlobTxForPool(t.tx)
		t.done <- convertResult{ptx: ptx, err: err}
		log.Trace("Converted legacy blob tx", "hash", t.tx.Hash(), "err", err, "elapsed", common.PrettyDuration(time.Since(start)))
	}
}

func (q *conversionQueue) loop() {
	defer close(q.closed)

	var (
		done      chan struct{} // Non-nil if background routine is active
		interrupt *atomic.Int32 // Flag to signal conversion interruption

		// The pending tasks for sidecar conversion. We assume the number of legacy
		// blob transactions requiring conversion will not be excessive. However,
		// a hard cap is applied as a protective measure.
		txTasks []*txConvert
	)

	for {
		select {
		case t := <-q.tasks:
			if len(txTasks) >= maxPendingConversionTasks {
				t.done <- convertResult{err: errors.New("conversion queue is overloaded")}
				continue
			}
			txTasks = append(txTasks, t)

			// Launch the background conversion thread if it's idle
			if done == nil {
				done, interrupt = make(chan struct{}), new(atomic.Int32)

				tasks := slices.Clone(txTasks)
				txTasks = txTasks[:0]
				go q.run(tasks, done, interrupt)
			}

		case <-done:
			done, interrupt = nil, nil
			if len(txTasks) > 0 {
				done, interrupt = make(chan struct{}), new(atomic.Int32)
				tasks := slices.Clone(txTasks)
				txTasks = txTasks[:0]
				go q.run(tasks, done, interrupt)
			}

		case fn := <-q.startConversion:
			q.queue = append(q.queue, fn)
			if q.taskDone == nil {
				q.runNextTask()
			}

		case <-q.taskDone:
			q.runNextTask()

		case <-q.quit:
			if done != nil {
				log.Debug("Waiting for blob proof conversion to exit")
				interrupt.Store(1)
				<-done
			}
			if q.taskDone != nil {
				log.Debug("Waiting for blobpool billy conversion to exit")
				<-q.taskDone
			}
			// Signal any tasks that were queued for the next batch but never started
			// so callers blocked in convert() receive an error instead of hanging.
			for _, t := range txTasks {
				// Best-effort notify; t.done is a buffered channel of size 1
				// created by convert(), and we send exactly once per task.
				t.done <- convertResult{err: errors.New("conversion queue closed")}
			}
			// Drop references to allow GC of the backing array.
			txTasks = txTasks[:0]
			return
		}
	}
}

func (q *conversionQueue) runNextTask() {
	if len(q.queue) == 0 {
		q.taskDone = nil
		return
	}
	fn := q.queue[0]
	q.queue = append(q.queue[:0], q.queue[1:]...)

	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	q.taskDone = done
}
