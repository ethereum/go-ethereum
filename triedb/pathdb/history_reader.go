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
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/sync/errgroup"
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
func newIndexReaderWithLimitTag(db ethdb.KeyValueReader, state stateIdent, limit uint64, bitmapSize int) (*indexReaderWithLimitTag, error) {
	r, err := newIndexReader(db, state, bitmapSize)
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

// stateHistoryReader is the structure to access historic state data.
type stateHistoryReader struct {
	disk    ethdb.KeyValueReader
	freezer ethdb.AncientReader
	readers map[string]*indexReaderWithLimitTag
}

// newStateHistoryReader constructs the history reader with the supplied db
// for accessing historical states.
func newStateHistoryReader(disk ethdb.KeyValueReader, freezer ethdb.AncientReader) *stateHistoryReader {
	return &stateHistoryReader{
		disk:    disk,
		freezer: freezer,
		readers: make(map[string]*indexReaderWithLimitTag),
	}
}

// readAccountMetadata resolves the account metadata within the specified
// state history.
func (r *stateHistoryReader) readAccountMetadata(address common.Address, historyID uint64) ([]byte, error) {
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
func (r *stateHistoryReader) readStorageMetadata(storageKey common.Hash, storageHash common.Hash, historyID uint64, slotOffset, slotNumber int) ([]byte, error) {
	data, err := rawdb.ReadStateStorageIndex(r.freezer, historyID, slotIndexSize*slotOffset, slotIndexSize*slotNumber)
	if err != nil {
		msg := fmt.Sprintf("id: %d, slot-offset: %d, slot-length: %d", historyID, slotOffset, slotNumber)
		return nil, fmt.Errorf("storage indices corrupted, %s, %w", msg, err)
	}
	// TODO(rj493456442) get rid of the metadata resolution
	var (
		m      meta
		target common.Hash
	)
	blob := rawdb.ReadStateHistoryMeta(r.freezer, historyID)
	if err := m.decode(blob); err != nil {
		return nil, err
	}
	if m.version == stateHistoryV0 {
		target = storageHash
	} else {
		target = storageKey
	}
	pos := sort.Search(slotNumber, func(i int) bool {
		slotID := data[slotIndexSize*i : slotIndexSize*i+common.HashLength]
		return bytes.Compare(slotID, target.Bytes()) >= 0
	})
	if pos == slotNumber {
		return nil, fmt.Errorf("storage metadata is not found, slot key: %#x, historyID: %d", storageKey, historyID)
	}
	offset := slotIndexSize * pos
	if target != common.BytesToHash(data[offset:offset+common.HashLength]) {
		return nil, fmt.Errorf("storage metadata is not found, slot key: %#x, historyID: %d", storageKey, historyID)
	}
	return data[offset : slotIndexSize*(pos+1)], nil
}

// readAccount retrieves the account data from the specified state history.
func (r *stateHistoryReader) readAccount(address common.Address, historyID uint64) ([]byte, error) {
	metadata, err := r.readAccountMetadata(address, historyID)
	if err != nil {
		return nil, err
	}
	length := int(metadata[common.AddressLength])                                                     // one byte for account data length
	offset := int(binary.BigEndian.Uint32(metadata[common.AddressLength+1 : common.AddressLength+5])) // four bytes for the account data offset

	data, err := rawdb.ReadStateAccountHistory(r.freezer, historyID, offset, length)
	if err != nil {
		return nil, fmt.Errorf("account data is truncated, address: %#x, historyID: %d, size: %d, offset: %d, len: %d", address, historyID, len(data), offset, length)
	}
	return data, nil
}

// readStorage retrieves the storage slot data from the specified state history.
func (r *stateHistoryReader) readStorage(address common.Address, storageKey common.Hash, storageHash common.Hash, historyID uint64) ([]byte, error) {
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

	data, err := rawdb.ReadStateStorageHistory(r.freezer, historyID, offset, length)
	if err != nil {
		return nil, fmt.Errorf("storage data is truncated, address: %#x, key: %#x, historyID: %d, size: %d, offset: %d, len: %d", address, storageKey, historyID, len(data), offset, length)
	}
	return data, nil
}

// read retrieves the state element data associated with the stateID.
// stateID: represents the ID of the state of the specified version;
// lastID: represents the ID of the latest/newest state history;
// latestValue: represents the state value at the current disk layer with ID == lastID;
func (r *stateHistoryReader) read(state stateIdentQuery, stateID uint64, lastID uint64, latestValue []byte) ([]byte, error) {
	lastIndexed, err := checkStateAvail(state.stateIdent, typeStateHistory, r.freezer, stateID, lastID, r.disk)
	if err != nil {
		return nil, err
	}
	// Construct the index reader to locate the corresponding history for
	// state retrieval
	ir, ok := r.readers[state.String()]
	if !ok {
		ir, err = newIndexReaderWithLimitTag(r.disk, state.stateIdent, lastIndexed, 0)
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

// trienodeReader is the structure to access historical trienode data.
type trienodeReader struct {
	disk            ethdb.KeyValueReader
	freezer         ethdb.AncientReader
	readConcurrency int // The concurrency used to load trie node data from history
}

// newTrienodeReader constructs the history reader with the supplied db
// for accessing historical trie nodes.
func newTrienodeReader(disk ethdb.KeyValueReader, freezer ethdb.AncientReader, readConcurrency int) *trienodeReader {
	return &trienodeReader{
		disk:            disk,
		freezer:         freezer,
		readConcurrency: readConcurrency,
	}
}

// readTrienode retrieves the trienode data from the specified trienode history.
func (r *trienodeReader) readTrienode(addrHash common.Hash, path string, historyID uint64) ([]byte, bool, error) {
	tr := newTrienodeHistoryReader(historyID, r.freezer)
	return tr.read(addrHash, path)
}

// assembleNode takes a complete node value as the base and applies a list of
// mutation records to assemble the final node value accordingly.
func assembleNode(blob []byte, elements [][][]byte, indices [][]int) ([]byte, error) {
	if len(elements) == 0 && len(indices) == 0 {
		return blob, nil
	}
	children, err := rlp.SplitListValues(blob)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(elements); i++ {
		for j, pos := range indices[i] {
			children[pos] = elements[i][j]
		}
	}
	return rlp.MergeListValues(children)
}

type resultQueue struct {
	data [][]byte
	lock sync.Mutex
}

func newResultQueue(size int) *resultQueue {
	return &resultQueue{
		data: make([][]byte, size, size*2),
	}
}

func (q *resultQueue) set(data []byte, pos int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if pos >= len(q.data) {
		newSize := pos + 1
		if cap(q.data) < newSize {
			newData := make([][]byte, newSize, newSize*2)
			copy(newData, q.data)
			q.data = newData
		}
		q.data = q.data[:newSize]
	}
	q.data[pos] = data
}

func (r *trienodeReader) readOptimized(state stateIdent, it HistoryIndexIterator, latestValue []byte) ([]byte, error) {
	var (
		elements [][][]byte
		indices  [][]int
		blob     = latestValue

		eg    errgroup.Group
		seq   int
		term  atomic.Bool
		queue = newResultQueue(r.readConcurrency * 2)
	)
	eg.SetLimit(r.readConcurrency)

	for {
		id, pos := it.ID(), seq
		seq += 1

		eg.Go(func() error {
			data, found, err := r.readTrienode(state.addressHash, state.path, id)
			if err != nil {
				term.Store(true)
				return err
			}
			// In optimistic readahead mode, it is theoretically possible to encounter a
			// NotFound error, where the trie node does not actually exist and the iterator
			// reports a false-positive mutation record. Terminate the iterator if so, as
			// all the necessary data (checkpoints and all diffs) required has already been
			// fetching.
			if !found {
				term.Store(true)
				log.Debug("Failed to read the trienode")
				return nil
			}
			full, _, err := decodeNodeFull(data)
			if err != nil {
				term.Store(true)
				return err
			}
			if full {
				term.Store(true)
			}
			queue.set(data, pos)
			return nil
		})
		if term.Load() || !it.Next() {
			break
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	for i := 0; i < seq; i++ {
		isComplete, fullBlob, err := decodeNodeFull(queue.data[i])
		if err != nil {
			return nil, err
		}
		// Terminate the loop is the node with full value has been found
		if isComplete {
			blob = fullBlob
			break
		}
		// Decode the partial encoded node and keep iterating the node history
		// until the node with full value being reached.
		element, index, err := decodeNodeCompressed(queue.data[i])
		if err != nil {
			return nil, err
		}
		elements, indices = append(elements, element), append(indices, index)
	}
	slices.Reverse(elements)
	slices.Reverse(indices)
	return assembleNode(blob, elements, indices)
}

// read retrieves the trie node data associated with the stateID.
// stateID: represents the ID of the state of the specified version;
// lastID: represents the ID of the latest/newest trie node history;
// latestValue: represents the trie node value at the current disk layer with ID == lastID;
func (r *trienodeReader) read(state stateIdent, stateID uint64, lastID uint64, latestValue []byte) ([]byte, error) {
	_, err := checkStateAvail(state, typeTrienodeHistory, r.freezer, stateID, lastID, r.disk)
	if err != nil {
		return nil, err
	}
	// Construct the index iterator to traverse the trienode history
	var (
		scheme *indexScheme
		it     HistoryIndexIterator
	)
	if state.addressHash == (common.Hash{}) {
		scheme = accountIndexScheme
	} else {
		scheme = storageIndexScheme
	}
	if state.addressHash == (common.Hash{}) && state.path == "" {
		it = newSeqIter(lastID)
	} else {
		chunkID, nodeID := scheme.splitPathLast(state.path)

		queryIdent := state
		queryIdent.path = chunkID
		ir, err := newIndexReader(r.disk, queryIdent, scheme.getBitmapSize(len(chunkID)))
		if err != nil {
			return nil, err
		}
		filter := extFilter(nodeID)
		it = ir.newIterator(&filter)
	}
	// Move the iterator to the first element whose id is greater than
	// the given number.
	found := it.SeekGT(stateID)
	if err := it.Error(); err != nil {
		return nil, err
	}
	// The state was not found in the trie node histories, as it has not been
	// modified since stateID. Use the data from the associated disk layer
	// instead (full value node as always)
	if !found {
		return latestValue, nil
	}
	return r.readOptimized(state, it, latestValue)
}

// checkStateAvail determines whether the requested historical state is available
// for accessing. What's more, it also returns the ID of the latest indexed history
// entry for subsequent usage.
//
// TODO(rjl493456442) it's really expensive to perform the check for every state
// retrieval, please rework this later.
func checkStateAvail(state stateIdent, exptyp historyType, freezer ethdb.AncientReader, stateID uint64, lastID uint64, db ethdb.KeyValueReader) (uint64, error) {
	if toHistoryType(state.typ) != exptyp {
		return 0, fmt.Errorf("unsupported history type: %d, want: %v", toHistoryType(state.typ), exptyp)
	}
	// firstID = tail+1
	tail, err := freezer.Tail()
	if err != nil {
		return 0, err
	}
	// stateID+1 == firstID is allowed, as all the subsequent history entries
	// are present with no gap inside.
	if stateID < tail {
		return 0, fmt.Errorf("historical state has been pruned, first: %d, state: %d", tail+1, stateID)
	}
	// To serve the request, all history entries from stateID+1 to lastID
	// must be indexed. It's not supposed to happen unless system is very
	// wrong.
	metadata := loadIndexMetadata(db, exptyp)
	if metadata == nil || metadata.Last < lastID {
		indexed := "null"
		if metadata != nil {
			indexed = fmt.Sprintf("%d", metadata.Last)
		}
		return 0, fmt.Errorf("history is not fully indexed, requested: %d, indexed: %s", stateID, indexed)
	}
	return metadata.Last, nil
}
