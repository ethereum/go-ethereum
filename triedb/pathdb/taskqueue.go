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

import "sync"

// taskQueueLimit is the maximum number of the pending tasks in a task queue.
// Scheduling blocks once the limit is reached, throttling the producer to the
// pace of the background thread instead of accumulating an unbounded backlog.
const taskQueueLimit = 64

// taskQueue runs the scheduled tasks one by one in a background thread, in the
// order of scheduling. The thread terminates once all the tasks are done.
type taskQueue struct {
	lock  sync.Mutex
	cond  *sync.Cond    // signaled when a pending task is taken by the thread
	tasks []func()      // pending tasks, bounded by taskQueueLimit
	done  chan struct{} // closed when the thread terminates, nil if not running
}

// newTaskQueue constructs an idle task queue.
func newTaskQueue() *taskQueue {
	q := &taskQueue{}
	q.cond = sync.NewCond(&q.lock)
	return q
}

// schedule appends the task, launching the thread if it's not active. It blocks
// if the pending tasks reach the limit, until the thread catches up.
func (q *taskQueue) schedule(task func()) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for len(q.tasks) >= taskQueueLimit {
		q.cond.Wait()
	}
	q.tasks = append(q.tasks, task)

	if q.done == nil {
		q.done = make(chan struct{})
		go q.run(q.done)
	}
}

// run executes the scheduled tasks in order, terminating once all are done.
func (q *taskQueue) run(done chan struct{}) {
	for {
		q.lock.Lock()
		if len(q.tasks) == 0 {
			q.done = nil
			q.lock.Unlock()
			close(done)
			return
		}
		task := q.tasks[0]
		q.tasks[0] = nil
		q.tasks = q.tasks[1:]
		q.cond.Signal()
		q.lock.Unlock()

		task()
	}
}

// wait blocks until the scheduled tasks are done.
func (q *taskQueue) wait() {
	q.lock.Lock()
	done := q.done
	q.lock.Unlock()

	if done != nil {
		<-done
	}
}
