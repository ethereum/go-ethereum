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
	"sync"

	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/lru"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/rlp"
)

// ResolverFn is a callback to resolve expired nodes from an archive file.
// Given an offset and size, it returns the serialized node data from the archive.
type ResolverFn func(offset, size uint64) ([]*Record, error)

// OffsetSize is the size of the file offset in bytes.
const OffsetSize = 8

var (
	EmptyArchiveRecord = errors.New("empty record")                             // The archive contained a size-zero record.
	ErrNoResolver      = errors.New("no archive resolver set for expired node") // An expired node is accessed without a resolver.
)

// Record contains an archive file record. It is not the most optimal
// structure, since any modification to it will need to be overwritten.
type Record struct {
	Path  []byte
	Value []byte
}

// ArchiveDataDir is the data directory where the archive file is stored.
var ArchiveDataDir string

// recordsCacheSize bounds the number of decoded subtree record groups kept
// in memory. Groups average a few KB, so this stays in the tens of MB.
const recordsCacheSize = 8192

var (
	readerMu   sync.Mutex
	readerFile *os.File // persistent read handle to the archive file
	readerDir  string   // ArchiveDataDir the handle was opened under

	// recordsCache caches decoded record groups keyed by file offset. The
	// archive file is append-only, so an offset uniquely identifies its
	// content and cached entries can never go stale within one file.
	recordsCache = lru.NewCache[uint64, []*Record](recordsCacheSize)
)

// archiveReader returns a shared read handle to the archive file, opening it
// on first use. If ArchiveDataDir changed since the last open (tests, or
// re-configuration), the stale handle is closed and the records cache is
// purged, since offsets from different files would otherwise collide.
func archiveReader() (*os.File, error) {
	readerMu.Lock()
	defer readerMu.Unlock()
	if readerFile != nil && readerDir == ArchiveDataDir {
		return readerFile, nil
	}
	if readerFile != nil {
		readerFile.Close()
		readerFile = nil
		recordsCache.Purge()
	}
	file, err := os.Open(filepath.Join(ArchiveDataDir, "geth", "nodearchive"))
	if err != nil {
		return nil, fmt.Errorf("error opening archive file: %w", err)
	}
	readerFile = file
	readerDir = ArchiveDataDir
	return file, nil
}

// ArchivedNodeResolver takes a buffer containing the archive data
// held by an expiring node (an offset and a size) and returns a
// list of records, which is a list of serialized leaf nodes. The
// caller knows the context (MPT, binary trie) and is responsible
// for decoding the nodes.
//
// Records are read through a shared pread handle (no per-call open/seek)
// and cached by offset; callers MUST treat the returned records and their
// byte slices as immutable.
func ArchivedNodeResolver(offset, size uint64) ([]*Record, error) {
	// Resolve the handle FIRST: archiveReader purges the offset-keyed cache
	// whenever ArchiveDataDir changed, so the cache must not be consulted
	// before that check (offsets from different files would collide).
	file, err := archiveReader()
	if err != nil {
		return nil, err
	}
	if records, ok := recordsCache.Get(offset); ok {
		return records, nil
	}
	data := make([]byte, size)
	if _, err := file.ReadAt(data, int64(offset)); err != nil {
		return nil, fmt.Errorf("error reading data from archive: %w", err)
	}
	var records []*Record
	stream := rlp.NewStream(bytes.NewReader(data), uint64(len(data)))
	for {
		var record Record
		if err := stream.Decode(&record); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("error decoding rlp record from archive data (offset=%d, size=%d): %w", offset, size, err)
		}
		records = append(records, &record)
	}
	recordsCache.Add(offset, records)
	return records, nil
}
