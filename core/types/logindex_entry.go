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

package types

import (
	"bytes"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

// EntryType discriminates the kind of lookup a single index entry provides.
//
// The entry type determines how clients interpret the index value and which
// position fields are meaningful:
//
//   - EntryTypeBlock:       index value is a block hash; tx/log indices are zero.
//   - EntryTypeTransaction: index value is a tx hash; log index is zero.
//   - EntryTypeLogAddress:  index value is the log's contract address.
//   - EntryTypeLogTopic0-3: index value is the log's topic[i].
type EntryType uint8

const (
	EntryTypeBlock       EntryType = 0 // block hash lookup
	EntryTypeTransaction EntryType = 1 // transaction hash lookup
	EntryTypeLogAddress  EntryType = 2 // log address lookup
	EntryTypeLogTopic0   EntryType = 3 // log topic[0] lookup
	EntryTypeLogTopic1   EntryType = 4 // log topic[1] lookup
	EntryTypeLogTopic2   EntryType = 5 // log topic[2] lookup
	EntryTypeLogTopic3   EntryType = 6 // log topic[3] lookup
)

// IndexEntrySize is the fixed size of each record in the sorted index table.
const IndexEntrySize = 64

// IndexEntry represents a single 64-byte record in the sorted index table.
//
// Layout (big-endian):
//
//	[0]        entry type (1 byte)
//	[1:33]     index value (32 bytes)
//	[33:41]    block number (8 bytes)
//	[41:45]    transaction index (4 bytes)
//	[45:49]    log index (4 bytes)
//	[49:64]    zero-reserved (15 bytes)
//
// Sorting is lexicographical on the full 64 bytes, giving ordering:
// entry_type, index_value, block_number, tx_index, log_index.
type IndexEntry [IndexEntrySize]byte

// EncodeEntry packs the components into a fixed-size 64-byte index entry.
// Non-applicable position fields should be passed as zero.
func EncodeEntry(typ EntryType, value common.Hash, blockNum uint64, txIdx uint32, logIdx uint32) IndexEntry {
	var e IndexEntry
	e[0] = byte(typ)
	copy(e[1:33], value[:])
	binary.BigEndian.PutUint64(e[33:41], blockNum)
	binary.BigEndian.PutUint32(e[41:45], txIdx)
	binary.BigEndian.PutUint32(e[45:49], logIdx)
	// bytes 49..63 remain zero
	return e
}

// EntryFields decodes an IndexEntry back to its components.
func EntryFields(entry IndexEntry) (typ EntryType, value common.Hash, blockNum uint64, txIdx uint32, logIdx uint32) {
	typ = EntryType(entry[0])
	copy(value[:], entry[1:33])
	blockNum = binary.BigEndian.Uint64(entry[33:41])
	txIdx = binary.BigEndian.Uint32(entry[41:45])
	logIdx = binary.BigEndian.Uint32(entry[45:49])
	return
}

// CompareEntries returns the result of a lexicographic byte comparison
// between two index entries.
//
// Return values follow bytes.Compare:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
func CompareEntries(a, b IndexEntry) int {
	return bytes.Compare(a[:], b[:])
}

// EntryTypeString returns a human-readable name for the entry type,
// useful for debugging and log output.
func EntryTypeString(typ EntryType) string {
	switch typ {
	case EntryTypeBlock:
		return "block"
	case EntryTypeTransaction:
		return "transaction"
	case EntryTypeLogAddress:
		return "log_address"
	case EntryTypeLogTopic0:
		return "log_topic0"
	case EntryTypeLogTopic1:
		return "log_topic1"
	case EntryTypeLogTopic2:
		return "log_topic2"
	case EntryTypeLogTopic3:
		return "log_topic3"
	default:
		return "unknown"
	}
}

// TableLevelCount is the number of chained-table levels in the index hierarchy.
// Level i covers a block range of TableSizes[i] blocks with staggered update
// schedules, forming a provable history of the full log index.
const TableLevelCount = 5

// TableSizes gives the block-range size for each level of the chained-table
// hierarchy. Level 0 uses single-block tables (updated every block), level 1
// uses 4-block tables, level 2 uses 16-block tables, level 3 uses 64-block
// tables, and level 4 uses 256-block tables.
var TableSizes = [TableLevelCount]int{1, 4, 16, 64, 256}
