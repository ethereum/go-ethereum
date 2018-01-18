// Copyright 2017 The go-ethereum Authors
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

package hashtree

// Package hashtree defines a general storage model for evolving tree-hashed data
// structures and implements garbage collection that removes elements which were only
// referenced by old versions of the structure that are no longer necessary to store.
//
// The storage model requires a definition of the data structure that assigns a position
// to each hashed element. The format of the position is defined by the data structure.
// A function is required that can tell for each (version, position, hash) tuple whether
// the given hashed element is part of the given version of the structure at the given
// position.
//
// Each version of the structure is identified by its root hash and also has a version
// number. Garbage collection can delete all elements that are only referenced in versions
// with a version number lower than a certain value ("GC version"). The evolution of the
// structure can be rolled back and version numbers can be reused but no rollback is
// allowed at or below the GC version.
//
// When writing a new version to the hash tree storage, each element not present in its
// parent version has to be written with the new version number. Elements are stored in
// the backing database in the following format:
//
//  position + hash + []byte{0} -> data
//  position + hash + version (uint64 big endian) -> NULL
//
// Reads only access the data entry, write operations always add the later (reference)
// entry too.

import (
	"encoding/binary"
	"sync/atomic"
	//	"fmt"
)

type DatabaseReader interface {
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
}

type DatabaseWriter interface {
	Put([]byte, []byte) error
}

// Reader provides read access to the hash tree storage
type Reader struct {
	db     DatabaseReader
	prefix []byte
	lpf    int
}

func NewReader(db DatabaseReader, prefix string) *Reader {
	return &Reader{db, []byte(prefix), len(prefix)}
}

// Get returns elements by position and hash
func (h *Reader) Get(position, hash []byte) ([]byte, error) {
	lp, lh := len(position), len(hash)
	key := make([]byte, h.lpf+lp+lh+1)
	copy(key[:h.lpf], h.prefix)
	copy(key[h.lpf:h.lpf+lp], position)
	copy(key[h.lpf+lp:h.lpf+lp+lh], hash)
	data, err := h.db.Get(key)
	if err != nil {
		//panic(nil)
		//fmt.Printf("READ ERR  %x  %v\n", key, err)
	}
	return data, err
}

func (h *Reader) Has(position, hash []byte) (bool, error) {
	lp, lh := len(position), len(hash)
	key := make([]byte, h.lpf+lp+lh+1)
	copy(key[:h.lpf], h.prefix)
	copy(key[h.lpf:h.lpf+lp], position)
	copy(key[h.lpf+lp:h.lpf+lp+lh], hash)
	return h.db.Has(key)
}

// Put should never be used, Reader still implements r/w database interfaces for convenient use with tries
func (h *Reader) Put(position, hash, data []byte) error {
	panic(nil)
}

// Writer provides write access to the hash tree storage. A new writer is required for each new version.
type Writer struct {
	db         DatabaseWriter
	prefix     []byte
	lpf        int
	version    uint64
	versionEnc [8]byte
	gc         *GarbageCollector
}

func NewWriter(db DatabaseWriter, prefix string, version uint64, gc *GarbageCollector) *Writer {
	w := &Writer{
		db:      db,
		prefix:  []byte(prefix),
		lpf:     len(prefix),
		version: version,
		gc:      gc,
	}
	binary.BigEndian.PutUint64(w.versionEnc[:], version)
	return w
}

// Put adds an element and a version reference entry to the hash tree
func (w *Writer) Put(position, hash, data []byte) error {
	if w.gc != nil {
		atomic.AddUint64(&w.gc.writeCounter, 1)
	}
	lp, lh := len(position), len(hash)
	key := make([]byte, w.lpf+lp+lh+9)
	copy(key[:w.lpf], w.prefix)
	copy(key[w.lpf:w.lpf+lp], position)
	copy(key[w.lpf+lp:w.lpf+lp+lh], hash)
	if err := w.db.Put(key[:w.lpf+lp+lh+1], data); err != nil {
		return err
	}
	copy(key[w.lpf+lp+lh:w.lpf+lp+lh+8], w.versionEnc[:])
	key[w.lpf+lp+lh+8] = 1
	return w.db.Put(key, nil)
}
