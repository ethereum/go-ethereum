// Copyright 2016 The go-ethereum Authors
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

// +build !windows

// Package rocksdb contains the RocksDB based database storage engine.
package rocksdb

import (
	"errors"
	"reflect"
	"runtime"
	"unsafe"

	_ "github.com/cockroachdb/c-rocksdb" // Placeholder package for CGO wrapper
	"github.com/ethereum/go-ethereum/ethdb"
)

// #include <stdlib.h>
// #include "rocksdb/c.h"
// #cgo CXXFLAGS: -std=c++11
// #cgo CPPFLAGS: -I ../../Godeps/_workspace/src/github.com/cockroachdb/c-rocksdb/internal/include
// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"

// Database is a RocksDB backed key/value store.
type Database struct {
	storage   *C.struct_rocksdb_t              // RocksDB database instance
	readOpts  *C.struct_rocksdb_readoptions_t  // Default read options to pass to get
	writeOpts *C.struct_rocksdb_writeoptions_t // Default write options to pass to put
}

// New returns a RocksDB backed implementation of the ethdb.Database interface.
func New(dir string, cache uint64, handles int) (ethdb.Database, error) {
	// Create the database tuning options
	blockOptions := C.rocksdb_block_based_options_create()
	C.rocksdb_block_based_options_set_block_cache(blockOptions, C.rocksdb_cache_create_lru(C.size_t(cache/2))) // Use half of the allowed cache for blocks

	dbOptions := C.rocksdb_options_create()
	C.rocksdb_options_set_create_if_missing(dbOptions, C.uchar(1))                       // Open or create the database
	C.rocksdb_options_set_write_buffer_size(dbOptions, C.size_t(cache/4))                // Allocate part of the cache to write buffers
	C.rocksdb_options_set_max_write_buffer_number(dbOptions, C.int(2))                   // Number of write buffer to use to accumulate changes
	C.rocksdb_options_set_min_write_buffer_number_to_merge(dbOptions, C.int(1))          // Number of write buffers to merge before flushing changes
	C.rocksdb_options_set_max_open_files(dbOptions, C.int(handles))                      // Limit the number of open files (1 file / 2 MB is needed)
	C.rocksdb_options_set_max_background_compactions(dbOptions, C.int(runtime.NumCPU())) // Allow full database compaction concurrency
	C.rocksdb_options_set_block_based_table_factory(dbOptions, blockOptions)             // Set the block tuning parameters

	// Open the RocksDB database via the C library
	cdir := C.CString(dir)
	defer C.free(unsafe.Pointer(cdir))

	var cerr *C.char
	storage := C.rocksdb_open(dbOptions, cdir, &cerr)
	if cerr != nil {
		defer C.free(unsafe.Pointer(cerr))
		return nil, errors.New(C.GoString(cerr))
	}
	// Assemble the RocksDB database wrapper
	return &Database{
		storage:   storage,
		readOpts:  C.rocksdb_readoptions_create(),
		writeOpts: C.rocksdb_writeoptions_create(),
	}, nil
}

// Put inserts the given key/value tuple into the database.
func (db *Database) Put(key []byte, value []byte) error {
	var (
		cerr   *C.char
		ckey   = (*C.char)(unsafe.Pointer(&key[0]))
		cvalue = (*C.char)(unsafe.Pointer(&value[0]))
	)
	C.rocksdb_put(db.storage, db.writeOpts, ckey, C.size_t(len(key)), cvalue, C.size_t(len(value)), &cerr)
	if cerr != nil {
		defer C.free(unsafe.Pointer(cerr))
		return errors.New(C.GoString(cerr))
	}
	return nil
}

// Get retrieves the value of the given key if it exists.
func (db *Database) Get(key []byte) ([]byte, error) {
	var (
		cerr    *C.char
		cvallen C.size_t
	)
	// Execute the database value retrieval
	ckey := (*C.char)(unsafe.Pointer(&key[0]))
	cvalue := C.rocksdb_get(db.storage, db.readOpts, ckey, C.size_t(len(key)), &cvallen, &cerr)
	if cerr != nil {
		defer C.free(unsafe.Pointer(cerr))
		return nil, errors.New(C.GoString(cerr))
	}
	// Move the resulting value back into a Go slice
	var value []byte
	sH := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	sH.Cap, sH.Len, sH.Data = int(cvallen), int(cvallen), uintptr(unsafe.Pointer(cvalue))
	return value, nil
}

// Delete removes the key from the database if it exists.
func (db *Database) Delete(key []byte) error {
	var (
		cerr *C.char
		ckey = (*C.char)(unsafe.Pointer(&key[0]))
	)
	C.rocksdb_delete(db.storage, db.writeOpts, ckey, C.size_t(len(key)), &cerr)
	if cerr != nil {
		defer C.free(unsafe.Pointer(cerr))
		return errors.New(C.GoString(cerr))
	}
	return nil
}

// Close closes the database by deallocating the underlying handle.
func (db *Database) Close() error {
	C.rocksdb_close(db.storage)
	return nil
}

// NewBatch returns a new batch wrapping this RocksDB database.
func (db *Database) NewBatch() ethdb.Batch {
	return &Batch{
		storage: db.storage,
		batch:   C.rocksdb_writebatch_create(),
		opts:    db.writeOpts,
	}
}

// Batch is a write collector wrapping a RocksDB database.
type Batch struct {
	storage *C.struct_rocksdb_t              // RocksDB database instance
	batch   *C.rocksdb_writebatch_t          // RocksDB batch to accumulate pending writes
	opts    *C.struct_rocksdb_writeoptions_t // Default write options to use during commit
}

// Put inserts the given key/value tuple into the batch.
func (b *Batch) Put(key, value []byte) error {
	ckey := (*C.char)(unsafe.Pointer(&key[0]))
	cvalue := (*C.char)(unsafe.Pointer(&value[0]))

	C.rocksdb_writebatch_put(b.batch, ckey, C.size_t(len(key)), cvalue, C.size_t(len(value)))
	return nil
}

// Commit atomically applies any batched updates to the underlying database.
func (b *Batch) Commit() error {
	defer C.rocksdb_writebatch_destroy(b.batch)

	var cerr *C.char
	C.rocksdb_write(b.storage, b.opts, b.batch, &cerr)
	if cerr != nil {
		defer C.free(unsafe.Pointer(cerr))
		return errors.New(C.GoString(cerr))
	}
	return nil
}
