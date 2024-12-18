package common

import (
	"container/heap"
)

// Heap is a generic min-heap (or max-heap, depending on Comparable behavior) implementation.
type Heap[T Comparable[T]] struct {
	heap innerHeap[T]
}

func NewHeap[T Comparable[T]]() *Heap[T] {
	return &Heap[T]{
		heap: make(innerHeap[T], 0),
	}
}

func (h *Heap[T]) Len() int {
	return len(h.heap)
}

func (h *Heap[T]) Push(element T) *HeapElement[T] {
	heapElement := NewHeapElement(element)
	heap.Push(&h.heap, heapElement)

	return heapElement
}

func (h *Heap[T]) Pop() *HeapElement[T] {
	return heap.Pop(&h.heap).(*HeapElement[T])
}

func (h *Heap[T]) Peek() *HeapElement[T] {
	if h.Len() == 0 {
		return nil
	}

	return h.heap[0]
}

func (h *Heap[T]) Remove(element *HeapElement[T]) {
	heap.Remove(&h.heap, element.index)
}

func (h *Heap[T]) Clear() {
	h.heap = make(innerHeap[T], 0)
}

type innerHeap[T Comparable[T]] []*HeapElement[T]

func (h innerHeap[T]) Len() int {
	return len(h)
}

func (h innerHeap[T]) Less(i, j int) bool {
	return h[i].Value().CompareTo(h[j].Value()) < 0
}

func (h innerHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *innerHeap[T]) Push(x interface{}) {
	data := x.(*HeapElement[T])
	*h = append(*h, data)
	data.index = len(*h) - 1
}

func (h *innerHeap[T]) Pop() interface{} {
	n := len(*h)
	element := (*h)[n-1]
	(*h)[n-1] = nil // avoid memory leak
	*h = (*h)[:n-1]
	element.index = -1

	return element
}

// Comparable is an interface for types that can be compared.
type Comparable[T any] interface {
	// CompareTo compares x with other.
	// To create a min heap, return:
	//	-1 if x <  other
	//	 0 if x == other
	//	+1 if x >  other
	// To create a max heap, return the opposite.
	CompareTo(other T) int
}

// HeapElement is a wrapper around the value stored in the heap.
type HeapElement[T Comparable[T]] struct {
	value T
	index int
}

func NewHeapElement[T Comparable[T]](value T) *HeapElement[T] {
	return &HeapElement[T]{
		value: value,
	}
}

func (h *HeapElement[T]) Value() T {
	return h.value
}

func (h *HeapElement[T]) Index() int {
	return h.index
}
