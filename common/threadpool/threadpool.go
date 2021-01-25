// Copyright 2020 The go-ethereum Authors
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

package threadpool

// ThreadPool allows to create a max amount of threads.
// Users can query how many threads they are allowed to create.
type ThreadPool struct {
	pool chan struct{}
	max  int
}

// NewThreadPool creates a new Threadpool with
func NewThreadPool(maxThreads int) *ThreadPool {
	tp := ThreadPool{
		pool: make(chan struct{}, maxThreads),
		max:  maxThreads,
	}
	for i := 0; i < maxThreads; i++ {
		tp.pool <- struct{}{}
	}
	return &tp
}

// Get requests `tasks` amount of threads from the pool.
// If the pool is not used much, a caller can get up to 1/3 of the available threads.
// Otherwise the caller gets only a single thread (once available).
// If the pool has more than `tasks` threads available it will only return `tasks`.
// It uses len(chan) which is a bit racy but shouldn't matter to much.
func (t *ThreadPool) Get(tasks int) int {
	threads := 1
	if len(t.pool) > t.max/2 {
		threads = len(t.pool) / 3
	}
	if tasks > 0 && threads > tasks {
		threads = tasks
	}
	for i := 0; i < threads; i++ {
		<-t.pool
	}
	return threads
}

// Put returns n threads back to the pool.
func (t *ThreadPool) Put(threads int) {
	for i := 0; i < threads; i++ {
		t.pool <- struct{}{}
	}
}
