// Copyright 2024 The go-ethereum Authors
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

package pathdb

import (
	"math/bits"
	"sync/atomic"
	"time"
)

const (
	// indexSlots is the number of the slots in the layer index, one per state
	// id, reused in a ring. It must be a power of two above the maximum number
	// of the diff layers.
	indexSlots = 256

	// indexMinRows and indexMaxRows bound the number of the rows in the layer
	// index, each holding a bit per slot.
	indexMinRows = 1 << 14
	indexMaxRows = 1 << 20

	// indexRowsPerKey is the number of the rows reserved per bloom key of a
	// single layer, yielding ~0.5% false positive rate per slot with 3 bits.
	indexRowsPerKey = 16
)

// indexRowCount returns the number of the rows required for indexing a layer
// with the given number of bloom keys.
func indexRowCount(keys int) int {
	return min(max(1<<bits.Len(uint(keys*indexRowsPerKey)), indexMinRows), indexMaxRows)
}

// indexRows is the transposed bloom filter of the layer index: the bloom filter
// of a slot is a column of the rows, so the slots possibly holding a key are
// resolved by intersecting the 3 rows of the key.
type indexRows struct {
	rows [][4]uint64
}

// positions returns the rows of the given bloom key, picked by the disjoint
// bit ranges of the key.
func (r *indexRows) positions(key uint64) (uint64, uint64, uint64) {
	mask := uint64(len(r.rows) - 1)
	return key & mask, (key >> 21) & mask, (key >> 42) & mask
}

// set marks the key in the column of the given slot, in a rows that isn't
// probed yet.
func (r *indexRows) set(key uint64, word uint64, bit uint64) {
	p0, p1, p2 := r.positions(key)
	r.rows[p0][word] |= bit
	r.rows[p1][word] |= bit
	r.rows[p2][word] |= bit
}

// indexEntry is a layer indexed in a slot.
type indexEntry struct {
	keys [][]uint64
}

// indexTask is a task of the layer indexer: either indexing the layer, or the
// slots at or below the state id of the disk layer being cleared.
type indexTask struct {
	layer *diffLayer
	base  uint64
}

// layerIndex locates the diff layers possibly holding an entry by its bloom key,
// avoiding probing the layers one by one. The layers are indexed by a background
// thread, strictly in the order of creation, thus the layers not indexed yet are
// always on top of the indexed ones in any chain, probed by their own filters.
type layerIndex struct {
	rows  atomic.Pointer[indexRows] // nil if no layer is indexed yet
	queue *taskQueue                // the indexer thread

	// The states below are only accessed by the indexer thread
	slots   [indexSlots][]indexEntry // the layers indexed in each slot
	cleared uint64                   // the state id up to which the slots are cleared
}

// newLayerIndex constructs the index on top of the disk layer with given state id.
func newLayerIndex(base uint64) *layerIndex {
	return &layerIndex{queue: newTaskQueue(), cleared: base}
}

// schedule appends the task to the indexer.
func (l *layerIndex) schedule(task indexTask) {
	l.queue.schedule(func() {
		if task.layer != nil {
			l.insert(task.layer)
		} else {
			l.clear(task.base)
		}
	})
}

// wait blocks until the scheduled tasks are done.
func (l *layerIndex) wait() {
	l.queue.wait()
}

// slotBit returns the word and the bit of the slot hosting the given state id.
func slotBit(id uint64) (uint64, uint64, uint64) {
	slot := id % indexSlots
	return slot, slot / 64, 1 << (slot % 64)
}

// insert indexes the layer, growing the rows if they're too few for the keys of
// the layer. The layer is marked as indexed afterwards.
func (l *layerIndex) insert(dl *diffLayer) {
	// Construct the per-layer bloom filter beforehand
	dl.prepare()

	var (
		items int
		start = time.Now()
	)
	for _, part := range dl.keys {
		items += len(part)
	}
	rows := l.rows.Load()
	if want := indexRowCount(items); rows == nil || len(rows.rows) < want {
		rows = l.grow(want)
	}
	slot, word, bit := slotBit(dl.id)
	for _, part := range dl.keys {
		for _, key := range part {
			p0, p1, p2 := rows.positions(key)
			atomic.OrUint64(&rows.rows[p0][word], bit)
			atomic.OrUint64(&rows.rows[p1][word], bit)
			atomic.OrUint64(&rows.rows[p2][word], bit)
		}
	}
	l.slots[slot] = append(l.slots[slot], indexEntry{keys: dl.keys})
	dl.indexed.Store(l)
	layerIndexTimer.UpdateSince(start)
}

// grow rebuilds the rows with the given number of rows, with all the indexed
// layers reinserted. The readers loading the rows after observing the indexed
// layers will always find them, either in the old rows or the new ones.
func (l *layerIndex) grow(n int) *indexRows {
	rows := &indexRows{
		rows: make([][4]uint64, n),
	}
	for slot := range l.slots {
		_, word, bit := slotBit(uint64(slot))
		for _, entry := range l.slots[slot] {
			for _, part := range entry.keys {
				for _, key := range part {
					rows.set(key, word, bit)
				}
			}
		}
	}
	l.rows.Store(rows)
	return rows
}

// clear clears the slots of the state ids at or below the given disk layer,
// whose layers are all persisted.
func (l *layerIndex) clear(base uint64) {
	rows := l.rows.Load()
	for ; l.cleared < base; l.cleared++ {
		slot, word, bit := slotBit(l.cleared + 1)
		for _, entry := range l.slots[slot] {
			for _, part := range entry.keys {
				for _, key := range part {
					p0, p1, p2 := rows.positions(key)
					atomic.AndUint64(&rows.rows[p0][word], ^bit)
					atomic.AndUint64(&rows.rows[p1][word], ^bit)
					atomic.AndUint64(&rows.rows[p2][word], ^bit)
				}
			}
		}
		l.slots[slot] = nil
	}
}

// candidates returns the bitmap of the layers possibly holding the given key,
// ordered by the distance from the given state id.
func (l *layerIndex) candidates(key uint64, id uint64) (c [4]uint64) {
	rows := l.rows.Load()
	if rows == nil {
		return c // no layer is indexed yet
	}
	// A slot is a candidate only if its bits in all the 3 rows of the key are set.
	// The rows are loaded atomically as the indexer mutates them concurrently.
	p0, p1, p2 := rows.positions(key)
	for i := range c {
		c[i] = atomic.LoadUint64(&rows.rows[p0][i]) &
			atomic.LoadUint64(&rows.rows[p1][i]) &
			atomic.LoadUint64(&rows.rows[p2][i])
	}
	// Rotate the slot of the state id to the highest bit
	return rotateSlots(c, indexSlots-1-id%indexSlots)
}

// rotateSlots rotates the bitmap of the slots to the left by n bits, moving the
// bit s to the bit (s+n)%indexSlots.
func rotateSlots(c [4]uint64, n uint64) (r [4]uint64) {
	words, shift := n/64, n%64
	for i := uint64(0); i < 4; i++ {
		// The word is shifted toward the higher bits, filled by the top bits of
		// the preceding word (none if the shift is 0, as the right shift by 64
		// yields 0).
		r[(i+words)%4] = c[i]<<shift | c[(i+3)%4]>>(64-shift)
	}
	return r
}

// nextCandidate pops the nearest candidate from the bitmap returned by
// candidates, returning its distance from the state id.
func nextCandidate(c *[4]uint64) (uint64, bool) {
	for w := 3; w >= 0; w-- {
		if c[w] == 0 {
			continue
		}
		b := 63 - bits.LeadingZeros64(c[w]) // the highest bit is the nearest
		c[w] &^= 1 << b
		return uint64(indexSlots - 1 - (w*64 + b)), true
	}
	return 0, false
}
