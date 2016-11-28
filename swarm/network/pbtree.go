package network

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
PbTree implements a pinned binary tree.
It is a generic container type for objects implementing the PbVal interface
Each item is pinned to the node at pos x such that all items pinned to all
ancestor nodes share an at least x bits long key prefix

PbTree
* does not need to copy keys of the item type.
* retrieval, insertion and deletion by key involves log(n) pointer lookups
* for any item retrieval respects proximity order on logarithmic distance
* provide syncronous iterators  respecting proximity ordering  wrt any item
* provide asyncronous iterator (for parallel execution of operations) over n items
* allows cheap iteration over ranges
* TODO: asymmetric parallelisable merge

*/
// PbTree is the root node type k(same for root, branching node and leat)
type PbTree struct {
	lock sync.RWMutex
	*pbTree
}

// pbTree is the node type (same for root, branching node and leat)
type pbTree struct {
	pin  PbVal
	bins []*pbTree
	size int
	pos  int
}

// PbVal is the interface the generic container item should implement
type PbVal interface {
	Prefix(PbVal, int) (pos int, eq bool)
	String() string
}

// PbTree constructor. Requires  value of type PbVal to pin
// and pos to point to a span in the PbVal key
// The pinned item counts towards the size
func NewPbTree(v PbVal, pos int) *PbTree {
	return &PbTree{
		pbTree: &pbTree{
			pin:  v,
			pos:  pos,
			size: 1,
		},
	}
}

// Pin() returns the pinned element (key) of the PbTree
func (t *PbTree) Pin() PbVal {
	return t.pin
}

// Size() returns the number of values in the PbTree
func (t *PbTree) Size() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.size
}

// Add(v) inserts v into the PbTree
func (t *PbTree) Add(val PbVal) (pos int, found bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.add(val)
}

func (t *pbTree) add(val PbVal) (pos int, found bool) {
	if t.pin == nil {
		t.pin = val
		t.size = 1
		return 0, false
	}
	pos, found = val.Prefix(t.pin, t.pos)
	if found {
		t.pin = val
		return pos, true
	}

	n, j := t.getPos(pos)
	if n != nil {
		p, f := n.add(val)
		if !f {
			t.size++
		}
		return p, f
	}
	// insert empty sub-pbTree and pin it to val
	ins := &pbTree{
		pin:  val,
		pos:  pos,
		size: 1,
	}
	t.size++
	t.bins = append(t.bins, nil)
	copy(t.bins[j+1:], t.bins[j:])
	t.bins[j] = ins
	return pos, false
}

// Remove(v) deletes v from the PbTree and returns
// the proximity order of v
func (t *PbTree) Remove(val PbVal) (pos int, found bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.remove(val)
}

func (t *pbTree) remove(val PbVal) (pos int, found bool) {
	pos, found = val.Prefix(t.pin, t.pos)
	if found {
		t.size -= 1
		if t.size == 0 {
			t.pin = nil
			return t.pos, true
		}
		i := len(t.bins) - 1
		last := t.bins[i]
		t.bins = append(t.bins[:i], last.bins...)
		t.pin = last.pin
		return t.pos, true
	}
	for j, n := range t.bins {
		if n.pos == pos {
			p, f := n.remove(val)
			if f {
				t.size--
			}
			last := len(t.bins) - 1
			copy(t.bins[j:], t.bins[j+1:])
			t.bins[last] = nil // or the zero value of T
			t.bins = t.bins[:last]
			return p, f
		}
		if n.pos > pos {
			return 0, false
		}
	}
	return 0, false
}

func (t *PbTree) Merge(t1 *PbTree) int {
	t.lock.Lock()
	defer t.lock.Unlock()
	t1.lock.RLock()
	defer t1.lock.RUnlock()
	return t.merge(t1.pbTree)
}

func (t *pbTree) merge(t1 *pbTree) int {
	if t.pin == nil {
		if t1.pin == nil {
			return 0
		}
		t.pin = t1.pin
		t.size = 1
		return t.merge(t1)
	}
	i := 0
	j := 0
	added := 0

	var bins []*pbTree
	var m, t2 *pbTree
	var is []int
	pos, _ := t.pin.Prefix(t1.pin, 0)
	n, l := t.getPos(pos)
	for {
		glog.V(4).Infof("%v-%v, i: %v, j: %v, pos: %v, n: %v, l: %v", t, t1, i, j, pos, n, l)

		if i == len(t.bins) && j == len(t1.bins) {
			if l < len(t.bins) {
				glog.V(4).Infof("l < len(t.bins): break")
				break
			}
		}

		if i == l && j <= len(t1.bins) {
			if m == nil {
				if n == nil {
					n = &pbTree{
						pin:  t1.pin,
						size: 1,
						pos:  pos,
					}
					added++
				} else {
					_, found := n.add(t1.pin)
					if !found {
						added++
					}
					glog.V(4).Infof("%v-%v: i: %v, j: %v. adding t1.pin (found: %v) to n: %v", t.pin, t1.pin, i, j, found, n)
				}
				m = n
				bins = append(bins, n)
			}
			if j < len(t1.bins) {
				if t2 == nil && t1.bins[j].pos == 0 {
					t2 = t1.bins[j]
					_, found := n.add(t2.pin)
					if !found {
						added++
					}
					glog.V(4).Infof("%v-%v: i: %v, j: %v. will merge into 0 the 0 pos branch from 1: %v", t.pin, t1.pin, i, j, t1.bins[j])
					glog.V(4).Infof("%v-%v: i: %v, j: %v. adding t2.pin: %v (found: %v) to n: %v", t.pin, t1.pin, i, j, t2.pin, found, n)
				} else {
					glog.V(4).Infof("%v-%v: i: %v, j: %v. merge into 0 from 1: %v", t.pin, t1.pin, i, j, t1.bins[j])
					added += n.merge(t1.bins[j])
				}
				j++
				continue
			}
			if t2 == nil && l == len(t.bins) {
				break
			}
			if l < len(t.bins) {
				i++
			}
			glog.V(4).Infof("%v-%v: i: %v, j: %v. t1 reset to %v", t.pin, t1.pin, i, j, t2)
			if t2 != nil {
				t1 = t2
				j = 0
				m = nil
				t2 = nil
				pos, _ = t.pin.Prefix(t1.pin, 0)
				n, l = t.getPos(pos)
			}
			continue
		}

		if j == len(t1.bins) || t1.bins[j].pos > t.bins[i].pos {
			glog.V(4).Infof("%v-%v: i: %v, j: %v. insert from 0: %v", t.pin, t1.pin, i, j, t.bins[i])
			bins = append(bins, t.bins[i])
			i++
			continue
		}

		if i < l && t1.bins[j].pos < t.bins[i].pos {
			glog.V(4).Infof("%v-%v: i: %v, j: %v. insert from 1: %v", t.pin, t1.pin, i, j, t1.bins[j])
			m := &pbTree{}
			added += m.merge(t1.bins[j])
			bins = append(bins, n)
			j++
			continue
		}

		glog.V(4).Infof("%v-%v: i: %v, j: %v. merge: %v", t.pin, t1.pin, i, j, t.bins[i], t1.bins[j])
		bins = append(bins, t.bins[i])
		is = append(is, i)
		i++
		j++
	}
	t.bins = bins
	wg := sync.WaitGroup{}
	if len(is) > 0 {
		wg.Add(len(is))
		for _, i := range is {
			go func(k int) {
				defer wg.Done()
				is[k] = bins[k].merge(t1.bins[k])
			}(i)
		}
		wg.Wait()
		for _, a := range is {
			added += a
		}
	}
	glog.V(4).Infof("%v-%v: added: %v", t.pin, t1.pin, added)
	t.size += added
	return added
}

// func (t *PbTree) Traverse(f func(val PbVal, pos int) (next bool, fork bool)) *PbTree {
//   t.lock.Lock()
//   defer t.lock.Unlock()
//   return t.traverse(f)
// }

// func (t *pbTree) traverse(n *pbTree, f func(val PbVal, pos int) (next bool, fork bool)) *PbTree {

// next, stop = pinf(t.pin,t.pos)
//  if !next
// }

// Each(f) is a synchronous iterator over the bins of a node
// it does NOT include the pinned item of the root
// respecting an ordering
// proximity > pinnedness
func (t *PbTree) Each(f func(PbVal, int) bool) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.each(f)
}

func (t *pbTree) each(f func(PbVal, int) bool) bool {
	var next bool
	for _, n := range t.bins {
		next = n.each(f)
		if !next {
			return false
		}
	}
	next = f(t.pin, t.pos)
	if !next {
		return false
	}

	return true
}

// syncronous iterator over neighbours of any target val
// even if an item at val's exact address is in the pbtree,
// it is not included in the iteration: $val \not\in Neighbours(val)$
func (t *PbTree) EachNeighbour(val PbVal, f func(PbVal, int) bool) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.eachNeighbour(val, f)
}

func (t *pbTree) eachNeighbour(val PbVal, f func(PbVal, int) bool) bool {
	var next bool
	l := len(t.bins)
	var n *pbTree
	ir := l
	il := l
	pos, eq := val.Prefix(t.pin, t.pos)
	if !eq {
		n, il = t.getPos(pos)
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

	next = f(t.pin, pos)
	if !next {
		return false
	}

	for i := l - 1; i > ir; i-- {
		next = t.bins[i].each(func(v PbVal, _ int) bool {
			return f(v, pos)
		})
		if !next {
			return false
		}
	}

	for i := il - 1; i >= 0; i-- {
		n := t.bins[i]
		next = n.each(func(v PbVal, _ int) bool {
			return f(v, n.pos)
		})
		if !next {
			return false
		}
	}
	return true
}

func (t *PbTree) EachNeighbourAsync(val PbVal, max int, maxPos int, f func(PbVal, int), wait bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if max > t.size {
		max = t.size
	}
	var wg *sync.WaitGroup
	if wait {
		wg = &sync.WaitGroup{}
	}
	_ = t.eachNeighbourAsync(val, max, maxPos, f, wg)
	if wait {
		wg.Wait()
	}
}

func (t *pbTree) eachNeighbourAsync(val PbVal, max int, maxPos int, f func(PbVal, int), wg *sync.WaitGroup) (extra int) {

	l := len(t.bins)
	var n *pbTree
	il := l
	ir := l
	// ic := l

	pos, eq := val.Prefix(t.pin, t.pos)
	glog.V(4).Infof("pin %v: each neighbour iteration async. count: %v/%v, t.pos: %v, pos: %v, maxPos: %v", t.pin, max, t.size, t.pos, pos, maxPos)

	// if pos is too close, set the pivot branch (pom) to maxPos
	pom := pos
	if pom > maxPos {
		pom = maxPos
	}
	n, il = t.getPos(pom)
	ir = il
	// if pivot branch exists and pos is not too close, iterate on the pivot branch
	if pom == pos {
		if n != nil {

			m := n.size
			if max < m {
				m = max
			}
			max -= m

			glog.V(4).Infof("pin %v recursive branch %v pos: %v/%v (%v), count: %v/%v", t.pin, n.pin, n.pos, maxPos, il, m, max)

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
		// on the close branches that are skipped (if pos is too close)
		for i := l - 1; i >= il; i-- {
			s := t.bins[i]
			m := s.size
			if max < m {
				m = max
			}
			max -= m
			extra += m
		}
		glog.V(4).Infof("count extra pos: %v/%v -> %v", pos, maxPos, extra)
	}
	glog.V(4).Infof("branch %v: %v/%v, il: %v, ir: %v, l: %v, extra: %v", t.pin, pos, maxPos, il, ir, l, extra)

	var m int
	// if max <= 0 {
	// 	return
	// }
	// unless pos was too close, call f on the pinned element
	if pom == pos {

		glog.V(4).Infof("pinned val %v, t.pos: %v, pos: %v (%v), count: %v, max: %v", t.pin, pos, maxPos, "pin", 1, max)
		glog.V(4).Infof("BEFORE %v %v %v", 1, max, extra)
		m, max, extra = need(1, max, extra)
		if m <= 0 {
			return
		}
		glog.V(4).Infof("AFTER %v %v %v", 1, max, extra)
		glog.V(4).Infof("pinned val %v, t.pos: %v, pos: %v (%v), count: %v, max: %v", t.pin, pos, maxPos, "pin", 1, max)

		if wg != nil {
			wg.Add(1)
		}
		go func() {
			if wg != nil {
				defer wg.Done()
			}
			f(t.pin, pos)
		}()

		// otherwise iterats
		glog.V(4).Infof("closer branches %v: %v/%v, il: %v, ir: %v, l: %v", t.pin, pos, maxPos, il, ir, l)
		for i := l - 1; i > ir; i-- {
			n := t.bins[i]

			glog.V(4).Infof("branch %v closer branch %v pos: %v/%v (%v), count: %v, size: %v, max: %v", t.pin, n.pin, pos, maxPos, i, m, n.size, max)
			glog.V(4).Infof("BEFORE %v %v %v", n.size, max, extra)
			m, max, extra = need(n.size, max, extra)
			if m <= 0 {
				glog.V(4).Infof("branch %v closer branch %v NOT ADDED pos: %v/%v (%v), count: %v, size: %v, max: %v", t.pin, n.pin, pos, maxPos, i, m, n.size, max)
				return
			}
			glog.V(4).Infof("AFTER %v %v %v", m, max, extra)

			glog.V(4).Infof("branch %v closer branch %v pos: %v/%v (%v), count: %v, size: %v, max: %v", t.pin, n.pin, pos, maxPos, i, m, n.size, max)

			if wg != nil {
				wg.Add(m)
			}
			go func(pn *pbTree, pm int) {
				pn.each(func(v PbVal, _ int) bool {
					if wg != nil {
						defer wg.Done()
					}
					glog.V(4).Infof("branch %v call f on %v pos: %v/%v (%v), count: %v/%v", pn.pin, v, pos, maxPos, i, pm, max)
					f(v, pos)
					pm--
					return pm > 0
				})
			}(n, m)

		}
	}

	// if max <= 0 {
	// 	return
	// }
	// iterate branches that are farther tham pom with their own po
	glog.V(4).Infof("further branches %v: %v/%v, il: %v, ir: %v, l: %v, extra: %v", t.pin, pos, maxPos, il, ir, l, extra)
	for i := il - 1; i >= 0; i-- {
		n := t.bins[i]
		// the first time max is less than the size of the entire branch
		// wait for the pivot thread to release extra elements
		glog.V(4).Infof("branch %v further branch %v pos: %v/%v (%v), count: %v, size: %v, max: %v", t.pin, n.pin, n.pos, maxPos, i, m, n.size, max)
		glog.V(4).Infof("BEFORE %v %v %v", n.size, max, extra)
		m, max, extra = need(n.size, max, extra)
		if m <= 0 {
			return
		}
		glog.V(4).Infof("AFTER %v %v %v", m, max, extra)
		glog.V(4).Infof("branch %v further branch %v pos: %v/%v (%v), count: %v, size: %v, max: %v", t.pin, n.pin, n.pos, maxPos, i, m, n.size, max)

		if wg != nil {
			wg.Add(m)
		}
		go func(pn *pbTree, pm int) {
			pn.each(func(v PbVal, _ int) bool {
				if wg != nil {
					defer wg.Done()
				}
				f(v, pn.pos)
				glog.V(4).Infof("branch %v call f on %v pos: %v/%v (%v), count: %v/%v", pn.pin, v, pn.pos, maxPos, i, pm, max)
				pm--
				return pm > 0
			})
		}(n, m)

	}
	return max + extra

}

// getPos(n) returns the forking node at PO n and its index if it exists
// otherwise nil
// caller is supposed to hold the lock
func (t *pbTree) getPos(pos int) (n *pbTree, i int) {
	for i, n = range t.bins {
		if pos > n.pos {
			continue
		}
		if pos < n.pos {
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

// func need(max int, more chan int) int {
// 	// if max <= 0 {
// 	c, ok := <-more
// 	if ok {
// 		defer close(more)
// 		if c > 0 {
// 			glog.V(4).Infof("need: %v + %v", max, c)
// 			return max + c
// 		}
// 	}
// 	// }
// 	return max
// }

func (t *pbTree) String() string {
	return t.sstring("")
}

func (t *pbTree) sstring(indent string) string {
	var s string
	indent += "  "
	s += fmt.Sprintf("%v%v (%v) %v \n", indent, t.pin, t.pos, t.size)
	for _, n := range t.bins {
		s += fmt.Sprintf("%v%v\n", indent, n.sstring(indent))
	}
	return s
}
