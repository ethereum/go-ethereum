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
	"io"
	"iter"

	"github.com/ethereum/go-ethereum/rlp"
)

// Range represents a range of integers.
type Range[T uint32 | uint64] struct {
	first, afterLast T
}

func NewRange[T uint32 | uint64](first, count T) Range[T] {
	return Range[T]{first, first + count}
}

func (r *Range[T]) EncodeRLP(w io.Writer) error {
	if err := rlp.Encode(w, &r.first); err != nil {
		return err
	}
	return rlp.Encode(w, &r.afterLast)
}

func (r *Range[T]) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&r.first); err != nil {
		return err
	}
	return s.Decode(&r.afterLast)
}

func (r Range[T]) First() T {
	return r.first
}

func (r Range[T]) Last() T {
	if r.first == r.afterLast {
		panic("last item of zero length range is not allowed")
	}
	return r.afterLast - 1
}

func (r Range[T]) AfterLast() T {
	return r.afterLast
}

func (r Range[T]) Count() T {
	return r.afterLast - r.first
}

func (r Range[T]) IsEmpty() bool {
	return r.first == r.afterLast
}

func (r Range[T]) Includes(v T) bool {
	return v >= r.first && v < r.afterLast
}

func (r *Range[T]) SetFirst(v T) {
	r.first = v
	if r.afterLast < r.first {
		r.afterLast = r.first
	}
}

func (r *Range[T]) SetAfterLast(v T) {
	r.afterLast = v
	if r.afterLast < r.first {
		r.first = r.afterLast
	}
}

func (r *Range[T]) SetLast(v T) {
	r.SetAfterLast(v + 1)
}

func (r Range[T]) Intersection(q Range[T]) Range[T] {
	if r.first > q.first {
		q.first = r.first
	}
	if r.afterLast < q.afterLast {
		q.afterLast = r.afterLast
	}
	if q.first > q.afterLast {
		return Range[T]{}
	}
	return q
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
