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

// Package pot see doc.go
package pot

import (
	"fmt"
	"sync"
)

const (
	maxkeylen = 256
)

// Pot is the node type (same for root, branching node and leaf)
type Pot struct {
	pin  Val
	bins []*Pot
	size int
	po   int
}

// Val is the element type for Pots
type Val interface{}

// Pof is the proximity order comparison operator function
type Pof func(Val, Val, int) (int, bool)

// NewPot constructor. Requires a value of type Val to pin
// and po to point to a span in the Val key
// The pinned item counts towards the size
func NewPot(v Val, po int) *Pot {
	var size int
	if v != nil {
		size++
	}
	return &Pot{
		pin:  v,
		po:   po,
		size: size,
	}
}

// Pin returns the pinned element (key) of the Pot
func (t *Pot) Pin() Val {
	return t.pin
}

// Size returns the number of values in the Pot
func (t *Pot) Size() int {
	if t == nil {
		return 0
	}
	return t.size
}

// Add inserts a new value into the Pot and
// returns the proximity order of v and a boolean
// indicating if the item was found
// Add called on (t, v) returns a new Pot that contains all the elements of t
// plus the value v, using the applicative add
// the second return value is the proximity order of the inserted element
// the third is boolean indicating if the item was found
func Add(t *Pot, val Val, pof Pof) (*Pot, int, bool) {
	return add(t, val, pof)
}

func (t *Pot) clone() *Pot {
	return &Pot{
		pin:  t.pin,
		size: t.size,
		po:   t.po,
		bins: t.bins,
	}
}

func add(t *Pot, val Val, pof Pof) (*Pot, int, bool) {
	var r *Pot
	if t == nil || t.pin == nil {
		r = t.clone()
		r.pin = val
		r.size++
		return r, 0, false
	}
	po, found := pof(t.pin, val, t.po)
	if found {
		r = t.clone()
		r.pin = val
		return r, po, true
	}

	var p *Pot
	var i, j int
	size := t.size
	for i < len(t.bins) {
		n := t.bins[i]
		if n.po == po {
			p, _, found = add(n, val, pof)
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
		p = &Pot{
			pin:  val,
			size: 1,
			po:   po,
		}
	}

	bins := append([]*Pot{}, t.bins[:i]...)
	bins = append(bins, p)
	bins = append(bins, t.bins[j:]...)
	r = &Pot{
		pin:  t.pin,
		size: size,
		po:   t.po,
		bins: bins,
	}

	return r, po, found
}

// Remove deletes element v from the Pot t and returns three parameters:
// 1. new Pot that contains all the elements of t minus the element v;
// 2. proximity order of the removed element v;
// 3. boolean indicating whether the item was found.
func Remove(t *Pot, v Val, pof Pof) (*Pot, int, bool) {
	return remove(t, v, pof)
}

func remove(t *Pot, val Val, pof Pof) (r *Pot, po int, found bool) {
	size := t.size
	po, found = pof(t.pin, val, t.po)
	if found {
		size--
		if size == 0 {
			return &Pot{}, po, true
		}
		i := len(t.bins) - 1
		last := t.bins[i]
		r = &Pot{
			pin:  last.pin,
			bins: append(t.bins[:i], last.bins...),
			size: size,
			po:   t.po,
		}
		return r, t.po, true
	}

	var p *Pot
	var i, j int
	for i < len(t.bins) {
		n := t.bins[i]
		if n.po == po {
			p, po, found = remove(n, val, pof)
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
	r = &Pot{
		pin:  t.pin,
		size: size,
		po:   t.po,
		bins: bins,
	}
	return r, po, found
}

// Swap called on (k, f) looks up the item at k
// and applies the function f to the value v at k or to nil if the item is not found
// if f(v) returns nil, the element is removed
// if f(v) returns v' <> v then v' is inserted into the Pot
// if (v) == v the Pot is not changed
// it panics if Pof(f(v), k) show that v' and v are not key-equal
func Swap(t *Pot, k Val, pof Pof, f func(v Val) Val) (r *Pot, po int, found bool, change bool) {
	var val Val
	if t.pin == nil {
		val = f(nil)
		if val == nil {
			return nil, 0, false, false
		}
		return NewPot(val, t.po), 0, false, true
	}
	size := t.size
	po, found = pof(k, t.pin, t.po)
	if found {
		val = f(t.pin)
		// remove element
		if val == nil {
			size--
			if size == 0 {
				r = &Pot{
					po: t.po,
				}
				// return empty pot
				return r, po, true, true
			}
			// actually remove pin, by merging last bin
			i := len(t.bins) - 1
			last := t.bins[i]
			r = &Pot{
				pin:  last.pin,
				bins: append(t.bins[:i], last.bins...),
				size: size,
				po:   t.po,
			}
			return r, po, true, true
		}
		// element found but no change
		if val == t.pin {
			return t, po, true, false
		}
		// actually modify the pinned element, but no change in structure
		r = t.clone()
		r.pin = val
		return r, po, true, true
	}

	// recursive step
	var p *Pot
	n, i := t.getPos(po)
	if n != nil {
		p, po, found, change = Swap(n, k, pof, f)
		// recursive no change
		if !change {
			return t, po, found, false
		}
		// recursive change
		bins := append([]*Pot{}, t.bins[:i]...)
		if p.size == 0 {
			size--
		} else {
			size += p.size - n.size
			bins = append(bins, p)
		}
		i++
		if i < len(t.bins) {
			bins = append(bins, t.bins[i:]...)
		}
		r = t.clone()
		r.bins = bins
		r.size = size
		return r, po, found, true
	}
	// key does not exist
	val = f(nil)
	if val == nil {
		// and it should not be created
		return t, po, false, false
	}
	// otherwise check val if equal to k
	if _, eq := pof(val, k, po); !eq {
		panic("invalid value")
	}
	///
	size++
	p = &Pot{
		pin:  val,
		size: 1,
		po:   po,
	}

	bins := append([]*Pot{}, t.bins[:i]...)
	bins = append(bins, p)
	if i < len(t.bins) {
		bins = append(bins, t.bins[i:]...)
	}
	r = t.clone()
	r.bins = bins
	r.size = size
	return r, po, found, true
}

// Union called on (t0, t1, pof) returns the union of t0 and t1
// calculates the union using the applicative union
// the second return value is the number of common elements
func Union(t0, t1 *Pot, pof Pof) (*Pot, int) {
	return union(t0, t1, pof)
}

func union(t0, t1 *Pot, pof Pof) (*Pot, int) {
	if t0 == nil || t0.size == 0 {
		return t1, 0
	}
	if t1 == nil || t1.size == 0 {
		return t0, 0
	}
	var pin Val
	var bins []*Pot
	var mis []int
	wg := &sync.WaitGroup{}
	wg.Add(1)
	pin0 := t0.pin
	pin1 := t1.pin
	bins0 := t0.bins
	bins1 := t1.bins
	var i0, i1 int
	var common int

	po, eq := pof(pin0, pin1, 0)

	for {
		l0 := len(bins0)
		l1 := len(bins1)
		var n0, n1 *Pot
		var p0, p1 int
		var a0, a1 bool

		for {

			if !a0 && i0 < l0 && bins0[i0] != nil && bins0[i0].po <= po {
				n0 = bins0[i0]
				p0 = n0.po
				a0 = p0 == po
			} else {
				a0 = true
			}

			if !a1 && i1 < l1 && bins1[i1] != nil && bins1[i1].po <= po {
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
				// wg.Add(1)
				// go func(b, m int, m0, m1 *Pot) {
				// 	defer wg.Done()
				// bins[b], mis[m] = union(m0, m1, pof)
				// }(bl, ml, n0, n1)
				bins[bl], mis[ml] = union(n0, n1, pof)
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

		i := i0
		if len(bins0) > i && bins0[i].po == po {
			i++
		}
		var size0 int
		for _, n := range bins0[i:] {
			size0 += n.size
		}
		np := &Pot{
			pin:  pin0,
			bins: bins0[i:],
			size: size0 + 1,
			po:   po,
		}

		bins2 := []*Pot{np}
		if n0 == nil {
			pin0 = pin1
			po = maxkeylen + 1
			eq = true
			common--

		} else {
			bins2 = append(bins2, n0.bins...)
			pin0 = pin1
			pin1 = n0.pin
			po, eq = pof(pin0, pin1, n0.po)

		}
		bins0 = bins1
		bins1 = bins2
		i0 = i1
		i1 = 0

	}

	wg.Done()
	wg.Wait()
	for _, c := range mis {
		common += c
	}
	n := &Pot{
		pin:  pin,
		bins: bins,
		size: t0.size + t1.size - common,
		po:   t0.po,
	}
	return n, common
}

// Each is a synchronous iterator over the elements of pot with function f.
func (t *Pot) Each(f func(Val) bool) bool {
	return t.each(f)
}

// each is a synchronous iterator over the elements of pot with function f.
// the iteration ends if the function return false or there are no more elements.
func (t *Pot) each(f func(Val) bool) bool {
	if t == nil || t.size == 0 {
		return false
	}
	for _, n := range t.bins {
		if !n.each(f) {
			return false
		}
	}
	return f(t.pin)
}

// eachFrom is a synchronous iterator over the elements of pot with function f,
// starting from certain proximity order po, which is passed as a second parameter.
// the iteration ends if the function return false or there are no more elements.
func (t *Pot) eachFrom(f func(Val) bool, po int) bool {
	if t == nil || t.size == 0 {
		return false
	}
	_, beg := t.getPos(po)
	for i := beg; i < len(t.bins); i++ {
		if !t.bins[i].each(f) {
			return false
		}
	}
	return f(t.pin)
}

// EachBin iterates over bins of the pivot node and offers iterators to the caller on each
// subtree passing the proximity order and the size
// the iteration continues until the function's return value is false
// or there are no more subtries
func (t *Pot) EachBin(val Val, pof Pof, po int, f func(int, int, func(func(val Val) bool) bool) bool) {
	t.eachBin(val, pof, po, f)
}

func (t *Pot) eachBin(val Val, pof Pof, po int, f func(int, int, func(func(val Val) bool) bool) bool) {
	if t == nil || t.size == 0 {
		return
	}
	spr, _ := pof(t.pin, val, t.po)
	_, lim := t.getPos(spr)
	var size int
	var n *Pot
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
		if spr >= po {
			f(spr, 1, func(g func(Val) bool) bool {
				return g(t.pin)
			})
		}
		return
	}

	n = t.bins[lim]

	spo := spr
	if n.po == spr {
		spo++
		size += n.size
	}
	if spr >= po {
		if !f(spr, t.size-size, func(g func(Val) bool) bool {
			return t.eachFrom(func(v Val) bool {
				return g(v)
			}, spo)
		}) {
			return
		}
	}
	if n.po == spr {
		n.eachBin(val, pof, po, f)
	}

}

// EachNeighbour is a synchronous iterator over neighbours of any target val
// the order of elements retrieved reflect proximity order to the target
// TODO: add maximum proxbin to start range of iteration
func (t *Pot) EachNeighbour(val Val, pof Pof, f func(Val, int) bool) bool {
	return t.eachNeighbour(val, pof, f)
}

func (t *Pot) eachNeighbour(val Val, pof Pof, f func(Val, int) bool) bool {
	if t == nil || t.size == 0 {
		return false
	}
	var next bool
	l := len(t.bins)
	var n *Pot
	ir := l
	il := l
	po, eq := pof(t.pin, val, t.po)
	if !eq {
		n, il = t.getPos(po)
		if n != nil {
			next = n.eachNeighbour(val, pof, f)
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
		next = t.bins[i].each(func(v Val) bool {
			return f(v, po)
		})
		if !next {
			return false
		}
	}

	for i := il - 1; i >= 0; i-- {
		n := t.bins[i]
		next = n.each(func(v Val) bool {
			return f(v, n.po)
		})
		if !next {
			return false
		}
	}
	return true
}

// EachNeighbourAsync called on (val, max, maxPos, f, wait) is an asynchronous iterator
// over elements not closer than maxPos wrt val.
// val does not need to be match an element of the Pot, but if it does, and
// maxPos is keylength than it is included in the iteration
// Calls to f are parallelised, the order of calls is undefined.
// proximity order is respected in that there is no element in the Pot that
// is not visited if a closer node is visited.
// The iteration is finished when max number of nearest nodes is visited
// or if the entire there are no nodes not closer than maxPos that is not visited
// if wait is true, the iterator returns only if all calls to f are finished
// TODO: implement minPos for proper prox range iteration
func (t *Pot) EachNeighbourAsync(val Val, pof Pof, max int, maxPos int, f func(Val, int), wait bool) {
	if max > t.size {
		max = t.size
	}
	var wg *sync.WaitGroup
	if wait {
		wg = &sync.WaitGroup{}
	}
	t.eachNeighbourAsync(val, pof, max, maxPos, f, wg)
	if wait {
		wg.Wait()
	}
}

func (t *Pot) eachNeighbourAsync(val Val, pof Pof, max int, maxPos int, f func(Val, int), wg *sync.WaitGroup) (extra int) {
	l := len(t.bins)

	po, eq := pof(t.pin, val, t.po)

	// if po is too close, set the pivot branch (pom) to maxPos
	pom := po
	if pom > maxPos {
		pom = maxPos
	}
	n, il := t.getPos(pom)
	ir := il
	// if pivot branch exists and po is not too close, iterate on the pivot branch
	if pom == po {
		if n != nil {

			m := n.size
			if max < m {
				m = max
			}
			max -= m

			extra = n.eachNeighbourAsync(val, pof, m, maxPos, f, wg)

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
			go func(pn *Pot, pm int) {
				pn.each(func(v Val) bool {
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
		go func(pn *Pot, pm int) {
			pn.each(func(v Val) bool {
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

// getPos called on (n) returns the forking node at PO n and its index if it exists
// otherwise nil
// caller is supposed to hold the lock
func (t *Pot) getPos(po int) (n *Pot, i int) {
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

// need called on (m, max, extra) uses max m out of extra, and then max
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

func (t *Pot) String() string {
	return t.sstring("")
}

func (t *Pot) sstring(indent string) string {
	if t == nil {
		return "<nil>"
	}
	var s string
	indent += "  "
	s += fmt.Sprintf("%v%v (%v) %v \n", indent, t.pin, t.po, t.size)
	for _, n := range t.bins {
		s += fmt.Sprintf("%v%v\n", indent, n.sstring(indent))
	}
	return s
}
