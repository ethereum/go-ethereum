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

package common

import "fmt"

// Arena is an allocation primitive that allows individual allocations
// out of a page of items and bulk de-allocations of last N allocations.
// The most common way of using an Arena is to pair it up with a sync.Pool
// of pages.
//
// Notably, arena is not thread safe, please manage the concurrency issues
// by yourselves.
type Arena[T any] struct {
	used        uint32
	pages       [][]T
	pageSize    uint32
	newPage     func() any
	releasePage func(any)
}

// NewArena constructs the arena for the given type.
func NewArena[T any](pageSize uint32, newPage func() any, releasePage func(any)) *Arena[T] {
	return &Arena[T]{
		pageSize:    pageSize,
		newPage:     newPage,
		releasePage: releasePage,
	}
}

// Alloc returns the next free item on the arena.
func (a *Arena[T]) Alloc() *T {
	pageIndex := a.used / a.pageSize
	pageOffset := a.used % a.pageSize

	// Allocate additional items if all pre-allocated ones have been exhausted
	if pageOffset == 0 && int(pageIndex) == len(a.pages) {
		items := a.newPage().([]T)
		if len(items) != int(a.pageSize) {
			panic(fmt.Errorf("invalid page size, want: %d, got: %d", a.pageSize, len(items)))
		}
		a.pages = append(a.pages, items)
	}
	a.used++
	return &a.pages[pageIndex][pageOffset]
}

// Used returns the number of items that live on this arena.
func (a *Arena[T]) Used() uint32 {
	return a.used
}

// Reset rollbacks the active set of live elements to the given number.
func (a *Arena[T]) Reset(to uint32) {
	a.used = to
}

// Release releases all the pages that the arena currently owns
func (a *Arena[T]) Release() {
	for _, page := range a.pages {
		a.releasePage(page)
	}
	a.pages = nil
	a.used = 0
}
