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

package trie

// bytesPool is a pool for byte slices. It is safe for concurrent use.
type bytesPool struct {
	c chan []byte
	w int
}

// newBytesPool creates a new bytesPool. The sliceCap sets the capacity of
// newly allocated slices, and the nitems determines how many items the pool
// will hold, at maximum.
func newBytesPool(sliceCap, nitems int) *bytesPool {
	return &bytesPool{
		c: make(chan []byte, nitems),
		w: sliceCap,
	}
}

// get returns a slice. Safe for concurrent use.
func (bp *bytesPool) get() []byte {
	select {
	case b := <-bp.c:
		return b
	default:
		return make([]byte, 0, bp.w)
	}
}

// getWithSize returns a slice with specified byte slice size.
func (bp *bytesPool) getWithSize(s int) []byte {
	b := bp.get()
	if cap(b) < s {
		return make([]byte, s)
	}
	return b[:s]
}

// put returns a slice to the pool. Safe for concurrent use. This method
// will ignore slices that are too small or too large (>3x the cap)
func (bp *bytesPool) put(b []byte) {
	if c := cap(b); c < bp.w || c > 3*bp.w {
		return
	}
	select {
	case bp.c <- b:
	default:
	}
}

// unsafeBytesPool is a pool for byte slices. It is not safe for concurrent use.
type unsafeBytesPool struct {
	items [][]byte
	w     int
}

// newUnsafeBytesPool creates a new unsafeBytesPool. The sliceCap sets the
// capacity of newly allocated slices, and the nitems determines how many
// items the pool will hold, at maximum.
func newUnsafeBytesPool(sliceCap, nitems int) *unsafeBytesPool {
	return &unsafeBytesPool{
		items: make([][]byte, 0, nitems),
		w:     sliceCap,
	}
}

// Get returns a slice with pre-allocated space.
func (bp *unsafeBytesPool) get() []byte {
	if len(bp.items) > 0 {
		last := bp.items[len(bp.items)-1]
		bp.items = bp.items[:len(bp.items)-1]
		return last
	}
	return make([]byte, 0, bp.w)
}

// put returns a slice to the pool. This method will ignore slices that are
// too small or too large (>3x the cap)
func (bp *unsafeBytesPool) put(b []byte) {
	if c := cap(b); c < bp.w || c > 3*bp.w {
		return
	}
	if len(bp.items) < cap(bp.items) {
		bp.items = append(bp.items, b)
	}
}
