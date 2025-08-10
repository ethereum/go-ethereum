// Copyright 2025 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// parseIndex parses the index data with the supplied byte stream. The index data
// is a list of fixed-sized metadata. Empty metadata is regarded as invalid.
func parseIndex(blob []byte) ([]*indexBlockDesc, error) {
	if len(blob) == 0 {
		return nil, errors.New("empty state history index")
	}
	if len(blob)%indexBlockDescSize != 0 {
		return nil, fmt.Errorf("corrupted state index, len: %d", len(blob))
	}
	var (
		lastID   uint32
		descList []*indexBlockDesc
	)
	for i := 0; i < len(blob)/indexBlockDescSize; i++ {
		var desc indexBlockDesc
		desc.decode(blob[i*indexBlockDescSize : (i+1)*indexBlockDescSize])
		if desc.empty() {
			return nil, errors.New("empty state history index block")
		}
		if lastID != 0 {
			if lastID+1 != desc.id {
				return nil, fmt.Errorf("index block id is out of order, last-id: %d, this-id: %d", lastID, desc.id)
			}
			// Theoretically, order should be validated between consecutive index blocks,
			// ensuring that elements within them are strictly ordered. However, since
			// tracking the minimum element in each block has non-trivial storage overhead,
			// this check is optimistically omitted.
			//
			// TODO(rjl493456442) the minimal element can be resolved from the index block,
			// evaluate the check cost (mostly IO overhead).

			/*	if desc.min <= lastMax {
				return nil, fmt.Errorf("index block range is out of order, last-max: %d, this-min: %d", lastMax, desc.min)
			}*/
		}
		lastID = desc.id
		descList = append(descList, &desc)
	}
	return descList, nil
}

// indexReader is the structure to look up the state history index records
// associated with the specific state element.
type indexReader struct {
	db       ethdb.KeyValueReader
	descList []*indexBlockDesc
	readers  map[uint32]*blockReader
	state    stateIdent
}

// loadIndexData loads the index data associated with the specified state.
func loadIndexData(db ethdb.KeyValueReader, state stateIdent) ([]*indexBlockDesc, error) {
	var blob []byte
	if state.account {
		blob = rawdb.ReadAccountHistoryIndex(db, state.addressHash)
	} else {
		blob = rawdb.ReadStorageHistoryIndex(db, state.addressHash, state.storageHash)
	}
	if len(blob) == 0 {
		return nil, nil
	}
	return parseIndex(blob)
}

// newIndexReader constructs a index reader for the specified state. Reader with
// empty data is allowed.
func newIndexReader(db ethdb.KeyValueReader, state stateIdent) (*indexReader, error) {
	descList, err := loadIndexData(db, state)
	if err != nil {
		return nil, err
	}
	return &indexReader{
		descList: descList,
		readers:  make(map[uint32]*blockReader),
		db:       db,
		state:    state,
	}, nil
}

// refresh reloads the last section of index data to account for any additional
// elements that may have been written to disk.
func (r *indexReader) refresh() error {
	// Release the reader for the last section of index data, as its content
	// may have been modified by additional elements written to the disk.
	if len(r.descList) != 0 {
		last := r.descList[len(r.descList)-1]
		if !last.full() {
			delete(r.readers, last.id)
		}
	}
	descList, err := loadIndexData(r.db, r.state)
	if err != nil {
		return err
	}
	r.descList = descList
	return nil
}

// readGreaterThan locates the first element that is greater than the specified
// id. If no such element is found, MaxUint64 is returned.
func (r *indexReader) readGreaterThan(id uint64) (uint64, error) {
	index := sort.Search(len(r.descList), func(i int) bool {
		return id < r.descList[i].max
	})
	if index == len(r.descList) {
		return math.MaxUint64, nil
	}
	desc := r.descList[index]

	br, ok := r.readers[desc.id]
	if !ok {
		var (
			err  error
			blob []byte
		)
		if r.state.account {
			blob = rawdb.ReadAccountHistoryIndexBlock(r.db, r.state.addressHash, desc.id)
		} else {
			blob = rawdb.ReadStorageHistoryIndexBlock(r.db, r.state.addressHash, r.state.storageHash, desc.id)
		}
		br, err = newBlockReader(blob)
		if err != nil {
			return 0, err
		}
		r.readers[desc.id] = br
	}
	// The supplied ID is not greater than block.max, ensuring that an element
	// satisfying the condition can be found.
	return br.readGreaterThan(id)
}

// indexWriter is responsible for writing index data for a specific state (either
// an account or a storage slot). The state index follows a two-layer structure:
// the first layer consists of a list of fixed-size metadata, each linked to a
// second-layer block. The index data (monotonically increasing list of state
// history ids) is stored in these second-layer index blocks, which are size
// limited.
type indexWriter struct {
	descList []*indexBlockDesc // The list of index block descriptions
	bw       *blockWriter      // The live index block writer
	frozen   []*blockWriter    // The finalized index block writers, waiting for flush
	lastID   uint64            // The ID of the latest tracked history
	state    stateIdent
	db       ethdb.KeyValueReader
}

// newIndexWriter constructs the index writer for the specified state.
func newIndexWriter(db ethdb.KeyValueReader, state stateIdent) (*indexWriter, error) {
	var blob []byte
	if state.account {
		blob = rawdb.ReadAccountHistoryIndex(db, state.addressHash)
	} else {
		blob = rawdb.ReadStorageHistoryIndex(db, state.addressHash, state.storageHash)
	}
	if len(blob) == 0 {
		desc := newIndexBlockDesc(0)
		bw, _ := newBlockWriter(nil, desc)
		return &indexWriter{
			descList: []*indexBlockDesc{desc},
			bw:       bw,
			state:    state,
			db:       db,
		}, nil
	}
	descList, err := parseIndex(blob)
	if err != nil {
		return nil, err
	}
	var (
		indexBlock []byte
		lastDesc   = descList[len(descList)-1]
	)
	if state.account {
		indexBlock = rawdb.ReadAccountHistoryIndexBlock(db, state.addressHash, lastDesc.id)
	} else {
		indexBlock = rawdb.ReadStorageHistoryIndexBlock(db, state.addressHash, state.storageHash, lastDesc.id)
	}
	bw, err := newBlockWriter(indexBlock, lastDesc)
	if err != nil {
		return nil, err
	}
	return &indexWriter{
		descList: descList,
		lastID:   lastDesc.max,
		bw:       bw,
		state:    state,
		db:       db,
	}, nil
}

// append adds the new element into the index writer.
func (w *indexWriter) append(id uint64) error {
	if id <= w.lastID {
		return fmt.Errorf("append element out of order, last: %d, this: %d", w.lastID, id)
	}
	if w.bw.full() {
		if err := w.rotate(); err != nil {
			return err
		}
	}
	if err := w.bw.append(id); err != nil {
		return err
	}
	w.lastID = id

	return nil
}

// rotate creates a new index block for storing index records from scratch
// and caches the current full index block for finalization.
func (w *indexWriter) rotate() error {
	var (
		err  error
		desc = newIndexBlockDesc(w.bw.desc.id + 1)
	)
	w.frozen = append(w.frozen, w.bw)
	w.bw, err = newBlockWriter(nil, desc)
	if err != nil {
		return err
	}
	w.descList = append(w.descList, desc)
	return nil
}

// finish finalizes all the frozen index block writers along with the live one
// if it's not empty, committing the index block data and the index meta into
// the supplied batch.
//
// This function is safe to be called multiple times.
func (w *indexWriter) finish(batch ethdb.Batch) {
	var (
		writers  = append(w.frozen, w.bw)
		descList = w.descList
	)
	// The live index block writer might be empty if the entire index write
	// is created from scratch, remove it from committing.
	if w.bw.empty() {
		writers = writers[:len(writers)-1]
		descList = descList[:len(descList)-1]
	}
	if len(writers) == 0 {
		return // nothing to commit
	}
	for _, bw := range writers {
		if w.state.account {
			rawdb.WriteAccountHistoryIndexBlock(batch, w.state.addressHash, bw.desc.id, bw.finish())
		} else {
			rawdb.WriteStorageHistoryIndexBlock(batch, w.state.addressHash, w.state.storageHash, bw.desc.id, bw.finish())
		}
	}
	w.frozen = nil // release all the frozen writers

	buf := make([]byte, 0, indexBlockDescSize*len(descList))
	for _, desc := range descList {
		buf = append(buf, desc.encode()...)
	}
	if w.state.account {
		rawdb.WriteAccountHistoryIndex(batch, w.state.addressHash, buf)
	} else {
		rawdb.WriteStorageHistoryIndex(batch, w.state.addressHash, w.state.storageHash, buf)
	}
}

// indexDeleter is responsible for deleting index data for a specific state.
type indexDeleter struct {
	descList []*indexBlockDesc // The list of index block descriptions
	bw       *blockWriter      // The live index block writer
	dropped  []uint32          // The list of index block id waiting for deleting
	lastID   uint64            // The ID of the latest tracked history
	state    stateIdent
	db       ethdb.KeyValueReader
}

// newIndexDeleter constructs the index deleter for the specified state.
func newIndexDeleter(db ethdb.KeyValueReader, state stateIdent) (*indexDeleter, error) {
	var blob []byte
	if state.account {
		blob = rawdb.ReadAccountHistoryIndex(db, state.addressHash)
	} else {
		blob = rawdb.ReadStorageHistoryIndex(db, state.addressHash, state.storageHash)
	}
	if len(blob) == 0 {
		// TODO(rjl493456442) we can probably return an error here,
		// deleter with no data is meaningless.
		desc := newIndexBlockDesc(0)
		bw, _ := newBlockWriter(nil, desc)
		return &indexDeleter{
			descList: []*indexBlockDesc{desc},
			bw:       bw,
			state:    state,
			db:       db,
		}, nil
	}
	descList, err := parseIndex(blob)
	if err != nil {
		return nil, err
	}
	var (
		indexBlock []byte
		lastDesc   = descList[len(descList)-1]
	)
	if state.account {
		indexBlock = rawdb.ReadAccountHistoryIndexBlock(db, state.addressHash, lastDesc.id)
	} else {
		indexBlock = rawdb.ReadStorageHistoryIndexBlock(db, state.addressHash, state.storageHash, lastDesc.id)
	}
	bw, err := newBlockWriter(indexBlock, lastDesc)
	if err != nil {
		return nil, err
	}
	return &indexDeleter{
		descList: descList,
		lastID:   lastDesc.max,
		bw:       bw,
		state:    state,
		db:       db,
	}, nil
}

// empty returns an flag indicating whether the state index is empty.
func (d *indexDeleter) empty() bool {
	return d.bw.empty() && len(d.descList) == 1
}

// pop removes the last written element from the index writer.
func (d *indexDeleter) pop(id uint64) error {
	if id == 0 {
		return errors.New("zero history ID is not valid")
	}
	if id != d.lastID {
		return fmt.Errorf("pop element out of order, last: %d, this: %d", d.lastID, id)
	}
	if err := d.bw.pop(id); err != nil {
		return err
	}
	if !d.bw.empty() {
		d.lastID = d.bw.desc.max
		return nil
	}
	// Discarding the last block writer if it becomes empty by popping an element
	d.dropped = append(d.dropped, d.descList[len(d.descList)-1].id)

	// Reset the entire index writer if it becomes empty after popping an element
	if d.empty() {
		d.lastID = 0
		return nil
	}
	d.descList = d.descList[:len(d.descList)-1]

	// Open the previous block writer for deleting
	var (
		indexBlock []byte
		lastDesc   = d.descList[len(d.descList)-1]
	)
	if d.state.account {
		indexBlock = rawdb.ReadAccountHistoryIndexBlock(d.db, d.state.addressHash, lastDesc.id)
	} else {
		indexBlock = rawdb.ReadStorageHistoryIndexBlock(d.db, d.state.addressHash, d.state.storageHash, lastDesc.id)
	}
	bw, err := newBlockWriter(indexBlock, lastDesc)
	if err != nil {
		return err
	}
	d.bw = bw
	d.lastID = bw.desc.max
	return nil
}

// finish deletes the empty index blocks and updates the index meta.
//
// This function is safe to be called multiple times.
func (d *indexDeleter) finish(batch ethdb.Batch) {
	for _, id := range d.dropped {
		if d.state.account {
			rawdb.DeleteAccountHistoryIndexBlock(batch, d.state.addressHash, id)
		} else {
			rawdb.DeleteStorageHistoryIndexBlock(batch, d.state.addressHash, d.state.storageHash, id)
		}
	}
	d.dropped = nil

	// Flush the content of last block writer, regardless it's dirty or not
	if !d.bw.empty() {
		if d.state.account {
			rawdb.WriteAccountHistoryIndexBlock(batch, d.state.addressHash, d.bw.desc.id, d.bw.finish())
		} else {
			rawdb.WriteStorageHistoryIndexBlock(batch, d.state.addressHash, d.state.storageHash, d.bw.desc.id, d.bw.finish())
		}
	}
	// Flush the index metadata into the supplied batch
	if d.empty() {
		if d.state.account {
			rawdb.DeleteAccountHistoryIndex(batch, d.state.addressHash)
		} else {
			rawdb.DeleteStorageHistoryIndex(batch, d.state.addressHash, d.state.storageHash)
		}
	} else {
		buf := make([]byte, 0, indexBlockDescSize*len(d.descList))
		for _, desc := range d.descList {
			buf = append(buf, desc.encode()...)
		}
		if d.state.account {
			rawdb.WriteAccountHistoryIndex(batch, d.state.addressHash, buf)
		} else {
			rawdb.WriteStorageHistoryIndex(batch, d.state.addressHash, d.state.storageHash, buf)
		}
	}
}
