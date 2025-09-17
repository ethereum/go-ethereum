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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// indexReaderWithLimitTag is a wrapper around indexReader that includes an
// additional index position. This position represents the ID of the last
// indexed state history at the time the reader was created, implying that
// indexes beyond this position are unavailable.
type indexReaderWithLimitTag struct {
	reader *indexReader
	limit  uint64
	db     ethdb.KeyValueReader
}

// newIndexReaderWithLimitTag constructs a index reader with indexing position.
func newIndexReaderWithLimitTag(db ethdb.KeyValueReader, state stateIdent, limit uint64) (*indexReaderWithLimitTag, error) {
	r, err := newIndexReader(db, state)
	if err != nil {
		return nil, err
	}
	return &indexReaderWithLimitTag{
		reader: r,
		limit:  limit,
		db:     db,
	}, nil
}

// readGreaterThan locates the first element that is greater than the specified
// id. If no such element is found, MaxUint64 is returned.
//
// Note: It is possible that additional histories have been indexed since the
// reader was created. The reader should be refreshed as needed to load the
// latest indexed data from disk.
func (r *indexReaderWithLimitTag) readGreaterThan(id uint64, lastID uint64) (uint64, error) {
	// Mark the index reader as stale if the tracked indexing position moves
	// backward. This can occur if the pathdb is reverted and certain state
	// histories are unindexed. For simplicity, the reader is marked as stale
	// instead of being refreshed, as this scenario is highly unlikely.
	if r.limit > lastID {
		return 0, fmt.Errorf("index reader is stale, limit: %d, last-state-id: %d", r.limit, lastID)
	}
	// Try to find the element which is greater than the specified target
	res, err := r.reader.readGreaterThan(id)
	if err != nil {
		return 0, err
	}
	// Short circuit if the element is found within the current index
	if res != math.MaxUint64 {
		return res, nil
	}
	// The element was not found, and no additional histories have been indexed.
	// Return a not-found result.
	if r.limit == lastID {
		return res, nil
	}
	// Refresh the index reader and attempt again. If the latest indexed position
	// is even below the ID of the disk layer, it indicates that state histories
	// are being removed. In this case, it would theoretically be better to block
	// the state rollback operation synchronously until all readers are released.
	// Given that it's very unlikely to occur and users try to perform historical
	// state queries while reverting the states at the same time. Simply returning
	// an error should be sufficient for now.
	metadata := loadIndexMetadata(r.db, toHistoryType(r.reader.state.typ))
	if metadata == nil || metadata.Last < lastID {
		return 0, errors.New("state history hasn't been indexed yet")
	}
	if err := r.reader.refresh(); err != nil {
		return 0, err
	}
	r.limit = metadata.Last

	return r.reader.readGreaterThan(id)
}

// historyReader is the structure to access historic state data.
type historyReader struct {
	disk    ethdb.KeyValueReader
	freezer ethdb.AncientReader
	readers map[string]*indexReaderWithLimitTag
}

// newHistoryReader constructs the history reader with the supplied db.
func newHistoryReader(disk ethdb.KeyValueReader, freezer ethdb.AncientReader) *historyReader {
	return &historyReader{
		disk:    disk,
		freezer: freezer,
		readers: make(map[string]*indexReaderWithLimitTag),
	}
}

// readAccountMetadata resolves the account metadata within the specified
// state history.
func (r *historyReader) readAccountMetadata(address common.Address, historyID uint64) ([]byte, error) {
	blob := rawdb.ReadStateAccountIndex(r.freezer, historyID)
	if len(blob) == 0 {
		return nil, fmt.Errorf("account index is truncated, historyID: %d", historyID)
	}
	if len(blob)%accountIndexSize != 0 {
		return nil, fmt.Errorf("account index is corrupted, historyID: %d, size: %d", historyID, len(blob))
	}
	n := len(blob) / accountIndexSize

	pos := sort.Search(n, func(i int) bool {
		h := blob[accountIndexSize*i : accountIndexSize*i+common.AddressLength]
		return bytes.Compare(h, address.Bytes()) >= 0
	})
	if pos == n {
		return nil, fmt.Errorf("account %#x is not found", address)
	}
	offset := accountIndexSize * pos
	if address != common.BytesToAddress(blob[offset:offset+common.AddressLength]) {
		return nil, fmt.Errorf("account %#x is not found", address)
	}
	return blob[offset : accountIndexSize*(pos+1)], nil
}

// readStorageMetadata resolves the storage slot metadata within the specified
// state history.
func (r *historyReader) readStorageMetadata(storageKey common.Hash, storageHash common.Hash, historyID uint64, slotOffset, slotNumber int) ([]byte, error) {
	// TODO(rj493456442) optimize it with partial read
	blob := rawdb.ReadStateStorageIndex(r.freezer, historyID)
	if len(blob) == 0 {
		return nil, fmt.Errorf("storage index is truncated, historyID: %d", historyID)
	}
	if len(blob)%slotIndexSize != 0 {
		return nil, fmt.Errorf("storage indices is corrupted, historyID: %d, size: %d", historyID, len(blob))
	}
	if slotIndexSize*(slotOffset+slotNumber) > len(blob) {
		return nil, fmt.Errorf("storage indices is truncated, historyID: %d, size: %d, offset: %d, length: %d", historyID, len(blob), slotOffset, slotNumber)
	}
	subSlice := blob[slotIndexSize*slotOffset : slotIndexSize*(slotOffset+slotNumber)]

	// TODO(rj493456442) get rid of the metadata resolution
	var (
		m      meta
		target common.Hash
	)
	blob = rawdb.ReadStateHistoryMeta(r.freezer, historyID)
	if err := m.decode(blob); err != nil {
		return nil, err
	}
	if m.version == stateHistoryV0 {
		target = storageHash
	} else {
		target = storageKey
	}
	pos := sort.Search(slotNumber, func(i int) bool {
		slotID := subSlice[slotIndexSize*i : slotIndexSize*i+common.HashLength]
		return bytes.Compare(slotID, target.Bytes()) >= 0
	})
	if pos == slotNumber {
		return nil, fmt.Errorf("storage metadata is not found, slot key: %#x, historyID: %d", storageKey, historyID)
	}
	offset := slotIndexSize * pos
	if target != common.BytesToHash(subSlice[offset:offset+common.HashLength]) {
		return nil, fmt.Errorf("storage metadata is not found, slot key: %#x, historyID: %d", storageKey, historyID)
	}
	return subSlice[offset : slotIndexSize*(pos+1)], nil
}

// readAccount retrieves the account data from the specified state history.
func (r *historyReader) readAccount(address common.Address, historyID uint64) ([]byte, error) {
	metadata, err := r.readAccountMetadata(address, historyID)
	if err != nil {
		return nil, err
	}
	length := int(metadata[common.AddressLength])                                                     // one byte for account data length
	offset := int(binary.BigEndian.Uint32(metadata[common.AddressLength+1 : common.AddressLength+5])) // four bytes for the account data offset

	// TODO(rj493456442) optimize it with partial read
	data := rawdb.ReadStateAccountHistory(r.freezer, historyID)
	if len(data) < length+offset {
		return nil, fmt.Errorf("account data is truncated, address: %#x, historyID: %d, size: %d, offset: %d, len: %d", address, historyID, len(data), offset, length)
	}
	return data[offset : offset+length], nil
}

// readStorage retrieves the storage slot data from the specified state history.
func (r *historyReader) readStorage(address common.Address, storageKey common.Hash, storageHash common.Hash, historyID uint64) ([]byte, error) {
	metadata, err := r.readAccountMetadata(address, historyID)
	if err != nil {
		return nil, err
	}
	// slotIndexOffset:
	//   The offset of storage indices associated with the specified account.
	// slotIndexNumber:
	//   The number of storage indices associated with the specified account.
	slotIndexOffset := int(binary.BigEndian.Uint32(metadata[common.AddressLength+5 : common.AddressLength+9]))
	slotIndexNumber := int(binary.BigEndian.Uint32(metadata[common.AddressLength+9 : common.AddressLength+13]))

	slotMetadata, err := r.readStorageMetadata(storageKey, storageHash, historyID, slotIndexOffset, slotIndexNumber)
	if err != nil {
		return nil, err
	}
	length := int(slotMetadata[common.HashLength])                                                  // one byte for slot data length
	offset := int(binary.BigEndian.Uint32(slotMetadata[common.HashLength+1 : common.HashLength+5])) // four bytes for slot data offset

	// TODO(rj493456442) optimize it with partial read
	data := rawdb.ReadStateStorageHistory(r.freezer, historyID)
	if len(data) < offset+length {
		return nil, fmt.Errorf("storage data is truncated, address: %#x, key: %#x, historyID: %d, size: %d, offset: %d, len: %d", address, storageKey, historyID, len(data), offset, length)
	}
	return data[offset : offset+length], nil
}

// read retrieves the state element data associated with the stateID.
// stateID: represents the ID of the state of the specified version;
// lastID: represents the ID of the latest/newest state history;
// latestValue: represents the state value at the current disk layer with ID == lastID;
func (r *historyReader) read(state stateIdentQuery, stateID uint64, lastID uint64, latestValue []byte) ([]byte, error) {
	tail, err := r.freezer.Tail()
	if err != nil {
		return nil, err
	} // firstID = tail+1

	// stateID+1 == firstID is allowed, as all the subsequent state histories
	// are present with no gap inside.
	if stateID < tail {
		return nil, fmt.Errorf("historical state has been pruned, first: %d, state: %d", tail+1, stateID)
	}

	// To serve the request, all state histories from stateID+1 to lastID
	// must be indexed. It's not supposed to happen unless system is very
	// wrong.
	metadata := loadIndexMetadata(r.disk, toHistoryType(state.typ))
	if metadata == nil || metadata.Last < lastID {
		indexed := "null"
		if metadata != nil {
			indexed = fmt.Sprintf("%d", metadata.Last)
		}
		return nil, fmt.Errorf("state history is not fully indexed, requested: %d, indexed: %s", stateID, indexed)
	}

	// Construct the index reader to locate the corresponding history for
	// state retrieval
	ir, ok := r.readers[state.String()]
	if !ok {
		ir, err = newIndexReaderWithLimitTag(r.disk, state.stateIdent, metadata.Last)
		if err != nil {
			return nil, err
		}
		r.readers[state.String()] = ir
	}
	historyID, err := ir.readGreaterThan(stateID, lastID)
	if err != nil {
		return nil, err
	}
	// The state was not found in the state histories, as it has not been modified
	// since stateID. Use the data from the associated disk layer instead.
	if historyID == math.MaxUint64 {
		return latestValue, nil
	}
	// Resolve data from the specified state history object. Notably, since the history
	// reader operates completely asynchronously with the indexer/unindexer, it's possible
	// that the associated state histories are no longer available due to a rollback.
	// Such truncation should be captured by the state resolver below, rather than returning
	// invalid data.
	if state.typ == typeAccount {
		return r.readAccount(state.address, historyID)
	}
	return r.readStorage(state.address, state.storageKey, state.storageHash, historyID)
}
