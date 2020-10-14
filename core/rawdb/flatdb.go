// Copyright 2020 The go-ethereum Authors
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
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	temporaryName = "tmp.db"
	syncedName    = "flat.db"
	indexName     = "flat.index"
	bufferGrowRec = 3000
	chunkSize     = 4 * 1024 * 1024
)

var (
	ErrReadOnly     = errors.New("read only")
	ErrWriteOnly    = errors.New("write only")
	ErrWriteFailure = errors.New("write failure")
	ErrReadFailure  = errors.New("read failure")
	ErrEmptyEntry   = errors.New("empty entry")
)

// FlatDatabase is the "database" based on the raw file. It implements the
// ethdb.KeyValueStore interfaces which can act as the kv database in some
// special scenarios which don't require random read but only write appendly
// and iteration(with writing order).
//
// All items stored in the flatDB will be marshalled in this format:
//
//   +------------+-----+--------------+-------+
//   | Key Length | Key | Value Length | Value |
//   +------------+-----+--------------+-------+
//
// The flatDB can only be opened for read only mode(iteration) or write only
// mode. Each write operation will append the blob into the file with or without
// sync operation. But in order to make the flat database readable, it should
// call Commit after all write opts and after that the db is not writable.
type FlatDatabase struct {
	lock      sync.Mutex
	path      string   // The directory for the flat database
	data      *os.File // File descriptor for the flat database.
	index     *os.File // File descriptor for the indexes.
	read      bool     // Indicator whether the db is read or write mode
	buff      []byte   // Auxiliary buffer for storing uncommitted data
	items     int      // Auxiliary number for counting uncommitted data
	iterating bool     // Indicator whether the db is iterating. Concurrent iteration is not supported
	offset    uint64   // Global offset of entry in the file
}

func NewFlatDatabase(path string, read bool) (*FlatDatabase, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	var (
		data  *os.File
		index *os.File
		err   error
	)
	if read {
		data, err = os.OpenFile(filepath.Join(path, syncedName), os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		index, err = os.OpenFile(filepath.Join(path, indexName), os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = os.OpenFile(filepath.Join(path, temporaryName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		index, err = os.OpenFile(filepath.Join(path, indexName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
	}
	return &FlatDatabase{
		path:  path,
		data:  data,
		index: index,
		read:  read,
	}, nil
}

// Has retrieves if a key is present in the flat data store.
func (db *FlatDatabase) Has(key []byte) (bool, error) { panic("not supported") }

// Get retrieves the given key if it's present in the flat data store.
func (db *FlatDatabase) Get(key []byte) ([]byte, error) { panic("not supported") }

// Delete removes the key from the key-value data store.
func (db *FlatDatabase) Delete(key []byte) error { panic("not supported") }

// Stat returns a particular internal stat of the database.
func (db *FlatDatabase) Stat(property string) (string, error) { panic("not supported") }

// Compact flattens the underlying data store for the given key range. In essence,
// deleted and overwritten versions are discarded, and the data is rearranged to
// reduce the cost of operations needed to access them.
//
// A nil start is treated as a key before all keys in the data store; a nil limit
// is treated as a key after all keys in the data store. If both is nil then it
// will compact entire data store.
func (db *FlatDatabase) Compact(start []byte, limit []byte) error { panic("not supported") }

// Put inserts the given value into the key-value data store.
func (db *FlatDatabase) Put(key []byte, value []byte) error {
	if len(key) == 0 || len(value) == 0 {
		return ErrEmptyEntry
	}
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.read {
		return ErrReadOnly
	}
	n := 2*binary.MaxVarintLen32 + len(key) + len(value)
	db.grow(n)
	offset, previous := len(db.buff), len(db.buff)
	db.buff = db.buff[:offset+n]
	offset += binary.PutUvarint(db.buff[offset:], uint64(len(key)))
	offset += copy(db.buff[offset:], key)
	offset += binary.PutUvarint(db.buff[offset:], uint64(len(value)))
	offset += copy(db.buff[offset:], value)
	db.buff = db.buff[:offset]
	db.items += 1

	// db.offset is monotonic increasing in "WRITE" mode which
	// indicates the offset of the last written entry in GLOBAL
	// view. So everytime only the diff is added.
	db.offset += uint64(offset) - uint64(previous)
	return db.writeChunk(false)
}

func (db *FlatDatabase) grow(n int) {
	o := len(db.buff)
	if cap(db.buff)-o < n {
		div := 1
		if db.items > bufferGrowRec {
			div = db.items / bufferGrowRec
		}
		ndata := make([]byte, o, o+n+o/div)
		copy(ndata, db.buff)
		db.buff = ndata
	}
}

func (db *FlatDatabase) writeChunk(force bool) error {
	if len(db.buff) < chunkSize && !force {
		return nil
	}
	// Step one, flush data
	n, err := db.data.Write(db.buff)
	if err != nil {
		return err
	}
	if n != len(db.buff) {
		return ErrWriteFailure
	}
	db.buff = db.buff[:0]
	db.items = 0

	// Step two, flush chunk offset
	var local [8]byte
	binary.BigEndian.PutUint64(local[:], db.offset)
	n, err = db.index.Write(local[:])
	if err != nil {
		return err
	}
	if n != 8 {
		return ErrWriteFailure
	}
	return nil
}

func (db *FlatDatabase) readChunk() error {
	// Step one, read chunk size
	var local [8]byte
	n, err := db.index.Read(local[:])
	if err != nil {
		return err // may return EOF
	}
	if n != 8 {
		return ErrReadFailure
	}
	offset := binary.BigEndian.Uint64(local[:])
	size := int(offset - db.offset)
	db.offset = offset

	db.grow(size)
	db.buff = db.buff[:size]
	n, err = db.data.Read(db.buff)
	if err != nil {
		return err // may return EOF
	}
	if n != size {
		return ErrReadFailure
	}
	return nil
}

// Commit flushs all in-memory data into the disk and switchs the db to read mode.
func (db *FlatDatabase) Commit() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if err := db.closeNoLock(); err != nil {
		return err
	}
	if err := rename(filepath.Join(db.path, temporaryName), filepath.Join(db.path, syncedName)); err != nil {
		return err
	}
	if err := syncDir(db.path); err != nil {
		return err
	}
	db.read = true
	db.offset = 0

	// Reopen the files in read-only mode
	var err error
	db.data, err = os.OpenFile(filepath.Join(db.path, syncedName), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	db.index, err = os.OpenFile(filepath.Join(db.path, indexName), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (db *FlatDatabase) closeNoLock() error {
	if err := db.writeChunk(true); err != nil {
		return err
	}
	if err := db.data.Sync(); err != nil {
		return err
	}
	if err := db.index.Sync(); err != nil {
		return err
	}
	if err := db.data.Close(); err != nil {
		return err
	}
	if err := db.index.Close(); err != nil {
		return err
	}
	return nil
}

func (db *FlatDatabase) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.closeNoLock()
}

// NewBatch creates a write-only database that buffers changes to its host db
// until a final write is called.
func (db *FlatDatabase) NewBatch() ethdb.Batch {
	return &flatBatch{db: db}
}

type flatBatch struct {
	db      *FlatDatabase
	keys    [][]byte
	vals    [][]byte
	keysize int
	valsize int
	lock    sync.RWMutex
}

// Put inserts the given value into the key-value data store.
func (fb *flatBatch) Put(key []byte, value []byte) error {
	fb.lock.Lock()
	defer fb.lock.Unlock()

	fb.keys = append(fb.keys, key)
	fb.vals = append(fb.vals, value)
	fb.keysize += len(key)
	fb.valsize += len(value)
	return nil
}

// Delete removes the key from the key-value data store.
func (fb *flatBatch) Delete(key []byte) error { panic("not supported") }

// ValueSize retrieves the amount of data queued up for writing.
func (fb *flatBatch) ValueSize() int {
	fb.lock.RLock()
	defer fb.lock.RUnlock()

	return fb.valsize
}

// Write flushes any accumulated data to disk.
func (fb *flatBatch) Write() error {
	fb.lock.Lock()
	defer fb.lock.Unlock()

	for i := 0; i < len(fb.keys); i++ {
		if err := fb.db.Put(fb.keys[i], fb.vals[i]); err != nil {
			return err
		}
	}
	return nil
}

// Reset resets the batch for reuse.
func (fb *flatBatch) Reset() {
	fb.lock.Lock()
	defer fb.lock.Unlock()

	fb.keysize, fb.valsize = 0, 0
	fb.keys = fb.keys[:0]
	fb.vals = fb.vals[:0]
}

// Replay replays the batch contents.
func (fb *flatBatch) Replay(w ethdb.KeyValueWriter) error {
	fb.lock.Lock()
	defer fb.lock.Unlock()

	for i := 0; i < len(fb.keys); i++ {
		if err := w.Put(fb.keys[i], fb.vals[i]); err != nil {
			return err
		}
	}
	return nil
}

// NewIterator creates a iterator over the **whole** database with first-in-first-out
// order. The passed `prefix` and `start` is useless, just only to follow the interface.
//
// If there already exists a un-released iterator, the nil will be returned since
// iteration concurrently is not supported by flatdb.
func (db *FlatDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.iterating {
		return nil
	}
	db.iterating = true
	db.data.Seek(0, 0)
	db.index.Seek(0, 0)
	db.offset = 0
	db.buff = db.buff[:0]
	return &flatIterator{db: db}
}

// flatIterator is the iterator used to itearate the whole db.
type flatIterator struct {
	db  *FlatDatabase
	key []byte
	val []byte
	err error
	eof bool
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (iter *flatIterator) Next() bool {
	if len(iter.db.buff) == 0 && !iter.eof {
		if err := iter.db.readChunk(); err != nil {
			if err == io.EOF {
				iter.eof = true
				return false
			} else {
				iter.err = err
				return false
			}
		}
	}
	var offset int
	x, n := binary.Uvarint(iter.db.buff)
	offset += n
	if n <= 0 {
		return false
	}
	key := iter.db.buff[offset : offset+int(x)]
	offset += int(x)
	x, n = binary.Uvarint(iter.db.buff[offset:])
	offset += n
	if n <= 0 {
		return false
	}
	val := iter.db.buff[offset : offset+int(x)]
	offset += int(x)

	iter.key = key
	iter.val = val
	iter.db.buff = iter.db.buff[offset:]
	return true
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error.
func (iter *flatIterator) Error() error {
	return iter.err
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (iter *flatIterator) Key() []byte {
	return iter.key
}

// Value returns the value of the current key/value pair, or nil if done. The
// caller should not modify the contents of the returned slice, and its contents
// may change on the next call to Next.
func (iter *flatIterator) Value() []byte {
	return iter.val
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (iter *flatIterator) Release() {
	iter.db.iterating = false
}

// HasAncient returns an indicator whether the specified data exists in the
// ancient store.
func (db *FlatDatabase) HasAncient(kind string, number uint64) (bool, error) { panic("not supported") }

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (db *FlatDatabase) Ancient(kind string, number uint64) ([]byte, error) { panic("not supported") }

// Ancients returns the ancient item numbers in the ancient store.
func (db *FlatDatabase) Ancients() (uint64, error) { panic("not supported") }

// AncientSize returns the ancient size of the specified category.
func (db *FlatDatabase) AncientSize(kind string) (uint64, error) { panic("not supported") }

// AppendAncient injects all binary blobs belong to block at the end of the
// append-only immutable table files.
func (db *FlatDatabase) AppendAncient(number uint64, hash, header, body, receipt, td []byte) error {
	panic("not supported")
}

// TruncateAncients discards all but the first n ancient data from the ancient store.
func (db *FlatDatabase) TruncateAncients(n uint64) error { panic("not supported") }

// Sync flushes all in-memory ancient store data to disk.
func (db *FlatDatabase) Sync() error { panic("not supported") }
