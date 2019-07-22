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

package rawdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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

	// errNotSupported is returned if the database doesn't support the required operation.
	errNotSupported = errors.New("this operation is not supported")
)

// indexEntry contains the number/id of the file that the data resides in, aswell as the
// offset within the file to the end of the data
// In serialized form, the filenum is stored as uint16.
type indexEntry struct {
	filenum uint32 // stored as uint16 ( 2 bytes)
	offset  uint32 // stored as uint32 ( 4 bytes)
}

const indexEntrySize = 6

// unmarshallBinary deserializes binary b into the rawIndex entry.
func (i *indexEntry) unmarshalBinary(b []byte) error {
	i.filenum = uint32(binary.BigEndian.Uint16(b[:2]))
	i.offset = binary.BigEndian.Uint32(b[2:6])
	return nil
}

// marshallBinary serializes the rawIndex entry into binary.
func (i *indexEntry) marshallBinary() []byte {
	b := make([]byte, indexEntrySize)
	binary.BigEndian.PutUint16(b[:2], uint16(i.filenum))
	binary.BigEndian.PutUint32(b[2:6], i.offset)
	return b
}

// freezerTable represents a single chained data table within the freezer (e.g. blocks).
// It consists of a data file (snappy encoded arbitrary data blobs) and an indexEntry
// file (uncompressed 64 bit indices into the data file).
type freezerTable struct {
	// WARNING: The `items` field is accessed atomically. On 32 bit platforms, only
	// 64-bit aligned fields can be atomic. The struct is guaranteed to be so aligned,
	// so take advantage of that (https://golang.org/pkg/sync/atomic/#pkg-note-BUG).
	items uint64 // Number of items stored in the table (including items removed from tail)

	noCompression bool   // if true, disables snappy compression. Note: does not work retroactively
	maxFileSize   uint32 // Max file size for data-files
	name          string
	path          string

	head   *os.File            // File descriptor for the data head of the table
	files  map[uint32]*os.File // open files
	headId uint32              // number of the currently active head file
	tailId uint32              // number of the earliest file
	index  *os.File            // File descriptor for the indexEntry file of the table

	// In the case that old items are deleted (from the tail), we use itemOffset
	// to count how many historic items have gone missing.
	itemOffset uint32 // Offset (number of discarded items)

	headBytes   uint32          // Number of bytes written to the head file
	readMeter   metrics.Meter   // Meter for measuring the effective amount of data read
	writeMeter  metrics.Meter   // Meter for measuring the effective amount of data written
	sizeCounter metrics.Counter // Counter for tracking the combined size of all freezer tables

	logger log.Logger   // Logger with database path and table name ambedded
	lock   sync.RWMutex // Mutex protecting the data file descriptors
}

// newTable opens a freezer table with default settings - 2G files
func newTable(path string, name string, readMeter metrics.Meter, writeMeter metrics.Meter, sizeCounter metrics.Counter, disableSnappy bool) (*freezerTable, error) {
	return newCustomTable(path, name, readMeter, writeMeter, sizeCounter, 2*1000*1000*1000, disableSnappy)
}

// openFreezerFileForAppend opens a freezer table file and seeks to the end
func openFreezerFileForAppend(filename string) (*os.File, error) {
	// Open the file without the O_APPEND flag
	// because it has differing behaviour during Truncate operations
	// on different OS's
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	// Seek to end for append
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}
	return file, nil
}

// openFreezerFileForReadOnly opens a freezer table file for read only access
func openFreezerFileForReadOnly(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDONLY, 0644)
}

// openFreezerFileTruncated opens a freezer table making sure it is truncated
func openFreezerFileTruncated(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

// truncateFreezerFile resizes a freezer table file and seeks to the end
func truncateFreezerFile(file *os.File, size int64) error {
	if err := file.Truncate(size); err != nil {
		return err
	}
	// Seek to end for append
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return nil
}

// newCustomTable opens a freezer table, creating the data and index files if they are
// non existent. Both files are truncated to the shortest common length to ensure
// they don't go out of sync.
func newCustomTable(path string, name string, readMeter metrics.Meter, writeMeter metrics.Meter, sizeCounter metrics.Counter, maxFilesize uint32, noCompression bool) (*freezerTable, error) {
	// Ensure the containing directory exists and open the indexEntry file
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	var idxName string
	if noCompression {
		// Raw idx
		idxName = fmt.Sprintf("%s.ridx", name)
	} else {
		// Compressed idx
		idxName = fmt.Sprintf("%s.cidx", name)
	}
	offsets, err := openFreezerFileForAppend(filepath.Join(path, idxName))
	if err != nil {
		return nil, err
	}
	// Create the table and repair any past inconsistency
	tab := &freezerTable{
		index:         offsets,
		files:         make(map[uint32]*os.File),
		readMeter:     readMeter,
		writeMeter:    writeMeter,
		sizeCounter:   sizeCounter,
		name:          name,
		path:          path,
		logger:        log.New("database", path, "table", name),
		noCompression: noCompression,
		maxFileSize:   maxFilesize,
	}
	if err := tab.repair(); err != nil {
		tab.Close()
		return nil, err
	}
	// Initialize the starting size counter
	size, err := tab.sizeNolock()
	if err != nil {
		tab.Close()
		return nil, err
	}
	tab.sizeCounter.Inc(int64(size))

	return tab, nil
}

// repair cross checks the head and the index file and truncates them to
// be in sync with each other after a potential crash / data loss.
func (t *freezerTable) repair() error {
	// Create a temporary offset buffer to init files with and read indexEntry into
	buffer := make([]byte, indexEntrySize)

	// If we've just created the files, initialize the index with the 0 indexEntry
	stat, err := t.index.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		if _, err := t.index.Write(buffer); err != nil {
			return err
		}
	}
	// Ensure the index is a multiple of indexEntrySize bytes
	if overflow := stat.Size() % indexEntrySize; overflow != 0 {
		truncateFreezerFile(t.index, stat.Size()-overflow) // New file can't trigger this path
	}
	// Retrieve the file sizes and prepare for truncation
	if stat, err = t.index.Stat(); err != nil {
		return err
	}
	offsetsSize := stat.Size()

	// Open the head file
	var (
		firstIndex  indexEntry
		lastIndex   indexEntry
		contentSize int64
		contentExp  int64
	)
	// Read index zero, determine what file is the earliest
	// and what item offset to use
	t.index.ReadAt(buffer, 0)
	firstIndex.unmarshalBinary(buffer)

	t.tailId = firstIndex.offset
	t.itemOffset = firstIndex.filenum

	t.index.ReadAt(buffer, offsetsSize-indexEntrySize)
	lastIndex.unmarshalBinary(buffer)
	t.head, err = t.openFile(lastIndex.filenum, openFreezerFileForAppend)
	if err != nil {
		return err
	}
	if stat, err = t.head.Stat(); err != nil {
		return err
	}
	contentSize = stat.Size()

	// Keep truncating both files until they come in sync
	contentExp = int64(lastIndex.offset)

	for contentExp != contentSize {
		// Truncate the head file to the last offset pointer
		if contentExp < contentSize {
			t.logger.Warn("Truncating dangling head", "indexed", common.StorageSize(contentExp), "stored", common.StorageSize(contentSize))
			if err := truncateFreezerFile(t.head, contentExp); err != nil {
				return err
			}
			contentSize = contentExp
		}
		// Truncate the index to point within the head file
		if contentExp > contentSize {
			t.logger.Warn("Truncating dangling indexes", "indexed", common.StorageSize(contentExp), "stored", common.StorageSize(contentSize))
			if err := truncateFreezerFile(t.index, offsetsSize-indexEntrySize); err != nil {
				return err
			}
			offsetsSize -= indexEntrySize
			t.index.ReadAt(buffer, offsetsSize-indexEntrySize)
			var newLastIndex indexEntry
			newLastIndex.unmarshalBinary(buffer)
			// We might have slipped back into an earlier head-file here
			if newLastIndex.filenum != lastIndex.filenum {
				// Release earlier opened file
				t.releaseFile(lastIndex.filenum)
				if t.head, err = t.openFile(newLastIndex.filenum, openFreezerFileForAppend); err != nil {
					return err
				}
				if stat, err = t.head.Stat(); err != nil {
					// TODO, anything more we can do here?
					// A data file has gone missing...
					return err
				}
				contentSize = stat.Size()
			}
			lastIndex = newLastIndex
			contentExp = int64(lastIndex.offset)
		}
	}
	// Ensure all reparation changes have been written to disk
	if err := t.index.Sync(); err != nil {
		return err
	}
	if err := t.head.Sync(); err != nil {
		return err
	}
	// Update the item and byte counters and return
	t.items = uint64(t.itemOffset) + uint64(offsetsSize/indexEntrySize-1) // last indexEntry points to the end of the data file
	t.headBytes = uint32(contentSize)
	t.headId = lastIndex.filenum

	// Close opened files and preopen all files
	if err := t.preopen(); err != nil {
		return err
	}
	t.logger.Debug("Chain freezer table opened", "items", t.items, "size", common.StorageSize(t.headBytes))
	return nil
}

// preopen opens all files that the freezer will need. This method should be called from an init-context,
// since it assumes that it doesn't have to bother with locking
// The rationale for doing preopen is to not have to do it from within Retrieve, thus not needing to ever
// obtain a write-lock within Retrieve.
func (t *freezerTable) preopen() (err error) {
	// The repair might have already opened (some) files
	t.releaseFilesAfter(0, false)
	// Open all except head in RDONLY
	for i := t.tailId; i < t.headId; i++ {
		if _, err = t.openFile(i, openFreezerFileForReadOnly); err != nil {
			return err
		}
	}
	// Open head in read/write
	t.head, err = t.openFile(t.headId, openFreezerFileForAppend)
	return err
}

// truncate discards any recent data above the provided threshold number.
func (t *freezerTable) truncate(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// If our item count is correct, don't do anything
	if atomic.LoadUint64(&t.items) <= items {
		return nil
	}
	// We need to truncate, save the old size for metrics tracking
	oldSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	// Something's out of sync, truncate the table's offset index
	t.logger.Warn("Truncating freezer table", "items", t.items, "limit", items)
	if err := truncateFreezerFile(t.index, int64(items+1)*indexEntrySize); err != nil {
		return err
	}
	// Calculate the new expected size of the data file and truncate it
	buffer := make([]byte, indexEntrySize)
	if _, err := t.index.ReadAt(buffer, int64(items*indexEntrySize)); err != nil {
		return err
	}
	var expected indexEntry
	expected.unmarshalBinary(buffer)

	// We might need to truncate back to older files
	if expected.filenum != t.headId {
		// If already open for reading, force-reopen for writing
		t.releaseFile(expected.filenum)
		newHead, err := t.openFile(expected.filenum, openFreezerFileForAppend)
		if err != nil {
			return err
		}
		// Release any files _after the current head -- both the previous head
		// and any files which may have been opened for reading
		t.releaseFilesAfter(expected.filenum, true)
		// Set back the historic head
		t.head = newHead
		atomic.StoreUint32(&t.headId, expected.filenum)
	}
	if err := truncateFreezerFile(t.head, int64(expected.offset)); err != nil {
		return err
	}
	// All data files truncated, set internal counters and return
	atomic.StoreUint64(&t.items, items)
	atomic.StoreUint32(&t.headBytes, expected.offset)

	// Retrieve the new size and update the total size counter
	newSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	t.sizeCounter.Dec(int64(oldSize - newSize))

	return nil
}

// Close closes all opened files.
func (t *freezerTable) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	var errs []error
	if err := t.index.Close(); err != nil {
		errs = append(errs, err)
	}
	t.index = nil

	for _, f := range t.files {
		if err := f.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	t.head = nil

	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// openFile assumes that the write-lock is held by the caller
func (t *freezerTable) openFile(num uint32, opener func(string) (*os.File, error)) (f *os.File, err error) {
	var exist bool
	if f, exist = t.files[num]; !exist {
		var name string
		if t.noCompression {
			name = fmt.Sprintf("%s.%04d.rdat", t.name, num)
		} else {
			name = fmt.Sprintf("%s.%04d.cdat", t.name, num)
		}
		f, err = opener(filepath.Join(t.path, name))
		if err != nil {
			return nil, err
		}
		t.files[num] = f
	}
	return f, err
}

// releaseFile closes a file, and removes it from the open file cache.
// Assumes that the caller holds the write lock
func (t *freezerTable) releaseFile(num uint32) {
	if f, exist := t.files[num]; exist {
		delete(t.files, num)
		f.Close()
	}
}

// releaseFilesAfter closes all open files with a higher number, and optionally also deletes the files
func (t *freezerTable) releaseFilesAfter(num uint32, remove bool) {
	for fnum, f := range t.files {
		if fnum > num {
			delete(t.files, fnum)
			f.Close()
			if remove {
				os.Remove(f.Name())
			}
		}
	}
}

// Append injects a binary blob at the end of the freezer table. The item number
// is a precautionary parameter to ensure data correctness, but the table will
// reject already existing data.
//
// Note, this method will *not* flush any data to disk so be sure to explicitly
// fsync before irreversibly deleting data from the database.
func (t *freezerTable) Append(item uint64, blob []byte) error {
	// Read lock prevents competition with truncate
	t.lock.RLock()
	// Ensure the table is still accessible
	if t.index == nil || t.head == nil {
		t.lock.RUnlock()
		return errClosed
	}
	// Ensure only the next item can be written, nothing else
	if atomic.LoadUint64(&t.items) != item {
		t.lock.RUnlock()
		return fmt.Errorf("appending unexpected item: want %d, have %d", t.items, item)
	}
	// Encode the blob and write it into the data file
	if !t.noCompression {
		blob = snappy.Encode(nil, blob)
	}
	bLen := uint32(len(blob))
	if t.headBytes+bLen < bLen ||
		t.headBytes+bLen > t.maxFileSize {
		// we need a new file, writing would overflow
		t.lock.RUnlock()
		t.lock.Lock()
		nextID := atomic.LoadUint32(&t.headId) + 1
		// We open the next file in truncated mode -- if this file already
		// exists, we need to start over from scratch on it
		newHead, err := t.openFile(nextID, openFreezerFileTruncated)
		if err != nil {
			t.lock.Unlock()
			return err
		}
		// Close old file, and reopen in RDONLY mode
		t.releaseFile(t.headId)
		t.openFile(t.headId, openFreezerFileForReadOnly)

		// Swap out the current head
		t.head = newHead
		atomic.StoreUint32(&t.headBytes, 0)
		atomic.StoreUint32(&t.headId, nextID)
		t.lock.Unlock()
		t.lock.RLock()
	}

	defer t.lock.RUnlock()
	if _, err := t.head.Write(blob); err != nil {
		return err
	}
	newOffset := atomic.AddUint32(&t.headBytes, bLen)
	idx := indexEntry{
		filenum: atomic.LoadUint32(&t.headId),
		offset:  newOffset,
	}
	// Write indexEntry
	t.index.Write(idx.marshallBinary())

	t.writeMeter.Mark(int64(bLen + indexEntrySize))
	t.sizeCounter.Inc(int64(bLen + indexEntrySize))

	atomic.AddUint64(&t.items, 1)
	return nil
}

// getBounds returns the indexes for the item
// returns start, end, filenumber and error
func (t *freezerTable) getBounds(item uint64) (uint32, uint32, uint32, error) {
	var startIdx, endIdx indexEntry
	buffer := make([]byte, indexEntrySize)
	if _, err := t.index.ReadAt(buffer, int64(item*indexEntrySize)); err != nil {
		return 0, 0, 0, err
	}
	startIdx.unmarshalBinary(buffer)
	if _, err := t.index.ReadAt(buffer, int64((item+1)*indexEntrySize)); err != nil {
		return 0, 0, 0, err
	}
	endIdx.unmarshalBinary(buffer)
	if startIdx.filenum != endIdx.filenum {
		// If a piece of data 'crosses' a data-file,
		// it's actually in one piece on the second data-file.
		// We return a zero-indexEntry for the second file as start
		return 0, endIdx.offset, endIdx.filenum, nil
	}
	return startIdx.offset, endIdx.offset, endIdx.filenum, nil
}

// Retrieve looks up the data offset of an item with the given number and retrieves
// the raw binary blob from the data file.
func (t *freezerTable) Retrieve(item uint64) ([]byte, error) {
	// Ensure the table and the item is accessible
	if t.index == nil || t.head == nil {
		return nil, errClosed
	}
	if atomic.LoadUint64(&t.items) <= item {
		return nil, errOutOfBounds
	}
	// Ensure the item was not deleted from the tail either
	offset := atomic.LoadUint32(&t.itemOffset)
	if uint64(offset) > item {
		return nil, errOutOfBounds
	}
	t.lock.RLock()
	startOffset, endOffset, filenum, err := t.getBounds(item - uint64(offset))
	if err != nil {
		t.lock.RUnlock()
		return nil, err
	}
	dataFile, exist := t.files[filenum]
	if !exist {
		t.lock.RUnlock()
		return nil, fmt.Errorf("missing data file %d", filenum)
	}
	// Retrieve the data itself, decompress and return
	blob := make([]byte, endOffset-startOffset)
	if _, err := dataFile.ReadAt(blob, int64(startOffset)); err != nil {
		t.lock.RUnlock()
		return nil, err
	}
	t.lock.RUnlock()
	t.readMeter.Mark(int64(len(blob) + 2*indexEntrySize))

	if t.noCompression {
		return blob, nil
	}
	return snappy.Decode(nil, blob)
}

// has returns an indicator whether the specified number data
// exists in the freezer table.
func (t *freezerTable) has(number uint64) bool {
	return atomic.LoadUint64(&t.items) > number
}

// size returns the total data size in the freezer table.
func (t *freezerTable) size() (uint64, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.sizeNolock()
}

// sizeNolock returns the total data size in the freezer table without obtaining
// the mutex first.
func (t *freezerTable) sizeNolock() (uint64, error) {
	stat, err := t.index.Stat()
	if err != nil {
		return 0, err
	}
	total := uint64(t.maxFileSize)*uint64(t.headId-t.tailId) + uint64(t.headBytes) + uint64(stat.Size())
	return total, nil
}

// Sync pushes any pending data from memory out to disk. This is an expensive
// operation, so use it with care.
func (t *freezerTable) Sync() error {
	if err := t.index.Sync(); err != nil {
		return err
	}
	return t.head.Sync()
}

// printIndex is a debug print utility function for testing
func (t *freezerTable) printIndex() {
	buf := make([]byte, indexEntrySize)

	fmt.Printf("|-----------------|\n")
	fmt.Printf("| fileno | offset |\n")
	fmt.Printf("|--------+--------|\n")

	for i := uint64(0); ; i++ {
		if _, err := t.index.ReadAt(buf, int64(i*indexEntrySize)); err != nil {
			break
		}
		var entry indexEntry
		entry.unmarshalBinary(buf)
		fmt.Printf("|  %03d   |  %03d   | \n", entry.filenum, entry.offset)
		if i > 100 {
			fmt.Printf(" ... \n")
			break
		}
	}
	fmt.Printf("|-----------------|\n")
}
