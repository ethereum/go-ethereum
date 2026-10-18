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

package pathdb

import (
	"errors"
	"sync"
)

// errWriterClosed is returned if a write is scheduled after the writer is closed.
var errWriterClosed = errors.New("async writer closed")

// asyncWriter runs the writes of the committed layers in background, strictly
// in the commit order, taking them off the block commit path.
type asyncWriter struct {
	tasks  chan func() error
	lock   sync.Mutex
	cond   *sync.Cond
	queued uint64 // the number of the scheduled tasks
	done   uint64 // the number of the finished tasks
	err    error  // the error of the first failed task
	closed bool   // whether the writer is closed
}

// newAsyncWriter constructs the writer and starts its background thread.
func newAsyncWriter() *asyncWriter {
	w := &asyncWriter{
		tasks: make(chan func() error, 128),
	}
	w.cond = sync.NewCond(&w.lock)

	go w.loop()
	return w
}

// loop runs the scheduled tasks in order until the writer is closed.
func (w *asyncWriter) loop() {
	for task := range w.tasks {
		err := task()

		w.lock.Lock()
		if err != nil && w.err == nil {
			w.err = err
		}
		w.done += 1
		w.cond.Broadcast()
		w.lock.Unlock()
	}
}

// submit schedules the task, returning the error of any failed task before
// instead.
//
// It must not run concurrently with itself, the submissions are serialized
// by the database, keeping the tasks run in the order they are counted.
func (w *asyncWriter) submit(task func() error) error {
	w.lock.Lock()
	if w.err != nil {
		w.lock.Unlock()
		return w.err
	}
	if w.closed {
		w.lock.Unlock()
		return errWriterClosed
	}
	w.queued += 1
	w.lock.Unlock()

	w.tasks <- task
	return nil
}

// wait blocks until the tasks scheduled so far are done, returning the error of
// the first failed task if any.
//
// The tasks scheduled afterwards are not waited for, so the continuous submissions
// can never stall the caller.
func (w *asyncWriter) wait() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	for queued := w.queued; w.done < queued; {
		w.cond.Wait()
	}
	return w.err
}

// close waits for the scheduled tasks and terminates the background thread.
// No task can be scheduled afterwards. It must not run concurrently with submit,
// both are serialized by the database.
func (w *asyncWriter) close() error {
	err := w.wait()

	w.lock.Lock()
	defer w.lock.Unlock()

	if !w.closed {
		w.closed = true
		close(w.tasks)
	}
	return err
}
