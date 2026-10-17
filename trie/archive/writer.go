// Copyright 2026 go-ethereum Authors
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

package archive

import (
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/rlp"
)

// ArchiveWriter is an append-only writer for archive files.
// It writes RLP-encoded records to a file and tracks the current offset.
type ArchiveWriter struct {
	file   *os.File
	offset uint64
	mu     sync.Mutex
}

// NewArchiveWriter creates a new archive writer that appends to the given file.
// If the file exists, it will be opened in append mode and writing continues
// from the current end of file. If it doesn't exist, it will be created.
func NewArchiveWriter(path string) (*ArchiveWriter, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	return &ArchiveWriter{
		file:   file,
		offset: uint64(info.Size()),
	}, nil
}

// WriteSubtree writes all records belonging to a subtree and returns
// the starting offset and total size of the written data.
// This is the atomic unit of archival - all records for a subtree are
// written together and can be retrieved together using the returned
// offset and size.
func (w *ArchiveWriter) WriteSubtree(records []*Record) (offset uint64, size uint64, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	startOffset := w.offset
	for _, rec := range records {
		encoded, err := rlp.EncodeToBytes(rec)
		if err != nil {
			return 0, 0, err
		}
		if _, err := w.file.Write(encoded); err != nil {
			return 0, 0, err
		}
		w.offset += uint64(len(encoded))
	}
	return startOffset, w.offset - startOffset, nil
}

// Sync flushes the file to disk. This should be called after writing
// a subtree and before modifying the database to ensure crash consistency.
func (w *ArchiveWriter) Sync() error {
	return w.file.Sync()
}

// Close closes the archive file.
func (w *ArchiveWriter) Close() error {
	return w.file.Close()
}

// Offset returns the current write offset in the file.
func (w *ArchiveWriter) Offset() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.offset
}
