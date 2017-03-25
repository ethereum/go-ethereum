// CookieJar - A contestant's algorithm toolbox
// Copyright (c) 2013 Peter Szilagyi. All rights reserved.
//
// CookieJar is dual licensed: use of this source code is governed by a BSD
// license that can be found in the LICENSE file. Alternatively, the CookieJar
// toolbox may be used in accordance with the terms and conditions contained
// in a signed written agreement between you and the author(s).

// Package prque implements a priority queue data structure supporting arbitrary
// value types and float priorities.
//
// The reasoning behind using floats for the priorities vs. ints or interfaces
// was larger flexibility without sacrificing too much performance or code
// complexity.
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
type Prque struct {
	cont *sstack
}

// Creates a new priority queue.
func New() *Prque {
	return &Prque{newSstack()}
}

// Pushes a value with a given priority into the queue, expanding if necessary.
func (p *Prque) Push(data interface{}, priority float32) {
	heap.Push(p.cont, &item{data, priority})
}

// Pops the value with the greates priority off the stack and returns it.
// Currently no shrinking is done.
func (p *Prque) Pop() (interface{}, float32) {
	item := heap.Pop(p.cont).(*item)
	return item.value, item.priority
}

// Pops only the item from the queue, dropping the associated priority value.
func (p *Prque) PopItem() interface{} {
	return heap.Pop(p.cont).(*item).value
}

// Checks whether the priority queue is empty.
func (p *Prque) Empty() bool {
	return p.cont.Len() == 0
}

// Returns the number of element in the priority queue.
func (p *Prque) Size() int {
	return p.cont.Len()
}

// Clears the contents of the priority queue.
func (p *Prque) Reset() {
	*p = *New()
}
