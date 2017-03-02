// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
package pot

import (
	"fmt"
	"sync"
)

const (
	// keylen = 4
	keylen    = 256
	maxkeylen = 256
)

// Pot is the root node type, allows locked non-applicative manipulation
type Pot struct {
	lock sync.RWMutex
	*pot
}

// pot is the node type (same for root, branching node and leaf)
type pot struct {
	pin  PotVal
	bins []*pot
	size int
	po   int
}

// PotVal is the interface the generic container item should implement
type PotVal interface {
	PO(PotVal, int) (po int, eq bool)
	String() string
}

// Pot constructor. Requires  value of type PotVal to pin
// and po to point to a span in the PotVal key
// The pinned item counts towards the size
func NewPot(v PotVal, po int) *Pot {
	var size int
	if v != nil {
		size++
	}
	return &Pot{
		pot: &pot{
			pin:  v,
			po:   po,
			size: size,
		},
	}
}

// Pin() returns the pinned element (key) of the Pot
func (t *Pot) Pin() PotVal {
	return t.pin
}

// Size() returns the number of values in the Pot
func (t *Pot) Size() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.size
}

// Add(v) inserts v into the Pot and
// returns the proximity order of v and a boolean
// indicating if the item was found
// Add locks the Pot while using applicative add on its pot
func (t *Pot) Add(val PotVal) (po int, found bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.pot, po, found = add(t.pot, val)
	return po, found
}

// Add(t, v) returns a new Pot that contains all the elements of t
// plus the value v, using the applicative add
// the second return value is the proximity order of the inserted element
// the third is boolean indicating if the item was found
// it only readlocks the Pot while reading its pot
func Add(t *Pot, val PotVal) (*Pot, int, bool) {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	r, po, found := add(n, val)
	return &Pot{pot: r}, po, found
}

func add(t *pot, val PotVal) (*pot, int, bool) {
	var r *pot
	if t == nil || t.pin == nil {
		r = &pot{
			pin:  val,
			size: t.size + 1,
			po:   t.po,
			bins: t.bins,
		}
		return r, 0, false
	}
	po, found := t.pin.PO(val, t.po)
	if found {
		r = &pot{
			pin:  val,
			size: t.size,
			po:   t.po,
			bins: t.bins,
		}
		return r, po, true
	}

	var p *pot
	var i, j int
	size := t.size
	for i < len(t.bins) {
		n := t.bins[i]
		if n.po == po {
			p, _, found = add(n, val)
			if !found {
				size++
			}
			j++
			break
		}
		if n.po > po {
			break
		}
		i++
		j++
	}
	if p == nil {
		size++
		p = &pot{
			pin:  val,
			size: 1,
			po:   po,
		}
	}

	bins := append([]*pot{}, t.bins[:i]...)
	bins = append(bins, p)
	bins = append(bins, t.bins[j:]...)
	r = &pot{
		pin:  t.pin,
		size: size,
		po:   t.po,
		bins: bins,
	}

	return r, po, found
}

// T.Re move(v) deletes v from the Pot and returns
// the proximity order of v and a boolean value indicating
// if the value was found
// Remove locks Pot while using applicative remove on its pot
func (t *Pot) Remove(val PotVal) (po int, found bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.pot, po, found = remove(t.pot, val)
	return po, found
}

// Remove(t, v) returns a new Pot that contains all the elements of t
// minus the value v, using the applicative remove
// the second return value is the proximity order of the inserted element
// the third is boolean indicating if the item was found
// it only readlocks the Pot while reading its pot
func Remove(t *Pot, v PotVal) (*Pot, int, bool) {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	r, po, found := remove(n, v)
	return &Pot{pot: r}, po, found
}

func remove(t *pot, val PotVal) (r *pot, po int, found bool) {
	size := t.size
	po, found = t.pin.PO(val, t.po)
	if found {
		size--
		if size == 0 {
			r = &pot{
				po: t.po,
			}
			return r, po, true
		}
		i := len(t.bins) - 1
		last := t.bins[i]
		r = &pot{
			pin:  last.pin,
			bins: append(t.bins[:i], last.bins...),
			size: size,
			po:   t.po,
		}
		return r, t.po, true
	}

	var p *pot
	var i, j int
	for i < len(t.bins) {
		n := t.bins[i]
		if n.po == po {
			p, po, found = remove(n, val)
			if found {
				size--
			}
			j++
			break
		}
		if n.po > po {
			return t, po, false
		}
		i++
		j++
	}
	bins := t.bins[:i]
	if p != nil && p.pin != nil {
		bins = append(bins, p)
	}
	bins = append(bins, t.bins[j:]...)
	r = &pot{
		pin:  val,
		size: size,
		po:   t.po,
		bins: bins,
	}
	return r, po, found
}

// Swap(k, f) looks up the item at k
// and applies the function f to the value v at k or nil if the item is not found
// if f returns nil, the element is removed
// if f returns v' <> v then v' is inserted into the Pot
// if v' == v the pot is not changed
// it panics if v'.PO(k, 0) says v and k are not equal
func (t *Pot) Swap(val PotVal, f func(v PotVal) PotVal) (po int, found bool, change bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	var t0 *pot
	t0, po, found, change = swap(t.pot, val, f)
	if change {
		t.pot = t0
	}
	return po, found, change
}

func swap(t *pot, k PotVal, f func(v PotVal) PotVal) (r *pot, po int, found bool, change bool) {
	var val PotVal
	if t == nil || t.pin == nil {
		val = f(nil)
		if val == nil {
			return t, t.po, false, false
		}
		if _, eq := val.PO(k, t.po); !eq {
			panic("value key mismatch")
		}
		r = &pot{
			pin:  val,
			size: t.size + 1,
			po:   t.po,
			bins: t.bins,
		}
		return r, t.po, false, true
	}
	size := t.size
	if k == nil {
		panic("k is nil")
	}
	po, found = k.PO(t.pin, t.po)
	if found {
		val = f(t.pin)
		if val == nil {
			size--
			if size == 0 {
				r = &pot{
					po: t.po,
				}
				return r, po, true, true
			}
			i := len(t.bins) - 1
			last := t.bins[i]
			r = &pot{
				pin:  last.pin,
				bins: append(t.bins[:i], last.bins...),
				size: size,
				po:   t.po,
			}
			return r, t.po, true, true
			// remove element
		} else if val == t.pin {
			return nil, po, true, false
		} else { // add element
			r = &pot{
				pin:  val,
				size: t.size,
				po:   t.po,
				bins: t.bins,
			}
			return r, po, true, true
		}
	}

	var p *pot
	var i, j int
	for i < len(t.bins) {
		n := t.bins[i]
		if n.po == po {
			p, po, found, change = swap(n, k, f)
			if !change {
				return nil, po, found, false
			}
			size += p.size - n.size
			j++
			break
		}
		if n.po > po {
			break
		}
		i++
		j++
	}
	if p == nil {
		val := f(nil)
		if val == nil {
			return nil, po, false, false
		}
		size++
		p = &pot{
			pin:  val,
			size: 1,
			po:   po,
		}
	}

	bins := append([]*pot{}, t.bins[:i]...)
	if p.pin != nil {
		bins = append(bins, p)
	}
	bins = append(bins, t.bins[j:]...)
	r = &pot{
		pin:  t.pin,
		size: size,
		po:   t.po,
		bins: bins,
	}

	return r, po, found, true
}

// t0.Merge(t1) changes t0 to contain all the elements of t1
// it locks t0, but only readlocks t1 while taking its pot
// uses applicative union
func (t *Pot) Merge(t1 *Pot) (c int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t1.lock.RLock()
	n1 := t1.pot
	t1.lock.RUnlock()
	t.pot, c = union(t.pot, n1)
	return c
}

// Union(t0, t1) return the union of t0 and t1
// it only readlocks the Pot-s to read their pots and
// calculates the union using the applicative union
// the second return value is the number of common elements
func Union(t0, t1 *Pot) (*Pot, int) {
	t0.lock.RLock()
	n0 := t0.pot
	t0.lock.RUnlock()
	t1.lock.RLock()
	n1 := t1.pot
	t1.lock.RUnlock()

	p, c := union(n0, n1)
	return &Pot{
		pot: p,
	}, c
}

func union(t0, t1 *pot) (*pot, int) {
	if t0 == nil || t0.size == 0 {
		return t1, 0
	}
	if t1 == nil || t1.size == 0 {
		return t0, 0
	}
	po, eq := t0.pin.PO(t1.pin, 0)
	var pin PotVal
	var bins []*pot
	var mis []int
	wg := &sync.WaitGroup{}
	pin0 := t0.pin
	pin1 := t1.pin
	bins0 := t0.bins
	bins1 := t1.bins
	var i0, i1 int
	var common int

	for {
		l0 := len(bins0)
		l1 := len(bins1)
		var n0, n1 *pot
		var p0, p1 int
		var a0, a1 bool

		for {

			if !a0 && i0 < l0 && bins0[i0].po <= po {
				n0 = bins0[i0]
				p0 = n0.po
				a0 = p0 == po
			} else {
				a0 = true
			}

			if !a1 && i1 < l1 && bins1[i1].po <= po {
				n1 = bins1[i1]
				p1 = n1.po
				a1 = p1 == po
			} else {
				a1 = true
			}
			if a0 && a1 {
				break
			}

			switch {
			case (p0 < p1 || a1) && !a0:
				bins = append(bins, n0)
				i0++
				n0 = nil
			case (p1 < p0 || a0) && !a1:
				bins = append(bins, n1)
				i1++
				n1 = nil
			case p1 < po:
				bl := len(bins)
				bins = append(bins, nil)
				ml := len(mis)
				mis = append(mis, 0)
				wg.Add(1)
				go func(b, m int, m0, m1 *pot) {
					defer wg.Done()
					bins[b], mis[m] = union(m0, m1)
				}(bl, ml, n0, n1)
				i0++
				i1++
				n0 = nil
				n1 = nil
			}
		}

		if eq {
			common++
			pin = pin1
			break
		}

		var size0 int
		for _, n := range bins0[i0:] {
			size0 += n.size
		}

		np := &pot{
			pin:  pin0,
			bins: bins0[i0:],
			size: size0 + 1,
			po:   po,
		}

		bins2 := []*pot{np}
		if n0 == nil {
			pin0 = pin1
			po = maxkeylen + 1
			eq = true
			common--
		} else {
			bins2 = append(bins2, n0.bins...)
			pin0 = pin1
			pin1 = n0.pin
			po, eq = pin0.PO(pin1, n0.po)
		}
		bins0 = bins1
		bins1 = bins2
		i0 = i1
		i1 = 0

	}

	wg.Wait()
	for _, c := range mis {
		common += c
	}
	n := &pot{
		pin:  pin,
		bins: bins,
		size: t0.size + t1.size - common,
		po:   t0.po,
	}
	return n, common
}

// Each(f) is a synchronous iterator over the bins of a node
// respecting an ordering
// proximity > pinnedness
func (t *Pot) Each(f func(PotVal, int) bool) bool {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	return n.each(f)
}

func (t *pot) each(f func(PotVal, int) bool) bool {
	var next bool
	for _, n := range t.bins {
		next = n.each(f)
		if !next {
			return false
		}
	}
	return f(t.pin, t.po)
}

// EachFrom(f, start) is a synchronous iterator over the elements of a pot
// within the inclusive range starting from proximity order start
// the function argument is passed the value and the proximity order wrt the root pin
// it does NOT include the pinned item of the root
// respecting an ordering
// proximity > pinnedness
// the iteration ends if the function return false or there are no more elements
// end of a po range can be implemented since po is passed to the function
func (t *Pot) EachFrom(f func(PotVal, int) bool, po int) bool {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	return n.eachFrom(f, po)
}

func (t *pot) eachFrom(f func(PotVal, int) bool, po int) bool {
	var next bool
	_, lim := t.getPos(po)
	for i := lim; i < len(t.bins); i++ {
		n := t.bins[i]
		next = n.each(f)
		if !next {
			return false
		}
	}
	return f(t.pin, t.po)
}

// EachBin iterates over bins of the pivot node and offers iterators to the caller on each
// subtree passing the proximity order and the size
// the iteration continues until the function's return value is false
// or there are no more subtries
func (t *Pot) EachBin(val PotVal, po int, f func(int, int, func(func(val PotVal, i int) bool) bool) bool) {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	n.eachBin(val, po, f)
}

func (t *pot) eachBin(val PotVal, po int, f func(int, int, func(func(val PotVal, i int) bool) bool) bool) {
	if t == nil || t.size == 0 {
		return
	}
	spr, _ := t.pin.PO(val, t.po)
	_, lim := t.getPos(spr)
	var size int
	var n *pot
	for i := 0; i < lim; i++ {
		n = t.bins[i]
		size += n.size
		if n.po < po {
			continue
		}
		if !f(n.po, n.size, n.each) {
			return
		}
	}
	if lim == len(t.bins) {
		f(spr, 1, func(g func(PotVal, int) bool) bool {
			return g(t.pin, spr)
		})
		return
	}
	n = t.bins[lim]

	spo := spr
	if n.po == spr {
		spo++
		size += n.size
	}
	if !f(spr, t.size-size, func(g func(PotVal, int) bool) bool {
		return t.eachFrom(func(v PotVal, j int) bool {
			return g(v, spr)
		}, spo)
	}) {
		return
	}
	if spo > spr {
		n.eachBin(val, spo, f)
	}
}

// syncronous iterator over neighbours of any target val
// the order of elements retrieved reflect proximity order to the target
// TODO: add maximum proxbin to start range of iteration
func (t *Pot) EachNeighbour(val PotVal, f func(PotVal, int) bool) bool {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	return n.eachNeighbour(val, f)
}

func (t *pot) eachNeighbour(val PotVal, f func(PotVal, int) bool) bool {
	if t == nil || t.size == 0 {
		return false
	}
	var next bool
	l := len(t.bins)
	var n *pot
	ir := l
	il := l
	po, eq := t.pin.PO(val, t.po)
	if !eq {
		n, il = t.getPos(po)
		if n != nil {
			next = n.eachNeighbour(val, f)
			if !next {
				return false
			}
			ir = il
		} else {
			ir = il - 1
		}
	}

	next = f(t.pin, po)
	if !next {
		return false
	}

	for i := l - 1; i > ir; i-- {
		next = t.bins[i].each(func(v PotVal, _ int) bool {
			return f(v, po)
		})
		if !next {
			return false
		}
	}

	for i := il - 1; i >= 0; i-- {
		n := t.bins[i]
		next = n.each(func(v PotVal, _ int) bool {
			return f(v, n.po)
		})
		if !next {
			return false
		}
	}
	return true
}

// EachNeighnbourAsync(val, max, maxPos, f, wait) is an asyncronous iterator
// over elements not closer than maxPos wrt val.
// val does not need to be match an element of the pot, but if it does, and
// maxPos is keylength than it is included in the iteration
// Calls to f are parallelised, the order of calls is undefined.
// proximity order is respected in that there is no element in the pot that
// is not visited if a closer node is visited.
// The iteration is finished when max number of nearest nodes is visited
// or if the entire there are no nodes not closer than maxPos that is not visited
// if wait is true, the iterator returns only if all calls to f are finished
// TODO: implement minPos for proper prox range iteration
func (t *Pot) EachNeighbourAsync(val PotVal, max int, maxPos int, f func(PotVal, int), wait bool) {
	t.lock.RLock()
	n := t.pot
	t.lock.RUnlock()
	if max > t.size {
		max = t.size
	}
	var wg *sync.WaitGroup
	if wait {
		wg = &sync.WaitGroup{}
	}
	_ = n.eachNeighbourAsync(val, max, maxPos, f, wg)
	if wait {
		wg.Wait()
	}
}

func (t *pot) eachNeighbourAsync(val PotVal, max int, maxPos int, f func(PotVal, int), wg *sync.WaitGroup) (extra int) {

	l := len(t.bins)
	var n *pot
	il := l
	ir := l
	// ic := l

	po, eq := t.pin.PO(val, t.po)

	// if po is too close, set the pivot branch (pom) to maxPos
	pom := po
	if pom > maxPos {
		pom = maxPos
	}
	n, il = t.getPos(pom)
	ir = il
	// if pivot branch exists and po is not too close, iterate on the pivot branch
	if pom == po {
		if n != nil {

			m := n.size
			if max < m {
				m = max
			}
			max -= m

			extra = n.eachNeighbourAsync(val, m, maxPos, f, wg)

		} else {
			if !eq {
				ir--
			}
		}
	} else {
		extra++
		max--
		if n != nil {
			il++
		}
		// before checking max, add up the extra elements
		// on the close branches that are skipped (if po is too close)
		for i := l - 1; i >= il; i-- {
			s := t.bins[i]
			m := s.size
			if max < m {
				m = max
			}
			max -= m
			extra += m
		}
	}

	var m int
	if pom == po {

		m, max, extra = need(1, max, extra)
		if m <= 0 {
			return
		}

		if wg != nil {
			wg.Add(1)
		}
		go func() {
			if wg != nil {
				defer wg.Done()
			}
			f(t.pin, po)
		}()

		// otherwise iterats
		for i := l - 1; i > ir; i-- {
			n := t.bins[i]

			m, max, extra = need(n.size, max, extra)
			if m <= 0 {
				return
			}

			if wg != nil {
				wg.Add(m)
			}
			go func(pn *pot, pm int) {
				pn.each(func(v PotVal, _ int) bool {
					if wg != nil {
						defer wg.Done()
					}
					f(v, po)
					pm--
					return pm > 0
				})
			}(n, m)

		}
	}

	// iterate branches that are farther tham pom with their own po
	for i := il - 1; i >= 0; i-- {
		n := t.bins[i]
		// the first time max is less than the size of the entire branch
		// wait for the pivot thread to release extra elements
		m, max, extra = need(n.size, max, extra)
		if m <= 0 {
			return
		}

		if wg != nil {
			wg.Add(m)
		}
		go func(pn *pot, pm int) {
			pn.each(func(v PotVal, _ int) bool {
				if wg != nil {
					defer wg.Done()
				}
				f(v, pn.po)
				pm--
				return pm > 0
			})
		}(n, m)

	}
	return max + extra
}

// getPos(n) returns the forking node at PO n and its index if it exists
// otherwise nil
// caller is suppoed to hold the lock
func (t *pot) getPos(po int) (n *pot, i int) {
	for i, n = range t.bins {
		if po > n.po {
			continue
		}
		if po < n.po {
			return nil, i
		}
		return n, i
	}
	return nil, len(t.bins)
}

// need(m, max, extra) uses max m out of extra, and then max
// if needed, returns the adjusted counts
func need(m, max, extra int) (int, int, int) {
	if m <= extra {
		return m, max, extra - m
	}
	max += extra - m
	if max <= 0 {
		return m + max, 0, 0
	}
	return m, max, 0
}

func (t *pot) String() string {
	return t.sstring("")
}

func (t *pot) sstring(indent string) string {
	var s string
	indent += "  "
	s += fmt.Sprintf("%v%v (%v) %v \n", indent, t.pin, t.po, t.size)
	for _, n := range t.bins {
		s += fmt.Sprintf("%v%v\n", indent, n.sstring(indent))
	}
	return s
}
