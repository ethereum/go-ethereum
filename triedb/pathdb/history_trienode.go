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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pathdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"iter"
	"maps"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// Each trie node history entry consists of three parts (stored in three freezer
// tables according):
//
// # Header
// The header records metadata, including:
//
//   - the history version    (1 byte)
//   - the parent state root  (32 bytes)
//   - the current state root (32 bytes)
//   - block number           (8 bytes)
//
//   - a lexicographically sorted list of trie IDs
//   - the corresponding offsets into the key and value sections for each trie data chunk
//
// Although some fields (e.g., parent state root, block number) are duplicated
// between the state history and the trienode history, these two histories
// operate independently. To ensure each remains self-contained and self-descriptive,
// we have chosen to maintain these duplicate fields.
//
// # Key section
// The key section stores trie node keys (paths) in a compressed format.
// It also contains relative offsets into the value section for resolving
// the corresponding trie node data. Note that these offsets are relative
// to the data chunk for the trie; the chunk offset must be added to obtain
// the absolute position.
//
// # Value section
// The value section is a concatenated byte stream of all trie node data.
// Each trie node can be retrieved using the offset and length specified
// by its index entry.
//
// The header and key sections are sufficient for locating a trie node,
// while a partial read of the value section is enough to retrieve its data.

// Header section:
//
//    +----------+------------------+---------------------+---------------------+-------+------------------+---------------------+---------------------|
//    | metadata | TrieID(32 bytes) | key offset(4 bytes) | val offset(4 bytes) |  ...  | TrieID(32 bytes) | key offset(4 bytes) | val offset(4 bytes) |
//    +----------+------------------+---------------------+---------------------+-------+------------------+---------------------+---------------------|
//
//
// Key section:
//
//      + restart point                 + restart point (depends on restart interval)
//     /                               /
//    +---------------+---------------+---------------+---------------+---------+
//    |  node entry 1 |  node entry 2 |      ...      |  node entry n | trailer |
//    +---------------+---------------+---------------+---------------+---------+
//     \                             /
//      +---- restart  block ------+
//
// node entry:
//
//              +---- key len ----+
//             /                   \
//    +-------+---------+-----------+---------+-----------------------+-----------------+
//    | shared (varint) | not shared (varint) | value length (varlen) | key (varlen)    |
//    +-----------------+---------------------+-----------------------+-----------------+
//
// trailer:
//
//      +---- 4-bytes ----+     +---- 4-bytes ----+
//     /                    \ /                     \
//    +----------------------+------------------------+-----+--------------------------+
//    | restart_1 key offset | restart_1 value offset | ... | restart number (4-bytes) |
//    +----------------------+------------------------+-----+--------------------------+
//
// Note: Both the key offset and the value offset are relative to the start of
// the trie data chunk. To obtain the absolute offset, add the offset of the
// trie data chunk itself.
//
// Value section:
//
//    +--------------+--------------+-------+---------------+
//    |  node data 1 |  node data 2 |  ...  |  node data n  |
//    +--------------+--------------+-------+---------------+
//
// NOTE: All fixed-length integer are big-endian.

const (
	trienodeHistoryV0           = uint8(0)                    // initial version of node history structure
	trienodeHistoryVersion      = trienodeHistoryV0           // the default node history version
	trienodeMetadataSize        = 1 + 2*common.HashLength + 8 // the size of metadata in the history
	trienodeTrieHeaderSize      = 8 + common.HashLength       // the size of a single trie header in history
	trienodeDataBlockRestartLen = 16                          // The restart interval length of trie node block
)

// trienodeMetadata describes the meta data of trienode history.
type trienodeMetadata struct {
	version uint8       // version tag of history object
	parent  common.Hash // prev-state root before the state transition
	root    common.Hash // post-state root after the state transition
	block   uint64      // associated block number
}

// trienodeHistory represents a set of trie node changes resulting from a state
// transition across the main account trie and all associated storage tries.
type trienodeHistory struct {
	meta     *trienodeMetadata                 // Metadata of the history
	owners   []common.Hash                     // List of trie identifier sorted lexicographically
	nodeList map[common.Hash][]string          // Set of node paths sorted lexicographically
	nodes    map[common.Hash]map[string][]byte // Set of original value of trie nodes before state transition
}

// newTrienodeHistory constructs a trienode history with the provided trie nodes.
func newTrienodeHistory(root common.Hash, parent common.Hash, block uint64, nodes map[common.Hash]map[string][]byte) *trienodeHistory {
	nodeList := make(map[common.Hash][]string)
	for owner, subset := range nodes {
		keys := sort.StringSlice(slices.Collect(maps.Keys(subset)))
		keys.Sort()
		nodeList[owner] = keys
	}
	return &trienodeHistory{
		meta: &trienodeMetadata{
			version: trienodeHistoryVersion,
			parent:  parent,
			root:    root,
			block:   block,
		},
		owners:   slices.SortedFunc(maps.Keys(nodes), common.Hash.Cmp),
		nodeList: nodeList,
		nodes:    nodes,
	}
}

// sharedLen returns the length of the common prefix shared by a and b.
func sharedLen(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// typ implements the history interface, returning the historical data type held.
func (h *trienodeHistory) typ() historyType {
	return typeTrienodeHistory
}

// forEach implements the history interface, returning an iterator to traverse the
// state entries in the history.
func (h *trienodeHistory) forEach() iter.Seq[stateIdent] {
	return func(yield func(stateIdent) bool) {
		for _, owner := range h.owners {
			for _, path := range h.nodeList[owner] {
				if !yield(newTrienodeIdent(owner, path)) {
					return
				}
			}
		}
	}
}

// encode serializes the contained trie nodes into bytes.
func (h *trienodeHistory) encode() ([]byte, []byte, []byte, error) {
	var (
		buf           = make([]byte, 64)
		headerSection bytes.Buffer
		keySection    bytes.Buffer
		valueSection  bytes.Buffer
	)
	binary.Write(&headerSection, binary.BigEndian, h.meta.version) // 1 byte
	headerSection.Write(h.meta.parent.Bytes())                     // 32 bytes
	headerSection.Write(h.meta.root.Bytes())                       // 32 bytes
	binary.Write(&headerSection, binary.BigEndian, h.meta.block)   // 8 byte

	for _, owner := range h.owners {
		// Fill the header section with offsets at key and value section
		headerSection.Write(owner.Bytes())                                       // 32 bytes
		binary.Write(&headerSection, binary.BigEndian, uint32(keySection.Len())) // 4 bytes

		// The offset to the value section is theoretically unnecessary, since the
		// individual value offset is already tracked in the key section. However,
		// we still keep it here for two reasons:
		// - It's cheap to store (only 4 bytes for each trie).
		// - It can be useful for decoding the trie data when key is not required (e.g., in hash mode).
		binary.Write(&headerSection, binary.BigEndian, uint32(valueSection.Len())) // 4 bytes

		// Fill the key section with node index
		var (
			prevKey   []byte
			restarts  []uint32
			prefixLen int

			internalKeyOffset uint32 // key offset for the trie internally
			internalValOffset uint32 // value offset for the trie internally
		)
		for i, path := range h.nodeList[owner] {
			key := []byte(path)
			if i%trienodeDataBlockRestartLen == 0 {
				restarts = append(restarts, internalKeyOffset)
				restarts = append(restarts, internalValOffset)
				prefixLen = 0
			} else {
				prefixLen = sharedLen(prevKey, key)
			}
			value := h.nodes[owner][path]

			// key section
			n := binary.PutUvarint(buf[0:], uint64(prefixLen))          // key length shared (varint)
			n += binary.PutUvarint(buf[n:], uint64(len(key)-prefixLen)) // key length not shared (varint)
			n += binary.PutUvarint(buf[n:], uint64(len(value)))         // value length (varint)

			if _, err := keySection.Write(buf[:n]); err != nil {
				return nil, nil, nil, err
			}
			// unshared key
			if _, err := keySection.Write(key[prefixLen:]); err != nil {
				return nil, nil, nil, err
			}
			n += len(key) - prefixLen
			prevKey = key

			// value section
			if _, err := valueSection.Write(value); err != nil {
				return nil, nil, nil, err
			}
			internalKeyOffset += uint32(n)
			internalValOffset += uint32(len(value))
		}

		// Encode trailer, the number of restart sections is len(restarts))/2,
		// as we track the offsets of both key and value sections.
		var trailer []byte
		for _, number := range append(restarts, uint32(len(restarts))/2) {
			binary.BigEndian.PutUint32(buf[:4], number)
			trailer = append(trailer, buf[:4]...)
		}
		if _, err := keySection.Write(trailer); err != nil {
			return nil, nil, nil, err
		}
	}
	return headerSection.Bytes(), keySection.Bytes(), valueSection.Bytes(), nil
}

// decodeHeader resolves the metadata from the header section. An error
// should be returned if the header section is corrupted.
func decodeHeader(data []byte) (*trienodeMetadata, []common.Hash, []uint32, []uint32, error) {
	if len(data) < trienodeMetadataSize {
		return nil, nil, nil, nil, fmt.Errorf("trienode history is too small, index size: %d", len(data))
	}
	version := data[0]
	if version != trienodeHistoryVersion {
		return nil, nil, nil, nil, fmt.Errorf("unregonized trienode history version: %d", version)
	}
	parent := common.BytesToHash(data[1 : common.HashLength+1])                          // 32 bytes
	root := common.BytesToHash(data[common.HashLength+1 : common.HashLength*2+1])        // 32 bytes
	block := binary.BigEndian.Uint64(data[common.HashLength*2+1 : trienodeMetadataSize]) // 8 bytes

	size := len(data) - trienodeMetadataSize
	if size%trienodeTrieHeaderSize != 0 {
		return nil, nil, nil, nil, fmt.Errorf("truncated trienode history data, size %d", len(data))
	}
	count := size / trienodeTrieHeaderSize

	var (
		owners     = make([]common.Hash, 0, count)
		keyOffsets = make([]uint32, 0, count)
		valOffsets = make([]uint32, 0, count)
	)
	for i := range count {
		n := trienodeMetadataSize + trienodeTrieHeaderSize*i
		owner := common.BytesToHash(data[n : n+common.HashLength])
		if i != 0 && bytes.Compare(owner.Bytes(), owners[i-1].Bytes()) <= 0 {
			return nil, nil, nil, nil, fmt.Errorf("trienode owners are out of order, prev: %v, cur: %v", owners[i-1], owner)
		}
		owners = append(owners, owner)

		// Decode the offset to the key section
		keyOffset := binary.BigEndian.Uint32(data[n+common.HashLength : n+common.HashLength+4])
		if i != 0 && keyOffset <= keyOffsets[i-1] {
			return nil, nil, nil, nil, fmt.Errorf("key offset is out of order, prev: %v, cur: %v", keyOffsets[i-1], keyOffset)
		}
		keyOffsets = append(keyOffsets, keyOffset)

		// Decode the offset into the value section. Note that identical value offsets
		// are valid if the node values in the last trie chunk are all zero (e.g., after
		// a trie deletion).
		valOffset := binary.BigEndian.Uint32(data[n+common.HashLength+4 : n+common.HashLength+8])
		if i != 0 && valOffset < valOffsets[i-1] {
			return nil, nil, nil, nil, fmt.Errorf("value offset is out of order, prev: %v, cur: %v", valOffsets[i-1], valOffset)
		}
		valOffsets = append(valOffsets, valOffset)
	}
	return &trienodeMetadata{
		version: version,
		parent:  parent,
		root:    root,
		block:   block,
	}, owners, keyOffsets, valOffsets, nil
}

func decodeSingle(keySection []byte, onValue func([]byte, int, int) error) ([]string, error) {
	var (
		prevKey    []byte
		items      int
		keyOffsets []uint32
		valOffsets []uint32

		keyOff int // the key offset within the single trie data
		valOff int // the value offset within the single trie data

		keys []string
	)
	// Decode the number of restart section
	if len(keySection) < 4 {
		return nil, fmt.Errorf("key section too short, size: %d", len(keySection))
	}
	nRestarts := binary.BigEndian.Uint32(keySection[len(keySection)-4:])

	if len(keySection) < int(8*nRestarts)+4 {
		return nil, fmt.Errorf("key section too short, restarts: %d, size: %d", nRestarts, len(keySection))
	}
	for i := range int(nRestarts) {
		o := len(keySection) - 4 - (int(nRestarts)-i)*8
		keyOffset := binary.BigEndian.Uint32(keySection[o : o+4])
		if i != 0 && keyOffset <= keyOffsets[i-1] {
			return nil, fmt.Errorf("key offset is out of order, prev: %v, cur: %v", keyOffsets[i-1], keyOffset)
		}
		keyOffsets = append(keyOffsets, keyOffset)

		// Same value offset is allowed just in case all the trie nodes in the last
		// section have zero-size value.
		valOffset := binary.BigEndian.Uint32(keySection[o+4 : o+8])
		if i != 0 && valOffset < valOffsets[i-1] {
			return nil, fmt.Errorf("value offset is out of order, prev: %v, cur: %v", valOffsets[i-1], valOffset)
		}
		valOffsets = append(valOffsets, valOffset)
	}
	keyLimit := len(keySection) - 4 - int(nRestarts)*8

	// Decode data
	for keyOff < keyLimit {
		// Validate the key and value offsets within the single trie data chunk
		if items%trienodeDataBlockRestartLen == 0 {
			if keyOff != int(keyOffsets[items/trienodeDataBlockRestartLen]) {
				return nil, fmt.Errorf("key offset is not matched, recorded: %d, want: %d", keyOffsets[items/trienodeDataBlockRestartLen], keyOff)
			}
			if valOff != int(valOffsets[items/trienodeDataBlockRestartLen]) {
				return nil, fmt.Errorf("value offset is not matched, recorded: %d, want: %d", valOffsets[items/trienodeDataBlockRestartLen], valOff)
			}
		}
		// Resolve the entry from key section
		nShared, nn := binary.Uvarint(keySection[keyOff:]) // key length shared (varint)
		keyOff += nn
		nUnshared, nn := binary.Uvarint(keySection[keyOff:]) // key length not shared (varint)
		keyOff += nn
		nValue, nn := binary.Uvarint(keySection[keyOff:]) // value length (varint)
		keyOff += nn

		// Resolve unshared key
		if keyOff+int(nUnshared) > len(keySection) {
			return nil, fmt.Errorf("key length too long, unshared key length: %d, off: %d, section size: %d", nUnshared, keyOff, len(keySection))
		}
		unsharedKey := keySection[keyOff : keyOff+int(nUnshared)]
		keyOff += int(nUnshared)

		// Assemble the full key
		var key []byte
		if items%trienodeDataBlockRestartLen == 0 {
			if nShared != 0 {
				return nil, fmt.Errorf("unexpected non-zero shared key prefix: %d", nShared)
			}
			key = unsharedKey
		} else {
			if int(nShared) > len(prevKey) {
				return nil, fmt.Errorf("unexpected shared key prefix: %d, prefix key length: %d", nShared, len(prevKey))
			}
			key = append([]byte{}, prevKey[:nShared]...)
			key = append(key, unsharedKey...)
		}
		if items != 0 && bytes.Compare(prevKey, key) >= 0 {
			return nil, fmt.Errorf("trienode paths are out of order, prev: %v, cur: %v", prevKey, key)
		}
		prevKey = key

		// Resolve value
		if onValue != nil {
			if err := onValue(key, valOff, valOff+int(nValue)); err != nil {
				return nil, err
			}
		}
		valOff += int(nValue)

		items++
		keys = append(keys, string(key))
	}
	if keyOff != keyLimit {
		return nil, fmt.Errorf("excessive key data after decoding, offset: %d, size: %d", keyOff, keyLimit)
	}
	return keys, nil
}

func decodeSingleWithValue(keySection []byte, valueSection []byte) ([]string, map[string][]byte, error) {
	var (
		offset int
		nodes  = make(map[string][]byte)
	)
	paths, err := decodeSingle(keySection, func(key []byte, start int, limit int) error {
		if start != offset {
			return fmt.Errorf("gapped value section offset: %d, want: %d", start, offset)
		}
		// start == limit is allowed for zero-value trie node (e.g., non-existent node)
		if start > limit {
			return fmt.Errorf("invalid value offsets, start: %d, limit: %d", start, limit)
		}
		if start > len(valueSection) || limit > len(valueSection) {
			return fmt.Errorf("value section out of range: start: %d, limit: %d, size: %d", start, limit, len(valueSection))
		}
		nodes[string(key)] = valueSection[start:limit]

		offset = limit
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if offset != len(valueSection) {
		return nil, nil, fmt.Errorf("excessive value data after decoding, offset: %d, size: %d", offset, len(valueSection))
	}
	return paths, nodes, nil
}

// decode deserializes the contained trie nodes from the provided bytes.
func (h *trienodeHistory) decode(header []byte, keySection []byte, valueSection []byte) error {
	metadata, owners, keyOffsets, valueOffsets, err := decodeHeader(header)
	if err != nil {
		return err
	}
	h.meta = metadata
	h.owners = owners
	h.nodeList = make(map[common.Hash][]string)
	h.nodes = make(map[common.Hash]map[string][]byte)

	for i := range len(owners) {
		// Resolve the boundary of key section
		keyStart := keyOffsets[i]
		keyLimit := len(keySection)
		if i != len(owners)-1 {
			keyLimit = int(keyOffsets[i+1])
		}
		if int(keyStart) > len(keySection) || keyLimit > len(keySection) {
			return fmt.Errorf("invalid key offsets: keyStart: %d, keyLimit: %d, size: %d", keyStart, keyLimit, len(keySection))
		}

		// Resolve the boundary of value section
		valStart := valueOffsets[i]
		valLimit := len(valueSection)
		if i != len(owners)-1 {
			valLimit = int(valueOffsets[i+1])
		}
		if int(valStart) > len(valueSection) || valLimit > len(valueSection) {
			return fmt.Errorf("invalid value offsets: valueStart: %d, valueLimit: %d, size: %d", valStart, valLimit, len(valueSection))
		}

		// Decode the key and values for this specific trie
		paths, nodes, err := decodeSingleWithValue(keySection[keyStart:keyLimit], valueSection[valStart:valLimit])
		if err != nil {
			return err
		}
		h.nodeList[owners[i]] = paths
		h.nodes[owners[i]] = nodes
	}
	return nil
}

type iRange struct {
	start uint32
	limit uint32
}

// singleTrienodeHistoryReader provides read access to a single trie within the
// trienode history. It stores an offset to the trie's position in the history,
// along with a set of per-node offsets that can be resolved on demand.
type singleTrienodeHistoryReader struct {
	id                   uint64
	reader               ethdb.AncientReader
	valueRange           iRange            // value range within the total value section
	valueInternalOffsets map[string]iRange // value offset within the single trie data
}

func newSingleTrienodeHistoryReader(id uint64, reader ethdb.AncientReader, keyRange iRange, valueRange iRange) (*singleTrienodeHistoryReader, error) {
	// TODO(rjl493456442) partial freezer read should be supported
	keyData, err := rawdb.ReadTrienodeHistoryKeySection(reader, id)
	if err != nil {
		return nil, err
	}
	keyStart := int(keyRange.start)
	keyLimit := int(keyRange.limit)
	if keyRange.limit == math.MaxUint32 {
		keyLimit = len(keyData)
	}
	if len(keyData) < keyStart || len(keyData) < keyLimit {
		return nil, fmt.Errorf("key section too short, start: %d, limit: %d, size: %d", keyStart, keyLimit, len(keyData))
	}

	valueOffsets := make(map[string]iRange)
	_, err = decodeSingle(keyData[keyStart:keyLimit], func(key []byte, start int, limit int) error {
		valueOffsets[string(key)] = iRange{
			start: uint32(start),
			limit: uint32(limit),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &singleTrienodeHistoryReader{
		id:                   id,
		reader:               reader,
		valueRange:           valueRange,
		valueInternalOffsets: valueOffsets,
	}, nil
}

// read retrieves the trie node data with the provided node path.
func (sr *singleTrienodeHistoryReader) read(path string) ([]byte, error) {
	offset, exists := sr.valueInternalOffsets[path]
	if !exists {
		return nil, fmt.Errorf("trienode %v not found", []byte(path))
	}
	// TODO(rjl493456442) partial freezer read should be supported
	valueData, err := rawdb.ReadTrienodeHistoryValueSection(sr.reader, sr.id)
	if err != nil {
		return nil, err
	}
	if len(valueData) < int(sr.valueRange.start) {
		return nil, fmt.Errorf("value section too short, start: %d, size: %d", sr.valueRange.start, len(valueData))
	}
	entryStart := sr.valueRange.start + offset.start
	entryLimit := sr.valueRange.start + offset.limit
	if len(valueData) < int(entryStart) || len(valueData) < int(entryLimit) {
		return nil, fmt.Errorf("value section too short, start: %d, limit: %d, size: %d", entryStart, entryLimit, len(valueData))
	}
	return valueData[int(entryStart):int(entryLimit)], nil
}

// trienodeHistoryReader provides read access to node data in the trie node history.
// It resolves data from the underlying ancient store only when needed, minimizing
// I/O overhead.
type trienodeHistoryReader struct {
	id        uint64                                       // ID of the associated trienode history
	reader    ethdb.AncientReader                          // Database reader of ancient store
	keyRanges map[common.Hash]iRange                       // Key ranges identifying trie chunks
	valRanges map[common.Hash]iRange                       // Value ranges identifying trie chunks
	iReaders  map[common.Hash]*singleTrienodeHistoryReader // readers for each individual trie chunk
}

// newTrienodeHistoryReader constructs the reader for specific trienode history.
func newTrienodeHistoryReader(id uint64, reader ethdb.AncientReader) (*trienodeHistoryReader, error) {
	r := &trienodeHistoryReader{
		id:        id,
		reader:    reader,
		keyRanges: make(map[common.Hash]iRange),
		valRanges: make(map[common.Hash]iRange),
		iReaders:  make(map[common.Hash]*singleTrienodeHistoryReader),
	}
	if err := r.decodeHeader(); err != nil {
		return nil, err
	}
	return r, nil
}

// decodeHeader decodes the header section of trienode history.
func (r *trienodeHistoryReader) decodeHeader() error {
	header, err := rawdb.ReadTrienodeHistoryHeader(r.reader, r.id)
	if err != nil {
		return err
	}
	_, owners, keyOffsets, valOffsets, err := decodeHeader(header)
	if err != nil {
		return err
	}
	for i, owner := range owners {
		// Decode the key range for this trie chunk
		var keyLimit uint32
		if i == len(owners)-1 {
			keyLimit = math.MaxUint32
		} else {
			keyLimit = keyOffsets[i+1]
		}
		r.keyRanges[owner] = iRange{
			start: keyOffsets[i],
			limit: keyLimit,
		}

		// Decode the value range for this trie chunk
		var valLimit uint32
		if i == len(owners)-1 {
			valLimit = math.MaxUint32
		} else {
			valLimit = valOffsets[i+1]
		}
		r.valRanges[owner] = iRange{
			start: valOffsets[i],
			limit: valLimit,
		}
	}
	return nil
}

// read retrieves the trie node data with the provided TrieID and node path.
func (r *trienodeHistoryReader) read(owner common.Hash, path string) ([]byte, error) {
	ir, ok := r.iReaders[owner]
	if !ok {
		keyRange, exists := r.keyRanges[owner]
		if !exists {
			return nil, fmt.Errorf("trie %x is unknown", owner)
		}
		valRange, exists := r.valRanges[owner]
		if !exists {
			return nil, fmt.Errorf("trie %x is unknown", owner)
		}
		var err error
		ir, err = newSingleTrienodeHistoryReader(r.id, r.reader, keyRange, valRange)
		if err != nil {
			return nil, err
		}
		r.iReaders[owner] = ir
	}
	return ir.read(path)
}

// writeTrienodeHistory persists the trienode history associated with the given diff layer.
// nolint:unused
func writeTrienodeHistory(writer ethdb.AncientWriter, dl *diffLayer) error {
	start := time.Now()
	h := newTrienodeHistory(dl.rootHash(), dl.parent.rootHash(), dl.block, dl.nodes.nodeOrigin)
	header, keySection, valueSection, err := h.encode()
	if err != nil {
		return err
	}
	// Write history data into five freezer table respectively.
	if err := rawdb.WriteTrienodeHistory(writer, dl.stateID(), header, keySection, valueSection); err != nil {
		return err
	}
	trienodeHistoryDataBytesMeter.Mark(int64(len(valueSection)))
	trienodeHistoryIndexBytesMeter.Mark(int64(len(header) + len(keySection)))
	trienodeHistoryBuildTimeMeter.UpdateSince(start)

	log.Debug(
		"Stored trienode history", "id", dl.stateID(), "block", dl.block,
		"header", common.StorageSize(len(header)),
		"keySection", common.StorageSize(len(keySection)),
		"valueSection", common.StorageSize(len(valueSection)),
		"elapsed", common.PrettyDuration(time.Since(start)),
	)
	return nil
}

// readTrienodeMetadata resolves the metadata of the specified trienode history.
// nolint:unused
func readTrienodeMetadata(reader ethdb.AncientReader, id uint64) (*trienodeMetadata, error) {
	header, err := rawdb.ReadTrienodeHistoryHeader(reader, id)
	if err != nil {
		return nil, err
	}
	metadata, _, _, _, err := decodeHeader(header)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// readTrienodeHistory resolves a single trienode history object with specific id.
func readTrienodeHistory(reader ethdb.AncientReader, id uint64) (*trienodeHistory, error) {
	header, keySection, valueSection, err := rawdb.ReadTrienodeHistory(reader, id)
	if err != nil {
		return nil, err
	}
	var h trienodeHistory
	if err := h.decode(header, keySection, valueSection); err != nil {
		return nil, err
	}
	return &h, nil
}

// readTrienodeHistories resolves a list of trienode histories with the specific range.
func readTrienodeHistories(reader ethdb.AncientReader, start uint64, count uint64) ([]history, error) {
	headers, keySections, valueSections, err := rawdb.ReadTrienodeHistoryList(reader, start, count)
	if err != nil {
		return nil, err
	}
	var res []history
	for i, header := range headers {
		var h trienodeHistory
		if err := h.decode(header, keySections[i], valueSections[i]); err != nil {
			return nil, err
		}
		res = append(res, &h)
	}
	return res, nil
}
