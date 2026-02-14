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
	"errors"
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
//   - the corresponding offsets into the key and value sections for each trie
//     data chunk. The offsets refer to the end position of each chunk, with
//     the assumption that the key and value sections for the first data chunk
//     start at offset 0.
//
// Although some fields (e.g., parent state root, block number) are duplicated
// between the state history and the trienode history, these two histories
// operate independently. To ensure each remains self-contained and self-descriptive,
// we have chosen to maintain these duplicate fields.
//
// # Key section
// The key section stores trie node keys (paths) in a compressed format.
// It also contains relative offsets into the value section for locating
// the corresponding trie node data. These offsets are relative to the
// beginning of the trie data chunk, the chunk's base offset must be added
// to obtain the absolute position in the value section.
//
// # Value section
// The value section is a concatenated byte stream of all trie node data.
// Each trie node can be retrieved using the offset and length specified
// by its index entry.
//
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
//    +-------+---------+-----------+---------+-----------------------+-----------------------+
//    | shared (varint) | not shared (varint) | value length (varlen) | unshared key (varlen) |
//    +-----------------+---------------------+-----------------------+-----------------------+
//
// trailer:
//
//      +---- 4-bytes ----+     +---- 4-bytes ----+
//     /                    \ /                     \
//    +----------------------+------------------------+-----+--------------------------+
//    | restart_1 key offset | restart_1 value offset | ... | restart number (4-bytes) |
//    +----------------------+------------------------+-----+--------------------------+
//
// Note: Both the key offset and the value offset are relative to the beginning
// of the trie data chunk. The chunk's base offset must be added to obtain the
// absolute position in the value section.
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
	nodeList := make(map[common.Hash][]string, len(nodes))
	for owner, subset := range nodes {
		keys := make(sort.StringSlice, 0, len(subset))
		for k := range subset {
			keys = append(keys, k)
		}
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

// typ implements the history interface, returning the historical data type held.
func (h *trienodeHistory) typ() historyType {
	return typeTrienodeHistory
}

// forEach implements the history interface, returning an iterator to traverse the
// state entries in the history.
func (h *trienodeHistory) forEach() iter.Seq[indexElem] {
	return func(yield func(indexElem) bool) {
		for _, owner := range h.owners {
			var (
				scheme  *indexScheme
				paths   = h.nodeList[owner]
				indexes = make(map[string]map[uint16]struct{})
			)
			if owner == (common.Hash{}) {
				scheme = accountIndexScheme
			} else {
				scheme = storageIndexScheme
			}
			for _, leaf := range findLeafPaths(paths) {
				chunks, ids := scheme.splitPath(leaf)
				for i := 0; i < len(chunks); i++ {
					if _, exists := indexes[chunks[i]]; !exists {
						indexes[chunks[i]] = make(map[uint16]struct{})
					}
					indexes[chunks[i]][ids[i]] = struct{}{}
				}
			}
			for chunk, ids := range indexes {
				elem := trienodeIndexElem{
					owner: owner,
					path:  chunk,
					data:  slices.Collect(maps.Keys(ids)),
				}
				if !yield(elem) {
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
		// Fill the key section with node index
		var (
			prevKey   []byte
			restarts  []uint32
			prefixLen int

			internalKeyOffset uint32 // key offset within the trie data internally
			internalValOffset uint32 // value offset within the trie data internally
		)
		for i, path := range h.nodeList[owner] {
			key := []byte(path)

			// Track the internal key and value offsets at the beginning of the
			// restart section. The absolute offsets within the key and value
			// sections should first include the offset of the trie chunk itself
			// stored in the header section.
			if i%trienodeDataBlockRestartLen == 0 {
				restarts = append(restarts, internalKeyOffset)
				restarts = append(restarts, internalValOffset)
				prefixLen = 0
			} else {
				prefixLen = commonPrefixLen(prevKey, key)
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

		// Fill the header section with the offsets of the key and value sections.
		// Note that key/value offsets are intentionally recorded *after* encoding
		// into their respective sections, so each offset refers to an end position.
		// For n trie chunks, n offset pairs are sufficient to uniquely locate each
		// chunk's data. For example, [0, offset_0] defines the range of trie chunk 0,
		// while [offset_{n-2}, offset_{n-1}] defines the range of trie chunk n-1.
		headerSection.Write(owner.Bytes())                                         // 32 bytes
		binary.Write(&headerSection, binary.BigEndian, uint32(keySection.Len()))   // 4 bytes
		binary.Write(&headerSection, binary.BigEndian, uint32(valueSection.Len())) // 4 bytes
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

// decodeKeyEntry resolves a single entry from the key section starting from
// the specified offset.
func decodeKeyEntry(keySection []byte, offset int) (uint64, uint64, []byte, int, error) {
	var byteRead int

	// Resolve the length of shared key
	nShared, nn := binary.Uvarint(keySection[offset:]) // key length shared (varint)
	if nn <= 0 {
		return 0, 0, nil, 0, fmt.Errorf("corrupted varint encoding for nShared at offset %d", offset)
	}
	byteRead += nn

	// Resolve the length of unshared key
	nUnshared, nn := binary.Uvarint(keySection[offset+byteRead:]) // key length not shared (varint)
	if nn <= 0 {
		return 0, 0, nil, 0, fmt.Errorf("corrupted varint encoding for nUnshared at offset %d", offset+byteRead)
	}
	byteRead += nn

	// Resolve the length of value
	nValue, nn := binary.Uvarint(keySection[offset+byteRead:]) // value length (varint)
	if nn <= 0 {
		return 0, 0, nil, 0, fmt.Errorf("corrupted varint encoding for nValue at offset %d", offset+byteRead)
	}
	byteRead += nn

	// Validate that the values can fit in an int to prevent overflow on 32-bit systems
	if nShared > uint64(math.MaxUint32) || nUnshared > uint64(math.MaxUint32) || nValue > uint64(math.MaxUint32) {
		return 0, 0, nil, 0, errors.New("key/value size too large")
	}

	// Resolve the unshared key
	if offset+byteRead+int(nUnshared) > len(keySection) {
		return 0, 0, nil, 0, fmt.Errorf("key length too long, unshared key length: %d, off: %d, section size: %d", nUnshared, offset+byteRead, len(keySection))
	}
	unsharedKey := keySection[offset+byteRead : offset+byteRead+int(nUnshared)]
	byteRead += int(nUnshared)

	return nShared, nValue, unsharedKey, byteRead, nil
}

// decodeRestartTrailer resolves all the offsets recorded at the trailer.
func decodeRestartTrailer(keySection []byte) ([]uint32, []uint32, int, error) {
	// Decode the number of restart section
	if len(keySection) < 4 {
		return nil, nil, 0, fmt.Errorf("key section too short, size: %d", len(keySection))
	}
	nRestarts := binary.BigEndian.Uint32(keySection[len(keySection)-4:])

	// Decode the trailer
	var (
		keyOffsets = make([]uint32, 0, int(nRestarts))
		valOffsets = make([]uint32, 0, int(nRestarts))
	)
	if len(keySection) < int(8*nRestarts)+4 {
		return nil, nil, 0, fmt.Errorf("key section too short, restarts: %d, size: %d", nRestarts, len(keySection))
	}
	for i := range int(nRestarts) {
		o := len(keySection) - 4 - (int(nRestarts)-i)*8
		keyOffset := binary.BigEndian.Uint32(keySection[o : o+4])
		if i != 0 && keyOffset <= keyOffsets[i-1] {
			return nil, nil, 0, fmt.Errorf("key offset is out of order, prev: %v, cur: %v", keyOffsets[i-1], keyOffset)
		}
		keyOffsets = append(keyOffsets, keyOffset)

		// Same value offset is allowed just in case all the trie nodes in the last
		// section have zero-size value.
		valOffset := binary.BigEndian.Uint32(keySection[o+4 : o+8])
		if i != 0 && valOffset < valOffsets[i-1] {
			return nil, nil, 0, fmt.Errorf("value offset is out of order, prev: %v, cur: %v", valOffsets[i-1], valOffset)
		}
		valOffsets = append(valOffsets, valOffset)
	}
	keyLimit := len(keySection) - 4 - int(nRestarts)*8 // End of key data
	return keyOffsets, valOffsets, keyLimit, nil
}

// decodeRestartSection resolves all entries in a restart section. The keyData
// contains the encoded keys for the section.
//
// onValue is the callback function being invoked for each resolved entry. The
// start and limit are the offsets within the restart section, the base value
// offset of the restart section itself should be added by the caller itself.
// What's more, this function should return `aborted == true` if the entry
// resolution should be terminated.
func decodeRestartSection(keyData []byte, onValue func(key []byte, start int, limit int) (bool, error)) error {
	var (
		prevKey []byte
		items   int

		keyOff int // the key offset within the single trie data
		valOff int // the value offset within the single trie data
	)
	// Decode data
	for keyOff < len(keyData) {
		nShared, nValue, unsharedKey, nn, err := decodeKeyEntry(keyData, keyOff)
		if err != nil {
			return err
		}
		keyOff += nn

		// Assemble the full key
		var key []byte
		if items%trienodeDataBlockRestartLen == 0 {
			if nShared != 0 {
				return fmt.Errorf("unexpected non-zero shared key prefix: %d", nShared)
			}
			key = unsharedKey
		} else {
			if int(nShared) > len(prevKey) {
				return fmt.Errorf("unexpected shared key prefix: %d, prefix key length: %d", nShared, len(prevKey))
			}
			key = make([]byte, int(nShared)+len(unsharedKey))
			copy(key[:nShared], prevKey[:nShared])
			copy(key[nShared:], unsharedKey)
		}
		if items != 0 && bytes.Compare(prevKey, key) >= 0 {
			return fmt.Errorf("trienode paths are out of order, prev: %v, cur: %v", prevKey, key)
		}
		prevKey = key

		valEnd := valOff + int(nValue)
		abort, err := onValue(key, valOff, valEnd)
		if err != nil {
			return err
		}
		if abort {
			return nil
		}
		valOff = valEnd
		items++
	}
	if keyOff != len(keyData) {
		return fmt.Errorf("excessive key data after decoding, offset: %d, size: %d", keyOff, len(keyData))
	}
	return nil
}

// onValue is the callback function being invoked for each resolved entry. The
// start and limit are the offsets within this trie chunk, the base value
// offset of the trie chunk itself should be added by the caller itself.
func decodeSingle(keySection []byte, onValue func([]byte, int, int) error) error {
	keyOffsets, valOffsets, keyLimit, err := decodeRestartTrailer(keySection)
	if err != nil {
		return err
	}
	for i := 0; i < len(keyOffsets); i++ {
		var keyData []byte
		if i == len(keyOffsets)-1 {
			keyData = keySection[keyOffsets[i]:keyLimit]
		} else {
			keyData = keySection[keyOffsets[i]:keyOffsets[i+1]]
		}
		err := decodeRestartSection(keyData, func(key []byte, start int, limit int) (bool, error) {
			valStart := int(valOffsets[i]) + start
			valLimit := int(valOffsets[i]) + limit

			// Possible in tests
			if onValue == nil {
				return false, nil
			}
			if err := onValue(key, valStart, valLimit); err != nil {
				return false, err
			}
			return false, nil // abort=false
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func decodeSingleWithValue(keySection []byte, valueSection []byte) ([]string, map[string][]byte, error) {
	var (
		offset    int
		estimated = len(keySection) / 8
		nodes     = make(map[string][]byte, estimated)
		paths     = make([]string, 0, estimated)
	)
	err := decodeSingle(keySection, func(key []byte, start int, limit int) error {
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
		strkey := string(key)
		paths = append(paths, strkey)
		nodes[strkey] = valueSection[start:limit]

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
		// Resolve the boundary of the key section, each offset referring
		// to the end position of this trie chunk.
		var keyStart, keyLimit uint32
		if i != 0 {
			keyStart = keyOffsets[i-1]
		}
		keyLimit = keyOffsets[i]
		if int(keyStart) > len(keySection) || int(keyLimit) > len(keySection) {
			return fmt.Errorf("invalid key offsets: keyStart: %d, keyLimit: %d, size: %d", keyStart, keyLimit, len(keySection))
		}

		// Resolve the boundary of the value section, each offset referring
		// to the end position of this trie chunk.
		var valStart, valLimit uint32
		if i != 0 {
			valStart = valueOffsets[i-1]
		}
		valLimit = valueOffsets[i]
		if int(valStart) > len(valueSection) || int(valLimit) > len(valueSection) {
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

func (ir iRange) len() uint32 {
	return ir.limit - ir.start
}

type singleTrienodeHistoryReader struct {
	id         uint64
	reader     ethdb.AncientReader
	keyData    []byte
	valueRange iRange
}

func newSingleTrienodeHistoryReader(id uint64, reader ethdb.AncientReader, keyRange iRange, valueRange iRange) (*singleTrienodeHistoryReader, error) {
	keyData, err := rawdb.ReadTrienodeHistoryKeySection(reader, id, uint64(keyRange.start), uint64(keyRange.len()))
	if err != nil {
		return nil, err
	}
	return &singleTrienodeHistoryReader{
		id:         id,
		reader:     reader,
		keyData:    keyData,
		valueRange: valueRange,
	}, nil
}

// searchSingle searches for a specific trie node identified by the provided
// key within a single trie node chunk.
//
// It returns the node value's offset range (start and limit) within the
// trie node data. An error is returned if the node cannot be found.
func (sr *singleTrienodeHistoryReader) searchSingle(key []byte) (int, int, bool, error) {
	keyOffsets, valOffsets, keyLimit, err := decodeRestartTrailer(sr.keyData)
	if err != nil {
		return 0, 0, false, err
	}
	// Binary search against the boundary keys for each restart section
	var (
		boundFind     bool
		boundValueLen uint64
	)
	pos := sort.Search(len(keyOffsets), func(i int) bool {
		_, nValue, dkey, _, derr := decodeKeyEntry(sr.keyData[keyOffsets[i]:], 0)
		if derr != nil {
			err = derr
			return false
		}
		n := bytes.Compare(key, dkey)
		if n == 0 {
			boundFind = true
			boundValueLen = nValue
		}
		return n <= 0
	})
	if err != nil {
		return 0, 0, false, err
	}
	// The node is found as the boundary of restart section
	if boundFind {
		start := valOffsets[pos]
		limit := valOffsets[pos] + uint32(boundValueLen)
		return int(start), int(limit), true, nil
	}
	// The node is not found as all others have larger key than the target
	if pos == 0 {
		return 0, 0, false, nil
	}
	// Search the target node within the restart section
	var keyData []byte
	if pos == len(keyOffsets) {
		keyData = sr.keyData[keyOffsets[pos-1]:keyLimit] // last section
	} else {
		keyData = sr.keyData[keyOffsets[pos-1]:keyOffsets[pos]] // non-last section
	}
	var (
		nStart int
		nLimit int
		found  bool
	)
	err = decodeRestartSection(keyData, func(ikey []byte, start, limit int) (bool, error) {
		if bytes.Equal(key, ikey) {
			nStart = int(valOffsets[pos-1]) + start
			nLimit = int(valOffsets[pos-1]) + limit
			found = true
			return true, nil // abort = true
		}
		return false, nil // abort = false
	})
	if err != nil {
		return 0, 0, false, err
	}
	if !found {
		return 0, 0, false, nil
	}
	return nStart, nLimit, true, nil
}

// read retrieves the trie node data with the provided node path.
func (sr *singleTrienodeHistoryReader) read(key []byte) ([]byte, bool, error) {
	start, limit, found, err := sr.searchSingle(key)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	valStart := uint64(start) + uint64(sr.valueRange.start)
	valLen := uint64(limit - start)
	value, err := rawdb.ReadTrienodeHistoryValueSection(sr.reader, sr.id, valStart, valLen)
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

// trienodeHistoryReader provides read access to node data in the trie node history.
// It resolves data from the underlying ancient store only when needed, minimizing
// I/O overhead.
type trienodeHistoryReader struct {
	id     uint64              // ID of the associated trienode history
	reader ethdb.AncientReader // Database reader of ancient store
}

// newTrienodeHistoryReader constructs the reader for specific trienode history.
func newTrienodeHistoryReader(id uint64, reader ethdb.AncientReader) *trienodeHistoryReader {
	return &trienodeHistoryReader{
		id:     id,
		reader: reader,
	}
}

// decodeHeader decodes the header section of trienode history.
func (r *trienodeHistoryReader) decodeHeader(owner common.Hash) (iRange, iRange, bool, error) {
	header, err := rawdb.ReadTrienodeHistoryHeader(r.reader, r.id)
	if err != nil {
		return iRange{}, iRange{}, false, err
	}
	_, owners, keyOffsets, valOffsets, err := decodeHeader(header)
	if err != nil {
		return iRange{}, iRange{}, false, err
	}
	pos := sort.Search(len(owners), func(i int) bool {
		return owner.Cmp(owners[i]) <= 0
	})
	if pos == len(owners) || owners[pos] != owner {
		return iRange{}, iRange{}, false, nil
	}
	var keyRange iRange
	if pos != 0 {
		keyRange.start = keyOffsets[pos-1]
	}
	keyRange.limit = keyOffsets[pos]

	var valRange iRange
	if pos != 0 {
		valRange.start = valOffsets[pos-1]
	}
	valRange.limit = valOffsets[pos]
	return keyRange, valRange, true, nil
}

// read retrieves the trie node data with the provided TrieID and node path.
func (r *trienodeHistoryReader) read(owner common.Hash, path string) ([]byte, bool, error) {
	keyRange, valRange, found, err := r.decodeHeader(owner)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	ir, err := newSingleTrienodeHistoryReader(r.id, r.reader, keyRange, valRange)
	if err != nil {
		return nil, false, err
	}
	return ir.read([]byte(path))
}

// writeTrienodeHistory persists the trienode history associated with the given diff layer.
func writeTrienodeHistory(writer ethdb.AncientWriter, dl *diffLayer, rate uint32) error {
	start := time.Now()
	nodes, err := dl.nodes.encodeNodeHistory(dl.root, rate)
	if err != nil {
		return err
	}
	h := newTrienodeHistory(dl.rootHash(), dl.parent.rootHash(), dl.block, nodes)
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
