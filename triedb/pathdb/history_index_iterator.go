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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// HistoryIndexIterator is an iterator to traverse the history indices.
type HistoryIndexIterator interface {
	// SeekGT moves the iterator to the first element whose id is greater than
	// the given number. It returns whether such element exists.
	SeekGT(id uint64) bool

	// Next moves the iterator to the next element. If the iterator has been
	// exhausted, and boolean with false should be returned.
	Next() bool

	// ID returns the id of the element where the iterator is positioned at.
	ID() uint64

	// Error returns any accumulated error. Exhausting all the elements is not
	// considered to be an error.
	Error() error
}

// extFilter provides utilities for filtering index entries based on their
// extension field.
//
// It supports two primary operations:
//
//   - determine whether a given target node ID or any of its descendants
//     appears explicitly in the extension list.
//
//   - determine whether a given target node ID or any of its descendants
//     is marked in the extension bitmap.
//
// Together, these checks allow callers to efficiently filter out the irrelevant
// index entries during the lookup.
type extFilter uint16

// exists takes the entire extension field in the index block and determines
// whether the target ID or its descendants appears. Note, any of descendant
// can implicitly mean the presence of ancestor.
func (f extFilter) exists(ext []byte) (bool, error) {
	fn := uint16(f)
	list, err := decodeIDs(ext)
	if err != nil {
		return false, err
	}
	for _, elem := range list {
		if elem == fn {
			return true, nil
		}
		if isAncestor(fn, elem) {
			return true, nil
		}
	}
	return false, nil
}

const (
	// bitmapBytesTwoLevels is the size of the bitmap for two levels of the
	// 16-ary tree (16 nodes total, excluding the root).
	bitmapBytesTwoLevels = 2

	// bitmapBytesThreeLevels is the size of the bitmap for three levels of
	// the 16-ary tree (272 nodes total, excluding the root).
	bitmapBytesThreeLevels = 34

	// bitmapElementThresholdTwoLevels is the total number of elements in the
	// two levels of a 16-ary tree (16 nodes total, excluding the root).
	bitmapElementThresholdTwoLevels = 16

	// bitmapElementThresholdThreeLevels is the total number of elements in the
	// two levels of a 16-ary tree (16 nodes total, excluding the root).
	bitmapElementThresholdThreeLevels = bitmapElementThresholdTwoLevels + 16*16
)

// contains takes the bitmap from the block metadata and determines whether the
// target ID or its descendants is marked in the bitmap. Note, any of descendant
// can implicitly mean the presence of ancestor.
func (f extFilter) contains(bitmap []byte) (bool, error) {
	id := int(f)
	if id == 0 {
		return true, nil
	}
	n := id - 1 // apply the position shift for excluding root node

	switch len(bitmap) {
	case 0:
		// Bitmap is not available, return "false positive"
		return true, nil
	case bitmapBytesTwoLevels:
		// Bitmap for 2-level trie with at most 16 elements inside
		if n >= bitmapElementThresholdTwoLevels {
			return false, fmt.Errorf("invalid extension filter %d for 2 bytes bitmap", id)
		}
		return isBitSet(bitmap, n), nil
	case bitmapBytesThreeLevels:
		// Bitmap for 3-level trie with at most 16+16*16 elements inside
		if n >= bitmapElementThresholdThreeLevels {
			return false, fmt.Errorf("invalid extension filter %d for 34 bytes bitmap", id)
		} else if n >= bitmapElementThresholdTwoLevels {
			return isBitSet(bitmap, n), nil
		} else {
			// Check the element itself first
			if isBitSet(bitmap, n) {
				return true, nil
			}
			// Check descendants: the presence of any descendant implicitly
			// represents a mutation of its ancestor.
			return bitmap[2+2*n] != 0 || bitmap[3+2*n] != 0, nil
		}
	default:
		return false, fmt.Errorf("unsupported bitmap size %d", len(bitmap))
	}
}

// blockIterator is the iterator to traverse the indices within a single block.
type blockIterator struct {
	// immutable fields
	data     []byte   // Reference to the data segment within the block reader
	restarts []uint16 // Offsets pointing to the restart sections within the data
	hasExt   bool     // Flag whether the extension is included in the data

	// Optional extension filter
	filter *extFilter // Filters index entries based on the extension field.

	// mutable fields
	id         uint64 // ID of the element at the iterators current position
	ext        []byte // Extension field of the element at the iterators current position
	dataPtr    int    // Current read position within the data slice
	restartPtr int    // Index of the restart section where the iterator is currently positioned
	exhausted  bool   // Flag whether the iterator has been exhausted
	err        error  // Accumulated error during the traversal
}

func (br *blockReader) newIterator(filter *extFilter) *blockIterator {
	it := &blockIterator{
		data:     br.data,     // hold the slice directly with no deep copy
		restarts: br.restarts, // hold the slice directly with no deep copy
		hasExt:   br.hasExt,   // flag whether the extension should be resolved
		filter:   filter,      // optional extension filter
	}
	it.reset()
	return it
}

func (it *blockIterator) set(dataPtr int, restartPtr int, id uint64, ext []byte) {
	it.id = id
	it.ext = ext

	it.dataPtr = dataPtr
	it.restartPtr = restartPtr
	it.exhausted = dataPtr == len(it.data)
}

func (it *blockIterator) setErr(err error) {
	if it.err != nil {
		return
	}
	it.err = err
}

func (it *blockIterator) reset() {
	it.id = 0
	it.ext = nil

	it.dataPtr = -1
	it.restartPtr = -1
	it.exhausted = false
	it.err = nil

	// Mark the iterator as exhausted if the associated index block is empty
	if len(it.data) == 0 || len(it.restarts) == 0 {
		it.exhausted = true
	}
}

func (it *blockIterator) resolveExt(pos int) ([]byte, int, error) {
	if !it.hasExt {
		return nil, 0, nil
	}
	length, n := binary.Uvarint(it.data[pos:])
	if n <= 0 {
		return nil, 0, fmt.Errorf("too short for extension, pos: %d, datalen: %d", pos, len(it.data))
	}
	if len(it.data[pos+n:]) < int(length) {
		return nil, 0, fmt.Errorf("too short for extension, pos: %d, length: %d, datalen: %d", pos, length, len(it.data))
	}
	return it.data[pos+n : pos+n+int(length)], n + int(length), nil
}

// seekGT moves the iterator to the first element whose id is greater than the
// given number. It returns whether such element exists.
//
// Note, this operation will unset the exhausted status and subsequent traversal
// is allowed.
func (it *blockIterator) seekGT(id uint64) bool {
	if it.err != nil {
		return false
	}
	var err error
	index := sort.Search(len(it.restarts), func(i int) bool {
		item, n := binary.Uvarint(it.data[it.restarts[i]:])
		if n <= 0 {
			err = fmt.Errorf("failed to decode item at restart %d", it.restarts[i])
		}
		return item > id
	})
	if err != nil {
		it.setErr(err)
		return false
	}
	if index == 0 {
		pos := int(it.restarts[0])
		item, n := binary.Uvarint(it.data[pos:])
		if n <= 0 {
			it.setErr(fmt.Errorf("failed to decode item at pos %d", it.restarts[0]))
			return false
		}
		pos = pos + n

		ext, shift, err := it.resolveExt(pos)
		if err != nil {
			it.setErr(err)
			return false
		}
		it.set(pos+shift, 0, item, ext)
		return true
	}
	var (
		start        int
		limit        int
		restartIndex int // The restart section being searched below
	)
	if index == len(it.restarts) {
		// The element being searched falls within the last restart section,
		// there is no guarantee such element can be found.
		start = int(it.restarts[len(it.restarts)-1])
		limit = len(it.data)
		restartIndex = len(it.restarts) - 1
	} else {
		// The element being searched falls within the non-last restart section,
		// such element can be found for sure.
		start = int(it.restarts[index-1])
		limit = int(it.restarts[index])
		restartIndex = index - 1
	}
	var (
		result uint64
		pos    = start
	)
	for pos < limit {
		x, n := binary.Uvarint(it.data[pos:])
		if n <= 0 {
			it.setErr(fmt.Errorf("failed to decode item at pos %d", pos))
			return false
		}
		if pos == start {
			result = x
		} else {
			result += x
		}
		pos += n

		ext, shift, err := it.resolveExt(pos)
		if err != nil {
			it.setErr(err)
			return false
		}
		pos += shift

		if result > id {
			if pos == limit {
				it.set(pos, restartIndex+1, result, ext)
			} else {
				it.set(pos, restartIndex, result, ext)
			}
			return true
		}
	}
	// The element which is greater than specified id is not found.
	if index == len(it.restarts) {
		it.reset()
		return false
	}
	// The element which is the first one greater than the specified id
	// is exactly the one located at the restart point.
	pos = int(it.restarts[index])
	item, n := binary.Uvarint(it.data[pos:])
	if n <= 0 {
		it.setErr(fmt.Errorf("failed to decode item at pos %d", it.restarts[index]))
		return false
	}
	pos = pos + n

	ext, shift, err := it.resolveExt(pos)
	if err != nil {
		it.setErr(err)
		return false
	}
	it.set(pos+shift, index, item, ext)
	return true
}

// SeekGT implements HistoryIndexIterator, is the wrapper of the seekGT with
// optional extension filter logic applied.
func (it *blockIterator) SeekGT(id uint64) bool {
	if !it.seekGT(id) {
		return false
	}
	if it.filter == nil {
		return true
	}
	for {
		found, err := it.filter.exists(it.ext)
		if err != nil {
			it.setErr(err)
			return false
		}
		if found {
			break
		}
		if !it.next() {
			return false
		}
	}
	return true
}

func (it *blockIterator) init() {
	if it.dataPtr != -1 {
		return
	}
	it.dataPtr = 0
	it.restartPtr = 0
}

// next moves the iterator to the next element. If the iterator has been exhausted,
// and boolean with false should be returned.
func (it *blockIterator) next() bool {
	if it.exhausted || it.err != nil {
		return false
	}
	it.init()

	// Decode the next element pointed by the iterator
	v, n := binary.Uvarint(it.data[it.dataPtr:])
	if n <= 0 {
		it.setErr(fmt.Errorf("failed to decode item at pos %d", it.dataPtr))
		return false
	}
	var val uint64
	if it.dataPtr == int(it.restarts[it.restartPtr]) {
		val = v
	} else {
		val = it.id + v
	}

	// Decode the extension field
	ext, shift, err := it.resolveExt(it.dataPtr + n)
	if err != nil {
		it.setErr(err)
		return false
	}

	// Move to the next restart section if the data pointer crosses the boundary
	nextRestartPtr := it.restartPtr
	if it.restartPtr < len(it.restarts)-1 && it.dataPtr+n+shift == int(it.restarts[it.restartPtr+1]) {
		nextRestartPtr = it.restartPtr + 1
	}
	it.set(it.dataPtr+n+shift, nextRestartPtr, val, ext)

	return true
}

// Next implements the HistoryIndexIterator, moving the iterator to the next
// element. It's a wrapper of next with optional extension filter logic applied.
func (it *blockIterator) Next() bool {
	if !it.next() {
		return false
	}
	if it.filter == nil {
		return true
	}
	for {
		found, err := it.filter.exists(it.ext)
		if err != nil {
			it.setErr(err)
			return false
		}
		if found {
			break
		}
		if !it.next() {
			return false
		}
	}
	return true
}

// ID implements HistoryIndexIterator, returning the id of the element where the
// iterator is positioned at.
func (it *blockIterator) ID() uint64 {
	return it.id
}

// Error implements HistoryIndexIterator, returning any accumulated error.
// Exhausting all the elements is not considered to be an error.
func (it *blockIterator) Error() error { return it.err }

// indexIterator is an iterator to traverse the history indices belonging to the
// specific state entry.
type indexIterator struct {
	// immutable fields
	descList []*indexBlockDesc
	reader   *indexReader

	// Optional extension filter
	filter *extFilter

	// mutable fields
	blockIt   *blockIterator
	blockPtr  int
	exhausted bool
	err       error
}

// newBlockIter initializes the block iterator with the specified block ID.
func (r *indexReader) newBlockIter(id uint32, filter *extFilter) (*blockIterator, error) {
	br, ok := r.readers[id]
	if !ok {
		var err error
		br, err = newBlockReader(readStateIndexBlock(r.state, r.db, id), r.bitmapSize != 0)
		if err != nil {
			return nil, err
		}
		r.readers[id] = br
	}
	return br.newIterator(filter), nil
}

// newIterator initializes the index iterator with the specified extension filter.
func (r *indexReader) newIterator(filter *extFilter) *indexIterator {
	it := &indexIterator{
		descList: r.descList,
		reader:   r,
		filter:   filter,
	}
	it.reset()
	return it
}

func (it *indexIterator) setErr(err error) {
	if it.err != nil {
		return
	}
	it.err = err
}

func (it *indexIterator) reset() {
	it.blockIt = nil
	it.blockPtr = -1
	it.exhausted = false
	it.err = nil

	if len(it.descList) == 0 {
		it.exhausted = true
	}
}

func (it *indexIterator) open(blockPtr int) error {
	blockIt, err := it.reader.newBlockIter(it.descList[blockPtr].id, it.filter)
	if err != nil {
		return err
	}
	it.blockIt = blockIt
	it.blockPtr = blockPtr
	return nil
}

func (it *indexIterator) applyFilter(index int) (int, error) {
	if it.filter == nil {
		return index, nil
	}
	for index < len(it.descList) {
		found, err := it.filter.contains(it.descList[index].extBitmap)
		if err != nil {
			return 0, err
		}
		if found {
			break
		}
		index++
	}
	return index, nil
}

// SeekGT moves the iterator to the first element whose id is greater than the
// given number. It returns whether such element exists.
//
// Note, this operation will unset the exhausted status and subsequent traversal
// is allowed.
func (it *indexIterator) SeekGT(id uint64) bool {
	if it.err != nil {
		return false
	}
	index := sort.Search(len(it.descList), func(i int) bool {
		return id < it.descList[i].max
	})
	index, err := it.applyFilter(index)
	if err != nil {
		it.setErr(err)
		return false
	}
	if index == len(it.descList) {
		return false
	}
	it.exhausted = false

	if it.blockIt == nil || it.blockPtr != index {
		if err := it.open(index); err != nil {
			it.setErr(err)
			return false
		}
	}
	// Terminate if the element which is greater than the id can be found in the
	// last block; otherwise move to the next block. It may happen that all the
	// target elements in this block are all less than id.
	if it.blockIt.SeekGT(id) {
		return true
	}
	return it.Next()
}

func (it *indexIterator) init() error {
	if it.blockIt != nil {
		return nil
	}
	return it.open(0)
}

// Next implements the HistoryIndexIterator, moving the iterator to the next
// element. If the iterator has been exhausted, and boolean with false should
// be returned.
func (it *indexIterator) Next() bool {
	if it.exhausted || it.err != nil {
		return false
	}
	if err := it.init(); err != nil {
		it.setErr(err)
		return false
	}
	if it.blockIt.Next() {
		return true
	}
	it.blockPtr++

	index, err := it.applyFilter(it.blockPtr)
	if err != nil {
		it.setErr(err)
		return false
	}
	it.blockPtr = index

	if it.blockPtr == len(it.descList) {
		it.exhausted = true
		return false
	}
	if err := it.open(it.blockPtr); err != nil {
		it.setErr(err)
		return false
	}
	return it.blockIt.Next()
}

// Error implements HistoryIndexIterator, returning any accumulated error.
// Exhausting all the elements is not considered to be an error.
func (it *indexIterator) Error() error {
	if it.err != nil {
		return it.err
	}
	if it.blockIt != nil {
		return it.blockIt.Error()
	}
	return nil
}

// ID implements HistoryIndexIterator, returning the id of the element where the
// iterator is positioned at.
func (it *indexIterator) ID() uint64 {
	return it.blockIt.ID()
}

// seqIter provides a simple iterator over a contiguous sequence of
// unsigned integers, ending at end (end is included).
type seqIter struct {
	cur  uint64 // current position
	end  uint64 // iteration stops at end-1
	done bool   // true when iteration is exhausted
}

func newSeqIter(last uint64) *seqIter {
	return &seqIter{end: last + 1}
}

// SeekGT positions the iterator at the smallest element > id.
// Returns false if no such element exists.
func (it *seqIter) SeekGT(id uint64) bool {
	if id+1 >= it.end {
		it.done = true
		return false
	}
	it.cur = id + 1
	it.done = false
	return true
}

// Next advances the iterator. Returns false if exhausted.
func (it *seqIter) Next() bool {
	if it.done {
		return false
	}
	if it.cur+1 < it.end {
		it.cur++
		return true
	}
	// this was the last element
	it.done = true
	return false
}

// ID returns the id of the element where the iterator is positioned at.
func (it *seqIter) ID() uint64 { return it.cur }

// Error returns any accumulated error. Exhausting all the elements is not
// considered to be an error.
func (it *seqIter) Error() error { return nil }
