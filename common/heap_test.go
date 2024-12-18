package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Int int

func (i Int) CompareTo(other Int) int {
	if i < other {
		return -1
	} else if i > other {
		return 1
	} else {
		return 0
	}
}

func TestHeap(t *testing.T) {
	h := NewHeap[Int]()

	require.Equal(t, 0, h.Len(), "Heap should be empty initially")

	h.Push(Int(3))
	h.Push(Int(1))
	h.Push(Int(2))

	require.Equal(t, 3, h.Len(), "Heap should have three elements after pushing")

	require.EqualValues(t, 1, h.Pop(), "Pop should return the smallest element")
	require.Equal(t, 2, h.Len(), "Heap should have two elements after popping")

	require.EqualValues(t, 2, h.Pop(), "Pop should return the next smallest element")
	require.Equal(t, 1, h.Len(), "Heap should have one element after popping")

	require.EqualValues(t, 3, h.Pop(), "Pop should return the last element")
	require.Equal(t, 0, h.Len(), "Heap should be empty after popping all elements")
}
