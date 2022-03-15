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
	"bytes"
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
// offset within the file to the end of the data.
// In serialized form, the filenum is stored as uint16.
type indexEntry struct {
	filenum uint32 // stored as uint16 ( 2 bytes )
	offset  uint32 // stored as uint32 ( 4 bytes )
}

const indexEntrySize = 6

// unmarshalBinary deserializes binary b into the rawIndex entry.
func (i *indexEntry) unmarshalBinary(b []byte) {
	i.filenum = uint32(binary.BigEndian.Uint16(b[:2]))
	i.offset = binary.BigEndian.Uint32(b[2:6])
}

// append adds the encoded entry to the end of b.
func (i *indexEntry) append(b []byte) []byte {
	offset := len(b)
	out := append(b, make([]byte, indexEntrySize)...)
	binary.BigEndian.PutUint16(out[offset:], uint16(i.filenum))
	binary.BigEndian.PutUint32(out[offset+2:], i.offset)
	return out
}

// bounds returns the start- and end- offsets, and the file number of where to
// read there data item marked by the two index entries. The two entries are
// assumed to be sequential.
func (i *indexEntry) bounds(end *indexEntry) (startOffset, endOffset, fileId uint32) {
	if i.filenum != end.filenum {
		// If a piece of data 'crosses' a data-file,
		// it's actually in one piece on the second data-file.
		// We return a zero-indexEntry for the second file as start
		return 0, end.offset, end.filenum
	}
	return i.offset, end.offset, end.filenum
}

// freezerTable represents a single chained data table within the freezer (e.g. blocks).
// It consists of a data file (snappy encoded arbitrary data blobs) and an indexEntry
// file (uncompressed 64 bit indices into the data file).
type freezerTable struct {
	// WARNING: The `items` field is accessed atomically. On 32 bit platforms, only
	// 64-bit aligned fields can be atomic. The struct is guaranteed to be so aligned,
	// so take advantage of that (https://golang.org/pkg/sync/atomic/#pkg-note-BUG).
	items      uint64 // Number of items stored in the table (including items removed from tail)
	itemOffset uint64 // Number of items removed from the table

	// itemHidden is the number of items marked as deleted. Tail deletion is
	// only supported at file level which means the actual deletion will be
	// delayed until the entire data file is marked as deleted. Before that
	// these items will be hidden to prevent being visited again. The value
	// should never be lower than itemOffset.
	itemHidden uint64

	noCompression bool // if true, disables snappy compression. Note: does not work retroactively
	readonly      bool
	maxFileSize   uint32 // Max file size for data-files
	name          string
	path          string

	head   *os.File            // File descriptor for the data head of the table
	index  *os.File            // File descriptor for the indexEntry file of the table
	meta   *os.File            // File descriptor for metadata of the table
	files  map[uint32]*os.File // open files
	headId uint32              // number of the currently active head file
	tailId uint32              // number of the earliest file

	headBytes  int64         // Number of bytes written to the head file
	readMeter  metrics.Meter // Meter for measuring the effective amount of data read
	writeMeter metrics.Meter // Meter for measuring the effective amount of data written
	sizeGauge  metrics.Gauge // Gauge for tracking the combined size of all freezer tables

	logger log.Logger   // Logger with database path and table name ambedded
	lock   sync.RWMutex // Mutex protecting the data file descriptors
}

// NewFreezerTable opens the given path as a freezer table.
func NewFreezerTable(path, name string, disableSnappy, readonly bool) (*freezerTable, error) {
	return newTable(path, name, metrics.NilMeter{}, metrics.NilMeter{}, metrics.NilGauge{}, freezerTableSize, disableSnappy, readonly)
}

// newTable opens a freezer table, creating the data and index files if they are
// non-existent. Both files are truncated to the shortest common length to ensure
// they don't go out of sync.
func newTable(path string, name string, readMeter metrics.Meter, writeMeter metrics.Meter, sizeGauge metrics.Gauge, maxFilesize uint32, noCompression, readonly bool) (*freezerTable, error) {
	// Ensure the containing directory exists and open the indexEntry file
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	var idxName string
	if noCompression {
		idxName = fmt.Sprintf("%s.ridx", name) // raw index file
	} else {
		idxName = fmt.Sprintf("%s.cidx", name) // compressed index file
	}
	var (
		err   error
		index *os.File
		meta  *os.File
	)
	if readonly {
		// Will fail if table doesn't exist
		index, err = openFreezerFileForReadOnly(filepath.Join(path, idxName))
		if err != nil {
			return nil, err
		}
		// TODO(rjl493456442) change it to read-only mode. Open the metadata file
		// in rw mode. It's a temporary solution for now and should be changed
		// whenever the tail deletion is actually used. The reason for this hack is
		// the additional meta file for each freezer table is added in order to support
		// tail deletion, but for most legacy nodes this file is missing. This check
		// will suddenly break lots of database relevant commands. So the metadata file
		// is always opened for mutation and nothing else will be written except
		// the initialization.
		meta, err = openFreezerFileForAppend(filepath.Join(path, fmt.Sprintf("%s.meta", name)))
		if err != nil {
			return nil, err
		}
	} else {
		index, err = openFreezerFileForAppend(filepath.Join(path, idxName))
		if err != nil {
			return nil, err
		}
		meta, err = openFreezerFileForAppend(filepath.Join(path, fmt.Sprintf("%s.meta", name)))
		if err != nil {
			return nil, err
		}
	}
	// Create the table and repair any past inconsistency
	tab := &freezerTable{
		index:         index,
		meta:          meta,
		files:         make(map[uint32]*os.File),
		readMeter:     readMeter,
		writeMeter:    writeMeter,
		sizeGauge:     sizeGauge,
		name:          name,
		path:          path,
		logger:        log.New("database", path, "table", name),
		noCompression: noCompression,
		readonly:      readonly,
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
	tab.sizeGauge.Inc(int64(size))

	return tab, nil
}

// repair cross-checks the head and the index file and truncates them to
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

	// Assign the tail fields with the first stored index.
	// The total removed items is represented with an uint32,
	// which is not enough in theory but enough in practice.
	// TODO: use uint64 to represent total removed items.
	t.tailId = firstIndex.filenum
	t.itemOffset = uint64(firstIndex.offset)

	// Load metadata from the file
	meta, err := loadMetadata(t.meta, t.itemOffset)
	if err != nil {
		return err
	}
	t.itemHidden = meta.VirtualTail

	// Read the last index, use the default value in case the freezer is empty
	if offsetsSize == indexEntrySize {
		lastIndex = indexEntry{filenum: t.tailId, offset: 0}
	} else {
		t.index.ReadAt(buffer, offsetsSize-indexEntrySize)
		lastIndex.unmarshalBinary(buffer)
	}
	if t.readonly {
		t.head, err = t.openFile(lastIndex.filenum, openFreezerFileForReadOnly)
	} else {
		t.head, err = t.openFile(lastIndex.filenum, openFreezerFileForAppend)
	}
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

			// Read the new head index, use the default value in case
			// the freezer is already empty.
			var newLastIndex indexEntry
			if offsetsSize == indexEntrySize {
				newLastIndex = indexEntry{filenum: t.tailId, offset: 0}
			} else {
				t.index.ReadAt(buffer, offsetsSize-indexEntrySize)
				newLastIndex.unmarshalBinary(buffer)
			}
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
	// Sync() fails for read-only files on windows.
	if !t.readonly {
		// Ensure all reparation changes have been written to disk
		if err := t.index.Sync(); err != nil {
			return err
		}
		if err := t.head.Sync(); err != nil {
			return err
		}
		if err := t.meta.Sync(); err != nil {
			return err
		}
	}
	// Update the item and byte counters and return
	t.items = t.itemOffset + uint64(offsetsSize/indexEntrySize-1) // last indexEntry points to the end of the data file
	t.headBytes = contentSize
	t.headId = lastIndex.filenum

	// Delete the leftover files because of head deletion
	t.releaseFilesAfter(t.headId, true)

	// Delete the leftover files because of tail deletion
	t.releaseFilesBefore(t.tailId, true)

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
	if t.readonly {
		t.head, err = t.openFile(t.headId, openFreezerFileForReadOnly)
	} else {
		// Open head in read/write
		t.head, err = t.openFile(t.headId, openFreezerFileForAppend)
	}
	return err
}

// truncateHead discards any recent data above the provided threshold number.
func (t *freezerTable) truncateHead(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Ensure the given truncate target falls in the correct range
	existing := atomic.LoadUint64(&t.items)
	if existing <= items {
		return nil
	}
	if items < atomic.LoadUint64(&t.itemHidden) {
		return errors.New("truncation below tail")
	}
	// We need to truncate, save the old size for metrics tracking
	oldSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	// Something's out of sync, truncate the table's offset index
	log := t.logger.Debug
	if existing > items+1 {
		log = t.logger.Warn // Only loud warn if we delete multiple items
	}
	log("Truncating freezer table", "items", existing, "limit", items)

	// Truncate the index file first, the tail position is also considered
	// when calculating the new freezer table length.
	length := items - atomic.LoadUint64(&t.itemOffset)
	if err := truncateFreezerFile(t.index, int64(length+1)*indexEntrySize); err != nil {
		return err
	}
	// Calculate the new expected size of the data file and truncate it
	var expected indexEntry
	if length == 0 {
		expected = indexEntry{filenum: t.tailId, offset: 0}
	} else {
		buffer := make([]byte, indexEntrySize)
		if _, err := t.index.ReadAt(buffer, int64(length*indexEntrySize)); err != nil {
			return err
		}
		expected.unmarshalBinary(buffer)
	}
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
		t.headId = expected.filenum
	}
	if err := truncateFreezerFile(t.head, int64(expected.offset)); err != nil {
		return err
	}
	// All data files truncated, set internal counters and return
	t.headBytes = int64(expected.offset)
	atomic.StoreUint64(&t.items, items)

	// Retrieve the new size and update the total size counter
	newSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	t.sizeGauge.Dec(int64(oldSize - newSize))
	return nil
}

// truncateTail discards any recent data before the provided threshold number.
func (t *freezerTable) truncateTail(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Ensure the given truncate target falls in the correct range
	if atomic.LoadUint64(&t.itemHidden) >= items {
		return nil
	}
	if atomic.LoadUint64(&t.items) < items {
		return errors.New("truncation above head")
	}
	// Load the new tail index by the given new tail position
	var (
		newTailId uint32
		buffer    = make([]byte, indexEntrySize)
	)
	if atomic.LoadUint64(&t.items) == items {
		newTailId = t.headId
	} else {
		offset := items - atomic.LoadUint64(&t.itemOffset)
		if _, err := t.index.ReadAt(buffer, int64((offset+1)*indexEntrySize)); err != nil {
			return err
		}
		var newTail indexEntry
		newTail.unmarshalBinary(buffer)
		newTailId = newTail.filenum
	}
	// Update the virtual tail marker and hidden these entries in table.
	atomic.StoreUint64(&t.itemHidden, items)
	if err := writeMetadata(t.meta, newMetadata(items)); err != nil {
		return err
	}
	// Hidden items still fall in the current tail file, no data file
	// can be dropped.
	if t.tailId == newTailId {
		return nil
	}
	// Hidden items fall in the incorrect range, returns the error.
	if t.tailId > newTailId {
		return fmt.Errorf("invalid index, tail-file %d, item-file %d", t.tailId, newTailId)
	}
	// Hidden items exceed the current tail file, drop the relevant
	// data files. We need to truncate, save the old size for metrics
	// tracking.
	oldSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	// Count how many items can be deleted from the file.
	var (
		newDeleted = items
		deleted    = atomic.LoadUint64(&t.itemOffset)
	)
	for current := items - 1; current >= deleted; current -= 1 {
		if _, err := t.index.ReadAt(buffer, int64((current-deleted+1)*indexEntrySize)); err != nil {
			return err
		}
		var pre indexEntry
		pre.unmarshalBinary(buffer)
		if pre.filenum != newTailId {
			break
		}
		newDeleted = current
	}
	// Commit the changes of metadata file first before manipulating
	// the indexes file.
	if err := t.meta.Sync(); err != nil {
		return err
	}
	// Truncate the deleted index entries from the index file.
	err = copyFrom(t.index.Name(), t.index.Name(), indexEntrySize*(newDeleted-deleted+1), func(f *os.File) error {
		tailIndex := indexEntry{
			filenum: newTailId,
			offset:  uint32(newDeleted),
		}
		_, err := f.Write(tailIndex.append(nil))
		return err
	})
	if err != nil {
		return err
	}
	// Reopen the modified index file to load the changes
	if err := t.index.Close(); err != nil {
		return err
	}
	t.index, err = openFreezerFileForAppend(t.index.Name())
	if err != nil {
		return err
	}
	// Release any files before the current tail
	t.tailId = newTailId
	atomic.StoreUint64(&t.itemOffset, newDeleted)
	t.releaseFilesBefore(t.tailId, true)

	// Retrieve the new size and update the total size counter
	newSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	t.sizeGauge.Dec(int64(oldSize - newSize))
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

	if err := t.meta.Close(); err != nil {
		errs = append(errs, err)
	}
	t.meta = nil

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

// releaseFilesBefore closes all open files with a lower number, and optionally also deletes the files
func (t *freezerTable) releaseFilesBefore(num uint32, remove bool) {
	for fnum, f := range t.files {
		if fnum < num {
			delete(t.files, fnum)
			f.Close()
			if remove {
				os.Remove(f.Name())
			}
		}
	}
}

// getIndices returns the index entries for the given from-item, covering 'count' items.
// N.B: The actual number of returned indices for N items will always be N+1 (unless an
// error is returned).
// OBS: This method assumes that the caller has already verified (and/or trimmed) the range
// so that the items are within bounds. If this method is used to read out of bounds,
// it will return error.
func (t *freezerTable) getIndices(from, count uint64) ([]*indexEntry, error) {
	// Apply the table-offset
	from = from - t.itemOffset
	// For reading N items, we need N+1 indices.
	buffer := make([]byte, (count+1)*indexEntrySize)
	if _, err := t.index.ReadAt(buffer, int64(from*indexEntrySize)); err != nil {
		return nil, err
	}
	var (
		indices []*indexEntry
		offset  int
	)
	for i := from; i <= from+count; i++ {
		index := new(indexEntry)
		index.unmarshalBinary(buffer[offset:])
		offset += indexEntrySize
		indices = append(indices, index)
	}
	if from == 0 {
		// Special case if we're reading the first item in the freezer. We assume that
		// the first item always start from zero(regarding the deletion, we
		// only support deletion by files, so that the assumption is held).
		// This means we can use the first item metadata to carry information about
		// the 'global' offset, for the deletion-case
		indices[0].offset = 0
		indices[0].filenum = indices[1].filenum
	}
	return indices, nil
}

// Retrieve looks up the data offset of an item with the given number and retrieves
// the raw binary blob from the data file.
func (t *freezerTable) Retrieve(item uint64) ([]byte, error) {
	items, err := t.RetrieveItems(item, 1, 0)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

// RetrieveItems returns multiple items in sequence, starting from the index 'start'.
// It will return at most 'max' items, but will abort earlier to respect the
// 'maxBytes' argument. However, if the 'maxBytes' is smaller than the size of one
// item, it _will_ return one element and possibly overflow the maxBytes.
func (t *freezerTable) RetrieveItems(start, count, maxBytes uint64) ([][]byte, error) {
	// First we read the 'raw' data, which might be compressed.
	diskData, sizes, err := t.retrieveItems(start, count, maxBytes)
	if err != nil {
		return nil, err
	}
	var (
		output     = make([][]byte, 0, count)
		offset     int // offset for reading
		outputSize int // size of uncompressed data
	)
	// Now slice up the data and decompress.
	for i, diskSize := range sizes {
		item := diskData[offset : offset+diskSize]
		offset += diskSize
		decompressedSize := diskSize
		if !t.noCompression {
			decompressedSize, _ = snappy.DecodedLen(item)
		}
		if i > 0 && uint64(outputSize+decompressedSize) > maxBytes {
			break
		}
		if !t.noCompression {
			data, err := snappy.Decode(nil, item)
			if err != nil {
				return nil, err
			}
			output = append(output, data)
		} else {
			output = append(output, item)
		}
		outputSize += decompressedSize
	}
	return output, nil
}

// retrieveItems reads up to 'count' items from the table. It reads at least
// one item, but otherwise avoids reading more than maxBytes bytes.
// It returns the (potentially compressed) data, and the sizes.
func (t *freezerTable) retrieveItems(start, count, maxBytes uint64) ([]byte, []int, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// Ensure the table and the item are accessible
	if t.index == nil || t.head == nil {
		return nil, nil, errClosed
	}
	var (
		items  = atomic.LoadUint64(&t.items)      // the total items(head + 1)
		hidden = atomic.LoadUint64(&t.itemHidden) // the number of hidden items
	)
	// Ensure the start is written, not deleted from the tail, and that the
	// caller actually wants something
	if items <= start || hidden > start || count == 0 {
		return nil, nil, errOutOfBounds
	}
	if start+count > items {
		count = items - start
	}
	var (
		output     = make([]byte, maxBytes) // Buffer to read data into
		outputSize int                      // Used size of that buffer
	)
	// readData is a helper method to read a single data item from disk.
	readData := func(fileId, start uint32, length int) error {
		// In case a small limit is used, and the elements are large, may need to
		// realloc the read-buffer when reading the first (and only) item.
		if len(output) < length {
			output = make([]byte, length)
		}
		dataFile, exist := t.files[fileId]
		if !exist {
			return fmt.Errorf("missing data file %d", fileId)
		}
		if _, err := dataFile.ReadAt(output[outputSize:outputSize+length], int64(start)); err != nil {
			return err
		}
		outputSize += length
		return nil
	}
	// Read all the indexes in one go
	indices, err := t.getIndices(start, count)
	if err != nil {
		return nil, nil, err
	}
	var (
		sizes      []int               // The sizes for each element
		totalSize  = 0                 // The total size of all data read so far
		readStart  = indices[0].offset // Where, in the file, to start reading
		unreadSize = 0                 // The size of the as-yet-unread data
	)

	for i, firstIndex := range indices[:len(indices)-1] {
		secondIndex := indices[i+1]
		// Determine the size of the item.
		offset1, offset2, _ := firstIndex.bounds(secondIndex)
		size := int(offset2 - offset1)
		// Crossing a file boundary?
		if secondIndex.filenum != firstIndex.filenum {
			// If we have unread data in the first file, we need to do that read now.
			if unreadSize > 0 {
				if err := readData(firstIndex.filenum, readStart, unreadSize); err != nil {
					return nil, nil, err
				}
				unreadSize = 0
			}
			readStart = 0
		}
		if i > 0 && uint64(totalSize+size) > maxBytes {
			// About to break out due to byte limit being exceeded. We don't
			// read this last item, but we need to do the deferred reads now.
			if unreadSize > 0 {
				if err := readData(secondIndex.filenum, readStart, unreadSize); err != nil {
					return nil, nil, err
				}
			}
			break
		}
		// Defer the read for later
		unreadSize += size
		totalSize += size
		sizes = append(sizes, size)
		if i == len(indices)-2 || uint64(totalSize) > maxBytes {
			// Last item, need to do the read now
			if err := readData(secondIndex.filenum, readStart, unreadSize); err != nil {
				return nil, nil, err
			}
			break
		}
	}
	return output[:outputSize], sizes, nil
}

// has returns an indicator whether the specified number data is still accessible
// in the freezer table.
func (t *freezerTable) has(number uint64) bool {
	return atomic.LoadUint64(&t.items) > number && atomic.LoadUint64(&t.itemHidden) <= number
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

// advanceHead should be called when the current head file would outgrow the file limits,
// and a new file must be opened. The caller of this method must hold the write-lock
// before calling this method.
func (t *freezerTable) advanceHead() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// We open the next file in truncated mode -- if this file already
	// exists, we need to start over from scratch on it.
	nextID := t.headId + 1
	newHead, err := t.openFile(nextID, openFreezerFileTruncated)
	if err != nil {
		return err
	}

	// Close old file, and reopen in RDONLY mode.
	t.releaseFile(t.headId)
	t.openFile(t.headId, openFreezerFileForReadOnly)

	// Swap out the current head.
	t.head = newHead
	t.headBytes = 0
	t.headId = nextID
	return nil
}

// Sync pushes any pending data from memory out to disk. This is an expensive
// operation, so use it with care.
func (t *freezerTable) Sync() error {
	if err := t.index.Sync(); err != nil {
		return err
	}
	if err := t.meta.Sync(); err != nil {
		return err
	}
	return t.head.Sync()
}

// DumpIndex is a debug print utility function, mainly for testing. It can also
// be used to analyse a live freezer table index.
func (t *freezerTable) DumpIndex(start, stop int64) {
	t.dumpIndex(os.Stdout, start, stop)
}

func (t *freezerTable) dumpIndexString(start, stop int64) string {
	var out bytes.Buffer
	out.WriteString("\n")
	t.dumpIndex(&out, start, stop)
	return out.String()
}

func (t *freezerTable) dumpIndex(w io.Writer, start, stop int64) {
	meta, err := readMetadata(t.meta)
	if err != nil {
		fmt.Fprintf(w, "Failed to decode freezer table %v\n", err)
		return
	}
	fmt.Fprintf(w, "Version %d deleted %d, hidden %d\n", meta.Version, atomic.LoadUint64(&t.itemOffset), atomic.LoadUint64(&t.itemHidden))

	buf := make([]byte, indexEntrySize)

	fmt.Fprintf(w, "| number | fileno | offset |\n")
	fmt.Fprintf(w, "|--------|--------|--------|\n")

	for i := uint64(start); ; i++ {
		if _, err := t.index.ReadAt(buf, int64((i+1)*indexEntrySize)); err != nil {
			break
		}
		var entry indexEntry
		entry.unmarshalBinary(buf)
		fmt.Fprintf(w, "|  %03d   |  %03d   |  %03d   | \n", i, entry.filenum, entry.offset)
		if stop > 0 && i >= uint64(stop) {
			break
		}
	}
	fmt.Fprintf(w, "|--------------------------|\n")
}
