/*
Open Source Initiative OSI - The MIT License (MIT):Licensing

The MIT License (MIT)
Copyright (c) 2013 Ralph Caraveo (deckarep@gmail.com)

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package mapset

import "sync"

type threadSafeSet struct {
	s threadUnsafeSet
	sync.RWMutex
}

func newThreadSafeSet() threadSafeSet {
	return threadSafeSet{s: newThreadUnsafeSet()}
}

func (set *threadSafeSet) Add(i interface{}) bool {
	set.Lock()
	ret := set.s.Add(i)
	set.Unlock()
	return ret
}

func (set *threadSafeSet) Contains(i ...interface{}) bool {
	set.RLock()
	ret := set.s.Contains(i...)
	set.RUnlock()
	return ret
}

func (set *threadSafeSet) IsSubset(other Set) bool {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	ret := set.s.IsSubset(&o.s)
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) IsProperSubset(other Set) bool {
	o := other.(*threadSafeSet)

	set.RLock()
	defer set.RUnlock()
	o.RLock()
	defer o.RUnlock()

	return set.s.IsProperSubset(&o.s)
}

func (set *threadSafeSet) IsSuperset(other Set) bool {
	return other.IsSubset(set)
}

func (set *threadSafeSet) IsProperSuperset(other Set) bool {
	return other.IsProperSubset(set)
}

func (set *threadSafeSet) Union(other Set) Set {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	unsafeUnion := set.s.Union(&o.s).(*threadUnsafeSet)
	ret := &threadSafeSet{s: *unsafeUnion}
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) Intersect(other Set) Set {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	unsafeIntersection := set.s.Intersect(&o.s).(*threadUnsafeSet)
	ret := &threadSafeSet{s: *unsafeIntersection}
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) Difference(other Set) Set {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	unsafeDifference := set.s.Difference(&o.s).(*threadUnsafeSet)
	ret := &threadSafeSet{s: *unsafeDifference}
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) SymmetricDifference(other Set) Set {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	unsafeDifference := set.s.SymmetricDifference(&o.s).(*threadUnsafeSet)
	ret := &threadSafeSet{s: *unsafeDifference}
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) Clear() {
	set.Lock()
	set.s = newThreadUnsafeSet()
	set.Unlock()
}

func (set *threadSafeSet) Remove(i interface{}) {
	set.Lock()
	delete(set.s, i)
	set.Unlock()
}

func (set *threadSafeSet) Cardinality() int {
	set.RLock()
	defer set.RUnlock()
	return len(set.s)
}

func (set *threadSafeSet) Each(cb func(interface{}) bool) {
	set.RLock()
	for elem := range set.s {
		if cb(elem) {
			break
		}
	}
	set.RUnlock()
}

func (set *threadSafeSet) Iter() <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		set.RLock()

		for elem := range set.s {
			ch <- elem
		}
		close(ch)
		set.RUnlock()
	}()

	return ch
}

func (set *threadSafeSet) Iterator() *Iterator {
	iterator, ch, stopCh := newIterator()

	go func() {
		set.RLock()
	L:
		for elem := range set.s {
			select {
			case <-stopCh:
				break L
			case ch <- elem:
			}
		}
		close(ch)
		set.RUnlock()
	}()

	return iterator
}

func (set *threadSafeSet) Equal(other Set) bool {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	ret := set.s.Equal(&o.s)
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) Clone() Set {
	set.RLock()

	unsafeClone := set.s.Clone().(*threadUnsafeSet)
	ret := &threadSafeSet{s: *unsafeClone}
	set.RUnlock()
	return ret
}

func (set *threadSafeSet) String() string {
	set.RLock()
	ret := set.s.String()
	set.RUnlock()
	return ret
}

func (set *threadSafeSet) PowerSet() Set {
	set.RLock()
	ret := set.s.PowerSet()
	set.RUnlock()
	return ret
}

func (set *threadSafeSet) Pop() interface{} {
	set.Lock()
	defer set.Unlock()
	return set.s.Pop()
}

func (set *threadSafeSet) CartesianProduct(other Set) Set {
	o := other.(*threadSafeSet)

	set.RLock()
	o.RLock()

	unsafeCartProduct := set.s.CartesianProduct(&o.s).(*threadUnsafeSet)
	ret := &threadSafeSet{s: *unsafeCartProduct}
	set.RUnlock()
	o.RUnlock()
	return ret
}

func (set *threadSafeSet) ToSlice() []interface{} {
	keys := make([]interface{}, 0, set.Cardinality())
	set.RLock()
	for elem := range set.s {
		keys = append(keys, elem)
	}
	set.RUnlock()
	return keys
}

func (set *threadSafeSet) MarshalJSON() ([]byte, error) {
	set.RLock()
	b, err := set.s.MarshalJSON()
	set.RUnlock()

	return b, err
}

func (set *threadSafeSet) UnmarshalJSON(p []byte) error {
	set.RLock()
	err := set.s.UnmarshalJSON(p)
	set.RUnlock()

	return err
}
