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
	"fmt"
)

// EntryType discriminates the kind of lookup a single index entry provides.
//
// The entry type determines how clients interpret the index value and which
// position fields are meaningful:
//
//   - EntryTypeBlock:       index value is a block hash; no position tail.
//   - EntryTypeTransaction: index value is a tx hash; the trailing position
//     field holds the cumulative log count of the block before the tx.
//   - EntryTypeLogAddress:  index value is the log's contract address.
//   - EntryTypeLogTopic0-3: index value is the log's topic[i].
//
// On the wire the type id is a 2-byte big-endian field.
type EntryType uint16

const (
	EntryTypeBlock       EntryType = 0 // block hash lookup
	EntryTypeTransaction EntryType = 1 // transaction hash lookup
	EntryTypeLogAddress  EntryType = 2 // log address lookup
	EntryTypeLogTopic0   EntryType = 3 // log topic[0] lookup
	EntryTypeLogTopic1   EntryType = 4 // log topic[1] lookup
	EntryTypeLogTopic2   EntryType = 5 // log topic[2] lookup
	EntryTypeLogTopic3   EntryType = 6 // log topic[3] lookup
)

// Field sizes of the EIP-8304 entry encodings. All numbers are fixed-size
// big-endian. The layouts are parameterized here on purpose: if the spec
// changes a field size, edit this block and the offset arithmetic in
// EncodeEntry and EntryFields stays correct by construction.
const (
	EntryTypeSize     = 2 // type id
	BlockHashSize     = 32
	TxHashSize        = 32
	AddressSize       = 20
	TopicSize         = 32
	BlockNumberSize   = 8
	TxIndexSize       = 4
	LogIndexSize      = 4 // log index within a transaction
	CumulativeLogSize = 4 // logs before the transaction in its block
)

// Total sizes of the four entry kinds, derived at compile time from the field
// sizes above: type || content || block number || tx index || log index.
const (
	BlockEntrySize       = EntryTypeSize + BlockHashSize + BlockNumberSize                                       // 42
	TransactionEntrySize = EntryTypeSize + TxHashSize + BlockNumberSize + TxIndexSize + CumulativeLogSize        // 50
	LogAddressEntrySize  = EntryTypeSize + AddressSize + BlockNumberSize + TxIndexSize + LogIndexSize            // 38
	LogTopicEntrySize    = EntryTypeSize + TopicSize + BlockNumberSize + TxIndexSize + LogIndexSize              // 50
)

// IndexEntry is a single variable-length record of a sorted EIP-8304 index
// table.
//
// Layout (big-endian):
//
//	type id (2) || content || block number (8) [|| tx index (4) || log index (4)]
//
// The content field is 32 bytes for block, transaction and topic entries and
// 20 bytes for address entries, giving the total sizes BlockEntrySize,
// TransactionEntrySize, LogAddressEntrySize and LogTopicEntrySize. Block
// entries stop after the block number; the other entry kinds carry the tx
// index and log index tail, which for transaction entries is the cumulative
// log count of the block before the transaction.
//
// Sorting is lexicographical on the full encoding, giving the ordering:
// type id, content, block number, tx index, log index.
type IndexEntry []byte

// contentSize returns the encoded length of the searchable content field for
// the given entry type.
func contentSize(typ EntryType) int {
	switch typ {
	case EntryTypeBlock:
		return BlockHashSize
	case EntryTypeTransaction:
		return TxHashSize
	case EntryTypeLogAddress:
		return AddressSize
	case EntryTypeLogTopic0, EntryTypeLogTopic1, EntryTypeLogTopic2, EntryTypeLogTopic3:
		return TopicSize
	default:
		panic(fmt.Sprintf("types: unknown entry type %d", typ))
	}
}

// entrySize returns the total encoded length of an entry of the given type.
func entrySize(typ EntryType) int {
	switch typ {
	case EntryTypeBlock:
		return BlockEntrySize
	case EntryTypeTransaction:
		return TransactionEntrySize
	case EntryTypeLogAddress:
		return LogAddressEntrySize
	case EntryTypeLogTopic0, EntryTypeLogTopic1, EntryTypeLogTopic2, EntryTypeLogTopic3:
		return LogTopicEntrySize
	default:
		panic(fmt.Sprintf("types: unknown entry type %d", typ))
	}
}

// EncodeEntry packs the components into a variable-length index entry. content
// must be exactly AddressSize (20) bytes for EntryTypeLogAddress and 32 bytes
// otherwise. For EntryTypeTransaction, logIdx is the cumulative log count of
// the block before the transaction; for all other types it is the log index
// within the transaction. Block entries have no position tail, so txIdx and
// logIdx are ignored for them.
func EncodeEntry(typ EntryType, content []byte, blockNum uint64, txIdx uint32, logIdx uint32) IndexEntry {
	if want := contentSize(typ); len(content) != want {
		panic(fmt.Sprintf("types: entry content length %d, want %d for type %d", len(content), want, typ))
	}
	e := make(IndexEntry, entrySize(typ))
	off := 0
	binary.BigEndian.PutUint16(e[off:off+EntryTypeSize], uint16(typ))
	off += EntryTypeSize
	copy(e[off:], content)
	off += len(content)
	binary.BigEndian.PutUint64(e[off:off+BlockNumberSize], blockNum)
	off += BlockNumberSize
	if typ == EntryTypeBlock {
		return e
	}
	binary.BigEndian.PutUint32(e[off:off+TxIndexSize], txIdx)
	off += TxIndexSize
	binary.BigEndian.PutUint32(e[off:off+LogIndexSize], logIdx)
	return e
}

// EntryFields decodes an entry back into its components. content is a fresh
// copy and does not alias the entry. It returns an error when the entry is
// shorter than the type id, carries an unknown type id, or its length does not
// match the type's fixed layout.
func EntryFields(entry IndexEntry) (typ EntryType, content []byte, blockNum uint64, txIdx uint32, logIdx uint32, err error) {
	if len(entry) < EntryTypeSize {
		return 0, nil, 0, 0, 0, fmt.Errorf("types: entry too short: %d bytes", len(entry))
	}
	typ = EntryType(binary.BigEndian.Uint16(entry[:EntryTypeSize]))
	if typ > EntryTypeLogTopic3 {
		return 0, nil, 0, 0, 0, fmt.Errorf("types: unknown entry type %d", typ)
	}
	if len(entry) != entrySize(typ) {
		return 0, nil, 0, 0, 0, fmt.Errorf("types: entry length %d, want %d for type %d", len(entry), entrySize(typ), typ)
	}
	content = make([]byte, contentSize(typ))
	copy(content, entry[EntryTypeSize:EntryTypeSize+len(content)])
	off := EntryTypeSize + len(content)
	blockNum = binary.BigEndian.Uint64(entry[off : off+BlockNumberSize])
	if typ == EntryTypeBlock {
		return typ, content, blockNum, 0, 0, nil
	}
	off += BlockNumberSize
	txIdx = binary.BigEndian.Uint32(entry[off : off+TxIndexSize])
	off += TxIndexSize
	logIdx = binary.BigEndian.Uint32(entry[off : off+LogIndexSize])
	return typ, content, blockNum, txIdx, logIdx, nil
}

// CompareEntries returns the result of a lexicographic byte comparison
// between two index entries.
//
// Return values follow bytes.Compare:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
//
// Since every field is fixed-size big-endian, this equals the field-wise
// ordering: type id, content, block number, tx index, log index.
func CompareEntries(a, b IndexEntry) int {
	return bytes.Compare(a, b)
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
