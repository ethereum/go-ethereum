// package priority_queues implement a channel based priority queue
// over arbitrary types. It provides an
// an autopop loop applying a function to the items always respecting
// their priority. The structure is only quasi consistent ie., if a lower
// priority item is autopopped, it is guaranteed that there was a point
// when no higher priority item was present, ie. it is not guaranteed
// that there was any point where the lower priority item was present
// but the higher was not

package priorityqueues

import (
	"context"
	"errors"
)

var (
	errContention  = errors.New("queue contention")
	errBadPriority = errors.New("bad priority")

	wakey = struct{}{}
)

// PriorityQueues is the basic structure
type PriorityQueues struct {
	queues []chan interface{}
	wakeup chan struct{}
}

// New is the constructor for PriorityQueues
func New(n int, l int) *PriorityQueues {
	var queues = make([]chan interface{}, n)
	for i := range queues {
		queues[i] = make(chan interface{}, l)
	}
	return &PriorityQueues{
		queues: queues,
		wakeup: make(chan struct{}, 1),
	}
}

// Run is a forever loop popping items from the queues
func (pq *PriorityQueues) Run(ctx context.Context, f func(interface{})) {
	top := len(pq.queues) - 1
	p := top
	q := pq.queues[p]
READ:
	for {
		select {
		case <-ctx.Done():
			return
		case x := <-q:
			f(x)
			p = top
		default:
			if p > 0 {
				p--
				continue READ
			}
			p = top
			select {
			case <-ctx.Done():
				return
			case <-pq.wakeup:
			}
		}
	}
}

// Push pushes an item to the appropriate queue specified in the priority argument
// if context is given it waits until either the item is pushed or the Context aborts
// otherwise returns errContention if the queue is full
func (pq *PriorityQueues) Push(ctx context.Context, x interface{}, p int) error {
	if p < 0 || p >= len(pq.queues) {
		return errBadPriority
	}
	if ctx == nil {
		select {
		case pq.queues[p] <- x:
		default:
			return errContention
		}
	} else {
		select {
		case pq.queues[p] <- x:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	select {
	case pq.wakeup <- wakey:
	default:
	}
	return nil
}
