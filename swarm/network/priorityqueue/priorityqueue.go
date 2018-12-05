// Copyright 2018 The go-ethereum Authors
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

// package priority_queue implement a channel based priority queue
// over arbitrary types. It provides an
// an autopop loop applying a function to the items always respecting
// their priority. The structure is only quasi consistent ie., if a lower
// priority item is autopopped, it is guaranteed that there was a point
// when no higher priority item was present, ie. it is not guaranteed
// that there was any point where the lower priority item was present
// but the higher was not

package priorityqueue

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrContention = errors.New("contention")

	errBadPriority = errors.New("bad priority")

	wakey = struct{}{}
)

// PriorityQueue is the basic structure
type PriorityQueue struct {
	Queues []chan interface{}
	wakeup chan struct{}
}

// New is the constructor for PriorityQueue
func New(n int, l int) *PriorityQueue {
	var queues = make([]chan interface{}, n)
	for i := range queues {
		queues[i] = make(chan interface{}, l)
	}
	return &PriorityQueue{
		Queues: queues,
		wakeup: make(chan struct{}, 1),
	}
}

// Run is a forever loop popping items from the queues
func (pq *PriorityQueue) Run(ctx context.Context, f func(interface{})) {
	top := len(pq.Queues) - 1
	p := top
READ:
	for {
		q := pq.Queues[p]
		select {
		case <-ctx.Done():
			return
		case x := <-q:
			log.Trace("priority.queue f(x)", "p", p, "len(Queues[p])", len(pq.Queues[p]))
			f(x)
			p = top
		default:
			if p > 0 {
				p--
				log.Trace("priority.queue p > 0", "p", p)
				continue READ
			}
			p = top
			select {
			case <-ctx.Done():
				return
			case <-pq.wakeup:
				log.Trace("priority.queue wakeup", "p", p)
			}
		}
	}
}

// Push pushes an item to the appropriate queue specified in the priority argument
// if context is given it waits until either the item is pushed or the Context aborts
func (pq *PriorityQueue) Push(x interface{}, p int) error {
	if p < 0 || p >= len(pq.Queues) {
		return errBadPriority
	}
	log.Trace("priority.queue push", "p", p, "len(Queues[p])", len(pq.Queues[p]))
	select {
	case pq.Queues[p] <- x:
	default:
		return ErrContention
	}
	select {
	case pq.wakeup <- wakey:
	default:
	}
	return nil
}
