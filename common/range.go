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

import (
	"iter"
)

// Range represents a range of integers.
type Range[T uint32 | uint64] struct {
	first, afterLast T
}

// NewRange creates a new range based of first element and number of elements.
func NewRange[T uint32 | uint64](first, count T) Range[T] {
	return Range[T]{first, first + count}
}

// First returns the first element of the range.
func (r Range[T]) First() T {
	return r.first
}

// Last returns the last element of the range. This panics for empty ranges.
func (r Range[T]) Last() T {
	if r.first == r.afterLast {
		panic("last item of zero length range is not allowed")
	}
	return r.afterLast - 1
}

// AfterLast returns the first element after the range. This allows obtaining
// information about the end part of zero length ranges.
func (r Range[T]) AfterLast() T {
	return r.afterLast
}

// Count returns the number of elements in the range.
func (r Range[T]) Count() T {
	return r.afterLast - r.first
}

// IsEmpty returns true if the range is empty.
func (r Range[T]) IsEmpty() bool {
	return r.first == r.afterLast
}

// Includes returns true if the given element is inside the range.
func (r Range[T]) Includes(v T) bool {
	return v >= r.first && v < r.afterLast
}

// SetFirst updates the first element of the list.
func (r *Range[T]) SetFirst(v T) {
	r.first = v
	if r.afterLast < r.first {
		r.afterLast = r.first
	}
}

// SetAfterLast updates the end of the range by specifying the first element
// after the range. This allows setting zero length ranges.
func (r *Range[T]) SetAfterLast(v T) {
	r.afterLast = v
	if r.afterLast < r.first {
		r.first = r.afterLast
	}
}

// SetLast updates last element of the range.
func (r *Range[T]) SetLast(v T) {
	r.SetAfterLast(v + 1)
}

// Intersection returns the intersection of two ranges.
func (r Range[T]) Intersection(q Range[T]) Range[T] {
	i := Range[T]{first: max(r.first, q.first), afterLast: min(r.afterLast, q.afterLast)}
	if i.first > i.afterLast {
		return Range[T]{}
	}
	return i
}

// Union returns the union of two ranges. Panics for gapped ranges.
func (r Range[T]) Union(q Range[T]) Range[T] {
	if max(r.first, q.first) > min(r.afterLast, q.afterLast) {
		panic("cannot create union; gap between ranges")
	}
	return Range[T]{first: min(r.first, q.first), afterLast: max(r.afterLast, q.afterLast)}
}

// Iter iterates all integers in the range.
func (r Range[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := r.first; i < r.afterLast; i++ {
			if !yield(i) {
				break
			}
		}
	}
}
