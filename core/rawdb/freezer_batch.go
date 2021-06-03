// Copyright 2021 The go-ethereum Authors
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
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/rlp"
)

// freezerBatch is a write operation of multiple items on a freezer.
type freezerBatch struct {
	freezer *freezer
	tables  map[string]*freezerTableBatch
}

func newFreezerBatch(f *freezer) *freezerBatch {
	batch := &freezerBatch{
		freezer: f,
		tables:  make(map[string]*freezerTableBatch, len(f.tables)),
	}
	for kind, table := range f.tables {
		batch.tables[kind] = table.newBatch()
	}
	return batch
}

// Append adds an RLP-encoded item of the given kind.
func (batch *freezerBatch) Append(kind string, num uint64, item interface{}) error {
	return batch.tables[kind].Append(num, item)
}

// AppendRaw adds an item of the given kind.
func (batch *freezerBatch) AppendRaw(kind string, num uint64, item []byte) error {
	return batch.tables[kind].AppendRaw(num, item)
}

func (batch *freezerBatch) Commit() error {
	// Check that count agrees on all batches.
	item := uint64(math.MaxUint64)
	count := 0
	for name, tb := range batch.tables {
		if item < math.MaxUint64 && tb.curItem != item {
			return fmt.Errorf("batch %s is at item %d, want %d", name, tb.curItem, item)
		}
		item = tb.curItem
		count++
	}

	// Commit all table batches.
	for _, tb := range batch.tables {
		if err := tb.Commit(); err != nil {
			return err
		}
	}

	// Bump frozen block index.
	atomic.AddUint64(&batch.freezer.frozen, uint64(count))
	return nil
}

const (
	freezerBatchBufferLimit = 2 * 1024 * 1024
)

// freezerTableBatch is a batch for a freezer table.
type freezerTableBatch struct {
	t *freezerTable

	sb          *BufferedSnapWriter
	encBuffer   writeBuffer
	dataBuffer  []byte
	indexBuffer []byte

	curItem   uint64
	headBytes uint32 // number of bytes in head file.
}

// newBatch creates a new batch for the freezer table.
func (t *freezerTable) newBatch() *freezerTableBatch {
	batch := &freezerTableBatch{t: t}
	batch.Reset()
	return batch
}

// Reset clears the batch for reuse.
func (batch *freezerTableBatch) Reset() {
	if !batch.t.noCompression {
		batch.sb = new(BufferedSnapWriter)
	} else {
		batch.sb = nil
	}
	batch.encBuffer.Reset()
	batch.dataBuffer = batch.dataBuffer[:0]
	batch.indexBuffer = batch.indexBuffer[:0]
	batch.curItem = atomic.LoadUint64(&batch.t.items)
	batch.headBytes = batch.t.headBytes
}

// Append rlp-encodes and adds data at the end of the freezer table. The item number is a
// precautionary parameter to ensure data correctness, but the table will reject already
// existing data.
func (batch *freezerTableBatch) Append(item uint64, data interface{}) error {
	if item != batch.curItem {
		return fmt.Errorf("appending unexpected item: want %d, have %d", batch.curItem, item)
	}

	// Encode the item.
	batch.encBuffer.Reset()
	if batch.sb != nil {
		// RLP-encode
		if err := rlp.Encode(batch.sb, data); err != nil {
			return err
		}
		// Snappy-encode to our buf
		if err := batch.sb.WriteTo(&batch.encBuffer); err != nil {
			return err
		}
	} else {
		if err := rlp.Encode(&batch.encBuffer, data); err != nil {
			return err
		}
	}

	return batch.appendItem(batch.encBuffer.data)
}

// AppendRaw injects a binary blob at the end of the freezer table. The item number is a
// precautionary parameter to ensure data correctness, but the table will reject already
// existing data.
func (batch *freezerTableBatch) AppendRaw(item uint64, blob []byte) error {
	if item != batch.curItem {
		return fmt.Errorf("appending unexpected item: want %d, have %d", batch.curItem, item)
	}

	data := blob
	if batch.sb != nil {
		batch.encBuffer.Reset()
		batch.sb.WriteDirectTo(&batch.encBuffer, blob)
		data = batch.encBuffer.data
	}
	return batch.appendItem(data)
}

func (batch *freezerTableBatch) appendItem(data []byte) error {
	itemSize := uint32(len(data))

	// Check if item fits into current data file.
	if batch.headBytes+itemSize > batch.t.maxFileSize {
		// It doesn't fit, go to next file first.
		if err := batch.commit(); err != nil {
			return err
		}
		if err := batch.t.advanceHead(); err != nil {
			return err
		}
		batch.headBytes = 0
	}

	// Put data to buffer.
	batch.dataBuffer = append(batch.dataBuffer, data...)

	// Put index entry to buffer.
	batch.headBytes += itemSize
	entry := indexEntry{filenum: batch.t.headId, offset: batch.headBytes}
	batch.indexBuffer = entry.append(batch.indexBuffer)
	batch.curItem++

	return batch.maybeCommit()
}

// Commit writes the batched items to the backing freezerTable.
func (batch *freezerTableBatch) Commit() error {
	if err := batch.commit(); err != nil {
		return err
	}
	atomic.StoreUint64(&batch.t.items, batch.curItem)

	// TODO: update the head bytes of the table
	batch.t.headBytes = batch.headBytes
	return nil
}

// maybeCommit writes the buffered data if the buffer is full enough.
func (batch *freezerTableBatch) maybeCommit() error {
	if len(batch.dataBuffer) > freezerBatchBufferLimit {
		return batch.commit()
	}
	return nil
}

func (batch *freezerTableBatch) commit() error {
	// Write data.
	_, err := batch.t.head.Write(batch.dataBuffer)
	if err != nil {
		return err
	}
	batch.dataBuffer = batch.dataBuffer[:0]

	// Write index.
	_, err = batch.t.index.Write(batch.indexBuffer)
	if err != nil {
		return err
	}
	batch.indexBuffer = batch.indexBuffer[:0]
	return nil
}
