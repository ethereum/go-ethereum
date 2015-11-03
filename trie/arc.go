// Copyright (c) 2015 Hans Alexander Gugel <alexander.gugel@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// This file contains a modified version of package arc from
// https://github.com/alexanderGugel/arc
//
// It implements the ARC (Adaptive Replacement Cache) algorithm as detailed in
// https://www.usenix.org/legacy/event/fast03/tech/full_papers/megiddo/megiddo.pdf

package trie

import (
	"container/list"
	"sync"
)

type arc struct {
	p     int
	c     int
	t1    *list.List
	b1    *list.List
	t2    *list.List
	b2    *list.List
	cache map[string]*entry
	mutex sync.Mutex
}

type entry struct {
	key   hashNode
	value node
	ll    *list.List
	el    *list.Element
}

// newARC returns a new Adaptive Replacement Cache with the
// given capacity.
func newARC(c int) *arc {
	return &arc{
		c:     c,
		t1:    list.New(),
		b1:    list.New(),
		t2:    list.New(),
		b2:    list.New(),
		cache: make(map[string]*entry, c),
	}
}

// Put inserts a new key-value pair into the cache.
// This optimizes future access to this entry (side effect).
func (a *arc) Put(key hashNode, value node) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	ent, ok := a.cache[string(key)]
	if ok != true {
		ent = &entry{key: key, value: value}
		a.req(ent)
		a.cache[string(key)] = ent
	} else {
		ent.value = value
		a.req(ent)
	}
	return ok
}

// Get retrieves a previously via Set inserted entry.
// This optimizes future access to this entry (side effect).
func (a *arc) Get(key hashNode) (value node, ok bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	ent, ok := a.cache[string(key)]
	if ok {
		a.req(ent)
		return ent.value, ent.value != nil
	}
	return nil, false
}

func (a *arc) req(ent *entry) {
	if ent.ll == a.t1 || ent.ll == a.t2 {
		// Case I
		ent.setMRU(a.t2)
	} else if ent.ll == a.b1 {
		// Case II
		// Cache Miss in t1 and t2

		// Adaptation
		var d int
		if a.b1.Len() >= a.b2.Len() {
			d = 1
		} else {
			d = a.b2.Len() / a.b1.Len()
		}
		a.p = a.p + d
		if a.p > a.c {
			a.p = a.c
		}

		a.replace(ent)
		ent.setMRU(a.t2)
	} else if ent.ll == a.b2 {
		// Case III
		// Cache Miss in t1 and t2

		// Adaptation
		var d int
		if a.b2.Len() >= a.b1.Len() {
			d = 1
		} else {
			d = a.b1.Len() / a.b2.Len()
		}
		a.p = a.p - d
		if a.p < 0 {
			a.p = 0
		}

		a.replace(ent)
		ent.setMRU(a.t2)
	} else if ent.ll == nil {
		// Case IV

		if a.t1.Len()+a.b1.Len() == a.c {
			// Case A
			if a.t1.Len() < a.c {
				a.delLRU(a.b1)
				a.replace(ent)
			} else {
				a.delLRU(a.t1)
			}
		} else if a.t1.Len()+a.b1.Len() < a.c {
			// Case B
			if a.t1.Len()+a.t2.Len()+a.b1.Len()+a.b2.Len() >= a.c {
				if a.t1.Len()+a.t2.Len()+a.b1.Len()+a.b2.Len() == 2*a.c {
					a.delLRU(a.b2)
				}
				a.replace(ent)
			}
		}

		ent.setMRU(a.t1)
	}
}

func (a *arc) delLRU(list *list.List) {
	lru := list.Back()
	list.Remove(lru)
	delete(a.cache, string(lru.Value.(*entry).key))
}

func (a *arc) replace(ent *entry) {
	if a.t1.Len() > 0 && ((a.t1.Len() > a.p) || (ent.ll == a.b2 && a.t1.Len() == a.p)) {
		lru := a.t1.Back().Value.(*entry)
		lru.value = nil
		lru.setMRU(a.b1)
	} else {
		lru := a.t2.Back().Value.(*entry)
		lru.value = nil
		lru.setMRU(a.b2)
	}
}

func (e *entry) setLRU(list *list.List) {
	e.detach()
	e.ll = list
	e.el = e.ll.PushBack(e)
}

func (e *entry) setMRU(list *list.List) {
	e.detach()
	e.ll = list
	e.el = e.ll.PushFront(e)
}

func (e *entry) detach() {
	if e.ll != nil {
		e.ll.Remove(e.el)
	}
}
