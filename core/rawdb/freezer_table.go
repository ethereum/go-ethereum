// Copyright 2018 The go-ethereum Authors
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
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/golang/snappy"
)

var (
	// errClosed is returned if an operation attempts to read from or write to the
	// freezer table after it has already been closed.
	errClosed = errors.New("closed")

	// errOutOfBounds is returned if the item requested is not contained within the
	// freezer table.
	errOutOfBounds = errors.New("out of bounds")
)

// freezerTable represents a single chained data table within the freezer (e.g. blocks).
// It consists of a data file (snappy encoded arbitrary data blobs) and an index
// file (uncompressed 64 bit indices into the data file).
type freezerTable struct {
	content *os.File // File descriptor for the data content of the table
	offsets *os.File // File descriptor for the index file of the table

	items uint64 // Number of items stored in the table
	bytes uint64 // Number of content bytes stored in the table

	readMeter  metrics.Meter // Meter for measuring the effective amount of data read
	writeMeter metrics.Meter // Meter for measuring the effective amount of data written

	logger log.Logger   // Logger with database path and table name ambedded
	lock   sync.RWMutex // Mutex protecting the data file descriptors
}

// newTable opens a freezer table, creating the data and index files if they are
// non existent. Both files are truncated to the shortest common length to ensure
// they don't go out of sync.
func newTable(path string, name string, readMeter metrics.Meter, writeMeter metrics.Meter) (*freezerTable, error) {
	// Ensure the containing directory exists and open the two data files
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	content, err := os.OpenFile(filepath.Join(path, name+".dat"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	offsets, err := os.OpenFile(filepath.Join(path, name+".idx"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		content.Close()
		return nil, err
	}
	// Create the table and repair any past inconsistency
	tab := &freezerTable{
		content:    content,
		offsets:    offsets,
		readMeter:  readMeter,
		writeMeter: writeMeter,
		logger:     log.New("database", path, "table", name),
	}
	if err := tab.repair(); err != nil {
		offsets.Close()
		content.Close()
		return nil, err
	}
	return tab, nil
}

// repair cross checks the content and the offsets file and truncates them to
// be in sync with each other after a potential crash / data loss.
func (t *freezerTable) repair() error {
	// Create a temporary offset buffer to init files with and read offsts into
	offset := make([]byte, 8)

	// If we've just created the files, initialize the offsets with the 0 index
	stat, err := t.offsets.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		if _, err := t.offsets.Write(offset); err != nil {
			return err
		}
	}
	// Ensure the offsets are a multiple of 8 bytes
	if overflow := stat.Size() % 8; overflow != 0 {
		t.offsets.Truncate(stat.Size() - overflow) // New file can't trigger this path
	}
	// Retrieve the file sizes and prepare for truncation
	if stat, err = t.offsets.Stat(); err != nil {
		return err
	}
	offsetsSize := stat.Size()

	if stat, err = t.content.Stat(); err != nil {
		return err
	}
	contentSize := stat.Size()

	// Keep truncating both files until they come in sync
	t.offsets.ReadAt(offset, offsetsSize-8)
	contentExp := int64(binary.LittleEndian.Uint64(offset))

	for contentExp != contentSize {
		// Truncate the content file to the last offset pointer
		if contentExp < contentSize {
			t.logger.Warn("Truncating dangling content", "indexed", common.StorageSize(contentExp), "stored", common.StorageSize(contentSize))
			if err := t.content.Truncate(contentExp); err != nil {
				return err
			}
			contentSize = contentExp
		}
		// Truncate the offsets to point within the content file
		if contentExp > contentSize {
			t.logger.Warn("Truncating dangling offsets", "indexed", common.StorageSize(contentExp), "stored", common.StorageSize(contentSize))
			if err := t.offsets.Truncate(offsetsSize - 8); err != nil {
				return err
			}
			offsetsSize -= 8

			t.offsets.ReadAt(offset, offsetsSize-8)
			contentExp = int64(binary.LittleEndian.Uint64(offset))
		}
	}
	// Ensure all reparation changes have been written to disk
	if err := t.offsets.Sync(); err != nil {
		return err
	}
	if err := t.content.Sync(); err != nil {
		return err
	}
	// Update the item and byte counters and return
	t.items = uint64(offsetsSize/8 - 1) // last index points to the end of the data file
	t.bytes = uint64(contentSize)

	t.logger.Debug("Chain freezer table opened", "items", t.items, "size", common.StorageSize(t.bytes))
	return nil
}

// truncate discards any recent data above the provided threashold number.
func (t *freezerTable) truncate(items uint64) error {
	// If out item count is corrent, don't do anything
	if t.items <= items {
		return nil
	}
	// Something's out of sync, truncate the table's offset index
	t.logger.Warn("Truncating freezer table", "items", t.items, "limit", items)
	if err := t.offsets.Truncate(int64(items+1) * 8); err != nil {
		return err
	}
	// Calculate the new expected size of the data file and truncate it
	offset := make([]byte, 8)
	t.offsets.ReadAt(offset, int64(items)*8)
	expected := binary.LittleEndian.Uint64(offset)

	if err := t.content.Truncate(int64(expected)); err != nil {
		return err
	}
	// All data files truncated, set internal counters and return
	t.items, t.bytes = items, expected
	return nil
}

// Close unmaps all active memory mapped regions.
func (t *freezerTable) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	var errs []error
	if err := t.offsets.Close(); err != nil {
		errs = append(errs, err)
	}
	t.offsets = nil

	if err := t.content.Close(); err != nil {
		errs = append(errs, err)
	}
	t.content = nil

	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// Append injects a binary blob at the end of the freezer table. The item index
// is a precautionary parameter to ensure data correctness, but the table will
// reject already existing data.
//
// Note, this method will *not* flush any data to disk so be sure to explicitly
// fsync before irreversibly deleting data from the database.
func (t *freezerTable) Append(item uint64, blob []byte) error {
	// Ensure the table is still accessible
	if t.offsets == nil || t.content == nil {
		return errClosed
	}
	// Ensure only the next item can be written, nothing else
	if t.items != item {
		panic(fmt.Sprintf("appending unexpected item: want %d, have %d", t.items, item))
	}
	// Encode the blob and write it into the data file
	blob = snappy.Encode(nil, blob)
	if _, err := t.content.Write(blob); err != nil {
		return err
	}
	t.bytes += uint64(len(blob))

	offset := make([]byte, 8)
	binary.LittleEndian.PutUint64(offset, t.bytes)
	if _, err := t.offsets.Write(offset); err != nil {
		return err
	}
	t.items++

	t.writeMeter.Mark(int64(len(blob) + 8)) // 8 = 1 x 8 byte offset
	return nil
}

// Retrieve looks up the data offset of an item with the given index and retrieves
// the raw binary blob from the data file.
func (t *freezerTable) Retrieve(item uint64) ([]byte, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// Ensure the table and the item is accessible
	if t.offsets == nil || t.content == nil {
		return nil, errClosed
	}
	if t.items <= item {
		return nil, errOutOfBounds
	}
	// Item reachable, retrieve the data content boundaries
	offset := make([]byte, 8)
	if _, err := t.offsets.ReadAt(offset, int64(item*8)); err != nil {
		return nil, err
	}
	start := binary.LittleEndian.Uint64(offset)

	if _, err := t.offsets.ReadAt(offset, int64((item+1)*8)); err != nil {
		return nil, err
	}
	end := binary.LittleEndian.Uint64(offset)

	// Retrieve the data itself, decompress and return
	blob := make([]byte, end-start)
	if _, err := t.content.ReadAt(blob, int64(start)); err != nil {
		return nil, err
	}
	t.readMeter.Mark(int64(len(blob) + 16)) // 16 = 2 x 8 byte offset
	return snappy.Decode(nil, blob)
}

// Sync pushes any pending data from memory out to disk. This is an expensive
// operation, so use it with care.
func (t *freezerTable) Sync() error {
	if err := t.offsets.Sync(); err != nil {
		return err
	}
	return t.content.Sync()
}
