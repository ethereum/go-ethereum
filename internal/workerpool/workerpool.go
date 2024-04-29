// Copyright 2024 The go-ethereum Authors
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

// Package workerpool implements a concurrent task processor.
package workerpool

import (
	"runtime"
	"sync"
)

// WorkerPool is a concurrent task processor, scheduling and running tasks from
// a source channel, feeding any errors into a sink.
type WorkerPool[T any, R any] struct {
	tasks   chan T         // Input channel waiting to consume tasks
	results chan R         // Result channel for consuers to wait on
	working sync.WaitGroup // Waitgroup blocking on worker liveness
}

// New creates a worker pool with the given number of max task capacity and an
// optional goroutine count to execute on. If 0 threads are requested, the pool
// will default to the number of (logical) CPUs.
func New[T any, R any](tasks int, threads int, f func(T) R) *WorkerPool[T, R] {
	// Create the worker pool
	pool := &WorkerPool[T, R]{
		tasks:   make(chan T, tasks),
		results: make(chan R, tasks),
	}
	// Start all the data processor routines
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	pool.working.Add(threads)
	for i := 0; i < threads; i++ {
		go pool.work(f)
	}
	return pool
}

// Close signals the end of the task stream. It does not block execution, rather
// returns immediately and users have to explicitly call Wait to block until the
// pool actually spins down. Alternatively, consumers can read the results chan,
// which will be closed after the last result is delivered.
//
// Calling Close multiple times will panic. Not particularly hard to avoid, but
// it's really a programming error.
func (pool *WorkerPool[T, R]) Close() {
	close(pool.tasks)
	go func() {
		pool.working.Wait()
		close(pool.results)
	}()
}

// Wait blocks until all the scheduled tasks have been processed.
func (pool *WorkerPool[T, R]) Wait() {
	pool.working.Wait()
}

// Schedule adds a task to the work queue.
func (pool *WorkerPool[T, R]) Schedule(task T) {
	pool.tasks <- task
}

// Results retrieves the result channel to consume the output of the individual
// work tasks. The channel will be closed after all tasks are done.
//
// Note, as long as the number of actually scheduled tasks are smaller or equal
// to the requested number form the constructor, it's fine to not consume this
// channel.
func (pool *WorkerPool[T, R]) Results() chan R {
	return pool.results
}

// work is the (one of many) goroutine consuming input tasks and executing them
// to compute the results.
func (pool *WorkerPool[T, R]) work(f func(T) R) {
	defer pool.working.Done()
	for task := range pool.tasks {
		pool.results <- f(task)
	}
}
