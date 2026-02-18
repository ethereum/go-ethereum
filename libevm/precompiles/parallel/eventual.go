// Copyright 2025-2026 the libevm authors.
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

package parallel

// An eventual type holds a value that is set at some unknown point in the
// future and used, possibly concurrently, by one or more peekers or a single
// taker (together, "getters"). The zero value is NOT ready for use.
type eventual[T any] struct {
	ch chan T
}

// eventually returns a new eventual value.
func eventually[T any]() eventual[T] {
	return eventual[T]{
		ch: make(chan T, 1),
	}
}

// put sets the value, unblocking any current and future getters. put itself is
// non-blocking, however it is NOT possible to overwrite the value without an
// intervening call to [eventual.take].
func (e eventual[T]) put(v T) {
	e.ch <- v
}

// peek returns the value after making it available for other getters. Although
// the act of peeking is threadsafe, the returned value might not be.
func (e eventual[T]) peek() T {
	v := <-e.ch
	e.ch <- v
	return v
}

// take returns the value and resets e to its default state as if immediately
// after construction.
func (e eventual[T]) take() T {
	return <-e.ch
}
