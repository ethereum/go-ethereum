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

// blockIterator is the iterator to traverse the indices within a single block.
type blockIterator struct {
	// immutable fields
	data     []byte   // Reference to the data segment within the block reader
	restarts []uint16 // Offsets pointing to the restart sections within the data

	// mutable fields
	id         uint64 // ID of the element at the iterators current position
	dataPtr    int    // Current read position within the data slice
	restartPtr int    // Index of the restart section where the iterator is currently positioned
	exhausted  bool   // Flag whether the iterator has been exhausted
	err        error  // Accumulated error during the traversal
}

func newBlockIterator(data []byte, restarts []uint16) *blockIterator {
	it := &blockIterator{
		data:     data,     // hold the slice directly with no deep copy
		restarts: restarts, // hold the slice directly with no deep copy
	}
	it.reset()
	return it
}

func (it *blockIterator) set(dataPtr int, restartPtr int, id uint64) {
	it.id = id
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
	it.dataPtr = -1
	it.restartPtr = -1
	it.exhausted = false
	it.err = nil

	// Mark the iterator as exhausted if the associated index block is empty
	if len(it.data) == 0 || len(it.restarts) == 0 {
		it.exhausted = true
	}
}

// SeekGT moves the iterator to the first element whose id is greater than the
// given number. It returns whether such element exists.
//
// Note, this operation will unset the exhausted status and subsequent traversal
// is allowed.
func (it *blockIterator) SeekGT(id uint64) bool {
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
		item, n := binary.Uvarint(it.data[it.restarts[0]:])

		// If the restart size is 1, then the restart pointer shouldn't be 0.
		// It's not practical and should be denied in the first place.
		it.set(int(it.restarts[0])+n, 0, item)
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

		if result > id {
			if pos == limit {
				it.set(pos, restartIndex+1, result)
			} else {
				it.set(pos, restartIndex, result)
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
	item, n := binary.Uvarint(it.data[it.restarts[index]:])
	it.set(int(it.restarts[index])+n, index, item)
	return true
}

func (it *blockIterator) init() {
	if it.dataPtr != -1 {
		return
	}
	it.dataPtr = 0
	it.restartPtr = 0
}

// Next implements the HistoryIndexIterator, moving the iterator to the next
// element. If the iterator has been exhausted, and boolean with false should
// be returned.
func (it *blockIterator) Next() bool {
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

	// Move to the next restart section if the data pointer crosses the boundary
	nextRestartPtr := it.restartPtr
	if it.restartPtr < len(it.restarts)-1 && it.dataPtr+n == int(it.restarts[it.restartPtr+1]) {
		nextRestartPtr = it.restartPtr + 1
	}
	it.set(it.dataPtr+n, nextRestartPtr, val)

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

// blockLoader defines the method to retrieve the specific block for reading.
type blockLoader func(id uint32) (*blockReader, error)

// indexIterator is an iterator to traverse the history indices belonging to the
// specific state entry.
type indexIterator struct {
	// immutable fields
	descList []*indexBlockDesc
	loader   blockLoader

	// mutable fields
	blockIt   *blockIterator
	blockPtr  int
	exhausted bool
	err       error
}

func newIndexIterator(descList []*indexBlockDesc, loader blockLoader) *indexIterator {
	it := &indexIterator{
		descList: descList,
		loader:   loader,
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
	id := it.descList[blockPtr].id
	br, err := it.loader(id)
	if err != nil {
		return err
	}
	it.blockIt = newBlockIterator(br.data, br.restarts)
	it.blockPtr = blockPtr
	return nil
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
	return it.blockIt.SeekGT(id)
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
	if it.blockPtr == len(it.descList)-1 {
		it.exhausted = true
		return false
	}
	if err := it.open(it.blockPtr + 1); err != nil {
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
