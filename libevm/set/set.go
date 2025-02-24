// Copyright 2025 the libevm authors.
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

// Package set provides a generic implementation of a set.
package set

// A Set is a generic set implementation.
type Set[T comparable] map[T]struct{}

// From returns a Set containing the elements.
func From[T comparable](elements ...T) Set[T] {
	s := make(Set[T], len(elements))
	for _, e := range elements {
		s[e] = struct{}{}
	}
	return s
}

// Sub returns the elements in `s` that aren't in `t`.
func (s Set[T]) Sub(t Set[T]) Set[T] {
	return s.alsoIn(t, false)
}

// Intersect returns the intersection of `s` and `t`.
func (s Set[T]) Intersect(t Set[T]) Set[T] {
	return s.alsoIn(t, true)
}

func (s Set[T]) alsoIn(t Set[T], inBoth bool) Set[T] {
	res := make(Set[T])
	for el := range s {
		if _, ok := t[el]; ok == inBoth {
			res[el] = struct{}{}
		}
	}
	return res
}

// Slice returns the elements of `s` as a slice.
func (s Set[T]) Slice() []T {
	sl := make([]T, 0, len(s))
	for el := range s {
		sl = append(sl, el)
	}
	return sl
}
