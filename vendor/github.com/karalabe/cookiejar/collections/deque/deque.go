// CookieJar - A contestant's algorithm toolbox
// Copyright (c) 2013 Peter Szilagyi. All rights reserved.
//
// CookieJar is dual licensed: use of this source code is governed by a BSD
// license that can be found in the LICENSE file. Alternatively, the CookieJar
// toolbox may be used in accordance with the terms and conditions contained
// in a signed written agreement between you and the author(s).

// Package deque implements a double ended queue supporting arbitrary types
// (even a mixture).
//
// Internally it uses a dynamically growing circular slice of blocks, resulting
// in faster resizes than a simple dynamic array/slice would allow, yet less gc
// overhead.
package deque

// The size of a block of data
const blockSize = 4096

// Double ended queue data structure.
type Deque struct {
	leftIdx  int
	leftOff  int
	rightIdx int
	rightOff int

	blocks [][]interface{}
	left   []interface{}
	right  []interface{}
}

// Creates a new, empty deque.
func New() *Deque {
	result := new(Deque)
	result.blocks = [][]interface{}{make([]interface{}, blockSize)}
	result.right = result.blocks[0]
	result.left = result.blocks[0]
	return result
}

// Pushes a new element into the queue from the right, expanding it if necessary.
func (d *Deque) PushRight(data interface{}) {
	d.right[d.rightOff] = data
	d.rightOff++
	if d.rightOff == blockSize {
		d.rightOff = 0
		d.rightIdx = (d.rightIdx + 1) % len(d.blocks)

		// If we wrapped over to the left, insert a new block and update indices
		if d.rightIdx == d.leftIdx {
			buffer := make([][]interface{}, len(d.blocks)+1)
			copy(buffer[:d.rightIdx], d.blocks[:d.rightIdx])
			buffer[d.rightIdx] = make([]interface{}, blockSize)
			copy(buffer[d.rightIdx+1:], d.blocks[d.rightIdx:])
			d.blocks = buffer
			d.leftIdx++
			d.left = d.blocks[d.leftIdx]
		}
		d.right = d.blocks[d.rightIdx]
	}
}

// Pops out an element from the queue from the right. Note, no bounds checking are done.
func (d *Deque) PopRight() (res interface{}) {
	d.rightOff--
	if d.rightOff < 0 {
		d.rightOff = blockSize - 1
		d.rightIdx = (d.rightIdx - 1 + len(d.blocks)) % len(d.blocks)
		d.right = d.blocks[d.rightIdx]
	}
	res, d.right[d.rightOff] = d.right[d.rightOff], nil
	return
}

// Returns the rightmost element from the deque. No bounds are checked.
func (d *Deque) Right() interface{} {
	if d.rightOff > 0 {
		return d.right[d.rightOff-1]
	} else {
		return d.blocks[(d.rightIdx-1+len(d.blocks))%len(d.blocks)][blockSize-1]
	}
}

// Pushes a new element into the queue from the left, expanding it if necessary.
func (d *Deque) PushLeft(data interface{}) {
	d.leftOff--
	if d.leftOff < 0 {
		d.leftOff = blockSize - 1
		d.leftIdx = (d.leftIdx - 1 + len(d.blocks)) % len(d.blocks)

		// If we wrapped over to the right, insert a new block and update indices
		if d.leftIdx == d.rightIdx {
			d.leftIdx++
			buffer := make([][]interface{}, len(d.blocks)+1)
			copy(buffer[:d.leftIdx], d.blocks[:d.leftIdx])
			buffer[d.leftIdx] = make([]interface{}, blockSize)
			copy(buffer[d.leftIdx+1:], d.blocks[d.leftIdx:])
			d.blocks = buffer
		}
		d.left = d.blocks[d.leftIdx]
	}
	d.left[d.leftOff] = data
}

// Pops out an element from the queue from the left. Note, no bounds checking are done.
func (d *Deque) PopLeft() (res interface{}) {
	res, d.left[d.leftOff] = d.left[d.leftOff], nil
	d.leftOff++
	if d.leftOff == blockSize {
		d.leftOff = 0
		d.leftIdx = (d.leftIdx + 1) % len(d.blocks)
		d.left = d.blocks[d.leftIdx]
	}
	return
}

// Returns the leftmost element from the deque. No bounds are checked.
func (d *Deque) Left() interface{} {
	return d.left[d.leftOff]
}

// Checks whether the queue is empty.
func (d *Deque) Empty() bool {
	return d.leftIdx == d.rightIdx && d.leftOff == d.rightOff
}

// Returns the number of elements in the queue.
func (d *Deque) Size() int {
	if d.rightIdx > d.leftIdx {
		return (d.rightIdx-d.leftIdx)*blockSize - d.leftOff + d.rightOff
	} else if d.rightIdx < d.leftIdx {
		return (len(d.blocks)-d.leftIdx+d.rightIdx)*blockSize - d.leftOff + d.rightOff
	} else {
		return d.rightOff - d.leftOff
	}
}

// Clears out the contents of the queue.
func (d *Deque) Reset() {
	*d = *New()
}
