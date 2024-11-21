package ethpepple

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxHeap(t *testing.T) {
	expectValueSize := []uint64{40, 30, 20, 10}
	// Create a heap and initialize it with some items
	h := &MaxHeap{
		{Distance: []byte{1}, ValueSize: 10},
		{Distance: []byte{2}, ValueSize: 20},
		{Distance: []byte{3}, ValueSize: 30},
	}
	heap.Init(h)

	// Push a new item into the heap
	heap.Push(h, Item{Distance: []byte{4}, ValueSize: 40})
	heap.Push(h, Item{Distance: []byte{5}, ValueSize: 50})

	removed := heap.Remove(h, 0)
	assert.Equal(t, removed.(Item).ValueSize, uint64(50))

	len := h.Len()
	// Pop and print the largest element
	for i := 0; i < len; i++ {
		item := heap.Pop(h).(Item)
		assert.Equal(t, item.ValueSize, expectValueSize[i])
	}
}
