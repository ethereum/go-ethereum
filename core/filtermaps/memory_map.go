// Copyright 2025 The go-ethereum Authors
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

package filtermaps

import (
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// memoryMap is an in-memory representation of a filter map that represents rows
// as linked lists and stores entries in a single slice.
// memoryMap allows adding new elements and is used for rendering maps. Completed
// maps can be transformed to an immutable finishedMap.
type memoryMap struct {
	entries   []mmEntry // size = valuesPerMap
	rows      []mmRow   // size = mapHeight
	nextEntry uint32
	blockPtrs []uint64
	lastBlock lastBlockOfMap
}

// mmEntry is a linked list entry. The field next points to the next entry in the
// entries slice. Zero singals the end of the list (entries[0] is always the first
// entry of a list).
type mmEntry struct {
	value, next uint32
}

// mmRow stores the index of the first and last entry of a linked list  in the
// entries slice. If length is zero then first and last are not initialized.
// Note that length could be determined by traversing the linked list but it is
// explicitly stored because map rendering needs quick access to row length.
type mmRow struct {
	first, last, length uint32
}

// newMemoryMap creates an empty memoryMap.
func (p *Params) newMemoryMap() *memoryMap {
	return &memoryMap{
		entries: make([]mmEntry, p.valuesPerMap),
		rows:    make([]mmRow, p.mapHeight),
	}
}

// clone creates a copy of the map.
func (m *memoryMap) clone() *memoryMap {
	return &memoryMap{
		//TODO no need to clone entries if one of the maps will never be changed further.
		entries:   slices.Clone(m.entries),
		rows:      slices.Clone(m.rows),
		nextEntry: m.nextEntry,
		blockPtrs: slices.Clone(m.blockPtrs),
		lastBlock: m.lastBlock,
	}
}

func (m *memoryMap) firstBlock() uint64 {
	return m.lastBlock.number + 1 - uint64(len(m.blockPtrs))
}

func (m *memoryMap) blocks() common.Range[uint64] {
	l := uint64(len(m.blockPtrs))
	return common.NewRange[uint64](m.lastBlock.number+1-l, l)
}

// addToRow adds a new entry to the specified row.
func (m *memoryMap) addToRow(rowIndex, value uint32) {
	row := &m.rows[rowIndex]
	if row.length == 0 {
		row.first = m.nextEntry
	} else {
		m.entries[row.last].next = m.nextEntry
	}
	row.last = m.nextEntry
	row.length++
	m.entries[m.nextEntry].value = value
	m.nextEntry++
}

// rowLength returns the length of a row.
func (m *memoryMap) rowLength(rowIndex uint32) uint32 {
	return m.rows[rowIndex].length
}

// getRow returns a row of the map, truncated if maxLen is smaller than the actual
// row length.
func (m *memoryMap) getRow(rowIndex, maxLen uint32) FilterRow {
	row := m.rows[rowIndex]
	length := min(row.length, maxLen)
	res := make(FilterRow, length)
	next := row.first
	for i := range length {
		entry := m.entries[next]
		res[i], next = entry.value, entry.next
	}
	return res
}

// finishedMap is an immutable memory representation of a single filter map.
// It is more compact and allows more efficient row lookup than memoryMap.
// Note that it assumes params.mapHeight <= 2**16 which is checked in deriveFields.
type finishedMap struct {
	rowPtrs   []uint16 // points to rowData index after end of row; 2**16 can wrap around to 0
	rowData   []uint32
	blockPtrs []uint64
	lastBlock lastBlockOfMap
}

// finished creates a new finishedMap from a memoryMap.
func (m *memoryMap) finished() *finishedMap {
	fm := &finishedMap{
		rowPtrs:   make([]uint16, len(m.rows)),
		rowData:   make([]uint32, m.nextEntry),
		blockPtrs: slices.Clone(m.blockPtrs),
		lastBlock: m.lastBlock,
	}
	var ptr uint16
	for i, row := range m.rows {
		next := row.first
		for range row.length {
			entry := m.entries[next]
			fm.rowData[ptr] = entry.value
			next = entry.next
			ptr++
		}
		fm.rowPtrs[i] = ptr
	}
	return fm
}

func (fm *finishedMap) firstBlock() uint64 {
	return fm.lastBlock.number + 1 - uint64(len(fm.blockPtrs))
}

func (fm *finishedMap) blocks() common.Range[uint64] {
	l := uint64(len(fm.blockPtrs))
	return common.NewRange[uint64](fm.lastBlock.number+1-l, l)
}

// getRow returns a row of the map, truncated if maxLen is smaller than the actual
// row length. If the row has zero length then it returns a zero length slice.
func (fm *finishedMap) getRow(rowIndex, maxLen uint32) FilterRow {
	var start uint16
	if rowIndex > 0 {
		start = fm.rowPtrs[rowIndex-1]
	}
	// Note: uint16 subtraction ensures correct result if
	// rowPtrs[rowIndex] has wrapped around to zero. For example if there are
	// exactly 2**16 entries and last row has a length of 5 then
	// rowPtr[0xFFFE] == 0xFFFB and rowPtr[0xFFFF] == 0x0000 and the uint16
	// subtraction is 0x0000-0xFFFB == 0x0005.
	// Also note that it is guaranteed by the filter map design that no single
	// row can have a length of 0x10000.
	length := fm.rowPtrs[rowIndex] - start
	return FilterRow(fm.rowData[start : uint32(start)+min(maxLen, uint32(length))])
}
