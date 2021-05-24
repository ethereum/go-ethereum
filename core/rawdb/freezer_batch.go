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

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/golang/snappy"
	"sync/atomic"
)

type freezerBatch struct {
	t         *freezerTable
	data      []byte
	startItem uint64
	count     uint64
	filenum   uint32
	sizes     []uint32

	headBytes uint32
}

func (t *freezerTable) NewBatch() *freezerBatch {
	return &freezerBatch{
		t:         t,
		data:      nil,
		startItem: math.MaxUint64,
		filenum:   t.headId, // TODO: Needs rlock
		count:     0,
		headBytes: 0,
	}
}

func (batch *freezerBatch) Append(item uint64, blob []byte) error {
	if batch.startItem == math.MaxUint64 {
		batch.startItem = item
	}
	if !batch.t.noCompression {
		blob = snappy.Encode(nil, blob)
	}
	bLen := len(blob)
	batch.data = append(batch.data, blob...)
	batch.sizes = append(batch.sizes, uint32(bLen))
	batch.count++
	return nil
}

func (batch *freezerBatch) Write() error {
	var (
		retry = false
		err   error
	)
	for {
		retry, err = batch.write(retry)
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
	}
}

func (batch *freezerBatch) write(newHead bool) (bool, error) {
	if !newHead {
		batch.t.lock.RLock()
		defer batch.t.lock.RUnlock()
	} else {
		batch.t.lock.Lock()
		defer batch.t.lock.Unlock()
	}
	if batch.t.index == nil || batch.t.head == nil {
		return false, errClosed
	}
	// Ensure we're in sync with the data
	if atomic.LoadUint64(&batch.t.items) != batch.startItem {
		return false, fmt.Errorf("appending unexpected item: want %d, have %d", batch.t.items, batch.startItem)
	}
	if newHead {
		if err := batch.t.advanceHead(); err != nil {
			return false, err
		}
		// And update the batch to point to the new file
		batch.headBytes = 0
		batch.filenum = atomic.LoadUint32(&batch.t.headId)
	}
	var indexData = make([]byte, 0, len(batch.sizes)*indexEntrySize)
	var count uint64
	var writtenDataSize int
	for _, size := range batch.sizes {
		if batch.headBytes+size <= batch.t.maxFileSize {
			writtenDataSize += int(size)
			idx := indexEntry{
				filenum: batch.filenum,
				offset:  batch.headBytes + size,
			}
			batch.headBytes += size
			idxData := idx.marshallBinary()
			indexData = append(indexData, idxData...)
		} else {
			// Writing will overflow, need to chunk up the batch into several writes
			break
		}
		count++
	}
	if writtenDataSize == 0 {
		return batch.count > 0, nil
	}
	// Write the actual data
	if _, err := batch.t.head.Write(batch.data[:writtenDataSize]); err != nil {
		return false, err
	}
	// Write the new indexdata
	if _, err := batch.t.index.Write(indexData); err != nil {
		return false, err
	}
	batch.t.writeMeter.Mark(int64(len(batch.data)) + int64(batch.count)*int64(indexEntrySize))
	batch.t.sizeGauge.Inc(int64(len(batch.data)) + int64(batch.count)*int64(indexEntrySize))
	atomic.AddUint64(&batch.t.items, count)
	batch.startItem += count
	batch.count -= count

	if batch.count > 0 {
		// Some data left to write on a retry.
		batch.data = batch.data[writtenDataSize:]
		batch.sizes = batch.sizes[count:]
		return true, nil
	}
	// All data written. We can simply truncate and keep using the buffer
	batch.data = batch.data[:0]
	batch.sizes = batch.sizes[:0]
	return false, nil
}
