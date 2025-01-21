package common

type HeapMap[K comparable, T Comparable[T]] struct {
	h              *Heap[T]
	m              *ShrinkingMap[K, *HeapElement[T]]
	keyFromElement func(T) K
}

func NewHeapMap[K comparable, T Comparable[T]](keyFromElement func(T) K) *HeapMap[K, T] {
	return &HeapMap[K, T]{
		h:              NewHeap[T](),
		m:              NewShrinkingMap[K, *HeapElement[T]](1000),
		keyFromElement: keyFromElement,
	}
}

func (hm *HeapMap[K, T]) Len() int {
	return hm.h.Len()
}

func (hm *HeapMap[K, T]) Push(element T) bool {
	k := hm.keyFromElement(element)

	if hm.m.Has(k) {
		return false
	}

	heapElement := hm.h.Push(element)
	hm.m.Set(k, heapElement)

	return true
}

func (hm *HeapMap[K, T]) Pop() T {
	element := hm.h.Pop()
	k := hm.keyFromElement(element.Value())
	hm.m.Delete(k)

	return element.Value()
}

func (hm *HeapMap[K, T]) Peek() T {
	return hm.h.Peek().Value()
}

func (hm *HeapMap[K, T]) RemoveByElement(element T) bool {
	key := hm.keyFromElement(element)
	heapElement, exists := hm.m.Get(key)
	if !exists {
		return false
	}

	hm.h.Remove(heapElement)
	hm.m.Delete(key)

	return true
}

func (hm *HeapMap[K, T]) RemoveByKey(key K) bool {
	heapElement, exists := hm.m.Get(key)
	if !exists {
		return false
	}

	hm.h.Remove(heapElement)
	hm.m.Delete(key)

	return true
}

func (hm *HeapMap[K, T]) Clear() {
	hm.h.Clear()
	hm.m = NewShrinkingMap[K, *HeapElement[T]](1000)
}

func (hm *HeapMap[K, T]) Keys() []K {
	return hm.m.Keys()
}

func (hm *HeapMap[K, T]) Elements() []T {
	var elements []T
	for _, element := range hm.m.Values() {
		elements = append(elements, element.Value())
	}
	return elements
}

func (hm *HeapMap[K, T]) Has(element T) bool {
	return hm.m.Has(hm.keyFromElement(element))
}
