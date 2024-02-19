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

package lightclient

type objectRequest[T any] map[chan T]struct{}

func newObjectRequest[T any]() objectRequest[T] {
	return make(objectRequest[T])
}

func (r objectRequest[T]) addRequest() chan T {
	ch := make(chan T, 1)
	r[ch] = struct{}{}
	return ch
}

func (r objectRequest[T]) cancelRequest(ch chan T) {
	if _, ok := r[ch]; !ok {
		return
	}
	delete(r, ch)
	close(ch)
}

func (r objectRequest[T]) isEmpty() bool {
	return len(r) == 0
}

func (r objectRequest[T]) deliver(value T) {
	for ch := range r {
		ch <- value
	}
}
