package ethpepple

import (
	"bytes"
)

const maxItem = 250_000 // every item has 40 bytes, so the heap most have 10MB

type Item struct {
	Distance  []byte
	ValueSize uint64
}

type MaxHeap []Item

func (m MaxHeap) Len() int {
	return len(m)
}

func (m MaxHeap) Less(i, j int) bool {
	// Compare Distance as byte slices
	return bytes.Compare(m[i].Distance, m[j].Distance) > 0
}

func (m MaxHeap) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m *MaxHeap) Pop() interface{} {
	old := *m
	n := len(old)
	item := old[n-1]
	*m = old[0 : n-1]
	return item
}

func (m *MaxHeap) Push(x interface{}) {
	*m = append(*m, x.(Item))
}
