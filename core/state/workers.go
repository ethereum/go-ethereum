// Copyright 2019 The go-ethereum Authors
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

package state

import "runtime"

// workers is a singleton pool of goroutines to run arbitrary tasks concurrently.
var workers = newWorkerPool(16 * runtime.GOMAXPROCS(0))

// workerPool is a set of goroutines that execute arbitrary tasks concurrently.
type workerPool struct {
	tasks chan func()
}

// newWorkerPool creates the worker task channel and launches the goroutines.
func newWorkerPool(queue int) *workerPool {
	pool := &workerPool{
		tasks: make(chan func(), queue),
	}
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			for task := range pool.tasks {
				task()
			}
		}()
	}
	return pool
}

// Schedule inserts a new task into the worker pool.
func (pool *workerPool) Schedule(fn func()) {
	pool.tasks <- fn
}
