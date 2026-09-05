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
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// IndexBuilder accumulates IndexEntry values and computes a Merkle root.
type IndexBuilder struct {
	entries []IndexEntry
}

func NewIndexBuilder() *IndexBuilder {
	return &IndexBuilder{}
}

// AddBlockEntries adds the EIP-8304 entries of one block: the parent block's
// block entry (one-block delay per the spec; none for the genesis block), one
// transaction entry per receipt carrying the cumulative log count of the block
// before that transaction, and per-log address and topic entries with log
// indices relative to their transaction.
func (b *IndexBuilder) AddBlockEntries(parentHash common.Hash, blockNumber uint64, receipts Receipts) {
	if blockNumber > 0 {
		// The table for block N contains the block entry of block N-1.
		b.entries = append(b.entries, EncodeEntry(EntryTypeBlock, parentHash[:], blockNumber-1, 0, 0))
	}
	var cumLogs uint32
	for txIdx, receipt := range receipts {
		// Transaction entry: the trailing field is the cumulative log count
		// of the block before this transaction.
		b.entries = append(b.entries, EncodeEntry(EntryTypeTransaction, receipt.TxHash[:], blockNumber, uint32(txIdx), cumLogs))
		for logIdx, log := range receipt.Logs {
			// Log indices are relative to the transaction.
			b.entries = append(b.entries, EncodeEntry(EntryTypeLogAddress, log.Address[:], blockNumber, uint32(txIdx), uint32(logIdx)))
			// Topic entries
			for topicIdx, topic := range log.Topics {
				if topicIdx < 4 {
					b.entries = append(b.entries, EncodeEntry(EntryTypeLogTopic0+EntryType(topicIdx), topic[:], blockNumber, uint32(txIdx), uint32(logIdx)))
				}
			}
		}
		cumLogs += uint32(len(receipt.Logs))
	}
}

// Entries returns the accumulated entries slice (unsorted). Callers that need
// sorted entries should call Build() or sort the returned slice themselves.
func (b *IndexBuilder) Entries() []IndexEntry {
	return b.entries
}

// Build sorts entries and returns a simple hash.
func (b *IndexBuilder) Build() common.Hash {
	sort.Slice(b.entries, func(i, j int) bool {
		return CompareEntries(b.entries[i], b.entries[j]) < 0
	})
	// Simple hash: Keccak256 of all concatenated entries
	var buf []byte
	for _, e := range b.entries {
		buf = append(buf, e...)
	}
	return crypto.Keccak256Hash(buf)
}
