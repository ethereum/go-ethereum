package boom

import (
	"bytes"
	"container/heap"
)

type Element struct {
	Data []byte
	Freq uint64
}

// An elementHeap is a min-heap of elements.
type elementHeap []*Element

func (e elementHeap) Len() int           { return len(e) }
func (e elementHeap) Less(i, j int) bool { return e[i].Freq < e[j].Freq }
func (e elementHeap) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

func (e *elementHeap) Push(x interface{}) {
	*e = append(*e, x.(*Element))
}

func (e *elementHeap) Pop() interface{} {
	old := *e
	n := len(old)
	x := old[n-1]
	*e = old[0 : n-1]
	return x
}

// TopK uses a Count-Min Sketch to calculate the top-K frequent elements in a
// stream.
type TopK struct {
	cms      *CountMinSketch
	k        uint
	n        uint
	elements *elementHeap
}

// NewTopK creates a new TopK backed by a Count-Min sketch whose relative
// accuracy is within a factor of epsilon with probability delta. It tracks the
// k-most frequent elements.
func NewTopK(epsilon, delta float64, k uint) *TopK {
	elements := make(elementHeap, 0, k)
	heap.Init(&elements)
	return &TopK{
		cms:      NewCountMinSketch(epsilon, delta),
		k:        k,
		elements: &elements,
	}
}

// Add will add the data to the Count-Min Sketch and update the top-k heap if
// applicable. Returns the TopK to allow for chaining.
func (t *TopK) Add(data []byte) *TopK {
	t.cms.Add(data)
	t.n++

	freq := t.cms.Count(data)
	if t.isTop(freq) {
		t.insert(data, freq)
	}

	return t
}

// Elements returns the top-k elements from lowest to highest frequency.
func (t *TopK) Elements() []*Element {
	if t.elements.Len() == 0 {
		return make([]*Element, 0)
	}

	elements := make(elementHeap, t.elements.Len())
	copy(elements, *t.elements)
	heap.Init(&elements)
	topK := make([]*Element, 0, t.k)

	for elements.Len() > 0 {
		topK = append(topK, heap.Pop(&elements).(*Element))
	}

	return topK
}

// Reset restores the TopK to its original state. It returns itself to allow
// for chaining.
func (t *TopK) Reset() *TopK {
	t.cms.Reset()
	elements := make(elementHeap, 0, t.k)
	heap.Init(&elements)
	t.elements = &elements
	t.n = 0
	return t
}

// isTop indicates if the given frequency falls within the top-k heap.
func (t *TopK) isTop(freq uint64) bool {
	if t.elements.Len() < int(t.k) {
		return true
	}

	return freq >= (*t.elements)[0].Freq
}

// insert adds the data to the top-k heap. If the data is already an element,
// the frequency is updated. If the heap already has k elements, the element
// with the minimum frequency is removed.
func (t *TopK) insert(data []byte, freq uint64) {
	for _, element := range *t.elements {
		if bytes.Compare(data, element.Data) == 0 {
			// Element already in top-k.
			element.Freq = freq
			return
		}
	}

	if t.elements.Len() == int(t.k) {
		// Remove minimum-frequency element.
		heap.Pop(t.elements)
	}

	// Add element to top-k.
	heap.Push(t.elements, &Element{Data: data, Freq: freq})
}
