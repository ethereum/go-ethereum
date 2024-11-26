// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

// Package sync extends the standard library's sync package.
package sync

import "sync"

// Aliases of stdlib sync's types to avoid having to import it alongside this
// package.
type (
	Cond      = sync.Cond
	Locker    = sync.Locker
	Map       = sync.Map
	Mutex     = sync.Mutex
	Once      = sync.Once
	RWMutex   = sync.RWMutex
	WaitGroup = sync.WaitGroup
)

// A Pool is a type-safe wrapper around [sync.Pool].
type Pool[T any] struct {
	New  func() T
	pool sync.Pool
	once Once
}

// Get is equivalent to [sync.Pool.Get].
func (p *Pool[T]) Get() T {
	p.once.Do(func() { // Do() guarantees at least once, not just only once
		p.pool.New = func() any { return p.New() }
	})
	return p.pool.Get().(T) //nolint:forcetypeassert
}

// Put is equivalent to [sync.Pool.Put].
func (p *Pool[T]) Put(t T) {
	p.pool.Put(t)
}
