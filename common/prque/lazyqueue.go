// Copyright 2019 The go-ethereum Authors
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

package prque

import (
	"container/heap"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

// LazyQueue is a priority queue data structure where priorities can change over
// time and are only evaluated on demand.
// Two callbacks are required:
//   - priority evaluates the actual priority of an item
//   - maxPriority gives an upper estimate for the priority in any moment between
//     now and the given absolute time
//
// If the upper estimate is exceeded then Update should be called for that item.
// A global Refresh function should also be called periodically.
type LazyQueue struct {
	clock mclock.Clock
	// Items are stored in one of two internal queues ordered by estimated max
	// priority until the next and the next-after-next refresh. Update and Refresh
	// always places items in queue[1].
	queue                      [2]*sstack
	popQueue                   *sstack
	period                     time.Duration
	maxUntil                   mclock.AbsTime
	indexOffset                int
	setIndex                   SetIndexCallback
	priority                   PriorityCallback
	maxPriority                MaxPriorityCallback
	lastRefresh1, lastRefresh2 mclock.AbsTime
}

type (
	PriorityCallback    func(data interface{}) int64                       // actual priority callback
	MaxPriorityCallback func(data interface{}, until mclock.AbsTime) int64 // estimated maximum priority callback
)

// NewLazyQueue creates a new lazy queue
func NewLazyQueue(setIndex SetIndexCallback, priority PriorityCallback, maxPriority MaxPriorityCallback, clock mclock.Clock, refreshPeriod time.Duration) *LazyQueue {
	q := &LazyQueue{
		popQueue:     newSstack(nil, false),
		setIndex:     setIndex,
		priority:     priority,
		maxPriority:  maxPriority,
		clock:        clock,
		period:       refreshPeriod,
		lastRefresh1: clock.Now(),
		lastRefresh2: clock.Now(),
	}
	q.Reset()
	q.refresh(clock.Now())
	return q
}

// Reset clears the contents of the queue
func (q *LazyQueue) Reset() {
	q.queue[0] = newSstack(q.setIndex0, false)
	q.queue[1] = newSstack(q.setIndex1, false)
}

// Refresh performs queue re-evaluation if necessary
func (q *LazyQueue) Refresh() {
	now := q.clock.Now()
	for time.Duration(now-q.lastRefresh2) >= q.period*2 {
		q.refresh(now)
		q.lastRefresh2 = q.lastRefresh1
		q.lastRefresh1 = now
	}
}

// refresh re-evaluates items in the older queue and swaps the two queues
func (q *LazyQueue) refresh(now mclock.AbsTime) {
	q.maxUntil = now.Add(q.period)
	for q.queue[0].Len() != 0 {
		q.Push(heap.Pop(q.queue[0]).(*item).value)
	}
	q.queue[0], q.queue[1] = q.queue[1], q.queue[0]
	q.indexOffset = 1 - q.indexOffset
	q.maxUntil = q.maxUntil.Add(q.period)
}

// Push adds an item to the queue
func (q *LazyQueue) Push(data interface{}) {
	heap.Push(q.queue[1], &item{data, q.maxPriority(data, q.maxUntil)})
}

// Update updates the upper priority estimate for the item with the given queue index
func (q *LazyQueue) Update(index int) {
	q.Push(q.Remove(index))
}

// Pop removes and returns the item with the greatest actual priority
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

// peekIndex returns the index of the internal queue where the item with the
// highest estimated priority is or -1 if both are empty
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

// MultiPop pops multiple items from the queue and is more efficient than calling
// Pop multiple times. Popped items are passed to the callback. MultiPop returns
// when the callback returns false or there are no more items to pop.
func (q *LazyQueue) MultiPop(callback func(data interface{}, priority int64) bool) {
	nextIndex := q.peekIndex()
	for nextIndex != -1 {
		data := heap.Pop(q.queue[nextIndex]).(*item).value
		heap.Push(q.popQueue, &item{data, q.priority(data)})
		nextIndex = q.peekIndex()
		for q.popQueue.Len() != 0 && (nextIndex == -1 || q.queue[nextIndex].blocks[0][0].priority < q.popQueue.blocks[0][0].priority) {
			i := heap.Pop(q.popQueue).(*item)
			if !callback(i.value, i.priority) {
				for q.popQueue.Len() != 0 {
					q.Push(heap.Pop(q.popQueue).(*item).value)
				}
				return
			}
			nextIndex = q.peekIndex() // re-check because callback is allowed to push items back
		}
	}
}

// PopItem pops the item from the queue only, dropping the associated priority value.
func (q *LazyQueue) PopItem() interface{} {
	i, _ := q.Pop()
	return i
}

// Remove removes the item with the given index.
func (q *LazyQueue) Remove(index int) interface{} {
	if index < 0 {
		return nil
	}
	return heap.Remove(q.queue[index&1^q.indexOffset], index>>1).(*item).value
}

// Empty checks whether the priority queue is empty.
func (q *LazyQueue) Empty() bool {
	return q.queue[0].Len() == 0 && q.queue[1].Len() == 0
}

// Size returns the number of items in the priority queue.
func (q *LazyQueue) Size() int {
	return q.queue[0].Len() + q.queue[1].Len()
}

// setIndex0 translates internal queue item index to the virtual index space of LazyQueue
func (q *LazyQueue) setIndex0(data interface{}, index int) {
	if index == -1 {
		q.setIndex(data, -1)
	} else {
		q.setIndex(data, index+index)
	}
}

// setIndex1 translates internal queue item index to the virtual index space of LazyQueue
func (q *LazyQueue) setIndex1(data interface{}, index int) {
	q.setIndex(data, index+index+1)
}
