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

// Package vectordb provides the vector database implementation.
package vectordb

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	// The current database version.
	currentVersion = 1

	// The file used to store the database metadata.
	metadataFile = "METADATA"
	// The file used to store the rawIndex of items stored in the database.
	indexFile = "INDEX"
	// The file used to store the rawData contained in the database.
	dataFile = "DATA"

	// Size of a serialized rawIndex entry. This should be updated if any changes
	// are made to indexEntry.
	indexEntryLen = 16 // 2 * sizeof(uint64)

	// File permissions.
	dbDirPerm        = 0755
	indexFilePerm    = 0644
	dataFilePerm     = 0644
	metadataFilePerm = 0644
)

var (
	dataFileFlags     = os.O_APPEND | os.O_CREATE | os.O_RDWR
	indexFileFlags    = os.O_APPEND | os.O_CREATE | os.O_RDWR
	metadataFileFlags = os.O_CREATE | os.O_RDWR

	// errClosed is returned if an operation attempts to manipulate the
	// database after it has been closed.
	errClosed = errors.New("vector database already closed")
)

// A VectorDB is a rawData store for storing sequences of binary blobs.
//
// Items are sequentially added and removed from the VectorDB, but
// provides random access to the elements contained within its bounds.
type VectorDB struct {
	// The path the database lives at.
	path string
	// Metadata about the database instance
	metadata *metadata
	// The number of items stored in the database.
	items uint64
	// Mutex protecting the rawData file descriptors
	lock sync.RWMutex
	// The file used to rawIndex the content in the rawData file.
	index *os.File
	// The file used to store rawData.
	data *os.File
}

// metadata contains information about the database.
type metadata struct {
	version uint64
}

// indexEntry contains the metadata associated with a stored rawData item.
type indexEntry struct {
	// The position the rawData item starts at within the rawData file.
	offset uint64
	// The length of the rawData item in the rawData file.
	length uint64
}

// indexOffset returns the file offset that corresponse to the blob
// located at the specified position in the database.
func indexOffset(pos int64) int64 {
	return pos * indexEntryLen
}

// unmarshallBinary deserializes binary b into the rawIndex entry.
func (e *indexEntry) unmarshalBinary(b []byte) error {
	e.offset = binary.BigEndian.Uint64(b[:8])
	e.length = binary.BigEndian.Uint64(b[8:16])
	return nil
}

// marshallBinary serializes the rawIndex entry into binary.
func (e *indexEntry) marshallBinary() []byte {
	b := make([]byte, indexEntryLen)
	binary.BigEndian.PutUint64(b[:8], e.offset)
	binary.BigEndian.PutUint64(b[8:16], e.length)
	return b
}

// Open opens a database instance with the specified name
// at the provided path.
func Open(name, path string) (*VectorDB, error) {
	databasePath := filepath.Join(path, name)
	fi, err := os.Stat(databasePath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(databasePath, dbDirPerm); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("open %q: not a directory", databasePath)
	}

	metadata, err := getOrCreateMetadataFile(path)
	if err != nil {
		return nil, err
	}
	index, err := os.OpenFile(filepath.Join(databasePath, indexFile), indexFileFlags, indexFilePerm)
	if err != nil {
		return nil, err
	}
	data, err := os.OpenFile(filepath.Join(databasePath, dataFile), dataFileFlags, dataFilePerm)
	if err != nil {
		return nil, err
	}

	db := &VectorDB{
		path:     databasePath,
		metadata: metadata,
		index:    index,
		data:     data,
	}

	if err := db.repair(); err != nil {
		return nil, err
	}

	return db, nil
}

func getOrCreateMetadataFile(path string) (*metadata, error) {
	metadataFilePath := filepath.Join(path, metadataFile)
	b, err := ioutil.ReadFile(metadataFilePath)
	if err == nil {
		var metadata metadata
		if err := json.Unmarshal(b, &metadata); err != nil {
			return nil, err
		}
		return &metadata, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	metadata := metadata{currentVersion}
	b, err = json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile(metadataFilePath, b, metadataFilePerm)
	return &metadata, nil
}

// Version returns the current database version.
func (db *VectorDB) Version() uint64 {
	return db.metadata.version
}

// Get retrieves the bytes stored at specified position pos.
func (db *VectorDB) Get(pos uint64) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if err := db.checkIsOpen(); err != nil {
		return nil, err
	}

	if err := db.checkBounds(pos); err != nil {
		return nil, err
	}

	entry, err := db.indexEntry(pos)
	if err != nil {
		return nil, err
	}

	b := make([]byte, entry.length)
	if _, err := db.data.ReadAt(b, int64(entry.offset)); err != nil {
		return nil, err
	}

	return b, nil
}

func (db *VectorDB) indexEntry(pos uint64) (*indexEntry, error) {
	b := make([]byte, indexEntryLen)
	_, err := db.index.ReadAt(b, indexOffset(int64(pos)))
	if err != nil {
		return nil, err
	}

	entry := new(indexEntry)
	if err := entry.unmarshalBinary(b); err != nil {
		return nil, err
	}

	return entry, nil
}

// Append adds the specified blob to the end of the database which should
// correspond to the specified pos, which is included as a precaution.
//
// The result of this operation is not guarranteed to be persisted until
// Sync() is called.
func (db *VectorDB) Append(pos uint64, blob []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if err := db.checkIsOpen(); err != nil {
		return err
	}

	if pos != db.items {
		return fmt.Errorf("proposed append position %d does not match current append position %d", pos, db.items)
	}

	offset, err := db.dataFileSize()
	if err != nil {
		return err
	}

	if _, err := db.data.Write(blob); err != nil {
		return err
	}

	entry := &indexEntry{uint64(offset), uint64(len(blob))}
	if _, err := db.index.Write(entry.marshallBinary()); err != nil {
		return err
	}

	db.items++
	return nil
}

func (db *VectorDB) dataFileSize() (int64, error) {
	fi, err := db.data.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// Truncate shortens the database to the desired length items.
//
// The result of this operation is not guarranteed to be persisted until
// Sync() is called.
func (db *VectorDB) Truncate(len uint64) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if err := db.checkIsOpen(); err != nil {
		return err
	}

	if err := db.checkBounds(len); err != nil {
		return err
	}

	db.items = len

	newIndexFileSize := len * indexEntryLen
	if err := db.truncateIndexFile(newIndexFileSize); err != nil {
		return err
	}

	lastEntry, err := db.indexEntry(len - 1)
	if err != nil {
		return err
	}

	newDataFileSize := lastEntry.offset + lastEntry.length
	if err := db.truncateDataFile(newDataFileSize); err != nil {
		return err
	}

	return nil
}

func (db *VectorDB) truncateIndexFile(size uint64) error {
	return db.index.Truncate(int64(size))
}

func (db *VectorDB) truncateDataFile(size uint64) error {
	return db.data.Truncate(int64(size))
}

// Items returns the length of the database as the number of entries it contains.
func (db *VectorDB) Items() uint64 {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.items
}

// Close closes the database.
func (db *VectorDB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if err := db.checkIsOpen(); err != nil {
		return err
	}

	db.items = 0
	if err := db.sync(); err != nil {
		return err
	}

	var errs []error
	if err := db.index.Close(); err != nil {
		errs = append(errs, fmt.Errorf("error closing rawIndex file: %v", err))
	}
	db.index = nil

	if err := db.data.Close(); err != nil {
		errs = append(errs, fmt.Errorf("error closing rawData file: %v", err))
	}
	db.data = nil

	if len(errs) > 0 {
		return fmt.Errorf("error closing vector database: %v", errs)
	}

	return nil
}

// Sync pushes any pending rawData from memory out to disk.
//
// Note: This is an expensive operation, so use it with care.
func (db *VectorDB) Sync() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.sync()
}

// sync is the non-thread safe version of Sync.
func (db *VectorDB) sync() error {
	if err := db.checkIsOpen(); err != nil {
		return err
	}

	// Commit rawData before updating indexes.
	if err := db.data.Sync(); err != nil {
		return err
	}

	if err := db.index.Sync(); err != nil {
		return err
	}

	return nil
}

func (db *VectorDB) checkIsOpen() error {
	if db.index == nil || db.data == nil {
		return errClosed
	}

	return nil
}

func (db *VectorDB) checkBounds(pos uint64) error {
	if pos >= db.items {
		return fmt.Errorf("position out of range (%d >= %d)", pos, db.items)
	}

	return nil
}

func (db *VectorDB) repair() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	indexFileSize, err := fileSize(db.index)
	if err != nil {
		return err
	}

	overflow := indexFileSize % indexEntryLen
	indexFileSize -= overflow
	if overflow > 0 {
		if err := db.index.Truncate(indexFileSize); err != nil {
			return err
		}
	}

	dataFileSize, err := fileSize(db.data)
	if err != nil {
		return err
	}

	items := uint64(indexFileSize / indexEntryLen)
	// Rewind until data file is consistent with what is reported in the index file.
	for items > 0 {
		entry, err := db.indexEntry(items - 1)
		if err != nil {
			return err
		}

		// Very likely the index and data files are consistent.
		if entry.offset+entry.length == uint64(dataFileSize) {
			break
		}

		// The index file is ahead of the data file.
		if entry.offset+entry.length > uint64(dataFileSize) {
			indexFileSize -= indexEntryLen
			if err := db.index.Truncate(indexFileSize); err != nil {
				return err
			}
			items--
			break
		}

		// The last blob in the data file must be corrupt.
		dataFileSize = int64(entry.offset + entry.length)
		if err := db.data.Truncate(dataFileSize); err != nil {
			return err
		}
	}

	db.items = items
	return nil
}

func fileSize(file *os.File) (int64, error) {
	fi, err := file.Stat()
	if err != nil {
		return -1, err
	}

	return fi.Size(), nil
}
