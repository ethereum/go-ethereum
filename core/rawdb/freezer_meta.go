// Copyright 2022 The go-ethereum Authors
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

package rawdb

import (
	"errors"
	"io"
	"math"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	freezerTableV1 = 1              // Initial version of metadata struct
	freezerTableV2 = 2              // Add field: 'flushOffset'
	freezerVersion = freezerTableV2 // The current used version
)

// freezerTableMeta is a collection of additional properties that describe the
// freezer table. These properties are designed with error resilience, allowing
// them to be automatically corrected after an error occurs without significantly
// impacting overall correctness.
type freezerTableMeta struct {
	file    *os.File // file handler of metadata
	version uint16   // version descriptor of the freezer table

	// virtualTail represents the number of items marked as deleted. It is
	// calculated as the sum of items removed from the table and the items
	// hidden within the table, and should never be less than the "actual
	// tail".
	//
	// If lost due to a crash or other reasons, it will be reset to the number
	// of items deleted from the table, causing the previously hidden items
	// to become visible, which is an acceptable consequence.
	virtualTail uint64

	// flushOffset represents the offset in the index file up to which the index
	// items along with the corresponding data items in data files has been flushed
	// (fsync’d) to disk. Beyond this offset, data integrity is not guaranteed,
	// the extra index items along with the associated data items should be removed
	// during the startup.
	//
	// The principle is that all data items above the flush offset are considered
	// volatile and should be recoverable if they are discarded after the unclean
	// shutdown. If data integrity is required, manually force a sync of the
	// freezer before proceeding with further operations (e.g. do freezer.Sync()
	// first and then write data to key value store in some circumstances).
	//
	// The offset could be moved forward by applying sync operation, or be moved
	// backward in cases of head/tail truncation, etc.
	flushOffset int64
}

// decodeV1 attempts to decode the metadata structure in v1 format. If fails or
// the result is incompatible, nil is returned.
func decodeV1(file *os.File) *freezerTableMeta {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil
	}
	type obj struct {
		Version uint16
		Tail    uint64
	}
	var o obj
	if err := rlp.Decode(file, &o); err != nil {
		return nil
	}
	if o.Version != freezerTableV1 {
		return nil
	}
	return &freezerTableMeta{
		file:        file,
		version:     o.Version,
		virtualTail: o.Tail,
	}
}

// decodeV2 attempts to decode the metadata structure in v2 format. If fails or
// the result is incompatible, nil is returned.
func decodeV2(file *os.File) *freezerTableMeta {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil
	}
	type obj struct {
		Version uint16
		Tail    uint64
		Offset  uint64
	}
	var o obj
	if err := rlp.Decode(file, &o); err != nil {
		return nil
	}
	if o.Version != freezerTableV2 {
		return nil
	}
	if o.Offset > math.MaxInt64 {
		log.Error("Invalid flushOffset %d in freezer metadata", o.Offset, "file", file.Name())
		return nil
	}
	return &freezerTableMeta{
		file:        file,
		version:     freezerTableV2,
		virtualTail: o.Tail,
		flushOffset: int64(o.Offset),
	}
}

// newMetadata initializes the metadata object, either by loading it from the file
// or by constructing a new one from scratch.
func newMetadata(file *os.File) (*freezerTableMeta, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() == 0 {
		m := &freezerTableMeta{
			file:        file,
			version:     freezerTableV2,
			virtualTail: 0,
			flushOffset: 0,
		}
		if err := m.write(true); err != nil {
			return nil, err
		}
		return m, nil
	}
	if m := decodeV2(file); m != nil {
		return m, nil
	}
	if m := decodeV1(file); m != nil {
		return m, nil // legacy metadata
	}
	return nil, errors.New("failed to decode metadata")
}

// setVirtualTail sets the virtual tail and flushes the metadata if sync is true.
func (m *freezerTableMeta) setVirtualTail(tail uint64, sync bool) error {
	m.virtualTail = tail
	return m.write(sync)
}

// setFlushOffset sets the flush offset and flushes the metadata if sync is true.
func (m *freezerTableMeta) setFlushOffset(offset int64, sync bool) error {
	m.flushOffset = offset
	return m.write(sync)
}

// write flushes the content of metadata into file and performs a fsync if required.
func (m *freezerTableMeta) write(sync bool) error {
	type obj struct {
		Version uint16
		Tail    uint64
		Offset  uint64
	}
	var o obj
	o.Version = freezerVersion // forcibly use the current version
	o.Tail = m.virtualTail
	o.Offset = uint64(m.flushOffset)

	_, err := m.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	if err := rlp.Encode(m.file, &o); err != nil {
		return err
	}
	if !sync {
		return nil
	}
	return m.file.Sync()
}
