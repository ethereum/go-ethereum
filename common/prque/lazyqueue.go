// CookieJar - A contestant's algorithm toolbox
// Copyright (c) 2013 Peter Szilagyi. All rights reserved.
//
// CookieJar is dual licensed: use of this source code is governed by a BSD
// license that can be found in the LICENSE file. Alternatively, the CookieJar
// toolbox may be used in accordance with the terms and conditions contained
// in a signed written agreement between you and the author(s).

// This is a duplicated and slightly modified version of "gopkg.in/karalabe/cookiejar.v2/collections/LazyQueue".

// Package LazyQueue implements a priority queue data structure supporting arbitrary
// value types and int64 priorities.
//
// If you would like to use a min-priority queue, simply negate the priorities.
//
// Internally the queue is based on the standard heap package working on a
// sortable version of the block based stack.
package prque

import (
	"container/heap"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

type (
	PriorityCallback    func(data interface{}, now mclock.AbsTime) int64
	MaxPriorityCallback func(data interface{}, until mclock.AbsTime) int64
)

// Priority queue data structure.
type LazyQueue struct {
	clock       mclock.Clock
	queue       [2]*sstack
	popQueue    *sstack
	period      time.Duration
	maxUntil    mclock.AbsTime
	indexOffset int
	setIndex    SetIndexCallback
	priority    PriorityCallback
	maxPriority MaxPriorityCallback
}

// New creates a new priority queue.
func NewLazyQueue(setIndex SetIndexCallback, priority PriorityCallback, maxPriority MaxPriorityCallback, clock mclock.Clock, period time.Duration) *LazyQueue {
	q := &LazyQueue{
		popQueue:    newSstack(nil),
		setIndex:    setIndex,
		priority:    priority,
		maxPriority: maxPriority,
		clock:       clock,
		period:      period}
	q.Reset()
	q.Refresh()
	return q
}

func (q *LazyQueue) Refresh() {
	q.maxUntil = q.clock.Now() + mclock.AbsTime(q.period)
	for q.queue[0].Len() != 0 {
		q.Push(heap.Pop(q.queue[0]).(*item).value)
	}
	q.queue[0], q.queue[1] = q.queue[1], q.queue[0]
	q.indexOffset = 1 - q.indexOffset
	q.maxUntil += mclock.AbsTime(q.period)
}

func (q *LazyQueue) setIndex0(data interface{}, index int) {
	q.setIndex(data, index+index)
}

func (q *LazyQueue) setIndex1(data interface{}, index int) {
	q.setIndex(data, index+index+1)
}

// Pushes a value with a given priority into the queue, expanding if necessary.
func (q *LazyQueue) Push(data interface{}) {
	heap.Push(q.queue[1], &item{data, q.maxPriority(data, q.maxUntil)})
}

func (q *LazyQueue) Update(i int) {
	q.Push(q.Remove(i))
}

// Pops the value with the greates priority off the stack and returns it.
// Currently no shrinking is done.
func (q *LazyQueue) Pop() (interface{}, int64) {
	var (
		resData interface{}
		resPri  int64
	)
	q.MultiPop(func(data interface{}, priority int64) bool {
		resData = data
		resPri = priority
		return false
	})
	return resData, resPri
}

func (q *LazyQueue) peekIndex() int {
	if q.queue[0].Len() != 0 {
		if q.queue[1].Len() != 0 && q.queue[1].blocks[0][0].priority > q.queue[0].blocks[0][0].priority {
			return 1
		}
		return 0
	}
	if q.queue[1].Len() != 0 {
		return 1
	}
	return -1
}

func (q *LazyQueue) MultiPop(callback func(data interface{}, priority int64) bool) {
	now := q.clock.Now()
	nextIndex := q.peekIndex()
	for nextIndex != -1 {
		data := heap.Pop(q.queue[nextIndex]).(*item).value
		q.popQueue.Push(&item{data, q.priority(data, now)})
		nextIndex = q.peekIndex()
		for q.popQueue.Len() != 0 && (nextIndex == -1 || q.queue[nextIndex].blocks[0][0].priority < q.popQueue.blocks[0][0].priority) {
			i := heap.Pop(q.popQueue).(*item)
			if !callback(i.value, i.priority) {
				for q.popQueue.Len() != 0 {
					q.Push(heap.Pop(q.popQueue).(*item).value)
				}
				return
			}
		}
	}
}

// Pops only the item from the queue, dropping the associated priority value.
func (q *LazyQueue) PopItem() interface{} {
	i, _ := q.Pop()
	return i.(*item).value
}

// Remove removes the element with the given index.
func (q *LazyQueue) Remove(i int) interface{} {
	if i < 0 {
		return nil
	}
	return heap.Remove(q.queue[i&1^q.indexOffset], i>>1)
}

// Checks whether the priority queue is empty.
func (q *LazyQueue) Empty() bool {
	return q.queue[0].Len() == 0 && q.queue[1].Len() == 0
}

// Returns the number of element in the priority queue.
func (q *LazyQueue) Size() int {
	return q.queue[0].Len() + q.queue[1].Len()
}

// Clears the contents of the priority queue.
func (q *LazyQueue) Reset() {
	q.queue[0] = newSstack(q.setIndex0)
	q.queue[1] = newSstack(q.setIndex1)
}
