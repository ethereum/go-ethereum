// Copyright 2024 The go-ethereum Authors
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
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// memoryTable is used to store a list of sequential items in memory.
type memoryTable struct {
	name   string   // Table name
	items  uint64   // Number of stored items in the table, including the deleted ones
	offset uint64   // Number of deleted items from the table
	data   [][]byte // List of rlp-encoded items, sort in order
	size   uint64   // Total memory size occupied by the table
	lock   sync.RWMutex
}

// newMemoryTable initializes the memory table.
func newMemoryTable(name string) *memoryTable {
	return &memoryTable{name: name}
}

// has returns an indicator whether the specified data exists.
func (t *memoryTable) has(number uint64) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return number >= t.offset && number < t.items
}

// retrieve retrieves multiple items in sequence, starting from the index 'start'.
// It will return:
//   - at most 'count' items,
//   - if maxBytes is specified: at least 1 item (even if exceeding the maxByteSize),
//     but will otherwise return as many items as fit into maxByteSize.
//   - if maxBytes is not specified, 'count' items will be returned if they are present
func (t *memoryTable) retrieve(start uint64, count, maxBytes uint64) ([][]byte, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var (
		size  uint64
		batch [][]byte
	)
	// Ensure the start is written, not deleted from the tail, and that the
	// caller actually wants something.
	if t.items <= start || t.offset > start || count == 0 {
		return nil, errOutOfBounds
	}
	// Cap the item count if the retrieval is out of bound.
	if start+count > t.items {
		count = t.items - start
	}
	for n := start; n < start+count; n++ {
		index := n - t.offset
		if len(batch) != 0 && maxBytes != 0 && size+uint64(len(t.data[index])) > maxBytes {
			return batch, nil
		}
		batch = append(batch, t.data[index])
		size += uint64(len(t.data[index]))
	}
	return batch, nil
}

// truncateHead discards any recent data above the provided threshold number.
func (t *memoryTable) truncateHead(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Short circuit if nothing to delete.
	if t.items <= items {
		return nil
	}
	if items < t.offset {
		return errors.New("truncation below tail")
	}
	t.data = t.data[:items-t.offset]
	t.items = items
	return nil
}

// truncateTail discards any recent data before the provided threshold number.
func (t *memoryTable) truncateTail(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Short circuit if nothing to delete.
	if t.offset >= items {
		return nil
	}
	if t.items < items {
		return errors.New("truncation above head")
	}
	t.data = t.data[items-t.offset:]
	t.offset = items
	return nil
}

// commit merges the given item batch into table. It's presumed that the
// batch is ordered and continuous with table.
func (t *memoryTable) commit(batch [][]byte) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, item := range batch {
		t.size += uint64(len(item))
	}
	t.data = append(t.data, batch...)
	t.items += uint64(len(batch))
	return nil
}

// memoryBatch is the singleton batch used for ancient write.
type memoryBatch struct {
	data map[string][][]byte
	next map[string]uint64
	size map[string]int64
}

func newMemoryBatch() *memoryBatch {
	return &memoryBatch{
		data: make(map[string][][]byte),
		next: make(map[string]uint64),
		size: make(map[string]int64),
	}
}

func (b *memoryBatch) reset(freezer *MemoryFreezer) {
	b.data = make(map[string][][]byte)
	b.next = make(map[string]uint64)
	b.size = make(map[string]int64)

	for name, table := range freezer.tables {
		b.next[name] = table.items
	}
}

// Append adds an RLP-encoded item.
func (b *memoryBatch) Append(kind string, number uint64, item interface{}) error {
	if b.next[kind] != number {
		return errOutOrderInsertion
	}
	blob, err := rlp.EncodeToBytes(item)
	if err != nil {
		return err
	}
	b.data[kind] = append(b.data[kind], blob)
	b.next[kind]++
	b.size[kind] += int64(len(blob))
	return nil
}

// AppendRaw adds an item without RLP-encoding it.
func (b *memoryBatch) AppendRaw(kind string, number uint64, blob []byte) error {
	if b.next[kind] != number {
		return errOutOrderInsertion
	}
	b.data[kind] = append(b.data[kind], common.CopyBytes(blob))
	b.next[kind]++
	b.size[kind] += int64(len(blob))
	return nil
}

// commit is called at the end of a write operation and writes all remaining
// data to tables.
func (b *memoryBatch) commit(freezer *MemoryFreezer) (items uint64, writeSize int64, err error) {
	// Check that count agrees on all batches.
	items = math.MaxUint64
	for name, next := range b.next {
		if items < math.MaxUint64 && next != items {
			return 0, 0, fmt.Errorf("table %s is at item %d, want %d", name, next, items)
		}
		items = next
	}
	// Commit all table batches.
	for name, batch := range b.data {
		table := freezer.tables[name]
		if err := table.commit(batch); err != nil {
			return 0, 0, err
		}
		writeSize += b.size[name]
	}
	return items, writeSize, nil
}

// MemoryFreezer is an ephemeral ancient store. It implements the ethdb.AncientStore
// interface and can be used along with ephemeral key-value store.
type MemoryFreezer struct {
	items      uint64                  // Number of items stored
	tail       uint64                  // Number of the first stored item in the freezer
	readonly   bool                    // Flag if the freezer is only for reading
	lock       sync.RWMutex            // Lock to protect fields
	tables     map[string]*memoryTable // Tables for storing everything
	writeBatch *memoryBatch            // Pre-allocated write batch
}

// NewMemoryFreezer initializes an in-memory freezer instance.
func NewMemoryFreezer(readonly bool, tableName map[string]bool) *MemoryFreezer {
	tables := make(map[string]*memoryTable)
	for name := range tableName {
		tables[name] = newMemoryTable(name)
	}
	return &MemoryFreezer{
		writeBatch: newMemoryBatch(),
		readonly:   readonly,
		tables:     tables,
	}
}

// HasAncient returns an indicator whether the specified data exists.
func (f *MemoryFreezer) HasAncient(kind string, number uint64) (bool, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	if table := f.tables[kind]; table != nil {
		return table.has(number), nil
	}
	return false, nil
}

// Ancient retrieves an ancient binary blob from the in-memory freezer.
func (f *MemoryFreezer) Ancient(kind string, number uint64) ([]byte, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	t := f.tables[kind]
	if t == nil {
		return nil, errUnknownTable
	}
	data, err := t.retrieve(number, 1, 0)
	if err != nil {
		return nil, err
	}
	return data[0], nil
}

// AncientRange retrieves multiple items in sequence, starting from the index 'start'.
// It will return
//   - at most 'count' items,
//   - if maxBytes is specified: at least 1 item (even if exceeding the maxByteSize),
//     but will otherwise return as many items as fit into maxByteSize.
//   - if maxBytes is not specified, 'count' items will be returned if they are present
func (f *MemoryFreezer) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	t := f.tables[kind]
	if t == nil {
		return nil, errUnknownTable
	}
	return t.retrieve(start, count, maxBytes)
}

// Ancients returns the ancient item numbers in the freezer.
func (f *MemoryFreezer) Ancients() (uint64, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.items, nil
}

// Tail returns the number of first stored item in the freezer.
// This number can also be interpreted as the total deleted item numbers.
func (f *MemoryFreezer) Tail() (uint64, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.tail, nil
}

// AncientSize returns the ancient size of the specified category.
func (f *MemoryFreezer) AncientSize(kind string) (uint64, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	if table := f.tables[kind]; table != nil {
		return table.size, nil
	}
	return 0, errUnknownTable
}

// ReadAncients runs the given read operation while ensuring that no writes take place
// on the underlying freezer.
func (f *MemoryFreezer) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return fn(f)
}

// ModifyAncients runs the given write operation.
func (f *MemoryFreezer) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (writeSize int64, err error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.readonly {
		return 0, errReadOnly
	}
	// Roll back all tables to the starting position in case of error.
	defer func(old uint64) {
		if err == nil {
			return
		}
		// The write operation has failed. Go back to the previous item position.
		for name, table := range f.tables {
			err := table.truncateHead(old)
			if err != nil {
				log.Error("Freezer table roll-back failed", "table", name, "index", old, "err", err)
			}
		}
	}(f.items)

	// Modify the ancients in batch.
	f.writeBatch.reset(f)
	if err := fn(f.writeBatch); err != nil {
		return 0, err
	}
	item, writeSize, err := f.writeBatch.commit(f)
	if err != nil {
		return 0, err
	}
	f.items = item
	return writeSize, nil
}

// TruncateHead discards any recent data above the provided threshold number.
// It returns the previous head number.
func (f *MemoryFreezer) TruncateHead(items uint64) (uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.readonly {
		return 0, errReadOnly
	}
	old := f.items
	if old <= items {
		return old, nil
	}
	for _, table := range f.tables {
		if err := table.truncateHead(items); err != nil {
			return 0, err
		}
	}
	f.items = items
	return old, nil
}

// TruncateTail discards any recent data below the provided threshold number.
func (f *MemoryFreezer) TruncateTail(tail uint64) (uint64, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.readonly {
		return 0, errReadOnly
	}
	old := f.tail
	if old >= tail {
		return old, nil
	}
	for _, table := range f.tables {
		if err := table.truncateTail(tail); err != nil {
			return 0, err
		}
	}
	f.tail = tail
	return old, nil
}

// Sync flushes all data tables to disk.
func (f *MemoryFreezer) Sync() error {
	return nil
}

// Close releases all the sources held by the memory freezer. It will panic if
// any following invocation is made to a closed freezer.
func (f *MemoryFreezer) Close() error {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.tables = nil
	f.writeBatch = nil
	return nil
}

// Reset drops all the data cached in the memory freezer and reset itself
// back to default state.
func (f *MemoryFreezer) Reset() error {
	f.lock.Lock()
	defer f.lock.Unlock()

	tables := make(map[string]*memoryTable)
	for name := range f.tables {
		tables[name] = newMemoryTable(name)
	}
	f.tables = tables
	f.items, f.tail = 0, 0
	return nil
}
