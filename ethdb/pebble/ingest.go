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
	"cmp"
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
const ingestDirName = "geth-ingest"

// errIngestRangeDeletion is returned if a range deletion is requested in the
// ingest batch.
var errIngestRangeDeletion = errors.New("range deletion is not supported in ingest batch")

// ingestOp is an operation recorded in the ingest batch, the key and value
// are referenced by the offsets in the shared buffer.
type ingestOp struct {
	key    int  // Start offset of the key
	value  int  // Start offset of the value
	end    int  // End offset of the value
	delete bool // Marker if the op refers to a delete
}

// ingestBatch is a write-only batch whose accumulated data is written into the
// sstables and ingested into the database as a whole when Write is called.
//
// Each SST is synced before Pebble atomically installs the files.
//
// A batch cannot be used concurrently.
type ingestBatch struct {
	db  *Database
	buf []byte
	ops []ingestOp
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
	return nil
}

// DeleteRange is not supported by the ingest batch.
func (b *ingestBatch) DeleteRange(start, end []byte) error {
	return errIngestRangeDeletion
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *ingestBatch) ValueSize() int {
	return len(b.buf)
}

// Write ingests the accumulated data into the database.
func (b *ingestBatch) Write() (err error) {
	b.db.quitLock.RLock()
	defer b.db.quitLock.RUnlock()

	if b.db.closed {
		return pebble.ErrClosed
	}
	if b.db.ingestOpts.ReadOnly {
		return pebble.ErrReadOnly
	}
	if len(b.ops) == 0 {
		return nil
	}
	paths, err := b.writeTables()
	defer func() {
		if err != nil {
			for _, path := range paths {
				os.Remove(path)
			}
		}
	}()
	if err != nil {
		return err
	}
	// Callers must serialize ingestion with overlapping ordinary writes when
	// their order matters. Use a single ordinary batch for writes that must
	// commit together. Call SyncKeyValue before ingestion if earlier ordinary
	// writes must already be durable.
	return b.db.db.Ingest(context.Background(), paths)
}

func (b *ingestBatch) sortedOps(yield func(ingestOp) bool) {
	// Strictly ordered batches can be read in ascending key order without
	// sorting. Repeated keys use the same fallback as unordered batches.
	direction := -1
	for i := 1; i < len(b.ops); i++ {
		prev, cur := b.ops[i-1], b.ops[i]
		order := bytes.Compare(b.buf[prev.key:prev.value], b.buf[cur.key:cur.value])
		if order == 0 || (i > 1 && order != direction) {
			direction = 0
			break
		}
		direction = order
	}
	switch direction {
	case -1:
		for _, op := range b.ops {
			if !yield(op) {
				return
			}
		}
		return
	case 1:
		for _, op := range slices.Backward(b.ops) {
			if !yield(op) {
				return
			}
		}
		return
	}
	// Sort indices instead of copying the operations: Replay preserves their
	// original order, and the final operation for a key must win.
	order := make([]int, len(b.ops))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(i, j int) int {
		x, y := b.ops[i], b.ops[j]
		return cmp.Or(bytes.Compare(b.buf[x.key:x.value], b.buf[y.key:y.value]), cmp.Compare(i, j))
	})
	for i, idx := range order {
		op := b.ops[idx]
		if i+1 < len(order) {
			next := b.ops[order[i+1]]
			if bytes.Equal(b.buf[op.key:op.value], b.buf[next.key:next.value]) {
				continue
			}
		}
		if !yield(op) {
			return
		}
	}
}

// writeTables writes the sorted operations into the sstables split by the target
// size of the level 0 files, returning the paths of the sstables created.
func (b *ingestBatch) writeTables() ([]string, error) {
	if err := os.MkdirAll(b.db.ingestDir, 0755); err != nil {
		return nil, err
	}
	var (
		paths  []string
		writer *sstable.Writer
		size   int
		opts   = b.db.ingestOpts.MakeWriterOptions(0, b.db.db.FormatMajorVersion().MaxTableFormat())
	)
	for op := range b.sortedOps {
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
		var (
			err error
			key = b.buf[op.key:op.value]
		)
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
