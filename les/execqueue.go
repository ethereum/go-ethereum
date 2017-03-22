// Copyright 2017 The go-ethereum Authors
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

package les

import (
	"sync/atomic"
)

// ExecQueue implements a queue that executes function calls in a single thread,
// in the same order as they have been queued.
type execQueue struct {
	chn                 chan func()
	cnt, stop, capacity int32
}

// NewExecQueue creates a new execution queue.
func newExecQueue(capacity int32) *execQueue {
	q := &execQueue{
		chn:      make(chan func(), capacity),
		capacity: capacity,
	}
	go q.loop()
	return q
}

func (q *execQueue) loop() {
	for f := range q.chn {
		atomic.AddInt32(&q.cnt, -1)
		if atomic.LoadInt32(&q.stop) != 0 {
			return
		}
		f()
	}
}

// CanQueue returns true if more  function calls can be added to the execution queue.
func (q *execQueue) canQueue() bool {
	return atomic.LoadInt32(&q.stop) == 0 && atomic.LoadInt32(&q.cnt) < q.capacity
}

// Queue adds a function call to the execution queue. Returns true if successful.
func (q *execQueue) queue(f func()) bool {
	if atomic.LoadInt32(&q.stop) != 0 {
		return false
	}
	if atomic.AddInt32(&q.cnt, 1) > q.capacity {
		atomic.AddInt32(&q.cnt, -1)
		return false
	}
	q.chn <- f
	return true
}

// Stop stops the exec queue.
func (q *execQueue) quit() {
	atomic.StoreInt32(&q.stop, 1)
}
