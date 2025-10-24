// Copyright 2025 The go-ethereum Authors
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

// txConvert represents a conversion task with an attached legacy blob transaction.
type txConvert struct {
	tx   *types.Transaction // Legacy blob transaction
	done chan error         // Channel for signaling back if the conversion succeeds
}

// conversionQueue is a dedicated queue for converting legacy blob transactions
// received from the network after the Osaka fork. Since conversion is expensive,
// it is performed in the background by a single thread, ensuring the main Geth
// process is not overloaded.
type conversionQueue struct {
	tasks      chan *txConvert
	startBilly chan func()
	quit       chan struct{}
	closed     chan struct{}

	billyQueue    []func()
	billyTaskDone chan struct{}

	// This channel will be closed when the first billy conversion finishes.
	// It's added for unit tests to synchronize with the conversion progress.
	anyBillyConversionDone chan struct{}
}

// newConversionQueue constructs the conversion queue.
func newConversionQueue() *conversionQueue {
	q := &conversionQueue{
		tasks:                  make(chan *txConvert),
		startBilly:             make(chan func()),
		quit:                   make(chan struct{}),
		closed:                 make(chan struct{}),
		anyBillyConversionDone: make(chan struct{}),
	}
	go q.loop()
	return q
}

// convert accepts a legacy blob transaction with version-0 blobs and queues it
// for conversion.
//
// This function may block for a long time until the transaction is processed.
func (q *conversionQueue) convert(tx *types.Transaction) error {
	done := make(chan error, 1)
	select {
	case q.tasks <- &txConvert{tx: tx, done: done}:
		return <-done
	case <-q.closed:
		return errors.New("conversion queue closed")
	}
}

// launchBillyConversion starts a conversion task in the background.
func (q *conversionQueue) launchBillyConversion(fn func()) error {
	select {
	case q.startBilly <- fn:
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
			t.done <- errors.New("conversion is interrupted")
			continue
		}
		sidecar := t.tx.BlobTxSidecar()
		if sidecar == nil {
			t.done <- errors.New("tx without sidecar")
			continue
		}
		// Run the conversion, the original sidecar will be mutated in place
		start := time.Now()
		err := sidecar.ToV1()
		t.done <- err
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

		firstBilly = true
	)

	for {
		select {
		case t := <-q.tasks:
			if len(txTasks) >= maxPendingConversionTasks {
				t.done <- errors.New("conversion queue is overloaded")
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

		case fn := <-q.startBilly:
			q.billyQueue = append(q.billyQueue, fn)
			q.runNextBillyTask()

		case <-q.billyTaskDone:
			if firstBilly {
				close(q.anyBillyConversionDone)
				firstBilly = false
			}
			q.runNextBillyTask()

		case <-q.quit:
			if done != nil {
				log.Debug("Waiting for blob proof conversion to exit")
				interrupt.Store(1)
				<-done
			}
			if q.billyTaskDone != nil {
				log.Debug("Waiting for blobpool billy conversion to exit")
				<-q.billyTaskDone
			}
			return
		}
	}
}

func (q *conversionQueue) runNextBillyTask() {
	if len(q.billyQueue) == 0 {
		q.billyTaskDone = nil
		return
	}

	fn := q.billyQueue[0]
	q.billyQueue = append(q.billyQueue[:0], q.billyQueue[1:]...)

	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	q.billyTaskDone = done
}
