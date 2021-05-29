// Copyright 2019 The go-ethereum Authors
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

package txpool

import (
	"container/heap"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type dbTx struct {
	nonce uint64
	price *big.Int
	slot  uint64
	addr  common.Address
}

type dbNonceList []dbTx

func (d dbNonceList) Len() int      { return len(d) }
func (d dbNonceList) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d dbNonceList) Less(i, j int) bool {
	return d[i].price.Cmp(d[j].price) < 0
}

func (d *dbNonceList) Push(x interface{}) {
	*d = append(*d, x.(dbTx))
}

func (d *dbNonceList) Pop() interface{} {
	old := *d
	n := len(old)
	x := old[n-1]
	*d = old[0 : n-1]
	return x
}

type dbHeap struct {
	m map[common.Address]dbNonceList
}

func newDbHeap() dbHeap {
	return dbHeap{m: make(map[common.Address]dbNonceList)}
}

// Add adds a new entry to the heap.
// Expects the transactions to be ordered between calls.
func (h *dbHeap) Add(entry *txEntry, dbSlot uint64) {
	h.m[entry.sender] = append(h.m[entry.sender], dbTx{nonce: entry.tx.Nonce(), price: entry.price, slot: dbSlot})
}

// Pop removes len elements from the heap.
// The elements are sorted by gas price and executable
func (h *dbHeap) Pop(length int) []uint64 {
	if length > len(h.m) {
		length = len(h.m)
	}
	results := make([]uint64, 0, length)
	heads := make(dbNonceList, 0, len(h.m))
	for sender, list := range h.m {
		elem := list[0]
		elem.addr = sender
		heads = append(heads, elem)
		h.m[sender] = list[1:]
	}
	heap.Init(&heads)
	for i := 0; i < length && len(h.m) > 0 && len(heads) > 0; i++ {
		acc := heads[0].addr
		results = append(results, heads[0].slot)
		if txs, ok := h.m[acc]; ok && len(txs) > 0 {
			heads[0], h.m[acc] = txs[0], txs[1:]
			heads[0].addr = acc
			heap.Fix(&heads, 0)
		} else {
			delete(h.m, acc)
			heap.Pop(&heads)
		}
	}
	// Write back heads for next pop operation
	for _, head := range heads {
		h.m[head.addr] = append(h.m[head.addr], dbTx{})
		copy(h.m[head.addr][1:], h.m[head.addr])
		h.m[head.addr][0] = head
	}
	return results
}

func (h *dbHeap) Len() int {
	len := 0
	for _, list := range h.m {
		len += list.Len()
	}
	return len
}
