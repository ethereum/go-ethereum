// CookieJar - A contestant's algorithm toolbox
// Copyright (c) 2013 Peter Szilagyi. All rights reserved.
//
// CookieJar is dual licensed: use of this source code is governed by a BSD
// license that can be found in the LICENSE file. Alternatively, the CookieJar
// toolbox may be used in accordance with the terms and conditions contained
// in a signed written agreement between you and the author(s).

// This is a duplicated and slightly modified version of "gopkg.in/karalabe/cookiejar.v2/collections/prque".

// Package prque implements a priority queue data structure supporting arbitrary
// value types and int64 priorities.
//
// If you would like to use a min-priority queue, simply negate the priorities.
//
// Internally the queue is based on the standard heap package working on a
// sortable version of the block based stack.
package prque

import (
	"container/heap"
)

// Priority queue data structure.
type Prque[V any] struct {
	cont *sstack[V]
}

// New creates a new priority queue.
func New[V any](setIndex SetIndexCallback[V]) *Prque[V] {
	return &Prque[V]{newSstack(setIndex, false)}
}

// NewWrapAround creates a new priority queue with wrap-around priority handling.
func NewWrapAround[V any](setIndex SetIndexCallback[V]) *Prque[V] {
	return &Prque[V]{newSstack(setIndex, true)}
}

// Pushes a value with a given priority into the queue, expanding if necessary.
func (p *Prque[V]) Push(data V, priority int64) {
	heap.Push(p.cont, &item[V]{data, priority})
}

// Peek returns the value with the greatest priority but does not pop it off.
func (p *Prque[V]) Peek() (V, int64) {
	item := p.cont.blocks[0][0]
	return item.value, item.priority
}

// Pops the value with the greatest priority off the stack and returns it.
// Currently no shrinking is done.
func (p *Prque[V]) Pop() (V, int64) {
	item := heap.Pop(p.cont).(*item[V])
	return item.value, item.priority
}

// Pops only the item from the queue, dropping the associated priority value.
func (p *Prque[V]) PopItem() V {
	return heap.Pop(p.cont).(*item[V]).value
}

// Remove removes the element with the given index.
func (p *Prque[V]) Remove(i int) V {
	return heap.Remove(p.cont, i).(*item[V]).value
}

// Checks whether the priority queue is empty.
func (p *Prque[V]) Empty() bool {
	return p.cont.Len() == 0
}

// Returns the number of element in the priority queue.
func (p *Prque[V]) Size() int {
	return p.cont.Len()
}

// Clears the contents of the priority queue.
func (p *Prque[V]) Reset() {
	*p = *New(p.cont.setIndex)
}
