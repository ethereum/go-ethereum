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

package freezer

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/edsrzf/mmap-go"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/golang/snappy"
)

const (
	growthIndex uint64 = 1024 * 1024       // Growth rate of the index file when remapping
	growthData  uint64 = 128 * 1024 * 1024 // Growth rate of the data file when remapping
)

var (
	// errAlreadyExists is returned if the user attempts to append an item to the
	// freezer table that already exists (i.e. it's offset is lower or equal to
	// the head of the immutable file).
	errAlreadyExists = errors.New("item already exists")

	// errGappedWrite is returned if the user attempts to append an item to the
	// freezer table that would produce a data gap (i.e. it's offset is larger than
	// the head of the immutable file).
	errGappedWrite = errors.New("item produces data gap")

	// errTableInaccessible is returned if a previously well functioning freezer
	// table becomes non-accessible after a memory remap (resize).
	errTableInaccessible = errors.New("table not accessible")

	// errOutOfBounds is returned if the item requested is not contained within the
	// freezer table.
	errOutOfBounds = errors.New("out of bounds")
)

// table represents a single chained data table within the freezer (e.g. blocks).
// It consists of a data file (snappy encoded arbitrary data blobs) and an index
// file (uncompressed 64 bit indices into the data file). In addition a counter
// (binary) file is also created which simply contains the number of items.
type table struct {
	path string // Database folder to store the files into
	name string // Table name to multiplex multiple tables into the same folder

	fileData    *os.File // File descriptor for the data region
	fileIndex   *os.File // File descriptor for the offset region
	fileCounter *os.File // File descriptor for the counter region

	mmapData    mmap.MMap // Memory mapping for the data region
	mmapIndex   mmap.MMap // Memory mapping for the offset region
	mmapCounter mmap.MMap // Memory mapping for the counter region

	rawData    []byte   // Direct memory region for the data file
	rawIndex   []uint64 // Direct memory region for the offset file
	rawCounter []uint64 // Direct memory region for the counter (single item)

	readMeter  metrics.Meter // Meter for measuring the effective amount of data read
	writeMeter metrics.Meter // Meter for measuring the effective amount of data written

	logger log.Logger   // Logger with database path and table name ambedded
	lock   sync.RWMutex // Lock protecting the reads from remaps
}

// newTable attempts to new freezer table by memory mapping the composing files.
// If no file exists, it will create new ones.
func newTable(path string, name string, readMeter metrics.Meter, writeMeter metrics.Meter) (*table, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	tab := &table{
		path:       path,
		name:       name,
		readMeter:  readMeter,
		writeMeter: writeMeter,
		logger:     log.New("path", path, "table", name),
	}
	if err := tab.ensure(); err != nil {
		return nil, err
	}
	return tab, nil
}

// ensure checks all the memory region limits and ensures that they conform to
// the required data counts. If anything is off, this method will recreate and
// remap accordingly.
func (t *table) ensure() error {
	// Figure out if we need to remap or not
	// If the regions are already open and large enough, leave as is
	if t.rawCounter != nil {
		// Ensure the data file is large enough, unmap it otherwise
		wantData := (t.rawIndex[t.rawCounter[0]]/growthData + 1) * growthData
		if uint64(len(t.rawData)) < wantData {
			if err := t.mmapData.Unmap(); err != nil {
				t.logger.Error("Failed to unmap freezer datastore", "err", err)
			}
			if err := t.fileData.Close(); err != nil {
				t.logger.Error("Failed to close freezer datastore", "err", err)
			}
			t.fileData, t.mmapData, t.rawData = nil, nil, nil
		}
		// Ensure the index file is large enough, unmap it otherwise
		wantIndex := (((t.rawCounter[0]+1)*8)/growthIndex + 1) * growthIndex
		if uint64(len(t.rawIndex)) < wantIndex {
			if err := t.mmapIndex.Unmap(); err != nil {
				t.logger.Error("Failed to unmap freezer index", "err", err)
			}
			if err := t.fileIndex.Close(); err != nil {
				t.logger.Error("Failed to close freezer index", "err", err)
			}
			t.fileIndex, t.mmapIndex, t.rawIndex = nil, nil, nil
		}
	}
	// Memory map the counter and retrieve the size of the offset file
	var err error

	if t.rawCounter == nil {
		if t.fileCounter, t.mmapCounter, t.rawCounter, err = mmapUints(filepath.Join(t.path, t.name+".len"), 8); err != nil {
			return err
		}
	}
	// Memory map the index file and retrieve the size of the data file
	if t.rawIndex == nil {
		size := (((t.rawCounter[0]+1)*8)/growthIndex + 1) * growthIndex
		if t.fileIndex, t.mmapIndex, t.rawIndex, err = mmapUints(filepath.Join(t.path, t.name+".idx"), size); err != nil {
			if err := t.mmapCounter.Unmap(); err != nil {
				t.logger.Error("Failed to unmap freezer counter", "err", err)
			}
			if err := t.fileCounter.Close(); err != nil {
				t.logger.Error("Failed to close freezer counter", "err", err)
			}
			t.fileCounter, t.mmapCounter, t.rawCounter = nil, nil, nil
			return err
		}
	}
	// Memory map the data file and return the final freezer table
	if t.rawData == nil {
		size := growthData
		if t.rawCounter[0] > 0 {
			size = (t.rawIndex[t.rawCounter[0]]/growthData + 1) * growthData
		}
		if t.fileData, t.mmapData, t.rawData, err = mmapBytes(filepath.Join(t.path, t.name+".dat"), size); err != nil {
			if err := t.mmapIndex.Unmap(); err != nil {
				t.logger.Error("Failed to unmap freezer index", "err", err)
			}
			if err := t.fileIndex.Close(); err != nil {
				t.logger.Error("Failed to close freezer index", "err", err)
			}
			t.fileIndex, t.mmapIndex, t.rawIndex = nil, nil, nil

			if err := t.mmapCounter.Unmap(); err != nil {
				t.logger.Error("Failed to unmap freezer counter", "err", err)
			}
			if err := t.fileCounter.Close(); err != nil {
				t.logger.Error("Failed to close freezer counter", "err", err)
			}
			t.fileCounter, t.mmapCounter, t.rawCounter = nil, nil, nil

			return err
		}
	}
	return nil
}

// Close unmaps all active memory mapped regions.
func (t *table) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.mmapData != nil {
		if err := t.mmapData.Unmap(); err != nil {
			t.logger.Error("Failed to unmap freezer datastore", "err", err)
		}
		if err := t.fileData.Close(); err != nil {
			t.logger.Error("Failed to close freezer datastore", "err", err)
		}
		t.fileData, t.mmapData, t.rawData = nil, nil, nil
	}
	if t.mmapIndex != nil {
		if err := t.mmapIndex.Unmap(); err != nil {
			t.logger.Error("Failed to unmap freezer index", "err", err)
		}
		if err := t.fileIndex.Close(); err != nil {
			t.logger.Error("Failed to close freezer index", "err", err)
		}
		t.fileIndex, t.mmapIndex, t.rawIndex = nil, nil, nil
	}
	if t.mmapCounter != nil {
		if err := t.mmapCounter.Unmap(); err != nil {
			t.logger.Error("Failed to unmap freezer counter", "err", err)
		}
		if err := t.fileCounter.Close(); err != nil {
			t.logger.Error("Failed to close freezer counter", "err", err)
		}
		t.fileCounter, t.mmapCounter, t.rawCounter = nil, nil, nil
	}
	return nil
}

// Append injects a binary blob at the end of the freezer table. The item index
// is a precautionary parameter to ensure data correctness, but the table will
// reject already existing data.
//
// Note, this method will *not* flush any data to disk (unless the files need to
// be resized). Be sure to explicitly flush before irreversibly deleting data
// from the fast chain database.
func (t *table) Append(item uint64, blob []byte) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Ensure the table is still accessible
	if t.rawCounter == nil {
		if err := t.ensure(); err != nil {
			t.logger.Error("Failed to re-access table", "err", err)
			return errTableInaccessible
		}
	}
	// Ensure only the next item can be written, nothing else
	items := t.rawCounter[0]
	if items > item {
		return errAlreadyExists
	}
	if items < item {
		return errGappedWrite
	}
	// Encode the blob and write it into the data file
	blob = snappy.Encode(nil, blob)

	t.rawIndex[items+1] = t.rawIndex[items] + uint64(len(blob))
	copy(t.rawData[t.rawIndex[items]:], blob)
	t.rawCounter[0]++

	t.writeMeter.Mark(int64(len(blob)))

	// Ensure we have enough space for future appends
	return t.ensure()
}

// Retrieve looks up the data offset of an item with the given index and retrieves
// the raw binary blob from the data file.
func (t *table) Retrieve(item uint64) ([]byte, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// Ensure the table and the item is accessible
	if t.rawCounter == nil {
		return nil, errTableInaccessible
	}
	if t.rawCounter[0] <= item {
		return nil, errOutOfBounds
	}
	// Item reachable, retrive and return to the user
	blob := t.rawData[t.rawIndex[item]:t.rawIndex[item+1]]
	t.writeMeter.Mark(int64(len(blob)))

	return snappy.Decode(nil, blob)
}

// Flush pushes any pending data from memory out to disk. This is an expensive
// operation, so use it with care.
func (t *table) Flush() error {
	if err := t.mmapData.Flush(); err != nil {
		return err
	}
	if err := t.mmapIndex.Flush(); err != nil {
		return err
	}
	if err := t.mmapCounter.Flush(); err != nil {
		return err
	}
	return nil
}
