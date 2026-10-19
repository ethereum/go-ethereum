// Copyright 2026 The go-ethereum Authors
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

package pebble

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/objstorage/objstorageprovider"
	"github.com/cockroachdb/pebble/v2/sstable"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/ethereum/go-ethereum/ethdb"
)

// ingestDirName is the name of the directory within the database for holding
// the sstables to be ingested. It's located in the database directory so that
// the sstables can be hard linked into the database.
const ingestDirName = "ingest"

// errIngestRangeDeletion is returned if a range deletion is requested in the
// ingest batch.
var errIngestRangeDeletion = errors.New("range deletion is not supported in ingest batch")

// ingestOp is an operation recorded in the ingest batch, the key and value
// are referenced by the offsets in the shared buffer.
type ingestOp struct {
	key    int // Start offset of the key
	value  int // Start offset of the value, which is also the end of the key
	end    int // End offset of the value
	delete bool
}

// ingestBatch is a write-only batch whose accumulated data is written into the
// sstables and ingested into the database as a whole when Write is called.
//
// Unlike the normal batch, the data is not written into the write-ahead log, so
// that the large write doesn't block the subsequent database writes until done.
// The ingestion is atomic and durable once Write returns.
//
// A batch cannot be used concurrently.
type ingestBatch struct {
	db   *Database
	buf  []byte
	ops  []ingestOp
	size int
}

// NewIngestBatch implements ethdb.Ingester, creating a write-only batch with
// pre-allocated buffer, whose accumulated data is ingested into the database
// as a whole.
func (d *Database) NewIngestBatch(size int) ethdb.Batch {
	return &ingestBatch{
		db:  d,
		buf: make([]byte, 0, size),
	}
}

// Put inserts the given value into the batch for later committing.
func (b *ingestBatch) Put(key, value []byte) error {
	start := len(b.buf)
	b.buf = append(b.buf, key...)
	b.buf = append(b.buf, value...)
	b.ops = append(b.ops, ingestOp{
		key:   start,
		value: start + len(key),
		end:   len(b.buf),
	})
	b.size += len(key) + len(value)
	return nil
}

// Delete inserts the key removal into the batch for later committing.
func (b *ingestBatch) Delete(key []byte) error {
	start := len(b.buf)
	b.buf = append(b.buf, key...)
	b.ops = append(b.ops, ingestOp{
		key:    start,
		value:  len(b.buf),
		end:    len(b.buf),
		delete: true,
	})
	b.size += len(key)
	return nil
}

// DeleteRange is not supported by the ingest batch.
func (b *ingestBatch) DeleteRange(start, end []byte) error {
	return errIngestRangeDeletion
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *ingestBatch) ValueSize() int {
	return b.size
}

// Write ingests the accumulated data into the database.
func (b *ingestBatch) Write() error {
	b.db.quitLock.RLock()
	defer b.db.quitLock.RUnlock()

	if b.db.closed {
		return pebble.ErrClosed
	}
	if len(b.ops) == 0 {
		return nil
	}
	// Sort the operations by key, the sstables require the keys in order. The
	// sort is stable, so that the latest operation of a key takes precedence.
	ops := slices.Clone(b.ops)
	slices.SortStableFunc(ops, func(x, y ingestOp) int {
		return bytes.Compare(b.buf[x.key:x.value], b.buf[y.key:y.value])
	})
	paths, err := b.writeTables(ops)
	if err != nil {
		for _, path := range paths {
			os.Remove(path)
		}
		return err
	}
	// Sync the write-ahead log before the ingestion, ensuring all the writes
	// committed prior are persisted ahead of the ingested data, which becomes
	// durable once ingested.
	if err := b.db.db.LogData(nil, pebble.Sync); err != nil {
		return err
	}
	// The ingested sstables are linked into the database and the original ones
	// are removed afterwards.
	if err := b.db.db.Ingest(context.Background(), paths); err != nil {
		for _, path := range paths {
			os.Remove(path)
		}
		return err
	}
	return nil
}

// writeTables writes the sorted operations into the sstables split by the target
// size of the level 0 files, returning the paths of the sstables created.
func (b *ingestBatch) writeTables(ops []ingestOp) ([]string, error) {
	if err := os.MkdirAll(b.db.ingestDir, 0755); err != nil {
		return nil, err
	}
	var (
		paths  []string
		writer *sstable.Writer
		size   int
		opts   = b.db.ingestOpts.MakeWriterOptions(0, b.db.db.FormatMajorVersion().MaxTableFormat())
	)
	for i, op := range ops {
		key := b.buf[op.key:op.value]

		// Skip the operation superseded by the following one of the same key
		if i+1 < len(ops) && bytes.Equal(key, b.buf[ops[i+1].key:ops[i+1].value]) {
			continue
		}
		if writer == nil {
			path := filepath.Join(b.db.ingestDir, fmt.Sprintf("%d.sst", b.db.ingestSeq.Add(1)))
			paths = append(paths, path)

			f, err := vfs.Default.Create(path, vfs.WriteCategoryUnspecified)
			if err != nil {
				return paths, err
			}
			writer = sstable.NewWriter(objstorageprovider.NewFileWritable(f), opts)
			size = 0
		}
		var err error
		if op.delete {
			err = writer.Delete(key)
		} else {
			err = writer.Set(key, b.buf[op.value:op.end])
		}
		if err != nil {
			writer.Close()
			return paths, err
		}
		size += op.end - op.key

		// Close the sstable once it reaches the target size, which also syncs
		// the file content.
		if size >= int(b.db.ingestOpts.TargetFileSizes[0]) {
			if err := writer.Close(); err != nil {
				return paths, err
			}
			writer = nil
		}
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			return paths, err
		}
	}
	return paths, nil
}

// Reset resets the batch for reuse.
func (b *ingestBatch) Reset() {
	b.buf = b.buf[:0]
	b.ops = b.ops[:0]
	b.size = 0
}

// Replay replays the batch contents in the order of the operations.
func (b *ingestBatch) Replay(w ethdb.KeyValueWriter) error {
	for _, op := range b.ops {
		var err error
		if op.delete {
			err = w.Delete(b.buf[op.key:op.value])
		} else {
			err = w.Put(b.buf[op.key:op.value], b.buf[op.value:op.end])
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes the batch and releases all associated resources.
func (b *ingestBatch) Close() {
	b.buf, b.ops = nil, nil
}
