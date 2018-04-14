// This is a duplicated and slightly modified version of "gopkg.in/karalabe/cookiejar.v2/collections/prque".

package prque

import (
	"container/heap"
)

// Priority queue data structure.
type Prque struct {
	cont *sstack
}

// Creates a new priority queue.
func New(compare compareFn) *Prque {
	return &Prque{newSstack(compare)}
}

// Pushes a value with a given priority into the queue, expanding if necessary.
func (p *Prque) Push(i interface{}) {
	heap.Push(p.cont, i)
}

// Pops the value with the greates priority off the stack and returns it.
// Currently no shrinking is done.
func (p *Prque) Pop() interface{} {
	return heap.Pop(p.cont)
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
	*p = *New(p.cont.compare)
}
