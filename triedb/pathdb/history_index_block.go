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
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	indexBlockDescSize   = 14        // The size of index block descriptor
	indexBlockEntriesCap = 4096      // The maximum number of entries can be grouped in a block
	indexBlockRestartLen = 256       // The restart interval length of index block
	historyIndexBatch    = 1_000_000 // The number of state history indexes for constructing or deleting as batch
)

// indexBlockDesc represents a descriptor for an index block, which contains a
// list of state mutation records associated with a specific state (either an
// account or a storage slot).
type indexBlockDesc struct {
	max     uint64 // The maximum state ID retained within the block
	entries uint16 // The number of state mutation records retained within the block
	id      uint32 // The id of the index block
}

func newIndexBlockDesc(id uint32) *indexBlockDesc {
	return &indexBlockDesc{id: id}
}

// empty indicates whether the block is empty with no element retained.
func (d *indexBlockDesc) empty() bool {
	return d.entries == 0
}

// full indicates whether the number of elements in the block exceeds the
// preconfigured limit.
func (d *indexBlockDesc) full() bool {
	return d.entries >= indexBlockEntriesCap
}

// encode packs index block descriptor into byte stream.
func (d *indexBlockDesc) encode() []byte {
	var buf [indexBlockDescSize]byte
	binary.BigEndian.PutUint64(buf[0:8], d.max)
	binary.BigEndian.PutUint16(buf[8:10], d.entries)
	binary.BigEndian.PutUint32(buf[10:14], d.id)
	return buf[:]
}

// decode unpacks index block descriptor from byte stream.
func (d *indexBlockDesc) decode(blob []byte) {
	d.max = binary.BigEndian.Uint64(blob[:8])
	d.entries = binary.BigEndian.Uint16(blob[8:10])
	d.id = binary.BigEndian.Uint32(blob[10:14])
}

// parseIndexBlock parses the index block with the supplied byte stream.
// The index block format can be illustrated as below:
//
//			+---->+------------------+
//			|     |      Chunk1      |
//			|     +------------------+
//			|     |      ......      |
//			| +-->+------------------+
//			| |   |      ChunkN      |
//			| |   +------------------+
//			+-|---|     Restart1     |
//			  |   |     Restart...   |   2N bytes
//			  +---|     RestartN     |
//			      +------------------+
//			      |  Restart count   |   1 byte
//			      +------------------+
//
//	  - Chunk list: A list of data chunks
//	  - Restart list: A list of 2-byte pointers, each pointing to the start position of a chunk
//	  - Restart count: The number of restarts in the block, stored at the end of the block (1 byte)
//
// Note: the pointer is encoded as a uint16, which is sufficient within a chunk.
// A uint16 can cover offsets in the range [0, 65536), which is more than enough
// to store 4096 integers.
//
// Each chunk begins with the full value of the first integer, followed by
// subsequent integers representing the differences between the current value
// and the preceding one. Integers are encoded with variable-size for best
// storage efficiency. Each chunk can be illustrated as below.
//
//		  Restart ---> +----------------+
//	                   |  Full integer  |
//		               +----------------+
//		               | Diff with prev |
//		               +----------------+
//		               |      ...       |
//		               +----------------+
//		               | Diff with prev |
//		               +----------------+
//
// Empty index block is regarded as invalid.
func parseIndexBlock(blob []byte) ([]uint16, []byte, error) {
	if len(blob) < 1 {
		return nil, nil, fmt.Errorf("corrupted index block, len: %d", len(blob))
	}
	restartLen := blob[len(blob)-1]
	if restartLen == 0 {
		return nil, nil, errors.New("corrupted index block, no restart")
	}
	tailLen := int(restartLen)*2 + 1
	if len(blob) < tailLen {
		return nil, nil, fmt.Errorf("truncated restarts, size: %d, restarts: %d", len(blob), restartLen)
	}
	restarts := make([]uint16, 0, restartLen)
	for i := int(restartLen); i > 0; i-- {
		restart := binary.BigEndian.Uint16(blob[len(blob)-1-2*i:])
		restarts = append(restarts, restart)
	}
	// Validate that restart points are strictly ordered and within the valid
	// data range.
	var prev uint16
	for i := 0; i < len(restarts); i++ {
		if i != 0 {
			if restarts[i] <= prev {
				return nil, nil, fmt.Errorf("restart out of order, prev: %d, next: %d", prev, restarts[i])
			}
		}
		if int(restarts[i]) >= len(blob)-tailLen {
			return nil, nil, fmt.Errorf("invalid restart position, restart: %d, size: %d", restarts[i], len(blob)-tailLen)
		}
		prev = restarts[i]
	}
	return restarts, blob[:len(blob)-tailLen], nil
}

// blockReader is the reader to access the element within a block.
type blockReader struct {
	restarts []uint16
	data     []byte
}

// newBlockReader constructs the block reader with the supplied block data.
func newBlockReader(blob []byte) (*blockReader, error) {
	restarts, data, err := parseIndexBlock(blob)
	if err != nil {
		return nil, err
	}
	return &blockReader{
		restarts: restarts,
		data:     data, // safe to own the slice
	}, nil
}

// readGreaterThan locates the first element in the block that is greater than
// the specified value. If no such element is found, MaxUint64 is returned.
func (br *blockReader) readGreaterThan(id uint64) (uint64, error) {
	var err error
	index := sort.Search(len(br.restarts), func(i int) bool {
		item, n := binary.Uvarint(br.data[br.restarts[i]:])
		if n <= 0 {
			err = fmt.Errorf("failed to decode item at restart %d", br.restarts[i])
		}
		return item > id
	})
	if err != nil {
		return 0, err
	}
	if index == 0 {
		item, _ := binary.Uvarint(br.data[br.restarts[0]:])
		return item, nil
	}
	var (
		start  int
		limit  int
		result uint64
	)
	if index == len(br.restarts) {
		// The element being searched falls within the last restart section,
		// there is no guarantee such element can be found.
		start = int(br.restarts[len(br.restarts)-1])
		limit = len(br.data)
	} else {
		// The element being searched falls within the non-last restart section,
		// such element can be found for sure.
		start = int(br.restarts[index-1])
		limit = int(br.restarts[index])
	}
	pos := start
	for pos < limit {
		x, n := binary.Uvarint(br.data[pos:])
		if pos == start {
			result = x
		} else {
			result += x
		}
		if result > id {
			return result, nil
		}
		pos += n
	}
	// The element which is greater than specified id is not found.
	if index == len(br.restarts) {
		return math.MaxUint64, nil
	}
	// The element which is the first one greater than the specified id
	// is exactly the one located at the restart point.
	item, _ := binary.Uvarint(br.data[br.restarts[index]:])
	return item, nil
}

type blockWriter struct {
	desc     *indexBlockDesc // Descriptor of the block
	restarts []uint16        // Offsets into the data slice, marking the start of each section
	scratch  []byte          // Buffer used for encoding full integers or value differences
	data     []byte          // Aggregated encoded data slice
}

func newBlockWriter(blob []byte, desc *indexBlockDesc) (*blockWriter, error) {
	scratch := make([]byte, binary.MaxVarintLen64)
	if len(blob) == 0 {
		return &blockWriter{
			desc:    desc,
			scratch: scratch,
			data:    make([]byte, 0, 1024),
		}, nil
	}
	restarts, data, err := parseIndexBlock(blob)
	if err != nil {
		return nil, err
	}
	return &blockWriter{
		desc:     desc,
		restarts: restarts,
		scratch:  scratch,
		data:     data, // safe to own the slice
	}, nil
}

// append adds a new element to the block. The new element must be greater than
// the previous one. The provided ID is assumed to always be greater than 0.
func (b *blockWriter) append(id uint64) error {
	if id == 0 {
		return errors.New("invalid zero id")
	}
	if id <= b.desc.max {
		return fmt.Errorf("append element out of order, last: %d, this: %d", b.desc.max, id)
	}
	// Rotate the current restart section if it's full
	if b.desc.entries%indexBlockRestartLen == 0 {
		// Save the offset within the data slice as the restart point
		// for the next section.
		b.restarts = append(b.restarts, uint16(len(b.data)))

		// The restart point item can either be encoded in variable
		// size or fixed size. Although variable-size encoding is
		// slightly slower (2ns per operation), it is still relatively
		// fast, therefore, it's picked for better space efficiency.
		//
		// The first element in a restart range is encoded using its
		// full value.
		n := binary.PutUvarint(b.scratch[0:], id)
		b.data = append(b.data, b.scratch[:n]...)
	} else {
		// The current section is not full, append the element.
		// The element which is not the first one in the section
		// is encoded using the value difference from the preceding
		// element.
		n := binary.PutUvarint(b.scratch[0:], id-b.desc.max)
		b.data = append(b.data, b.scratch[:n]...)
	}
	b.desc.entries++

	// The state history ID must be greater than 0.
	//if b.desc.min == 0 {
	//	b.desc.min = id
	//}
	b.desc.max = id
	return nil
}

// scanSection traverses the specified section and terminates if fn returns true.
func (b *blockWriter) scanSection(section int, fn func(uint64, int) bool) {
	var (
		value uint64
		start = int(b.restarts[section])
		pos   = start
		limit int
	)
	if section == len(b.restarts)-1 {
		limit = len(b.data)
	} else {
		limit = int(b.restarts[section+1])
	}
	for pos < limit {
		x, n := binary.Uvarint(b.data[pos:])
		if pos == start {
			value = x
		} else {
			value += x
		}
		if fn(value, pos) {
			return
		}
		pos += n
	}
}

// sectionLast returns the last element in the specified section.
func (b *blockWriter) sectionLast(section int) uint64 {
	var n uint64
	b.scanSection(section, func(v uint64, _ int) bool {
		n = v
		return false
	})
	return n
}

// sectionSearch looks up the specified value in the given section,
// the position and the preceding value will be returned if found.
func (b *blockWriter) sectionSearch(section int, n uint64) (found bool, prev uint64, pos int) {
	b.scanSection(section, func(v uint64, p int) bool {
		if n == v {
			pos = p
			found = true
			return true // terminate iteration
		}
		prev = v
		return false // continue iteration
	})
	return found, prev, pos
}

// pop removes the last element from the block. The assumption is held that block
// writer must be non-empty.
func (b *blockWriter) pop(id uint64) error {
	if id == 0 {
		return errors.New("invalid zero id")
	}
	if id != b.desc.max {
		return fmt.Errorf("pop element out of order, last: %d, this: %d", b.desc.max, id)
	}
	// If there is only one entry left, the entire block should be reset
	if b.desc.entries == 1 {
		//b.desc.min = 0
		b.desc.max = 0
		b.desc.entries = 0
		b.restarts = nil
		b.data = b.data[:0]
		return nil
	}
	// Pop the last restart section if the section becomes empty after removing
	// one element.
	if b.desc.entries%indexBlockRestartLen == 1 {
		b.data = b.data[:b.restarts[len(b.restarts)-1]]
		b.restarts = b.restarts[:len(b.restarts)-1]
		b.desc.max = b.sectionLast(len(b.restarts) - 1)
		b.desc.entries -= 1
		return nil
	}
	// Look up the element preceding the one to be popped, in order to update
	// the maximum element in the block.
	found, prev, pos := b.sectionSearch(len(b.restarts)-1, id)
	if !found {
		return fmt.Errorf("pop element is not found, last: %d, this: %d", b.desc.max, id)
	}
	b.desc.max = prev
	b.data = b.data[:pos]
	b.desc.entries -= 1
	return nil
}

func (b *blockWriter) empty() bool {
	return b.desc.empty()
}

func (b *blockWriter) full() bool {
	return b.desc.full()
}

// finish finalizes the index block encoding by appending the encoded restart points
// and the restart counter to the end of the block.
//
// This function is safe to be called multiple times.
func (b *blockWriter) finish() []byte {
	var buf []byte
	for _, number := range b.restarts {
		binary.BigEndian.PutUint16(b.scratch[:2], number)
		buf = append(buf, b.scratch[:2]...)
	}
	buf = append(buf, byte(len(b.restarts)))
	return append(b.data, buf...)
}
