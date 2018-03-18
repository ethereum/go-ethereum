// Copyright 2018 The go-ethereum Authors
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

// Package flowcontrol implements a client side flow control mechanism
package les

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/common/prque"
)

// servingQueue runs serving tasks in a limited number of threads and puts the
// waiting tasks in a priority queue
type servingQueue struct {
	lock        sync.Mutex
	threadCount int                 // number of currently running threads
	stopCount   int                 // number of threads to be stopped after they finish their current task
	queue       *prque.Prque        // priority queue for waiting or suspended tasks
	best        *servingTask        // either best == nil (queue empty) or waitingForTask is empty
	waiting     []chan *servingTask // threads waiting for a task
	suspendBias int64               // priority bias against suspending an already running task
}

// servingTask represents a request serving task. Tasks can be implemented to
// run in multiple steps, allowing the serving queue to suspend execution between
// steps if higher priority tasks are entered. The creator of the task should
// set the following fields:
//
// - priority: greater value means higher priority; values can wrap around the int64 range
// - run: execute a single step; return true if finished
// - after: executed after run finishes or returns an error, receives the total serving time
type servingTask struct {
	servingTime uint64
	done        bool
	err         error
	priority    int64
	run         func() (finished bool, err error)
	after       func(servingTime uint64, err error)
}

// newServingQueue returns a new servingQueue
func newServingQueue(_suspendBias int64) *servingQueue {
	return &servingQueue{
		queue:       prque.New(nil),
		suspendBias: _suspendBias,
	}
}

// addTask adds a new task, either starting it immediately or queueing it
func (sq *servingQueue) addTask(task *servingTask) {
	sq.lock.Lock()
	defer sq.lock.Unlock()

	if l := len(sq.waiting); l != 0 {
		l--
		sq.waiting[l] <- task
		sq.waiting = sq.waiting[:l]
		return
	}

	if sq.best == nil {
		sq.best = task
		return
	}
	if task.priority < sq.best.priority {
		sq.queue.Push(sq.best, sq.best.priority)
		sq.best = task
		return
	}
	sq.queue.Push(task, task.priority)
}

// getNewTask selects a new task to be processed. If blocking == true then it waits
// until a runnable task arrives or returns nil if the thread should be stopped.
// if currentTask != nil then it returns immediately and only returns a new task
// if the current one should be suspended.
// Note: either blocking should be false or currentTask should be nil.
func (sq *servingQueue) getNewTask(currentTask *servingTask, blocking bool) *servingTask {
	sq.lock.Lock()
	if sq.stopCount == 0 {
		if sq.best != nil && (currentTask == nil || sq.best.priority <= currentTask.priority-sq.suspendBias) {
			best := sq.best
			sq.best, _ = sq.queue.PopItem().(*servingTask)
			sq.lock.Unlock()
			return best
		}
		if blocking {
			ch := make(chan *servingTask)
			sq.waiting = append(sq.waiting, ch)
			sq.lock.Unlock()
			return <-ch
		}
	} else {
		sq.stopCount--
		sq.threadCount--
	}
	sq.lock.Unlock()
	return nil
}

// setThreads sets the processing thread count, suspending tasks as soon as
// possible if necessary.
func (sq *servingQueue) setThreads(threadCount int) {
	sq.lock.Lock()
	defer sq.lock.Unlock()

	diff := threadCount - sq.threadCount + sq.stopCount
	if diff > 0 {
		// start more threads
		if sq.stopCount >= diff {
			sq.stopCount -= diff
		} else {
			diff -= sq.stopCount
			sq.stopCount = 0
			for ; diff > 0; diff-- {
				go sq.servingThread()
			}
		}
	}
	if diff < 0 {
		// stop some threads
		lw := len(sq.waiting)
		for diff < 0 && lw > 0 {
			diff++
			lw--
			sq.waiting[lw] <- nil
		}
		sq.waiting = sq.waiting[:lw]
		sq.stopCount += diff
	}
}

// stop stops task processing as soon as possible
func (sq *servingQueue) stop() {
	sq.setThreads(0)
}

// servingThread implements a single serving thread
func (sq *servingQueue) servingThread() {
	for {
		task := sq.getNewTask(nil, true)
		if task == nil {
			return
		}
		task.servingTime -= uint64(mclock.Now())
		for {
			task.done, task.err = task.run()
			if task.done || task.err != nil {
				task.servingTime += uint64(mclock.Now())
				task.after(task.servingTime, task.err)
				break
			}
			if newTask := sq.getNewTask(task, false); newTask != nil {
				now := uint64(mclock.Now())
				task.servingTime += now
				sq.addTask(task)
				task = newTask
				task.servingTime -= now
			}
		}
	}
}
