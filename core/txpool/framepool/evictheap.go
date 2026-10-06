// Copyright 2026 The go-ethereum Authors
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

package framepool

import (
	"bytes"
	"cmp"
	"container/heap"

	"github.com/ethereum/go-ethereum/core/types"
)

func numSlots(tx *types.Transaction) uint64 {
	return (tx.Size() + txSlotSize - 1) / txSlotSize
}

// comparePriority orders the first eviction candidate before the most valuable
// transaction. A missing expiry is strictly better than any finite deadline.
// Equal deadlines and effective tips have equal admission priority.
func comparePriority(a, b *frameTx) int {
	switch {
	case a.expiryDeadline == nil && b.expiryDeadline != nil:
		return 1
	case a.expiryDeadline != nil && b.expiryDeadline == nil:
		return -1
	case a.expiryDeadline != nil && b.expiryDeadline != nil:
		if c := cmp.Compare(*a.expiryDeadline, *b.expiryDeadline); c != 0 {
			return c
		}
	}
	return a.effectiveTip.Cmp(b.effectiveTip)
}

// compareEviction breaks equal priorities deterministically, without allowing
// a hash tie-break to count as a fee bump for admission.
func compareEviction(a, b *frameTx) int {
	if c := comparePriority(a, b); c != 0 {
		return c
	}
	ah, bh := a.tx.Hash(), b.tx.Hash()
	return bytes.Compare(ah[:], bh[:])
}

type evictHeap []*frameTx

func (h evictHeap) Len() int           { return len(h) }
func (h evictHeap) Less(i, j int) bool { return compareEviction(h[i], h[j]) < 0 }
func (h evictHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}
func (h *evictHeap) Push(x any) {
	tx := x.(*frameTx)
	tx.index = len(*h)
	*h = append(*h, tx)
}
func (h *evictHeap) Pop() any {
	last := len(*h) - 1
	tx := (*h)[last]
	(*h)[last] = nil
	*h = (*h)[:last]
	tx.index = -1
	return tx
}

// evictionCursor walks a heap in priority order without modifying it. Only the
// visited frontier is allocated; a rejected admission never pops live entries.
type evictionCursor struct {
	heap  evictHeap
	front []int
}

func (c evictionCursor) Len() int { return len(c.front) }
func (c evictionCursor) Less(i, j int) bool {
	return compareEviction(c.heap[c.front[i]], c.heap[c.front[j]]) < 0
}
func (c evictionCursor) Swap(i, j int) { c.front[i], c.front[j] = c.front[j], c.front[i] }
func (c *evictionCursor) Push(x any)   { c.front = append(c.front, x.(int)) }
func (c *evictionCursor) Pop() any {
	last := len(c.front) - 1
	i := c.front[last]
	c.front = c.front[:last]
	return i
}
func (c *evictionCursor) next() *frameTx {
	if len(c.front) == 0 {
		return nil
	}
	i := heap.Pop(c).(int)
	for _, child := range [...]int{2*i + 1, 2*i + 2} {
		if child < len(c.heap) {
			heap.Push(c, child)
		}
	}
	return c.heap[i]
}
