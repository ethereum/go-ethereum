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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/log"
)

const (
	indexBlockDescSize   = 14   // The size of index block descriptor
	indexBlockMaxSize    = 4096 // The maximum size of a single index block
	indexBlockRestartLen = 256  // The restart interval length of index block
)

// indexBlockDesc represents a descriptor for an index block, which contains a
// list of state mutation records associated with a specific state (either an
// account or a storage slot).
type indexBlockDesc struct {
	max       uint64 // The maximum state ID retained within the block
	entries   uint16 // The number of state mutation records retained within the block
	id        uint32 // The id of the index block
	extBitmap []byte // Optional fixed-size bitmap for the included extension elements
}

func newIndexBlockDesc(id uint32, bitmapSize int) *indexBlockDesc {
	var bitmap []byte
	if bitmapSize > 0 {
		bitmap = make([]byte, bitmapSize)
	}
	return &indexBlockDesc{id: id, extBitmap: bitmap}
}

// empty indicates whether the block is empty with no element retained.
func (d *indexBlockDesc) empty() bool {
	return d.entries == 0
}

// encode packs index block descriptor into byte stream.
func (d *indexBlockDesc) encode() []byte {
	buf := make([]byte, indexBlockDescSize+len(d.extBitmap))
	binary.BigEndian.PutUint64(buf[0:8], d.max)
	binary.BigEndian.PutUint16(buf[8:10], d.entries)
	binary.BigEndian.PutUint32(buf[10:14], d.id)
	copy(buf[indexBlockDescSize:], d.extBitmap)
	return buf[:]
}

// decode unpacks index block descriptor from byte stream. It's unsafe to mutate
// the provided byte stream after the function call.
func (d *indexBlockDesc) decode(blob []byte) {
	d.max = binary.BigEndian.Uint64(blob[:8])
	d.entries = binary.BigEndian.Uint16(blob[8:10])
	d.id = binary.BigEndian.Uint32(blob[10:14])
	d.extBitmap = blob[indexBlockDescSize:] // no-deep copy!
}

// copy returns a deep-copied object.
func (d *indexBlockDesc) copy() *indexBlockDesc {
	return &indexBlockDesc{
		max:       d.max,
		entries:   d.entries,
		id:        d.id,
		extBitmap: bytes.Clone(d.extBitmap),
	}
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
// Each chunk begins with a full integer value for the first element, followed
// by subsequent integers encoded as differences (deltas) from their preceding
// values. All integers use variable-length encoding for optimal space efficiency.
//
// In the updated format, each element in the chunk may optionally include an
// "extension" section. If an extension is present, it starts with a var-size
// integer indicating the length of the remaining extension payload, followed by
// that many bytes. If no extension is present, the element format is identical
// to the original version (i.e., only the integer or delta value is encoded).
//
// In the trienode history index, the extension field contains the list of
// trie node IDs that fall within this range. For the given state transition,
// these IDs represent the specific nodes in this range that were mutated.
//
// Whether an element includes an extension is determined by the block reader
// based on the specification. Conceptually, a chunk is structured as:
//
//	Restart ---> +----------------+
//	             |  Full integer  |
//	             +----------------+
//	             |  (Extension?)  |
//	             +----------------+
//	             | Diff with prev |
//	             +----------------+
//	             |  (Extension?)  |
//	             +----------------+
//	             |       ...      |
//	             +----------------+
//	             | Diff with prev |
//	             +----------------+
//	             |  (Extension?)  |
//	             +----------------+
//
// Empty index block is regarded as invalid.
func parseIndexBlock(blob []byte) ([]uint16, []byte, error) {
	if len(blob) < 1 {
		return nil, nil, fmt.Errorf("corrupted index block, len: %d", len(blob))
	}
	restartLen := int(blob[len(blob)-1])
	if restartLen == 0 {
		return nil, nil, errors.New("corrupted index block, no restart")
	}
	tailLen := restartLen*2 + 1
	if len(blob) < tailLen {
		return nil, nil, fmt.Errorf("truncated restarts, size: %d, restarts: %d", len(blob), restartLen)
	}
	restarts := make([]uint16, restartLen)
	dataEnd := len(blob) - tailLen

	// Extract and validate that restart points are strictly ordered and within the valid
	// data range.
	for i := 0; i < restartLen; i++ {
		off := dataEnd + 2*i
		restarts[i] = binary.BigEndian.Uint16(blob[off : off+2])

		if i > 0 && restarts[i] <= restarts[i-1] {
			return nil, nil, fmt.Errorf("restart out of order, prev: %d, next: %d", restarts[i-1], restarts[i])
		}
		if int(restarts[i]) >= dataEnd {
			return nil, nil, fmt.Errorf("invalid restart position, restart: %d, size: %d", restarts[i], dataEnd)
		}
	}
	return restarts, blob[:dataEnd], nil
}

// blockReader is the reader to access the element within a block.
type blockReader struct {
	restarts []uint16
	data     []byte
	hasExt   bool
}

// newBlockReader constructs the block reader with the supplied block data.
func newBlockReader(blob []byte, hasExt bool) (*blockReader, error) {
	restarts, data, err := parseIndexBlock(blob)
	if err != nil {
		return nil, err
	}
	return &blockReader{
		restarts: restarts,
		data:     data,   // safe to own the slice
		hasExt:   hasExt, // flag whether extension should be resolved
	}, nil
}

// readGreaterThan locates the first element in the block that is greater than
// the specified value. If no such element is found, MaxUint64 is returned.
func (br *blockReader) readGreaterThan(id uint64) (uint64, error) {
	it := br.newIterator(nil)
	found := it.SeekGT(id)
	if err := it.Error(); err != nil {
		return 0, err
	}
	if !found {
		return math.MaxUint64, nil
	}
	return it.ID(), nil
}

type blockWriter struct {
	desc     *indexBlockDesc // Descriptor of the block
	restarts []uint16        // Offsets into the data slice, marking the start of each section
	data     []byte          // Aggregated encoded data slice
	hasExt   bool            // Flag whether the extension field for each element exists
}

// newBlockWriter constructs a block writer. In addition to the existing data
// and block description, it takes an element ID and prunes all existing elements
// above that ID. It's essential as the recovery mechanism after unclean shutdown
// during the history indexing.
func newBlockWriter(blob []byte, desc *indexBlockDesc, limit uint64, hasExt bool) (*blockWriter, error) {
	if len(blob) == 0 {
		return &blockWriter{
			desc:   desc,
			data:   make([]byte, 0, 1024),
			hasExt: hasExt,
		}, nil
	}
	restarts, data, err := parseIndexBlock(blob)
	if err != nil {
		return nil, err
	}
	writer := &blockWriter{
		desc:     desc,
		restarts: restarts,
		data:     data, // safe to own the slice
		hasExt:   hasExt,
	}
	var trimmed int
	for !writer.empty() && writer.last() > limit {
		if err := writer.pop(writer.last()); err != nil {
			return nil, err
		}
		trimmed += 1
	}
	if trimmed > 0 {
		log.Debug("Truncated extraneous elements", "count", trimmed, "limit", limit)
	}
	return writer, nil
}

// setBitmap applies the given extension elements into the bitmap.
func (b *blockWriter) setBitmap(ext []uint16) {
	for _, n := range ext {
		// Node ID zero is intentionally filtered out. Any element in this range
		// can indicate that the sub-tree's root node was mutated, so storing zero
		// is redundant and saves one byte for bitmap.
		if n != 0 {
			setBit(b.desc.extBitmap, int(n-1))
		}
	}
}

// append adds a new element to the block. The new element must be greater than
// the previous one. The provided ID is assumed to always be greater than 0.
//
// ext refers to the optional extension field attached to the appended element.
// This extension mechanism is used by trie-node history and represents a list of
// trie node IDs that fall within the range covered by the index element
// (typically corresponding to a sub-trie in trie-node history).
func (b *blockWriter) append(id uint64, ext []uint16) error {
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
		b.data = binary.AppendUvarint(b.data, id)
	} else {
		// The element which is not the first one in the section
		// is encoded using the value difference from the preceding
		// element.
		b.data = binary.AppendUvarint(b.data, id-b.desc.max)
	}
	// Extension validation
	if (len(ext) == 0) != !b.hasExt {
		if len(ext) == 0 {
			return errors.New("missing extension")
		}
		return errors.New("unexpected extension")
	}
	// Append the extension if it is not nil. The extension is prefixed with a
	// length indicator, and the block reader MUST understand this scheme and
	// decode the extension accordingly.
	if len(ext) > 0 {
		b.setBitmap(ext)
		enc := encodeIDs(ext)
		b.data = binary.AppendUvarint(b.data, uint64(len(enc)))
		b.data = append(b.data, enc...)
	}
	b.desc.entries++
	b.desc.max = id
	return nil
}

// scanSection traverses the specified section and terminates if fn returns true.
func (b *blockWriter) scanSection(section int, fn func(uint64, int, []uint16) bool) error {
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
		// Resolve the extension if exists
		var (
			err    error
			ext    []uint16
			extLen int
		)
		if b.hasExt {
			l, ln := binary.Uvarint(b.data[pos+n:])
			extLen = ln + int(l)
			ext, err = decodeIDs(b.data[pos+n+ln : pos+n+extLen])
		}
		if err != nil {
			return err
		}
		if fn(value, pos, ext) {
			return nil
		}
		// Shift to next position
		pos += n
		pos += extLen
	}
	return nil
}

// sectionLast returns the last element in the specified section.
func (b *blockWriter) sectionLast(section int) (uint64, error) {
	var n uint64
	if err := b.scanSection(section, func(v uint64, _ int, _ []uint16) bool {
		n = v
		return false
	}); err != nil {
		return 0, err
	}
	return n, nil
}

// sectionSearch looks up the specified value in the given section,
// the position and the preceding value will be returned if found.
// It assumes that the preceding element exists in the section.
func (b *blockWriter) sectionSearch(section int, n uint64) (found bool, prev uint64, pos int, err error) {
	if err := b.scanSection(section, func(v uint64, p int, _ []uint16) bool {
		if n == v {
			pos = p
			found = true
			return true // terminate iteration
		}
		prev = v
		return false // continue iteration
	}); err != nil {
		return false, 0, 0, err
	}
	return found, prev, pos, nil
}

// rebuildBitmap scans the entire block and rebuilds the bitmap.
func (b *blockWriter) rebuildBitmap() error {
	clear(b.desc.extBitmap)
	for i := 0; i < len(b.restarts); i++ {
		if err := b.scanSection(i, func(v uint64, p int, ext []uint16) bool {
			b.setBitmap(ext)
			return false // continue iteration
		}); err != nil {
			return err
		}
	}
	return nil
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
		b.desc.max = 0
		b.desc.entries = 0
		clear(b.desc.extBitmap)
		b.restarts = nil
		b.data = b.data[:0]
		return nil
	}
	// Pop the last restart section if the section becomes empty after removing
	// one element.
	if b.desc.entries%indexBlockRestartLen == 1 {
		b.data = b.data[:b.restarts[len(b.restarts)-1]]
		b.restarts = b.restarts[:len(b.restarts)-1]
		last, err := b.sectionLast(len(b.restarts) - 1)
		if err != nil {
			return err
		}
		b.desc.max = last
		b.desc.entries -= 1
		return b.rebuildBitmap()
	}
	// Look up the element preceding the one to be popped, in order to update
	// the maximum element in the block.
	found, prev, pos, err := b.sectionSearch(len(b.restarts)-1, id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("pop element is not found, last: %d, this: %d", b.desc.max, id)
	}
	b.desc.max = prev
	b.data = b.data[:pos]
	b.desc.entries -= 1
	return b.rebuildBitmap()
}

func (b *blockWriter) empty() bool {
	return b.desc.empty()
}

func (b *blockWriter) estimateFull(ext []uint16) bool {
	size := 8 + 2*len(ext)
	return len(b.data)+size > indexBlockMaxSize
}

// last returns the last element in the block. It should only be called when
// writer is not empty, otherwise the returned data is meaningless.
func (b *blockWriter) last() uint64 {
	if b.empty() {
		return 0
	}
	return b.desc.max
}

// finish finalizes the index block encoding by appending the encoded restart points
// and the restart counter to the end of the block.
//
// This function is safe to be called multiple times.
func (b *blockWriter) finish() []byte {
	buf := make([]byte, len(b.restarts)*2+1)
	for i, restart := range b.restarts {
		binary.BigEndian.PutUint16(buf[2*i:], restart)
	}
	buf[len(buf)-1] = byte(len(b.restarts))
	return append(b.data, buf...)
}
