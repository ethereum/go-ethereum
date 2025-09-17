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

	"github.com/ethereum/go-ethereum/core/types"
)

// cTask represents a conversion task with an attached result channel.
type cTask struct {
	tx   *types.Transaction // Blob transaction, sidecar is expected
	done chan error         // Channel for signaling back if the conversion succeeds
}

// conversionQueue is a dedicated queue for converting legacy blob transactions
// received from the network after the Osaka fork. Since conversion is expensive,
// it is performed in the background by a single thread, ensuring the main Geth
// process is not overloaded.
type conversionQueue struct {
	tasks  chan *cTask
	quit   chan struct{}
	closed chan struct{}
}

// newConversionQueue constructs the conversion queue.
func newConversionQueue() *conversionQueue {
	q := &conversionQueue{
		tasks:  make(chan *cTask),
		quit:   make(chan struct{}),
		closed: make(chan struct{}),
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
	case q.tasks <- &cTask{tx: tx, done: done}:
		return <-done
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

func (q *conversionQueue) run(tasks []*cTask, done chan struct{}) {
	defer close(done)

	for _, t := range tasks {
		sidecar := t.tx.BlobTxSidecar()
		if sidecar == nil {
			t.done <- errors.New("tx without sidecar")
			continue
		}
		// Run the conversion, the original sidecar will be mutated in place
		t.done <- sidecar.ToV1()
	}
}

func (q *conversionQueue) loop() {
	defer close(q.closed)

	var (
		done   = make(chan struct{})
		cTasks []*cTask
	)
	for {
		select {
		case t := <-q.tasks:
			cTasks = append(cTasks, t)
			if done == nil {
				done = make(chan struct{})

				tasks := cTasks
				cTasks = cTasks[:0]
				go q.run(tasks, done)
			}
		case <-done:
			done = nil

		case <-q.quit:
			return
		}
	}
}
